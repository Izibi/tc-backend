
package jase

import (
  "io"
)

type invalid struct {
  err error
}

func (i *invalid) Write(w io.Writer) (int, error) {
  return 0, i.err
}
