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

func TestNewFrame_PutRow(t *testing.T) {
	tc := []struct {
		label       string
		bgRune1     rune
		bgRune2     rune
		mix         string
		initialized bool
		expectSeq   string
		row         int
		expectRow   string
	}{
		{
			"empty screen update one wrap line", ' ', ' ', "\x1B[11;74Houtput for normal wrap line.", true,
			"\x1b[?25l\x1b[11;74Houtput for\x1b[12;5Hnormal\x1b[12;12Hwrap\x1b[12;17Hline.\x1b[?25h", 11,
			"[ 11] for.normal.wrap.line............................................................",
		},
		{
			"same screen update one wrap line", 'X', 'X', "\x1B[24;74Houtput for normal wrap line.", true,
			"\x1b[?25l\x1b[24;74Houtput for normal wrap line.\x1b[?25h", 24,
			"[ 24] for.normal.wrap.line.XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX",
		},
		{
			"new screen with empty line", 'U', 'U', "\x1B[4;4HErase to the end of line\x1B[0K.", true,
			"\x1b[?25l\x1b[4;4HErase to the end of line.\x1b[K\x1b[?25h", 3,
			"[  3] UUUErase.to.the.end.of.line.....................................................",
		},
		{
			"new screen with big space gap", 'V', 'V',
			"\x1B[5;1H1st space\x1B[0K\x1b[5;21H2nd!   \x1B[1;37;40m   3rd\x1b[5;79HEOL", true,
			"\x1b[?25l\r\n1st space\x1b[11X\x1b[5;21H2nd!   \x1b[0;1;37;40m   3rd\x1b[45X\x1b[5;79H\x1b[0;1;37;40mE\x1b[5;80HOL\x1b[?25h", 4,
			"[  4] 1st.space...........2nd!......3rd.............................................EO",
		},
		{
			"last cell", 'W', 'W', "\x1B[6;77HLAST", true,
			"\x1b[?25l\x1b[6;77HLAST\r\n\x1b[6;80H\x1b[?25h", 5,
			"[  5] WWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWLAST",
		},
		{
			"last chinese cell", ' ', ' ', "\x1B[7;7H左边\x1B[7;77H中文", true,
			"\x1b[?25l\x1b[7;7H左边\x1b[7;77H中文\r\n\x1b[7;80H\x1b[?25h", 6,
			"[  6] ......左边..................................................................中文",
		},
		{
			"last chinese cell early wrap", ' ', ' ', "\x1B[8;7H提早\x1B[8;78H换行", true,
			"\x1b[?25l\x1b[8;7H提早\x1b[8;78H换\r\n行\x1b[?25h", 7,
			"[  7] ......提早...................................................................换.",
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
		oldE.cf.fillCells(v.bgRune1, oldE.attrs)
		newE.cf.fillCells(v.bgRune2, newE.attrs)

		// make difference between terminal states
		// fmt.Printf("#test NewFrame() newE cursor at (%2d,%2d)\n", newE.GetCursorRow(), newE.GetCursorCol())
		newE.HandleStream(v.mix)

		// check the difference sequence
		gotSeq := d.NewFrame(v.initialized, oldE, newE)
		if gotSeq != v.expectSeq {
			t.Errorf("%q expect \n%q, got \n%q\n", v.label, v.expectSeq, gotSeq)
		}

		// apply difference sequence to target
		// fmt.Printf("#test NewFrame() oldE cursor at (%2d,%2d)\n", oldE.GetCursorRow(), oldE.GetCursorCol())
		oldE.HandleStream(gotSeq)
		gotRow := printCells(oldE.cf, v.row)

		// check the replicate result.
		if !strings.Contains(gotRow, v.expectRow) {
			t.Errorf("%q expect \n%s, got \n%s\n", v.label, v.expectRow, gotRow)
		}
	}
}
