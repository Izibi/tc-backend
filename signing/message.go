
package signing

import (
  "crypto/hmac"
  "crypto/sha512"
)

func hashMessage(apiKey []byte, message []byte) []byte {
  //fmt.Printf("hashMessage %s %s\n", hex.EncodeToString(apiKey), string(message))
  hasher := hmac.New(sha512.New, []byte(apiKey))
  hasher.Write([]byte(message))
  return hasher.Sum(nil)[:32]
}
