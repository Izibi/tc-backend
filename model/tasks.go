
package model

import (
  "database/sql"
  "fmt"
  "github.com/go-errors/errors"
  "github.com/jmoiron/sqlx"
  j "tezos-contests.izibi.com/backend/jase"
)

type Task struct {
  Id int64
  Created_at string
  Updated_at string
  Title string
}

type TaskResource struct {
  Id int64
  Created_at string
  Updated_at string
  Task_id int64
  Rank string
  Title string
  Description string
  Url string
  Html string
}

func (m *Model) loadTasks(ids []int64) error {
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

func (m *Model) loadTaskRow(row IRow) (*Task, error) {
  var res Task
  err := row.StructScan(&res)
  if err == sql.ErrNoRows { return nil, nil }
  if err != nil { return nil, errors.Wrap(err, 0) }
  view := j.Object()
  view.Prop("id", j.String(m.ExportId(res.Id)))
  view.Prop("title", j.String(res.Title))
  view.Prop("createdAt", j.String(res.Created_at))
  view.Prop("updatedAt", j.String(res.Updated_at))
  m.Add(fmt.Sprintf("tasks %s", m.ExportId(res.Id)), view)
  return &res, nil
}

func (m *Model) loadTaskResources(taskId int64) error {
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

func (m *Model) loadTaskResourceRow(row IRow) (*TaskResource, error) {
  var res TaskResource
  err := row.StructScan(&res)
  if err == sql.ErrNoRows { return nil, nil }
  if err != nil { return nil, errors.Wrap(err, 0) }
  view := j.Object()
  view.Prop("id", j.String(m.ExportId(res.Id)))
  view.Prop("createdAt", j.String(res.Created_at))
  view.Prop("updatedAt", j.String(res.Updated_at))
  view.Prop("taskId", j.String(m.ExportId(res.Task_id)))
  view.Prop("rank", j.String(res.Rank))
  view.Prop("title", j.String(res.Title))
  view.Prop("description", j.String(res.Description))
  view.Prop("url", j.String(res.Url))
  view.Prop("html", j.String(res.Html))
  m.Add(fmt.Sprintf("taskResources %s", m.ExportId(res.Id)), view)
  return &res, nil
}
