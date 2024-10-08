// Copyright (C) 2013-2018 by Maxim Bublis <b@codemonkey.ru>
//
// Permission is hereby granted, free of charge, to any person obtaining
// a copy of this software and associated documentation files (the
// "Software"), to deal in the Software without restriction, including
// without limitation the rights to use, copy, modify, merge, publish,
// distribute, sublicense, and/or sell copies of the Software, and to
// permit persons to whom the Software is furnished to do so, subject to
// the following conditions:
//
// The above copyright notice and this permission notice shall be
// included in all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
// EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
// MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
// NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE
// LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION
// OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION
// WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package uuid

import (
	"bytes"
	"encoding/hex"
	"fmt"
)

// FromBytes returns UUID converted from raw byte slice input.
// It will return error if the slice isn't 16 bytes long.
func FromBytes(input []byte) (u UUID, err error) {
	if len(input) != Size {
		return Nil, fmt.Errorf("uuid: expected %d bytes, got %d bytes", Size, len(input))
	}
	err = u.UnmarshalBinary(input)
	return
}

// FromBytesOrNil returns UUID converted from raw byte slice input.
// Same behavior as FromBytes, but returns a Nil UUID on error.
func FromBytesOrNil(input []byte) UUID {
	uuid, err := FromBytes(input)
	if err != nil {
		return Nil
	}
	return uuid
}

// FromString returns UUID parsed from string input.
// Input is expected in a form accepted by UnmarshalText.
func FromString(input string) (u UUID, err error) {
	err = u.UnmarshalText([]byte(input))
	if err != nil {
		return Nil, fmt.Errorf("uuid: failed to parse UUID from string: %s", input)
	}
	return
}

// FromStringOrNil returns UUID parsed from string input.
// Same behavior as FromString, but returns a Nil UUID on error.
func FromStringOrNil(input string) UUID {
	uuid, err := FromString(input)
	if err != nil {
		return Nil
	}
	return uuid
}

// MarshalText implements the encoding.TextMarshaler interface.
// The encoding is the same as returned by String.
func (u UUID) MarshalText() (text []byte, err error) {
	text = []byte(u.String())
	return
}

// UnmarshalText implements the encoding.TextUnmarshaler interface.
// Following formats are supported:
//
//	"6ba7b810-9dad-11d1-80b4-00c04fd430c8",
//	"{6ba7b810-9dad-11d1-80b4-00c04fd430c8}",
//	"urn:uuid:6ba7b810-9dad-11d1-80b4-00c04fd430c8"
//	"6ba7b8109dad11d180b400c04fd430c8"
//
// ABNF for supported UUID text representation follows:
//
//	uuid := canonical | hashlike | braced | urn
//	plain := canonical | hashlike
//	canonical := 4hexoct '-' 2hexoct '-' 2hexoct '-' 6hexoct
//	hashlike := 12hexoct
//	braced := '{' plain '}'
//	urn := URN ':' UUID-NID ':' plain
//	URN := 'urn'
//	UUID-NID := 'uuid'
//	12hexoct := 6hexoct 6hexoct
//	6hexoct := 4hexoct 2hexoct
//	4hexoct := 2hexoct 2hexoct
//	2hexoct := hexoct hexoct
//	hexoct := hexdig hexdig
//	hexdig := '0' | '1' | '2' | '3' | '4' | '5' | '6' | '7' | '8' | '9' |
//	          'a' | 'b' | 'c' | 'd' | 'e' | 'f' |
//	          'A' | 'B' | 'C' | 'D' | 'E' | 'F'
func (u *UUID) UnmarshalText(text []byte) (err error) {
	switch len(text) {
	case 32:
		return u.decodeHashLike(text)
	case 36:
		return u.decodeCanonical(text)
	case 38:
		return u.decodeBraced(text)
	case 41:
		fallthrough
	case 45:
		return u.decodeURN(text)
	default:
		return fmt.Errorf("uuid: incorrect UUID length: %s", text)
	}
}

// decodeCanonical decodes UUID string in format
// "6ba7b810-9dad-11d1-80b4-00c04fd430c8".
func (u *UUID) decodeCanonical(t []byte) (err error) {
	if t[8] != '-' || t[13] != '-' || t[18] != '-' || t[23] != '-' {
		return fmt.Errorf("uuid: incorrect UUID format %s", string(t))
	}

	dst := u[:]
	for i, byteGroup := range byteGroups {
		if i > 0 {
			t = t[1:] // skip dash
		}
		if _, err := hex.Decode(dst[:byteGroup/2], t[:byteGroup]); err != nil {
			return err
		}
		t = t[byteGroup:]
		dst = dst[byteGroup/2:]
	}

	return nil
}

// decodeHashLike decodes UUID string in format
// "6ba7b8109dad11d180b400c04fd430c8".
func (u *UUID) decodeHashLike(t []byte) (err error) {
	if _, err = hex.Decode(u[:], t); err != nil {
		return fmt.Errorf("uuid: failed to decode hash-like format: %s", string(t))
	}
	return nil
}

// decodeBraced decodes UUID string in format
// "{6ba7b810-9dad-11d1-80b4-00c04fd430c8}" or in format
// "{6ba7b8109dad11d180b400c04fd430c8}".
func (u *UUID) decodeBraced(t []byte) (err error) {
	if len(t) < 2 || t[0] != '{' || t[len(t)-1] != '}' {
		return fmt.Errorf("uuid: incorrect UUID format %s", string(t))
	}
	return u.decodePlain(t[1 : len(t)-1])
}

// decodeURN decodes UUID string in format
// "urn:uuid:6ba7b810-9dad-11d1-80b4-00c04fd430c8" or in format
// "urn:uuid:6ba7b8109dad11d180b400c04fd430c8".
func (u *UUID) decodeURN(t []byte) (err error) {
	if len(t) < 9 || !bytes.Equal(t[:9], urnPrefix) {
		return fmt.Errorf("uuid: incorrect URN format: %s", string(t))
	}
	return u.decodePlain(t[9:])
}

// decodePlain decodes UUID string in canonical format
// "6ba7b810-9dad-11d1-80b4-00c04fd430c8" or in hash-like format
// "6ba7b8109dad11d180b400c04fd430c8".
func (u *UUID) decodePlain(t []byte) (err error) {
	switch len(t) {
	case 32:
		return u.decodeHashLike(t)
	case 36:
		return u.decodeCanonical(t)
	default:
		return fmt.Errorf("uuid: incorrect UUID length: %s", string(t))
	}
}

// MarshalBinary implements the encoding.BinaryMarshaler interface.
func (u UUID) MarshalBinary() (data []byte, err error) {
	return u.Bytes(), nil
}

// UnmarshalBinary implements the encoding.BinaryUnmarshaler interface.
// It will return error if the slice isn't 16 bytes long.
func (u *UUID) UnmarshalBinary(data []byte) (err error) {
	if len(data) != Size {
		return fmt.Errorf("uuid: UUID must be exactly 16 bytes long, got %d bytes", len(data))
	}
	copy(u[:], data)
	return nil
}
