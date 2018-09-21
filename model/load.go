
package model

import (
  "github.com/go-errors/errors"
  "github.com/golang-collections/collections/set"
  j "tezos-contests.izibi.com/backend/jase"
)

type LoadSet struct {
  loaded *set.Set
  toLoad *set.Set
}

func (s *LoadSet) Need(id string) {
  if s.toLoad == nil { s.toLoad = set.New() }
  s.toLoad.Insert(id)
}

func (s *LoadSet) Load(loader func (ids []string) error) error {
  if s.toLoad == nil { s.toLoad = set.New() }
  ids := keysOfSet(s.toLoad)
  if len(ids) == 0 { return nil }
  err := loader(ids)
  if err != nil { return err }
  if s.loaded == nil {
    s.loaded = s.toLoad
  } else {
    s.loaded = s.loaded.Union(s.toLoad)
  }
  s.toLoad = set.New()
  return nil
}

/*** load from contests ***/

type Contest struct {
  Id string
  Title string
  Description string
  Logo_url string
  Task_id string
  Is_registration_open bool
  Starts_at string
  Ends_at string
  Required_badge_id string
}

func (m *Model) loadContestRow(row IRow) (*Contest, error) {
  var contest Contest
  err := row.StructScan(&contest)
  if err != nil { return nil, errors.Wrap(err, 0) }
  view := j.Object()
  view.Prop("id", j.String(contest.Id))
  view.Prop("title", j.String(contest.Title))
  view.Prop("description", j.String(contest.Description))
  view.Prop("logoUrl", j.String(contest.Logo_url))
  view.Prop("taskId", j.String(contest.Task_id))
  view.Prop("startsAt", j.String(contest.Starts_at))
  view.Prop("endsAt", j.String(contest.Ends_at))
  m.Add("contests."+contest.Id, view)
  return &contest, nil
}

/*** load from tasks ***/

type Task struct {
  Id string
  Title string
  Created_at string
}

func (m *Model) loadTaskRow(row IRow) (*Task, error) {
  var task Task
  err := row.StructScan(&task)
  if err != nil { return nil, errors.Wrap(err, 0) }
  view := j.Object()
  view.Prop("id", j.String(task.Id))
  view.Prop("title", j.String(task.Title))
  m.Add("tasks."+task.Id, view)
  return &task, nil
}
