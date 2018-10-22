/*
  Keeps SSE connections alive when proxied through nginx.
  A browser will stop reconnecting if it gets a 502 errors while trying to
  reconnect.
  This simple tool accepts connections on the same route as the backend,
  and closes the connection after 1 second.
  Start with this command:

    LISTEN=:8082 MOUNT_PATH=/tezos/backend ./hold-sse

  Then configure nginx fallback when the backend is unreachable:

  upstream backend {
    server 127.0.0.1::;
    server 127.0.0.1:82 backup;
    server $BACKEND-IP  backup;
  }

  location ^~ /backend/Events {
    chunked_transfer_encoding off;
    proxy_http_version 1.1;
    proxy_set_header Connection '';
    proxy_buffering off;
    proxy_cache off;
    proxy_read_timeout 900;
    proxy_intercept_errors on;
    proxy_next_upstream error;
    proxy_pass http://backend;
  }

*/


package main

import (
  "io"
  "os"
  "time"
  "github.com/gin-gonic/gin"
)

func main() {

  var engine = gin.Default()
  var router gin.IRoutes = engine

  mountPath := os.Getenv("MOUNT_PATH")
  if mountPath != "" {
    router = engine.Group(mountPath)
  }

  router.GET("/Events/:key", func (c *gin.Context) {
    c.Header("Content-Type", "text/event-stream")
    c.Header("Cache-Control", "no-cache")
    c.Header("Connection", "keep-alive")
    c.Header("X-Accel-Buffering", "no")
    first := true
    c.Stream(func (w io.Writer) bool {
      if first {
        first = false
        w.Write([]byte("\nretry: 2000\n"))
        return true
      }
      time.Sleep(1000)
      return false
    })
  })

  engine.Run(os.Getenv("LISTEN"))

}
