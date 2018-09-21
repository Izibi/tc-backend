
package main

import (

  //"fmt"
  "html/template"
  //"io"
  "io/ioutil"
  "log"
  "net/http"
  "database/sql"

  "github.com/gin-gonic/gin"
  "github.com/gin-contrib/sessions"
  "github.com/gin-contrib/sessions/cookie"
  //"github.com/gin-contrib/cors"
  "github.com/utrack/gin-csrf"
  "github.com/json-iterator/go"  // https://godoc.org/github.com/json-iterator/go
  _ "github.com/go-sql-driver/mysql"
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
  var err error
  var db *sql.DB

  db, err = sql.Open("mysql", config.Get("db").ToString())
  if err != nil {
    log.Panicln("Failed to connect to database: %s", err)
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
    m := model.New(db)
    userId, err := m.ViewUser(id)
    if err != nil { resp.Error(err); return }
    m.Set("userId", userId)
    contestIds, err := m.ViewUserContests(id)
    if err != nil { resp.Error(err); return }
    m.Set("contestIds", contestIds)
    resp.Send(m)
  })

  backend.GET("/Contests/:contestId", func(c *gin.Context) {
    resp := utils.NewResponse(c)
    userId, ok := auth.GetUserId(c)
    if !ok { resp.StringError("you don't exist"); return }
    m := model.New(db)
    contestId := c.Param("contestId")
    err := m.ViewUserContest(userId, contestId)
    if err != nil { resp.Error(err); return }
    resp.Send(m)
  })

  backend.GET("/Contests/:contestId/Team", func(c *gin.Context) {
    resp := utils.NewResponse(c)
    userId, ok := auth.GetUserId(c)
    if !ok { resp.StringError("you don't exist"); return }
    m := model.New(db)
    contestId := c.Param("contestId")
    err := m.ViewUserContestTeam(userId, contestId)
    if err != nil { resp.Error(err); return }
    resp.Send(m)
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

  var err error
  var configFile []byte
  configFile, err = ioutil.ReadFile("config.json")
  if err != nil { panic(err); return }
  config := jsoniter.Get(configFile)

  /*
  f, _ := os.Create("gin.log")
  gin.DefaultWriter = io.MultiWriter(f)
  gin.SetMode(gin.ReleaseMode)
  */

  r := setupRouter(config)
  r.Run(config.Get("listen").ToString())
}
