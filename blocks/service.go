
package blocks

import (
  "path/filepath"
  "github.com/go-redis/redis"
  "tezos-contests.izibi.com/backend/config"
)

type Service struct {
  config *config.Config
  redis *redis.Client
}

func NewService(cfg *config.Config, rc *redis.Client) *Service {
  return &Service{cfg, rc}
}

func (svc *Service) taskToolsPath(taskBlockHash string) string {
  return filepath.Join(svc.config.Blocks.Path, taskBlockHash, svc.config.Blocks.TaskToolsCmd)
}

func (svc *Service) taskHelperPath(taskBlockHash string) string {
  return filepath.Join(svc.config.Blocks.Path, taskBlockHash, svc.config.Blocks.TaskHelperCmd)
}

func (svc *Service) blockDir(hash string) string {
  return filepath.Join(svc.config.Blocks.Path, hash)
}
