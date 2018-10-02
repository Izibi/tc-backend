
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

func NewKeyPair () (*KeyPair, error) {
  pub, pri, err := ed25519.GenerateKey(rand.Reader)
  if err != nil { return nil, err }
  return &KeyPair{
    Curve: "ed25519",
    Public: wrapKey(pub),
    Private: wrapKey(pri),
  }, nil
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

func (kp *KeyPair) Encode() ([]byte, error) {
  res := j.Object()
  res.Prop("curve", j.String(kp.Curve))
  res.Prop("public", j.String(kp.Public))
  res.Prop("private", j.String(kp.Private))
  return j.ToPrettyBytes(res)
}

func (kp *KeyPair) Write (w io.Writer) error {
  err := json.NewEncoder(w).Encode(kp)
  if err != nil { return err }
  return nil
}
