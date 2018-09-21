
package model

import (
  "time"
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

func (model *Model) FindUserByForeignId(foreignId string) (string, error) {
  var id string
  err := model.db.QueryRow(`SELECT id FROM users WHERE foreign_id = ?`, foreignId).Scan(&id)
  if err != nil { return "", err }
  return id, nil
}

func (model *Model) ImportUserProfile(profile UserProfile, now time.Time) (string, error) {
  var userId string
  foreignId := profile.Id()
  rows, err := model.db.Query(`SELECT id FROM users WHERE foreign_id = ?`, foreignId)
  if err != nil { return "", errors.Wrap(err, 0) }
  if rows.Next() {
    err = rows.Scan(&userId)
    rows.Close()
    if err != nil { return "", errors.Wrap(err, 0) }
    _, err := model.db.Exec(
      `UPDATE users SET updated_at = ?, username = ?, firstname = ?, lastname = ? WHERE id = ?`,
      now, profile.Username(), profile.Firstname(), profile.Lastname(), userId)
    if err != nil { return "", errors.Wrap(err, 0) }
  } else {
    rows.Close()
    res, err := model.db.Exec(
      `INSERT INTO users (foreign_id, created_at, updated_at, username, firstname, lastname) VALUES (?, ?, ?, ?, ?, ?)`,
      foreignId, now, now, profile.Username(), profile.Firstname(), profile.Lastname())
    if err != nil { return "", errors.Wrap(err, 0) }
    newId, err := res.LastInsertId()
    if err != nil { return "", errors.Wrap(err, 0) }
    userId = string(newId)
  }
  err = model.UpdateBadges(userId, profile.Badges())
  if err != nil { return "", errors.Wrap(err, 0) }
  return userId, nil
}

func (model *Model) UpdateBadges(userId string, badges []string) error {

  /* If the user holds no badges, delete all badges unconditionnally. */
  if len(badges) == 0 {
    _, err := model.db.Exec(`DELETE FROM user_badges
      WHERE user_badges.user_id = ?`, userId)
    return err
  }

  /* Delete any badges the user no longer holds. */
  query, args, err := sqlx.In(`DELETE FROM user_badges
    USING user_badges INNER JOIN badges
    WHERE user_badges.user_id = ?
      AND user_badges.badge_id = badges.id
      AND badges.symbol NOT IN (?)`, userId, badges)
  if err != nil { return errors.Wrap(err, 0) }
  _, err = model.db.Exec(query, args...)
  if err != nil { return errors.Wrap(err, 0) }

  /* Insert any badges the user did not previously hold. */
  query, args, err = sqlx.In(`INSERT IGNORE INTO user_badges (user_id, badge_id)
    SELECT ?, id FROM badges
      WHERE id NOT IN (SELECT badge_id FROM user_badges WHERE user_badges.user_id = ?)
      AND symbol IN (?)`,
     userId, userId, badges)
  if err != nil { return errors.Wrap(err, 0) }
  _, err = model.db.Exec(query, args...)
  if err != nil { return errors.Wrap(err, 0) }

  return nil
}

func (model *Model) ViewUser(id string) (j.Value, error) {
  rows, err := model.db.Query(
    `select username, firstname, lastname from users where id = ?`, id)
  if err != nil { return j.Null, errors.Wrap(err, 0) }
  defer rows.Close()
  if !rows.Next() { return j.Null, nil }
  var username, firstname, lastname string
  err = rows.Scan(&username, &firstname, &lastname)
  if err != nil { return j.Null, errors.Wrap(err, 0) }
  user := j.Object()
  user.Prop("id", j.String(id))
  user.Prop("username", j.String(username))
  user.Prop("firstname", j.String(firstname))
  user.Prop("lastname", j.String(lastname))
  model.Add("users."+id, user)
  return j.String(id), nil
}

func (m *Model) ViewUserContests(userId string) (j.Value, error) {
  var err error
  rows, err := m.db.Queryx(
    `select
      c.id, c.title, c.description, c.logo_url, c.task_id, c.is_registration_open,
      c.starts_at, c.ends_at
    from user_badges ub, contests c
    where ub.user_id = ? and ub.badge_id = c.required_badge_id`, userId)
  if err != nil { return j.Null, errors.Wrap(err, 0) }
  defer rows.Close()
  contestIds := j.Array()
  for rows.Next() {
    contest, err := m.loadContestRow(rows)
    if err != nil { return j.Null, errors.Wrap(err, 0) }
    contestIds.Item(j.String(contest.Id))
    m.tasks.Need(contest.Task_id)
  }
  err = m.tasks.Load(m.loadTasks)
  if err != nil { return j.Null, errors.Wrap(err, 0) }
  return contestIds, nil
}

func (m *Model) ViewUserContest(userId string, contestId string) error {
  var err error
  /* verify user has access to contest */
  row := m.db.QueryRow(
    `select count(c.id) from user_badges ub, contests c
     where c.id = ? and ub.user_id = ? and ub.badge_id = c.required_badge_id`, contestId, userId)
  var count int
  err = row.Scan(&count)
  if err != nil { return errors.Wrap(err, 0) }
  if count != 1 { return errors.Errorf("access denied") }

  /* load contest, task */
  contest, err := m.loadContestRow(m.db.QueryRowx(`select
    id, title, description, logo_url, task_id, is_registration_open,
    starts_at, ends_at from contests where id = ?`, contestId))
  if err != nil { return errors.Wrap(err, 0) }
  m.tasks.Need(contest.Task_id)
  err = m.tasks.Load(m.loadTasks)
  if err != nil { return errors.Wrap(err, 0) }

  // model.loadTaskResources(contest.TaskId)
  // TODO: load task resources

  return nil
}

func (model *Model) loadTasks(ids []string) error {
  query, args, err := sqlx.In(`select id, title from tasks where id in (?)`, ids)
  if err != nil { return errors.Wrap(err, 0) }
  rows, err := model.db.Queryx(query, args...)
  if err != nil { return errors.Wrap(err, 0) }
  defer rows.Close()
  for rows.Next() {
    _, err = model.loadTaskRow(rows)
    if err != nil { return err }
  }
  return nil
}
