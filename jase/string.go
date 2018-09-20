// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package jase

import (
  "bytes"
  "unicode/utf8"
)

var hex = "0123456789abcdef"

// NOTE: keep in sync with stringBytes below.
func String(s string) Value {
  e := bytes.Buffer{}
  /* Begin code copied from encoding/json/encode.go, with HTML and JSONP concerns removed. */
  e.WriteByte('"')
  start := 0
  for i := 0; i < len(s); {
    if b := s[i]; b < utf8.RuneSelf {
      if safeSet[b] {
        i++
        continue
      }
      if start < i {
        e.WriteString(s[start:i])
      }
      e.WriteByte('\\')
      switch b {
      case '\\', '"':
        e.WriteByte(b)
      case '\n':
        e.WriteByte('n')
      case '\r':
        e.WriteByte('r')
      case '\t':
        e.WriteByte('t')
      default:
        // This encodes bytes < 0x20 except for \t, \n and \r.
        e.WriteString(`u00`)
        e.WriteByte(hex[b>>4])
        e.WriteByte(hex[b&0xF])
      }
      i++
      start = i
      continue
    }
    c, size := utf8.DecodeRuneInString(s[i:])
    if c == utf8.RuneError && size == 1 {
      if start < i {
        e.WriteString(s[start:i])
      }
      e.WriteString(`\ufffd`)
      i += size
      start = i
      continue
    }
    i += size
  }
  if start < len(s) {
    e.WriteString(s[start:])
  }
  e.WriteByte('"')
  /* End code copied from encoding/json/encode.go */
  return Raw(e.Bytes())
}

func StringBytes(s []byte) Value {
  e := bytes.Buffer{}
  /* Begin code copied from encoding/json/encode.go, with HTML and JSONP concerns removed. */
  e.WriteByte('"')
  start := 0
  for i := 0; i < len(s); {
    if b := s[i]; b < utf8.RuneSelf {
      if safeSet[b] {
        i++
        continue
      }
      if start < i {
        e.Write(s[start:i])
      }
      e.WriteByte('\\')
      switch b {
      case '\\', '"':
        e.WriteByte(b)
      case '\n':
        e.WriteByte('n')
      case '\r':
        e.WriteByte('r')
      case '\t':
        e.WriteByte('t')
      default:
        // This encodes bytes < 0x20 except for \t, \n and \r.
        e.WriteString(`u00`)
        e.WriteByte(hex[b>>4])
        e.WriteByte(hex[b&0xF])
      }
      i++
      start = i
      continue
    }
    c, size := utf8.DecodeRune(s[i:])
    if c == utf8.RuneError && size == 1 {
      if start < i {
        e.Write(s[start:i])
      }
      e.WriteString(`\ufffd`)
      i += size
      start = i
      continue
    }
    i += size
  }
  if start < len(s) {
    e.Write(s[start:])
  }
  e.WriteByte('"')
  /* End code copied from encoding/json/encode.go */
  return Raw(e.Bytes())
}
