
package blocks

import (
  "bytes"
  "fmt"
  "io"
  "os/exec"
  "github.com/go-errors/errors"
)

type command struct {
  name string
  cmd *exec.Cmd
  Stdout bytes.Buffer
  Stderr bytes.Buffer
}

func newCommand(name string, args ...string) *command {
  fmt.Printf("RUN %s %v\n", name, args)
  res := new(command)
  res.name = name
  res.cmd = exec.Command(name, args...)
  res.cmd.Stdout = &res.Stdout
  res.cmd.Stderr = &res.Stderr
  return res
}

func (c *command) Dir(dir string) {
  c.cmd.Dir = dir
}

func (c *command) Run(w io.WriterTo) error {
  var err error
  var stdin io.WriteCloser
  if w != nil {
    stdin, err = c.cmd.StdinPipe()
    if err != nil {
      return errors.Wrap(err, 0)
    }
  }
  err = c.cmd.Start()
  if err != nil {
    if stdin != nil {
      _ = stdin.Close()
    }
    return errors.Wrap(err, 0)
  }
  if w != nil {
    _, err = w.WriteTo(stdin)
    if err != nil {
      return errors.Wrap(err, 0)
    }
    err = stdin.Close()
    if err != nil {
      return errors.Wrap(err, 0)
    }
  }
  err = c.cmd.Wait()
  if err != nil {
    ee := err.(*exec.ExitError)
    if ee != nil {
      stderr := string(c.Stderr.Bytes())
      return errors.New(stderr)
    }
    return errors.Wrap(err, 0)
  }
  if !c.cmd.ProcessState.Success() {
    return errors.Errorf("%s failed", c.name)
  }
  return nil
}
