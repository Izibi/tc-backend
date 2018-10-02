
package signing

import (
  "io"
  "strings"
  "crypto/rand"
  "encoding/base64"
  "encoding/json"
  "golang.org/x/crypto/ed25519"
  j "tezos-contests.izibi.com/backend/jase"
)

type KeyPair struct {
  Curve string `json:"curve"`
  Public string `json:"public"`
  Private string `json:"private"`
}

func NewKeyPair () ([]byte, error) {
  pub, pri, err := ed25519.GenerateKey(rand.Reader)
  if err != nil { return nil, err }
  res := j.Object()
  res.Prop("curve", j.String("ed25519"))
  res.Prop("public", j.String(wrapKey(pub)))
  res.Prop("private", j.String(wrapKey(pri)))
  return j.ToPrettyBytes(res)
}

func wrapKey(key []byte) string {
  return base64.StdEncoding.EncodeToString(key) + ".ed25519"
}

func unwrapKey(key string) ([]byte, error) {
  b64 := strings.Split(key, ".")[0]
  return base64.StdEncoding.DecodeString(b64)
}

func ReadKeyPair (r io.Reader) (*KeyPair, error) {
  var res KeyPair
  err := json.NewDecoder(r).Decode(&res)
  if err != nil {
    return nil, err
  }
  return &res, nil
}

func (kp *KeyPair) WriteKeyPair (w io.Writer) error {
  err := json.NewEncoder(w).Encode(kp)
  if err != nil { return err }
  return nil
}
