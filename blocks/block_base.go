
package blocks

import (
  j "tezos-contests.izibi.com/backend/jase"
)

type BlockBase struct {
  Kind string `json:"type"`
  Parent string `json:"parent"`
  Sequence uint64 `json:"sequence"`
  Task string `json:"task"`
  Protocol string `json:"protocol"`
  Setup string `json:"setup"`
  Round uint64 `json:"round"`
}

func (b *BlockBase) Base() *BlockBase {
  return b
}

func (b *BlockBase) marshalBase() j.IObject {
  res := j.Object()
  res.Prop("type", j.String(b.Kind))
  res.Prop("parent", j.String(b.Parent))
  res.Prop("sequence", j.Uint64(b.Sequence))
  if b.Task != "" {
    res.Prop("task", j.String(b.Task))
  }
  if b.Protocol != "" {
    res.Prop("protocol", j.String(b.Protocol))
  }
  if b.Setup != "" {
    res.Prop("setup", j.String(b.Setup))
  }
  res.Prop("round", j.Uint64(b.Round))
  return res
}
