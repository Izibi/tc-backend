
package utils

import (
  "bufio"
  "fmt"
  "path/filepath"
  "runtime"
  "strings"

  "github.com/fatih/color"
)

var traceMark = color.New(color.Bold, color.FgRed)

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
