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
	"errors"
	"os"
	"strings"
	"testing"
)

func TestDisplay(t *testing.T) {
	tc := []struct {
		label    string
		useEnv   bool
		termEnv  string
		err      error
		hasECH   bool
		hasBCE   bool
		hasTitle bool
	}{
		{"useEnvironment, base TERM", true, "alacritty", nil, true, true, false},
		{"useEnvironment, base TERM, title support", true, "xterm", nil, true, true, true},
		{"useEnvironment, dynamic TERM", true, "sun", nil, true, true, false}, // we choose sun, because sun fade out from the market
		{"useEnvironment, wrong TERM", true, "stranger", errors.New("infocmp: couldn't open terminfo file"), false, false, false},
		{"not useEnvironment ", false, "anything", nil, true, true, true},
	}

	for _, v := range tc {
		os.Setenv("TERM", v.termEnv)
		d, e := NewDisplay(v.useEnv)

		if e == nil {

			if d.hasBCE != v.hasBCE {
				t.Errorf("%q expect bce %t, got %t\n", v.label, v.hasBCE, d.hasBCE)
			}
			if d.hasECH != v.hasECH {
				t.Errorf("%q expect ech %t, got %t\n", v.label, v.hasECH, d.hasECH)
			}
			if d.hasTitle != v.hasTitle {
				t.Errorf("%q expect title %t, got %t\n", v.label, v.hasTitle, d.hasTitle)
			}
		} else {
			if !strings.HasPrefix(e.Error(), v.err.Error()) {
				t.Errorf("%q expect err %q, got %q\n", v.label, v.err, e)
			}
		}
	}
}

func TestOpenClose(t *testing.T) {
	os.Setenv("TERM", "xterm-256color")
	d, _ := NewDisplay(true)

	expect := "\x1b[?1049h\x1b[22;0;0t\x1b[?1h"
	got := d.open()
	if got != expect {
		t.Errorf("#test open() expect %q, got %q\n", expect, got)
	}

	expect = "\x1b[?1l\x1b[0m\x1b[?25h\x1b[?1003l\x1b[?1002l\x1b[?1001l\x1b[?1000l\x1b[?1015l\x1b[?1006l\x1b[?1005l\x1b[?1049l\x1b[23;0;0t"
	got = d.close()
	if got != expect {
		t.Errorf("#test close() expect %q, got %q\n", expect, got)
	}
}

func TestNewFrame(t *testing.T) {
	tc := []struct {
		label       string
		bgRune      rune
		mix         string
		initialized bool
		expectSeq   string
		row         int
		expectRow   string
	}{
		{
			"empty screen update one wrap line", 'N', "\x1B[11;74Houtput for normal warp line.", true,
			"\x1b[?25l\x1b[11;74Houtput for normal warp line.\x1b[?25h", 11,
			"[ 11] for.normal.warp.line............................................................",
		},
		{
			"same screen update one wrap line", 'X', "\x1B[24;74Houtput for normal warp line.", true,
			"\x1b[?25l\x1b[24;74Houtput for normal warp line.\x1b[?25h", 24,
			"[ 24] for.normal.warp.line.XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX",
		},
	}
	oldE := NewEmulator3(80, 40, 40)
	newE := NewEmulator3(80, 40, 40)

	os.Setenv("TERM", "xterm-256color")
	d, e := NewDisplay(true)
	if e != nil {
		t.Errorf("#test NewFrame() create display error: %s\n", e)
	}

	for _, v := range tc {
		// 'N' means don't fill the screen
		if v.bgRune != 'N' {
			oldE.cf.fillCells(v.bgRune, oldE.attrs)
			newE.cf.fillCells(v.bgRune, newE.attrs)
		}

		// make difference between terminal states
		newE.HandleStream(v.mix)

		// check the difference sequence
		gotSeq := d.NewFrame(v.initialized, oldE, newE)
		if gotSeq != v.expectSeq {
			t.Errorf("%q expect %q, got %q\n", v.label, v.expectSeq, gotSeq)
		}

		// apply difference sequence to target
		oldE.HandleStream(gotSeq)
		gotRow := printCells(oldE.cf, v.row)

		// check the replicate result.
		if !strings.Contains(gotRow, v.expectRow) {
			t.Errorf("%q expect \n%s, got \n%s\n", v.label, v.expectRow, gotRow)
		}
	}
}
