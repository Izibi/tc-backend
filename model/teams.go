
package model

import (
  "database/sql"
  "strings"
  "time"
  "github.com/go-errors/errors"
  "github.com/jmoiron/sqlx"
  "tezos-contests.izibi.com/backend/utils"
)

type Team struct {
  Id int64
  Created_at string
  Updated_at time.Time
  Deleted_at sql.NullString
  Access_code string
  Contest_id int64
  Is_open bool
  Is_locked bool
  Name string
  Public_key string
}

type TeamMember struct {
  Team_id int64
  User_id int64
  Joined_at string
  Is_creator bool
}

func (m *Model) LoadTeam(teamId int64) (*Team, error) {
  var team Team
  err := m.dbMap.Get(&team, teamId)
  if err != nil { return nil, errors.Wrap(err, 0) }
  return &team, nil
}

func (m *Model) LoadUserContestTeam(userId int64, contestId int64) (*Team, error) {
  var team Team
  err := m.dbMap.SelectOne(&team,
    `SELECT t.* FROM teams t LEFT JOIN team_members tm ON t.id = tm.team_id
     WHERE t.contest_id = ? AND tm.user_id = ? LIMIT 1`, contestId, userId)
  if err == sql.ErrNoRows { return nil, nil }
  if err != nil { return nil, errors.Wrap(err, 0) }
  return &team, nil
}

func (m *Model) LoadContestTeams(contestId int64) ([]Team, error) {
  var teams []Team
  err := m.dbMap.Select(&teams, `SELECT * FROM teams WHERE contest_id = ? AND deleted_at IS NULL`, contestId)
  if err != nil { return nil, errors.Wrap(err, 0) }
  return teams, nil
}

func (m *Model) LoadTeamsById(ids []int64) ([]Team, error) {
  var teams []Team
  query, args, err := sqlx.In(`SELECT * FROM teams WHERE id IN (?)`, ids)
  if err != nil { return nil, errors.Wrap(err, 0) }
  err = m.dbMap.Select(&teams, query, args...)
  if err != nil { return nil, errors.Wrap(err, 0) }
  return teams, nil
}

func (m *Model) LoadTeamMembersByTeamId(teamIds []int64) ([]TeamMember, error) {
  var items []TeamMember
  query, args, err := sqlx.In(`SELECT * FROM team_members WHERE team_id IN (?) ORDER BY team_id, joined_at`, teamIds)
  if err != nil { return nil, errors.Wrap(err, 0) }
  err = m.dbMap.Select(&items, query, args...)
  if err != nil { return nil, errors.Wrap(err, 0) }
  return items, nil
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
  accessCode, err := utils.NewAccessCode()
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

  return nil
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
  var team Team
  m.dbMap.SelectOne(&team,
    `SELECT * FROM teams WHERE contest_id = ? AND access_code = ? AND deleted_at IS NULL`,
     contestId, accessCode)
  if err == sql.ErrNoRows { return errors.Errorf("bad access code") }
  if err != nil { return err }
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

  return nil
}

func (m *Model) LeaveTeam(teamId int64, userId int64) error {
  /* Load the team and verify it is not locked. */
  team, err := m.LoadTeam(teamId)
  if err != nil { return err }
  if team.Is_locked { return errors.Errorf("team is locked") }

  /* Load team_member row to determine if the creator is leaving the team. */
  var member TeamMember
  err = m.dbMap.SelectOne(&member,
    `SELECT * FROM team_members WHERE user_id = ? and team_id = ?`, userId, teamId)
  if err == sql.ErrNoRows { return nil }
  if err != nil { return err }

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
func (m *Model) UpdateTeam(teamId int64, userId int64, arg UpdateTeamArg) (*Team, error) {

  /* Verify the user making the request is in the team. */
  isMember, err := m.IsUserInTeam(userId, teamId)
  if err != nil { return nil, err }
  if !isMember { return nil, errors.Errorf("forbidden") }

  /* Load the team and verify it is not locked. */
  team, err := m.LoadTeam(teamId)
  if err != nil { return nil, err }
  if team.Is_locked { return nil, errors.Errorf("team is locked") }

  save := false
  if arg.IsOpen != nil && *arg.IsOpen != team.Is_open {
    save = true
    team.Is_open = *arg.IsOpen
  }
  if arg.PublicKey != nil && *arg.PublicKey != team.Public_key {
    save = true
    team.Public_key = *arg.PublicKey
  }
  if save {
    team.Updated_at = time.Now()
    m.dbMap.Update(team)
  }

  return team, nil
}

func (m *Model) RenewTeamAccessCode(teamId int64, userId int64) (*Team, error) {
  /* Verify the user making the request is in the team. */
  isMember, err := m.IsUserInTeam(userId, teamId)
  if err != nil { return nil, err }
  if !isMember { return nil, errors.Errorf("forbidden") }
  /* Load the team and verify it is not locked. */
  team, err := m.LoadTeam(teamId)
  if err != nil { return nil, err }
  if team.Is_locked { return nil, errors.Errorf("team is locked") }
  /* Renew the access code. */
  accessCode, err := utils.NewAccessCode()
  if err != nil { return nil, err }
  _, err = m.db.Exec(
    `UPDATE teams SET access_code = ? WHERE id = ?`, accessCode, teamId)
  if err != nil {
    // TODO: retry a few times in case of access code conflict
    return nil, errors.Wrap(err, 0)
  }
  team.Access_code = accessCode
  return team, nil
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
