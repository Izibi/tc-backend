
package routes

import (
  "errors"
  "github.com/gin-gonic/gin"
  "github.com/go-redis/redis"
  "tezos-contests.izibi.com/backend/auth"
  "tezos-contests.izibi.com/backend/blocks"
  "tezos-contests.izibi.com/backend/config"
  "tezos-contests.izibi.com/backend/events"
  "tezos-contests.izibi.com/backend/model"
  "tezos-contests.izibi.com/backend/utils"
)

type Service struct {
  config *config.Config
  rc *redis.Client
  model *model.Model
  auth *auth.Service
  events *events.Service
  store *blocks.Service
}

func NewService(config *config.Config, rc *redis.Client, model *model.Model, auth *auth.Service, events *events.Service, store *blocks.Service) *Service {
  return &Service{
    config: config,
    rc: rc,
    model: model,
    auth: auth,
    events: events,
    store: store,
  }
}

func (svc *Service) RouteAll(r gin.IRoutes) {
  svc.RouteChains(r)
  svc.RouteContests(r)
  svc.RouteGames(r)
  svc.RouteLanding(r)
  svc.RouteTeams(r)
}

func (svc *Service) signedRequest(c *gin.Context, req interface{}) (*utils.Response, error) {
  r := utils.NewResponse(c)
  err := utils.NewRequest(c, svc.config.ApiKey).Signed(req)
  return r, err
}

func (svc *Service) checkAuthor(author string) (int64, error) {
  teamId, err := svc.model.FindTeamIdByKey(author[1:])
  if err != nil { return 0, err }
  if teamId == 0 { return 0, errors.New("team key is not recognized") }
  return teamId, nil
}
