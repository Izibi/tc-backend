
package blocks

import (
  "fmt"
  "os"
  j "tezos-contests.izibi.com/backend/jase"
)

type TaskBlock struct {
  BlockBase
  Identifier string `json:"identifier"`
  Revision uint64 `json:"revision"`
}

func (b *TaskBlock) Marshal() j.IObject {
  res := b.marshalBase()
  res.Prop("identifier", j.String(b.Identifier))
  res.Prop("revision", j.Uint64(b.Revision))
  return res
}

func (svc *Service) MakeTaskBlock(parentHash string, identifier string, revision uint64) (hash string, err error) {

  block := TaskBlock{
    Identifier: identifier,
    Revision: revision,
  }
  err = svc.chainBlock(&block.BlockBase, "task", parentHash)
  if err != nil { return }
  encodedBlock := block.Marshal()
  fmt.Printf("TASK %v\n", block)
  hash, err = svc.writeBlock(encodedBlock)
  if os.IsExist(err) { return hash, nil }
  if err != nil { return }

  return
}
