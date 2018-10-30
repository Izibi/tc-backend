/*

  Game API

  Requests that need authentication are designed to be used by the tc-node
  command line tool.
  These requests must be signed with the team's private key and include the
  team's public key (prefixed with '@') in the "author" request parameter.
  This request format is inspired by secure-scuttlebutt messages.

  Routes include the game key in the URL (in addition to the request) to
  permit sharding games across multiple servers.

*/

package routes

import (
  "encoding/json"
  "errors"
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

const (
  GameApiRevision = 1 /* increment when the /Games/:gameKey response changes */
)

/* Union of all game API requests (except creation) */
type GameRequest struct {
  Action string `json:"action"`
  Author string `json:"author"`
  BotIds []uint32 `json:"botIds"` /* "register bots", "ping" */
  Commands string `json:"commands"` /* "enter commands" */
  CurrentBlock string `json:"current_block"` /* "enter commands", "close round" */
  GameKey string `json:"gameKey"` /* all */
  Payload string `json:"payload"` /* "pong" */
  Player uint32 `json:"player"` /* "enter commands" */
  Timestamp string `json:"timestamp"` /* "ping", "pong" -- Unix time, milliseconds, as string */
}

func (svc *Service) RouteGames(routes gin.IRoutes) {

  routes.GET("/Games/:gameKey", func (c *gin.Context) {
    r := utils.NewResponse(c)
    gameKey := c.Param("gameKey")
    game, err := svc.model.LoadGame(gameKey)
    if err != nil { r.Error(err); return }
    if game == nil { r.StringError("bad key"); return }
    /* The hash of the last block is a convenient ETag value. */
    etag := fmt.Sprintf("\"%s %d\"", game.Last_block, GameApiRevision)
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

  /* Consider: to avoid cache issues if the page size has to change, use a
     range rather than a page number. */
  routes.GET("/Games/:gameKey/Index/:page", func (c *gin.Context) {
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

  routes.POST("/Games", func (c *gin.Context) {
    var req struct {
      Author string `json:"author"`
      FirstBlock string `json:"first_block"`
      Timestamp string `json:"timestamp"`
    }
    r, err := svc.signedRequest(c, &req)
    if err != nil { r.Error(err); return }
    teamId, err := svc.checkAuthor(req.Author)
    if err != nil { r.Error(err); return }
    /* TODO: check that req.Timestamp is recent */
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
    var gameKey string
    err = svc.model.Transaction(c, func () (err error) {
      /* TODO: check that there is no game by the same team with created_at = req.Timestamp ? */
      gameKey, err = svc.model.CreateGame(teamId, req.FirstBlock, gameParams)
      return
    })
    if err != nil { r.Error(err); return }
    game, err := svc.model.LoadGame(gameKey)
    if err != nil { r.Error(err); return }
    r.Result(ViewGame(game))
  })

  routes.POST("/Games/:gameKey", func (c *gin.Context) {
    var req GameRequest
    r, err := svc.signedRequest(c, &req)
    if err != nil { r.Error(err); return }
    teamId, err := svc.checkAuthor(req.Author)
    if err != nil { r.Error(err); return }
    if req.GameKey != c.Param("gameKey") {
      r.StringError("game key mismatch")
      return
    }
    // Some actions can only be performed by the game owner.
    switch req.Action {
    case "close round", "cancel_round", "ping":
      ok, err := svc.model.IsGameOwner(req.GameKey, teamId)
      if err != nil { r.Error(err); return }
      if !ok { r.StringError("not game owner"); return }
    }
    switch req.Action {
    case "register bots":
      gameRegisterBots(svc, c, r, &req, teamId)
    case "enter commands":
      gameEnterCommands(svc, c, r, &req, teamId)
    case "close round":
      gameCloseRound(svc, c, r, &req)
    case "cancel_round":
      gameCancelRound(svc, c, r, &req)
    case "ping":
      gamePing(svc, c, r, &req)
    case "pong":
      gamePong(svc, c, r, &req)
    default:
      r.StringError("bad action")
      return
    }
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

func gameRegisterBots(svc *Service, c *gin.Context, r *utils.Response, req *GameRequest, teamId int64) {
  var err error
  var ranks []uint32
  err = svc.model.Transaction(c, func () (err error) {
    ranks, err = svc.model.RegisterGamePlayers(req.GameKey, teamId, req.BotIds)
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
}

func gameEnterCommands(svc *Service, c *gin.Context, r *utils.Response, req *GameRequest, teamId int64) {
  var err error
  block, err := svc.store.ReadBlock(req.CurrentBlock)
  if err != nil { r.Error(err); return }
  cmds, err := svc.store.CheckCommands(block.Base(), req.Commands)
  if err != nil { r.Error(err); return }
  err = svc.model.Transaction(c, func () error {
    return svc.model.SetPlayerCommands(req.GameKey, req.CurrentBlock, teamId, req.Player, cmds)
  })
  if err != nil { r.Error(err); return }
  r.Result(j.Raw(cmds))
}

func gameCloseRound(svc *Service, c *gin.Context, r *utils.Response, req *GameRequest) {
  var err error
  var game *model.Game
  err = svc.model.Transaction(c, func () (err error) {
    game, err = svc.model.CloseRound(req.GameKey, req.CurrentBlock)
    return err
  })
  if err != nil { r.Error(err); return }
  go func () {
    /* XXX the game will not unlock if the backend crashes before this routine completes */
    var err error
    var newBlock string
    newBlock, err = svc.store.MakeCommandBlock(game.Last_block, game.Next_block_commands)
    if err != nil { /* TODO: mark error in block */ return }
    err = svc.store.ClearHeadIndex(req.GameKey)
    if err != nil { /* TODO: mark error in block */ return }
    err = svc.model.Transaction(c, func () (err error) {
      _, err = svc.model.EndRoundAndUnlock(req.GameKey, newBlock)
      if err != nil { /* TODO: mark error in block */ return }
      return
    })
    if err != nil {
      // TODO: post an error!
    } else {
      svc.events.PostGameMessage(req.GameKey, newBlockMessage(newBlock))
    }
  }()
  res := j.Object()
  res.Prop("commands", j.Raw(game.Next_block_commands))
  r.Result(res)
}

func gameCancelRound(svc *Service, c *gin.Context, r *utils.Response, req *GameRequest) {
  var err error
  if err != nil { r.Error(err); return }
  err = svc.model.Transaction(c, func () (err error) {
    _, err = svc.model.CancelRound(req.GameKey)
    return
  })
  if err != nil { r.Error(err); return }
  r.Result(j.Null)
}

func gamePing(svc *Service, c *gin.Context, r *utils.Response, req *GameRequest) {
  var err error
  var pingTime time.Time
  pingTime, err = parseUnixMillis(req.Timestamp)
  if err != nil { r.Error(err); return }
  // TODO: verify pingTime is recent enough
  var bots []model.GamePlayerId
  bots, err = svc.model.LoadGamePlayerIds(req.GameKey)
  if err != nil { r.Error(err); return }
  key, err := utils.NewKey()
  if err != nil { r.StringError("failed to generate a key"); return }
  sub := svc.rc.Subscribe(pingChannel(key))
  var message = fmt.Sprintf("ping %s", key)
  svc.events.PostGameMessage(req.GameKey, message)
  var timeout = time.NewTimer(2 * time.Second)
  var ch = sub.Channel()
  var nbExpected = len(bots)
  var received = make([]bool, nbExpected, nbExpected)
  c.Header("Content-Type", "text/event-stream")
  c.Header("X-Accel-Buffering", "no")
  var started bool
  c.Stream(func (w io.Writer) bool {
    var err error
    if !started {
      /* Write an initial body line to force sending headers. */
      started = true
      w.Write([]byte("START\n"))
      return true
    }
    for {
      select {
      case <-timeout.C: {
        for i := range bots {
          if !received[i] {
            bot := &bots[i]
            w.Write([]byte(fmt.Sprintf("timeout %d %s %d\n", bot.Rank, bot.Team_key, bot.Bot_id)))
          }
        }
        w.Write([]byte("ERROR\n"))
        sub.Close()
        return false
      }
      case m := <-ch:
        if m == nil {
          w.Write([]byte("pubsub error\n"))
          timeout.Stop()
          sub.Close()
          return false
        }
        // expect payload of form "@key tsUnixMs botId(,botId)*"
        var msg *PongMessage
        msg, err = DecodePongMessage(m.Payload)
        if err != nil {
          fmt.Printf("Bad pong message: %v\n  message: %s\n", err, m.Payload)
          return true
        }
        var pongTime time.Time
        pongTime, err = parseUnixMillis(msg.Timestamp)
        if err != nil {
          fmt.Printf("Bad pong message: %v\n  message: %s\n", err, m.Payload)
          return true
        }
        millis := pongTime.Sub(pingTime).Nanoseconds() / 1000000
        for _, botId := range msg.BotIds {
          for i := range bots {
            bot := &bots[i]
            if bot.Team_key == msg.TeamKey && bot.Bot_id == botId && !received[i] {
              w.Write([]byte(fmt.Sprintf("pong %d %s %d %d\n", bot.Rank, bot.Team_key, bot.Bot_id, millis)))
              received[i] = true
              nbExpected -= 1
            }
          }
        }
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
}

func gamePong(svc *Service, c *gin.Context, r *utils.Response, req *GameRequest) {
  message := PongMessage{req.Timestamp, req.Author[1:], req.BotIds}
  svc.rc.Publish(pingChannel(req.Payload), message.Encode())
  r.Result(j.Boolean(true))
}

func newBlockMessage(hash string) string {
  return fmt.Sprintf("block %s", hash)
}

func pingChannel(key string) string {
  return fmt.Sprintf("ping:%s", key)
}

type PongMessage struct {
  Timestamp string
  TeamKey string
  BotIds []uint32
}

func (m *PongMessage) Encode() string {
  ids := make([]string, len(m.BotIds))
  for i, id := range m.BotIds {
    ids[i] = strconv.FormatInt(int64(id), 10)
  }
  return strings.Join([]string{
    m.Timestamp,
    m.TeamKey,
    strings.Join(ids, ","),
  }, " ")
}

func DecodePongMessage(msg string) (*PongMessage, error) {
  var err error
  parts := strings.Split(msg, " ")
  if len(parts) != 3 {
    return nil, errors.New("bad pong message")
  }
  timestamp := parts[0]
  teamKey := parts[1]
  strIds := strings.Split(parts[2], ",")
  ids := make([]uint32, len(strIds))
  for i, strId := range strIds {
    var id uint64
    id, err = strconv.ParseUint(strId, 10, 32)
    if err != nil { return nil, err }
    ids[i] = uint32(id)
  }
  return &PongMessage{
    Timestamp: timestamp,
    TeamKey: teamKey,
    BotIds: ids,
  }, nil
}

func parseUnixMillis(s string) (t time.Time, err error) {
  var millis int64
  millis, err = strconv.ParseInt(s, 10, 64)
  if err != nil { return }
  t = time.Unix(millis / 1000, (millis % 1000) * 1000000)
  return
}
