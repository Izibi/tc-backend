
package model

import (
  "database/sql"
  _ "github.com/go-sql-driver/mysql"
  j "tezos-contests.izibi.com/backend/jase"
)

type Model struct {
  db *sql.DB
  result j.IObject
  entities map[string]j.Value
  tasks LoadSet
}

func New (db *sql.DB) *Model {
  model := new(Model)
  model.db = db
  model.result = j.Object()
  model.entities = make(map[string]j.Value)
  return model
}

func (m *Model) Set(key string, value j.Value) {
  m.result.Prop(key, value)
}

func (m *Model) Add(key string, view j.Value) {
  m.entities[key] = view
}

func (m *Model) Has(key string) bool {
  _, ok := m.entities[key]
  return ok
}

func (m *Model) Result() j.IObject {
  return m.result
}

func (m *Model) Entities() j.IObject {
  entities := j.Object()
  for _, key := range orderedMapKeys(m.entities) {
    entities.Prop(key, m.entities[key])
  }
  return entities
}
