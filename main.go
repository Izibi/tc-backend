
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
  "github.com/Masterminds/semver"
  //"golang.org/x/net/context"
  "gopkg.in/yaml.v2"

  //j "tezos-contests.izibi.com/backend/jase"
  "tezos-contests.izibi.com/backend/utils"
  "tezos-contests.izibi.com/backend/model"
  "tezos-contests.izibi.com/backend/auth"
  "tezos-contests.izibi.com/backend/teams"
  "tezos-contests.izibi.com/backend/contests"
  "tezos-contests.izibi.com/backend/game"
  "tezos-contests.izibi.com/backend/blockchain"

)

type Config struct {
  Listen string `yaml:"listen"`
  MountPath string `yaml:"mount_path"`
  SessionSecret string `yaml:"session_secret"`
  CsrfSecret string `yaml:"csrf_secret"`
  DataSource string `yaml:"data_source"`
  FrontendOrigin string `yaml:"frontend_origin"`
  ApiVersion string `yaml:"api_version"`
  Auth auth.Config `yaml:"auth"`
  Game game.Config `yaml:"game"`
  Blockchain blockchain.Store `yaml:"blockchain"`
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
  game.SetupRoutes(backend, config.Game, db)
  teams.SetupRoutes(backend, db)
  contests.SetupRoutes(backend, db)
  blockchain.SetupRoutes(backend, &config.Blockchain)

  r.GET("/Time", func (c *gin.Context) {
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
      ServerTime string `json:"server_time"`
    }
    res := Response{ServerTime: time.Now().Format(time.RFC3339)}
    c.JSON(200, &res)
  })

  r.POST("/Keypair", func (c *gin.Context) {
    // ssbKeys.generate()
  })

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
  if config.Blockchain.Path == "" {
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
