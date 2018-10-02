
package games

import (
  "io"
  "database/sql"
  "github.com/gin-gonic/gin"
  "tezos-contests.izibi.com/backend/model"
  "tezos-contests.izibi.com/backend/utils"
  "tezos-contests.izibi.com/backend/blocks"
  "tezos-contests.izibi.com/backend/signing"
  j "tezos-contests.izibi.com/backend/jase"
)

type Config struct {
  ApiKey string
}

func SetupRoutes(r gin.IRoutes, config Config, store *blocks.Store, db *sql.DB) {

  r.POST("/Games", func (c *gin.Context) {
    resp := utils.NewResponse(c)
    var err error
    var body struct {
      FirstBlock string `json:"first_block"`
    }
    err = c.ShouldBindJSON(&body)
    if err != nil { resp.Error(err); return }
    if !store.IsBlock(body.FirstBlock) {
      resp.StringError("bad first block")
      return
    }
    m := model.New(c, db)
    gameKey, err := m.CreateGame(body.FirstBlock)
    if err != nil { resp.Error(err); return }
    game, err := m.ViewGame(gameKey)
    if err != nil { resp.Error(err); return }
    resp.Send(game)
  })

  r.GET("/Games/:gameKey", func (c *gin.Context) {
    resp := utils.NewResponse(c)
    gameKey := c.Param("gameKey")
    m := model.New(c, db)
    game, err := m.ViewGame(gameKey)
    if err != nil { resp.Error(err); return }
    resp.Send(game)
  })

  r.POST("/Games/:gameKey/Commands", func (c *gin.Context) {
    resp := utils.NewResponse(c)
    var err error
    body, err := c.GetRawData()
    if err != nil { resp.Error(err); return }
    err = signing.Verify(config.ApiKey, body)
    if err != nil { resp.Error(err); return }
    var req struct {
      Author string `json:"author"`
      Player uint `json:"player"`
      Commands string `json:"commands"`
    }
    err = c.ShouldBindJSON(&req)
    if err != nil { resp.Error(err); return }
    gameKey := c.Param("gameKey")
    m := model.New(c, db)
    err = m.SetPlayerCommands(gameKey, req.Author[1:], req.Player, req.Commands)
    if err != nil { resp.Error(err); return }
    resp.Send(j.Null)
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

  r.POST("/Games/:gameKey/EndRound", func (c *gin.Context) {
    // endGameRound(config, req.body)
  })

}
