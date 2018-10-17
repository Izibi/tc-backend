
package blocks

import (
  "os"
  j "tezos-contests.izibi.com/backend/jase"
)

type TaskBlock struct {
  BlockBase
  Identifier string `json:"identifier"`
}

func (b *TaskBlock) Marshal() j.IObject {
  res := b.marshalBase()
  res.Prop("identifier", j.String(b.Identifier))
  return res
}

func (svc *Service) MakeTaskBlock(parentHash string, identifier string) (hash string, err error) {

  block := TaskBlock{
    Identifier: identifier,
  }
  err = svc.chainBlock(&block.BlockBase, "task", parentHash)
  if err != nil { return }
  encodedBlock := block.Marshal()
  hash, err = svc.writeBlock(encodedBlock)
  if os.IsExist(err) { return hash, nil }
  if err != nil { return }

  return
}
