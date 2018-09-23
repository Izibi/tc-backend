
// TODO: move session stuff into session.go

package auth

import (

  "database/sql"
  "io/ioutil"
  "net/http"
  "time"

  "github.com/gin-gonic/gin"
  "github.com/gin-contrib/sessions"
  "github.com/json-iterator/go"  // https://godoc.org/github.com/json-iterator/go
  "golang.org/x/oauth2"

  j "tezos-contests.izibi.com/backend/jase"
  "tezos-contests.izibi.com/backend/utils"
  "tezos-contests.izibi.com/backend/model"

)

func SetupRoutes(r gin.IRoutes, config jsoniter.Any, db *sql.DB) {

  oauthConf := &oauth2.Config{
      ClientID: config.Get("oauth_client_id").ToString(),
      ClientSecret: config.Get("oauth_secret").ToString(),
      RedirectURL: config.Get("oauth_callback_url").ToString(),
      Endpoint: oauth2.Endpoint{
        AuthURL: config.Get("oauth_auth_url").ToString(),
        TokenURL: config.Get("oauth_token_url").ToString(),
      },
      Scopes: []string{"account"},
  }

  r.GET("/User", func (c *gin.Context) {
    resp := utils.NewResponse(c)
    m := model.New(db)
    session := sessions.Default(c)
    val := session.Get("userId")
    if val != nil {
      userId := val.(string)
      err := m.ViewUser(userId)
      if err != nil { resp.Error(err); return }
    }
    resp.Send(m)
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
    c.Redirect(http.StatusSeeOther, oauthConf.AuthCodeURL(state))
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
    token, err := oauthConf.Exchange(c, c.Query("code"))
    if err != nil { c.AbortWithError(500, err); return }

    client := oauthConf.Client(c, token)
    resp, err := client.Get(config.Get("profile_url").ToString())
    if err != nil { c.AbortWithError(500, err); return }
    defer resp.Body.Close()

    body, err := ioutil.ReadAll(resp.Body)
    if err != nil { c.AbortWithError(500, err); return }

    profile := LoadUserProfile(body)
    m := model.New(db)
    userId, err := m.ImportUserProfile(profile, time.Now())
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
      Target: "https://home.epixode.fr", // TODO: from config, frontend_origin
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
      Target: "https://home.epixode.fr", // TODO: from config, frontend_origin
      LogoutUrl: config.Get("logout_url").ToString(),
    }

    c.HTML(http.StatusOK, "logoutComplete", data)
  })

}
