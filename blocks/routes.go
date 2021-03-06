
package blocks

import (
  "bytes"
  "encoding/json"
  "github.com/gin-gonic/gin"
  "tezos-contests.izibi.com/backend/utils"
  j "tezos-contests.izibi.com/backend/jase"
)

type Context struct {
  resp *utils.Response
}

func (svc *Service) Wrap(c *gin.Context) *Context {
  return &Context{
    utils.NewResponse(c),
  }
}

func (svc *Service) Route(r gin.IRoutes) {

  r.GET("/Blocks/:hash/zip", func (c *gin.Context) {
    var err error
    hash := c.Param("hash")
    if !svc.IsBlock(hash) {
      c.String(404, "block not found")
      return
    }
    buf := new(bytes.Buffer)
    err = writeZip(svc.blockDir(hash), buf)
    if err != nil { c.String(500, "packing error: %s", err) }
    c.Data(200, "application/zip", buf.Bytes())
  })

  r.POST("/Blocks/:parentHash/Task", func (c *gin.Context) {
    ctx := svc.Wrap(c)
    var err error
    var body struct {
      Identifier string `json:"identifier"`
      Revision uint64 `json:"revision"`
      /* Consider: move task "tools" and "helper" from config.yaml to here:
      Task_tools_cmd string `json:"task_tools_cmd"`
      Task_helper_cmd string `json:"task_helper_cmd"`
      */
    }
    err = c.ShouldBindJSON(&body)
    if err != nil { ctx.resp.Error(err); return }
    hash, err := svc.MakeTaskBlock(c.Param("parentHash"), body.Identifier, body.Revision)
    if err != nil { ctx.resp.Error(err); return }
    ctx.HashResponse(hash)
  })

  r.POST("/Blocks/:parentHash/Protocol", func (c *gin.Context) {
    ctx := svc.Wrap(c)
    var err error
    var body struct {
      Interface string `json:"interface"`
      Implementation string `json:"implementation"`
    }
    err = c.ShouldBindJSON(&body)
    if err != nil { ctx.resp.Error(err); return }
    hash, err := svc.MakeProtocolBlock(c.Param("parentHash"),
      []byte(body.Interface), []byte(body.Implementation))
    if err != nil { ctx.resp.Error(err) }
    ctx.HashResponse(hash)
  })

  r.POST("/Blocks/:parentHash/Setup", func (c *gin.Context) {
    ctx := svc.Wrap(c)
    var err error
    var body struct {
      Params json.RawMessage `json:"params"`
    }
    err = c.ShouldBindJSON(&body)
    if err != nil { ctx.resp.Error(err); return }
    hash, err := svc.MakeSetupBlock(c.Param("parentHash"), body.Params)
    if err != nil { ctx.resp.Error(err) }
    ctx.HashResponse(hash)
  })

  r.POST("/Blocks/:parentHash/Command", func (c *gin.Context) {
    ctx := svc.Wrap(c)
    var err error
    var body struct {
      Commands json.RawMessage `json:"commands"`
    }
    err = c.ShouldBindJSON(&body)
    if err != nil { ctx.resp.Error(err); return }
    hash, err := svc.MakeCommandBlock(c.Param("parentHash"), body.Commands)
    if err != nil { ctx.resp.Error(err); return }
    ctx.HashResponse(hash)
  })

}

func (ctx *Context) HashResponse(hash string) {
  res := j.Object()
  res.Prop("hash", j.String(hash))
  ctx.resp.Send(res)
}
