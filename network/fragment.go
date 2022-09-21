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
	"strings"
	"unsafe"

	pb "github.com/ericwq/aprilsh/protobufs"
	"google.golang.org/protobuf/proto"
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
	buf.WriteString(f.contents)

	return string(buf.Bytes())
}

type FragmentAssembly struct {
	fragments        []*Fragment
	currentId        uint64 // instruction id
	fragmentsArrived int
	fragmentsTotal   int
}

func NewFragmentAssembly() *FragmentAssembly {
	f := new(FragmentAssembly)

	f.fragmentsArrived = 0
	f.fragmentsTotal = -1
	f.fragments = make([]*Fragment, 0)
	return f
}

// check makeFragments() for the addFragment() logic
func (f *FragmentAssembly) addFragment(frag *Fragment) bool {
	// see if this is a totally new packet
	if f.currentId != frag.id {
		// fmt.Printf("#addFragment add* #%d\n", frag.fragmentNum)
		f.fragments = make([]*Fragment, 0)
		f.fragments = append(f.fragments, frag)
		f.fragmentsArrived = 1
		f.fragmentsTotal = -1 // unknown
		f.currentId = frag.id
	} else { // not a new packet
		// see if we already have this fragments
		if len(f.fragments) > int(frag.fragmentNum) && f.fragments[frag.fragmentNum].initialized {
			// make sure new version is same as what we already have
			// fmt.Printf("#addFragment skip #%d\n", frag.fragmentNum)
			if *(f.fragments[frag.fragmentNum]) == *frag {
				// do nothing
			}
		} else {
			// fmt.Printf("#addFragment add #%d\n", frag.fragmentNum)
			f.fragments = append(f.fragments, frag)
			f.fragmentsArrived++
		}
	}

	if frag.final {
		f.fragmentsTotal = int(frag.fragmentNum) + 1
	}

	// return true means all the fragments is arrived.
	return f.fragmentsArrived == f.fragmentsTotal
}

// convert fragments into Instruction
func (f *FragmentAssembly) getAssembly() *pb.Instruction {
	var encoded strings.Builder

	for i := 0; i < f.fragmentsTotal; i++ {
		encoded.WriteString(f.fragments[i].contents)
	}

	ret := pb.Instruction{}
	b, err := GetCompressor().Uncompress([]byte(encoded.String()))
	if err != nil {
		return nil
	}

	err = proto.Unmarshal(b, &ret)
	if err != nil {
		return nil // TODO error handling.
	}

	f.fragments = make([]*Fragment, 0)
	f.fragmentsArrived = 0
	f.fragmentsTotal = -1

	return &ret
}

type Fragmenter struct {
	nextInstructionId uint64
	lastInstruction   *pb.Instruction
	lastMTU           int
}

func NewFragmenter() *Fragmenter {
	f := new(Fragmenter)
	f.nextInstructionId = 0
	f.lastMTU = -1
	f.lastInstruction = new(pb.Instruction)
	f.lastInstruction.OldNum = 0
	f.lastInstruction.NewNum = 0

	return f
}

func (f *Fragmenter) lastAckSent() uint64 {
	return f.lastInstruction.AckNum
}

// convert Instruction into Fragments slice.
func (f *Fragmenter) makeFragments(inst *pb.Instruction, mtu int) []*Fragment {
	mtu -= new(Fragment).fragHeaderLen()

	if !proto.Equal(inst, f.lastInstruction) || f.lastMTU != mtu {
		f.nextInstructionId++
	}

	// TODO: why add this?
	// if inst.OldNum == f.lastInstruction.OldNum && inst.NewNum == f.lastInstruction.NewNum {
	// 	if !reflect.DeepEqual(inst.Diff, f.lastInstruction.Diff) {
	// 		return nil
	// 	}
	// }

	f.lastInstruction = inst
	f.lastMTU = mtu

	data, _ := proto.Marshal(inst)
	p0, _ := GetCompressor().Compress(data)
	payload := []byte(p0)

	var fragmentNum uint16 = 0
	ret := make([]*Fragment, 0)

	pos := 0
	for payload != nil {
		final := false
		contents := ""

		if len(payload[pos:]) > mtu {
			contents = string(payload[pos : pos+mtu])
			pos += mtu
		} else {
			contents = string(payload[pos:])
			payload = nil
			final = true
		}

		ret = append(ret, NewFragment(f.nextInstructionId, fragmentNum, final, contents))
		fragmentNum++
	}

	return ret
}
