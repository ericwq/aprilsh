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

package network

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"io"
	"sync"
)

type Compressor struct{}

var combo struct {
	compressor *Compressor
	sync.Mutex
}

func GetCompressor() *Compressor {
	combo.Lock()
	defer combo.Unlock()

	if combo.compressor == nil {
		combo.compressor = &Compressor{}
	}
	return combo.compressor
}

func (c *Compressor) Compress(input []byte) ([]byte, error) {
	var buf bytes.Buffer

	w := zlib.NewWriter(&buf)
	w.Write(input)
	// _, err := w.Write(input)
	// if err != nil {
	// 	return nil, err
	// }
	w.Close()

	return buf.Bytes(), nil
}

const bufSize = 1250 // a little bit bigger than network MTU

func (c *Compressor) Uncompress(input []byte) ([]byte, error) {
	buf := make([]byte, bufSize)

	if len(input) > bufSize {
		return nil, fmt.Errorf("content length exceed the buffer size %d", bufSize)
	}
	b := bytes.NewReader(input)

	r, err := zlib.NewReader(b)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	n, err := r.Read(buf)
	if err == io.EOF && n > 0 {
		return buf[:n], nil
	}

	return nil, err
}
