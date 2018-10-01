
package game

import (
  "io"
  "database/sql"
  "github.com/gin-gonic/gin"
)

type Config struct {}

func SetupRoutes(r gin.IRoutes, config Config, db *sql.DB) {

  r.POST("/Games", func (c *gin.Context) {
    // newGame(config, req.body)
  })

  r.GET("/Games/:gameKey", func (c *gin.Context) {
    // showGame(config, req.params.gameKey)
  })

  r.GET("/Games/:gameKey/Events", func (c *gin.Context) {
    c.Header("Content-Type", "text/event-stream")
    c.Header("Cache-Control", "no-cache")
    c.Header("Connection", "keep-alive")
    c.Header("X-Accel-Buffering", "no")
    c.Stream(func (w io.Writer) bool {
      w.Write([]byte("retry: 30000\n\n"))
      // TODO: verify the game actually exists
      // TODO: this is wrong, teams should register for events (with a signed message)
      // registerGameSink(c.Param("gameKey"), w);
      return true
    })
  })

  r.POST("/Games/:gameKey/Commands", func (c *gin.Context) {
    // inputCommands(config, req.body)
  })

  r.POST("/Games/:gameKey/EndRound", func (c *gin.Context) {
    // endGameRound(config, req.body)
  })

}
