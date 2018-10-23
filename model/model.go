
package model

import (
  "database/sql"
  "github.com/jmoiron/sqlx"
  "github.com/jmoiron/modl"
  "github.com/go-errors/errors"
  "context"
  _ "github.com/go-sql-driver/mysql"
)

type Model struct {
  ctx context.Context
  db *sqlx.DB
  dbMap *modl.DbMap
  tables Tables
}

func New (ctx context.Context, db *sql.DB) *Model {
  model := new(Model)
  model.ctx = ctx
  if err := db.Ping(); err != nil {
    panic("database is unreachable")
  }
  model.db = sqlx.NewDb(db, "mysql")
  model.dbMap = modl.NewDbMap(db, modl.MySQLDialect{"InnoDB", "UTF8"})
  model.tables.Map(model.dbMap)
  return model
}

func (m *Model) transaction(cb func () error) error {
  tx, err := m.db.BeginTx(m.ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
  if err != nil { return errors.Wrap(err, 0) }
  err = cb()
  if err != nil {
    tx.Rollback()
    return err
  }
  err = tx.Commit()
  if err != nil {
    return errors.Wrap(err, 0)
  }
  return nil
}

type IRow interface {
  Scan(dest ...interface{}) error
  StructScan(dest interface{}) error
}
