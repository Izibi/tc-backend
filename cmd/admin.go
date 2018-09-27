
package main

import (
  "database/sql"
  "errors"
  "flag"
  "fmt"
  "io/ioutil"
  "log"
  "os"

  "github.com/json-iterator/go"
  _ "github.com/go-sql-driver/mysql"
)

var config jsoniter.Any
var chainId *uint = flag.Uint("chainId", 0, "id of chain to alter")
var db *sql.DB

func importChainProtocol() error {
  var err error
  var impl, intf []byte
  impl, err = ioutil.ReadFile("protocol.ml")
  if err != nil { return err }
  intf, err = ioutil.ReadFile("protocol.mli")
  if err != nil { return err }
  var res sql.Result
  res, err = db.Exec(`UPDATE chains SET interface_text = ?, implementation_text = ? WHERE id = ?`,
    string(intf), string(impl), *chainId)
  if err != nil { return err }
  var rows int64
  rows, err = res.RowsAffected()
  fmt.Printf("%d rows affected\n", rows)
  return nil
}

func main() {
  var err error

  var configFile []byte
  configFile, err = ioutil.ReadFile("config.json")
  if err != nil {
    log.Panicf("Failed to read configuration file: %s\n", err)
  }
  config = jsoniter.Get(configFile)

  db, err = sql.Open("mysql", config.Get("db").ToString())
  if err != nil {
    log.Panicf("Failed to connect to database: %s\n", err)
  }

  flag.Parse()
  var cmd string = flag.Arg(0)
  switch cmd {
  case "importChainProtocol":
    err = importChainProtocol()
  default:
    err = errors.New("unknown command")
  }
  if err != nil {
    fmt.Fprintf(os.Stderr, "error: %v\n", err)
    os.Exit(1)
  }
}
