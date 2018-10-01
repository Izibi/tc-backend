
package utils

import (
  "github.com/gin-gonic/gin"
  j "tezos-contests.izibi.com/backend/jase"
)

type ModelResponse interface {
  Result() j.IObject
  Entities() j.IObject
}

type Response struct {
  context *gin.Context
}

func NewResponse(c *gin.Context) *Response {
  return &Response{
    context: c,
  }
}

func (r *Response) Send(data j.Value) {
  bs, err := j.ToBytes(data)
  if err != nil { r.Error(err); return }
  r.context.Data(200, "application/json", bs)
}
