
package auth

import (
  "github.com/gin-gonic/gin"
  "github.com/gin-contrib/sessions"
)

func GetUserId(c *gin.Context) (id string, ok bool) {
  session := sessions.Default(c)
  val := session.Get("userId")
  if val == nil {
    return "", false
  }
  return val.(string), true
}

