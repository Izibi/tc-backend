
package blocks

import (
  "encoding/json"
  "fmt"
  "io"
  "io/ioutil"
  "os"
  "path/filepath"
  "github.com/go-errors/errors"
  j "tezos-contests.izibi.com/backend/jase"
)

type Block interface {
  Base() *BlockBase
}

func (store *Store) IsBlock(hash string) bool {
  if !validateHash(hash) { return false }
  blockDir := store.blockDir(hash)
  fi, err := os.Stat(blockDir)
  return err != nil && fi.IsDir()
}

func (store *Store) readBlock(hash string) (block Block, err error) {
  if !validateHash(hash) { return nil, errors.New("invalid hash") }
  blockPath := filepath.Join(store.blockDir(hash), "block.json")
  blockBytes, err := ioutil.ReadFile(blockPath)
  if err != nil { err = errors.Wrap(err, 0); return }
  var base BlockBase
  err = json.Unmarshal(blockBytes, &base)
  if err != nil { err = errors.Wrap(err, 0); return }
  switch base.Kind {
  case "task":
    block = new(TaskBlock)
  case "protocol":
    block = new(ProtocolBlock)
  case "setup":
    block = new(SetupBlock)
  case "commands":
    block = new(CommandBlock)
  default:
    block = &base
    return
  }
  err = json.Unmarshal(blockBytes, block)
  if err != nil { err = errors.Wrap(err, 0); return }
  return
}

func (store *Store) chainBlock(dst *BlockBase, kind string, parentHash string) error {
  /* Load the parent block. */
  parentBlock, err := store.readBlock(parentHash)
  if err != nil { return err }
  parentBase := parentBlock.Base()
  *dst = *parentBase
  (*dst).Kind = kind
  (*dst).Sequence = parentBase.Sequence + 1
  (*dst).Parent = parentHash
  switch parentBase.Kind {
    case "task":
      (*dst).Task = parentHash
      (*dst).Protocol = ""
      (*dst).Setup = ""
    case "protocol":
      (*dst).Protocol = parentHash
      (*dst).Setup = ""
    case "setup":
      (*dst).Setup = parentHash
  }
  return nil
}

func (store *Store) writeBlock(block j.Value) (hash string, err error) {
  blockBytes, err := j.ToPrettyBytes(block)
  if err != nil { err = errors.Wrap(err, 0); return }
  hash = hashBlock(blockBytes)
  blockDir := store.blockDir(hash)
  err = os.MkdirAll(blockDir, 0755)
  if err != nil { err = errors.Wrap(err, 0); return }
  blockPath := filepath.Join(blockDir, "block.json")
  err = createFile(blockPath, blockBytes, 0644)
  if os.IsExist(err) { return /* unwrapped */ }
  if err != nil { err = errors.Wrap(err, 0); return }
  fmt.Printf("[store] create %s\n", hash)
  return
}

func (store *Store) finalizeBlock(hash string, block Block, stdout io.Reader) error {

  fmt.Printf("[store] finalize %s\n", hash)
  blockDir := store.blockDir(hash)

  /* Decode the command output.
     TODO: do this in a goroutine instead of reading all input into a buffer. */
  logFile, err := os.OpenFile(filepath.Join(blockDir, "output.json"), os.O_CREATE | os.O_RDWR, 0644)
  if err != nil { return errors.Wrap(err, 0) }
  defer logFile.Close()
  err = writeMessages(logFile, stdout)
  if err != nil { return err }

  /* Rewing the structured message output, extract and save the final 'state'
     as 'state.json' in the block directory. */
  logFile.Seek(0, os.SEEK_SET)
  state, err := findLastState(logFile)
  if err != nil { return err }
  err = ioutil.WriteFile(filepath.Join(blockDir, "state.json"), state, 0644)
  if err != nil { return errors.Wrap(err, 0) }

  /* Run the helper. */
  cmd := newCommand(store.taskHelperPath(block.Base().Task), store.blockDir(hash))
  err = cmd.Run(nil)
  if err != nil { return errors.Wrap(err, 0) }

  return nil
}

func (store *Store) deleteBlock(hash string) error {
  fmt.Printf("[store] delete %s\n", hash)
  return os.RemoveAll(store.blockDir(hash))
}
