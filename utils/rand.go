
package utils

import (
  "crypto/rand"
  "encoding/base64"
  "encoding/binary"
  "fmt"
  "math/big"
  "github.com/go-errors/errors"
  "github.com/itchyny/base58-go"
)

func NewState() (string, error) {
  b := make([]byte, 32)
  _, err := rand.Read(b)
  if err != nil { return "", err }
  return base64.URLEncoding.EncodeToString(b), nil
}

func NewKey() (string, error) {
  bs := make([]byte, 24, 24)
  _, err := rand.Read(bs)
  if err != nil { return "", err }
  var num big.Int
  key, err := base58.BitcoinEncoding.Encode([]byte(num.SetBytes(bs).String()))
  if err != nil { return "", errors.Wrap(err, 0) }
  return string(key), nil
}


func NewAccessCode() (string, error) {
  binCode := make([]byte, 8)
  _, err := rand.Read(binCode)
  if err != nil { return "", errors.Wrap(err, 0) }
  intCode := binary.LittleEndian.Uint64(binCode)
  strCode := fmt.Sprintf("%d", intCode)
  accessCode, err := base58.BitcoinEncoding.Encode([]byte(strCode))
  if err != nil { return "", errors.Wrap(err, 0) }
  return string(accessCode), nil
}

