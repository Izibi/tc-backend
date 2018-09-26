
package model

import (
  "github.com/golang-collections/collections/set"
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

func keysOfSet(s *set.Set) []string {
  n := s.Len()
  if n == 0 { return nil }
  keys := make([]string, n)
  i := 0
  s.Do(func (key interface{}) {
    keys[i] = key.(string);
    i++
  })
  return keys
}
