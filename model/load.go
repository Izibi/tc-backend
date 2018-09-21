
package model

import (
  "database/sql"
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

/*** load from users ***/

type User struct {
  Id string
  Foreign_id string
  Created_at string
  Updated_at string
  Is_admin bool
  Username string
  Firstname string
  Lastname string
}

func (m *Model) loadUserRow(row IRow, f Facets) (*User, error) {
  var res User
  err := row.StructScan(&res)
  if err == sql.ErrNoRows { return nil, nil }
  if err != nil { return nil, errors.Wrap(err, 0) }
  if f.Base {
    view := j.Object()
    view.Prop("id", j.String(res.Id))
    view.Prop("username", j.String(res.Username))
    view.Prop("firstname", j.String(res.Firstname))
    view.Prop("lastname", j.String(res.Lastname))
    m.Add(fmt.Sprintf("users %s", res.Id), view)
  }
  if f.Admin {
    view := j.Object()
    view.Prop("foreignId", j.String(res.Foreign_id))
    view.Prop("createdAt", j.String(res.Created_at))
    view.Prop("updatedAt", j.String(res.Updated_at))
    view.Prop("isAdmin", j.Boolean(res.Is_admin))
    m.Add(fmt.Sprintf("users#admin %s", res.Id), view)
  }
  return &res, nil
}

/*** load from teams ***/

type Team struct {
  Id string
  Created_at string
  Access_code string
  Contest_id string
  Is_open bool
  Is_locked bool
  Name string
  Public_key sql.NullString
}

func (m *Model) loadTeamRow(row IRow, f Facets) (*Team, error) {
  var res Team
  err := row.StructScan(&res)
  if err == sql.ErrNoRows { return nil, nil }
  if err != nil { return nil, errors.Wrap(err, 0) }
  if f.Base {
    view := j.Object()
    view.Prop("id", j.String(res.Id))
    view.Prop("createdAt", j.String(res.Created_at))
    view.Prop("contestId", j.String(res.Contest_id))
    view.Prop("isOpen", j.Boolean(res.Is_open))
    view.Prop("isLocked", j.Boolean(res.Is_locked))
    view.Prop("name", j.String(res.Name))
    publicKey := j.Null
    if res.Public_key.Valid {
      publicKey = j.String(res.Public_key.String)
    }
    view.Prop("publicKey", publicKey)
    m.Add(fmt.Sprintf("teams %s", res.Id), view)
  }
  if f.Member {
    view := j.Object()
    view.Prop("accessCode", j.String(res.Access_code))
    m.Add(fmt.Sprintf("teams#member %s", res.Id), view)
  }
  return &res, nil
}

/*** load from team_members ***/

type TeamMember struct {
  Team_id string
  User_id string
  Joined_at string
  Is_creator bool
}

func (m *Model) loadTeamMemberRow(row IRow, f Facets) (*TeamMember, error) {
  var res TeamMember
  err := row.StructScan(&res)
  if err == sql.ErrNoRows { return nil, nil }
  if err != nil { return nil, errors.Wrap(err, 0) }
  if f.Base {
    view := j.Object()
    view.Prop("teamId", j.String(res.Team_id))
    view.Prop("userId", j.String(res.User_id))
    view.Prop("joinedAt", j.String(res.Joined_at))
    view.Prop("isCreator", j.Boolean(res.Is_creator))
    m.Add(fmt.Sprintf("teamMembers %s.%s", res.Team_id, res.User_id), view)
  }
  return &res, nil
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

func (m *Model) loadContestRow(row IRow, f Facets) (*Contest, error) {
  var res Contest
  err := row.StructScan(&res)
  if err == sql.ErrNoRows { return nil, nil }
  if err != nil { return nil, errors.Wrap(err, 0) }
  if f.Base {
    view := j.Object()
    view.Prop("id", j.String(res.Id))
    view.Prop("title", j.String(res.Title))
    view.Prop("description", j.String(res.Description))
    view.Prop("logoUrl", j.String(res.Logo_url))
    view.Prop("taskId", j.String(res.Task_id))
    view.Prop("startsAt", j.String(res.Starts_at))
    view.Prop("endsAt", j.String(res.Ends_at))
    m.Add(fmt.Sprintf("contests %s", res.Id), view)
  }
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
  if err == sql.ErrNoRows { return nil, nil }
  if err != nil { return nil, errors.Wrap(err, 0) }
  view := j.Object()
  view.Prop("id", j.String(res.Id))
  view.Prop("title", j.String(res.Title))
  m.Add(fmt.Sprintf("tasks %s", res.Id), view)
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
  if err == sql.ErrNoRows { return nil, nil }
  if err != nil { return nil, errors.Wrap(err, 0) }
  view := j.Object()
  view.Prop("id", j.String(res.Id))
  view.Prop("taskId", j.String(res.Task_id))
  view.Prop("rank", j.String(res.Rank))
  view.Prop("title", j.String(res.Title))
  view.Prop("description", j.String(res.Description))
  view.Prop("url", j.String(res.Url))
  view.Prop("html", j.String(res.Html))
  m.Add(fmt.Sprintf("taskResources %s", res.Id), view)
  return &res, nil
}
