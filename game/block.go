package game

import (
  "bytes"
  "crypto/sha1"
  "encoding/base64"
  "encoding/json"
  "io"
  "io/ioutil"
  "os"
  "path/filepath"
  "github.com/go-errors/errors"
  j "tezos-contests.izibi.com/backend/jase"
)

type Block interface {
  Base() *BlockBase
  Succ() uint64
}
type BlockBase struct {
  Kind string `json:"type"`
  Sequence uint64 `json:"sequence"`
}
type ProtocolBlock struct {
  BlockBase
  Interface string `json:"interface"`
  Implementation string `json:"implementation"`
}
type SetupBlock struct {
  BlockBase
  Parent string `json:"parent"`
  Protocol string `json:"protocol"`
  Game_params json.RawMessage `json:"game_params"`
  Task_params json.RawMessage `json:"task_params"`
}

func (b *BlockBase) Base() *BlockBase {
  return b
}

func (b *BlockBase) Succ() uint64 {
  return b.Sequence + 1
}

func readBlock(config Config, hash string) (block Block, err error) {
  blockPath := filepath.Join(config.BlockDir(hash), "block.json")
  blockBytes, err := ioutil.ReadFile(blockPath)
  if err != nil { err = errors.Wrap(err, 0); return }
  var base BlockBase
  err = json.Unmarshal(blockBytes, &base)
  if err != nil { err = errors.Wrap(err, 0); return }
  switch base.Kind {
  case "protocol":
    block = new(ProtocolBlock)
  case "setup":
    block = new(SetupBlock)
  default:
    block = &base
    return
  }
  err = json.Unmarshal(blockBytes, base)
  if err != nil { err = errors.Wrap(err, 0); return }
  return
}

func writeBlock(config Config, block j.Value) (hash string, err error) {
  blockBytes, err := j.ToPrettyBytes(block)
  if err != nil { err = errors.Wrap(err, 0); return }
  hashState := sha1.New()
  io.Copy(hashState, bytes.NewReader(blockBytes))
  hash = base64.RawURLEncoding.EncodeToString(hashState.Sum(nil))
  blockDir := config.BlockDir(hash)
  err = os.MkdirAll(blockDir, 0755)
  if err != nil { err = errors.Wrap(err, 0); return }
  blockPath := filepath.Join(blockDir, "block.json")
  err = ioutil.WriteFile(blockPath, blockBytes, 0644)
  if err != nil { err = errors.Wrap(err, 0); return }
  return
}
