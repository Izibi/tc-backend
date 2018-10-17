
// TODO: move session stuff into session.go

package auth

import (

  "io/ioutil"
  "net/http"
  "database/sql"

  "github.com/gin-gonic/gin"
  "github.com/gin-contrib/sessions"
  "golang.org/x/oauth2"
  j "tezos-contests.izibi.com/backend/jase"

  "tezos-contests.izibi.com/backend/config"
  "tezos-contests.izibi.com/backend/utils"
  "tezos-contests.izibi.com/backend/model"

)

type Service struct {
  config *config.Config
  oauth *oauth2.Config
  db *sql.DB
}

type Context struct {
  c *gin.Context
  resp *utils.Response
  model *model.Model
}

func NewService(config *config.Config, db *sql.DB) *Service {
  oauthConf := &oauth2.Config{
      ClientID: config.Auth.ClientID,
      ClientSecret: config.Auth.ClientSecret,
      RedirectURL: config.Auth.RedirectURL,
      Endpoint: oauth2.Endpoint{
        AuthURL: config.Auth.AuthURL,
        TokenURL: config.Auth.TokenURL,
      },
      Scopes: []string{"account"},
  }
  return &Service{config, oauthConf, db}
}

func (svc *Service) Wrap(c *gin.Context) *Context {
  return &Context{
    c,
    utils.NewResponse(c),
    model.New(c, svc.db),
  }
}

func (svc *Service) Route(r gin.IRoutes) {

  r.GET("/User", func (c *gin.Context) {
    ctx := svc.Wrap(c)
    session := sessions.Default(c)
    val := session.Get("userId")
    if val != nil {
      userId := val.(string)
      err := ctx.model.ViewUser(userId)
      if err != nil { ctx.resp.Error(err); return }
    }
    ctx.resp.Send(ctx.model.Flat())
  })

  r.GET("/Login", func (c *gin.Context) {
    /* Open this route in a new window to redirect the user to the identity
       provider (IdP) for authentication.  The IdP will eventually redirect
       the user to the /LoginComplete route.
       Do not open this route in an iframe, as it may prevent the IdP from
       getting/ting the user's cookies (see Block Third-party cookies). */
    state, err := utils.NewState()
    if err != nil { c.AbortWithError(500, err); return }
    session := sessions.Default(c)
    session.Set("state", state)
    session.Save()
    c.Redirect(http.StatusSeeOther, svc.oauth.AuthCodeURL(state))
  })

  r.GET("/Login/:userId", func (c *gin.Context) {
    /* TEMPORARY, BYPASS OAUTH */
    userId := c.Param("userId")
    session := sessions.Default(c)
    session.Set("userId", userId)
    session.Save()
    c.Redirect(http.StatusSeeOther, "/LoginComplete")
  })

  r.GET("/LoginComplete", func (c *gin.Context) {
    ctx := svc.Wrap(c)

    errStr := c.Query("error")
    if errStr != "" {
      c.HTML(http.StatusOK, "loginError", loginErrorData{Error: errStr, Message: ""})
    }

    session := sessions.Default(c)
    state := session.Get("state")
    if state == nil || state.(string) != c.Query("state") {
      c.String(400, "bad state")
      return
    }

    // verboseC := context.WithValue(c, oauth2.HTTPClient, utils.VerboseHttpClient())
    token, err := svc.oauth.Exchange(c, c.Query("code"))
    if err != nil { c.AbortWithError(500, err); return }

    client := svc.oauth.Client(c, token)
    resp, err := client.Get(svc.config.Auth.ProfileURL)
    if err != nil { c.AbortWithError(500, err); return }
    defer resp.Body.Close()

    body, err := ioutil.ReadAll(resp.Body)
    if err != nil { c.AbortWithError(500, err); return }

    profile := LoadUserProfile(body)
    userId, err := ctx.model.ImportUserProfile(profile)
    if err != nil { c.AbortWithError(500, err); return }

    session.Set("userId", userId)
    session.Save()

    message := j.Object()
    message.Prop("type", j.String("login"))
    message.Prop("userId", j.String(userId))
    messageStr, err := j.ToString(message)
    if err != nil { c.AbortWithError(500, err) }
    data := loginCompleteData{
      Message: messageStr,
      Target: svc.config.FrontendOrigin,
    }
    c.HTML(http.StatusOK, "loginComplete", data)
  })

  r.GET("/Logout", func (c *gin.Context) {
    /* Open this route in a new window to clear the user's session, and
       redirect to the IdP's logout page.
       Do not open this route in an iframe, as it may prevent the IdP from
       getting/setting the user's cookies (see Block Third-party cookies). */

    session := sessions.Default(c)
    session.Clear()
    session.Save()

    message := j.Object()
    message.Prop("type", j.String("logout"))
    messageStr, err := j.ToString(message)
    if err != nil { c.AbortWithError(500, err) }
    data := logoutCompleteData{
      Message: messageStr,
      Target: svc.config.FrontendOrigin,
      LogoutUrl: svc.config.Auth.LogoutURL,
    }

    c.HTML(http.StatusOK, "logoutComplete", data)
  })

}
