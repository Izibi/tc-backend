
package games

import (
  "fmt"
  "database/sql"
  "io"
  "strconv"
  "time"
  "github.com/go-redis/redis"
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
  rc *redis.Client
}

type Context struct {
  c *gin.Context
  req *utils.Request
  resp *utils.Response
  model *model.Model
}

func NewService(config *config.Config, db *sql.DB, rc *redis.Client, events *events.Service, store *blocks.Service) *Service {
  return &Service{config, events, store, db, rc}
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

  r.POST("/Games/:gameKey/Register", func (c *gin.Context) {
    ctx := svc.Wrap(c)
    var err error
    var req struct {
      Author string `json:"author"`
      Ids []uint32 `json:"ids"`
    }
    err = ctx.req.Signed(&req)
    if err != nil { ctx.resp.Error(err); return }
    teamId, err := ctx.model.FindTeamIdByKey(req.Author[1:])
    if err != nil { ctx.resp.Error(err); return }
    if teamId == 0 { ctx.resp.StringError("team key is not recognized"); return }
    gameKey := c.Param("gameKey")
    var ranks []uint32
    ranks, err = ctx.model.RegisterGamePlayers(gameKey, teamId, req.Ids)
    if err != nil { ctx.resp.Error(err); return }
    res := j.Object()
    jRanks := j.Array()
    for _, n := range(ranks) {
      jRanks.Item(j.Uint32(n))
    }
    res.Prop("ranks", jRanks)
    ctx.resp.Result(res)
  })

  r.POST("/Games/:gameKey/Commands", func (c *gin.Context) {
    ctx := svc.Wrap(c)
    var err error
    var req struct {
      Author string `json:"author"`
      CurrentBlock string `json:"current_block"`
      Player uint32 `json:"player"`
      Commands string `json:"commands"`
    }
    err = ctx.req.Signed(&req)
    if err != nil { ctx.resp.Error(err); return }
    teamId, err := ctx.model.FindTeamIdByKey(req.Author[1:])
    if err != nil { ctx.resp.Error(err); return }
    if teamId == 0 { ctx.resp.StringError("team key is not recognized"); return }
    block, err := svc.store.ReadBlock(req.CurrentBlock)
    if err != nil { ctx.resp.Error(err); return }
    cmds, err := svc.store.CheckCommands(block.Base(), req.Commands)
    if err != nil { ctx.resp.Error(err); return }
    gameKey := c.Param("gameKey")
    /* XXX pass raw commands to SetPlayerCommands */
    err = ctx.model.SetPlayerCommands(gameKey, req.CurrentBlock, teamId, req.Player, cmds)
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
      svc.events.PostGameMessage(gameKey, newBlockMessage(newBlock))
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
    ctx := svc.Wrap(c)
    var err error
    var keys []string
    keys, err = ctx.model.LoadGamePlayerTeamKeys(c.Param("gameKey"))
    if err != nil { ctx.resp.Error(err); return }
    key, err := utils.NewKey()
    if err != nil { ctx.resp.Error(err); return }
    sub := svc.rc.Subscribe(fmt.Sprintf("ping:%s", key))
    svc.events.PostGameMessage(c.Param("gameKey"), newPingMessage(key))
    timeout := time.NewTimer(1 * time.Second)
    ch := sub.Channel()
    responders := make(map[string]bool)
    nbExpected := len(keys)
    for _, key := range keys {
      responders["@"+key] = false
    }
    c.Stream(func (w io.Writer) bool {
      for {
        select {
        case <-timeout.C:
          w.Write([]byte("timeout\n"))
          sub.Close()
          return false
        case m := <-ch:
          if m == nil {
            w.Write([]byte("pubsub error\n"))
            timeout.Stop()
            sub.Close()
            return false
          }
          identity := m.Payload
          received, expected := responders[identity]
          if !expected {
            w.Write([]byte(fmt.Sprintf("unexpected %s\n", identity)))
            return true
          }
          if received {
            w.Write([]byte(fmt.Sprintf("duplicate %s\n", identity)))
            return true
          }
          w.Write([]byte(fmt.Sprintf("received %s\n", identity)))
          responders[m.Payload] = true
          nbExpected -= 1
          if nbExpected == 0 {
            w.Write([]byte("OK\n"))
            timeout.Stop()
            sub.Close()
            return false
          }
          return true
        }
      }
    })
  })

  r.POST("/Games/:gameKey/Pong", func (c *gin.Context) {
    ctx := svc.Wrap(c)
    var req struct {
      Author string `json:"author"`
      Payload string `json:"payload"`
    }
    var err = ctx.req.Signed(&req)
    if err != nil { ctx.resp.Error(err); return }
    svc.rc.Publish(fmt.Sprintf("ping:%s", req.Payload), req.Author)
    ctx.resp.Result(j.Boolean(true))
  })

}

func newPingMessage(payload string) string {
  return fmt.Sprintf("ping %s", payload)
}

func newBlockMessage(hash string) string {
  return fmt.Sprintf("block %s", hash)
}
