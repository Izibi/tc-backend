
package game

import (
  "io"
  "encoding/json"
  //"fmt"
  "time"
  "github.com/gin-gonic/gin"
  "github.com/Masterminds/semver"
  "tezos-contests.izibi.com/backend/utils"
  "tezos-contests.izibi.com/backend/blockchain"
)

type Config struct {
  ApiVersion string `yaml:"api_version"`
}

func SetupRoutes(r gin.IRoutes, config Config, store *blockchain.Store) {

  apiVersion := semver.MustParse(config.ApiVersion)

  r.GET("/Time", func (c *gin.Context) {
    reqVersion := c.GetHeader("X-Api-Version")
    req, err := semver.NewConstraint(reqVersion)
    if err != nil {
      c.String(400, "Client sent a bad semver constraint")
      return;
    }
    if !req.Check(apiVersion) {
      c.String(400, "Client is incompatible with Server API %s", apiVersion)
      return
    }
    type Response struct {
      ServerTime string `json:"server_time"`
    }
    res := Response{ServerTime: time.Now().Format(time.RFC3339)}
    c.JSON(200, &res)
  })

  r.POST("/Games", func (c *gin.Context) {
    // newGame(config, req.body)
  })

  r.GET("/Games/:gameKey", func (c *gin.Context) {
    // showGame(config, req.params.gameKey)
  })

  r.GET("/Games/:gameKey/Events", func (c *gin.Context) {
    c.Header("Content-Type", "text/event-stream")
    c.Header("Cache-Control", "no-cache")
    c.Header("Connection", "keep-alive")
    c.Header("X-Accel-Buffering", "no")
    c.Stream(func (w io.Writer) bool {
      w.Write([]byte("retry: 30000\n\n"))
      // TODO: verify the game actually exists
      // TODO: this is wrong, teams should register for events (with a signed message)
      // registerGameSink(c.Param("gameKey"), w);
      return true
    })
  })

  r.POST("/Games/:gameKey/Commands", func (c *gin.Context) {
    // inputCommands(config, req.body)
  })

  r.POST("/Games/:gameKey/EndRound", func (c *gin.Context) {
    // endGameRound(config, req.body)
  })

  r.POST("/Blocks/:parentHash/Task", func (c *gin.Context) {
    resp := utils.NewResponse(c)
    var err error
    var body struct {
      Identifier string `json:"identifier"`
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

  r.POST("/Keypair", func (c *gin.Context) {
    // ssbKeys.generate()
  })

}

func hashResponse(c *gin.Context, hash string) {
  type Response struct {
    Hash string `json:"hash"`
  }
  c.JSON(200, &Response{Hash: hash})
}
