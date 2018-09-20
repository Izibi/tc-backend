package utils

import (
  "io"
  "net/http"
  "sort"

  "github.com/gin-gonic/gin"

  j "tezos-contests.izibi.com/backend/jase"
)

type Response struct {
  context *gin.Context
  result j.IObject
  entities map[string]j.Value
}

func NewResponse(c *gin.Context) *Response {
  return &Response{
    context: c,
    result: j.Object(),
    entities: map[string]j.Value{},
  }
}

func (r *Response) Set(key string, value j.Value) {
  r.result.Prop(key, value)
}

func (r *Response) Add(key string, view j.Value) {
  r.entities[key] = view
}

func (r *Response) Has(key string) bool {
  _, ok := r.entities[key]
  return ok
}

func (r *Response) Send() {
  r.context.Status(http.StatusOK)
  r.context.Stream(func (w io.Writer) bool {
    res := j.Object()
    res.Prop("result", r.result)
    entities := j.Object()
    res.Prop("entities", entities)
    for _, key := range orderedKeys(r.entities) {
      entities.Prop(key, r.entities[key])
    }
    res.Write(w)
    return false
  })
}

func orderedKeys(m map[string]j.Value) []string {
  keys := make([]string, len(m))
  i := 0
  for key := range m {
    keys[i] = key
    i++
  }
  sort.Strings(keys)
  return keys
}
