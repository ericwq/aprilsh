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
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"unsafe"
)

// pseudorandom number generator
type prng struct {
	randfile io.Reader
}

const (
	rdev = "/dev/urandom"
)

func newPrng() *prng {
	p := new(prng)

	// TODO which is the best way to generate random number, sys call or /dev/urandom file?
	randfile, err := os.Open(rdev)
	if err != nil {
		panic(fmt.Sprintf("Open prng file error: %s\n", err))
	}

	p.randfile = randfile
	return p
}

func (p *prng) fill(size int) (dst []byte) {
	if size == 0 {
		return dst
	}

	dst = make([]byte, size)
	err := binary.Read(p.randfile, hostEndian, dst)
	if err != nil {
		panic(fmt.Sprintf("Could not read slice from %q. error: %s\n", rdev, err))
	}
	return
}

// https://medium.com/learning-the-go-programming-language/encoding-data-with-the-go-binary-package-42c7c0eb3e73
func (p *prng) uint8() uint8 {
	var u8 uint8

	err := binary.Read(p.randfile, hostEndian, &u8)
	if err != nil {
		panic(fmt.Sprintf("Could not read uint8 from %q. error: %s\n", rdev, err))
	}
	return u8
}

// https://stackoverflow.com/questions/51332658/any-better-way-to-check-endianness-in-go
var hostEndian binary.ByteOrder

func init() {
	buf := [2]byte{}
	*(*uint16)(unsafe.Pointer(&buf[0])) = uint16(0xABCD)

	switch buf {
	case [2]byte{0xCD, 0xAB}:
		hostEndian = binary.LittleEndian
	case [2]byte{0xAB, 0xCD}:
		hostEndian = binary.BigEndian
	default:
		panic("Could not determine native endianness.")
	}
}
