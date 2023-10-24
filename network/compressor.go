// Copyright 2022 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package network

import (
	"bytes"
	"compress/zlib"
	"io"
	"sync"
)

type Compressor struct {
	bufSize int
	buf     []byte
}

var combo struct {
	compressor *Compressor
	sync.Mutex
}

func GetCompressor() *Compressor {
	combo.Lock()
	defer combo.Unlock()

	if combo.compressor == nil {
		combo.compressor = &Compressor{}
		combo.compressor.bufSize = 2048 * 2048
		combo.compressor.buf = make([]byte, combo.compressor.bufSize)
	}
	return combo.compressor
}

func (c *Compressor) Compress(input []byte) ([]byte, error) {
	var buf bytes.Buffer

	w := zlib.NewWriter(&buf)
	w.Write(input)
	// n, err := w.Write(input)
	// if err != nil {
	// 	return nil, err
	// }
	w.Close()

	return buf.Bytes(), nil
}

func (c *Compressor) Uncompress(input []byte) ([]byte, error) {
	b := bytes.NewReader(input)

	r, err := zlib.NewReader(b)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	n, err := r.Read(c.buf)
	if err == io.EOF && n < c.bufSize {
		return c.buf[:n], nil
	}

	return nil, err
}
