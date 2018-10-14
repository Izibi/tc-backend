
package main

import (

  "database/sql"
  "fmt"
  "html/template"
  "io/ioutil"
  "log"
  "net/http"
  "time"

  "github.com/fatih/color"
  "github.com/gin-gonic/gin"
  "github.com/gin-contrib/sessions"
  "github.com/gin-contrib/sessions/cookie"
  "github.com/gin-contrib/cors"
  "github.com/go-redis/redis"
  "github.com/utrack/gin-csrf"
  _ "github.com/go-sql-driver/mysql"
  "github.com/Masterminds/semver"
  "gopkg.in/yaml.v2"

  "tezos-contests.izibi.com/backend/utils"
  "tezos-contests.izibi.com/backend/model"
  "tezos-contests.izibi.com/backend/auth"
  "tezos-contests.izibi.com/backend/teams"
  "tezos-contests.izibi.com/backend/contests"
  "tezos-contests.izibi.com/backend/games"
  "tezos-contests.izibi.com/backend/blocks"
  "tezos-contests.izibi.com/backend/events"

)

type Config struct {
  Listen string `yaml:"listen"`
  MountPath string `yaml:"mount_path"`
  SelfUrl string `yaml:"self_url"`
  SessionSecret string `yaml:"session_secret"`
  CsrfSecret string `yaml:"csrf_secret"`
  DataSource string `yaml:"datasource"`
  FrontendOrigin string `yaml:"frontend_origin"`
  ApiVersion string `yaml:"api_version"`
  ApiKey string `yaml:"api_key"`
  Auth auth.Config `yaml:"auth"`
  Game games.Config `yaml:"game"`
  Blocks blocks.Config `yaml:"blocks"`
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

  apiVersion := semver.MustParse(config.ApiVersion)

  db, err = sql.Open("mysql", config.DataSource)
  if err != nil {
    log.Panicf("Failed to connect to database: %s\n", err)
  }

  rc := redis.NewClient(&redis.Options{
    Addr:     "localhost:6379",
    Password: "", // no password set
    DB:       0,  // use default DB
  })
  err = rc.Ping().Err()
  if err != nil {
    log.Panicf("Failed to connect to redis: %s\n", err)
  }

  eventService, err := events.NewService(rc, config.SelfUrl)
  if err != nil {
    log.Panicf("Failed to connect to create event service: %s\n", err)
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
  r.Use(csrf.Middleware(csrf.Options{
        Secret: config.CsrfSecret,
        ErrorFunc: func(c *gin.Context) {
          /* Requests with no cookie can safely omit the CSRF token. */
          if c.GetHeader("Cookie") != "" {
            c.JSON(http.StatusOK, gin.H{"error": "CSRF token mismatch"})
            c.Abort()
          }
        },
    }))
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

  var router gin.IRoutes
  mountPath := config.MountPath
  if mountPath == "" {
    router = r
  } else {
    router = r.Group(mountPath)
  }

  newApi := func (c *gin.Context) *utils.Response {
    return utils.NewResponse(c, config.ApiKey)
  }

  blockStore := blocks.NewStore(config.Blocks, rc)

  auth.SetupRoutes(router, newApi, config.Auth, db)
  teams.SetupRoutes(router, newApi, db)
  contests.SetupRoutes(router, newApi, db)
  blocks.SetupRoutes(router, newApi, blockStore)
  games.SetupRoutes(router, newApi, config.Game, blockStore, db, eventService)
  eventService.SetupRoutes(router, newApi)

  router.GET("/ping", func(c *gin.Context) {
    c.String(http.StatusOK, "pong")
  })

  router.GET("/Time", func (c *gin.Context) {
    var err error
    reqVersion := c.GetHeader("X-Api-Version")
    req, err := semver.NewConstraint(reqVersion)
    if err != nil {
      c.String(400, "Client sent a bad semver constraint")
      return;
    }
    if !req.Check(apiVersion) {
      c.String(400, "Client is incompatible with Server API %s", apiVersion)
      return
    }
    type Response struct {
      Result string `json:"result"`
    }
    res := Response{Result: time.Now().Format(time.RFC3339Nano)}
    c.JSON(200, &res)
  })

  router.GET("/CsrfToken.js", func(c *gin.Context) {
    /* Token is base64-encoded and thus safe to inject with %s. */
    token := csrf.GetToken(c)
    c.Data(200, "application/javascript",
      []byte(fmt.Sprintf(";window.csrfToken = \"%s\";", token)))
  });

  router.GET("/AuthenticatedUserLanding", func(c *gin.Context) {
    var err error
    api := newApi(c)
    id, ok := auth.GetUserId(c)
    if !ok { api.BadUser(); return }
    m := model.New(c, db)
    err = m.ViewUser(id)
    if err != nil { api.Error(err); return }
    err = m.ViewUserContests(id)
    if err != nil { api.Error(err); return }
    api.Send(m.Flat())
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

  color.NoColor = false

  var err error
  var configFile []byte
  configFile, err = ioutil.ReadFile("config.yaml")
  if err != nil { panic(err) }
  var config Config
  err = yaml.Unmarshal(configFile, &config)
  if err != nil { panic(err) }
  if config.Blocks.Path == "" {
    // TODO
  }
  if config.Auth.FrontendOrigin == "" {
    config.Auth.FrontendOrigin = config.FrontendOrigin
  }
  if config.Game.ApiKey == "" {
    config.Game.ApiKey = config.ApiKey
  }

  /*
  f, _ := os.Create("gin.log")
  gin.DefaultWriter = io.MultiWriter(f)
  gin.SetMode(gin.ReleaseMode)
  */

  r := setupRouter(config)
  r.Run(config.Listen)
}
