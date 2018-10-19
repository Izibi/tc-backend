/*
  Naming of redis channels.
*/

package events

import (
  "fmt"
)

func (svc *Service) getContestChannel(contestId int64) (string, error) {
  return fmt.Sprintf("contest:%d", contestId), nil
}

func (svc *Service) getTeamChannel(teamId int64) (string, error) {
  return fmt.Sprint("team:%d", teamId), nil
}

func (svc *Service) getGameChannel(gameKey string) (string, error) {
  return fmt.Sprintf("game:%s", gameKey), nil
}
