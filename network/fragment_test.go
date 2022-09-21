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
	"testing"
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
