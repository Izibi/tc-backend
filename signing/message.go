
package signing

import (
  "crypto/hmac"
  "crypto/sha512"
)

func hashMessage(apiKey []byte, message []byte) []byte {
  hasher := hmac.New(sha512.New, []byte(apiKey))
  hasher.Write([]byte(message))
  return hasher.Sum(nil)[:32]
}
