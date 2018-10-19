
package events

import (
  "fmt"
  "encoding/base64"
  "errors"
  "strconv"
  "time"
  "github.com/go-redis/redis"
  "github.com/fatih/color"
)

var SuccessFmt = color.New(color.Bold, color.FgGreen)

type stream struct {
  svc *Service
  key string
  idleSince time.Time
  serverUrl string
  pubSub *redis.PubSub
  lastId uint64 /* greatest id of events prepared */
  recvId uint64 /* greatest id received by client, set on reconnect */
  userId int64
  teamId int64
  contestId int64
  recent []*SSEvent
}

type SSEvent struct {
  Id uint64
  Event string
  Data string
  Timestamp time.Time
}

func (svc *Service) newStream() (*stream, error) {
  keyBytes := make([]byte, 32, 32)
  _, err := svc.rng.Read(keyBytes)
  if err != nil { return nil, err }
  key := base64.RawURLEncoding.EncodeToString(keyBytes[:])
  var st *stream = &stream{
    svc: svc,
    key: key,
    idleSince: time.Now(),
    serverUrl: svc.config.SelfUrl,
    pubSub: svc.redis.Subscribe("system"),
    lastId: 1,
    recvId: 0,
    userId: 0,
    teamId: 0,
    contestId: 0,
    recent: nil,
  }
  err = svc.redis.Set(streamKey(key), st.serverUrl, 5 * time.Minute).Err()
  if err != nil { return nil, err }
  {
    svc.mutex.Lock()
    svc.idleStreams[key] = st
    svc.mutex.Unlock()
  }
  return st, nil
}

func (svc *Service) connectStream(key string, recvId string) (*stream, bool, error) {
  var err error
  var st *stream
  var idleFound, activeFound bool
  {
    svc.mutex.Lock()
    st, activeFound = svc.streams[key]
    if !activeFound {
      st, idleFound = svc.idleStreams[key]
      if idleFound {
        delete(svc.idleStreams, key)
        svc.streams[key] = st
      }
    }
    svc.mutex.Unlock()
  }
  if activeFound {
    /* Found but already connected. */
    return st, false, nil
  }
  if !idleFound {
    /* Atomically mark ourselves as the stream controller in redis. */
    var serverUrl string
    serverUrl, err = svc.redis.GetSet(streamKey(key), svc.config.SelfUrl).Result()
    if err != nil { return nil, false, err }
    if serverUrl != svc.config.SelfUrl {
      /* TODO: transfer state from serverUrl */
      return nil, false, errors.New("not implemented")
    }
    /* Client reconnecting to the same server. */
    fmt.Printf("XXX transfer state from %s\n", serverUrl)
    var lastId uint64
    st = &stream{
      svc: svc,
      key: key,
      idleSince: time.Now(),
      serverUrl: svc.config.SelfUrl,
      pubSub: svc.redis.Subscribe("system"),
      lastId: lastId,
      recvId: 0,
      userId: 0,
      teamId: 0,
      contestId: 0,
      recent: nil,
    }
    {
      svc.mutex.Lock()
      svc.streams[key] = st
      svc.mutex.Unlock()
    }
  }
  st.recvId, _ = strconv.ParseUint(recvId, 10, 64)
  if verbose {
    hi1.Printf("+ %s\n", key)
  }
  return st, true, nil
}

func (svc *Service) disconnectStream(st *stream) error {
  if verbose {
    hi1.Printf("- %s\n", st.key)
  }
  svc.mutex.Lock()
  svc.idleStreams[st.key] = st
  delete(svc.streams, st.key)
  svc.mutex.Unlock()
  /* TODO: save the client state in redis? */
  return nil
}

func (svc *Service) refreshStream(st *stream) error {
  return svc.redis.Expire(streamKey(st.key), RedisStreamKeyExpiry).Err()
}

func (svc *Service) getStream(key string) (*stream, error) {
  var st *stream
  var ok bool
  {
    svc.mutex.RLock()
    /* TODO: search in idle clients */
    st, ok = svc.streams[key]
    svc.mutex.RUnlock()
  }
  if !ok {
    /* TODO: acquire control of the stream */
    return nil, errors.New("not implemented")
  }
  return st, nil
}

/* Call to build an event that will be sent to the client.
   Not synchronized, call only from inside the http stream writer.
 */
func (st *stream) Push(msg string) *SSEvent {

  SuccessFmt.Printf("Push %d %d\n", st.recvId, st.lastId)

  st.lastId++
  now := time.Now()
  event := &SSEvent{
    Id: st.lastId,
    Event: "message",
    Data: msg,
    Timestamp: now,
  }

  /* Keep up to 1 minute of recent events. */
  st.recent = append(st.recent, event)
  cutoff := now.Add(-60 * time.Second)
  var i int
  var ev *SSEvent
  for i, ev = range st.recent {
    if ev.Timestamp.After(cutoff) {
      break
    }
  }
  st.recent = st.recent[i:]

  return event
}

func (st *stream) Resend() *SSEvent {
  if st.recvId != 0 {
    var ev *SSEvent
    for _, ev = range st.recent {
      if ev.Id > st.recvId {
        /* Do not truncate the recent events list in case the client
           disconnects (again). */
        SuccessFmt.Printf("Resending event %d > %d\n", ev.Id, st.recvId)
        st.recvId = ev.Id
        return ev
      }
    }
    st.recvId = 0
  }
  return nil
}

func (st *stream) Subscribe(channels ...string) error {
  err := st.pubSub.Subscribe(channels...)
  if err != nil { return err }
  return nil
}

func (st *stream) Unsubscribe(channels ...string) error {
  err := st.pubSub.Unsubscribe(channels...)
  if err != nil { return err }
  return nil
}

func (st *stream) SetUserId(userId int64) error {
  st.userId = userId
  return nil
}

func (st *stream) SetTeamId(teamId int64) error {
  var err error
  if st.teamId != 0 {
    var ch string
    ch, err := st.svc.getTeamChannel(st.teamId)
    if err != nil {
      _ = st.pubSub.Unsubscribe(ch)
    }
  }
  st.teamId = teamId
  if st.teamId != 0 {
    var ch string
    ch, err = st.svc.getTeamChannel(st.teamId)
    if err != nil { return err }
    err = st.pubSub.Subscribe(ch)
    if err != nil { return err }
  }
  return nil
}
