
package contests

import (
  "database/sql"
  "github.com/gin-gonic/gin"
  "tezos-contests.izibi.com/backend/auth"
  "tezos-contests.izibi.com/backend/model"
  "tezos-contests.izibi.com/backend/utils"
)

func SetupRoutes(r gin.IRoutes, db *sql.DB) {

  r.GET("/Contests/:contestId", func(c *gin.Context) {
    resp := utils.NewResponse(c)
    userId, ok := auth.GetUserId(c)
    if !ok { resp.BadUser(); return }
    m := model.New(c, db)
    contestId := c.Param("contestId")
    err := m.ViewUserContest(userId, contestId)
    if err != nil { resp.Error(err); return }
    resp.Send(m.Flat())
  })

  r.GET("/Contests/:contestId/Team", func(c *gin.Context) {
    resp := utils.NewResponse(c)
    userId, ok := auth.GetUserId(c)
    if !ok { resp.BadUser(); return }
    m := model.New(c, db)
    contestId := c.Param("contestId")
    err := m.ViewUserContestTeam(userId, contestId)
    if err != nil { resp.Error(err); return }
    resp.Send(m.Flat())
  })

  r.POST("/Contests/:contestId/CreateTeam", func(c *gin.Context) {
    var err error
    resp := utils.NewResponse(c)
    userId, ok := auth.GetUserId(c)
    if !ok { resp.BadUser(); return }
    contestId := c.Param("contestId")
    type Body struct {
      TeamName string `json:"teamName"`
    }
    var body Body
    err = c.ShouldBindJSON(&body)
    if err != nil { resp.Error(err); return }
    m := model.New(c, db)
    err = m.CreateTeam(userId, contestId, body.TeamName)
    if err != nil { resp.Error(err); return }
    resp.Send(m.Flat())
  })

  r.POST("/Contests/:contestId/JoinTeam", func(c *gin.Context) {
    var err error
    resp := utils.NewResponse(c)
    userId, ok := auth.GetUserId(c)
    if !ok { resp.BadUser(); return }
    contestId := c.Param("contestId")
    type Body struct {
      AccessCode string `json:"accessCode"`
    }
    var body Body
    err = c.ShouldBindJSON(&body)
    if err != nil { resp.Error(err); return }
    m := model.New(c, db)
    err = m.JoinTeam(userId, contestId, body.AccessCode)
    if err != nil { resp.Error(err); return }
    resp.Send(m.Flat())
  })

  r.GET("/Contests/:contestId/Chains", func(c *gin.Context) {
    resp := utils.NewResponse(c)
    userId, ok := auth.GetUserId(c)
    if !ok { resp.BadUser(); return }
    contestId := c.Param("contestId")
    m := model.New(c, db)
    err := m.ViewChains(userId, contestId, model.ChainFilters{})
    if err != nil { resp.Error(err); return }
    resp.Send(m.Flat())
  })

}
