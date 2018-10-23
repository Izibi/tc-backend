
package main

import (

  "database/sql"
  "fmt"
  "html/template"
  //"io"
  //"os"
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

  "tezos-contests.izibi.com/backend/auth"
  "tezos-contests.izibi.com/backend/landing"
  "tezos-contests.izibi.com/backend/contests"
  "tezos-contests.izibi.com/backend/blocks"
  "tezos-contests.izibi.com/backend/events"
  "tezos-contests.izibi.com/backend/teams"
  "tezos-contests.izibi.com/backend/chains"
  "tezos-contests.izibi.com/backend/games"
  "tezos-contests.izibi.com/backend/model"
  cfg "tezos-contests.izibi.com/backend/config"

)

func buildRootTemplate() *template.Template {
  t := template.New("")
  auth.SetupTemplates(t)
  template.Must(t.New("notImplemented").Parse(`<!DOCTYPE html>
<head lang="en"><meta charset="utf-8"><title>Not Implemented</title></head>
<body><p>Not Implemented</p></body>`))
  return t
}

func main() {

  color.NoColor = false

  var err error
  var configFile []byte
  configFile, err = ioutil.ReadFile("config.yaml")
  if err != nil { panic(err) }
  var config cfg.Config
  err = yaml.Unmarshal(configFile, &config)
  if err != nil { panic(err) }
  if config.Blocks.Path == "" {
    // TODO
  }
  if config.Auth.FrontendOrigin == "" {
    config.Auth.FrontendOrigin = config.FrontendOrigin
  }

  var db *sql.DB
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

  apiVersion := semver.MustParse(config.ApiVersion)

  /*
  f, _ := os.Create("gin.log")
  gin.DefaultWriter = io.MultiWriter(f)
  */
  // gin.SetMode(gin.ReleaseMode)

  // Disable Console Color
  // gin.DisableConsoleColor()

  store := cookie.NewStore([]byte(config.SessionSecret))
  store.Options(sessions.Options{
    Path:     "/" /* TODO: mount path */,
    Domain:   "",
    MaxAge:   24 * 60 * 60,
    Secure:   false,
    HttpOnly: false,
  })

  var engine = gin.Default()
  engine.SetHTMLTemplate(buildRootTemplate())
  engine.Use(sessions.Sessions("session_name", store))
  engine.Use(csrf.Middleware(csrf.Options{
        Secret: config.CsrfSecret,
        ErrorFunc: func(c *gin.Context) {
          /* Requests with no cookie can safely omit the CSRF token. */
          if c.GetHeader("Cookie") != "" {
            c.JSON(http.StatusOK, gin.H{"error": "CSRF token mismatch"})
            c.Abort()
          }
        },
    }))
  engine.Use(cors.New(cors.Config{
    //AllowAllOrigins: true,
    AllowOrigins:     []string{config.FrontendOrigin},
    AllowMethods:     []string{"GET", "POST"},
    AllowHeaders:     []string{"X-Csrf-Token"},
    ExposeHeaders:    []string{"Content-Length"},
    AllowCredentials: true,
    MaxAge: 12 * time.Hour,
  }))

  var router gin.IRoutes = engine
  if config.MountPath != "" {
    router = engine.Group(config.MountPath)
  }

  model := model.New(db)
  authService := auth.NewService(&config, model)
  authService.Route(router)
  blockStore := blocks.NewService(&config, rc)
  blockStore.Route(router)
  eventService, err := events.NewService(&config, rc, model, authService)
  if err != nil {
    log.Panicf("Failed to connect to create event service: %s\n", err)
  }
  go eventService.Run()
  eventService.Route(router)
  chains.NewService(&config, eventService, model, authService, blockStore).Route(router)
  landing.NewService(&config, model, authService).Route(router)
  teams.NewService(&config, model, authService).Route(router)
  games.NewService(&config, rc, model, eventService, blockStore).Route(router)
  contests.NewService(&config, model, authService).Route(router)

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


  router.POST("/System/Ping", func(c *gin.Context) {
    rc.Publish("system", "ping")
    c.String(200, "OK")
  })

  /*router.GET("/CsrfToken", func(c *gin.Context) {
    c.Data(200, "text/plain", []byte(csrf.GetToken(c)))
  });*/

  router.GET("/CsrfToken.js", func(c *gin.Context) {
    /* Token is base64-encoded and thus safe to inject with %s. */
    token := csrf.GetToken(c)
    c.Data(200, "application/javascript",
      []byte(fmt.Sprintf(";window.csrfToken = \"%s\";", token)))
  });

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

  engine.Run(config.Listen)
}
