
package routes

import (
  "encoding/json"
  "fmt"
  "io"
  "strings"
  "strconv"
  "time"
  "github.com/gin-gonic/gin"
  "tezos-contests.izibi.com/backend/blocks"
  "tezos-contests.izibi.com/backend/model"
  "tezos-contests.izibi.com/backend/utils"
  "tezos-contests.izibi.com/backend/view"
  j "tezos-contests.izibi.com/backend/jase"
)

func (svc *Service) RouteGames(r gin.IRoutes) {

  r.GET("/Games/:gameKey", func (c *gin.Context) {
    r := utils.NewResponse(c)
    gameKey := c.Param("gameKey")
    game, err := svc.model.LoadGame(gameKey)
    if err != nil { r.Error(err); return }
    if game == nil { r.StringError("bad key"); return }
    /* The hash of the last block is a convenient ETag value. */
    etag := fmt.Sprintf("\"%s\"", game.Last_block)
    if strings.Contains(c.GetHeader("If-None-Match"), etag) {
      c.Status(304)
      return
    }
    result := j.Object()
    result.Prop("game", ViewGame(game))
    lastPage, blocks, err := svc.store.GetHeadIndex(game.Game_key, game.Last_block)
    if err != nil { r.Error(err); return }
    result.Prop("page", j.Uint64(lastPage))
    result.Prop("blocks", j.Raw(blocks))
    ps, err := svc.model.LoadRegisteredGamePlayer(game.Id)
    if err != nil { r.Error(err); return }
    result.Prop("players", ViewPlayers(ps))
    scores, err := svc.store.ReadResource(game.Last_block, "scores.txt")
    if err == nil {
      result.Prop("scores", j.String(string(scores)))
    }
    c.Header("ETag", etag)
    c.Header("Cache-Control", "public, no-cache") // 1 day
    r.Result(result)
  })

  r.GET("/Games/:gameKey/Index/:page", func (c *gin.Context) {
    r := utils.NewResponse(c)
    gameKey := c.Param("gameKey")
    page, err := strconv.ParseUint(c.Param("page"), 10, 64)
    if err != nil { r.Error(err); return }
    game, err := svc.model.LoadGame(gameKey)
    if err != nil { r.Error(err); return }
    if game == nil { c.AbortWithStatus(404); return }
    blocks, err := svc.store.GetPageIndex(game.Game_key, game.Last_block, page)
    if err != nil { r.Error(err); return }
    result := j.Object()
    result.Prop("page", j.Uint64(page))
    result.Prop("blocks", j.Raw(blocks))
    c.Header("Cache-Control", "public, max-age=86400, immutable") // 1 day
    r.Result(result)
  })

  r.POST("/Games", func (c *gin.Context) {
    var req struct {
      Author string `json:"author"`
      FirstBlock string `json:"first_block"`
      // TODO: add a nonce
    }
    r, err := svc.SignedRequest(c, &req)
    if err != nil { r.Error(err); return }
    ownerId, err := svc.model.FindTeamIdByKey(req.Author[1:])
    if ownerId == 0 { r.StringError("team key is not recognized"); return }
    if err != nil { r.Error(err); return }
    // Read the requested first block.
    var block blocks.Block
    block, err = svc.store.ReadBlock(req.FirstBlock)
    if err != nil {
      r.StringError("bad first block")
      return
    }
    // Find the last setup block in the chain.
    setupBlock := blocks.LastSetupBlock(req.FirstBlock, block)
    if setupBlock == "" { r.StringError("no setup block"); return }
    // Read and parse the params from the setup block.
    bsParams, err := svc.store.ReadResource(setupBlock, "params.json")
    if err != nil { r.Error(err); return }
    var setupParams struct {
      NbCyclesPerRound uint32 `json:"cycles_per_round"`
      Nb_players uint32 `json:"nb_players"`
      Nb_rounds uint64 `json:"nb_rounds"`
    }
    err = json.Unmarshal(bsParams, &setupParams)
    if err != nil { r.Error(err); return }
    gameParams := model.GameParams{
      First_round: block.Base().Round,
      Nb_rounds: setupParams.Nb_rounds,
      Nb_players: setupParams.Nb_players,
      Cycles_per_round: setupParams.NbCyclesPerRound,
    }
    gameKey, err := svc.model.CreateGame(ownerId, req.FirstBlock, gameParams)
    if err != nil { r.Error(err); return }
    game, err := svc.model.LoadGame(gameKey)
    if err != nil { r.Error(err); return }
    r.Result(ViewGame(game))
  })

  r.POST("/Games/:gameKey/Register", func (c *gin.Context) {
    var req struct {
      Author string `json:"author"`
      // TODO: put the game key here
      Ids []uint32 `json:"ids"`
    }
    r, err := svc.SignedRequest(c, &req)
    if err != nil { r.Error(err); return }
    teamId, err := svc.model.FindTeamIdByKey(req.Author[1:])
    if err != nil { r.Error(err); return }
    if teamId == 0 { r.StringError("team key is not recognized"); return }
    gameKey := c.Param("gameKey")
    var ranks []uint32
    err = svc.model.Transaction(c, func () (err error) {
      ranks, err = svc.model.RegisterGamePlayers(gameKey, teamId, req.Ids)
      return
    })
    if err != nil { r.Error(err); return }
    res := j.Object()
    jRanks := j.Array()
    for _, n := range(ranks) {
      jRanks.Item(j.Uint32(n))
    }
    res.Prop("ranks", jRanks)
    r.Result(res)
  })

  r.POST("/Games/:gameKey/Commands", func (c *gin.Context) {
    var req struct {
      Author string `json:"author"`
      CurrentBlock string `json:"current_block"`
      Player uint32 `json:"player"`
      Commands string `json:"commands"`
    }
    r, err := svc.SignedRequest(c, &req)
    if err != nil { r.Error(err); return }
    teamId, err := svc.model.FindTeamIdByKey(req.Author[1:])
    if err != nil { r.Error(err); return }
    if teamId == 0 { r.StringError("team key is not recognized"); return }
    block, err := svc.store.ReadBlock(req.CurrentBlock)
    if err != nil { r.Error(err); return }
    cmds, err := svc.store.CheckCommands(block.Base(), req.Commands)
    if err != nil { r.Error(err); return }
    gameKey := c.Param("gameKey")
    err = svc.model.Transaction(c, func () error {
      return svc.model.SetPlayerCommands(gameKey, req.CurrentBlock, teamId, req.Player, cmds)
    })
    if err != nil { r.Error(err); return }
    r.Result(j.Raw(cmds))
  })

  r.POST("/Games/:gameKey/CloseRound", func (c *gin.Context) {
    var req struct {
      Author string `json:"author"`
      // TODO: put the game key here
      CurrentBlock string `json:"current_block"`
    }
    r, err := svc.SignedRequest(c, &req)
    if err != nil { r.Error(err); return }
    teamId, err := svc.model.FindTeamIdByKey(req.Author[1:])
    if err != nil { r.Error(err); return }
    if teamId == 0 { r.StringError("team key is not recognized"); return }
    gameKey := c.Param("gameKey")
    var game *model.Game
    err = svc.model.Transaction(c, func () (err error) {
      game, err = svc.model.CloseRound(gameKey, teamId, req.CurrentBlock)
      return err
    })
    if err != nil { r.Error(err); return }
    go func () {
      var err error
      var newBlock string
      newBlock, err = svc.store.MakeCommandBlock(game.Last_block, game.Next_block_commands)
      if err != nil { /* TODO: mark error in block */ return }
      err = svc.store.ClearHeadIndex(gameKey)
      if err != nil { /* TODO: mark error in block */ return }
      err = svc.model.Transaction(c, func () (err error) {
        _, err = svc.model.EndRoundAndUnlock(gameKey, newBlock)
        if err != nil { /* TODO: mark error in block */ return }
        return
      })
      if err != nil {
        // TODO: post an error!
      } else {
        svc.events.PostGameMessage(gameKey, newBlockMessage(newBlock))
      }
    }()
    res := j.Object()
    res.Prop("commands", j.Raw(game.Next_block_commands))
    r.Result(res)
  })

  r.POST("/Games/:gameKey/CancelRound", func (c *gin.Context) {
    var req struct {
      Author string `json:"author"`
      GameKey string `json:"gameKey"`
      CurrentBlock string `json:"currentBlock"`
    }
    r, err := svc.SignedRequest(c, &req)
    if err != nil { r.Error(err); return }
    err = svc.model.Transaction(c, func () (err error) {
      _, err = svc.model.CancelRound(req.GameKey)
      return
    })
    if err != nil { r.Error(err); return }
    r.Result(j.Null)
  })

  r.POST("/Games/:gameKey/Ping", func (c *gin.Context) {
    // TODO: this request should be signed
    var err error
    var keys []string
    keys, err = svc.model.LoadGamePlayerTeamKeys(c.Param("gameKey"))
    if err != nil { c.String(400, "failed to load players"); return }
    key, err := utils.NewKey()
    if err != nil { c.String(500, "failed to generate a key"); return }
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
    var req struct {
      Author string `json:"author"`
      Payload string `json:"payload"`
    }
    r, err := svc.SignedRequest(c, &req)
    if err != nil { r.Error(err); return }
    svc.rc.Publish(fmt.Sprintf("ping:%s", req.Payload), req.Author)
    r.Result(j.Boolean(true))
  })

}

func ViewGame(game *model.Game) j.Value {
  if game == nil {
    return j.Null
  }
  obj := j.Object()
  obj.Prop("key", j.String(game.Game_key))
  obj.Prop("createdAt", j.Time(game.Created_at))
  obj.Prop("updatedAt", j.Time(game.Updated_at))
  obj.Prop("ownerId", j.String(view.ExportId(game.Owner_id)))
  obj.Prop("firstBlock", j.String(game.First_block))
  obj.Prop("lastBlock", j.String(game.Last_block))
  if game.Started_at.Valid {
    obj.Prop("startedAt", j.Time(game.Started_at.Time))
  }
  if game.Round_ends_at.Valid {
    obj.Prop("roundEndsAt", j.Time(game.Round_ends_at.Time))
  }
  obj.Prop("isLocked", j.Boolean(game.Locked))
  obj.Prop("currentRound", j.Uint64(game.Current_round))
  obj.Prop("nbCyclesPerRound", j.Uint32(game.Nb_cycles_per_round))
  obj.Prop("nbRounds", j.Uint64(game.Max_nb_rounds))
  obj.Prop("nbPlayers", j.Uint32(game.Max_nb_players))
  return obj
}

func ViewPlayers(players []model.RegisteredGamePlayer) j.Value {
  items := j.Array()
  for i := range players {
    player := &players[i]
    obj := j.Object()
    obj.Prop("rank", j.Uint32(player.Rank))
    obj.Prop("teamId", j.String(view.ExportId(player.Team_id)))
    obj.Prop("botId", j.Uint32(player.Team_player /* TODO res.Bot_id */))
    items.Item(obj)
  }
  return items
}

func newPingMessage(payload string) string {
  return fmt.Sprintf("ping %s", payload)
}

func newBlockMessage(hash string) string {
  return fmt.Sprintf("block %s", hash)
}
