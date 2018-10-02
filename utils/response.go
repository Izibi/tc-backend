
/*
  Helper file to handle API requests/responses in a uniform way.
  TODO: move to api package and rename Response to API
*/

package utils

import (
  "fmt"
  "encoding/json"
  "github.com/gin-gonic/gin"
  "github.com/fatih/color"
  "tezos-contests.izibi.com/backend/signing"
  j "tezos-contests.izibi.com/backend/jase"
)

var notice = color.New(color.Bold, color.FgGreen)

type ModelResponse interface {
  Result() j.IObject
  Entities() j.IObject
}

type Response struct {
  context *gin.Context
  apiKey string
}

type NewApi func (c *gin.Context) *Response

func NewResponse(c *gin.Context, apiKey string) *Response {
  return &Response{
    context: c,
    apiKey: apiKey,
  }
}

func (r *Response) Send(data j.Value) {
  bs, err := j.ToBytes(data)
  if err != nil { r.Error(err); return }
  notice.Print("-> ")
  fmt.Printf("%s\n", string(bs))
  r.context.Data(200, "application/json", bs)
}

func (r *Response) Result(val j.Value) {
  res := j.Object()
  res.Prop("result", val)
  r.Send(res)
}

func (r *Response) logRequestBody(bs []byte) {
  notice.Print("<- ")
  fmt.Printf("%s\n", string(bs))
}

func (r *Response) SignedRequest(req interface{}) error {
  // TODO: check r.context.ContentType() is "application/json"
  body, err := r.context.GetRawData()
  if err != nil { return err }
  r.logRequestBody(body)
  err = signing.Verify(r.apiKey, body)
  if err != nil { return err }
  err = json.Unmarshal(body, req)
  if err != nil { return err }
  return nil
}
