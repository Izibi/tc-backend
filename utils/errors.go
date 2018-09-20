
package utils

import (
  "bufio"
  "fmt"
  "io"
  "net/http"
  "path/filepath"
  "runtime"
  "strings"

  "github.com/go-errors/errors"

  j "tezos-contests.izibi.com/backend/jase"
)

func (r *Response) Error(err error) {
  r.context.Status(http.StatusOK)
  r.context.Stream(func (w io.Writer) bool {
    JError(err).Write(w)
    return false
  })
}

func (r *Response) StringError(msg string) {
  r.Error(errors.New(msg))
}

func JError(err error) j.Value {
  o := j.Object()
  o.Prop("error", j.String(err.Error()))
  o.Prop("location", j.String(traceLocation(err.(*errors.Error).ErrorStack())))
  return o
}

/* XXX use StackFrames() instead of going through lines */
func traceLocation(str string) string {
  scanner := bufio.NewScanner(strings.NewReader(str))
  _, caller, _, ok := runtime.Caller(0)
  if !ok { return str }
  dir := filepath.Dir(filepath.Dir(caller))
  dir, err := filepath.EvalSymlinks(dir)
  if err != nil { return str }
  for scanner.Scan() {
    line := scanner.Text()
    fmt.Println(line)
    if strings.HasPrefix(line, dir) {
      return line
    }
  }
  return str
}
