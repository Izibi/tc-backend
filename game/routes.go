
package game

import (
  "io"
  //"fmt"
  "time"
  "github.com/gin-gonic/gin"
  "github.com/Masterminds/semver"
)

func SetupRoutes(r gin.IRoutes, config Config) {

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
    c.String(200, time.Now().Format(time.RFC3339))
  })

  r.POST("/Games", func (c *gin.Context) {
    // newGame(config, req.body)
  })

  r.GET("/Games/:gameKey", func (c *gin.Context) {
    // showGame(config, req.params.gameKey)
  })

  r.GET("/Games/:gameKey/events", func (c *gin.Context) {
    c.Header("Content-Type", "text/event-stream")
    c.Header("Cache-Control", "no-cache")
    c.Header("Connection", "keep-alive")
    c.Stream(func (w io.Writer) bool {
      w.Write([]byte("retry: 30000\n\n"))
      // registerGameSink(req.params.gameKey, w);
      return true
    })
  })

  r.POST("/Games/Commands", func (c *gin.Context) {
    // inputCommands(config, req.body)
  })

  r.POST("/Games/EndRound", func (c *gin.Context) {
    // endGameRound(config, req.body)
  })

  r.POST("/Protocols", func (c *gin.Context) {
    var err error
    type ProtocolBody struct {
      Interface string `json:"interface"`
      Implementation string `json:"implementation"`
    }
    var body ProtocolBody
    err = c.ShouldBindJSON(&body)
    if err != nil { c.String(400, err.Error()) }
    hash, err := makeProtocolBlock(config, body.Interface, body.Implementation)
    if err != nil {
      c.String(400, err.Error())
      return
    }
    c.String(200, hash)
    // result := makeProtocolBlock(config, req.body)
  })

  r.POST("/Commands", func (c *gin.Context) {
    // const {chain, commands} = req.body
    // checkCommands(config, chain, commands)
  })

  r.POST("/Keypair", func (c *gin.Context) {
    // ssbKeys.generate()
  })

}
