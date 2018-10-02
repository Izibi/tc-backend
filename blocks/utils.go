
package blocks

import (
  "archive/zip"
  "path/filepath"
  "io"
  "io/ioutil"
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

func writeZip(dir string, w io.Writer) error {
  zw := zip.NewWriter(w)
  err := filepath.Walk(dir, func (path string, info os.FileInfo, err error) error {
    if err != nil { return err }
    if info.IsDir() { return nil }
    f, err := zw.Create(info.Name())
    if err != nil { return err }
    bs, err := ioutil.ReadFile(path)
    if err != nil { return err }
    _, err = f.Write(bs)
    if err != nil { return err }
    return nil
  })
  if err != nil { return err }
  err = zw.Close()
  if err != nil { return err }
  return nil
}
