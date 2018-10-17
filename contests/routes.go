
package contests

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

  r.GET("/Contests/:contestId", func(c *gin.Context) {
    ctx := svc.Wrap(c)
    userId, ok := auth.GetUserId(c)
    if !ok { ctx.resp.BadUser(); return }
    contestId := c.Param("contestId")
    err := ctx.model.ViewUserContest(userId, contestId)
    if err != nil { ctx.resp.Error(err); return }
    ctx.resp.Send(ctx.model.Flat())
  })

  r.GET("/Contests/:contestId/Team", func(c *gin.Context) {
    ctx := svc.Wrap(c)
    userId, ok := auth.GetUserId(c)
    if !ok { ctx.resp.BadUser(); return }
    contestId := c.Param("contestId")
    err := ctx.model.ViewUserContestTeam(userId, contestId)
    if err != nil { ctx.resp.Error(err); return }
    ctx.resp.Send(ctx.model.Flat())
  })

  r.POST("/Contests/:contestId/CreateTeam", func(c *gin.Context) {
    var err error
    ctx := svc.Wrap(c)
    userId, ok := auth.GetUserId(c)
    if !ok { ctx.resp.BadUser(); return }
    contestId := c.Param("contestId")
    type Body struct {
      TeamName string `json:"teamName"`
    }
    var body Body
    err = c.ShouldBindJSON(&body)
    if err != nil { ctx.resp.Error(err); return }
    err = ctx.model.CreateTeam(userId, contestId, body.TeamName)
    if err != nil { ctx.resp.Error(err); return }
    ctx.resp.Send(ctx.model.Flat())
  })

  r.POST("/Contests/:contestId/JoinTeam", func(c *gin.Context) {
    var err error
    ctx := svc.Wrap(c)
    userId, ok := auth.GetUserId(c)
    if !ok { ctx.resp.BadUser(); return }
    contestId := c.Param("contestId")
    type Body struct {
      AccessCode string `json:"accessCode"`
    }
    var body Body
    err = c.ShouldBindJSON(&body)
    if err != nil { ctx.resp.Error(err); return }
    err = ctx.model.JoinTeam(userId, contestId, body.AccessCode)
    if err != nil { ctx.resp.Error(err); return }
    ctx.resp.Send(ctx.model.Flat())
  })

  r.GET("/Contests/:contestId/Chains", func(c *gin.Context) {
    ctx := svc.Wrap(c)
    userId, ok := auth.GetUserId(c)
    if !ok { ctx.resp.BadUser(); return }
    contestId := c.Param("contestId")
    err := ctx.model.ViewChains(userId, contestId, model.ChainFilters{})
    if err != nil { ctx.resp.Error(err); return }
    ctx.resp.Send(ctx.model.Flat())
  })

}
