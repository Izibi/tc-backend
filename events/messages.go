
package events

type contestMessage struct {
  contestId int64
  payload string
}

type teamMessage struct {
  teamId int64
  payload string
}

type gameMessage struct {
  gameKey string
  payload string
}

func (svc *Service) PostContestMessage(contestId int64, payload string) {
  svc.channel <- &contestMessage{contestId: contestId, payload: payload}
}

func (svc *Service) PostTeamMessage(teamId int64, payload string) {
  svc.channel <- &teamMessage{teamId: teamId, payload: payload}
}

func (svc *Service) PostGameMessage(gameKey string, payload string) {
  svc.channel <- &gameMessage{gameKey: gameKey, payload: payload}
}
