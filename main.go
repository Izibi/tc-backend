/*

  NOTES
  =====

  Build/run with -tags=jsoniter

*/

package main

import (

  "fmt"
  "html/template"
  //"io"
  "io/ioutil"
  "log"
  "net/http"
  "time"

  "github.com/gin-gonic/gin"
  "github.com/gin-contrib/sessions"
  "github.com/gin-contrib/sessions/cookie"
  "github.com/gin-contrib/cors"
  "github.com/utrack/gin-csrf"
  "github.com/json-iterator/go"  // https://godoc.org/github.com/json-iterator/go
  "golang.org/x/oauth2"
  //"golang.org/x/net/context"

  j "tezos-contests.izibi.com/backend/jase"
  "tezos-contests.izibi.com/backend/utils"
  "tezos-contests.izibi.com/backend/model"
  "tezos-contests.izibi.com/backend/auth"

)

func buildRootTemplate() *template.Template {
  t := template.New("")
  template.Must(t.New("loginComplete").Parse(`<!DOCTYPE html>
<head lang="en"><meta charset="utf-8"><title>login complete</title></head>
<body><script type="text/javascript">
  window.opener.postMessage({{.Message}}, {{.Target}});
  window.close();
</script></body>`))
  template.Must(t.New("logoutComplete").Parse(`<!DOCTYPE html>
<head lang="en"><meta charset="utf-8"><title>logout complete</title></head>
<body><script type="text/javascript">
  window.opener.postMessage({{.Message}}, {{.Target}});
  window.location.href = {{.LogoutUrl}};
</script></body>`))
  template.Must(t.New("noSession").Parse(`<!DOCTYPE html>
<head lang="en"><meta charset="utf-8"><title>no session</title></head>
<body><p>No session found, please try again.</p></body>`))
  template.Must(t.New("notImplemented").Parse(`<!DOCTYPE html>
<head lang="en"><meta charset="utf-8"><title>Not Implemented</title></head>
<body><p>Not Implemented</p></body>`))
  template.Must(t.New("loginError").Parse(`<!DOCTYPE html>
<head lang="en"><meta charset="utf-8"><title>Authentication Failed</title></head>
<body><h1>{{.Error}}</h1><p>{{.Message}}</p></body>`))
  return t
}
type loginErrorData struct {
  Error string
  Message string
}
type loginCompleteData struct {
  Message string
  Target string
}
type logoutCompleteData struct {
  Message string
  Target string
  LogoutUrl string
}


func setupRouter(config jsoniter.Any) *gin.Engine {

  db, err := model.Connect(config.Get("db").ToString())
  if err != nil {
    log.Panicln("Failed to connect to model: %s", err)
  }

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

  // Disable Console Color
  // gin.DisableConsoleColor()
  r := gin.Default()
  r.SetHTMLTemplate(buildRootTemplate())

  store := cookie.NewStore([]byte(config.Get("secret").ToString()))
  store.Options(sessions.Options{
    Path:     "/" /* TODO: mount path */,
    Domain:   "",
    MaxAge:   24 * 60 * 60,
    Secure:   false,
    HttpOnly: false,
  })
  r.Use(sessions.Sessions("session_name", store))
  r.Use(csrf.Middleware(csrf.Options{
        Secret: config.Get("csrf_secret").ToString(),
        ErrorFunc: func(c *gin.Context) {
          c.String(400, "CSRF token mismatch")
          c.Abort()
        },
    }))
  r.Use(cors.New(cors.Config{
    AllowOrigins:     []string{"http://localhost:8000", "http://localhost:8080"},
    AllowMethods:     []string{"GET", "POST"},
    AllowHeaders:     []string{"Origin"},
    ExposeHeaders:    []string{"Content-Length"},
    AllowCredentials: true,
    MaxAge: 12 * time.Hour,
  }))

  // Ping test
  r.GET("/ping", func(c *gin.Context) {
    c.String(http.StatusOK, "pong")
  })

  var backend gin.IRoutes
  mountPath := config.Get("mount_path").ToString()
  if mountPath == "" {
    backend = r
  } else {
    backend = r.Group(mountPath)
  }

  backend.GET("/User", func (c *gin.Context) {
    session := sessions.Default(c)
    val := session.Get("userId")
    if val == nil {
      c.JSON(http.StatusOK, nil)
    } else {
      c.JSON(http.StatusOK, gin.H{"userId": val.(string)})
    }
  })

  backend.GET("/Login", func (c *gin.Context) {
    /* Open this route in a new window to redirect the user to the identity
       provider (IdP) for authentication.  The IdP will eventually redirect
       the user to the /LoginComplete route.
       Do not open this route in an iframe, as it may prevent the IdP from
       getting/setting the user's cookies (see Block Third-party cookies). */
    state, err := utils.NewState()
    if err != nil { c.AbortWithError(500, err); return }
    session := sessions.Default(c)
    session.Set("state", state)
    session.Save()
    c.Redirect(http.StatusSeeOther, oauthConf.AuthCodeURL(state))
  })

  backend.GET("/Login/:userId", func (c *gin.Context) {
    /* TEMPORARY, BYPASS OAUTH */
    userId := c.Param("userId")
    session := sessions.Default(c)
    session.Set("userId", userId)
    session.Save()
    c.Redirect(http.StatusSeeOther, "/LoginComplete")
  })

  backend.GET("/LoginComplete", func (c *gin.Context) {

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

    profile := auth.LoadUserProfile(body)
    userId, err := db.ImportUserProfile(profile, time.Now())
    if err != nil { c.AbortWithError(500, err); return }

    session.Set("userId", userId)
    session.Save()

    message := j.Object()
    message.Prop("type", j.String("login"))
    message.Prop("userId", j.String(userId))
    message.Prop("csrfToken", j.String(csrf.GetToken(c)))
    messageStr, err := j.ToString(message)
    if err != nil { c.AbortWithError(500, err) }
    data := loginCompleteData{
      Message: messageStr,
      Target: "https://home.epixode.fr",
    }
    c.HTML(http.StatusOK, "loginComplete", data)
  })

  backend.GET("/Logout", func (c *gin.Context) {
    /* Open this route in a new window to clear the user's session, and
       redirect to the IdP's logout page.
       Do not open this route in an iframe, as it may prevent the IdP from
       getting/setting the user's cookies (see Block Third-party cookies). */

    session := sessions.Default(c)
    session.Clear()

    message := j.Object()
    message.Prop("type", j.String("logout"))
    message.Prop("csrfToken", j.String(csrf.GetToken(c)))
    messageStr, err := j.ToString(message)
    if err != nil { c.AbortWithError(500, err) }
    data := logoutCompleteData{
      Message: messageStr,
      Target: "https://home.epixode.fr",
      LogoutUrl: config.Get("logout_url").ToString(),
    }

    c.HTML(http.StatusOK, "logoutComplete", data)
  })

  backend.GET("/AuthenticatedUserLanding", func(c *gin.Context) {
    resp := utils.NewResponse(c)
    id, ok := getUserId(c)
    if !ok { resp.StringError("you don't exist"); return }
    userId, err := db.ViewUser(resp, id)
    if err != nil { resp.Error(err); return }
    resp.Set("userId", userId)
    contestIds, err := db.ViewUserContests(resp, id)
    if err != nil { resp.Error(err); return }
    resp.Set("contestIds", contestIds)
    resp.Send()
  })

/*
  // Authorized group (uses gin.BasicAuth() middleware)
  // Same as:
  // authorized := r.Group("/")
  // authorized.Use(gin.BasicAuth(gin.Credentials{
  //    "foo":  "bar",
  //    "manu": "123",
  //}))
  authorized := r.Group("/", gin.BasicAuth(gin.Accounts{
    "foo":  "bar", // user:foo password:bar
    "manu": "123", // user:manu password:123
  }))
  authorized.POST("admin", func(c *gin.Context) {
    user := c.MustGet(gin.AuthUserKey).(string)
    var json struct {
      Value string `json:"value" binding:"required"`
    }
    if c.Bind(&json) == nil {
      c.JSON(http.StatusOK, gin.H{"status": "ok"})
    }
  })
*/

  return r
}

func getUserId(c *gin.Context) (id string, ok bool) {
  session := sessions.Default(c)
  val := session.Get("userId")
  fmt.Printf("ViewUser(%s)", val)
  if val == nil {
    return "", false
  }
  return val.(string), true
}

func main() {
  config := jsoniter.Get([]byte(`{
    "port": 8080,
    "db": "tezos:kerl1Olhog_@tcp(tezos)/tezos_platform",
    "secret": "0123456789",
    "csrf_secret": "AAAAAAAA",
    "oauth_client_id": "32",
    "oauth_secret": "jDbaEtPfCaKwnss0jLuCOl1PAWPDzagEKGLLKzHY",
    "oauth_callback_url": "https://home.epixode.fr/tezos/backend/LoginComplete",
    "oauth_auth_url": "https://login.france-ioi.org/oauth/authorize",
    "oauth_token_url": "https://login.france-ioi.org/oauth/token",
    "profile_url": "https://login.france-ioi.org/user_api/account",
    "logout_url": "https://login.france-ioi.org/logout",
    "mount_path": "/tezos/backend"
  }`))

  /*
  f, _ := os.Create("gin.log")
  gin.DefaultWriter = io.MultiWriter(f)
  gin.SetMode(gin.ReleaseMode)
  */

  r := setupRouter(config)
  r.Run(fmt.Sprintf(":%d", config.Get("port").ToUint32()))
}
