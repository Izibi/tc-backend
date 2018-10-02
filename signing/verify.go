
package signing

import (
  "bytes"
  "encoding/base64"
  "golang.org/x/crypto/ed25519"
  "github.com/json-iterator/go"
  "github.com/go-errors/errors"
  j "tezos-contests.izibi.com/backend/jase"
)

func Verify(apiKey string, message []byte) error {
  pub, err := extractAuthorKey(message)
  if err != nil { return err }
  rawApiKey, _ :=  base64.StdEncoding.DecodeString(apiKey)
  message, err = j.PrettyBytes(message)
  if err != nil { return errors.WrapPrefix(err, "bad message", 0) }
  message, sig := extractSignature(message)
  if sig == nil { return errors.New("signature not found") }
  hash := hashMessage(rawApiKey, message)
  if !ed25519.Verify(pub, hash, sig) { return errors.New("bad signature") }
  return nil
}

func extractAuthorKey(message []byte) ([]byte, error) {
  author := jsoniter.Get(message, "author").ToString()
  if len(author) == 0 || author[0] != '@' {
    return nil, errors.New("bad author")
  }
  pub, err := unwrapKey(author[1:])
  if err != nil {
    return nil, errors.New("bad public key")
  }
  return pub, nil
}

func extractSignature(msg []byte) (bare []byte, sig []byte) {
  out := new(bytes.Buffer)
  l := len(msg)
  if string(msg[l-121:l-103]) != ",\n  \"signature\": \"" { return msg, nil }
  if string(msg[l-15:]) != ".sig.ed25519\"\n}" { return msg, nil }
  out.Write(msg[:l-121])
  out.Write([]byte("\n}"))
  return out.Bytes(), msg[l-103:l-15]
}
