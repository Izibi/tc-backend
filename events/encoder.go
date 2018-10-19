
package events

import (
  "bytes"
  "fmt"
  "strconv"
  "strings"
  j "tezos-contests.izibi.com/backend/jase"
)

type Encoder struct {
  lastId uint64
  lastEvent string
}

func NewEncoder() *Encoder {
  return &Encoder{0, "message"}
}

func (enc *Encoder) Encode(m *SSEvent) []byte {
  var buf bytes.Buffer
  if m.Id != 0 && m.Id != enc.lastId {
    buf.WriteString(fmt.Sprintf("id: %s\n", strconv.FormatUint(m.Id, 10)))
    enc.lastId = m.Id
  }
  if m.Event != "" && m.Event != enc.lastEvent {
    buf.WriteString(fmt.Sprintf("event: %s\n", noLF(m.Event)))
    enc.lastEvent = m.Event
  }
  if len(m.Data) > 0 {
    lines := strings.Split(m.Data, "\n")
    for _, line := range lines {
      buf.WriteString(fmt.Sprintf("data: %s\n", line))
    }
  }
  buf.WriteString("\n")
  return buf.Bytes()
}

func noLF(s string) string {
  return strings.Replace(s, "\n", "", -1)
}

func encodeMessage(channel string, payload string) string {
  var obj = j.Object()
  obj.Prop("channel", j.String(channel))
  obj.Prop("payload", j.String(payload))
  res, err := j.ToString(obj)
  if err != nil { panic(err) }
  return res
}
