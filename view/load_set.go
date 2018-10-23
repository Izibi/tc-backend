
package view

import (
  "github.com/golang-collections/collections/set"
)

type LoadSet struct {
  loaded *set.Set
  toLoad *set.Set
}

func (s *LoadSet) Need(id int64) {
  if s.toLoad == nil { s.toLoad = set.New() }
  s.toLoad.Insert(id)
}

func (s *LoadSet) Load(loader func (ids []int64) error) error {
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

func keysOfSet(s *set.Set) []int64 {
  n := s.Len()
  if n == 0 { return nil }
  keys := make([]int64, n)
  i := 0
  s.Do(func (key interface{}) {
    keys[i] = key.(int64);
    i++
  })
  return keys
}
