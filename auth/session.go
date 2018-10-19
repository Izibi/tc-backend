
package auth

import (
  "github.com/gin-contrib/sessions"
)

func (ctx *Context) GetUserId() (id int64, ok bool) {
  session := sessions.Default(ctx.c)
  val := session.Get("userId")
  if val == nil {
    return 0, false
  }
  return ctx.model.ImportId(val.(string)), true
}
