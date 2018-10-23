
package events

import (
  "bytes"
  crand "crypto/rand"
  "encoding/binary"
  "errors"
  "math/rand"
  "sync"
  "time"
  "github.com/go-redis/redis"
  "github.com/fatih/color"
  "tezos-contests.izibi.com/backend/config"
  "tezos-contests.izibi.com/backend/model"
  "tezos-contests.izibi.com/backend/auth"
)

const (
  verbose = true
)
var MaxStreamIdleDuration = 5 * time.Minute
var RedisStreamKeyExpiry = 5 * time.Minute
var RedisStreamKeyRefresh = 4 * time.Minute

var hi1 = color.New(color.Bold, color.FgCyan)
var hi2 = color.New(color.Bold, color.FgBlue)

type Service struct {
  config *config.Config
  redis *redis.Client
  model *model.Model
  auth *auth.Service
  rng *rand.Rand
  channel chan interface{}
  mutex sync.RWMutex
  idleStreams map[string]*stream
  streams map[string]*stream
}

func NewService(cfg *config.Config, rc *redis.Client, model *model.Model, auth *auth.Service) (*Service, error) {
  var err error
  rng, err := seededRng()
  if err != nil { return nil, err }
  return &Service{
    cfg,
    rc,
    model,
    auth,
    rng,
    make(chan interface{}),
    sync.RWMutex{},
    map[string]*stream{},
    map[string]*stream{},
  }, nil
}

func (svc *Service) Run() {
  /* This method is intended zbq7L2cSHydtb414go1gX5n03y1dkBfc-bht_5HJ72Ato be invoked as a go routine.
     It reads (contest, team, game) messages from a channel and ensures that
     they reach their intended audience.
   */
  periodic := time.NewTicker(10 * time.Second)
  for {
    select {
      case _ = <-periodic.C:
        svc.periodicTask()
      case m := <-svc.channel:
        _ = svc.handleMessage(m)
    }
  }
}

func (svc *Service) periodicTask() {
  /* Handle the periodic cleanup of idle event streams. A stream is considered
     idle if it has been disconnected for a period of time. */
  var streams []*stream
  var now = time.Now()
  svc.mutex.Lock()
  for _, st := range svc.idleStreams {
    if now.Sub(st.idleSince) > MaxStreamIdleDuration {
      streams = append(streams, st)
    }
  }
  for _, st := range streams {
    delete(svc.idleStreams, st.key)
  }
  svc.mutex.Unlock()
  for _, st := range streams {
    _ = st.pubSub.Close()
    _ = svc.redis.Del(streamKey(st.key)).Err()
  }
}

func (svc *Service) handleMessage(msg interface{}) error {
  var err error
  var key string
  var payload string
  switch m := msg.(type) {
  case *contestMessage:
    payload = m.payload
    key, err = svc.getContestChannel(m.contestId)
    if err != nil { return err }
  case *teamMessage:
    payload = m.payload
    key, err = svc.getTeamChannel(m.teamId)
    if err != nil { return err }
  case *gameMessage:
    payload = m.payload
    key, err = svc.getGameChannel(m.gameKey)
  default:
    return errors.New("unhandled message type")
  }
  err = svc.redis.Publish(key, payload).Err()
  if err != nil { return err }
  return nil
}

func seededRng() (*rand.Rand, error) {
  var err error
  bs := make([]byte, 8, 8)
  _, err = crand.Read(bs)
  if err != nil { return nil, err }
  var seed int64
  err = binary.Read(bytes.NewBuffer(bs), binary.LittleEndian, &seed)
  if err != nil { return nil, err }
  return rand.New(rand.NewSource(seed)), nil
}
