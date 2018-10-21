
package jase

import (
  "io"
  "strconv"
)

type atom struct {
  raw []byte
}

func (a *atom) WriteTo(w io.Writer) (int64, error) {
  n, err := w.Write(a.raw)
  return int64(n), err
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

func Int64(i int64) Value {
  return Raw([]byte(strconv.FormatInt(i, 10)))
}

func Uint64(u uint64) Value {
  return Raw([]byte(strconv.FormatUint(u, 10)))
}
