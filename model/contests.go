
package model

import (
  "database/sql"
  "fmt"
  "github.com/go-errors/errors"
  j "tezos-contests.izibi.com/backend/jase"
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

func (m *Model) ViewUserContests(userId int64) error {
  var err error
  rows, err := m.db.Queryx(
    `SELECT c.* FROM user_badges ub, contests c
     WHERE ub.user_id = ? AND ub.badge_id = c.required_badge_id`, userId)
  if err != nil { return errors.Wrap(err, 0) }
  defer rows.Close()
  contestIds := j.Array()
  for rows.Next() {
    contest, err := m.loadContestRow(rows, BaseFacet)
    if err != nil { return err }
    contestIds.Item(j.String(m.ExportId(contest.Id)))
    m.tasks.Need(contest.Task_id)
  }
  err = m.tasks.Load(m.loadTasks)
  if err != nil { return err }
  m.Set("contestIds", contestIds)
  return nil
}

func (m *Model) ViewUserContest(userId int64, contestId int64) error {
  var err error
  /* verify user has access to contest */
  ok, err := m.CanUserAccessContest(userId, contestId)
  if err != nil { return err }
  if !ok { return errors.Errorf("access denied") }

  /* load contest, task */
  contest, err := m.loadContest(contestId, BaseFacet)
  if err != nil { return err }
  m.tasks.Need(contest.Task_id)
  err = m.tasks.Load(m.loadTasks)
  if err != nil { return err }
  err = m.loadTaskResources(contest.Task_id)
  if err != nil { return err }

  return nil
}

func (m *Model) ViewUserContestTeam(userId int64, contestId int64) error {
  _, err := m.loadContest(contestId, BaseFacet)
  if err != nil { return err }
  team, err := m.LoadUserContestTeam(userId, contestId, Facets{Base: true, Member: true})
  if err != nil { return err }
  if team == nil {
    m.Set("teamId", j.Null)
    return nil
  }
  _, err = m.loadTeamMembers(team.Id, BaseFacet)
  if err != nil { return err }
  m.Set("teamId", j.String(m.ExportId(team.Id)))
  return nil
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

func (m *Model) loadContest(contestId int64, f Facets) (*Contest, error) {
  return m.loadContestRow(m.db.QueryRowx(
    `SELECT * FROM contests WHERE id = ?`, contestId), f)
}

func (m *Model) loadContestRow(row IRow, f Facets) (*Contest, error) {
  var res Contest
  err := row.StructScan(&res)
  if err == sql.ErrNoRows { return nil, nil }
  if err != nil { return nil, errors.Wrap(err, 0) }
  if f.Base {
    view := j.Object()
    view.Prop("id", j.String(m.ExportId(res.Id)))
    view.Prop("createdAt", j.String(res.Created_at))
    view.Prop("updatedAt", j.String(res.Updated_at))
    view.Prop("title", j.String(res.Title))
    view.Prop("description", j.String(res.Description))
    view.Prop("logoUrl", j.String(res.Logo_url))
    view.Prop("taskId", j.String(m.ExportId(res.Task_id)))
    view.Prop("startsAt", j.String(res.Starts_at))
    view.Prop("endsAt", j.String(res.Ends_at))
    m.Add(fmt.Sprintf("contests %s", m.ExportId(res.Id)), view)
  }
  return &res, nil
}
