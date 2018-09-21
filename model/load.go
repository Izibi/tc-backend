
package model

import (
  "fmt"
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
  var res Contest
  err := row.StructScan(&res)
  if err != nil { return nil, errors.Wrap(err, 0) }
  view := j.Object()
  view.Prop("id", j.String(res.Id))
  view.Prop("title", j.String(res.Title))
  view.Prop("description", j.String(res.Description))
  view.Prop("logoUrl", j.String(res.Logo_url))
  view.Prop("taskId", j.String(res.Task_id))
  view.Prop("startsAt", j.String(res.Starts_at))
  view.Prop("endsAt", j.String(res.Ends_at))
  m.Add(fmt.Sprintf("contests.%s", res.Id), view)
  return &res, nil
}

/*** load from tasks ***/

type Task struct {
  Id string
  Title string
  Created_at string
}

func (m *Model) loadTaskRow(row IRow) (*Task, error) {
  var res Task
  err := row.StructScan(&res)
  if err != nil { return nil, errors.Wrap(err, 0) }
  view := j.Object()
  view.Prop("id", j.String(res.Id))
  view.Prop("title", j.String(res.Title))
  m.Add(fmt.Sprintf("tasks.%s", res.Id), view)
  return &res, nil
}

/*** load from task_resources ***/

type TaskResource struct {
  Id string
  Task_id string
  Rank string
  Title string
  Description string
  Url string
  Html string
}

func (m *Model) loadTaskResourceRow(row IRow) (*TaskResource, error) {
  var res TaskResource
  err := row.StructScan(&res)
  if err != nil { return nil, errors.Wrap(err, 0) }
  view := j.Object()
  view.Prop("id", j.String(res.Id))
  view.Prop("taskId", j.String(res.Task_id))
  view.Prop("rank", j.String(res.Rank))
  view.Prop("title", j.String(res.Title))
  view.Prop("description", j.String(res.Description))
  view.Prop("url", j.String(res.Url))
  view.Prop("html", j.String(res.Html))
  m.Add(fmt.Sprintf("taskResources.%s", res.Id), view)
  return &res, nil
}
