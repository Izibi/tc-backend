
package model

import (
  "time"
  "github.com/go-errors/errors"
  "github.com/jmoiron/sqlx"
  j "tezos-contests.izibi.com/backend/jase"
)

type Response interface {
  Add(key string, view j.Value)
}

type UserProfile interface {
  Id() string
  Username() string
  Firstname() string
  Lastname() string
  Badges() []string
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

func (model *Model) ViewUser(resp Response, id string) (j.Value, error) {
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
  resp.Add("users."+id, user)
  return j.String(id), nil
}

func (model *Model) ViewUserContests(resp Response, userId string) (j.Value, error) {
  var err error
  rows, err := model.db.Query(
    `select
      c.id, c.title, c.description, c.logo_url, c.task_id, c.is_registration_open,
      c.starts_at, c.ends_at
    from user_badges ub, contests c
    where ub.user_id = ? and ub.badge_id = c.required_badge_id`, userId)
  if err != nil { return j.Null, errors.Wrap(err, 0) }
  defer rows.Close()
  contestIds := j.Array()
  taskIds := make(map[string]bool)
  for rows.Next() {
    var id, title, description, logoUrl, taskId, startsAt, endsAt string
    var isRegistrationOpen bool
    err = rows.Scan(&id, &title, &description, &logoUrl, &taskId, &isRegistrationOpen, &startsAt, &endsAt)
    if err != nil { return j.Null, errors.Wrap(err, 0) }
    contest := j.Object()
    contest.Prop("id", j.String(id))
    contest.Prop("title", j.String(title))
    contest.Prop("description", j.String(description))
    contest.Prop("logoUrl", j.String(logoUrl))
    contest.Prop("taskId", j.String(taskId))
    contest.Prop("startsAt", j.String(startsAt))
    contest.Prop("endsAt", j.String(endsAt))
    taskIds[taskId] = true
    resp.Add("contests."+id, contest)
    contestIds.Item(j.String(id))
  }
  err = model.ViewTasks(resp, keysOfSet(taskIds))
  if err != nil { return j.Null, errors.Wrap(err, 0) }
  return contestIds, nil
}

func (model *Model) ViewTasks(resp Response, ids []string) error {
  if len(ids) == 0 { return nil }
  query, args, err := sqlx.In(`select id, title from tasks where id in (?)`, ids)
  if err != nil { return errors.Wrap(err, 0) }
  rows, err := model.db.Query(query, args...)
  if err != nil { return errors.Wrap(err, 0) }
  defer rows.Close()
  for rows.Next() {
    var id, title string
    err = rows.Scan(&id, &title)
    if err != nil { return errors.Wrap(err, 0) }
    task := j.Object()
    task.Prop("id", j.String(id))
    task.Prop("title", j.String(title))
    resp.Add("tasks."+id, task)
  }
  return nil
}

func keysOfSet(m map[string]bool) []string {
  keys := make([]string, len(m))
  i := 0
  for key := range m {
    keys[i] = key
    i++
  }
  return keys
}
