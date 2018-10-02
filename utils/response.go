
package utils

import (
  "fmt"
  "github.com/gin-gonic/gin"
  "github.com/fatih/color"
  j "tezos-contests.izibi.com/backend/jase"
)

var notice = color.New(color.Bold, color.FgGreen)

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
  notice.Print("-> ")
  fmt.Printf("%s\n", string(bs))
  r.context.Data(200, "application/json", bs)
}
