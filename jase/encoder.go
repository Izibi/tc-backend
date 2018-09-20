
package jase

import (
  "bytes"
  "io"
  "io/ioutil"
)

type Value interface {
  Write(w io.Writer) (int, error)
}

type IObject interface {
  Value
  Prop(key string, val Value)
}

type IArray interface {
  Value
  Item(val Value)
}

func ToBytes(v Value) ([]byte, error) {
  n, _ := v.Write(ioutil.Discard)
  b := bytes.NewBuffer(make([]byte, n))
  b.Reset()
  _, err := v.Write(b)
  if err != nil { return []byte{}, err }
  return b.Bytes(), nil
}

func ToString(v Value) (string, error) {
  bs, err := ToBytes(v)
  if err != nil { return "", err }
  return string(bs), nil
}
