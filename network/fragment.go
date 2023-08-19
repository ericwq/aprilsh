// Copyright 2022 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package network

import (
	"bytes"
	"encoding/binary"
	"sort"
	"strings"
	"unsafe"

	pb "github.com/ericwq/aprilsh/protobufs"
	"github.com/ericwq/aprilsh/util"
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

func (f *FragmentAssembly) has(frag *Fragment) (ret bool, idx int) {
	idx = -1
	for i, fs := range f.fragments {
		if fs.fragmentNum == frag.fragmentNum {
			ret = true
			idx = i
			break
		}
	}
	return
}

// check makeFragments() for the addFragment() logic
func (f *FragmentAssembly) addFragment(frag *Fragment) bool {
	if f.currentId != frag.id {
		// this is a totally new packet

		// fmt.Printf("#addFragment add* #%d\n", frag.fragmentNum)
		f.fragments = make([]*Fragment, 0)
		f.fragments = append(f.fragments, frag) // consider the order problem caused by UDP
		f.fragmentsArrived = 1
		f.fragmentsTotal = -1 // unknown
		f.currentId = frag.id
	} else {
		// not a new packet
		// see if we already have this fragments
		hasThis, idx := f.has(frag)
		if hasThis {
			// fmt.Printf("#addFragment skip #%d\n", frag.fragmentNum)
			if *(f.fragments[idx]) == *frag {
				// make sure new version is same as what we already have
			}
		} else {
			// fmt.Printf("#addFragment add #%d\n", frag.fragmentNum)
			f.fragments = append(f.fragments, frag)
			f.fragmentsArrived++
			// sort the fragment slice
			sort.SliceStable(f.fragments, func(i, j int) bool {
				return f.fragments[i].fragmentNum < f.fragments[j].fragmentNum
			})
		}
	}

	if frag.final {
		f.fragmentsTotal = int(frag.fragmentNum) + 1
	}

	// fmt.Printf("#addFragment arrived=%d, total=%d\n", f.fragmentsArrived, f.fragmentsTotal)

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
		util.Log.With("error", err).Warn("#getAssembly uncompress")
		return nil
	}

	err = proto.Unmarshal(b, &ret)
	if err != nil {
		util.Log.With("error", err).Warn("#getAssembly unmarshal")
		return nil
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
	return uint64(f.lastInstruction.AckNum)
	// return f.lastInstruction.AckNum
}

// last instruction contains AckNum equals -1, which means shutdown.
func (f *Fragmenter) lastAckSentShutdown() bool {
	return f.lastInstruction.AckNum == -1
}

// convert Instruction into Fragments slice.
func (f *Fragmenter) makeFragments(inst *pb.Instruction, mtu int) (ret []*Fragment) {
	// each fragment needs to consider the actually: mtu - header
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
	pos := 0

	util.Log.With("p0", len(p0)).With("payload", len(payload)).With("mtu", mtu).Debug("send fragments")

	for payload != nil {
		final := false
		thisFragment := ""

		if len(payload[pos:]) > mtu {
			thisFragment = string(payload[pos : pos+mtu])
			pos += mtu
		} else {
			thisFragment = string(payload[pos:])
			payload = nil
			final = true
		}

		ret = append(ret, NewFragment(f.nextInstructionId, fragmentNum, final, thisFragment))
		fragmentNum++
	}

	return ret
}
