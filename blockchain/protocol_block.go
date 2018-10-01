
package blockchain

import (
  "encoding/json"
  "io/ioutil"
  "os"
  "path/filepath"
  "github.com/go-errors/errors"
  j "tezos-contests.izibi.com/backend/jase"
)

type ProtocolBlock struct {
  BlockBase
  Interface string `json:"interface"`
  Implementation string `json:"implementation"`
}

func (b *ProtocolBlock) Marshal() j.IObject {
  res := b.marshalBase()
  res.Prop("interface", j.String(b.Interface))
  res.Prop("implementation", j.String(b.Implementation))
  return res
}

func (store *Store) MakeProtocolBlock (parentHash string, intf, impl []byte) (hash string, err error) {

  block := ProtocolBlock{
    Interface: hashResource(intf),
    Implementation: hashResource(impl),
  }
  err = store.chainBlock(&block.BlockBase, "protocol", parentHash)
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
  err = ioutil.WriteFile(filepath.Join(blockPath, "bare_protocol.mli"), intf, 0644)
  if err != nil { return }
  err = ioutil.WriteFile(filepath.Join(blockPath, "bare_protocol.ml"), impl, 0644)
  if err != nil { return }

  cmd := newCommand(
    store.taskToolsPath(block.Task),
    "-t", store.blockDir(block.Task),
    "-p", blockPath,
    "build_protocol")
  err = cmd.Run(nil)
  if err != nil { return }

  var output struct {
    Error string `json:"error"`
    Details string `json:"details"`
    InterfaceLog string `json:"interface_log"`
    ImplementationLog string `json:"implementation_log"`
  }
  err = json.Unmarshal(cmd.Stdout.Bytes(), &output)
  if err != nil {
    err = errors.Errorf("failed to parse output: %s\n%s", err, cmd.Stdout.Bytes())
    return
  }

  return
}
