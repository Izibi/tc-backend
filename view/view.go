
package view

import (
  "sort"
  "strconv"
  j "tezos-contests.izibi.com/backend/jase"
  "tezos-contests.izibi.com/backend/model"
)

type View struct {
  model *model.Model

  userId int64
  teamId int64
  contestId int64
  isAdmin bool

  tasks LoadSet
  users LoadSet
  teams LoadSet
  teamMembers LoadSet

  result j.IObject
  entities map[string]j.Value
}

func New(model *model.Model) *View {
  return &View{
    model: model,
    result: j.Object(),
    entities: make(map[string]j.Value),
  }
}

func (v *View) Set(key string, value j.Value) {
  v.result.Prop(key, value)
}

func (v *View) Add(key string, view j.Value) {
  v.entities[key] = view
}

func (v *View) Has(key string) bool {
  _, ok := v.entities[key]
  return ok
}

func (v *View) Flat() j.Value {
  res := j.Object()
  res.Prop("result", v.result)
  entities := j.Object()
  for _, key := range orderedMapKeys(v.entities) {
    entities.Prop(key, v.entities[key])
  }
  res.Prop("entities", entities)
  return res
}

func orderedMapKeys(m map[string]j.Value) []string {
  keys := make([]string, len(m))
  i := 0
  for key := range m {
    keys[i] = key
    i++
  }
  sort.Strings(keys)
  return keys
}

func ImportId(id string) int64 {
  n, err := strconv.ParseInt(id, 10, 64)
  if err != nil { return 0 }
  return n
}

func ExportId(id int64) string {
  return strconv.FormatInt(id, 10)
}
