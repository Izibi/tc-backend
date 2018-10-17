
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
}

type Context struct {
  c *gin.Context
  resp *utils.Response
  model *model.Model
}

func NewService(config *config.Config, db *sql.DB) *Service {
  return &Service{config, db}
}

func (svc *Service) Wrap(c *gin.Context) *Context {
  return &Context{
    c,
    utils.NewResponse(c),
    model.New(c, svc.db),
  }
}

func (svc *Service) Route(r gin.IRoutes) {

  r.POST("/Teams/:teamId/Leave", func(c *gin.Context) {
    ctx := svc.Wrap(c)
    var err error
    userId, ok := auth.GetUserId(c)
    if !ok { ctx.resp.BadUser(); return }
    teamId := c.Param("teamId")
    err = ctx.model.LeaveTeam(teamId, userId)
    if err != nil { ctx.resp.Error(err); return }
    ctx.resp.Send(ctx.model.Flat())
  })

  r.POST("/Teams/:teamId/AccessCode", func(c *gin.Context) {
    ctx := svc.Wrap(c)
    var err error
    userId, ok := auth.GetUserId(c)
    if !ok { ctx.resp.BadUser(); return }
    teamId := c.Param("teamId")
    err = ctx.model.RenewTeamAccessCode(teamId, userId)
    if err != nil { ctx.resp.Error(err); return }
    ctx.resp.Send(ctx.model.Flat())
  })

  r.POST("/Teams/:teamId/Update", func(c *gin.Context) {
    ctx := svc.Wrap(c)
    var err error
    userId, ok := auth.GetUserId(c)
    if !ok { ctx.resp.BadUser(); return }
    teamId := c.Param("teamId")
    var arg model.UpdateTeamArg
    err = c.ShouldBindJSON(&arg)
    if err != nil { ctx.resp.Error(err); return }
    err = ctx.model.UpdateTeam(teamId, userId, arg)
    if err != nil { ctx.resp.Error(err); return }
    ctx.resp.Send(ctx.model.Flat())
  })

}
