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
	"encoding/binary"
	"unsafe"

	pb "github.com/ericwq/aprilsh/protobufs"
)

type Fragment struct {
	id          uint64
	fragmentNum uint16
	final       bool
	initialized bool
	contents    string
}

func NewFragment(id uint64, fragmentNum uint16, final bool, contents string) *Fragment {
	f := new(Fragment)
	f.id = id
	f.fragmentNum = fragmentNum
	f.final = final
	f.initialized = true
	f.contents = contents

	return f
}

// convert network order string into Fragment
func NewFragmentFrom(x string) *Fragment {
	f := new(Fragment)
	f.initialized = true

	if len(x) < f.fragHeaderLen() {
		return nil
	}

	buf := []byte(x)
	// read id
	f.id = binary.BigEndian.Uint64(buf[0:])

	// fragmentNum + final
	combo := binary.BigEndian.Uint16(buf[unsafe.Sizeof(f.id):])

	// fragmentNum
	f.fragmentNum = combo & 0x7fff

	// final
	if combo&0x8000 > 0 {
		f.final = true
	}

	// data
	f.contents = string(buf[f.fragHeaderLen():])
	return f
}

func (f *Fragment) fragHeaderLen() int {
	return int(unsafe.Sizeof(f.id) + unsafe.Sizeof(f.fragmentNum))
}

// convert Fragment into network order string
func (f *Fragment) String() string {
	if !f.initialized {
		return ""
	}

	buf := new(bytes.Buffer)

	// id
	var p []byte
	p = make([]byte, unsafe.Sizeof(f.id))
	binary.BigEndian.PutUint64(p, f.id)
	buf.Write(p)

	// fragmentNum + final
	p = make([]byte, unsafe.Sizeof(f.fragmentNum))
	combo := f.fragmentNum
	if f.final {
		combo |= 0x8000
	}
	binary.BigEndian.PutUint16(p, combo)
	buf.Write(p)

	// contents
	binary.Write(buf, binary.BigEndian, f.contents)

	return string(buf.Bytes())
}

type FragmentAssembly struct {
	fragments        []*Fragment
	currentId        uint64
	fragmentsArrived int
	fragmentsTotal   int
}

type Fragmenter struct {
	nextInstructionId uint64
	lastInstruction   pb.Instruction
	lastMTU           int
}
