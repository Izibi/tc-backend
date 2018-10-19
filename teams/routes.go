
package teams

import (
  "database/sql"
  "github.com/gin-gonic/gin"
  "tezos-contests.izibi.com/backend/auth"
  "tezos-contests.izibi.com/backend/config"
  "tezos-contests.izibi.com/backend/model"
  "tezos-contests.izibi.com/backend/utils"
)

type Service struct {
  config *config.Config
  db *sql.DB
  auth *auth.Service
}

type Context struct {
  c *gin.Context
  resp *utils.Response
  model *model.Model
  auth *auth.Context
}

func NewService(config *config.Config, db *sql.DB, auth *auth.Service) *Service {
  return &Service{config, db, auth}
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

  r.POST("/Teams/:teamId/Leave", func(c *gin.Context) {
    ctx := svc.Wrap(c)
    var err error
    userId, ok := ctx.auth.GetUserId()
    if !ok { ctx.resp.BadUser(); return }
    teamId := ctx.model.ImportId(c.Param("teamId"))
    err = ctx.model.LeaveTeam(teamId, userId)
    if err != nil { ctx.resp.Error(err); return }
    ctx.resp.Send(ctx.model.Flat())
  })

  r.POST("/Teams/:teamId/AccessCode", func(c *gin.Context) {
    ctx := svc.Wrap(c)
    var err error
    userId, ok := ctx.auth.GetUserId()
    if !ok { ctx.resp.BadUser(); return }
    teamId := ctx.model.ImportId(c.Param("teamId"))
    err = ctx.model.RenewTeamAccessCode(teamId, userId)
    if err != nil { ctx.resp.Error(err); return }
    ctx.resp.Send(ctx.model.Flat())
  })

  r.POST("/Teams/:teamId/Update", func(c *gin.Context) {
    ctx := svc.Wrap(c)
    var err error
    userId, ok := ctx.auth.GetUserId()
    if !ok { ctx.resp.BadUser(); return }
    teamId := ctx.model.ImportId(c.Param("teamId"))
    var arg model.UpdateTeamArg
    err = c.ShouldBindJSON(&arg)
    if err != nil { ctx.resp.Error(err); return }
    err = ctx.model.UpdateTeam(teamId, userId, arg)
    if err != nil { ctx.resp.Error(err); return }
    ctx.resp.Send(ctx.model.Flat())
  })

}
