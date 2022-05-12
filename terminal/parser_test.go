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

func TestParseProcessInput(t *testing.T) {
	tc := []struct {
		name  string
		raw   string
		hName string
	}{
		{"OSC 0;Pt BEL ", "\x1B]0;ada\x07", "osc 0,1,2"},
		{"OSC 1;Pt 7bit ST ", "\x1B]1;ada\x1B\\", "osc 0,1,2"},
		{"OSC 2;Pt BEL chinese", "\x1B]2;a道德经a\x07", "osc 0,1,2"},
		{"CSI Ps;PsH", "\x1B[24;14H", "cup"},
		{"CSI Ps;Psf", "\x1B[41;42f", "cup"},
		{"CSI Ps A", "\x1B[41A", "cuu"},
		{"CSI Ps B", "\x1B[31B", "cud"},
		{"CSI Ps C", "\x1B[21C", "cuf"},
		{"CSI Ps D", "\x1B[11D", "cub"},
		{"CR", "\x0D", "c0-cr"},
		{"LF", "\n", "c0-lf"},
		{"VT", "\v", "c0-lf"},
		{"FF", "\f", "c0-lf"},
	}

	p := NewParser()
	var hd *Handler
	for _, v := range tc {
		for _, ch := range v.raw {
			hd = p.processInput(ch)
		}
		if hd != nil && hd.name == v.hName {
			// ac.handle(&clear{})
			continue
		} else {
			if hd != nil {
				if hd.name != v.hName {
					t.Errorf("%s:\t raw=%q, expect %s, got %s, ch=%q\n", v.name, v.raw, v.hName, hd.name, hd.ch)
				}
			} else {
				t.Errorf("%s;\t raw=%q, result should not be nil.", v.name, v.raw)
			}
		}

	}
}
