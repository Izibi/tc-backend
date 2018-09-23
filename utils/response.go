
package utils

import (
  "io"
  "net/http"

  "github.com/gin-gonic/gin"
  "github.com/utrack/gin-csrf"

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
  r.context.Stream(func (w io.Writer) bool {
    res := j.Object()
    res.Prop("result", m.Result())
    res.Prop("entities", m.Entities())
    /* Automatically send the CSRF token to GET requests. */
    if r.context.Request.Method == "GET" {
      res.Prop("csrfToken", j.String(csrf.GetToken(r.context)))
    }
    res.Write(w)
    return false
  })
}
