
package auth

import (
  "github.com/gin-gonic/gin"
  "github.com/gin-contrib/sessions"
  "tezos-contests.izibi.com/backend/view"
)

func GetUserId(c *gin.Context) (userId int64, ok bool) {

  // XXX Disable in production!
  xUserId := c.GetHeader("X-User-Id")
  if xUserId != "" { return view.ImportId(xUserId), true }

  session := sessions.Default(c)
  val := session.Get("userId")
  if val == nil {
    return 0, false
  }
  return view.ImportId(val.(string)), true
}
