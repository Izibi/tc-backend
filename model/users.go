
package model

import (
  "database/sql"
  "github.com/go-errors/errors"
  "github.com/jmoiron/sqlx"
)

type User struct {
  Id int64
  Foreign_id string
  Created_at string
  Updated_at string
  Is_admin bool
  Username string
  Firstname string
  Lastname string
}

type UserProfile interface {
  Id() string
  Username() string
  Firstname() string
  Lastname() string
  Badges() []string
}

func (m *Model) LoadUser(id int64) (*User, error) {
  var user User
  err := m.dbMap.Get(&user, id)
  if err == sql.ErrNoRows { return nil, nil }
  if err != nil { return nil, errors.Wrap(err, 0) }
  return &user, nil
}

func (m *Model) LoadUsersById(ids []int64) ([]User, error) {
  var users []User
  query, args, err := sqlx.In(`SELECT * FROM users WHERE id IN (?)`, ids)
  if err != nil { return nil, errors.Wrap(err, 0) }
  err = m.dbMap.Select(&users, query, args...)
  if err != nil { return nil, errors.Wrap(err, 0) }
  return users, nil
}

func (m *Model) FindUserByForeignId(foreignId string) (int64, error) {
  var id int64
  err := m.db.QueryRow(`SELECT id FROM users WHERE foreign_id = ?`, foreignId).Scan(&id)
  if err != nil { return 0, err }
  return id, nil
}

func (m *Model) ImportUserProfile(profile UserProfile) (int64, error) {
  var userId int64
  foreignId := profile.Id()
  rows, err := m.db.Query(`SELECT id FROM users WHERE foreign_id = ?`, foreignId)
  if err != nil { return 0, errors.Wrap(err, 0) }
  if rows.Next() {
    err = rows.Scan(&userId)
    rows.Close()
    if err != nil { return 0, errors.Wrap(err, 0) }
    _, err := m.db.Exec(
      `UPDATE users SET username = ?, firstname = ?, lastname = ? WHERE id = ?`,
      profile.Username(), profile.Firstname(), profile.Lastname(), userId)
    if err != nil { return 0, errors.Wrap(err, 0) }
  } else {
    rows.Close()
    res, err := m.db.Exec(
      `INSERT INTO users (foreign_id, username, firstname, lastname) VALUES (?, ?, ?, ?)`,
      foreignId, profile.Username(), profile.Firstname(), profile.Lastname())
    if err != nil { return 0, errors.Wrap(err, 0) }
    userId, err = res.LastInsertId()
    if err != nil { return 0, errors.Wrap(err, 0) }
  }
  err = m.UpdateBadges(userId, profile.Badges())
  if err != nil { return 0, err }
  return userId, nil
}

func (m *Model) UpdateBadges(userId int64, badges []string) error {

  /* If the user holds no badges, delete all badges unconditionnally. */
  if len(badges) == 0 {
    _, err := m.db.Exec(`DELETE FROM user_badges
      WHERE user_badges.user_id = ?`, userId)
    if err != nil { return errors.Wrap(err, 0) }
    return nil
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

func (m *Model) IsUserAdmin(userId int64) bool {
  user, err := m.LoadUser(userId)
  if err != nil || user == nil { return false }
  return user.Is_admin
}
