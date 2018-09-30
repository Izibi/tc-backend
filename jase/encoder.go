
package jase

import (
  "bytes"
  "io"
  "io/ioutil"
)

type Value interface {
  io.WriterTo
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
  n, _ := v.WriteTo(ioutil.Discard)
  b := bytes.NewBuffer(make([]byte, n))
  b.Reset()
  _, err := v.WriteTo(b)
  if err != nil { return []byte{}, err }
  return b.Bytes(), nil
}

func ToString(v Value) (string, error) {
  bs, err := ToBytes(v)
  if err != nil { return "", err }
  return string(bs), nil
}

func ToPrettyBytes(v Value) ([]byte, error) {
  var err error
  var bs []byte
  bs, err = ToBytes(v)
  if err != nil { return []byte{}, err }
  bs, err = PrettyBytes(bs)
  if err != nil { return []byte{}, err }
  return bs, nil
}

func ToPrettyString(v Value) (string, error) {
  bs, err := ToPrettyBytes(v)
  if err != nil { return "", err }
  return string(bs), nil
}
