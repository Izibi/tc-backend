
package game

import (
  "bytes"
  "io"
  "fmt"
  "os/exec"
  "github.com/go-errors/errors"
)

type command struct {
  name string
  cmd *exec.Cmd
  stdin io.WriteCloser
  stdout bytes.Buffer
  stderr bytes.Buffer
}

type Writable interface {
  Write(w io.Writer) (int, error)
}

func run (name string, args ...string) (*command, error) {
  var err error
  res := new(command)
  res.name = name
  res.cmd = exec.Command(name, args...)
  res.cmd.Stdout = &res.stdout
  res.cmd.Stderr = &res.stderr
  res.stdin, err = res.cmd.StdinPipe()
  if err != nil {
    fmt.Println("A")
    return nil, errors.Wrap(err, 0)
  }
  err = res.cmd.Start()
  if err != nil {
    fmt.Println("B")
    _ = res.stdin.Close()
    return nil, errors.Wrap(err, 0)
  }
  return res, nil
}

func (c *command) SendInput(w Writable) error {
  _, err := w.Write(c.stdin)
  if err != nil {
    fmt.Println("C")
    return errors.Wrap(err, 0)
  }
  err = c.stdin.Close()
  if err != nil {
    fmt.Println("D")
    return errors.Wrap(err, 0)
  }
  return nil
}

func (c *command) Wait() error {
  err := c.cmd.Wait()
  if err != nil {
    fmt.Printf("E %v\n", err)
    ee := err.(*exec.ExitError)
    if ee != nil {
      fmt.Printf("EE %v\n", ee)
      return nil
    }
    return errors.Wrap(err, 0)
  }
  if !c.cmd.ProcessState.Success() {
    fmt.Println("F")
    return errors.Errorf("%s failed", c.name)
  }
  return nil
}

func (c *command) Stdout() []byte {
  return c.stdout.Bytes()
}

func (c *command) Stderr() []byte {
  return c.stderr.Bytes()
}
