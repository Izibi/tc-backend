
package model

import (
  "database/sql"
  "strconv"
  "time"
  "github.com/go-errors/errors"
  "github.com/go-sql-driver/mysql"
  j "tezos-contests.izibi.com/backend/jase"
)

func timeProp(obj j.IObject, key string, val time.Time) {
  obj.Prop(key, j.String(val.Format(time.RFC3339)))
}

func nullTimeProp(obj j.IObject, key string, val mysql.NullTime) {
  if val.Valid {
    obj.Prop(key, j.String(val.Time.Format(time.RFC3339)))
  } else {
    obj.Prop(key, j.Null)
  }
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

func (m *Model) ImportId(id string) int64 {
  n, err := strconv.ParseInt(id, 10, 64)
  if err != nil { return 0 }
  return n
}

func (m *Model) ExportId(id int64) string {
  return strconv.FormatInt(id, 10)
}
