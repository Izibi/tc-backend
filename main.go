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
  "log"
  "net/http"

  "github.com/gin-gonic/gin"
  "github.com/gin-contrib/sessions"
  "github.com/gin-contrib/sessions/cookie"
  //"github.com/gin-contrib/cors"
  "github.com/utrack/gin-csrf"
  "github.com/json-iterator/go"  // https://godoc.org/github.com/json-iterator/go
  //"golang.org/x/net/context"

  //j "tezos-contests.izibi.com/backend/jase"
  "tezos-contests.izibi.com/backend/utils"
  "tezos-contests.izibi.com/backend/model"
  "tezos-contests.izibi.com/backend/auth"

)

func buildRootTemplate() *template.Template {
  t := template.New("")
  auth.SetupTemplates(t)
  template.Must(t.New("notImplemented").Parse(`<!DOCTYPE html>
<head lang="en"><meta charset="utf-8"><title>Not Implemented</title></head>
<body><p>Not Implemented</p></body>`))
  return t
}

func setupRouter(config jsoniter.Any) *gin.Engine {

  db, err := model.Connect(config.Get("db").ToString())
  if err != nil {
    log.Panicln("Failed to connect to model: %s", err)
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
  /*
  r.Use(cors.New(cors.Config{
    AllowOrigins:     []string{"http://localhost:8000", "http://localhost:8080"},
    AllowMethods:     []string{"GET", "POST"},
    AllowHeaders:     []string{"Origin"},
    ExposeHeaders:    []string{"Content-Length"},
    AllowCredentials: true,
    MaxAge: 12 * time.Hour,
  }))
  */

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

  auth.SetupRoutes(backend, config, db)

  backend.GET("/AuthenticatedUserLanding", func(c *gin.Context) {
    resp := utils.NewResponse(c)
    id, ok := auth.GetUserId(c)
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
