
package utils

import (
  //"fmt"
  "encoding/json"
  "github.com/gin-gonic/gin"
  "tezos-contests.izibi.com/backend/signing"
)

type Request struct {
  context *gin.Context
  apiKey string
}

func NewRequest(c *gin.Context, apiKey string) *Request {
  return &Request{c, apiKey}
}

func (r *Request) Plain(req interface{}) error {  // XXX prefer context.BindJSON?
  body, err := r.context.GetRawData()
  if err != nil { return err }
  return json.Unmarshal(body, req)
}

func (r *Request) Signed(req interface{}) error {
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

func (r *Request) logRequestBody(bs []byte) {
  /*
    notice.Print("<- ") // XXX from response.go
    fmt.Printf("%s\n", string(bs))
  */
}
