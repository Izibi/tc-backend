
package signing

import (
  "bytes"
  "fmt"
  "encoding/base64"
  "golang.org/x/crypto/ed25519"
  "github.com/go-errors/errors"
  j "tezos-contests.izibi.com/backend/jase"
)

func Sign(privKey string, apiKey string, message []byte) ([]byte, error) {
  var err error
  message, err = j.PrettyBytes(message)
  if err != nil { return nil, errors.WrapPrefix(err, "bad message", 0) }
  pri, err := unwrapKey(privKey)
  if err != nil { return nil, errors.New("bad private key") }
  rawApiKey, _ :=  base64.StdEncoding.DecodeString(apiKey)
  hash := hashMessage(rawApiKey, message)
  rawSig := ed25519.Sign(pri, hash)
  encSig := base64.StdEncoding.EncodeToString(rawSig) + ".sig.ed25519"
  return injectSignature(message, encSig), nil
}

func injectSignature(message []byte, sig string) []byte {
  out := new(bytes.Buffer)
  l := len(message)
  out.Write(message[:l - 2]) /* replace trailing "\n}" */
  fmt.Fprintf(out, ",\n  %q: %q\n}", "signature", sig)
  return out.Bytes()
}
