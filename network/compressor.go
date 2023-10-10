// Copyright 2022 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package network

import (
	"bytes"
	"compress/zlib"
	"errors"
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

// const bufSize = 2250 // a little bit bigger than network MTU
// buffer size bigger than mtu is wrong. some Uncompress data is larger than payload

func (c *Compressor) Uncompress(input []byte) ([]byte, error) {
	bufSize := len(input) * 4 // limited by mtu and compress ratio: we assume 4
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
		if n > bufSize {
			return nil, errors.New("overflow buffer size")
		}
		return buf[:n], nil
	}

	return nil, err
}
