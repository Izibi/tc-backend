
package main

import (

  "database/sql"
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
  _ "github.com/go-sql-driver/mysql"
  //"golang.org/x/net/context"
  "gopkg.in/yaml.v2"

  //j "tezos-contests.izibi.com/backend/jase"
  "tezos-contests.izibi.com/backend/utils"
  "tezos-contests.izibi.com/backend/model"
  "tezos-contests.izibi.com/backend/auth"
  "tezos-contests.izibi.com/backend/game"

)

type Config struct {
  Listen string `yaml:"listen"`
  MountPath string `yaml:"mount_path"`
  SessionSecret string `yaml:"session_secret"`
  CsrfSecret string `yaml:"csrf_secret"`
  DataSource string `yaml:"data_source"`
  FrontendOrigin string `yaml:"frontend_origin"`
  Auth auth.Config `yaml:"auth"`
  Game game.Config `yaml:"game"`
}

func buildRootTemplate() *template.Template {
  t := template.New("")
  auth.SetupTemplates(t)
  template.Must(t.New("notImplemented").Parse(`<!DOCTYPE html>
<head lang="en"><meta charset="utf-8"><title>Not Implemented</title></head>
<body><p>Not Implemented</p></body>`))
  return t
}

func setupRouter(config Config) *gin.Engine {
  var err error
  var db *sql.DB

  db, err = sql.Open("mysql", config.DataSource)
  if err != nil {
    log.Panicf("Failed to connect to database: %s\n", err)
  }

  // Disable Console Color
  // gin.DisableConsoleColor()
  r := gin.Default()
  r.SetHTMLTemplate(buildRootTemplate())

  store := cookie.NewStore([]byte(config.SessionSecret))
  store.Options(sessions.Options{
    Path:     "/" /* TODO: mount path */,
    Domain:   "",
    MaxAge:   24 * 60 * 60,
    Secure:   false,
    HttpOnly: false,
  })
  r.Use(sessions.Sessions("session_name", store))
  /* TODO: re-enable
  r.Use(csrf.Middleware(csrf.Options{
        Secret: config.CsrfSecret,
        ErrorFunc: func(c *gin.Context) {
          c.JSON(http.StatusOK, gin.H{"error": "CSRF token mismatch"})
          c.Abort()
        },
    }))
  */
  r.Use(cors.New(cors.Config{
    AllowOrigins:     []string{
      config.FrontendOrigin,
    },
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
  mountPath := config.MountPath
  if mountPath == "" {
    backend = r
  } else {
    backend = r.Group(mountPath)
  }

  auth.SetupRoutes(backend, config.Auth, db)
  game.SetupRoutes(backend, config.Game)

  backend.GET("/CsrfToken.js", func(c *gin.Context) {
    /* Token is base64-encoded and thus safe to inject with %s. */
    token := csrf.GetToken(c)
    c.Data(200, "application/javascript",
      []byte(fmt.Sprintf(";window.csrfToken = \"%s\";", token)))
  });

  backend.GET("/AuthenticatedUserLanding", func(c *gin.Context) {
    var err error
    resp := utils.NewResponse(c)
    id, ok := auth.GetUserId(c)
    if !ok { resp.BadUser(); return }
    m := model.New(db)
    err = m.ViewUser(id)
    if err != nil { resp.Error(err); return }
    err = m.ViewUserContests(id)
    if err != nil { resp.Error(err); return }
    resp.Send(m)
  })

  backend.GET("/Contests/:contestId", func(c *gin.Context) {
    resp := utils.NewResponse(c)
    userId, ok := auth.GetUserId(c)
    if !ok { resp.BadUser(); return }
    m := model.New(db)
    contestId := c.Param("contestId")
    err := m.ViewUserContest(userId, contestId)
    if err != nil { resp.Error(err); return }
    resp.Send(m)
  })

  backend.GET("/Contests/:contestId/Team", func(c *gin.Context) {
    resp := utils.NewResponse(c)
    userId, ok := auth.GetUserId(c)
    if !ok { resp.BadUser(); return }
    m := model.New(db)
    contestId := c.Param("contestId")
    err := m.ViewUserContestTeam(userId, contestId)
    if err != nil { resp.Error(err); return }
    resp.Send(m)
  })

  backend.POST("/Contests/:contestId/CreateTeam", func(c *gin.Context) {
    var err error
    resp := utils.NewResponse(c)
    userId, ok := auth.GetUserId(c)
    if !ok { resp.BadUser(); return }
    contestId := c.Param("contestId")
    type Body struct {
      TeamName string `json:"teamName"`
    }
    var body Body
    err = c.ShouldBindJSON(&body)
    if err != nil { resp.Error(err); return }
    m := model.New(db)
    err = m.CreateTeam(userId, contestId, body.TeamName)
    if err != nil { resp.Error(err); return }
    resp.Send(m)
  })

  backend.POST("/Contests/:contestId/JoinTeam", func(c *gin.Context) {
    var err error
    resp := utils.NewResponse(c)
    userId, ok := auth.GetUserId(c)
    if !ok { resp.BadUser(); return }
    contestId := c.Param("contestId")
    type Body struct {
      AccessCode string `json:"accessCode"`
    }
    var body Body
    err = c.ShouldBindJSON(&body)
    if err != nil { resp.Error(err); return }
    m := model.New(db)
    err = m.JoinTeam(userId, contestId, body.AccessCode)
    if err != nil { resp.Error(err); return }
    resp.Send(m)
  })

  backend.POST("/Teams/:teamId/Leave", func(c *gin.Context) {
    var err error
    resp := utils.NewResponse(c)
    userId, ok := auth.GetUserId(c)
    if !ok { resp.BadUser(); return }
    teamId := c.Param("teamId")
    m := model.New(db)
    err = m.LeaveTeam(teamId, userId)
    if err != nil { resp.Error(err); return }
    resp.Send(m)
  })

  backend.POST("/Teams/:teamId/AccessCode", func(c *gin.Context) {
    var err error
    resp := utils.NewResponse(c)
    userId, ok := auth.GetUserId(c)
    if !ok { resp.BadUser(); return }
    teamId := c.Param("teamId")
    m := model.New(db)
    err = m.RenewTeamAccessCode(teamId, userId)
    if err != nil { resp.Error(err); return }
    resp.Send(m)
  })

  backend.POST("/Teams/:teamId/Update", func(c *gin.Context) {
    var err error
    resp := utils.NewResponse(c)
    userId, ok := auth.GetUserId(c)
    if !ok { resp.BadUser(); return }
    teamId := c.Param("teamId")
    var arg model.UpdateTeamArg
    err = c.ShouldBindJSON(&arg)
    if err != nil { resp.Error(err); return }
    m := model.New(db)
    err = m.UpdateTeam(teamId, userId, arg)
    if err != nil { resp.Error(err); return }
    resp.Send(m)
  })

  backend.GET("/Contests/:contestId/Chains", func(c *gin.Context) {
    resp := utils.NewResponse(c)
    userId, ok := auth.GetUserId(c)
    if !ok { resp.BadUser(); return }
    contestId := c.Param("contestId")
    m := model.New(db)
    err := m.ViewChains(userId, contestId, model.ChainFilters{})
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
  configFile, err = ioutil.ReadFile("config.yaml")
  if err != nil { panic(err) }
  var config Config
  err = yaml.Unmarshal(configFile, &config)
  if err != nil { panic(err) }
  if config.Game.BlockStorePath == "" {
    // TODO
  }
  if config.Auth.FrontendOrigin == "" {
    config.Auth.FrontendOrigin = config.FrontendOrigin
  }

  /*
  f, _ := os.Create("gin.log")
  gin.DefaultWriter = io.MultiWriter(f)
  gin.SetMode(gin.ReleaseMode)
  */

  r := setupRouter(config)
  r.Run(config.Listen)
}
