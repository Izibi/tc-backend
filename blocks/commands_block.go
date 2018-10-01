
package blocks

import (
  "path/filepath"
  "io/ioutil"
  "os"
  "github.com/go-errors/errors"
  j "tezos-contests.izibi.com/backend/jase"
)

type CommandBlock struct {
  BlockBase
  Nb_cycles uint `json:"nb_cycles"`
  Commands string
}

func (b *CommandBlock) Marshal() j.IObject {
  res := b.marshalBase()
  res.Prop("nb_cycles", j.Uint(b.Nb_cycles))
  res.Prop("commands", j.String(b.Commands))
  return res
}

func (store *Store) MakeCommandBlock(parentHash string, nbCycles uint, commands []byte) (hash string, err error) {

  commands, err = j.PrettyBytes(commands)
  if err != nil { return }

  block := CommandBlock{
    Nb_cycles: nbCycles,
    Commands: hashResource(commands),
  }
  err = store.chainBlock(&block.BlockBase, "command", parentHash)
  if err != nil { return }
  encodedBlock := block.Marshal()
  hash, err = store.writeBlock(encodedBlock)
  if os.IsExist(err) { return hash, nil }
  if err != nil { return }
  defer func () {
    if err != nil {
      store.deleteBlock(hash)
    }
  }()

  blockPath := store.blockDir(hash)
  err = ioutil.WriteFile(filepath.Join(blockPath, "commands.json"), commands, 0644)
  if err != nil { return }

  /* Compile the commands.  The task tool will look for commands.json in the
     block directory (-b). */
  cmd := newCommand(
    store.taskToolsPath(block.Task),
    "-t", store.blockDir(block.Task),
    "-p", store.blockDir(block.Protocol),
    "-b", store.blockDir(hash),
    "build_commands")
  cmd.Dir(blockPath)
  err = cmd.Run(nil)
  // TODO: error {error: "error compiling the commands", details: buildOutcome.stderr}
  if err != nil { return }

  /* Generate the initial state. */
  cmd = newCommand(
    store.taskToolsPath(block.Task),
    "-t", store.blockDir(block.Task),
    "-p", store.blockDir(block.Protocol),
    "-b", store.blockDir(hash),
    "run_commands")
  /* The task tool will load its inital state from state.json in the current
     directory, so run the tool in the directory of the parent block.
     XXX If the protocol writes files, it has the opportunity to alter the parent block.
   */
  cmd.Dir(store.blockDir(parentHash))
  err = cmd.Run(nil)
  if err != nil { err = errors.Wrap(err, 0); return }
  // TODO {error: "error running setup", details: runOutcome.stderr};

  err = store.finalizeBlock(hash, &block, &cmd.Stdout)
  if err != nil { return }

  return
}
