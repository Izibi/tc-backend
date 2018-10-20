
package auth

import (
  "github.com/gin-contrib/sessions"
)

func (ctx *Context) GetUserId() (id int64, ok bool) {
  // XXX Disable in production!
  userId := ctx.c.GetHeader("X-User-Id")
  if userId != "" { return ctx.model.ImportId(userId), true }
  session := sessions.Default(ctx.c)
  val := session.Get("userId")
  if val == nil {
    return 0, false
  }
  return ctx.model.ImportId(val.(string)), true
}
