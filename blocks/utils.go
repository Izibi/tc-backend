
package blocks

import (
  "io"
  "os"
  "regexp"
  "crypto/sha1"
  "crypto/sha256"
  "encoding/base64"
)

var reHash = regexp.MustCompile("^[0-9A-Za-z_-]*$")

func validateHash(hash string) bool {
  return len(hash) == 27 && reHash.Match([]byte(hash))
}

func hashBlock(bs []byte) string {
  hasher := sha1.New()
  hasher.Write(bs)
  return base64.RawURLEncoding.EncodeToString(hasher.Sum(nil))
}

func hashResource(bs []byte) string {
  hasher := sha256.New()
  hasher.Write(bs)
  return base64.URLEncoding.EncodeToString(hasher.Sum(nil))
}

func createFile(filename string, data []byte, perm os.FileMode) error {
  f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_EXCL, perm)
  if err != nil {
    return err
  }
  n, err := f.Write(data)
  if err == nil && n < len(data) {
    err = io.ErrShortWrite
  }
  if err1 := f.Close(); err == nil {
    err = err1
  }
  return err
}

/*
TODO:
async function checkCommands (store, parentHash, input) {
  const parentBlock = await readBlock(store, parentHash);
  const protoHash = parentBlock.type === 'protocol' ? parentHash : parentBlock.protocol;
  const protoPath = path.join(store.blockStorePath, protoHash);
  const args = ["-t", store.taskPath, "-c", protoPath, "check_commands"];
  input = input.replace(/[\r\n]+/g, "\n");
  const outcome = await spawn(store.taskToolsBin, args, input);
  if (outcome.exit_code !== 0 || outcome.stderr.length > 0) {
    return {error: outcome.stderr};
  }
  return JSON.parse(outcome.stdout);
}
module.exports = {makeProtocolBlock, makeSetupBlock, makeCommandBlock, checkCommands};
*/
