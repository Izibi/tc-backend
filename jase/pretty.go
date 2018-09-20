
package jase

import (
  "bytes"
  "encoding/json"
  "fmt"
  "io"
  "strings"
  "github.com/golang-collections/collections/stack"
  "github.com/pkg/errors"
)

type recoder struct {
  dec *json.Decoder
  out *bytes.Buffer
  nesting *stack.Stack
  firstItem bool
  depth int
}

func (r *recoder) Value() (err error) {
  var t json.Token
  t, err = r.Token()
  if err != nil { return }
  switch v := t.(type) {
  case json.Delim: // [ ] { }
    switch v {
    case '{':
      r.Object()
    case '[':
      r.Array()
    default:
      return errors.Errorf("unexpected token %v", v)
    }
  case string:
    fmt.Fprintf(r.out, "%q", v)
  case float64:
    fmt.Fprintf(r.out, "%v", v)
  default:
    if v == nil {
      fmt.Fprint(r.out, "null")
    } else {
      fmt.Fprintf(r.out, "%v", v)
    }
  }
  return nil
}

func (r *recoder) Token() (res interface{}, err error) {
  res, err = r.dec.Token()
  // fmt.Printf("token %v %v\n", res, err)
  if err == io.EOF {
    res = r.nesting.Pop()
    if res == nil { return }
    err = nil
  } else if err != nil {
    return
  }
  switch v := res.(type) {
  case rune:
    switch v {
    case '}':
      r.CloseObject()
    case ']':
      r.CloseArray()
    default:
      return nil, errors.New("bad delimiter on stack")
    }
  case json.Delim: // [ ] { }
    switch v {
    case '{':
      r.depth++
      r.nesting.Push('}')
    case '}':
      r.CloseObject()
    case '[':
      r.depth++
      r.nesting.Push(']')
    case ']':
      r.CloseArray()
    default:
      return nil, errors.New("bad delimiter")
    }
  }
  return
}

func (r *recoder) Array() (err error) {
  fmt.Fprintf(r.out, "[")
  r.firstItem = true
  for r.dec.More() {
    r.Item()
    if err = r.Value(); err != nil { return }
  }
  r.Token()
  return nil
}

func (r *recoder) CloseArray() {
  r.depth--
  if (r.firstItem) {
    r.firstItem = false;
  } else {
    fmt.Fprintf(r.out, "\n")
    fmt.Fprint(r.out, strings.Repeat("  ", r.depth))
  }
  fmt.Fprintf(r.out, "]")
}

func (r *recoder) Object() (err error) {
  fmt.Fprintf(r.out, "{")
  r.firstItem = true
  for r.dec.More() {
    r.Item()
    if err = r.Key(); err != nil { return }
    if err = r.Value(); err != nil { return }
  }
  r.Token()
  return nil
}

func (r *recoder) CloseObject() {
  r.depth--
  if (r.firstItem) {
    r.firstItem = false;
  } else {
    fmt.Fprintf(r.out, "\n")
    fmt.Fprint(r.out, strings.Repeat("  ", r.depth))
  }
  fmt.Fprintf(r.out, "}")
}

func (r *recoder) Item() {
  if (r.firstItem) {
    r.firstItem = false
    fmt.Fprint(r.out, "\n")
  } else {
    fmt.Fprint(r.out, ",\n")
  }
  fmt.Fprint(r.out, strings.Repeat("  ", r.depth))
}

func (r *recoder) Key() error {
  t, err := r.Token()
  if err != nil {
    return err
  }
  switch v := t.(type) {
  case string:
    fmt.Fprintf(r.out, "%q: ", v)
    return nil
  default:
    return errors.Errorf("expected key, got %v", v)
  }
}

func (r *recoder) Bytes() []byte {
  return r.out.Bytes()
}

func Pretty(b []byte) ([]byte, error) {
  r := recoder{
    dec: json.NewDecoder(bytes.NewReader(b)),
    out: &bytes.Buffer{},
    nesting: stack.New(),
    firstItem: false,
    depth: 0,
  }
  err := r.Value()
  if err != nil {
    return []byte{}, err
  }
  return r.Bytes(), nil
}
