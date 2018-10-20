
package chains

import (
  "fmt"
  "database/sql"
  "time"
  "github.com/gin-gonic/gin"
  "tezos-contests.izibi.com/backend/auth"
  j "tezos-contests.izibi.com/backend/jase"
  "tezos-contests.izibi.com/backend/config"
  "tezos-contests.izibi.com/backend/events"
  "tezos-contests.izibi.com/backend/blocks"
  "tezos-contests.izibi.com/backend/model"
  "tezos-contests.izibi.com/backend/utils"
)

type Service struct {
  config *config.Config
  db *sql.DB
  events *events.Service
  auth *auth.Service
  blockStore *blocks.Service
}

type Context struct {
  c *gin.Context
  resp *utils.Response
  model *model.Model
  auth *auth.Context
}

func NewService(config *config.Config, db *sql.DB, events *events.Service, auth *auth.Service, blockStore *blocks.Service) *Service {
  return &Service{config, db, events, auth, blockStore}
}

func (svc *Service) Wrap(c *gin.Context) *Context {
  m := model.New(c, svc.db)
  return &Context{
    c,
    utils.NewResponse(c),
    m,
    svc.auth.Wrap(c, m),
  }
}

func (svc *Service) Route(r gin.IRoutes) {

  r.POST("/Chains/:chainId/Fork", func(c *gin.Context) {
    ctx := svc.Wrap(c)
    userId, ok := ctx.auth.GetUserId()
    if !ok { ctx.resp.BadUser(); return }
    oldChainId := ctx.model.ImportId(c.Param("chainId"))
    oldChain, err := ctx.model.LoadChain(oldChainId, model.NullFacet)
    if err != nil { ctx.resp.Error(err); return }
    /*
      The user must belong to a team in contest chain.contest_id.
      TODO: quotas on number of private chains per team?
    */
    team, err := ctx.model.LoadUserContestTeam(userId, oldChain.Contest_id, model.NullFacet)
    if err != nil { ctx.resp.Error(err); return }
    if team == nil { ctx.resp.StringError("access denied"); return }

    newChainId, err := ctx.model.ForkChain(team.Id, oldChainId)
    if err != nil { ctx.resp.Error(err); return }

    /* Attempt to initialize a game on the new chain. */
    newChain, err := ctx.model.LoadChain(newChainId, model.NullFacet)
    if err != nil { ctx.resp.Error(err); return }
    oldGame, err := ctx.model.LoadGame(oldChain.Game_key, model.NullFacet)
    if err == nil {
      gameKey, err := ctx.model.CreateGame(newChain.Owner_id.Int64, oldGame.Last_block)
      if err == nil {
        _ = ctx.model.SetChainGameKey(newChainId, gameKey)
      }
    }

    /* XXX Temporary */
    message := fmt.Sprintf("chain %s created", ctx.model.ExportId(newChainId))
    svc.events.PostContestMessage(1, message)

    ctx.resp.Result(j.String(ctx.model.ExportId(newChainId)))
  })

  r.POST("/Chains/:chainId/Delete", func(c *gin.Context) {
    ctx := svc.Wrap(c)
    userId, ok := ctx.auth.GetUserId()
    if !ok { ctx.resp.BadUser(); return }
    chainId := ctx.model.ImportId(c.Param("chainId"))
    chain, err := ctx.model.DeleteChain(userId, chainId)
    if err != nil { ctx.resp.Error(err); return }

    message := fmt.Sprintf("chain %s deleted", ctx.model.ExportId(chain.Id))
    svc.events.PostContestMessage(1, message)
    /*
    if chain.Status_id == 1 { // XXX should query model to test if chain is private
      svc.events.PostTeamMessage(chain.Owner_id.Int64, message)
    } else {
      svc.events.PostContestMessage(chain.Contest_id, message)
    }
    */

    ctx.resp.Result(j.Boolean(true))
  })

  /* Install a game on a chain. */
  r.POST("/Chains/:chainId/ChangeGame", func(c *gin.Context) {
    var err error
    ctx := svc.Wrap(c)
    type Request struct {
      GameKey string `json:"gameKey"`
    }
    var req Request
    if c.Bind(&req) != nil { return }

    userId, ok := ctx.auth.GetUserId()
    if !ok { ctx.resp.BadUser(); return }
    if !ctx.model.IsUserAdmin(userId) { ctx.resp.StringError("Not Authorized") }

    chainId := ctx.model.ImportId(c.Param("chainId"))
    fmt.Printf("Game key: %s\n", req.GameKey)
    fmt.Printf("Chain id: %s\n", chainId)

    chain, err := ctx.model.LoadChain(chainId, model.NullFacet)
    if err != nil { ctx.resp.Error(err); return }

    var game *model.Game
    game, err = ctx.model.LoadGame(req.GameKey, model.NullFacet)
    if err != nil { ctx.resp.Error(err); return }
    if game == nil { ctx.resp.StringError("no such game"); return }
    var block blocks.Block
    block, err = svc.blockStore.ReadBlock(game.Last_block)
    if err != nil { ctx.resp.Error(err); return }
    protocolHash := block.Base().Protocol

    var intf, impl []byte
    intf, impl, err = svc.blockStore.LoadProtocol(protocolHash)
    if err != nil { ctx.resp.Error(err); return }

    now := time.Now().Format(time.RFC3339)
    chain.Updated_at = now
    chain.Started_at = sql.NullString{}
    chain.Interface_text = string(intf)
    chain.Implementation_text = string(impl)
    chain.Game_key = req.GameKey
    chain.Round = -1  // XXX not used?
    chain.Parent_id = sql.NullString{}
    chain.Protocol_hash = protocolHash
    chain.New_protocol_hash = protocolHash
    chain.Nb_votes_approve = 0
    chain.Nb_votes_reject = 0
    chain.Nb_votes_unknown = 0
    err = ctx.model.SaveChain(chain)
    if err != nil { ctx.resp.Error(err); return }

    ctx.resp.Result(j.Boolean(true))
  })

}
