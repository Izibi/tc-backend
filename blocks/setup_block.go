
package blocks

import (
  "bytes"
  "path/filepath"
  "io/ioutil"
  "os"
  "github.com/go-errors/errors"
  j "tezos-contests.izibi.com/backend/jase"
)

type SetupBlock struct {
  BlockBase
  Params string `json:"params"`
}

func (b *SetupBlock) Marshal() j.IObject {
  res := b.marshalBase()
  res.Prop("params", j.String(b.Params))
  return res
}

func (store *Store) MakeSetupBlock(parentHash string, params []byte) (hash string, err error) {

  params, err = j.PrettyBytes(params)
  if err != nil { err = errors.Wrap(err, 0); return }

  block := SetupBlock{
    Params: hashResource(params),
  }
  err = store.chainBlock(&block.BlockBase, "setup", parentHash)
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
  err = ioutil.WriteFile(filepath.Join(blockPath, "params.json"), params, 0644)
  if err != nil { err = errors.Wrap(err, 0); return }

  /* Compile the setup code. */
  cmd := newCommand(
    store.taskToolsPath(block.Task),
    "-t", store.blockDir(block.Task),
    "-p", store.blockDir(block.Protocol),
    "-b", store.blockDir(hash),
    "build_setup")
  err = cmd.Run(nil)
  // TODO: error {error: "error building setup", details: buildOutcome.stderr}
  if err != nil { return }

  /* Generate the initial state. */
  cmd = newCommand(
    store.taskToolsPath(block.Task),
    "-t", store.blockDir(block.Task),
    "-p", store.blockDir(block.Protocol),
    "-b", store.blockDir(hash),
    "run_setup")
  /* task_tool looks for params.json in its current directory */
  cmd.Dir(blockPath)
  err = cmd.Run(bytes.NewReader(params))
  if err != nil {
    err = errors.Errorf("Failed to run setup\n  params: %s\n  details: %s",
      string(params), string(cmd.Stderr.Bytes()))
    return
  }

  err = store.finalizeBlock(hash, &block, &cmd.Stdout)
  if err != nil { return }

  return
}
