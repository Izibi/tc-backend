
package jase

import (
  "io"
)

type array struct {
  items []Value
}

func Array() IArray {
  return &array{}
}

func (a *array) Write(w io.Writer) (int, error) {
  m := 0
  if n, err := w.Write([]byte("[")); err != nil { return m, err } else { m += n }
  if len(a.items) > 0 {
    if n, err := a.items[0].Write(w); err != nil { return m, err } else { m += n }
    for _, item := range a.items[1:] {
      if n, err := w.Write([]byte(",")); err != nil { return m, err } else { m += n }
      if n, err := item.Write(w); err != nil { return m, err } else { m += n }
    }
  }
  if n, err := w.Write([]byte("]")); err != nil { return m, err } else { m += n }
  return m, nil
}

func (a *array) Item(val Value) {
  a.items = append(a.items, val)
}
