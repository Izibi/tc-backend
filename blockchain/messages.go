
package blockchain

import (
  "bufio"
  "io"
  "regexp"
  "strings"
  "unsafe"
  "github.com/go-errors/errors"
  "github.com/json-iterator/go"
  j "tezos-contests.izibi.com/backend/jase"
)

var reMessagePrefix = regexp.MustCompile("^Prefix: (.*)$")

func writeMessages (w io.Writer, r io.Reader) error {
  var err error
  scanner := bufio.NewScanner(r)
  if scanner.Scan() {
    firstLine := scanner.Text()
    ms := reMessagePrefix.FindStringSubmatch(firstLine)
    if len(ms) != 2 {
      return errors.Errorf("bad prefix line: %s", firstLine)
    }
    prefix := ms[1]
    log := j.Array()
    for scanner.Scan() {
      line := scanner.Text()
      if strings.HasPrefix(line, prefix) {
        /* Write out and clear text log. */
        err = writeLog(w, log)
        if err != nil { return err }
        log = j.Array()
        /* Write out delimited message. */
        message := []byte(line[len(prefix) + 1 : len(line)])
        writeMessage(w, message)
        if err != nil { return err }
      } else {
        log.Item(j.String(line))
      }
    }
    writeLog(w, log)
    if err != nil { return err }
  }
  err = scanner.Err()
  if err != nil { return errors.Wrap(err, 0) }
  return nil
}

func writeLog(w io.Writer, log j.Value) error {
  var err error
  entry := j.Object()
  entry.Prop("log", log)
  _, err = entry.WriteTo(w)
  if err != nil { return errors.Wrap(err, 0) }
  _, err = w.Write([]byte("\n"))
  if err != nil { return errors.Wrap(err, 0) }
  return nil
}

func writeMessage(w io.Writer, message []byte) error {
  var err error
  entry := j.Object()
  entry.Prop("message", j.Raw(message))
  _, err = entry.WriteTo(w)
  if err != nil { return errors.Wrap(err, 0) }
  _, err = w.Write([]byte("\n"))
  if err != nil { return errors.Wrap(err, 0) }
  return nil
}

/* findLastState scans the message log to find the last 'state' type message,
   returning the raw value of its 'state' property. */
func findLastState(r io.Reader) ([]byte, error) {
  var err error
  var state []byte
  scanner := bufio.NewScanner(r)
  for scanner.Scan() {
    line := scanner.Bytes()
    message := jsoniter.Get(line, "message")
    if message.LastError() == nil && message.Get("type").ToString() == "state" {
      /* XXX (*objectLazyAny)ToString uses a runtime trick to avoid copying
         the underlying bytes; we use the same trick to recover an alias to
         the same array (keeping in mind the original slice's capacity is lost).
         Is there a better way to do this?
       */
      state = veryUnsafeStringToByteSlice(message.Get("state").ToString())
    }
  }
  err = scanner.Err()
  if err != nil { return nil, errors.Wrap(err, 0) }
  return state, nil
}

func veryUnsafeStringToByteSlice(bs string) []byte {
  return *(*[]byte)(unsafe.Pointer(&bs))
}
