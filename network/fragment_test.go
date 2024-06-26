// Copyright 2022 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package network

import (
	"io"
	"log/slog"
	"reflect"
	"strings"
	"testing"

	pb "github.com/ericwq/aprilsh/protobufs"
	"github.com/ericwq/aprilsh/util"
	"google.golang.org/protobuf/proto"
)

func TestFragment(t *testing.T) {
	tc := []struct {
		name     string
		contents string
		id       uint64
		num      uint16
		final    bool
	}{
		{"english false frag", "first fragment.", 12, 1, false},
		{"english true frag", "first fragment.", 0, 0, true},
		{"chinese true frag", "第一块分片。", 0, 0, true},
		{"chinese false frag", "第一块分片。", 23, 5, false},
	}

	for _, v := range tc {

		f0 := NewFragment(v.id, v.num, v.final, v.contents)

		mid := f0.String()

		f1 := NewFragmentFrom(mid)

		if !reflect.DeepEqual(f0, f1) { // *f0 != *f1 {
			t.Errorf("%q expect \n%#v, got \n%#v\n", v.name, f0, f1)
		}
	}

	f0 := NewFragment(1, 2, true, "not initialized frag")
	f0.initialized = false
	ret := f0.String()
	if ret != "" {
		t.Errorf("%q expect %q, got %q\n", f0.contents, "", ret)
	}

	x := "< fragLen"
	f1 := NewFragmentFrom(x)
	if f1 != nil {
		t.Errorf("%q expect nil, got %#v\n", x, f1)
	}
}

func TestFragmenter(t *testing.T) {
	name := "inst-> frag -> inst"
	fe := NewFragmenter()

	in0 := new(pb.Instruction)
	in0.ProtocolVersion = APRILSH_PROTOCOL_VERSION
	in0.OldNum = 9
	in0.NewNum = 10
	in0.AckNum = 8
	in0.ThrowawayNum = 6
	in0.Diff = []byte("This is the diff part of instruction. A string is a sequence of one or more characters (letters, numbers, symbols) that can be either a constant or a variable. Made up of Unicode, strings are immutable sequences, meaning they are unchanging. Because text is such a common form of data that we use in everyday life, the string data type is a very important building block of programming. This Go tutorial will go over how to create and print strings, how to concatenate and replicate strings, and how to store strings in variables.`")
	in0.Chaff = []byte("what is the chaff?")

	mtu := 120

	frags := fe.makeFragments(in0, mtu)
	got := fe.lastAckSent()
	if got != uint64(in0.AckNum) {
		t.Errorf("%q expect AckNum=%d, got %d\n", name, in0.AckNum, got)
	}

	fa := NewFragmentAssembly()
	success := false
	for i, frag := range frags {
		success = fa.addFragment(frag)
		if success && i != len(frags)-1 {
			t.Errorf("%q expect success=%t, got %t\n", name, false, success)
		}
		// fmt.Printf("%q %d success=%t\n", name, i, success)
	}
	// fmt.Printf("%q id=%d, total=%d, arrival=%d, len=%d\n",
	// 	name, fa.currentId, fa.fragmentsTotal, fa.fragmentsArrived, len(fa.fragments))

	in1 := fa.getAssembly()
	if in1 == nil {
		t.Errorf("%q expct instruction=\n%#v, got nil\n", name, in1)
	}
	if !proto.Equal(in0, in1) {
		t.Errorf("%q expect \n%#v, got %#v\n", name, in0, in1)
	}
}

// func TestLastAckSentMax(t *testing.T) {
// 	fe := NewFragmenter()
//
// 	in0 := new(pb.Instruction)
// 	in0.ProtocolVersion = APRILSH_PROTOCOL_VERSION
// 	in0.OldNum = 9
// 	in0.NewNum = 10
// 	in0.AckNum = math.MaxUint64
// 	in0.ThrowawayNum = 6
// 	in0.Diff = []byte("simple message")
// 	in0.Chaff = []byte("chaff")
//
// 	mtu := 120
//
// 	fe.makeFragments(in0, mtu)
// 	got := fe.lastAckSentShutdown()
// 	if !got {
// 		t.Errorf("#test lastAckSentMax expect true, got %t\n", got)
// 	}
// }

func TestAddFragmentSkip(t *testing.T) {
	tc := []struct {
		contents string
		id       uint64
		num      uint16
		final    bool
	}{
		{"you ", 1, 2, false},
		{"I ", 1, 0, false},
		{"you ", 1, 2, false}, // repeated frag
		{"love ", 1, 1, false},
		{"love ", 1, 1, false}, // repeated frag
		{"too.", 1, 3, true},
	}

	// expected result
	expect := "I love you too."
	name := "out-of-order and repeat fragments"

	// prepare the out-of-order fragments
	var frags []*Fragment
	for _, v := range tc {
		f := NewFragment(v.id, v.num, v.final, v.contents)
		frags = append(frags, f)
	}

	fa := NewFragmentAssembly()
	for _, frag := range frags {
		success := fa.addFragment(frag)
		if success {
			break
		}
		// fmt.Printf("%q %d success=%t\n", name, i, success)
	}

	var b strings.Builder
	for i := range fa.fragments {
		b.WriteString(fa.fragments[i].contents)
		// fmt.Printf("#test %d - %#v\n", i, fa.fragments[i])
	}
	got := b.String()
	if got != expect {
		t.Errorf("%q expect %q, got %q\n", name, expect, got)
	}
}

func TestGetAssemblyFail(t *testing.T) {
	tc := []struct {
		contents string
		id       uint64
		num      uint16
		final    bool
	}{
		{"you ", 1, 2, false},
		{"I ", 1, 0, false},
		{"love ", 1, 1, false},
		{"too.", 1, 3, true},
	}

	// prepare the out-of-order fragments
	var frags []*Fragment
	for _, v := range tc {
		f := NewFragment(v.id, v.num, v.final, v.contents)
		frags = append(frags, f)
	}

	fa := NewFragmentAssembly()
	for _, frag := range frags {
		success := fa.addFragment(frag)
		if success {
			break
		}
	}

	// intercept the log
	util.Logger.CreateLogger(io.Discard, true, slog.LevelDebug)
	// util.Logger.CreateLogger(os.Stdout, true, slog.LevelDebug)

	// validate uncompress error
	// # this can also test zlib.NewReader(b)
	in := fa.getAssembly()
	if in != nil {
		t.Errorf("#test getAssembly() failed expect nil, got %#v\n", in)
	}

	// prepare data for unmarshal error condition
	f0 := NewFragment(2022, 12, true, "no instruction data")
	mid, _ := GetCompressor().Compress([]byte(f0.contents))
	f0.contents = string(mid)

	fa.fragmentsArrived = 1
	fa.fragmentsTotal = 1
	fa.currentId = 2022
	fa.fragments = make([]*Fragment, 0)
	fa.fragments = append(fa.fragments, f0)

	in = fa.getAssembly()
	if in != nil {
		t.Errorf("#test getAssembly() failed expect nil, got %#v\n", in)
	}
}
