
package model

import (
  "crypto/rand"
  "database/sql"
  "encoding/binary"
  "fmt"
  "strings"
  "time"
  "github.com/go-errors/errors"
  "github.com/itchyny/base58-go"
  "github.com/jmoiron/sqlx"
  j "tezos-contests.izibi.com/backend/jase"
)

type Team struct {
  Id int64
  Created_at string
  Updated_at string
  Deleted_at sql.NullString
  Access_code string
  Contest_id int64
  Is_open bool
  Is_locked bool
  Name string
  Public_key sql.NullString
}

type TeamMember struct {
  Team_id int64
  User_id int64
  Joined_at string
  Is_creator bool
}

func (m *Model) CreateTeam(userId int64, contestId int64, teamName string) error {
  var err error

  /* Verify the user has access to the contest. */
  ok, err := m.CanUserAccessContest(userId, contestId)
  if err != nil { return err }
  if !ok { return errors.Errorf("access denied") }

  /* Verify the user is not already in a team. */
  row := m.db.QueryRowx(
    `SELECT COUNT(t.id) FROM teams t LEFT JOIN team_members tm ON t.id = tm.team_id
     WHERE t.contest_id = ? AND tm.user_id = ?`, contestId, userId)
  var teamCount int
  if err = row.Scan(&teamCount); err != nil { return errors.Wrap(err, 0) }
  if teamCount != 0 { return errors.Errorf("already in a team") }

  /* Verify the team name is unique for the contest. */
  teamName = strings.TrimSpace(teamName)
  if len(teamName) == 0 { return errors.Errorf("team name is too short") }
  row = m.db.QueryRowx(
    `SELECT COUNT(t.id) FROM teams t WHERE t.contest_id = ? AND t.deleted_at IS NULL AND t.name = ?`,
     contestId, teamName)
  if err = row.Scan(&teamCount); err != nil { return errors.Wrap(err, 0) }
  if teamCount != 0 { return errors.Errorf("team name is not unique") }

  /* Create the team. */
  accessCode, err := generateAccessCode()
  if err != nil { return err }

  res, err := m.db.Exec(
    `INSERT INTO teams (access_code, contest_id, is_open, is_locked, name)
     VALUES (?, ?, 1, 0, ?)`, accessCode, contestId, teamName)
  if err != nil {
    // TODO: retry a few times in case of access code conflict
    return errors.Wrap(err, 0)
  }
  teamId, err := res.LastInsertId()

  /* Add the user as team creator */
  _, err = m.db.Exec(
    `INSERT INTO team_members (team_id, user_id, joined_at, is_creator)
     VALUES (?, ?, NOW(), 1)`, teamId, userId)
  if err != nil { return errors.Wrap(err, 0) }

  return m.ViewUserContestTeam(userId, contestId)
}

func (m *Model) JoinTeam(userId int64, contestId int64, accessCode string) error {
  var err error

  /* Verify the user has access to the contest. */
  ok, err := m.CanUserAccessContest(userId, contestId)
  if err != nil { return err }
  if !ok { return errors.Errorf("access denied") }

  /* Verify the user is not already in a team. */
  ok, err = m.isUserNotInAnyTeam(userId, contestId)
  if err != nil { return err }
  if !ok { return errors.Errorf("already in a team") }

  /* Find the team based on the access code provided. */
  team, err := m.loadTeamRow(m.db.QueryRowx(
    `SELECT * FROM teams WHERE contest_id = ? AND access_code = ? AND deleted_at IS NULL`,
     contestId, accessCode), BaseFacet)
  if err != nil { return err }
  if team == nil { return errors.Errorf("bad access code") }
  if team.Is_locked { return errors.Errorf("team is locked") }
  if !team.Is_open { return errors.Errorf("team is closed") }

  /* Verify the max team size is not exceeded. */
  teamSize, err := m.getTeamMembersCount(team.Id)
  if err != nil { return err }
  if teamSize >= 3 { return errors.Errorf("team is full") }

  /* Add the user as team member */
  _, err = m.db.Exec(
    `INSERT INTO team_members (team_id, user_id, joined_at, is_creator)
     VALUES (?, ?, NOW(), 0)`, team.Id, userId)
  if err != nil { return errors.Wrap(err, 0) }

  return m.ViewUserContestTeam(userId, contestId)
}

func (m *Model) LeaveTeam(teamId int64, userId int64) error {
  /* Load the team and verify it is not locked. */
  team, err := m.LoadTeam(teamId, NullFacet)
  if err != nil { return err }
  if team.Is_locked { return errors.Errorf("team is locked") }

  /* Load team_member row to determine if the creator is leaving the team. */
  member, err := m.loadTeamMemberRow(m.db.QueryRowx(
    `SELECT * FROM team_members WHERE user_id = ? and team_id = ?`, userId, teamId), NullFacet)
  if err != nil { return err }
  if member == nil { return nil }

  /* Remove the member from the team. */
  _, err = m.db.Exec(`DELETE FROM team_members WHERE team_id = ? AND user_id = ?`, teamId, userId)
  if err != nil { return errors.Wrap(err, 0) }

  /* If the user leaving the team is not its creator, we are done. */
  if !member.Is_creator {
    return nil
  }

  /* Find the oldest user remaining in the team to transfer creator status. */
  row := m.db.QueryRowx(
    `SELECT user_id FROM team_members
     WHERE team_id = ? AND user_id <> ?
     ORDER BY joined_at LIMIT 1`, teamId, userId)
  var newCreatorUserId int64
  err = row.Scan(&newCreatorUserId)
  if err == sql.ErrNoRows {
    /* The team became empty, mark it as deleted. */
    _, err = m.db.Exec(`UPDATE teams SET deleted_at = ? WHERE id = ?`, time.Now(), teamId)
    if err != nil { return errors.Wrap(err, 0) }
    return nil
  }
  if err != nil { return errors.Wrap(err, 0) }

  /* Transfer creator status. */
  _, err = m.db.Exec(
    `UPDATE team_members SET is_creator = 1
     WHERE team_id = ? AND user_id = ?`, teamId, newCreatorUserId)
  if err != nil { return errors.Wrap(err, 0) }

  /* TODO: push team update to connected members */
  return nil
}

type UpdateTeamArg struct {
  IsOpen *bool `json:"isOpen"`
  PublicKey *string `json:"publicKey"`
}
func (m *Model) UpdateTeam(teamId int64, userId int64, arg UpdateTeamArg) error {

  /* Verify the user making the request is in the team. */
  isMember, err := m.IsUserInTeam(userId, teamId)
  if err != nil { return err }
  if !isMember { return nil }

  /* Load the team and verify it is not locked. */
  team, err := m.LoadTeam(teamId, NullFacet)
  if err != nil { return err }
  if team.Is_locked { return errors.Errorf("team is locked") }

  if arg.IsOpen != nil {
    _, err = m.db.Exec(
      `UPDATE teams SET is_open = ? WHERE id = ?`, *arg.IsOpen, teamId)
    if err != nil { return errors.Wrap(err, 0) }
  }
  if arg.PublicKey != nil {
    _, err = m.db.Exec(
      `UPDATE teams SET public_key = ? WHERE id = ?`, *arg.PublicKey, teamId)
    if err != nil { return errors.Wrap(err, 0) }
  }

  return m.ViewUserContestTeam(userId, team.Contest_id)
}

func (m *Model) RenewTeamAccessCode(teamId int64, userId int64) error {
  /* Verify the user making the request is in the team. */
  isMember, err := m.IsUserInTeam(userId, teamId)
  if err != nil { return err }
  if !isMember { return nil }
  /* Load the team and verify it is not locked. */
  team, err := m.LoadTeam(teamId, NullFacet)
  if err != nil { return err }
  if team.Is_locked { return errors.Errorf("team is locked") }
  /* Renew the access code. */
  accessCode, err := generateAccessCode()
  if err != nil { return err }
  _, err = m.db.Exec(
    `UPDATE teams SET access_code = ? WHERE id = ?`, accessCode, teamId)
  if err != nil {
    // TODO: retry a few times in case of access code conflict
    return errors.Wrap(err, 0)
  }
  /* Send back the updated team. */
  _, err = m.LoadTeam(teamId, Facets{Base: true, Member: isMember})
  return err
}

func (m *Model) FindTeamIdByKey(publicKey string) (int64, error) {
  row := m.db.QueryRow(`SELECT id FROM teams WHERE public_key = ? AND deleted_at IS NULL`, publicKey)
  var id int64
  err := row.Scan(&id)
  if err == sql.ErrNoRows { return 0, nil }
  if err != nil { return 0, err }
  return id, nil
}

func (m *Model) isUserNotInAnyTeam(userId int64, contestId int64) (bool, error) {
  var teamCount int
  err := m.db.QueryRowx(
    `SELECT COUNT(t.id) FROM teams t LEFT JOIN team_members tm ON t.id = tm.team_id
     WHERE t.contest_id = ? AND tm.user_id = ?`, contestId, userId).Scan(&teamCount)
  if err != nil { return false, errors.Wrap(err, 0) }
  return teamCount == 0, nil
}

func (m *Model) IsUserInTeam(userId int64, teamId int64) (bool, error) {
  row := m.db.QueryRowx(
    `SELECT COUNT(*) FROM team_members WHERE user_id = ? and team_id = ?`, userId, teamId)
  var count int
  if err := row.Scan(&count); err != nil { return false, errors.Wrap(err, 0) }
  return count == 1, nil
}

func (m *Model) getTeamMembersCount(teamId int64) (int, error) {
  row := m.db.QueryRowx(
    `SELECT COUNT(*) FROM team_members WHERE team_id = ?`, teamId)
  var count int
  if err := row.Scan(&count); err != nil { return 0, errors.Wrap(err, 0) }
  return count, nil
}

func generateAccessCode() (string, error) {
  binCode := make([]byte, 64)
  _, err := rand.Read(binCode)
  if err != nil { return "", errors.Wrap(err, 0) }
  intCode := binary.LittleEndian.Uint64(binCode)
  strCode := fmt.Sprintf("%d", intCode)
  accessCode, err := base58.BitcoinEncoding.Encode([]byte(strCode))
  if err != nil { return "", errors.Wrap(err, 0) }
  return string(accessCode), nil
}

func (m *Model) LoadTeam(teamId int64, f Facets) (*Team, error) {
  return m.loadTeamRow(m.db.QueryRowx(
    `SELECT * FROM teams WHERE id = ? AND deleted_at IS NULL`, teamId), f)
}

func (m *Model) LoadUserContestTeam(userId int64, contestId int64, f Facets) (*Team, error) {
  return m.loadTeamRow(m.db.QueryRowx(
    `SELECT t.* FROM teams t LEFT JOIN team_members tm ON t.id = tm.team_id
     WHERE t.contest_id = ? AND tm.user_id = ? LIMIT 1`, contestId, userId), f)
}

func (m *Model) loadTeams(ids []int64) error {
  query, args, err := sqlx.In(`SELECT * FROM teams WHERE id IN (?)`, ids)
  if err != nil { return errors.Wrap(err, 0) }
  rows, err := m.db.Queryx(query, args...)
  if err != nil { return errors.Wrap(err, 0) }
  defer rows.Close()
  for rows.Next() {
    _, err = m.loadTeamRow(rows, BaseFacet)
    if err != nil { return err }
  }
  return nil
}

func (m *Model) loadTeamRow(row IRow, f Facets) (*Team, error) {
  var res Team
  err := row.StructScan(&res)
  if err == sql.ErrNoRows { return nil, nil }
  if err != nil { return nil, errors.Wrap(err, 0) }
  if f.Base {
    view := j.Object()
    view.Prop("id", j.String(m.ExportId(res.Id)))
    view.Prop("createdAt", j.String(res.Created_at))
    view.Prop("updatedAt", j.String(res.Updated_at))
    if res.Deleted_at.Valid {
      view.Prop("deletedAt", j.String(res.Deleted_at.String))
    }
    view.Prop("contestId", j.String(m.ExportId(res.Contest_id)))
    view.Prop("isOpen", j.Boolean(res.Is_open))
    view.Prop("isLocked", j.Boolean(res.Is_locked))
    view.Prop("name", j.String(res.Name))
    publicKey := j.Null
    if res.Public_key.Valid {
      publicKey = j.String(res.Public_key.String)
    }
    view.Prop("publicKey", publicKey)
    m.Add(fmt.Sprintf("teams %s", m.ExportId(res.Id)), view)
  }
  if f.Member {
    view := j.Object()
    view.Prop("accessCode", j.String(res.Access_code))
    m.Add(fmt.Sprintf("teams#member %s", m.ExportId(res.Id)), view)
  }
  return &res, nil
}

func (m *Model) loadTeamMembers(teamId int64, f Facets) ([]TeamMember, error) {
  rows, err := m.db.Queryx(
    `SELECT * FROM team_members WHERE team_id = ? ORDER BY joined_at`, teamId)
  if err != nil { return nil, errors.Wrap(err, 0) }
  defer rows.Close()
  var members []TeamMember
  for rows.Next() {
    member, err := m.loadTeamMemberRow(rows, f)
    if err != nil { return nil, err }
    members = append(members, *member)
    m.users.Need(member.User_id)
  }
  err = m.users.Load(m.loadUsers)
  if err != nil { return nil, errors.Wrap(err, 0) }
  return members, nil
}

func (m *Model) loadTeamMemberRow(row IRow, f Facets) (*TeamMember, error) {
  var res TeamMember
  err := row.StructScan(&res)
  if err == sql.ErrNoRows { return nil, nil }
  if err != nil { return nil, errors.Wrap(err, 0) }
  if f.Base {
    view := j.Object()
    view.Prop("teamId", j.String(m.ExportId(res.Team_id)))
    view.Prop("userId", j.String(m.ExportId(res.User_id)))
    view.Prop("joinedAt", j.String(res.Joined_at))
    view.Prop("isCreator", j.Boolean(res.Is_creator))
    m.Add(fmt.Sprintf("teamMembers %s.%s", m.ExportId(res.Team_id), m.ExportId(res.User_id)), view)
  }
  return &res, nil
}
