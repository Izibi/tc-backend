
package chains

import (
  "database/sql"
  "github.com/gin-gonic/gin"
  "tezos-contests.izibi.com/backend/auth"
  "tezos-contests.izibi.com/backend/model"
  j "tezos-contests.izibi.com/backend/jase"
  "tezos-contests.izibi.com/backend/utils"
)

func SetupRoutes(r gin.IRoutes, newApi utils.NewApi, db *sql.DB) {

  r.POST("/Chains/:chainId/Fork", func(c *gin.Context) {
    api := newApi(c)
    userId, ok := auth.GetUserId(c)
    if !ok { api.BadUser(); return }
    chainId := c.Param("chainId")
    m := model.New(c, db)
    id, err := m.ForkChain(userId, chainId)
    if err != nil { api.Error(err); return }
    api.Result(j.String(id))
  })

}
