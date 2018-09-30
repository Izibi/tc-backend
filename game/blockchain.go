
package game

import (
  "encoding/json"
  "github.com/go-errors/errors"
  j "tezos-contests.izibi.com/backend/jase"
)

type BuildProtocolOutput struct {
  Error string `json:"error"`
  Details string `json:"details"`
  InterfaceLog string `json:"interface_log"`
  ImplementationLog string `json:"implementation_log"`
}

func makeProtocolBlock (config Config, intf, impl string) (hash string, err error) {

  /* Write the block. */
  block := j.Object()
  block.Prop("type", j.String("protocol"))
  block.Prop("sequence", j.Int(0))
  block.Prop("interface", j.String(intf))
  block.Prop("implementation", j.String(impl))
  hash, protoPath, err := writeBlock(config, block)
  if err != nil { return }

  cmd, err := run(config.TaskToolsBin,
    "-t", config.TaskPath,
    "-c", protoPath,
    "build_protocol")
  if err != nil { return }
  err = cmd.SendInput(block)
  if err != nil { return }
  err = cmd.Wait()
  if err != nil { return }

  stderr := string(cmd.Stderr())
  if stderr != "" {
    err = errors.Errorf("build_protocol failed: %s", stderr)
    return
  }

  var output BuildProtocolOutput
  err = json.Unmarshal(cmd.Stdout(), &output)
  if err != nil {
    err = errors.Errorf("failed to parse output: %s\n%s", err, cmd.Stdout())
    return
  }

  return
}
