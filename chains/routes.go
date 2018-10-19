
package chains

import (
  "fmt"
  "database/sql"
  "github.com/gin-gonic/gin"
  "tezos-contests.izibi.com/backend/auth"
  j "tezos-contests.izibi.com/backend/jase"
  "tezos-contests.izibi.com/backend/config"
  "tezos-contests.izibi.com/backend/events"
  "tezos-contests.izibi.com/backend/model"
  "tezos-contests.izibi.com/backend/utils"
)

type Service struct {
  config *config.Config
  db *sql.DB
  events *events.Service
  auth *auth.Service
}

type Context struct {
  c *gin.Context
  resp *utils.Response
  model *model.Model
  auth *auth.Context
}

func NewService(config *config.Config, db *sql.DB, events *events.Service, auth *auth.Service) *Service {
  return &Service{config, db, events, auth}
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

}
