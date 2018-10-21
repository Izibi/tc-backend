
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

func (svc *Service) MakeSetupBlock(parentHash string, params []byte) (hash string, output j.Value, err error) {

  params, err = j.PrettyBytes(params)
  if err != nil { err = errors.Wrap(err, 0); return }

  block := SetupBlock{
    Params: hashResource(params),
  }
  err = svc.chainBlock(&block.BlockBase, "setup", parentHash)
  if err != nil { return }
  encodedBlock := block.Marshal()
  hash, err = svc.writeBlock(encodedBlock)
  if os.IsExist(err) { return hash, nil, nil }
  if err != nil { return }
  defer func () {
    if err != nil {
      svc.deleteBlock(hash)
    }
  }()

  blockPath := svc.blockDir(hash)
  err = ioutil.WriteFile(filepath.Join(blockPath, "params.json"), params, 0644)
  if err != nil { err = errors.Wrap(err, 0); return }

  /* Compile the setup code. */
  cmd := newCommand(
    svc.taskToolsPath(block.Task),
    "-t", svc.blockDir(block.Task),
    "-p", svc.blockDir(block.Protocol),
    "-b", svc.blockDir(hash),
    "build_setup")
  err = cmd.Run(nil)
  if err != nil { return }

  /* Generate the initial state. */
  cmd = newCommand(
    svc.taskToolsPath(block.Task),
    "-t", svc.blockDir(block.Task),
    "-p", svc.blockDir(block.Protocol),
    "-b", svc.blockDir(hash),
    "run_setup")
  /* task_tool looks for params.json in its current directory */
  cmd.Dir(blockPath)
  err = cmd.Run(bytes.NewReader(params))
  if err != nil {
    err = errors.Errorf("Failed to run setup\n  params: %s\n  details: %s",
      string(params), string(cmd.Stderr.Bytes()))
    return
  }
  output = nil

  err = svc.finalizeBlock(hash, &block, &cmd.Stdout)
  if err != nil { return }

  return
}
