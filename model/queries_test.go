
package model

import (
  "database/sql"
  "fmt"
  "log"
  "os"
  "testing"
  "github.com/ory/dockertest"
  "github.com/rubenv/sql-migrate"
)

var db *sql.DB

func TestMain(m *testing.M) {
  var err error

  // uses a sensible default on windows (tcp/http) and linux/osx (socket)
  pool, err := dockertest.NewPool("")
  if err != nil {
    log.Fatalf("Could not connect to docker: %s", err)
  }

  // pulls an image, creates a container based on it and runs it
  log.Println("Starting mariadb container...")
  resource, err := pool.Run("alpine-mariadb", "latest", []string{
    "MYSQL_ROOT_PASSWORD=password",
    "MYSQL_USER=testing",
    "MYSQL_PASSWORD=testing",
    "MYSQL_DATABASE=testing",
  })
  if err != nil {
    log.Fatalf("Could not start resource: %s", err)
  }

  err = pool.Retry(func() error {
    var err error
    db, err = sql.Open("mysql", fmt.Sprintf("testing:testing@(localhost:%s)/testing?parseTime=true", resource.GetPort("3306/tcp")))
    if err != nil { return err }
    return db.Ping()
  })
  if err != nil {
    log.Fatalf("Could not connect to docker: %s", err)
  }

  migrations := &migrate.FileMigrationSource{
    Dir: "../db/migrations",
  }
  n, err := migrate.Exec(db, "mysql", migrations, migrate.Up)
  if err != nil {
    log.Fatalf("Failed to apply migrations: %s", err)
  }
  fmt.Printf("Applied %d migrations.\n", n)

  code := m.Run()

  if err = pool.Purge(resource); err != nil {
    log.Fatalf("Could not purge resource: %s", err)
  }

  os.Exit(code)
}

type userProfile struct {
  id string
  username string
  firstname string
  lastname string
  badges []string
}
func (p *userProfile) Id() string { return p.id }
func (p *userProfile) Username() string { return p.username }
func (p *userProfile) Firstname() string { return p.firstname }
func (p *userProfile) Lastname() string { return p.lastname }
func (p *userProfile) Badges() []string { return p.badges }

func TestSomething(t *testing.T) {
  model := New(db)
  userId, err := model.ImportUserProfile(&userProfile{
    id: "1",
    username: "username",
    firstname: "firstname",
    lastname: "lastname",
    badges: []string{"badge1"},
  })
  if err != nil { t.Error(err) }
  err = model.ViewUser(userId)
  if err != nil { t.Error(err) }
}
