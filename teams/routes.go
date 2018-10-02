
package teams

import (
  "database/sql"
  "github.com/gin-gonic/gin"
  "tezos-contests.izibi.com/backend/auth"
  "tezos-contests.izibi.com/backend/model"
  "tezos-contests.izibi.com/backend/utils"
)

func SetupRoutes(r gin.IRoutes, newApi utils.NewApi, db *sql.DB) {

  r.POST("/Teams/:teamId/Leave", func(c *gin.Context) {
    var err error
    api := newApi(c)
    userId, ok := auth.GetUserId(c)
    if !ok { api.BadUser(); return }
    teamId := c.Param("teamId")
    m := model.New(c, db)
    err = m.LeaveTeam(teamId, userId)
    if err != nil { api.Error(err); return }
    api.Send(m.Flat())
  })

  r.POST("/Teams/:teamId/AccessCode", func(c *gin.Context) {
    var err error
    api := newApi(c)
    userId, ok := auth.GetUserId(c)
    if !ok { api.BadUser(); return }
    teamId := c.Param("teamId")
    m := model.New(c, db)
    err = m.RenewTeamAccessCode(teamId, userId)
    if err != nil { api.Error(err); return }
    api.Send(m.Flat())
  })

  r.POST("/Teams/:teamId/Update", func(c *gin.Context) {
    var err error
    api := newApi(c)
    userId, ok := auth.GetUserId(c)
    if !ok { api.BadUser(); return }
    teamId := c.Param("teamId")
    var arg model.UpdateTeamArg
    err = c.ShouldBindJSON(&arg)
    if err != nil { api.Error(err); return }
    m := model.New(c, db)
    err = m.UpdateTeam(teamId, userId, arg)
    if err != nil { api.Error(err); return }
    api.Send(m.Flat())
  })

}
