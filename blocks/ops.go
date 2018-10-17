
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

func (svc *Service) IsBlock(hash string) bool {
  if !validateHash(hash) { return false }
  blockDir := svc.blockDir(hash)
  fi, err := os.Stat(blockDir)
  return err == nil && fi.IsDir()
}

func (svc *Service) ReadBlock(hash string) (block Block, err error) {
  if !validateHash(hash) { return nil, errors.New("invalid hash") }
  blockPath := filepath.Join(svc.blockDir(hash), "block.json")
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

func (svc *Service) chainBlock(dst *BlockBase, kind string, parentHash string) error {
  /* Load the parent block. */
  parentBlock, err := svc.ReadBlock(parentHash)
  if err != nil { return fmt.Errorf("failed to read parent block %s", parentHash) }
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

func (svc *Service) writeBlock(block j.Value) (hash string, err error) {
  blockBytes, err := j.ToPrettyBytes(block)
  if err != nil { err = errors.Wrap(err, 0); return }
  hash = hashBlock(blockBytes)
  blockDir := svc.blockDir(hash)
  err = os.MkdirAll(blockDir, 0755)
  if err != nil { err = errors.Wrap(err, 0); return }
  blockPath := filepath.Join(blockDir, "block.json")
  err = createFile(blockPath, blockBytes, 0644)
  if os.IsExist(err) { return /* unwrapped */ }
  if err != nil { err = errors.Wrap(err, 0); return }
  fmt.Printf("[svc] create %s\n", hash)
  return
}

func (svc *Service) finalizeBlock(hash string, block Block, stdout io.Reader) error {

  fmt.Printf("[svc] finalize %s\n", hash)
  blockDir := svc.blockDir(hash)

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
  cmd := newCommand(svc.taskHelperPath(block.Base().Task), svc.blockDir(hash))
  err = cmd.Run(nil)
  if err != nil { return errors.Wrap(err, 0) }

  return nil
}

func (svc *Service) deleteBlock(hash string) error {
  if svc.config.Blocks.SkipDelete {
    fmt.Printf("[svc] delete %s (skipped)\n", hash)
    return nil
  }
  fmt.Printf("[svc] delete %s\n", hash)
  return os.RemoveAll(svc.blockDir(hash))
}
