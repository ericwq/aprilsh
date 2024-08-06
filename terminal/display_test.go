// Copyright 2022 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package terminal

import (
	"encoding/base64"
	"fmt"
	"io"
	"log/slog"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/ericwq/aprilsh/encrypt"
	"github.com/ericwq/aprilsh/util"
)

func TestDisplay(t *testing.T) {
	tc := []struct {
		label        string
		err          string
		termEnv      string
		useEnv       bool
		hasECH       bool
		hasBCE       bool
		supportTitle bool
	}{
		{
			"useEnvironment, base TERM", "", "alacritty",
			true, true, true, true,
		},
		{
			"useEnvironment, base TERM, title support", "", "xterm",
			true, true, true, true,
		},
		{
			"useEnvironment, dynamic TERM", "terminal entry not found", "sun",
			true, true, true, false,
		}, // we choose sun, because sun fade out from the market
		{
			"useEnvironment, wrong TERM", "infocmp: couldn't open terminfo file", "stranger",
			true, true, true, false,
		},
		{
			"not useEnvironment ", "", "anything",
			false, true, true, true,
		},
	}

	for _, v := range tc {
		t.Run(v.label, func(t *testing.T) {
			os.Setenv("TERM", v.termEnv)
			d, e := NewDisplay(v.useEnv)

			if e == nil {

				if d.hasBCE != v.hasBCE {
					t.Errorf("%q expect bce %t, got %t\n", v.label, v.hasBCE, d.hasBCE)
				}
				if d.hasECH != v.hasECH {
					t.Errorf("%q expect ech %t, got %t\n", v.label, v.hasECH, d.hasECH)
				}
				if d.supportTitle != v.supportTitle {
					t.Errorf("%q expect title %t, got %t\n", v.label, v.supportTitle, d.supportTitle)
				}
			} else {
				// fmt.Printf("#test NewDisplay() %q return %q ,expect %q\n", v.label, e, v.err)
				if !strings.HasPrefix(e.Error(), v.err) {
					t.Errorf("%q expect err %q, got %q\n", v.label, v.err, e)
				}
			}
		})
	}
}

func TestOpenClose(t *testing.T) {
	os.Setenv("TERM", "xterm-256color")
	d, _ := NewDisplay(true)

	expect := "\x1b[?1049h\x1b[22;0;0t\x1b[?1h"
	got := d.Open()
	if got != expect {
		t.Errorf("#test open() expect %q, got %q\n", expect, got)
	}

	expect = "\x1b[?1l\x1b[0m\x1b[?25h\x1b[?1003l\x1b[?1002l\x1b[?1001l\x1b[?1000l\x1b[?1015l\x1b[?1006l\x1b[?1005l\x1b[?1049l\x1b[23;0;0t"
	got = d.Close()
	if got != expect {
		t.Errorf("#test close() expect %q, got %q\n", expect, got)
	}
}

func TestNewFrame_PutRow(t *testing.T) {
	tc := []struct {
		label       string
		mix         string
		expectSeq   string
		expectRow   string
		row         int
		bgRune1     rune
		bgRune2     rune
		initialized bool
	}{
		{
			"empty screen update one wrap line", "\x1B[11;74Houtput for normal wrap line.",
			"\x1b[K\x1b[?25l\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[73X\x1b[73Coutput for normal wrap line.\x1b[K\x1b[?25h",
			"[ 11] for.normal.wrap.line............................................................",
			11, ' ', ' ', true,
		},
		{
			"same screen update one wrap line", "\x1B[24;74Houtput for normal wrap line.",
			"XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX\r\nXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX\r\nXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX\r\nXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX\r\nXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX\r\nXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX\r\nXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX\r\nXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX\r\nXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX\r\nXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX\r\nXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX\r\nXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX\r\nXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX\r\nXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX\r\nXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX\r\nXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX\r\nXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX\r\nXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX\r\nXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX\r\nXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX\r\nXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX\r\nXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX\r\nXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX\r\nXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXoutput for normal wrap line.XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX\r\n\x1b[25;22H",
			"[ 24] for.normal.wrap.line.XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX",
			24, 'X', 'X', true,
		},
		{
			"new screen with empty line", "\x1B[4;4HErase to the end of line\x1B[0K.",
			"UUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUU\r\nUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUU\r\nUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUU\r\nUUUErase to the end of line.\x1b[K",
			"[  3] UUUErase.to.the.end.of.line.....................................................",
			3, 'U', 'U', true,
		},
		{
			"new screen with big space gap",
			"\x1B[5;1H1st space\x1B[0K\x1b[5;21H2nd!   \x1B[1;37;40m   3rd\x1b[5;79HEOL  \x1b[0m",
			"\x1b[K\x1b[?25l\n\x1b[K\n\x1b[K\n\x1b[K\n1st space\x1b[11X\x1b[11C2nd!   \x1b[0;1;37;40m   3rd\x1b[0m\x1b[45X\x1b[45C\x1b[0;1;37;40mE\x1b[5;80HOL  \x1b[0m\x1b[K\x1b[?25h",
			"[  4] 1st.space...........2nd!......3rd.............................................EO",
			4, ' ', ' ', true,
		},
		{
			"last cell", "\x1B[6;77HLAST",
			"WWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWW\r\nWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWW\r\nWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWW\r\nWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWW\r\nWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWW\r\nWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWLAST\r\n\x1b[6;80H",
			"[  5] WWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWLAST",
			5, 'W', 'W', true,
		},
		{
			"last chinese cell", "\x1B[7;7H左边\x1B[7;77H中文",
			"\x1b[K\x1b[?25l\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[6X\x1b[6C左边\x1b[66X\x1b[66C中文\r\n\x1b[7;80H\x1b[?25h",
			"[  6] ......左边..................................................................中文",
			6, ' ', ' ', true,
		},
		{
			"last chinese cell early wrap", "\x1B[8;7H提早\x1B[8;78H换行",
			"\x1b[K\x1b[?25l\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[6X\x1b[6C提早\x1b[67X\x1b[67C换\r\n行\x1b[K\x1b[?25h",
			"[  7] ......提早...................................................................换.",
			7, ' ', ' ', true,
		},
		{
			"backspace case", "\x1b[9;1Hbackspace case\x1b[9;11H",
			"\x1b[K\x1b[?25l\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\nbackspace case\x1b[K\b\b\b\b\x1b[?25h",
			"[  8] backspace.case..................................................................",
			8, ' ', ' ', true,
		},
		{
			"mix color case", "\x1b[10;1H\x1b[1;34mdevelop\x1b[m  \x1b[1;34mproj     \x1b[m",
			"\x1b[K\x1b[?25l\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[0;1;34mdevelop\x1b[0m  \x1b[0;1;34mproj\x1b[5X\x1b[5C\x1b[0m\x1b[K\x1b[?25h",
			"[  9] develop..proj...................................................................",
			9, ' ', ' ', true,
		},
		{
			"mix color, false initialized case",
			"\x1b[10;1H\x1b[1;34mdevelop\x1b[m  \x1b[1;35mproj\x1b[m",
			"\x1b[?5l\x1b[r\x1b[0m\x1b[H\x1b[2J\x1b[?25l\x1b[?1047l\x1b[r\x1b[?69l\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[0;1;34mdevelop\x1b[0m  \x1b[0;1;35mproj\x1b[0m\x1b[K\x1b[0C\x1b[?25h\x1b[1 q\x1b]112\a\x1b[0m\x1b[?2004l\x1b[?1003l\x1b[?1002l\x1b[?1001l\x1b[?1000l\x1b[?1004l\x1b[?1015l\x1b[?1006l\x1b[?1005l\x1b[?7h\x1b[20l\x1b[2l\x1b[4l\x1b[12h\x1b[?67l\x1b[?1036h\x1b[?1007l\x1b[?1l\x1b[?6l\x1b>\x1b[?3l\x1b[3g\x1b[64\"p\x1b[>4;1m",
			"[  9] develop..proj...................................................................",
			9, ' ', ' ', false,
		},
	}

	util.Logger.CreateLogger(io.Discard, true, slog.LevelDebug)

	os.Setenv("TERM", "xterm-256color")
	d, e := NewDisplay(true)
	if e != nil {
		t.Errorf("#test NewFrame() create display error: %s\n", e)
	}

	for _, v := range tc {
		t.Run(v.label, func(t *testing.T) {
			oldE := NewEmulator3(80, 40, 40)
			newE := NewEmulator3(80, 40, 40)
			// oldE.resetAttrs()
			// newE.resetAttrs()
			oldE.cf.fillCells(v.bgRune1, oldE.attrs)
			newE.cf.fillCells(v.bgRune2, newE.attrs)

			// use mix to create difference in newE
			// fmt.Printf("#test NewFrame() newE cursor at (%2d,%2d)\n", newE.GetCursorRow(), newE.GetCursorCol())
			newE.HandleStream(v.mix)

			// calculate the difference sequence
			diff := d.NewFrame(v.initialized, oldE, newE)
			if diff != v.expectSeq {
				t.Errorf("%q expect \n%q, got \n%q\n", v.label, v.expectSeq, diff)
			}

			// apply difference sequence to oldE
			// fmt.Printf("#test NewFrame() oldE cursor at (%2d,%2d)\n", oldE.GetCursorRow(), oldE.GetCursorCol())
			oldE.HandleStream(diff)
			gotRow := printCells(oldE.cf, v.row)

			// check the replicate result.
			skipHeader := 80 + 7 // rule row + header
			if !strings.Contains(gotRow, v.expectRow) {
				for i := range v.expectRow {
					if v.expectRow[i] != gotRow[skipHeader+i] {
						t.Logf("%q col=%d expect=%q, got=%q\n", v.label, i-6, v.expectRow[i], gotRow[skipHeader+i])
					}
				}
				t.Errorf("%q expect \n%s, got \n%s\n", v.label, v.expectRow, gotRow)
			}
		})
	}
}

func TestNewFrame_ScrollUp(t *testing.T) {
	tc := []struct {
		label       string
		mixSeq      string
		extraSeq    string
		scrollSeq   string
		expectSeq   string
		bgRune1     rune
		bgRune2     rune
		initialized bool
	}{
		{
			"scroll up 5 lines", "\x1B[5;1Hscroll\r\ndown\r\nmore\r\nthan\r\n5 lines!",
			"\r\ndifferent line", "\x1B[4S",
			"\x1b[?25l\r5 lines!\x1b[K\r\ndifferent line\x1b[K\r\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\x1b[10;15H\x1b[?25h",
			' ', ' ', true,
		},
		{
			"scroll up 6 lines", "\x1B[35;1Hscroll\r\ndown\r\nmore\r\nthan\r\n6\r\nlines!",
			"", "\x1B[34S",
			// "\x1b[0m\r\x1b[34S\x1b[40;7H",
			"\x1b[?25l\rlines!\x1b[K\r\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\x1b[40;7H\x1b[?25h",
			' ', ' ', true,
		},
	}

	oldE := NewEmulator3(80, 40, 40)
	newE := NewEmulator3(80, 40, 40)

	// util.Logger.CreateLogger(io.Discard, true, slog.LevelDebug)

	os.Setenv("TERM", "xterm-256color")
	d, e := NewDisplay(true)
	if e != nil {
		t.Errorf("#test NewFrame() create display error: %s\n", e)
	}

	for _, v := range tc {
		// reset the terminal to avoid overlap
		oldE.resetTerminal()
		newE.resetTerminal()

		// prepare the screen background
		oldE.cf.fillCells(v.bgRune1, oldE.attrs)
		newE.cf.fillCells(v.bgRune2, newE.attrs)

		// make scroll difference between terminal states
		newE.HandleStream(v.mixSeq + v.extraSeq + v.scrollSeq)
		oldE.HandleStream(v.mixSeq)
		// fmt.Printf("NEW:\n%s", printCells(newE.cf))
		// fmt.Printf("OLD:\n%s", printCells(oldE.cf))

		// check the expect difference sequence
		gotSeq := d.NewFrame(v.initialized, oldE, newE)
		if gotSeq != v.expectSeq {
			t.Errorf("%q expect \n%q, got \n%q\n", v.label, v.expectSeq, gotSeq)
		}
		// fmt.Printf("gotSeq=%q\n", gotSeq)
		// fmt.Printf("OLD:\n%s", printCells(oldE.cf))

		// apply difference sequence to target
		oldE.HandleStream(gotSeq)
		// fmt.Printf("new scrollHead=%d, old scrollHead=%d\n", newE.cf.scrollHead, oldE.cf.scrollHead)
		// fmt.Printf("NEW:\n%s", printCells(newE.cf))
		// fmt.Printf("OLD:\n%s", printCells(oldE.cf))

		// compare the first row to validate the scroll
		newRow := getRow(newE, 0)
		oldRow := getRow(oldE, 0)
		if !reflect.DeepEqual(newRow, oldRow) {
			t.Errorf("%q expect \n%q\n, got \n%q\n", v.label, newRow[:], oldRow[:])
		}

		// fmt.Printf("OLD with diff:\n%s", printCells(oldE.cf))
	}
}

func TestNewFrame_Bell(t *testing.T) {
	tc := []struct {
		label       string
		expectSeq   string
		initialized bool
		bell        bool
	}{
		{"no bell", "", true, false},
		{"has bell", "\a", true, true},
	}
	oldE := NewEmulator3(80, 40, 40)
	newE := NewEmulator3(80, 40, 40)

	util.Logger.CreateLogger(io.Discard, true, slog.LevelDebug)

	os.Setenv("TERM", "xterm-256color")
	d, e := NewDisplay(true)
	if e != nil {
		t.Errorf("#test NewFrame() create display error: %s\n", e)
	}

	for _, v := range tc {
		// reset the terminal to avoid overlap
		oldE.resetTerminal()
		newE.resetTerminal()

		if v.bell {
			newE.ringBell()
		}

		// check the expect difference sequence
		gotSeq := d.NewFrame(v.initialized, oldE, newE)
		if gotSeq != v.expectSeq {
			t.Errorf("%q expect \n%q, got \n%q\n", v.label, v.expectSeq, gotSeq)
		}
	}
}

func TestNewFrame_CursorStyle(t *testing.T) {
	tc := []struct {
		label     string
		expectSeq string
		showStyle CursorStyle
	}{
		{"same blink block", "", CursorStyle_BlinkBlock},
		{"steady block", "\x1B[2 q", CursorStyle_SteadyBlock},
		{"blink underline", "\x1B[3 q", CursorStyle_BlinkUnderline},
		{"steady underline", "\x1B[4 q", CursorStyle_SteadyUnderline},
		{"blink bar", "\x1B[5 q", CursorStyle_BlinkBar},
		{"steady bar", "\x1B[6 q", CursorStyle_SteadyBar},
	}
	oldE := NewEmulator3(80, 40, 40)
	newE := NewEmulator3(80, 40, 40)

	// util.Logger.CreateLogger(io.Discard, true, slog.LevelDebug)

	os.Setenv("TERM", "xterm-256color")
	d, e := NewDisplay(true)
	if e != nil {
		t.Errorf("#test NewFrame() create display error: %s\n", e)
	}

	for _, v := range tc {
		// reset the terminal to avoid overlap
		oldE.resetTerminal()
		newE.resetTerminal()

		// new cursor show style
		newE.cf.cursor.showStyle = v.showStyle

		// check the expect difference sequence
		gotSeq := d.NewFrame(true, oldE, newE)
		if gotSeq != v.expectSeq {
			t.Errorf("%q expect \n%q, got \n%q\n", v.label, v.expectSeq, gotSeq)
		}
	}
}

func TestNewFrame_WindowTitleIconName(t *testing.T) {
	tc := []struct {
		label       string
		windowTitle string
		iconName    string
		expectSeq   string
		initialized bool
	}{
		{"no window title and icon name", "", "", "", true},
		{"has window title", "window title", "", "\x1b]2;window title\a", true},
		{"has chinese icon name", "", "图标名称", "\x1b]1;图标名称\a", true},
		{"has same window title & icon name", "中文标题", "中文标题", "\x1b]0;中文标题\a", true},
	}
	oldE := NewEmulator3(80, 40, 40)
	newE := NewEmulator3(80, 40, 40)

	util.Logger.CreateLogger(io.Discard, true, slog.LevelDebug)

	os.Setenv("TERM", "xterm-256color")
	d, e := NewDisplay(true)
	if e != nil {
		t.Errorf("#test NewFrame() create display error: %s\n", e)
	}

	for _, v := range tc {
		// reset the terminal to avoid overlap
		oldE.resetTerminal()
		newE.resetTerminal()

		if v.windowTitle != "" {
			newE.setWindowTitle(v.windowTitle)
			newE.setTitleInitialized()
		}

		if v.iconName != "" {
			newE.setIconLabel(v.iconName)
			newE.setTitleInitialized()
		}

		// check the expect difference sequence
		gotSeq := d.NewFrame(v.initialized, oldE, newE)
		if gotSeq != v.expectSeq {
			t.Errorf("%q expect \n%q, got \n%q\n", v.label, v.expectSeq, gotSeq)
		}
	}
}

func TestNewFrame_TitleStack(t *testing.T) {
	tc := []struct {
		label       string
		expectSeq   string
		newStack    []string
		oldStack    []string
		initialized bool
	}{
		{
			"no stack", "",
			[]string{},
			[]string{},
			true,
		},
		{
			"new stack = old stack", "",
			[]string{"a1", "a2"},
			[]string{"a1", "a2"},
			true,
		},
		{
			"new stack > old stack", "\x1b]2;c\a\x1b[22;0t",
			[]string{"a", "b", "c"},
			[]string{"a", "b"},
			true,
		},
		{
			"new stack < old stack", "\x1b[23;0t\x1b]2;t2\a",
			[]string{"t1", "t2"},
			[]string{"t1", "t2", "t3"},
			true,
		},
		{
			"max stack with diff", "\x1b]2;w9\a\x1b[22;0t",
			[]string{"w1", "w2", "w3", "w4", "w5", "w6", "w7", "w8", "w9"},
			[]string{"w0", "w1", "w2", "w3", "w4", "w5", "w6", "w7", "w8"},
			true,
		},
	}

	oldE := NewEmulator3(80, 40, 40)
	newE := NewEmulator3(80, 40, 40)

	util.Logger.CreateLogger(io.Discard, true, slog.LevelDebug)

	os.Setenv("TERM", "xterm-256color")
	d, e := NewDisplay(true)
	if e != nil {
		t.Errorf("#test NewFrame() create display error: %s\n", e)
	}

	for _, v := range tc {
		oldE.resetTerminal()
		newE.resetTerminal()
		// reset the terminal to avoid overlap
		t.Run(v.label, func(t *testing.T) {
			// prepare new stack
			for i := range v.newStack {
				newE.setTitleInitialized()
				newE.setWindowTitle(v.newStack[i])
				newE.saveWindowTitleOnStack()
			}
			// prepare old stack
			for i := range v.oldStack {
				oldE.setTitleInitialized()
				oldE.setWindowTitle(v.oldStack[i])
				oldE.saveWindowTitleOnStack()
			}

			// check the expect difference sequence
			gotSeq := d.NewFrame(v.initialized, oldE, newE)
			if gotSeq != v.expectSeq {
				t.Errorf("%q expect \n%q, got \n%q\n", v.label, v.expectSeq, gotSeq)
			}
		})
	}
}

func TestNewFrame_ReverseVideo(t *testing.T) {
	tc := []struct {
		label        string
		seq          string
		expectSeq    string
		initialized  bool
		reverseVideo bool // determine the reverseVideo value of pair terminal
	}{
		{"has reverse video", "\x1B[?5h", "\x1b[?5h", true, true},
		{"no reverse video", "\x1B[?5h", "\x1B[?5l", true, false},
	}
	oldE := NewEmulator3(80, 40, 40)
	newE := NewEmulator3(80, 40, 40)

	os.Setenv("TERM", "xterm-256color")
	d, e := NewDisplay(true)
	if e != nil {
		t.Errorf("#test NewFrame() create display error: %s\n", e)
	}

	for _, v := range tc {
		// reset the terminal to avoid overlap
		oldE.resetTerminal()
		newE.resetTerminal()

		if v.reverseVideo {
			// reverseVideo: newE true, oldE false
			newE.HandleStream(v.seq)
		} else {
			// reverseVideo: newE false, oldE true
			oldE.HandleStream(v.seq)
		}

		// check the expect difference sequence
		gotSeq := d.NewFrame(v.initialized, oldE, newE)
		if gotSeq != v.expectSeq {
			t.Errorf("%q expect \n%q, got \n%q\n", v.label, v.expectSeq, gotSeq)
		}
	}
}

func TestNewFrame_Resize(t *testing.T) {
	tc := []struct {
		label         string
		initialized   bool // for resize, it's always set to false internally
		width, height int
	}{
		{"extend width and height", true, 90, 50},
		{"shrink both width and height", false, 70, 30},
	}
	os.Setenv("TERM", "xterm-256color")
	d, e := NewDisplay(true)
	if e != nil {
		t.Errorf("#test NewFrame() create display error: %s\n", e)
	}

	for _, v := range tc {
		oldE := NewEmulator3(80, 40, 40)
		newE := NewEmulator3(80, 40, 40)

		newE.resize(v.width, v.height)

		// fmt.Printf("OLD: w=%d, h=%d\n%s", oldE.GetWidth(), oldE.GetHeight(), printCells(oldE.cf))
		// fmt.Printf("NEW: w=%d, h=%d\n%s", newE.GetWidth(), newE.GetHeight(), printCells(newE.cf))

		// resize result in initialize, we can't predict the got sequence on different platform.
		gotSeq := d.NewFrame(v.initialized, oldE, newE)
		if len(gotSeq) < 100 {
			t.Errorf("%q , the diff seq should be greater than 100, got %d\n%q\n", v.label, len(gotSeq), gotSeq)
		}
	}
}

func TestNewFrame_AltScreenBufferMode(t *testing.T) {
	tc := []struct {
		label               string
		seq                 string
		expectSeq           string
		initialized         bool
		altScreenBufferMode bool
	}{
		{"already initialized, has altScreenBufferMode", "\x1B[?1047h", "\x1B[?1047h", true, true},
		{"already initialized, no altScreenBufferMode", "\x1B[?1047h", "\x1B[?1047l", true, false},
	}
	oldE := NewEmulator3(8, 4, 4)
	newE := NewEmulator3(8, 4, 4)

	os.Setenv("TERM", "xterm-256color")
	d, e := NewDisplay(true)
	if e != nil {
		t.Errorf("#test create display error: %s\n", e)
	}

	for _, v := range tc {
		// reset the terminal to avoid overlap
		oldE.resetTerminal()
		newE.resetTerminal()

		if v.altScreenBufferMode {
			// altScreenBufferMode: newE true, oldE false
			newE.HandleStream(v.seq)
		} else {
			// altScreenBufferMode: newE false, oldE true
			oldE.HandleStream(v.seq)
		}

		// check the expect difference sequence
		gotSeq := d.NewFrame(v.initialized, oldE, newE)
		if !strings.Contains(gotSeq, v.expectSeq) {
			t.Errorf("%q expect \n%q\n contains %q\n", v.label, gotSeq, v.expectSeq)
		}
	}
}

func TestNewFrame_Margin(t *testing.T) {
	tc := []struct {
		label       string
		seq         string
		expectSeq   string
		initialized bool
		margin      bool
	}{
		{
			"already initialized, new has margin", "\x1B[2;6r",
			"\x1b[2;6r\x1b[K\x1b[?25l\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\x1b[1;1H\x1b[?25h",
			true, true,
		},
		{
			"already initialized, old has margin", "\x1B[2;6r",
			"\x1b[r\x1b[K\x1b[?25l\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\x1b[1;1H\x1b[?25h",
			true, false,
		},
	}
	oldE := NewEmulator3(8, 8, 4)
	newE := NewEmulator3(8, 8, 4)

	os.Setenv("TERM", "xterm-256color")
	d, e := NewDisplay(true)
	if e != nil {
		t.Errorf("#test create display error: %s\n", e)
	}

	for _, v := range tc {
		// reset the terminal to avoid overlap
		oldE.resetTerminal()
		newE.resetTerminal()

		if v.margin {
			// margin: newE true, oldE false
			newE.HandleStream(v.seq)
		} else {
			// margin: newE false, oldE true
			oldE.HandleStream(v.seq)
		}

		// check the expect difference sequence
		gotSeq := d.NewFrame(v.initialized, oldE, newE)
		if gotSeq != v.expectSeq {
			t.Errorf("%q expect \n%q, got \n%q\n", v.label, v.expectSeq, gotSeq)
		}
	}
}

func TestNewFrame_HMargin(t *testing.T) {
	tc := []struct {
		label       string
		seq         string
		expectSeq   string
		initialized bool
		margin      bool
	}{
		{"already initialized, new has margin", "\x1B[?69h\x1B[2;6s", "\x1b[?69h\x1b[2;6s", true, true},
		{"already initialized, old has margin", "\x1B[?69h\x1B[2;6s", "\x1b[?69l", true, false},
		{"already initialized, both no margin", "", "", true, false},
	}
	oldE := NewEmulator3(8, 8, 4)
	newE := NewEmulator3(8, 8, 4)

	os.Setenv("TERM", "xterm-256color")
	d, e := NewDisplay(true)
	if e != nil {
		t.Errorf("#test create display error: %s\n", e)
	}

	for _, v := range tc {
		// reset the terminal to avoid overlap
		oldE.resetTerminal()
		newE.resetTerminal()

		if v.margin {
			// margin: newE true, oldE false
			newE.HandleStream(v.seq)
		} else {
			// margin: newE false, oldE true
			oldE.HandleStream(v.seq)
		}

		// check the expect difference sequence
		gotSeq := d.NewFrame(v.initialized, oldE, newE)
		if gotSeq != v.expectSeq {
			t.Errorf("%q expect \n%q, got \n%q\n", v.label, v.expectSeq, gotSeq)
		}
	}
}

func TestNewFrame_Decsc(t *testing.T) {
	tc := []struct {
		label       string
		seq         string
		expectSeq   string
		initialized bool
		decsc       bool
	}{
		{"already initialized, new has decsc", "\x1B7", "\x1b[?1048h", true, true},
		{"already initialized, old has decsc", "\x1B7", "\x1b[?1048l", true, false},
		{"already initialized, both no decsc", "", "", true, false},
	}
	oldE := NewEmulator3(8, 8, 4)
	newE := NewEmulator3(8, 8, 4)

	os.Setenv("TERM", "xterm-256color")
	d, e := NewDisplay(true)
	if e != nil {
		t.Errorf("#test create display error: %s\n", e)
	}

	for _, v := range tc {
		// reset the terminal to avoid overlap
		oldE.resetTerminal()
		newE.resetTerminal()

		if v.decsc {
			// decsc: newE true, oldE false
			newE.HandleStream(v.seq)
		} else {
			// decsc: newE false, oldE true
			oldE.HandleStream(v.seq)
		}

		// check the expect difference sequence
		gotSeq := d.NewFrame(v.initialized, oldE, newE)
		if gotSeq != v.expectSeq {
			t.Errorf("%q expect \n%q, got \n%q\n", v.label, v.expectSeq, gotSeq)
		}
	}
}

func TestNewFrame_Scosc(t *testing.T) {
	tc := []struct {
		label       string
		seq         string
		expectSeq   string
		initialized bool
		scosc       bool
	}{
		{"already initialized, new has scosc", "\x1B[s", "\x1b[s", true, true},
		{"already initialized, old has scosc", "\x1B[s", "\x1b[u", true, false},
		{"already initialized, both no scosc", "", "", true, false},
	}
	oldE := NewEmulator3(8, 8, 4)
	newE := NewEmulator3(8, 8, 4)

	os.Setenv("TERM", "xterm-256color")
	d, e := NewDisplay(true)
	if e != nil {
		t.Errorf("#test create display error: %s\n", e)
	}

	for _, v := range tc {
		// reset the terminal to avoid overlap
		oldE.resetTerminal()
		newE.resetTerminal()

		if v.scosc {
			// scosc: newE true, oldE false
			newE.HandleStream(v.seq)
		} else {
			// scosc: newE false, oldE true
			oldE.HandleStream(v.seq)
		}

		// check the expect difference sequence
		gotSeq := d.NewFrame(v.initialized, oldE, newE)
		if gotSeq != v.expectSeq {
			t.Errorf("%q expect \n%q, got \n%q\n", v.label, v.expectSeq, gotSeq)
		}
	}
}

func TestNewFrame_ShowCursorMode(t *testing.T) {
	tc := []struct {
		label                string
		seq                  string
		expectSeq            string
		initialized          bool
		showcursorModeForNew bool
	}{
		{"already initialized, new show no cursor", "\x1B[?25l", "\x1b[?25l", true, true},
		{"already initialized, old show no cursor", "\x1B[?25l", "\x1b[?25h", true, false},
		{"already initialized, both show cursor", "", "", true, false},
	}
	oldE := NewEmulator3(8, 8, 4)
	newE := NewEmulator3(8, 8, 4)

	os.Setenv("TERM", "xterm-256color")
	d, e := NewDisplay(true)
	if e != nil {
		t.Errorf("#test create display error: %s\n", e)
	}

	for _, v := range tc {
		// reset the terminal to avoid conflict
		oldE.resetTerminal()
		newE.resetTerminal()

		if v.showcursorModeForNew {
			// newE false, oldE true
			newE.HandleStream(v.seq)
		} else {
			// newE true, oldE false
			oldE.HandleStream(v.seq)
		}

		// check the expect difference sequence
		gotSeq := d.NewFrame(v.initialized, oldE, newE)
		if gotSeq != v.expectSeq {
			t.Errorf("%q expect \n%q, got \n%q\n", v.label, v.expectSeq, gotSeq)
		}
	}
}

func TestNewFrame_BracketedPasteMode(t *testing.T) {
	tc := []struct {
		label              string
		seq                string
		expectSeq          string
		initialized        bool
		bracketedPasteMode bool
	}{
		{"already initialized, new has bracketedPasteMode", "\x1B[?2004h", "\x1b[?2004h", true, true},
		{"already initialized, old has bracketedPasteMode", "\x1B[?2004h", "\x1b[?2004l", true, false},
		{"already initialized, both no bracketedPasteMode", "", "", true, false},
	}
	oldE := NewEmulator3(8, 8, 4)
	newE := NewEmulator3(8, 8, 4)

	os.Setenv("TERM", "xterm-256color")
	d, e := NewDisplay(true)
	if e != nil {
		t.Errorf("#test create display error: %s\n", e)
	}

	for _, v := range tc {
		// reset the terminal to avoid overlap
		oldE.resetTerminal()
		newE.resetTerminal()

		if v.bracketedPasteMode {
			// newE true, oldE false
			newE.HandleStream(v.seq)
		} else {
			// newE false, oldE true
			oldE.HandleStream(v.seq)
		}

		// check the expect difference sequence
		gotSeq := d.NewFrame(v.initialized, oldE, newE)
		if gotSeq != v.expectSeq {
			t.Errorf("%q expect \n%q, got \n%q\n", v.label, v.expectSeq, gotSeq)
		}
	}
}

func TestNewFrame_MouseTrk(t *testing.T) {
	tc := []struct {
		label     string
		diffCase  string // see the switch statement for the means
		seq       string
		expectSeq string
	}{
		{"New is diffrent mode, old is default", "new", "\x1b[?1001h", "\x1b[?1001h"},
		{"New is default, old is different mode", "old", "\x1b[?1003h", "\x1b[?1003l\x1b[?1002l\x1b[?1001l\x1b[?1000l"},
		{"both have different mode", "\x1b[?1002h", "\x1b[?1003h", "\x1b[?1003l\x1b[?1002h"},
		{"both terminal keep default value", "both", "", ""},
		{"New is diffrent encoding, old is default", "new", "\x1b[?1005h", "\x1b[?1005h"},
		{"New is default, old is different encoding", "old", "\x1b[?1006h", "\x1b[?1015l\x1b[?1006l\x1b[?1005l"},
		{"both has different encoding", "\x1b[?1006h", "\x1b[?1015h", "\x1b[?1015l\x1b[?1006h"},
	}
	oldE := NewEmulator3(8, 8, 4)
	newE := NewEmulator3(8, 8, 4)

	os.Setenv("TERM", "xterm-256color")
	d, e := NewDisplay(true)
	if e != nil {
		t.Errorf("#test create display error: %s\n", e)
	}

	for _, v := range tc {
		// reset the terminal to avoid overlap
		oldE.resetTerminal()
		newE.resetTerminal()

		switch v.diffCase {
		case "new":
			newE.HandleStream(v.seq)
		case "old":
			oldE.HandleStream(v.seq)
		case "both":
		default:
			newE.HandleStream(v.diffCase)
			oldE.HandleStream(v.seq)
		}

		// check the expect difference sequence
		gotSeq := d.NewFrame(true, oldE, newE)
		if gotSeq != v.expectSeq {
			t.Errorf("%q expect \n%q, got \n%q\n", v.label, v.expectSeq, gotSeq)
		}
	}
}

func TestNewFrame_MouseTrkFocusEventMode(t *testing.T) {
	tc := []struct {
		label          string
		seq            string
		expectSeq      string
		focusEventMode bool
	}{
		{"new has focusEventMode", "\x1B[?1004h", "\x1b[?1004h", true},
		{"old has focusEventMode", "\x1B[?1004h", "\x1b[?1004l", false},
		{"both no focusEventMode", "", "", false},
	}
	oldE := NewEmulator3(8, 8, 4)
	newE := NewEmulator3(8, 8, 4)

	os.Setenv("TERM", "xterm-256color")
	d, e := NewDisplay(true)
	if e != nil {
		t.Errorf("#test create display error: %s\n", e)
	}

	for _, v := range tc {
		// reset the terminal to avoid overlap
		oldE.resetTerminal()
		newE.resetTerminal()

		if v.focusEventMode {
			// newE true, oldE false
			newE.HandleStream(v.seq)
		} else {
			// newE false, oldE true
			oldE.HandleStream(v.seq)
		}

		// check the expect difference sequence
		gotSeq := d.NewFrame(true, oldE, newE)
		if gotSeq != v.expectSeq {
			t.Errorf("%q expect \n%q, got \n%q\n", v.label, v.expectSeq, gotSeq)
		}
	}
}

func TestNewFrame_Modes(t *testing.T) {
	tc := []struct {
		label     string
		newSeq    string
		oldSeq    string
		expectSeq string
	}{
		{"new has autoNewlineMode", "\x1B[20h", "\x1B[20l", "\x1b[20h"},
		{"old has autoNewlineMode", "\x1B[20l", "\x1B[20h", "\x1b[20l"},
		{"new has localEcho", "\x1B[12h", "\x1B[12l", "\x1b[12h"},
		{"old has localEcho", "\x1B[12l", "\x1B[12h", "\x1b[12l"},
		{"new has insertMode", "\x1B[4h", "\x1B[4l", "\x1b[4h"},
		{"old has insertMode", "\x1B[4l", "\x1B[4h", "\x1b[4l"},
		{"new has keyboardLocked", "\x1B[2h", "\x1B[2l", "\x1b[2h"},
		{"old has keyboardLocked", "\x1B[2l", "\x1B[2h", "\x1b[2l"},
		{"new has keypadMode", "\x1B=", "\x1B>", "\x1b="},
		{"old has keypadMode", "\x1B>", "\x1B=", "\x1b>"},
		{"equal mode", "", "", ""},
		{"new has altSendsEscape", "\x1B[?1036h", "\x1B[?1036l", "\x1b[?1036h"},
		{"old has altSendsEscape", "\x1B[?1036l", "\x1B[?1036h", "\x1b[?1036l"},
		{"new has altScrollMode", "\x1B[?1007h", "\x1B[?1007l", "\x1b[?1007h"},
		{"old has altScrollMode", "\x1B[?1007l", "\x1B[?1007h", "\x1b[?1007l"},
		{"new has bkspSendsDel", "\x1B[?67l", "\x1B[?67h", "\x1b[?67l"},
		{"old has bkspSendsDel", "\x1B[?67h", "\x1B[?67l", "\x1b[?67h"},
		{"new has autoWrapMode", "\x1B[?7h", "\x1B[?7l", "\x1b[?7h"},
		{"old has autoWrapMode", "\x1B[?7l", "\x1B[?7h", "\x1b[?7l"},
		{"new has originMode", "\x1B[?6h", "\x1B[?6l", "\x1b[?6h"},
		{"old has originMode", "\x1B[?6l", "\x1B[?6h", "\x1b[?6l"},
		{"new has colMode", "\x1B[?3h", "\x1B[?3l", "\x1b[?3h"},
		{"old has colMode", "\x1B[?3l", "\x1B[?3h", "\x1b[?3l"},
		{"new has cursorKeyMode", "\x1B[?1h", "\x1B[?1l", "\x1b[?1h"},
		{"old has cursorKeyMode", "\x1B[?1l", "\x1B[?1h", "\x1b[?1l"},
		{"new is VT52 compatLevel", "\x1B[?2l", "\x1B[62\"p", "\x1B[?2l"},
		{"new is VT400 compatLevel", "\x1B[64\"p", "\x1B[61\"p", "\x1B[64\"p"},
		{"new is VT100 compatLevel", "\x1B[61\"p", "\x1B[62\"p", "\x1B[61\"p"},
		{"new has modifyOtherKeys = 0", "\x1B[>4m", "\x1B[>4;1m", "\x1B[>4;0m"},
		{"new has modifyOtherKeys = 1", "\x1B[>4;1m", "\x1B[>4;2m", "\x1B[>4;1m"},
		{"new has modifyOtherKeys = 2", "\x1B[>4;2m", "\x1B[>4;1m", "\x1B[>4;2m"},
	}
	oldE := NewEmulator3(8, 8, 4)
	newE := NewEmulator3(8, 8, 4)

	util.Logger.CreateLogger(io.Discard, true, slog.LevelDebug)

	os.Setenv("TERM", "xterm-256color")
	d, e := NewDisplay(true)
	if e != nil {
		t.Errorf("#test create display error: %s\n", e)
	}

	for _, v := range tc {
		// reset the terminal to avoid overlap
		oldE.resetTerminal()
		newE.resetTerminal()

		newE.HandleStream(v.newSeq)
		oldE.HandleStream(v.oldSeq)

		// check the expect difference sequence
		gotSeq := d.NewFrame(true, oldE, newE)
		if gotSeq != v.expectSeq {
			t.Errorf("%q expect \n%q, got \n%q\n", v.label, v.expectSeq, gotSeq)
		}
	}
}

func TestNewFrame_TabStops(t *testing.T) {
	tc := []struct {
		label     string
		newSeq    string
		oldSeq    string
		expectSeq string
	}{
		{
			"new has 3 tab stops",
			"\x1B[1;7H\x1BH\x1B[1;17H\x1BH\x1B[1;27H\x1BH\x1B[8;8H",
			"\x1B[8;8H",
			"\x1b[1;7H\x1bH\x1b[10C\x1bH\x1b[10C\x1bH\x1b[8;8H",
		},
		{
			"old has 3 tab stops",
			"\x1B[1;1H",
			"\x1B[1;7H\x1BH\x1B[1;17H\x1BH\x1B[1;27H\x1BH\x1B[1;1H",
			"\x1b[3g",
		},
	}

	oldE := NewEmulator3(80, 8, 4)
	newE := NewEmulator3(80, 8, 4)

	util.Logger.CreateLogger(io.Discard, true, slog.LevelDebug)

	os.Setenv("TERM", "xterm-256color")
	d, e := NewDisplay(true)
	if e != nil {
		t.Errorf("#test create display error: %s\n", e)
	}

	for _, v := range tc {
		// reset the terminal to avoid overlap
		oldE.resetTerminal()
		newE.resetTerminal()

		newE.HandleStream(v.newSeq)
		oldE.HandleStream(v.oldSeq)

		// check the expect difference sequence
		gotSeq := d.NewFrame(true, oldE, newE)
		if gotSeq != v.expectSeq {
			t.Errorf("%q expect \n%q, got \n%q\n", v.label, v.expectSeq, gotSeq)
		}
	}
}

func TestNewFrame_SelectionData(t *testing.T) {
	tc := []struct {
		label     string
		newRaw    string
		oldRaw    string
		expectSeq string
	}{
		{
			"use new selection data",
			"new terminal has selection data",
			"old terminal has selection data",
			"\x1b]52;pc;bmV3IHRlcm1pbmFsIGhhcyBzZWxlY3Rpb24gZGF0YQ==\x1b\\",
		},
		{
			"clear selection data",
			"",
			"old terminal has seelction data",
			"\x1b]52;pc;\x1b\\",
		},
	}

	oldE := NewEmulator3(80, 8, 4)
	newE := NewEmulator3(80, 8, 4)

	util.Logger.CreateLogger(io.Discard, true, slog.LevelDebug)

	os.Setenv("TERM", "xterm-256color")
	d, e := NewDisplay(true)
	if e != nil {
		t.Errorf("#test create display error: %s\n", e)
	}

	for _, v := range tc {
		// reset the terminal to avoid overlap
		oldE.resetTerminal()
		newE.resetTerminal()

		newE.HandleStream(buildSelectionDataSequence(v.newRaw))
		oldE.HandleStream(buildSelectionDataSequence(v.oldRaw))

		// check the expect difference sequence
		gotSeq := d.NewFrame(true, oldE, newE)
		if gotSeq != v.expectSeq {
			t.Errorf("%q expect \n%q, got \n%q\n", v.label, v.expectSeq, gotSeq)
		}
	}
}

func TestPutRow(t *testing.T) {
	preSeq := []string{
		"nvide:0.8.9\r\n",
		"\r\n",
		"Lua, C/C++ and Golang Integrated Development Environment.\r\n",
		"\r\n",
		"Powered by neovim, luals, gopls and clangd.\r\n",
		"\r\n",
		"\r\n",
		"ide@openrc-nvide:~ $ ls\r\n",
		"develop  proj     s.log    s.time\r\n",
		"ide@openrc-nvide:~ $ cd develop/\r\n",
		"ide@openrc-nvide:~/develop $ ls -al\r\n",
		"done\r\n",
	}
	postSeq := []string{
		"ide@openrc-nvide:~/develop $ ls -al\r\n",
		"total 972\r\n",
		"drwxr-xr-x   19 ide      develop\r\n",
		"drwxr-sr-x    1 ide      develop       4096 Oct 30 13:06 ..\r\n",
		"\r\n",
		"drwxr-xr-x    9 ide      develop        288 Jul 21 09:29 NvChad\r\n",
		"drwxr-xr-x   19 ide      develop        608 Oct 27 13:46 aprilsh\r\n",
		"drwxr-xr-x   18 ide      develop        576 Jan 27  2022 dotfiles\r\n",
		"-rwx------   demo.key\r\n",
		"-rw-r--r--    1 ide      develop        go.work\r\n",
		"-rw-r--r--    1 ide      develop        141 Sep 27 17:05 git.md\r\n",
		"\r\n",
	}

	tc := []struct {
		label  string
		expect string
		row    int // last position
		col    int
	}{
		{"blank old row zero start", "\x1b[?25l\ntotal 972\x1b[K", 1, 0},
		{"blank old row", "\x1b[?25l\r\ntotal 972\x1b[K", 1, 5},
		{"blank new row zero start", "\x1b[?25l\n\x1b[K", 4, 0},
		{"blank new row", "\x1b[?25l\r\n\x1b[K", 4, 4},
		{"old row is longer than new one", "\x1b[?25l\n-rwx------   demo.key\x1b[K", 8, 0},
		{
			"new row is longer than old one",
			"\x1b[?25l\n-rw-r--r--    1 ide\x1b[6X\x1b[6Cdevelop\x1b[8X\x1b[8Cgo.work\x1b[K", 9, 0,
		},
	}

	util.Logger.CreateLogger(io.Discard, true, slog.LevelDebug)

	oldE := NewEmulator3(80, 8, 4)
	newE := NewEmulator3(80, 8, 4)

	for _, v := range tc {
		t.Run(v.label, func(t *testing.T) {
			// init the screen content
			for _, seq := range preSeq {
				oldE.HandleStream(seq)
			}
			for _, seq := range postSeq {
				newE.HandleStream(seq)
			}
			d, _ := NewDisplay(false)
			// d.printFramebufferInfo(oldE, newE)

			frame := new(FrameState)
			frame.cursorX = v.col     // last position
			frame.cursorY = v.row - 1 // last position
			frame.currentRendition = Renditions{}
			frame.showCursorMode = oldE.showCursorMode
			frame.lastFrame = oldE
			frame.out = &strings.Builder{}

			var oldRow []Cell
			var newRow []Cell

			rawY := v.row
			frameY := v.row

			// print info
			util.Logger.Debug("TestPutRow", "Before: ", fmt.Sprintf("fs.cursor=(%2d,%2d)", frame.cursorY, frame.cursorX))
			util.Logger.Debug("TestPutRow", "OldRow", printRow(oldE.cf.cells, rawY, oldE.nCols))
			util.Logger.Debug("TestPutRow", "NewRow", printRow(newE.cf.cells, rawY, newE.nCols))

			oldRow = oldE.cf.getRow(rawY)
			newRow = newE.cf.getRow(rawY)
			wrap := false

			// run it
			d.putRow2(false, frame, newE, newRow, frameY, oldRow, wrap)

			// print info
			util.Logger.Debug("TestPutRow", "After:  ", fmt.Sprintf("fs.cursor=(%2d,%2d)", frame.cursorY, frame.cursorX))
			util.Logger.Debug("TestPutRow", "frameY", frameY, "out", frame.output())

			// validate result
			if frame.output() != v.expect {
				t.Errorf("#TestPutRow %q expect %q got %q\n", v.label, v.expect, frame.output())
			}
		})
	}
}

func TestCalculateRows(t *testing.T) {
	tc := []struct {
		label               string
		oldHead, oldY, oldX int
		newHead, newY, newX int
		expect              int
	}{
		{"new head > old head, same heigh", 10, 19, 0, 11, 19, 0, 2},
		{"new head > old head, diff heigh", 0, 9, 0, 11, 19, 0, 22},
		{"new head = old head, diff heigh", 1, 9, 0, 1, 19, 0, 10},
		{"new head = old head, diff heigh", 50, 9, 0, 50, 19, 0, 10},
		{"new head = old head, same heigh", 50, 19, 0, 50, 19, 0, 0},
		{"new head = old head, diff x    ", 50, 19, 0, 50, 19, 5, 1},
		{"new head < old head, diff heigh", 50, 9, 0, 10, 19, 0, 31},  // rewind happens
		{"new head < old head, same heigh", 40, 19, 0, 20, 19, 0, 41}, // rewind happens
		{"new head < old head, full frame", 40, 19, 0, 39, 19, 0, 60}, // rewind and full frame
		{"new head = 0 = old, diff height", 0, 4, 23, 0, 5, 23, 2},
		{"new head = 0 = old, diff column", 0, 4, 0, 0, 4, 20, 1},
		{"old head = 0 = old, start zero ", 0, 0, 0, 0, 4, 20, 5},
	}

	oldE := NewEmulator3(80, 20, 40)
	newE := NewEmulator3(80, 20, 40)

	for _, v := range tc {
		t.Run(v.label, func(t *testing.T) {
			// prepare for condition
			oldE.posY = v.oldY
			oldE.posX = v.oldX
			oldE.cf.scrollHead = v.oldHead
			newE.posY = v.newY
			newE.posX = v.newX
			newE.cf.scrollHead = v.newHead

			got := calculateRows(oldE, newE)
			if got != v.expect {
				t.Errorf("%q expect %d, got %d\n", v.label, v.expect, got)
			}
		})
	}
}

func buildSelectionDataSequence(raw string) string {
	Pd := base64.StdEncoding.EncodeToString([]byte(raw))
	// s := fmt.Sprintf("\x1B]%d;%s;%s\x1B\\", 52, "pc", Pd)
	// fmt.Printf("#test buildSelectionDataSequence() s=%q\n", s)
	// return s
	return fmt.Sprintf("\x1B]%d;%s;%s\x1B\\", 52, "pc", Pd)
}

func TestDisplayClone(t *testing.T) {
	os.Setenv("TERM", "xterm-256color")
	d, e := NewDisplay(true)
	if e != nil {
		t.Errorf("#test create display error: %s\n", e)
	}

	// clone and make some difference
	d.smcup = "clone"
	// d.currentRendition.buildRendition(34)

	c := d.Clone()

	if c.smcup != d.smcup {
		t.Errorf("#test Clone() expect smcup %q, got %q\n", d.smcup, c.smcup)
	}

	rend := Renditions{}
	rend.buildRendition(34)
	// if c.currentRendition != rend {
	// 	t.Errorf("#test Clone() expect currentRendition %#v, got %#v\n", rend, c.currentRendition)
	// }
}

/*
	func equalIntSlice(a, b []int) bool {
		if len(a) != len(b) {
			return false
		}
		for i := range a {
			if a[i] != b[i] {
				return false
			}
		}
		return true
	}

	func equalStringlice(a, b []string) bool {
		if len(a) != len(b) {
			return false
		}
		for i := range a {
			if a[i] != b[i] {
				return false
			}
		}
		return true
	}
*/

func TestEqualSlice(t *testing.T) {
	a := []int{1, 2, 3, 4}
	b := []int{1, 2, 3, 4}

	if !equalSlice(a, b) {
		t.Errorf("compare two int slice should return true\n")
	}

	c := []string{"h", "e", "l", "l", "o"}
	d := []string{"h", "e", "l", "l", "o"}

	if !equalSlice(c, d) {
		t.Errorf("compare two string slice should return true\n")
	}
}

func BenchmarkEqualSlice(b *testing.B) {
	a := []int{1, 2, 3, 4}
	c := []int{1, 2, 3, 2}

	for i := 0; i < b.N; i++ {
		equalSlice(a, c)
	}
}

func BenchmarkEqualSlice2(b *testing.B) {
	c := []string{"h", "e", "l", "l", "o"}
	d := []string{"h", "e", "l", "l", "a"}

	for i := 0; i < b.N; i++ {
		equalSlice(c, d)
	}
}

func BenchmarkEqualRow(b *testing.B) {
	var x, y []Cell
	x = make([]Cell, 80*80)
	y = make([]Cell, 80*80)

	for i := 0; i < 80*80; i++ {
		x[i] = Cell{contents: fmt.Sprintf("%b", i)}
		y[i] = Cell{contents: fmt.Sprintf("%b", i*2)}
	}

	for i := 0; i < b.N; i++ {
		equalRow(x, y)
	}
}

func BenchmarkStringBuilder(b *testing.B) {
	buf := encrypt.PrngFill(16)
	var bd strings.Builder

	bd.Grow(len(buf) * 5)
	for i := 0; i < b.N; i++ {
		bd.Write(buf)
	}
}

func BenchmarkAppend(b *testing.B) {
	var sb strings.Builder

	frame := new(FrameState)
	frame.cursorX = 0
	frame.cursorY = 0
	frame.currentRendition = Renditions{}
	frame.out = &sb

	for i := 0; i < b.N; i++ {
		frame.append("payload")
	}
}

func BenchmarkAppendx(b *testing.B) {
	var sb strings.Builder

	frame := new(FrameState)
	frame.cursorX = 0
	frame.cursorY = 0
	frame.currentRendition = Renditions{}
	frame.out = &sb

	for i := 0; i < b.N; i++ {
		frame.append("%s", "powerpoint")
	}
}
