
package games

import (
  "io"
  "fmt"
  "database/sql"
  "github.com/gin-gonic/gin"
  "tezos-contests.izibi.com/backend/model"
  "tezos-contests.izibi.com/backend/utils"
  "tezos-contests.izibi.com/backend/blocks"
  j "tezos-contests.izibi.com/backend/jase"
)

type Config struct {
  ApiKey string
}

func SetupRoutes(r gin.IRoutes, newApi utils.NewApi, config Config, store *blocks.Store, db *sql.DB) {

  r.POST("/Games", func (c *gin.Context) {
    api := newApi(c)
    var err error
    var req struct {
      Author string `json:"author"`
      FirstBlock string `json:"first_block"`
    }
    err = api.SignedRequest(&req)
    if err != nil { api.Error(err); return }
    fmt.Printf("new game request %v\n", req)
    if !store.IsBlock(req.FirstBlock) {
      api.StringError("bad first block")
      return
    }
    m := model.New(c, db)
    ownerId, err := m.FindTeamIdByKey(req.Author[1:])
    if err != nil { api.Error(err); return }
    gameKey, err := m.CreateGame(ownerId, req.FirstBlock)
    if err != nil { api.Error(err); return }
    game, err := m.ViewGame(gameKey)
    if err != nil { api.Error(err); return }
    api.Result(game)
  })

  r.GET("/Games/:gameKey", func (c *gin.Context) {
    api := newApi(c)
    gameKey := c.Param("gameKey")
    m := model.New(c, db)
    game, err := m.ViewGame(gameKey)
    if err != nil { api.Error(err); return }
    api.Result(game)
  })

  r.POST("/Games/:gameKey/Commands", func (c *gin.Context) {
    api := newApi(c)
    var err error
    var req struct {
      Author string `json:"author"`
      CurrentBlock string `json:"current_block"`
      Player uint `json:"player"`
      Commands string `json:"commands"`
    }
    err = api.SignedRequest(&req)
    if err != nil { api.Error(err); return }
    block, err := store.ReadBlock(req.CurrentBlock)
    if err != nil { api.Error(err); return }
    cmds, err := store.CheckCommands(block.Base(), req.Commands)
    if err != nil { api.Error(err); return }
    gameKey := c.Param("gameKey")
    m := model.New(c, db)
    /* XXX pass raw commands to SetPlayerCommands */
    err = m.SetPlayerCommands(gameKey, req.Author[1:], req.CurrentBlock, req.Player, cmds)
    if err != nil { api.Error(err); return }
    api.Result(j.Raw(cmds))
  })

  r.POST("/Games/:gameKey/CloseRound", func (c *gin.Context) {
    api := newApi(c)
    var err error
    var req struct {
      Author string `json:"author"`
      CurrentBlock string `json:"current_block"`
    }
    err = api.SignedRequest(&req)
    if err != nil { api.Error(err); return }
    gameKey := c.Param("gameKey")
    m := model.New(c, db)
    cmds, err := m.CloseRound(gameKey, req.Author[1:], req.CurrentBlock)
    if err != nil { api.Error(err); return }
    res := j.Object()
    res.Prop("commands", j.Raw(cmds))
    api.Result(res)
  })

  r.POST("/Games/:gameKey/CancelRound", func (c *gin.Context) {
    api := newApi(c)
    gameKey := c.Param("gameKey")
    m := model.New(c, db)
    err := m.CancelRound(gameKey)
    if err != nil { api.Error(err); return }
    api.Result(j.Null)
  })

  r.POST("/Games/:gameKey/Execute", func (c *gin.Context) {
    api := newApi(c)
    gameKey := c.Param("gameKey")
    m := model.New(c, db)
    game, err := m.LoadGame(gameKey, model.NullFacet)
    if err != nil { api.Error(err); return }
    hash, err := store.MakeCommandBlock(game.Last_block, game.Next_block_commands)
    if err != nil { api.Error(err); return }
    // build the command block [on remote server?]
    // begin transaction
    //   if success, commit "pending" commands, set new last_block, clear "pending" status of game
    //   if failure, clear "pending" commands, clear "pending" status of game
    // end transaction
    api.Result(j.String(hash))
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

}
