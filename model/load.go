
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

func (m *Model) loadContestRow(row IRow) (string, error) {
  var id, title, description, logoUrl, taskId, startsAt, endsAt string
  var isRegistrationOpen bool
  err := row.Scan(&id, &title, &description, &logoUrl, &taskId, &isRegistrationOpen, &startsAt, &endsAt)
  if err != nil { return "", errors.Wrap(err, 0) }
  contest := j.Object()
  contest.Prop("id", j.String(id))
  contest.Prop("title", j.String(title))
  contest.Prop("description", j.String(description))
  contest.Prop("logoUrl", j.String(logoUrl))
  contest.Prop("taskId", j.String(taskId))
  contest.Prop("startsAt", j.String(startsAt))
  contest.Prop("endsAt", j.String(endsAt))
  m.tasks.Need(taskId)
  m.Add("contests."+id, contest)
  return id, nil
}

func (m *Model) loadTaskRow(row IRow) (string, error) {
  var id, title string
  err := row.Scan(&id, &title)
  if err != nil { return "", errors.Wrap(err, 0) }
  task := j.Object()
  task.Prop("id", j.String(id))
  task.Prop("title", j.String(title))
  m.Add("tasks."+id, task)
  return id, nil
}
