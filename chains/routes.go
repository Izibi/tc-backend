
package chains

import (
  "fmt"
  "database/sql"
  "encoding/json"
  "time"
  "github.com/gin-gonic/gin"
  "tezos-contests.izibi.com/backend/auth"
  j "tezos-contests.izibi.com/backend/jase"
  "tezos-contests.izibi.com/backend/config"
  "tezos-contests.izibi.com/backend/events"
  "tezos-contests.izibi.com/backend/blocks"
  "tezos-contests.izibi.com/backend/model"
  "tezos-contests.izibi.com/backend/utils"
  "tezos-contests.izibi.com/backend/view"
)

type Service struct {
  config *config.Config
  events *events.Service
  model *model.Model
  auth *auth.Service
  blockStore *blocks.Service
}

func NewService(config *config.Config, events *events.Service, model *model.Model, auth *auth.Service, blockStore *blocks.Service) *Service {
  return &Service{config, events, model, auth, blockStore}
}

func (svc *Service) Route(r gin.IRoutes) {

  r.GET("/Chains", func(c *gin.Context) {
    r := utils.NewResponse(c)
    v := view.New(svc.model)
    userId, ok := auth.GetUserId(c)
    if !ok { r.BadUser(); return }
    contestId := view.ImportId(c.Query("contestId"))
    err := v.ViewChains(userId, contestId, view.ChainFilters{
      Status: c.Query("status"),
    })
    if err != nil { r.Error(err); return }
    r.Send(v.Flat())
  })

  r.POST("/Chains/:chainId/Fork", func(c *gin.Context) {
    r := utils.NewResponse(c)
    userId, ok := auth.GetUserId(c)
    if !ok { r.BadUser(); return }
    oldChainId := view.ImportId(c.Param("chainId"))
    oldChain, err := svc.model.LoadChain(oldChainId)
    if err != nil { r.Error(err); return }
    /*
      The user must belong to a team in contest chain.contest_id.
      TODO: quotas on number of private chains per team?
    */
    team, err := svc.model.LoadUserContestTeam(userId, oldChain.Contest_id)
    if err != nil { r.Error(err); return }
    if team == nil { r.StringError("access denied"); return }

    newChainId, err := svc.model.ForkChain(team.Id, oldChainId)
    if err != nil { r.Error(err); return }

    /* Attempt to initialize a game on the new chain. */
    newChain, err := svc.model.LoadChain(newChainId)
    if err != nil { r.Error(err); return }
    oldGame, err := svc.model.LoadGame(oldChain.Game_key)
    if err == nil {
      block, err := svc.blockStore.ReadBlock(oldGame.Last_block)
      if err == nil {
        bb := block.Base()
        var firstBlock string
        if bb.Kind == "setup" {
          firstBlock = oldGame.Last_block
        } else if bb.Setup != "" {
          firstBlock = bb.Setup
        }
        if firstBlock != "" {
          gameParams := model.GameParams{
            First_round: 0,
            Nb_rounds: oldGame.Max_nb_rounds,
            Nb_players: oldGame.Max_nb_players,
            Cycles_per_round: oldGame.Nb_cycles_per_round,
          }
          gameKey, err := svc.model.CreateGame(newChain.Owner_id.Int64, firstBlock, gameParams)
          if err == nil {
            err = svc.model.SetChainGameKey(newChainId, gameKey)
          }
        }
      }
    }

    /* XXX Temporary, post on team channel as chain is private */
    message := fmt.Sprintf("chain %s created", view.ExportId(newChainId))
    svc.events.PostContestMessage(team.Contest_id, message)

    r.Result(j.String(view.ExportId(newChainId)))
  })

  r.POST("/Chains/:chainId/Delete", func(c *gin.Context) {
    r := utils.NewResponse(c)
    userId, ok := auth.GetUserId(c)
    if !ok { r.BadUser(); return }
    chainId := view.ImportId(c.Param("chainId"))
    chain, err := svc.model.DeleteChain(userId, chainId)
    if err != nil { r.Error(err); return }

    /* XXX temporary */
    message := fmt.Sprintf("chain %s deleted", view.ExportId(chain.Id))
    svc.events.PostContestMessage(chain.Contest_id, message)
    /*
    if chain.Status_id == 1 { // XXX should query model to test if chain is private
      svc.events.PostTeamMessage(chain.Owner_id.Int64, message)
    } else {
      svc.events.PostContestMessage(chain.Contest_id, message)
    }
    */

    r.Result(j.Boolean(true))
  })

  r.POST("/Chains/:chainId/Restart", func(c *gin.Context) {
    r := utils.NewResponse(c)
    userId, ok := auth.GetUserId(c)
    if !ok { r.BadUser(); return }
    chainId := view.ImportId(c.Param("chainId"))
    chain, err := svc.model.LoadChain(chainId)
    if err != nil { r.Error(err); return }
    team, err := svc.model.LoadUserContestTeam(userId, chain.Contest_id)
    if err != nil { r.Error(err); return }
    if team == nil { r.StringError("access denied"); return }
    protoHash := chain.New_protocol_hash
    if protoHash == "" { r.StringError("invalid new protocol"); return }
    game, err := svc.model.LoadGame(chain.Game_key)
    if err != nil { r.Error(err); return }
    if game == nil { r.StringError("no game on chain"); return }
    lastBlock, err := svc.blockStore.ReadBlock(game.Last_block)
    if err != nil { r.Error(err); return }
    setupHash := lastBlock.Base().Setup
    if setupHash == "" { r.StringError("no setup block"); return }
    /* Load params from the store to keep task-specific params. */
    bsParams, err := svc.blockStore.ReadResource(setupHash, "params.json")
    if err != nil { r.Error(err) }
    var gameParams model.GameParams
    err = json.Unmarshal(bsParams, &gameParams)
    if err != nil { r.Error(err); return }
    setupHash, err = svc.blockStore.MakeSetupBlock(protoHash, bsParams)
    if err != nil { r.Error(err) }
    gameKey, err := svc.model.CreateGame(team.Id, setupHash, gameParams)
    if err != nil { r.Error(err); return }
    now := time.Now().Format(time.RFC3339)
    chain.Updated_at = now
    chain.Started_at = sql.NullString{}
    chain.Game_key = gameKey
    err = svc.model.SaveChain(chain)
    if err != nil { r.Error(err); return }
    message := fmt.Sprintf("chain %s restarted", view.ExportId(chainId))
    svc.events.PostContestMessage(team.Contest_id, message)
    r.Result(j.Boolean(true))
  })

  /* Install a game on a chain. */
  r.POST("/Chains/:chainId/ChangeGame", func(c *gin.Context) {
    var err error
    r := utils.NewResponse(c)
    type Request struct {
      GameKey string `json:"gameKey"`
    }
    var req Request
    if c.Bind(&req) != nil { return }

    userId, ok := auth.GetUserId(c)
    if !ok { r.BadUser(); return }
    if !svc.model.IsUserAdmin(userId) { r.StringError("Not Authorized") }

    chainId := view.ImportId(c.Param("chainId"))
    fmt.Printf("Game key: %s\n", req.GameKey)
    fmt.Printf("Chain id: %s\n", chainId)

    chain, err := svc.model.LoadChain(chainId)
    if err != nil { r.Error(err); return }

    var game *model.Game
    game, err = svc.model.LoadGame(req.GameKey)
    if err != nil { r.Error(err); return }
    if game == nil { r.StringError("no such game"); return }
    var block blocks.Block
    block, err = svc.blockStore.ReadBlock(game.Last_block)
    if err != nil { r.Error(err); return }
    protocolHash := block.Base().Protocol

    var intf, impl []byte
    intf, impl, err = svc.blockStore.LoadProtocol(protocolHash)
    if err != nil { r.Error(err); return }

    now := time.Now().Format(time.RFC3339)
    chain.Updated_at = now
    chain.Started_at = sql.NullString{}
    chain.Interface_text = string(intf)
    chain.Implementation_text = string(impl)
    chain.Game_key = req.GameKey
    chain.Parent_id = sql.NullInt64{}
    chain.Protocol_hash = protocolHash
    chain.New_protocol_hash = protocolHash
    chain.Nb_votes_approve = 0
    chain.Nb_votes_reject = 0
    chain.Nb_votes_unknown = 0
    err = svc.model.SaveChain(chain)
    if err != nil { r.Error(err); return }

    r.Result(j.Boolean(true))
  })

}
