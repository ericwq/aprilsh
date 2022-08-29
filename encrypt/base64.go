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
	"fmt"
)

// use sys call to generate random number
func prngFill(size int) (dst []byte) {
	dst = make([]byte, size)
	if size == 0 {
		return dst
	}

	_, err := rand.Read(dst)
	// err := binary.Read(p.randfile, hostEndian, dst)
	if err != nil {
		// panic(fmt.Sprintf("Could not read slice from %q. error: %s\n", rdev, err))
		panic(fmt.Sprintf("Could not read random number. %s", err))
	}
	return
}

func prngUint8() uint8 {
	var u8 uint8

	dst := prngFill(1)
	u8 = dst[0]
	return u8
}
