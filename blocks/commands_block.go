
package blocks

import (
  "encoding/json"
  "strings"
  "path/filepath"
  "io/ioutil"
  "os"
  "github.com/go-errors/errors"
  j "tezos-contests.izibi.com/backend/jase"
)

type CommandBlock struct {
  BlockBase
  Commands string `json:"commands"`
}

func (b *CommandBlock) Marshal() j.IObject {
  res := b.marshalBase()
  res.Prop("commands", j.String(b.Commands))
  return res
}

func (svc *Service) MakeCommandBlock(parentHash string, commands []byte) (hash string, err error) {

  commands, err = j.PrettyBytes(commands)
  if err != nil { return }

  block := CommandBlock{
    Commands: hashResource(commands),
  }
  err = svc.chainBlock(&block.BlockBase, "command", parentHash)
  if err != nil { return }
  encodedBlock := block.Marshal()
  hash, err = svc.writeBlock(encodedBlock)
  if os.IsExist(err) { return hash, nil }
  if err != nil { return }
  defer func () {
    if err != nil {
      svc.deleteBlock(hash)
    }
  }()

  blockPath := svc.blockDir(hash)
  err = ioutil.WriteFile(filepath.Join(blockPath, "commands.json"), commands, 0644)
  if err != nil { return }

  /* Compile the commands.  The task tool will look for commands.json in the
     block directory (-b). */
  cmd := newCommand(
    svc.taskToolsPath(block.Task),
    "-t", svc.blockDir(block.Task),
    "-p", svc.blockDir(block.Protocol),
    "-b", svc.blockDir(hash),
    "build_commands")
  cmd.Dir(blockPath)
  err = cmd.Run(nil)
  // TODO: error {error: "error compiling the commands", details: buildOutcome.stderr}
  if err != nil { return }

  /* Generate the initial state. */
  cmd = newCommand(
    svc.taskToolsPath(block.Task),
    "-t", svc.blockDir(block.Task),
    "-p", svc.blockDir(block.Protocol),
    "-b", svc.blockDir(hash),
    "run_commands")
  /* The task tool will load its inital state from state.json in the current
     directory, so run the tool in the directory of the parent block.
     XXX If the protocol writes files, it has the opportunity to alter the parent block.
   */
  cmd.Dir(svc.blockDir(parentHash))
  err = cmd.Run(nil)
  if err != nil { err = errors.Wrap(err, 0); return }
  // TODO {error: "error running setup", details: runOutcome.stderr};

  err = svc.finalizeBlock(hash, &block, &cmd.Stdout)
  if err != nil { return }

  return
}

func (svc *Service) CheckCommands(block *BlockBase, commands string) (result []byte, err error) {

  commands = strings.Replace(commands, "\r\n", "\n", -1)

  if block.Protocol == "" {
    err = errors.New("block has protocol")
    return
  }

  cmd := newCommand(
    svc.taskToolsPath(block.Task),
    "-t", svc.blockDir(block.Task),
    "-p", svc.blockDir(block.Protocol),
    "check_commands")
  err = cmd.Run(strings.NewReader(commands))
  if err != nil {
    err = errors.WrapPrefix(err, "error checking commands", 0)
    return
  }

  var res struct {
    Commands json.RawMessage `json:"commands"`
    Error string `json:"error"`
    Details string `json:"details"`
  }
  json.Unmarshal(cmd.Stdout.Bytes(), &res)
  if res.Error != "" {
    err = errors.Errorf("%s\n%s", res.Error, res.Details)
    return
  }

  return res.Commands, nil
}
