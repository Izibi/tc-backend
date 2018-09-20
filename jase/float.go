// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package jase

import (
  "errors"
  "math"
  "strconv"
)

type floatEncoder int // number of bits

func float(bits int, f float64) Value {

  if math.IsInf(f, 0) || math.IsNaN(f) {
    return &invalid{errors.New("no JSON encoding for " + strconv.FormatFloat(f, 'g', -1, int(bits)))}
  }

  // Convert as if by ES6 number to string conversion.
  // This matches most other JSON generators.
  // See golang.org/issue/6384 and golang.org/issue/14135.
  // Like fmt %g, but the exponent cutoffs are different
  // and exponents themselves are not padded to two digits.
  abs := math.Abs(f)
  fmt := byte('f')
  // Note: Must use float32 comparisons for underlying float32 value to get precise cutoffs right.
  if abs != 0 {
    if bits == 64 && (abs < 1e-6 || abs >= 1e21) || bits == 32 && (float32(abs) < 1e-6 || float32(abs) >= 1e21) {
      fmt = 'e'
    }
  }
  b := []byte(strconv.FormatFloat(f, fmt, -1, bits))
  if fmt == 'e' {
    // clean up e-09 to e-9
    n := len(b)
    if n >= 4 && b[n-4] == 'e' && b[n-3] == '-' && b[n-2] == '0' {
      b[n-2] = b[n-1]
      b = b[:n-1]
    }
  }

  return Raw(b)
}

func Float32(f float32) Value {
  return float(32, float64(f))
}

func Float64(f float64) Value {
  return float(64, f)
}
