
package model

import (
  "github.com/go-errors/errors"
)

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

func (m *Model) LoadTaskResources(taskId int64) ([]TaskResource, error) {
  var taskResources []TaskResource
  err := m.dbMap.Select(&taskResources,
    `SELECT * FROM task_resources WHERE task_id = ?`, taskId)
  if err != nil { return nil, errors.Wrap(err, 0) }
  return taskResources, nil
}
