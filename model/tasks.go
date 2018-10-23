
package model

import (
  "github.com/go-errors/errors"
  "github.com/jmoiron/sqlx"
)

type Task struct {
  Id int64
  Created_at string
  Updated_at string
  Title string
}

func (m *Model) LoadTasksById(ids []int64) ([]Task, error) {
  var tasks []Task
  query, args, err := sqlx.In(`SELECT * FROM tasks WHERE id IN (?)`, ids)
  if err != nil { return nil, errors.Wrap(err, 0) }
  err = m.dbMap.Select(&tasks, query, args...)
  if err != nil { return nil, errors.Wrap(err, 0) }
  return tasks, nil
}
