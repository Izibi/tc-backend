package game

import (
  "bytes"
  "crypto/sha1"
  "encoding/base64"
  "io"
  "io/ioutil"
  "os"
  "path/filepath"
  "github.com/go-errors/errors"
  j "tezos-contests.izibi.com/backend/jase"
)

func writeBlock(config Config, block j.Value) (hash string, blockDir string, err error) {
  blockBytes, err := j.ToPrettyBytes(block)
  if err != nil { err = errors.Wrap(err, 0); return }
  hashState := sha1.New()
  io.Copy(hashState, bytes.NewReader(blockBytes))
  hash = base64.RawURLEncoding.EncodeToString(hashState.Sum(nil))
  blockDir = filepath.Join(config.BlockStorePath, hash)
  err = os.MkdirAll(blockDir, 0755)
  if err != nil { err = errors.Wrap(err, 0); return }
  blockPath := filepath.Join(blockDir, "block.json")
  err = ioutil.WriteFile(blockPath, blockBytes, 0644)
  if err != nil { err = errors.Wrap(err, 0); return }
  return
}
