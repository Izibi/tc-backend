
package jase

import (
  "io"
)

type invalid struct {
  err error
}

func (i *invalid) WriteTo(w io.Writer) (int64, error) {
  return 0, i.err
}
