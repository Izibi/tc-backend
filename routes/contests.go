
package routes

import (
  "github.com/gin-gonic/gin"
  "tezos-contests.izibi.com/backend/auth"
  "tezos-contests.izibi.com/backend/view"
  "tezos-contests.izibi.com/backend/utils"
)

func (svc *Service) RouteContests(r gin.IRoutes) {

  r.GET("/Contests/:contestId", func(c *gin.Context) {
    r := utils.NewResponse(c)
    v := view.New(svc.model)
    userId, ok := auth.GetUserId(c)
    if !ok { r.BadUser(); return }
    contestId := view.ImportId(c.Param("contestId"))
    err := v.ViewUserContest(userId, contestId)
    if err != nil { r.Error(err); return }
    r.Send(v.Flat())
  })

  r.GET("/Contests/:contestId/Team", func(c *gin.Context) {
    r := utils.NewResponse(c)
    v := view.New(svc.model)
    userId, ok := auth.GetUserId(c)
    if !ok { r.BadUser(); return }
    contestId := view.ImportId(c.Param("contestId"))
    err := v.ViewUserContestTeam(userId, contestId)
    if err != nil { r.Error(err); return }
    r.Send(v.Flat())
  })

  r.POST("/Contests/:contestId/CreateTeam", func(c *gin.Context) {
    r := utils.NewResponse(c)
    v := view.New(svc.model)
    var err error
    userId, ok := auth.GetUserId(c)
    if !ok { r.BadUser(); return }
    contestId := view.ImportId(c.Param("contestId"))
    type Body struct {
      TeamName string `json:"teamName"`
    }
    var body Body
    err = c.ShouldBindJSON(&body)
    if err != nil { r.Error(err); return }
    err = svc.model.CreateTeam(userId, contestId, body.TeamName)
    if err != nil { r.Error(err); return }
    err = v.ViewUserContestTeam(userId, contestId)
    if err != nil { r.Error(err); return }
    r.Send(v.Flat())
  })

  r.POST("/Contests/:contestId/JoinTeam", func(c *gin.Context) {
    r := utils.NewResponse(c)
    v := view.New(svc.model)
    var err error
    userId, ok := auth.GetUserId(c)
    if !ok { r.BadUser(); return }
    contestId := view.ImportId(c.Param("contestId"))
    type Body struct {
      AccessCode string `json:"accessCode"`
    }
    var body Body
    err = c.ShouldBindJSON(&body)
    if err != nil { r.Error(err); return }
    err = svc.model.JoinTeam(userId, contestId, body.AccessCode)
    if err != nil { r.Error(err); return }
    err = v.ViewUserContestTeam(userId, contestId)
    if err != nil { r.Error(err); return }
    r.Send(v.Flat())
  })

}
