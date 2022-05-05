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

package terminal

import (
	"testing"
)

func TestDispatcherGetParam(t *testing.T) {
	tc := []struct {
		name   string
		params string
		want   []int
	}{
		{"normal", "4;57;", []int{4, 57}},
		{"normal", ";5;58", []int{5, 58}},
		{"abnormal", ";12;45;", []int{12, 45}},
		{"too large", "65536;4500;", []int{0, 4500}},
	}

	for _, v := range tc {
		// prepare the test case
		d := Dispatcher{}
		d.params.WriteString(v.params)
		// t.Logf("%v\n", d.params)

		d.parseAll()

		for i := range d.parsedParams {
			got := d.getParam(i, 0)
			if i < len(v.want) {
				if v.want[i] == got {
					continue
				} else {
					t.Errorf("%s:\t case:%s\t [%02d parameter]: expect %d, got %d\n", v.name, v.params, i, v.want[i], got)
				}
			} else {
				t.Errorf("%s:\t case:%s\t got too much result as wanted.", v.name, v.params)
			}
		}
	}
}

func TestDispatcherNewParamChar(t *testing.T) {
	tc := []struct {
		name   string
		params string
		want   []int
	}{
		{"normal", ";9;21;", []int{9, 21}},
		{"too large", ";65536;210;", []int{0, 210}},
	}

	d := Dispatcher{}
	for _, v := range tc {
		d.clear(clear{})
		for _, ch := range v.params {
			d.newParamChar(&param{action{ch, true}})
		}

		if len(v.want) != d.getParamCount() {
			t.Errorf("%s expect %d result, got %d result.\n", v.name, len(v.want), d.getParamCount())
		} else {
			for i, w := range v.want {
				got := d.getParam(i, 0)
				if got != w {
					t.Errorf("%s:\t %q [%02d parameter]: expect %d, got %d\n", v.name, v.params, i, w, got)
				}

			}
		}

	}
}
