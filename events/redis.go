
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
  "github.com/gin-gonic/gin"
  "github.com/go-redis/redis"
  "tezos-contests.izibi.com/backend/utils"
  "github.com/fatih/color"
  j "tezos-contests.izibi.com/backend/jase"
)

const (
  verbose = true
)

var hi1 = color.New(color.Bold, color.FgCyan)
var hi2 = color.New(color.Bold, color.FgBlue)

type Service struct {
  selfUrl string
  client *redis.Client
  rng *rand.Rand
  clients map[string]*redis.PubSub /* key -> pubsub instance */
  mutex sync.RWMutex
}

type Message struct {
  Channel string
  Payload string
}

func NewService(selfUrl string) (*Service, error) {
  var err error
  client := redis.NewClient(&redis.Options{
    Addr:     "localhost:6379",
    Password: "", // no password set
    DB:       0,  // use default DB
  })
  err = client.Ping().Err()
  if err != nil { return nil, err }
  rng, err := seededRng()
  if err != nil { return nil, err }
  return &Service{selfUrl, client, rng, map[string]*redis.PubSub{}, sync.RWMutex{}}, nil
}

func (svc *Service) SetupRoutes(router gin.IRoutes, newApi utils.NewApi) {
  router.GET("/Events", func (c *gin.Context) {
    c.Header("Content-Type", "text/event-stream")
    c.Header("Cache-Control", "no-cache")
    c.Header("Connection", "keep-alive")
    c.Header("X-Accel-Buffering", "no")
    var key string
    var source <-chan *redis.Message
    clientGone := c.Writer.CloseNotify()
    c.Stream(func (w io.Writer) bool {
      var err error
      if source == nil {
        var ps *redis.PubSub
        key, ps, err = svc.newClient(svc.selfUrl)
        if err != nil {
          writeEvent(w, &SSEvent{Event: "error", Data: err.Error()})
          return false
        }
        source = ps.Channel()
        writeEvent(w, &SSEvent{Event: "key", Data: key})
        return true
      }
      select {
        case <-clientGone:
          svc.removeClient(key)
          return false
        case msg := <-source:
          if msg == nil {
            svc.removeClient(key)
            return false
          }
          writeEvent(w, &SSEvent{Event: msg.Channel, Data: msg.Payload})
          return true
      }
    })
  })
  router.POST("/Events/:key", func (c *gin.Context) {
    api := newApi(c)
    var err error
    var req struct {
      Subscribe []string `json:"subscribe"`
      Unsubscribe []string `json:"unsubscribe"`
    }
    err = api.Request(&req)
    if err != nil { api.Error(err); return }
    var ps *redis.PubSub
    var serverUrl string
    key := c.Param("key")
    ps, serverUrl = svc.getClient(key)
    if ps == nil {
      if len(serverUrl) == 0 {
        api.StringError("no such client")
        return
      }
      if serverUrl == svc.selfUrl {
        svc.client.Del(key)
        api.StringError("connection lost")
        return
      }
      /* TODO: proxy request to serverUrl */
      hi1.Printf("forward request to %s\n", serverUrl)
      api.StringError("event request forwarding is not implemented")
      return
    }
    if len(req.Unsubscribe) > 0 {
      err = ps.Unsubscribe(req.Unsubscribe...)
      if err != nil { api.Error(err); return }
    }
    if len(req.Subscribe) > 0 {
      err = ps.Subscribe(req.Subscribe...)
      if err != nil { api.Error(err); return }
    }
    api.Result(j.Boolean(true))
  })
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

func writeEvent(w io.Writer, m *SSEvent) {
  var buf bytes.Buffer
  if len(m.Id) > 0 {
    buf.WriteString(fmt.Sprintf("id: %s\n", noLF(m.Id)))
  }
  if len(m.Event) > 0 {
    buf.WriteString(fmt.Sprintf("event: %s\n", noLF(m.Event)))
  }
  if len(m.Data) > 0 {
    lines := strings.Split(m.Data, "\n")
    for _, line := range lines {
      buf.WriteString(fmt.Sprintf("data: %s\n", line))
    }
  }
  buf.WriteString("\n")
  w.Write(buf.Bytes())
}

func noLF(s string) string {
  return strings.Replace(s, "\n", "", -1)
}
