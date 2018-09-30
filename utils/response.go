
package utils

import (
  "io"
  "net/http"

  "github.com/gin-gonic/gin"

  j "tezos-contests.izibi.com/backend/jase"
  "tezos-contests.izibi.com/backend/model"
)

type Response struct {
  context *gin.Context
}

func NewResponse(c *gin.Context) *Response {
  return &Response{
    context: c,
  }
}

func (r *Response) Send(m *model.Model) {
  r.context.Status(http.StatusOK)
  r.context.Header("Content-Type", "application/json")
  r.context.Stream(func (w io.Writer) bool {
    res := j.Object()
    res.Prop("result", m.Result())
    res.Prop("entities", m.Entities())
    res.WriteTo(w)
    return false
  })
}
