
package routes

import (
  "github.com/gin-gonic/gin"
  "tezos-contests.izibi.com/backend/auth"
  "tezos-contests.izibi.com/backend/view"
  "tezos-contests.izibi.com/backend/utils"
)

func (svc *Service) RouteLanding(r gin.IRoutes) {

  r.GET("/AuthenticatedUserLanding", func(c *gin.Context) {
    r := utils.NewResponse(c)
    v := view.New(svc.model)
    var err error
    userId, ok := auth.GetUserId(c)
    if !ok { r.BadUser(); return }
    err = v.ViewUser(userId)
    if err != nil { r.Error(err); return }
    err = v.ViewUserContests(userId)
    if err != nil { r.Error(err); return }
    r.Send(v.Flat())
  })

}
