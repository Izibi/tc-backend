
package jase

import (
  "io"
  "strconv"
)

type atom struct {
  raw []byte
}

func (a *atom) Write(w io.Writer) (int, error) {
  return w.Write(a.raw)
}

func Raw(val []byte) Value {
  return &atom{val}
}

var Null Value = Raw([]byte("null"))

func Boolean(b bool) Value {
  if (b) {
    return Raw([]byte("true"))
  } else {
    return Raw([]byte("false"))
  }
}

func Int(i int) Value {
  return Raw([]byte(strconv.FormatInt(int64(i), 10)))
}

func Uint(u uint) Value {
  return Raw([]byte(strconv.FormatUint(uint64(u), 10)))
}
