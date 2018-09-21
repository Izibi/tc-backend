
package model

import (
  "sort"
  "github.com/golang-collections/collections/set"
  j "tezos-contests.izibi.com/backend/jase"
)

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
