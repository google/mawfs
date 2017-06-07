// Copyright 2016 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Various utilities.

package blockstore

import (
	"bytes"
	"errors"
)

func encode(buf *bytes.Buffer, val uint32) {
	b := byte(val)
	switch {
	case b < 26:
		buf.WriteByte(b + 'A')
	case b < 52:
		buf.WriteByte('a' + (b - 26))
	case b < 62:
		buf.WriteByte('0' + (b - 52))
	default:
		if b&1 == 1 {
			buf.WriteByte('_')
		} else {
			buf.WriteByte('.')
		}
	}
}

// Encodes 'data' in an encoding similar to base 64 except that:
//
//   1) '+' is replaced with '.', '/' is replaced with '_'.
//   2) No newlines or whitespace are ever added.
//   3) The results are not padded with '='.
//
// This encoding has been implemented because, unlike true base64, it's
// representation is suitable for use as a posix filename and also in URLs.
func altEncode(data []byte) string {
	result := bytes.Buffer{}

	for i := 0; i < len(data); {
		var (
			accum    uint32
			numChars = 2
		)

		accum = uint32(data[i]) << 16
		i += 1
		if i < len(data) {
			accum |= uint32(data[i]) << 8
			i += 1
			numChars += 1
		}
		if i < len(data) {
			accum |= uint32(data[i])
			i += 1
			numChars += 1
		}

		encode(&result, accum>>18)
		encode(&result, (accum>>12)&0x3f)
		if numChars >= 3 {
			encode(&result, (accum>>6)&0x3f)
		}
		if numChars == 4 {
			encode(&result, accum&0x3f)
		}
	}

	return string(result.Bytes())
}

// Decodes a string encoded by altEncode().
func altDecode(encoded string) ([]byte, error) {
	var (
		result = bytes.Buffer{}
		accum  uint32
		i      int
		ch     rune
	)

	for i, ch = range encoded {
		switch {
		case ch >= '0' && ch <= '9':
			ch = ch - '0' + 52
		case ch >= 'a' && ch <= 'z':
			ch = ch - 'a' + 26
		case ch >= 'A' && ch <= 'Z':
			ch -= 'A'
		case ch == '.':
			ch = 62
		case ch == '_':
			ch = 63
		default:
			return nil, errors.New("Invalid character in decode")
		}

		accum = accum<<6 | uint32(ch)

		// Write every 4th byte.
		if (i+1)%4 == 0 {
			result.WriteByte(byte(accum >> 16))
			result.WriteByte(byte(accum >> 8))
			result.WriteByte(byte(accum))
			accum = 0
		}
	}
	i += 1

	if i%4 == 2 {
		result.WriteByte(byte(accum >> 4))
	} else if i%4 == 3 {
		result.WriteByte(byte(accum >> 10))
		result.WriteByte(byte(accum >> 2))
	}

	return result.Bytes(), nil
}
