/*

MIT License

Copyright (c) 2022 wangqi

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.

*/

package encrypt

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

// use sys call to generate random number
func prngFill(size int) (dst []byte) {
	dst = make([]byte, size)
	if size == 0 {
		return dst
	}

	rand.Read(dst)
	// _, err := rand.Read(dst)
	// if err != nil {
	// 	panic(fmt.Sprintf("Could not read random number. %s", err))
	// }
	return
}

func PrngUint8() uint8 {
	var u8 uint8

	dst := prngFill(1)
	u8 = dst[0]
	return u8
}

type Base64Key struct {
	key []uint8
}

// random key
func NewBase64Key() *Base64Key {
	b := &Base64Key{}
	b.key = prngFill(16)
	return b
}

func NewBase64Key2(printableKey string) *Base64Key {
	defer func() {
		if err := recover(); err != nil {
			return
		}
	}()

	key, err := base64.StdEncoding.DecodeString(printableKey)
	if err != nil {
		panic(fmt.Sprintf("Key must be well-formed base64. %s\n", err))
	}

	if len(key) != 16 {
		panic("Key must represent 16 octets.")
	}

	b := &Base64Key{}
	b.key = key
	// // to catch changes after the first 128 bits
	// if printableKey != b.printableKey() {
	// 	panic("Base64 key was not encoded 128-bit key.")
	// }

	return b
}

func (b *Base64Key) printableKey() string {
	return base64.StdEncoding.EncodeToString(b.key)
}

func (b *Base64Key) data() []uint8 {
	return b.key
}
