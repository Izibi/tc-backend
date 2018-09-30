
package model

import (
  "time"
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
