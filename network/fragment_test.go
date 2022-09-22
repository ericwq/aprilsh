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
	"reflect"
	"strings"
	"testing"

	pb "github.com/ericwq/aprilsh/protobufs"
	"google.golang.org/protobuf/proto"
)

func TestFragment(t *testing.T) {
	tc := []struct {
		name     string
		id       uint64
		num      uint16
		final    bool
		contents string
	}{
		{"english false frag", 12, 1, false, "first fragment."},
		{"english true frag", 0, 0, true, "first fragment."},
		{"chinese true frag", 0, 0, true, "第一块分片。"},
		{"chinese false frag", 23, 5, false, "第一块分片。"},
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
	in0.ProtocolVersion = 2
	in0.OldNum = 9
	in0.NewNum = 10
	in0.AckNum = 8
	in0.ThrowawayNum = 6
	in0.Diff = []byte("This is the diff part of instruction. A string is a sequence of one or more characters (letters, numbers, symbols) that can be either a constant or a variable. Made up of Unicode, strings are immutable sequences, meaning they are unchanging. Because text is such a common form of data that we use in everyday life, the string data type is a very important building block of programming. This Go tutorial will go over how to create and print strings, how to concatenate and replicate strings, and how to store strings in variables.`")
	in0.Chaff = []byte("what is the chaff?")

	mtu := 120

	frags := fe.makeFragments(in0, mtu)
	got := fe.lastAckSent()
	if got != in0.AckNum {
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

func TestAddFragment(t *testing.T) {
	tc := []struct {
		id       uint64
		num      uint16
		final    bool
		contents string
	}{
		{1, 2, false, "you "},
		{1, 0, false, "I "},
		{1, 2, false, "you "}, // repeated frag
		{1, 1, false, "love "},
		{1, 1, false, "love "}, // repeated frag
		{1, 3, true, "too."},
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
