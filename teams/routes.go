
package teams

import (
  "github.com/gin-gonic/gin"
  "tezos-contests.izibi.com/backend/auth"
  "tezos-contests.izibi.com/backend/config"
  "tezos-contests.izibi.com/backend/model"
  "tezos-contests.izibi.com/backend/view"
  "tezos-contests.izibi.com/backend/utils"
)

type Service struct {
  config *config.Config
  model *model.Model
  auth *auth.Service
}

func NewService(config *config.Config, model *model.Model, auth *auth.Service) *Service {
  return &Service{config, model, auth}
}

func (svc *Service) Route(r gin.IRoutes) {

  r.POST("/Teams/:teamId/Leave", func (c *gin.Context) {
    r := utils.NewResponse(c)
    var err error
    userId, ok := auth.GetUserId(c)
    if !ok { r.BadUser(); return }
    teamId := view.ImportId(c.Param("teamId"))
    err = svc.model.LeaveTeam(teamId, userId)
    if err != nil { r.Error(err); return }
    r.Ok()
  })

  r.POST("/Teams/:teamId/AccessCode", func (c *gin.Context) {
    r := utils.NewResponse(c)
    v := view.New(svc.model)
    var err error
    userId, ok := auth.GetUserId(c)
    if !ok { r.BadUser(); return }
    teamId := view.ImportId(c.Param("teamId"))
    var team *model.Team
    team, err = svc.model.RenewTeamAccessCode(teamId, userId)
    if err != nil { r.Error(err); return }
    err = v.ViewUserContestTeam(userId, team.Contest_id)
    if err != nil { r.Error(err); return }
    r.Send(v.Flat())
  })

  r.POST("/Teams/:teamId/Update", func (c *gin.Context) {
    r := utils.NewResponse(c)
    v := view.New(svc.model)
    var err error
    userId, ok := auth.GetUserId(c)
    if !ok { r.BadUser(); return }
    teamId := view.ImportId(c.Param("teamId"))
    var arg model.UpdateTeamArg
    err = c.ShouldBindJSON(&arg)
    if err != nil { r.Error(err); return }
    var team *model.Team
    team, err = svc.model.UpdateTeam(teamId, userId, arg)
    if err != nil { r.Error(err); return }
    err = v.ViewUserContestTeam(userId, team.Contest_id)
    if err != nil { r.Error(err); return }
    r.Send(v.Flat())
  })

  r.GET("/TeamByCode/:code", func(c *gin.Context) {
    /*
      c := svc.Wrap(c)
      team by access code -> (id, name)
    */
  })

}
