
package chains

import (
  "fmt"
  "database/sql"
  "github.com/gin-gonic/gin"
  "tezos-contests.izibi.com/backend/auth"
  j "tezos-contests.izibi.com/backend/jase"
  "tezos-contests.izibi.com/backend/utils"
  "tezos-contests.izibi.com/backend/model"
  "tezos-contests.izibi.com/backend/config"
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

  r.POST("/Chains/:chainId/Fork", func(c *gin.Context) {
    ctx := svc.Wrap(c)
    userId, ok := auth.GetUserId(c)
    if !ok { ctx.resp.BadUser(); return }
    chainId := c.Param("chainId")
    id, err := ctx.model.ForkChain(userId, chainId)
    if err != nil { ctx.resp.Error(err); return }
    ctx.resp.Result(j.String(id))
  })

}
