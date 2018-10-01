
package teams

import (
  "database/sql"
  "github.com/gin-gonic/gin"
  "tezos-contests.izibi.com/backend/auth"
  "tezos-contests.izibi.com/backend/model"
  "tezos-contests.izibi.com/backend/utils"
)

func SetupRoutes(r gin.IRoutes, db *sql.DB) {

  r.POST("/Teams/:teamId/Leave", func(c *gin.Context) {
    var err error
    resp := utils.NewResponse(c)
    userId, ok := auth.GetUserId(c)
    if !ok { resp.BadUser(); return }
    teamId := c.Param("teamId")
    m := model.New(db)
    err = m.LeaveTeam(teamId, userId)
    if err != nil { resp.Error(err); return }
    resp.Send(m.Flat())
  })

  r.POST("/Teams/:teamId/AccessCode", func(c *gin.Context) {
    var err error
    resp := utils.NewResponse(c)
    userId, ok := auth.GetUserId(c)
    if !ok { resp.BadUser(); return }
    teamId := c.Param("teamId")
    m := model.New(db)
    err = m.RenewTeamAccessCode(teamId, userId)
    if err != nil { resp.Error(err); return }
    resp.Send(m.Flat())
  })

  r.POST("/Teams/:teamId/Update", func(c *gin.Context) {
    var err error
    resp := utils.NewResponse(c)
    userId, ok := auth.GetUserId(c)
    if !ok { resp.BadUser(); return }
    teamId := c.Param("teamId")
    var arg model.UpdateTeamArg
    err = c.ShouldBindJSON(&arg)
    if err != nil { resp.Error(err); return }
    m := model.New(db)
    err = m.UpdateTeam(teamId, userId, arg)
    if err != nil { resp.Error(err); return }
    resp.Send(m.Flat())
  })

}
