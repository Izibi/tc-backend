
package events

import (
  "io"
  "github.com/gin-gonic/gin"
  "github.com/go-redis/redis"
  j "tezos-contests.izibi.com/backend/jase"
  "tezos-contests.izibi.com/backend/utils"
)

type Context struct {
  resp *utils.Response
}

func (svc *Service) Wrap(c *gin.Context) *Context {
  return &Context{
    utils.NewResponse(c),
  }
}

func (svc *Service) Route(router gin.IRoutes) {

  router.GET("/Events", func (c *gin.Context) {
    c.Header("Content-Type", "text/event-stream")
    c.Header("Cache-Control", "no-cache")
    c.Header("Connection", "keep-alive")
    c.Header("X-Accel-Buffering", "no")
    /* TODO: send an id and use c.GetHeader("Last-Event-ID") resent by browser
       to send any missed events */
    var key string
    var source <-chan *redis.Message
    clientGone := c.Writer.CloseNotify()
    c.Stream(func (w io.Writer) bool {
      var err error
      var sink = EventSink{w, nil}
      if source == nil {
        var ps *redis.PubSub
        key, ps, err = svc.newClient(svc.config.SelfUrl)
        if err != nil {
          sink.Write(&SSEvent{Event: "error", Data: err.Error()})
          return false
        }
        source = ps.Channel()
        sink.Write(&SSEvent{Event: "message", Data: encodeMessage("key", key)})
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
          sink.Write(&SSEvent{Event: "message", Data: encodeMessage(msg.Channel, msg.Payload)})
          return true
      }
    })
  })

  router.POST("/Events/:key", func (c *gin.Context) {
    ctx := svc.Wrap(c)
    var err error
    /* The request does not need to be signed or authenticated because
       - the event source key is secret
       - the channel keys are secret
    */
    var req struct {
      Subscribe []string `json:"subscribe"`
      Unsubscribe []string `json:"unsubscribe"`
    }
    err = c.BindJSON(&req)
    if err != nil { ctx.resp.Error(err); return }
    var ps *redis.PubSub
    var serverUrl string
    key := c.Param("key")
    ps, serverUrl = svc.getClient(key)
    if ps == nil {
      if len(serverUrl) == 0 {
        ctx.resp.StringError("no such client")
        return
      }
      if serverUrl == svc.config.SelfUrl {
        svc.client.Del(key)
        ctx.resp.StringError("connection lost")
        return
      }
      /* TODO: proxy request to serverUrl */
      hi1.Printf("forward request to %s\n", serverUrl)
      ctx.resp.StringError("event request forwarding is not implemented")
      return
    }
    /* TODO: either filter subscription to contest, team channel,
       or generate unguessable contest/team channel keys. */
    if len(req.Unsubscribe) > 0 {
      err = ps.Unsubscribe(req.Unsubscribe...)
      if err != nil { ctx.resp.Error(err); return }
    }
    if len(req.Subscribe) > 0 {
      err = ps.Subscribe(req.Subscribe...)
      if err != nil { ctx.resp.Error(err); return }
    }
    ctx.resp.Result(j.Boolean(true))
  })

}
