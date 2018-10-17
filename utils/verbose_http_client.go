
package utils

import (
  "fmt"
  "os"
  "bytes"
  "io/ioutil"
  "net/http"
)

func VerboseHttpClient() *http.Client {
  return &http.Client{
    Transport: wrappedRT{http.DefaultTransport},
  }
}

type wrappedRT struct {
    base http.RoundTripper
}

func (rt wrappedRT) RoundTrip(r *http.Request) (*http.Response, error) {
  fmt.Printf("http> %s\n", r.URL)
  r.Header.Write(os.Stdout)
  var bs []byte
  bs, err := ioutil.ReadAll(r.Body)
  if err != nil { return nil, err }
  fmt.Printf("http> Body %s\n", string(bs))
  r.Body = ioutil.NopCloser(bytes.NewReader(bs))
  return rt.base.RoundTrip(r)
}
