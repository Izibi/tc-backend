
package model

import (
  "time"
  "database/sql"
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
