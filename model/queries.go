
package model

import (
  "crypto/rand"
  "fmt"
  "encoding/binary"
  "strings"
  "time"
  "database/sql"
  "github.com/go-errors/errors"
  "github.com/jmoiron/sqlx"
  "github.com/itchyny/base58-go"
  j "tezos-contests.izibi.com/backend/jase"
)

type UserProfile interface {
  Id() string
  Username() string
  Firstname() string
  Lastname() string
  Badges() []string
}

type IRow interface {
  Scan(dest ...interface{}) error
  StructScan(dest interface{}) error
}

func (m *Model) FindUserByForeignId(foreignId string) (string, error) {
  var id string
  err := m.db.QueryRow(`SELECT id FROM users WHERE foreign_id = ?`, foreignId).Scan(&id)
  if err != nil { return "", err }
  return id, nil
}

func (m *Model) ImportUserProfile(profile UserProfile, now time.Time) (string, error) {
  var userId string
  foreignId := profile.Id()
  rows, err := m.db.Query(`SELECT id FROM users WHERE foreign_id = ?`, foreignId)
  if err != nil { return "", errors.Wrap(err, 0) }
  if rows.Next() {
    err = rows.Scan(&userId)
    rows.Close()
    if err != nil { return "", errors.Wrap(err, 0) }
    _, err := m.db.Exec(
      `UPDATE users SET updated_at = ?, username = ?, firstname = ?, lastname = ? WHERE id = ?`,
      now, profile.Username(), profile.Firstname(), profile.Lastname(), userId)
    if err != nil { return "", errors.Wrap(err, 0) }
  } else {
    rows.Close()
    res, err := m.db.Exec(
      `INSERT INTO users (foreign_id, created_at, updated_at, username, firstname, lastname) VALUES (?, ?, ?, ?, ?, ?)`,
      foreignId, now, now, profile.Username(), profile.Firstname(), profile.Lastname())
    if err != nil { return "", errors.Wrap(err, 0) }
    newId, err := res.LastInsertId()
    if err != nil { return "", errors.Wrap(err, 0) }
    userId = string(newId)
  }
  err = m.UpdateBadges(userId, profile.Badges())
  if err != nil { return "", err }
  return userId, nil
}

func (m *Model) UpdateBadges(userId string, badges []string) error {

  /* If the user holds no badges, delete all badges unconditionnally. */
  if len(badges) == 0 {
    _, err := m.db.Exec(`DELETE FROM user_badges
      WHERE user_badges.user_id = ?`, userId)
    return errors.Wrap(err, 0)
  }

  /* Delete any badges the user no longer holds. */
  query, args, err := sqlx.In(`DELETE FROM user_badges
    USING user_badges INNER JOIN badges
    WHERE user_badges.user_id = ?
      AND user_badges.badge_id = badges.id
      AND badges.symbol NOT IN (?)`, userId, badges)
  if err != nil { return errors.Wrap(err, 0) }
  _, err = m.db.Exec(query, args...)
  if err != nil { return errors.Wrap(err, 0) }

  /* Insert any badges the user did not previously hold. */
  query, args, err = sqlx.In(`INSERT IGNORE INTO user_badges (user_id, badge_id)
    SELECT ?, id FROM badges
      WHERE id NOT IN (SELECT badge_id FROM user_badges WHERE user_badges.user_id = ?)
      AND symbol IN (?)`,
     userId, userId, badges)
  if err != nil { return errors.Wrap(err, 0) }
  _, err = m.db.Exec(query, args...)
  if err != nil { return errors.Wrap(err, 0) }

  return nil
}

func (m *Model) ViewUser(id string) error {
  row := m.db.QueryRowx(`SELECT * FROM users WHERE id = ?`, id)
  user, err := m.loadUserRow(row, BaseFacet)
  if err != nil { return err }
  m.Set("userId", j.String(user.Id))
  return nil
}

func (m *Model) loadUsers(ids []string) error {
  query, args, err := sqlx.In(`SELECT * FROM users WHERE id IN (?)`, ids)
  if err != nil { return errors.Wrap(err, 0) }
  rows, err := m.db.Queryx(query, args...)
  if err != nil { return errors.Wrap(err, 0) }
  defer rows.Close()
  for rows.Next() {
    _, err = m.loadUserRow(rows, BaseFacet)
    if err != nil { return err }
  }
  return nil
}

func (m *Model) ViewUserContests(userId string) error {
  var err error
  rows, err := m.db.Queryx(
    `select c.* from user_badges ub, contests c
     where ub.user_id = ? and ub.badge_id = c.required_badge_id`, userId)
  if err != nil { return errors.Wrap(err, 0) }
  defer rows.Close()
  contestIds := j.Array()
  for rows.Next() {
    contest, err := m.loadContestRow(rows, BaseFacet)
    if err != nil { return err }
    contestIds.Item(j.String(contest.Id))
    m.tasks.Need(contest.Task_id)
  }
  err = m.tasks.Load(m.loadTasks)
  if err != nil { return err }
  m.Set("contestIds", contestIds)
  return nil
}

func (m *Model) ViewUserContest(userId string, contestId string) error {
  var err error
  /* verify user has access to contest */
  ok, err := m.testUserContestAccess(userId, contestId)
  if err != nil { return err }
  if !ok { return errors.Errorf("access denied") }

  /* load contest, task */
  contest, err := m.loadContestRow(m.db.QueryRowx(
    `select * from contests where id = ?`, contestId), BaseFacet)
  if err != nil { return err }
  m.tasks.Need(contest.Task_id)
  err = m.tasks.Load(m.loadTasks)
  if err != nil { return err }
  err = m.loadTaskResources(contest.Task_id)
  if err != nil { return err }

  return nil
}

func (m *Model) testUserContestAccess(userId string, contestId string) (bool, error) {
  row := m.db.QueryRow(
    `select count(c.id) from user_badges ub, contests c
     where c.id = ? and ub.user_id = ? and ub.badge_id = c.required_badge_id`, contestId, userId)
  var count int
  err := row.Scan(&count)
  if err != nil { return false, errors.Wrap(err, 0) }
  return count == 1, nil
}

func (m *Model) loadTasks(ids []string) error {
  query, args, err := sqlx.In(`select * from tasks where id in (?)`, ids)
  if err != nil { return errors.Wrap(err, 0) }
  rows, err := m.db.Queryx(query, args...)
  if err != nil { return errors.Wrap(err, 0) }
  defer rows.Close()
  for rows.Next() {
    _, err = m.loadTaskRow(rows)
    if err != nil { return err }
  }
  return nil
}

func (m *Model) loadTaskResources(taskId string) error {
  rows, err := m.db.Queryx(
    `SELECT * FROM task_resources WHERE task_id = ? ORDER BY rank`, taskId)
  if err != nil { return errors.Wrap(err, 0) }
  defer rows.Close()
  for rows.Next() {
    _, err = m.loadTaskResourceRow(rows)
    if err != nil { return err }
  }
  return nil
}

func (m *Model) ViewUserContestTeam(userId string, contestId string) error {
  team, err := m.loadUserContestTeam(userId, contestId, Facets{Base: true, Member: true})
  if err != nil { return err }
  if team == nil {
    m.Set("teamId", j.Null)
    return nil
  }
  _, err = m.loadTeamMembers(team.Id, BaseFacet)
  if err != nil { return err }
  m.Set("teamId", j.String(team.Id))
  return nil
}

func (m *Model) loadUserContestTeam(userId string, contestId string, f Facets) (*Team, error) {
  return m.loadTeamRow(m.db.QueryRowx(
    `SELECT t.* FROM teams t LEFT JOIN team_members tm ON t.id = tm.team_id
     WHERE t.contest_id = ? AND tm.user_id = ? LIMIT 1`, contestId, userId), f)
}

func (m *Model) loadTeam(teamId string, f Facets) (*Team, error) {
  return m.loadTeamRow(m.db.QueryRowx(
    `SELECT * FROM teams WHERE id = ?`, teamId), f)
}

func (m *Model) loadTeamMembers(teamId string, f Facets) ([]TeamMember, error) {
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

func (m *Model) CreateTeam(userId string, contestId string, teamName string) error {
  var err error

  /* Verify the user has access to the contest. */
  ok, err := m.testUserContestAccess(userId, contestId)
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
  row = m.db.QueryRowx(
    `SELECT COUNT(t.id) FROM teams t WHERE t.contest_id = ? AND t.name = ?`,
     contestId, teamName)
  if err = row.Scan(&teamCount); err != nil { return errors.Wrap(err, 0) }
  if teamCount != 0 { return errors.Errorf("name is not unique") }

  /* Create the team. */
  accessCode, err := generateAccessCode()
  if err != nil { return err }
  res, err := m.db.Exec(
    `INSERT INTO teams (created_at, access_code, contest_id, is_open, is_locked, name, public_key)
     VALUES (NOW(), ?, ?, 1, 0, ?, NULL)`, accessCode, contestId, teamName)
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

func (m *Model) testUserNotInTeam(userId string, contestId string) (bool, error) {
  var teamCount int
  err := m.db.QueryRowx(
    `SELECT COUNT(t.id) FROM teams t LEFT JOIN team_members tm ON t.id = tm.team_id
     WHERE t.contest_id = ? AND tm.user_id = ?`, contestId, userId).Scan(&teamCount)
  if err != nil { return false, errors.Wrap(err, 0) }
  return teamCount == 0, nil
}

func (m *Model) JoinTeam(userId string, contestId string, accessCode string) error {
  var err error

  /* Verify the user has access to the contest. */
  ok, err := m.testUserContestAccess(userId, contestId)
  if err != nil { return err }
  if !ok { return errors.Errorf("access denied") }

  /* Verify the user is not already in a team. */
  ok, err = m.testUserNotInTeam(userId, contestId)
  if err != nil { return err }
  if !ok { return errors.Errorf("access denied") }

  /* Find the team based on the access code provided. */
  var teamId string
  err = m.db.QueryRowx(
    `SELECT id FROM teams WHERE contest_id = ? AND access_code = ?`,
     contestId, accessCode).Scan(&teamId)
  if err == sql.ErrNoRows { return errors.Errorf("bad access code") }
  if err != nil { return errors.Wrap(err, 0) }

  /* Add the user as team member */
  _, err = m.db.Exec(
    `INSERT INTO team_members (team_id, user_id, joined_at, is_creator)
     VALUES (?, ?, NOW(), 0)`, teamId, userId)
  if err != nil { return errors.Wrap(err, 0) }

  return m.ViewUserContestTeam(userId, contestId)
}

func (m *Model) isUserInTeam(userId string, teamId string) (bool, error) {
  row := m.db.QueryRowx(
    `SELECT COUNT(*) FROM team_members WHERE user_id = ? and team_id = ?`, userId, teamId)
  var count int
  if err := row.Scan(&count); err != nil { return false, errors.Wrap(err, 0) }
  return count == 1, nil
}

func (m *Model) RenewTeamAccessCode(teamId string, userId string) error {
  /* Load the team and verify it is not locked. */
  team, err := m.loadTeam(teamId, NullFacet)
  if err != nil { return err }
  if team.Is_locked {
    return errors.Errorf("team is locked")
  }
  /* Verify the user making the request is in the team. */
  isMember, err := m.isUserInTeam(userId, teamId)
  if err != nil { return err }
  if !isMember { return nil }
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
  _, err = m.loadTeam(teamId, Facets{Base: true, Member: isMember})
  return err
}

func (m *Model) LeaveTeam(teamId string, userId string) error {
  /* Load the team and verify it is not locked. */
  team, err := m.loadTeam(teamId, NullFacet)
  if err != nil { return err }
  if team.Is_locked {
    return errors.Errorf("team is locked")
  }

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
  var newCreatorUserId string
  err = row.Scan(&newCreatorUserId)
  if err == sql.ErrNoRows {
    /* The team became empty, delete it. */
    _, err = m.db.Exec(`DELETE FROM teams WHERE id = ?`, teamId)
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
