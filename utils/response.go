
package utils

import (
  //"fmt"
  "github.com/gin-gonic/gin"
  "github.com/fatih/color"
  "github.com/go-errors/errors"
  j "tezos-contests.izibi.com/backend/jase"
)

var notice = color.New(color.Bold, color.FgGreen) // XXX move to separate file

type Response struct {
  context *gin.Context
}

func NewResponse(c *gin.Context) *Response {
  return &Response{c}
}

func (r *Response) Send(data j.Value) {
  bs, err := j.ToBytes(data)
  if err != nil { r.Error(err); return }
  // notice.Print("-> ")
  // fmt.Printf("%s\n", string(bs))
  r.context.Data(200, "application/json", bs)
}

func (r *Response) Result(val j.Value) {
  res := j.Object()
  res.Prop("result", val)
  r.Send(res)
}

func (r *Response) Error(err error) {
  res := j.Object()
  res.Prop("error", j.String(err.Error()))
  err2, ok := err.(*errors.Error)
  if ok {
    res.Prop("location", j.String(traceLocation(err2.ErrorStack())))
  }
  r.Send(res)
}

func (r *Response) StringError(msg string) {
  r.Error(errors.Wrap(msg, 1))
}

func (r *Response) BadUser() {
  r.StringError("you don't exist")
}
