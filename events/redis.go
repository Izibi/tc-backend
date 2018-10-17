
package events

import (
  "bytes"
  crand "crypto/rand"
  "encoding/binary"
  "encoding/base64"
  "fmt"
  "io"
  "math/rand"
  "strings"
  "sync"
  "github.com/go-redis/redis"
  "github.com/fatih/color"
  "tezos-contests.izibi.com/backend/config"
  j "tezos-contests.izibi.com/backend/jase"
)

const (
  verbose = true
)

var hi1 = color.New(color.Bold, color.FgCyan)
var hi2 = color.New(color.Bold, color.FgBlue)

type Service struct {
  config *config.Config
  client *redis.Client
  rng *rand.Rand
  clients map[string]*redis.PubSub /* key -> pubsub instance */
  mutex sync.RWMutex
}

type EventSink struct {
  w io.Writer
  last *SSEvent
}

func NewService(cfg *config.Config, client *redis.Client) (*Service, error) {
  var err error
  rng, err := seededRng()
  if err != nil { return nil, err }
  return &Service{cfg, client, rng, map[string]*redis.PubSub{}, sync.RWMutex{}}, nil
}

func (svc *Service) newClient(serverUrl string) (string, *redis.PubSub, error) {
  keyBytes := make([]byte, 32, 32)
  _, err := svc.rng.Read(keyBytes)
  if err != nil { return "", nil, err }
  key := base64.RawURLEncoding.EncodeToString(keyBytes[:])
  ps := svc.client.Subscribe("system")
  err = svc.client.Set(key, serverUrl, 0).Err()
  if err != nil { return "", nil, err }
  {
    svc.mutex.Lock()
    svc.clients[key] = ps
    svc.mutex.Unlock()
  }
  if verbose {
    hi1.Printf("+ %s\n", key)
  }
  return key, ps, nil
}

func (svc *Service) getClient(key string) (*redis.PubSub, string) {
  var ps *redis.PubSub
  var ok bool
  {
    svc.mutex.RLock()
    ps, ok = svc.clients[key]
    svc.mutex.RUnlock()
  }
  if !ok {
    serverUrl, err := svc.client.Get(key).Result()
    if err != nil { return nil, "" }
    return nil, serverUrl
  }
  return ps, ""
}

func (svc *Service) removeClient(key string) {
  if verbose {
    hi2.Printf("- %s\n", key)
  }
  var ps *redis.PubSub
  {
    svc.mutex.Lock()
    var ok bool
    ps, ok = svc.clients[key]
    if ok {
      delete(svc.clients, key)
    }
    svc.mutex.Unlock()
  }
  if ps != nil {
    _ = ps.Close()
    _ = svc.client.Del(key).Err()
  }
}

func (svc *Service) Publish(channel string, message string) error {
  return svc.client.Publish(channel, message).Err()
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

type SSEvent struct {
  Id string
  Event string
  Data string
}

func (sink *EventSink) Write(m *SSEvent) {
  var buf bytes.Buffer
  if len(m.Id) > 0 && (sink.last == nil || m.Id != sink.last.Id) {
    buf.WriteString(fmt.Sprintf("id: %s\n", noLF(m.Id)))
  }
  if len(m.Event) > 0 && (sink.last == nil || m.Event != sink.last.Event)  {
    buf.WriteString(fmt.Sprintf("event: %s\n", noLF(m.Event)))
  }
  if len(m.Data) > 0 {
    lines := strings.Split(m.Data, "\n")
    for _, line := range lines {
      buf.WriteString(fmt.Sprintf("data: %s\n", line))
    }
  }
  buf.WriteString("\n")
  sink.w.Write(buf.Bytes())
  sink.last = m
}

func noLF(s string) string {
  return strings.Replace(s, "\n", "", -1)
}

func encodeMessage(channel string, payload string) string {
  var obj = j.Object()
  obj.Prop("channel", j.String(channel))
  obj.Prop("payload", j.String(payload))
  res, err := j.ToString(obj)
  if err != nil { panic(err) }
  return res
}
