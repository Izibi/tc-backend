
package model

import (
  "github.com/go-errors/errors"
)

type Contest struct {
  Id int64
  Created_at string
  Updated_at string
  Title string
  Description string
  Logo_url string
  Task_id int64
  Is_registration_open bool
  Starts_at string
  Ends_at string
  Required_badge_id int64
  // Contest_period_id string
}

func (m *Model) LoadContest(id int64) (*Contest, error) {
  var err error
  var contest Contest
  err = m.dbMap.Get(&contest, id)
  if err != nil { return nil, errors.Wrap(err, 0) }
  return &contest, nil
}


func (m *Model) CanUserAccessContest(userId int64, contestId int64) (bool, error) {
  row := m.db.QueryRow(
    `SELECT count(c.id) FROM user_badges ub, contests c
     WHERE c.id = ? AND ub.user_id = ? AND ub.badge_id = c.required_badge_id`, contestId, userId)
  var count int
  err := row.Scan(&count)
  if err != nil { return false, errors.Wrap(err, 0) }
  return count == 1, nil
}

func (m *Model) LoadUserContests(userId int64) ([]Contest, error) {
  var err error
  var contests []Contest
  err = m.dbMap.Select(&contests,
    `SELECT c.* FROM user_badges ub, contests c
     WHERE ub.user_id = ? AND ub.badge_id = c.required_badge_id`, userId)
  if err != nil { return nil, errors.Wrap(err, 0) }
  return contests, nil
}
