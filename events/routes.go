
package events

import (
  "io"
  //"fmt"
  "time"
  "github.com/gin-gonic/gin"
  j "tezos-contests.izibi.com/backend/jase"
  "tezos-contests.izibi.com/backend/auth"
  "tezos-contests.izibi.com/backend/utils"
)

type Context struct {
  resp *utils.Response
  req *utils.Request
}

func (svc *Service) Wrap(c *gin.Context) *Context {
  return &Context{
    utils.NewResponse(c),
    utils.NewRequest(c, svc.config.ApiKey),
  }
}

func (svc *Service) Route(router gin.IRoutes) {

/*
  go func() {
    i := 0
    for {
      time.Sleep(100 * time.Millisecond)
      _ = svc.redis.Publish("system", fmt.Sprintf("%d", i)).Err()
      i += 1
    }
  }()
*/

  /*
    Clients use this route to create an event stream and obtain its key.
  */
  router.POST("/Events", func (c *gin.Context) {
    var err error
    ctx := svc.Wrap(c)
    var st *stream
    st, err = svc.newStream()
    if err != nil { ctx.resp.Error(err); return }
    var req struct {
      Author string `json:"author"` /* team public key if signed, absent otherwise */
      /* TODO: add a timestamp to avoid replay attacks */
    }
    err = ctx.req.Signed(&req)
    if err == nil {
      var teamId int64
      teamId, err = svc.model.FindTeamIdByKey(req.Author[1:])
      if err != nil { ctx.resp.Error(err); return }
      if teamId == 0 { ctx.resp.StringError("team not found"); return }
      st.SetTeamId(teamId)
    } else {
      userId, ok := auth.GetUserId(c)
      if !ok { ctx.resp.StringError("authentication required"); return }
      st.SetUserId(userId)
    }
    ctx.resp.Result(j.String(st.key))
  })

  /*
    A client must connect to this event stream route within 60 seconds of
    obtaining a stream key to receive events.
    If the client disconnects, it can reconnect with 60 seconds to continue
    receiving events.
    If the "Last-Event-ID" header contains the Id of the last event received
    by the client, no events will be missed.
  */
  router.GET("/Events/:key", func (c *gin.Context) {
    st, ok, err := svc.connectStream(c.Param("key"), c.GetHeader("Last-Event-ID"))
    if err != nil { c.AbortWithError(500, err); return }
    if st == nil { c.String(404, "Not Found"); return }
    if !ok { c.String(400, "Bad Request"); return }
    c.Header("Content-Type", "text/event-stream")
    c.Header("Cache-Control", "no-cache")
    c.Header("Connection", "keep-alive")
    c.Header("X-Accel-Buffering", "no")
    source := st.pubSub.Channel()
    clientGone := c.Writer.CloseNotify()
    refreshTicker := time.NewTicker(RedisStreamKeyRefresh)
    encoder := NewEncoder()
    var initDone bool
    cleanup := func () {
      svc.disconnectStream(st)
      refreshTicker.Stop()
    }
    c.Stream(func (w io.Writer) bool {
      var err error
      if !initDone {
        /* Send an initial chunk to cause the status and header to be sent
           immediately to the client. */
        w.Write([]byte("\nretry: 500\nid: 1\n"))
        initDone = true
        return true
      }
      var event *SSEvent
      event = st.Resend()
      if event != nil {
        _, err = w.Write(encoder.Encode(event))
        if err != nil { cleanup(); return false }
        return true
      }
      select {
        case <-clientGone:
          cleanup()
          return false
        case <-refreshTicker.C:
          svc.refreshStream(st)
          return true
        case msg := <-source:
          if msg == nil {
            /* XXX Disconnected from redis, what to do? */
            cleanup()
            return false
          }
          event = st.Push(encodeMessage(msg.Channel, msg.Payload))
          _, err = w.Write(encoder.Encode(event))
          if err != nil { cleanup(); return false }
          return true
        case <-st.closeChan:
          cleanup()
          return false
      }
    })
  })

  router.POST("/Events/:key/Close", func (c *gin.Context) {
    ctx := svc.Wrap(c)
    st, err := svc.getStream(c.Param("key"))
    if err != nil { ctx.resp.Error(err); return }
    if st == nil { ctx.resp.StringError("not found"); return }
    st.closeChan <- true
    time.Sleep(100 * time.Millisecond)
    err = svc.redis.Publish("system", "HELLO!").Err()
    ctx.resp.Result(j.Boolean(true))
  })

  /*
    This route enables a client to manage the channel subscriptions of an event
    stream.
    XXX As is, this API allows spying on any contest or team, by guessing their
    id.  Either add permission checking on the channels, or use secret keys
    for contest and team channels.
  */
  router.POST("/Events/:key", func (c *gin.Context) {
    ctx := svc.Wrap(c)
    var err error
    var req struct {
      Subscribe []string `json:"subscribe"`
      Unsubscribe []string `json:"unsubscribe"`
    }
    err = c.BindJSON(&req)
    if err != nil { ctx.resp.Error(err); return }
    key := c.Param("key")
    var st *stream
    st, err = svc.getStream(key)
    if err != nil { ctx.resp.Error(err); return }
    if st.pubSub == nil {
      if len(st.serverUrl) == 0 {
        ctx.resp.StringError("no such stream")
        return
      }
      if st.serverUrl == svc.config.SelfUrl {
        /* TODO: delegate to stream cleanup function? */
        svc.redis.Del(streamKey(key))
        ctx.resp.StringError("connection lost")
        return
      }
      /* TODO: proxy request to serverUrl */
      hi1.Printf("forward request to %s\n", st.serverUrl)
      ctx.resp.StringError("forwarding is not implemented")
      return
    }
    /* TODO: either filter subscription to contest, team channel,
       or generate unguessable contest/team channel keys. */
    if len(req.Unsubscribe) > 0 {
      err = st.Unsubscribe(req.Unsubscribe...)
      if err != nil { ctx.resp.Error(err); return }
    }
    if len(req.Subscribe) > 0 {
      err = st.Subscribe(req.Subscribe...)
      if err != nil { ctx.resp.Error(err); return }
    }
    ctx.resp.Result(j.Boolean(true))
    /*
      // fmt.Printf("request is signed by %s\n", req.Author)
      team, err := svc.model.LoadTeam(teamId, model.NullFacet)
      if err != nil { ctx.resp.Error(err); return }
      contestId = team.Contest_id
      contestId = view.ImportId(req.ContestId)
      team, err := view.LoadUserContestTeam(userId, contestId)
      if err != nil { ctx.resp.Error(err); return }
      teamId = team.Id
      fmt.Printf("teamId %d, contestId %d\n", teamId, contestId)
      TODO: subscribe to team id, contest id
    */
  })

}
