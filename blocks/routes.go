
package blocks

import (
  "encoding/json"
  "github.com/gin-gonic/gin"
  "tezos-contests.izibi.com/backend/utils"
)

type Config struct {
  ApiVersion string `yaml:"api_version"`
}

func SetupRoutes(r gin.IRoutes, store *Store) {

  r.POST("/Blocks/:parentHash/Task", func (c *gin.Context) {
    resp := utils.NewResponse(c)
    var err error
    var body struct {
      Identifier string `json:"identifier"`
      /* Consider: move task "tools" and "helper" from config.yaml to here:
      Task_tools_cmd string `json:"task_tools_cmd"`
      Task_helper_cmd string `json:"task_helper_cmd"`
      */
    }
    err = c.ShouldBindJSON(&body)
    if err != nil { resp.Error(err); return }
    hash, err := store.MakeTaskBlock(c.Param("parentHash"), body.Identifier)
    if err != nil { resp.Error(err); return }
    hashResponse(c, hash)
  })

  r.POST("/Blocks/:parentHash/Protocol", func (c *gin.Context) {
    resp := utils.NewResponse(c)
    var err error
    var body struct {
      Interface string `json:"interface"`
      Implementation string `json:"implementation"`
    }
    err = c.ShouldBindJSON(&body)
    if err != nil { resp.Error(err); return }
    hash, err := store.MakeProtocolBlock(c.Param("parentHash"),
      []byte(body.Interface), []byte(body.Implementation))
    if err != nil { resp.Error(err); return }
    hashResponse(c, hash)
  })

  r.POST("/Blocks/:parentHash/Setup", func (c *gin.Context) {
    resp := utils.NewResponse(c)
    var err error
    var body struct {
      Params json.RawMessage `json:"params"`
    }
    err = c.ShouldBindJSON(&body)
    if err != nil { resp.Error(err); return }
    hash, err := store.MakeSetupBlock(c.Param("parentHash"), body.Params)
    if err != nil { resp.Error(err); return }
    hashResponse(c, hash)
  })

  r.POST("/Blocks/:parentHash/Command", func (c *gin.Context) {
    resp := utils.NewResponse(c)
    var err error
    var body struct {
      Nb_cycles uint
      Commands json.RawMessage `json:"commands"`
    }
    err = c.ShouldBindJSON(&body)
    if err != nil { resp.Error(err); return }
    hash, err := store.MakeCommandBlock(c.Param("parentHash"), body.Nb_cycles, body.Commands)
    if err != nil { resp.Error(err); return }
    hashResponse(c, hash)
  })

}

func hashResponse(c *gin.Context, hash string) {
  type Response struct {
    Hash string `json:"hash"`
  }
  c.JSON(200, &Response{Hash: hash})
}
