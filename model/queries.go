
package model

import (
  "time"
  "database/sql"
  "github.com/go-errors/errors"
  "github.com/jmoiron/sqlx"
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
