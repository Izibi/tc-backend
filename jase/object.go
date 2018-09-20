
package jase

import (
  "io"
)

type object struct {
  props []property
}

type property struct {
  name string
  value Value
}

func Object() IObject {
  return &object{}
}

func (o *object) Prop(name string, val Value) {
  o.props = append(o.props, property{name, val})
}

func (o *object) Write(w io.Writer) (int, error) {
  m := 0
  if n, err := w.Write([]byte("{")); err != nil { return m, err } else { m += n }
  if len(o.props) > 0 {
    if n, err := o.props[0].Write(w); err != nil { return m, err } else { m += n }
    for _, prop := range o.props[1:] {
      if n, err := w.Write([]byte(",")); err != nil { return m, err } else { m += n }
      if n, err := prop.Write(w); err != nil { return m, err } else { m += n }
    }
  }
  if n, err := w.Write([]byte("}")); err != nil { return m, err } else { m += n }
  return m, nil
}

func (p *property) Write(w io.Writer) (int, error) {
  m := 0
  if n, err := String(p.name).Write(w); err != nil { return m, err } else { m += n }
  if n, err := w.Write([]byte(":")); err != nil { return m, err } else { m += n }
  if p.value == nil {
    if n, err := Null.Write(w); err != nil { return m, err } else { m += n }
  } else {
    if n, err := p.value.Write(w); err != nil { return m, err } else { m += n }
  }
  return m, nil
}
