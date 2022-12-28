// Copyright 2022 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

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
