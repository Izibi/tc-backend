
package blocks

import (
  "fmt"
  "bytes"
  "encoding/json"
  "github.com/gin-gonic/gin"
  "tezos-contests.izibi.com/backend/utils"
  j "tezos-contests.izibi.com/backend/jase"
)

func SetupRoutes(r gin.IRoutes, newApi utils.NewApi, store *Store) {

  r.GET("/Blocks/:hash/zip", func (c *gin.Context) {
    fmt.Printf("A\n")
    var err error
    hash := c.Param("hash")
    if !store.IsBlock(hash) {
      c.String(404, "block not found")
      return
    }
    buf := new(bytes.Buffer)
    err = writeZip(store.blockDir(hash), buf)
    if err != nil { c.String(500, "packing error: %s", err) }
    c.Data(200, "application/zip", buf.Bytes())
  })

  r.POST("/Blocks/:parentHash/Task", func (c *gin.Context) {
    api := newApi(c)
    var err error
    var body struct {
      Identifier string `json:"identifier"`
      /* Consider: move task "tools" and "helper" from config.yaml to here:
      Task_tools_cmd string `json:"task_tools_cmd"`
      Task_helper_cmd string `json:"task_helper_cmd"`
      */
    }
    err = c.ShouldBindJSON(&body)
    if err != nil { api.Error(err); return }
    hash, err := store.MakeTaskBlock(c.Param("parentHash"), body.Identifier)
    if err != nil { api.Error(err); return }
    hashResponse(api, hash)
  })

  r.POST("/Blocks/:parentHash/Protocol", func (c *gin.Context) {
    api := newApi(c)
    var err error
    var body struct {
      Interface string `json:"interface"`
      Implementation string `json:"implementation"`
    }
    err = c.ShouldBindJSON(&body)
    if err != nil { api.Error(err); return }
    hash, err := store.MakeProtocolBlock(c.Param("parentHash"),
      []byte(body.Interface), []byte(body.Implementation))
    if err != nil { api.Error(err); return }
    hashResponse(api, hash)
  })

  r.POST("/Blocks/:parentHash/Setup", func (c *gin.Context) {
    api := newApi(c)
    var err error
    var body struct {
      Params json.RawMessage `json:"params"`
    }
    err = c.ShouldBindJSON(&body)
    if err != nil { api.Error(err); return }
    hash, err := store.MakeSetupBlock(c.Param("parentHash"), body.Params)
    if err != nil { api.Error(err); return }
    hashResponse(api, hash)
  })

  r.POST("/Blocks/:parentHash/Command", func (c *gin.Context) {
    api := newApi(c)
    var err error
    var body struct {
      Commands json.RawMessage `json:"commands"`
    }
    err = c.ShouldBindJSON(&body)
    if err != nil { api.Error(err); return }
    hash, err := store.MakeCommandBlock(c.Param("parentHash"), body.Commands)
    if err != nil { api.Error(err); return }
    hashResponse(api, hash)
  })

}

func hashResponse(api *utils.Response, hash string) {
  res := j.Object()
  res.Prop("hash", j.String(hash))
  api.Send(res)
}
