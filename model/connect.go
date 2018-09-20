
package model

import (
  "database/sql"
  _ "github.com/go-sql-driver/mysql"
)

type Model struct {
  db *sql.DB
}

func Connect (target string) (*Model, error) {
  db, err := sql.Open("mysql", target)
  if err != nil { return nil, err }
  return &Model{db}, nil
}
