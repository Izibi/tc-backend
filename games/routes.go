
package games

import (
  "fmt"
  "database/sql"
  "strconv"
  "github.com/gin-gonic/gin"
  "tezos-contests.izibi.com/backend/blocks"
  "tezos-contests.izibi.com/backend/config"
  "tezos-contests.izibi.com/backend/events"
  "tezos-contests.izibi.com/backend/model"
  "tezos-contests.izibi.com/backend/utils"
  j "tezos-contests.izibi.com/backend/jase"
)

type Service struct {
  config *config.Config
  events *events.Service
  store *blocks.Service
  db *sql.DB
}

type Context struct {
  c *gin.Context
  req *utils.Request
  resp *utils.Response
  model *model.Model
}

func NewService(config *config.Config, db *sql.DB, events *events.Service, store *blocks.Service) *Service {
  return &Service{config, events, store, db}
}

func (svc *Service) Wrap(c *gin.Context) *Context {
  return &Context{
    c,
    utils.NewRequest(c, svc.config.ApiKey),
    utils.NewResponse(c),
    model.New(c, svc.db),
  }
}

func (svc *Service) Route(r gin.IRoutes) {

  r.POST("/Games", func (c *gin.Context) {
    ctx := svc.Wrap(c)
    var err error
    var req struct {
      Author string `json:"author"`
      FirstBlock string `json:"first_block"`
    }
    err = ctx.req.Signed(&req)
    if err != nil { ctx.resp.Error(err); return }
    fmt.Printf("new game request %v\n", req)
    var block blocks.Block
    block, err = svc.store.ReadBlock(req.FirstBlock)
    if err != nil {
      ctx.resp.StringError("bad first block")
      return
    }
    ownerId, err := ctx.model.FindTeamIdByKey(req.Author[1:])
    if ownerId == 0 { ctx.resp.StringError("team key is not recognized"); return }
    if err != nil { ctx.resp.Error(err); return }
    gameKey, err := ctx.model.CreateGame(ownerId, req.FirstBlock, block.Base().Round)
    if err != nil { ctx.resp.Error(err); return }
    game, err := ctx.model.LoadGame(gameKey, model.NullFacet)
    if err != nil { ctx.resp.Error(err); return }
    ctx.resp.Result(ctx.model.ViewGame(game))
  })

  r.GET("/Games/:gameKey", func (c *gin.Context) {
    ctx := svc.Wrap(c)
    gameKey := c.Param("gameKey")
    game, err := ctx.model.LoadGame(gameKey, model.NullFacet)
    if err != nil { ctx.resp.Error(err); return }
    if game == nil { ctx.resp.StringError("bad key"); return }
    /* The hash of the last block is a convenient ETag value. */
    etag := fmt.Sprintf("\"%s\"", game.Last_block)
    if c.GetHeader("If-None-Match") == etag {
      c.Status(304)
      return
    }
    result := j.Object()
    result.Prop("game", ctx.model.ViewGame(game))
    if game != nil {
      lastPage, blocks, err := svc.store.GetHeadIndex(game.Game_key, game.Last_block)
      if err != nil { ctx.resp.Error(err); return }
      result.Prop("page", j.Uint64(lastPage))
      result.Prop("blocks", j.Raw(blocks))
    }
    c.Header("ETag", etag)
    c.Header("Cache-Control", "public, no-cache") // 1 day
    ctx.resp.Result(result)
  })

  r.GET("/Games/:gameKey/Index/:page", func (c *gin.Context) {
    ctx := svc.Wrap(c)
    gameKey := c.Param("gameKey")
    page, err := strconv.ParseUint(c.Param("page"), 10, 64)
    if err != nil { ctx.resp.Error(err); return }
    game, err := ctx.model.LoadGame(gameKey, model.NullFacet)
    if err != nil { ctx.resp.Error(err); return }
    result := j.Object()
    result.Prop("game", ctx.model.ViewGame(game))
    if game != nil {
      blocks, err := svc.store.GetPageIndex(game.Game_key, game.Last_block, page)
      if err != nil { ctx.resp.Error(err); return }
      result.Prop("page", j.Uint64(page))
      result.Prop("blocks", j.Raw(blocks))
    }
    c.Header("Cache-Control", "public, max-age=86400, immutable") // 1 day
    ctx.resp.Result(result)
  })

  r.POST("/Games/:gameKey/Commands", func (c *gin.Context) {
    ctx := svc.Wrap(c)
    var err error
    var req struct {
      Author string `json:"author"`
      CurrentBlock string `json:"current_block"`
      Player uint `json:"player"`
      Commands string `json:"commands"`
    }
    err = ctx.req.Signed(&req)
    if err != nil { ctx.resp.Error(err); return }
    block, err := svc.store.ReadBlock(req.CurrentBlock)
    if err != nil { ctx.resp.Error(err); return }
    cmds, err := svc.store.CheckCommands(block.Base(), req.Commands)
    if err != nil { ctx.resp.Error(err); return }
    gameKey := c.Param("gameKey")
    /* XXX pass raw commands to SetPlayerCommands */
    err = ctx.model.SetPlayerCommands(gameKey, req.Author[1:], req.CurrentBlock, req.Player, cmds)
    if err != nil { ctx.resp.Error(err); return }
    ctx.resp.Result(j.Raw(cmds))
  })

  r.POST("/Games/:gameKey/CloseRound", func (c *gin.Context) {
    ctx := svc.Wrap(c)
    var err error
    var req struct {
      Author string `json:"author"`
      CurrentBlock string `json:"current_block"`
    }
    err = ctx.req.Signed(&req)
    if err != nil { ctx.resp.Error(err); return }
    gameKey := c.Param("gameKey")
    game, err := ctx.model.CloseRound(gameKey, req.Author[1:], req.CurrentBlock)
    if err != nil { ctx.resp.Error(err); return }
    go func () {
      var err error
      var newBlock string
      newBlock, err = svc.store.MakeCommandBlock(game.Last_block, game.Next_block_commands)
      if err != nil { /* TODO: mark error in block */ return }
      err = svc.store.ClearHeadIndex(gameKey)
      if err != nil { /* TODO: mark error in block */ return }
      err = ctx.model.EndRoundAndUnlock(gameKey, newBlock)
      if err != nil { /* TODO: mark error in block */ return }
      svc.events.PostGameMessage(gameKey, ctx.NewBlockMessage(newBlock))
    }()
    res := j.Object()
    res.Prop("commands", j.Raw(game.Next_block_commands))
    ctx.resp.Result(res)
  })

  r.POST("/Games/:gameKey/CancelRound", func (c *gin.Context) {
    ctx := svc.Wrap(c)
    gameKey := c.Param("gameKey")
    err := ctx.model.CancelRound(gameKey)
    if err != nil { ctx.resp.Error(err); return }
    ctx.resp.Result(j.Null)
  })

  r.POST("/Games/:gameKey/Ping", func (c *gin.Context) {
    svc.events.PostGameMessage(c.Param("gameKey"), "ping")
    c.Status(204)
  })

}

func (c *Context) NewBlockMessage(hash string) string {
  return fmt.Sprintf("block %s", hash)
}
