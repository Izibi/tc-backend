
package routes

import (
  "fmt"
  "database/sql"
  "encoding/json"
  "time"
  "github.com/gin-gonic/gin"
  "tezos-contests.izibi.com/backend/auth"
  j "tezos-contests.izibi.com/backend/jase"
  "tezos-contests.izibi.com/backend/blocks"
  "tezos-contests.izibi.com/backend/model"
  "tezos-contests.izibi.com/backend/utils"
  "tezos-contests.izibi.com/backend/view"
)

func (svc *Service) RouteChains(r gin.IRoutes) {

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

  r.GET("/Chains/:chainId", func(c *gin.Context) {
    r := utils.NewResponse(c)
    v := view.New(svc.model)
    userId, ok := auth.GetUserId(c)
    if !ok { r.BadUser(); return }
    chainId := view.ImportId(c.Param("chainId"))
    chain, err := svc.model.LoadChain(chainId)
    if err != nil { r.Error(err); return }
    team, err := svc.model.LoadUserContestTeam(userId, chain.Contest_id)
    if err != nil { r.Error(err); return }
    if team == nil { r.StringError("access denied"); return }
    v.SetTeam(team.Id) // view will protect private chains from other teams
    err = v.ViewChainDetails(chainId)
    if err != nil { r.Error(err); return }
    r.Send(v.Flat())
  })

  r.POST("/Chains/:chainId/Update", func (c *gin.Context) {
    r := utils.NewResponse(c)
    v := view.New(svc.model)
    var err error
    userId, ok := auth.GetUserId(c)
    if !ok { r.BadUser(); return }
    chainId := view.ImportId(c.Param("chainId"))
    chain, err := svc.model.LoadChain(chainId)
    if err != nil { r.Error(err); return }
    if !svc.model.IsUserAdmin(userId) {
      if !chain.Owner_id.Valid {
        r.StringError("access denied"); return
      }
      team, err := svc.model.LoadUserContestTeam(userId, chain.Contest_id)
      if err != nil { r.Error(err); return }
      if team == nil || team.Id != chain.Owner_id.Int64 {
        r.StringError("access denied"); return
      }
      v.SetTeam(team.Id) // view will protect private chains from other teams
    }
    err = svc.model.SaveChainRevision(chain)
    if err != nil { r.Error(err); return }
    var arg struct {
      StatusId *string `json:"statusId"`
      Description *string `json:"description"`
      Interface_text *string `json:"interfaceText"`
      Implementation_text *string `json:"implementationText"`
    }
    err = c.Bind(&arg)
    if err != nil { r.Error(err); return }
    if arg.StatusId != nil {
      chain.Status_id = view.ImportId(*arg.StatusId)
    }
    if arg.Description != nil {
      chain.Description = *arg.Description
    }
    if arg.Interface_text != nil && *arg.Interface_text != chain.Interface_text {
      chain.Interface_text = *arg.Interface_text
      chain.Needs_recompile = true
    }
    if arg.Implementation_text != nil && *arg.Implementation_text != chain.Implementation_text {
      chain.Implementation_text = *arg.Implementation_text
      chain.Needs_recompile = true
    }
    chain.Updated_at = time.Now()
    err = svc.model.SaveChain(chain)
    if err != nil { r.Error(err); return }
    err = v.ViewChainDetails(chain.Id)
    if err != nil { r.Error(err); return }
    r.Send(v.Flat())
  })

  r.POST("/Chains/:chainId/Fork", func(c *gin.Context) {
    var err error
    r := utils.NewResponse(c)
    type Request struct {
      Title string `json:"title"`
    }
    var req Request
    err = c.Bind(&req)
    if err != nil { r.Error(err); return }

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

    var newChainId int64
    err = svc.model.Transaction(c, func() (err error) {

      newChainId, err = svc.model.ForkChain(team.Id, oldChainId, req.Title)
      if err != nil { return }

      /* Initialize a game on the new chain. */
      newChain, err := svc.model.LoadChain(newChainId)
      if err != nil { return }
      oldGame, err := svc.model.LoadGame(oldChain.Game_key)
      if err != nil { return }
      block, err := svc.store.ReadBlock(oldGame.Last_block)
      if err != nil { return }
      firstBlock := blocks.LastSetupBlock(oldGame.Last_block, block)
      if firstBlock == "" { return fmt.Errorf("no setup block") }
      // Read and parse the params from the setup block.
      bsParams, err := svc.store.ReadResource(firstBlock, "params.json")
      if err != nil { return }
      var setupParams struct {
        NbCyclesPerRound uint32 `json:"cycles_per_round"`
        Nb_players uint32 `json:"nb_players"`
        Nb_rounds uint64 `json:"nb_rounds"`
      }
      err = json.Unmarshal(bsParams, &setupParams)
      if err != nil { return }
      gameParams := model.GameParams{
        First_round: 0,
        Nb_rounds: setupParams.Nb_rounds,
        Nb_players: setupParams.Nb_players,
        Cycles_per_round: setupParams.NbCyclesPerRound,
      }
      gameKey, err := svc.model.CreateGame(newChain.Owner_id.Int64, firstBlock, gameParams)
      if err != nil { return }
      err = svc.model.SetChainGameKey(newChainId, gameKey)
      if err != nil { return }
      return nil

    })
    if err != nil { r.Error(err); return }

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
    lastBlock, err := svc.store.ReadBlock(game.Last_block)
    if err != nil { r.Error(err); return }
    setupHash := blocks.LastSetupBlock(game.Last_block, lastBlock)
    if setupHash == "" { r.StringError("no setup block"); return }
    /* Load params from the store to keep task-specific params. */
    bsParams, err := svc.store.ReadResource(setupHash, "params.json")
    if err != nil { r.Error(err) }
    var gameParams model.GameParams
    err = json.Unmarshal(bsParams, &gameParams)
    if err != nil { r.Error(err); return }
    setupHash, err = svc.store.MakeSetupBlock(protoHash, bsParams)
    if err != nil { r.Error(err) }
    gameKey, err := svc.model.CreateGame(team.Id, setupHash, gameParams)
    if err != nil { r.Error(err); return }
    err = svc.model.SaveChainRevision(chain)
    if err != nil { r.Error(err); return }
    chain.Updated_at = time.Now()
    chain.Started_at = sql.NullString{}
    chain.Game_key = gameKey
    err = svc.model.SaveChain(chain)
    if err != nil { r.Error(err); return }
    message := fmt.Sprintf("chain %s restarted", view.ExportId(chainId))
    svc.events.PostContestMessage(team.Contest_id, message)
    v := view.New(svc.model)
    err = v.ViewChain(userId, chainId)
    if err != nil { r.Error(err); return }
    r.Send(v.Flat())
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
    // fmt.Printf("Game key: %s\n", req.GameKey)
    // fmt.Printf("Chain id: %d\n", chainId)

    chain, err := svc.model.LoadChain(chainId)
    if err != nil { r.Error(err); return }

    var game *model.Game
    game, err = svc.model.LoadGame(req.GameKey)
    if err != nil { r.Error(err); return }
    if game == nil { r.StringError("no such game"); return }
    var block blocks.Block
    block, err = svc.store.ReadBlock(game.Last_block)
    if err != nil { r.Error(err); return }
    protocolHash := block.Base().Protocol

    var intf, impl []byte
    intf, impl, err = svc.store.LoadProtocol(protocolHash)
    if err != nil { r.Error(err); return }

    err = svc.model.SaveChainRevision(chain)
    if err != nil { r.Error(err); return }
    chain.Updated_at = time.Now()
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
