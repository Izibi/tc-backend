
package utils

import (
  "bufio"
  "fmt"
  "path/filepath"
  "runtime"
  "strings"

  "github.com/fatih/color"
  "github.com/go-errors/errors"

  j "tezos-contests.izibi.com/backend/jase"
)

var traceMark = color.New(color.Bold, color.FgRed)

func (r *Response) Error(err error) {
  res := j.Object()
  res.Prop("error", j.String(err.Error()))
  err2, ok := err.(*errors.Error)
  if ok {
    res.Prop("location", j.String(traceLocation(err2.ErrorStack())))
  }
  r.Send(res)
}

func (r *Response) StringError(msg string) {
  r.Error(errors.Wrap(msg, 1))
}

func (r *Response) BadUser() {
  r.StringError("you don't exist")
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
    traceMark.Print("XX ")
    fmt.Println(line)
    if strings.HasPrefix(line, dir) {
      return line[len(dir)+1:]
    }
  }
  return str
}
