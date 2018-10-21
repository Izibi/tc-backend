
package blocks

import (
  "io/ioutil"
  "os"
  "path/filepath"
  j "tezos-contests.izibi.com/backend/jase"
)

type ProtocolBlock struct {
  BlockBase
  Interface string `json:"interface"`
  Implementation string `json:"implementation"`
}

type BuildProtocolOutput struct { // documentation
  Error string `json:"error"`
  Details string `json:"details"`
  InterfaceLog string `json:"interface_log"`
  ImplementationLog string `json:"implementation_log"`
}

func (b *ProtocolBlock) Marshal() j.IObject {
  res := b.marshalBase()
  res.Prop("interface", j.String(b.Interface))
  res.Prop("implementation", j.String(b.Implementation))
  return res
}

func (svc *Service) MakeProtocolBlock(parentHash string, intf, impl []byte) (hash string, output j.Value, err error) {

  output = nil
  block := ProtocolBlock{
    Interface: hashResource(intf),
    Implementation: hashResource(impl),
  }
  err = svc.chainBlock(&block.BlockBase, "protocol", parentHash)
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
  err = ioutil.WriteFile(filepath.Join(blockPath, "bare_protocol.mli"), intf, 0644)
  if err != nil { return }
  err = ioutil.WriteFile(filepath.Join(blockPath, "bare_protocol.ml"), impl, 0644)
  if err != nil { return }

  cmd := newCommand(
    svc.taskToolsPath(block.Task),
    "-t", svc.blockDir(block.Task),
    "-p", blockPath,
    "build_protocol")
  err = cmd.Run(nil)
  if err != nil { return }

  return
}

func (svc *Service) LoadProtocol(hash string) (intf []byte, impl []byte, err error) {
  blockPath := svc.blockDir(hash)
  intf, err = ioutil.ReadFile(filepath.Join(blockPath, "bare_protocol.mli"))
  if err != nil { return }
  impl, err = ioutil.ReadFile(filepath.Join(blockPath, "bare_protocol.ml"))
  if err != nil { return }
  return
}
