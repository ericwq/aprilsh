// Copyright 2022 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package terminal

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/ericwq/aprilsh/util"
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
		{"useEnvironment, dynamic TERM", true, "sun", errors.New("terminal entry not found"), true, true, false}, // we choose sun, because sun fade out from the market
		{"useEnvironment, wrong TERM", true, "stranger", errors.New("infocmp: couldn't open terminfo file"), false, false, false},
		{"not useEnvironment ", false, "anything", nil, true, true, true},
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
				if d.hasTitle != v.hasTitle {
					t.Errorf("%q expect title %t, got %t\n", v.label, v.hasTitle, d.hasTitle)
				}
			} else {
				// fmt.Printf("#test NewDisplay() %q return %q ,expect %q\n", v.label, e, v.err)
				if !strings.HasPrefix(e.Error(), v.err.Error()) {
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
			"\x1b[?25l\x1b[11;74Houtput for\x1b[1Cnormal\x1b[1Cwrap\x1b[1Cline.\x1b[?25h", 11,
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
			"new screen with big space gap", ' ', ' ',
			"\x1B[5;1H1st space\x1B[0K\x1b[5;21H2nd!   \x1B[1;37;40m   3rd\x1b[5;79HEOL  \x1b[0m", true,
			"\x1b[?25l\n\n\n\n1st\x1b[1Cspace\x1b[5;21H2nd!\x1b[3C\x1b[0;1;37;40m   3rd\x1b[0m\x1b[5;79H\x1b[0;1;37;40mE\x1b[5;80HOL  \x1b[0m\x1b[K\x1b[?25h", 4,
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
		{
			"backspace case", ' ', ' ', "\x1b[9;1Hbackspace case\x1b[9;11H", true,
			"\x1b[?25l\x1b[9;1Hbackspace\x1b[1Ccase\b\b\b\b\x1b[?25h", 8,
			"[  8] backspace.case..................................................................",
		},
		{
			"mix color case", ' ', ' ', "\x1b[10;1H\x1b[1;34mdevelop\x1b[m  \x1b[1;34mproj     \x1b[m", true,
			"\x1b[?25l\x1b[10;1H\x1b[0;1;34mdevelop\x1b[0m\x1b[2C\x1b[0;1;34mproj\x1b[5X\x1b[10;19H\x1b[0m\x1b[K\x1b[?25h", 9,
			"[  9] develop..proj...................................................................",
		},
		{
			"mix color, false initialized case", ' ', ' ',
			"\x1b[10;1H\x1b[1;34mdevelop\x1b[m  \x1b[1;35mproj\x1b[m", false,
			"\x1b[?5l\x1b[r\x1b[0m\x1b[H\x1b[2J\x1b[?25l\x1b[?47l\x1b[r\x1b[?69l\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[0;1;34mdevelop\x1b[0m  \x1b[0;1;35mproj\x1b[0m\x1b[K\r\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\x1b[10;14H\x1b[?25h\x1b[1 q\x1b]112\a\x1b[0m\x1b[?2004l\x1b[?1003l\x1b[?1002l\x1b[?1001l\x1b[?1000l\x1b[?1004l\x1b[?1015l\x1b[?1006l\x1b[?1005l\x1b[?7h\x1b[20l\x1b[2l\x1b[4l\x1b[12h\x1b[?67l\x1b[?1036h\x1b[?1007l\x1b[?1l\x1b[?6l\x1b>\x1b[?3l\x1b[3g\x1b[64\"p\x1b[>4;1m", 9,
			"[  9] develop..proj...................................................................",
		},
	}

	defer util.Log.Restore()
	util.Log.SetOutput(io.Discard)

	os.Setenv("TERM", "xterm-256color")
	d, e := NewDisplay(true)
	if e != nil {
		t.Errorf("#test NewFrame() create display error: %s\n", e)
	}

	for _, v := range tc {
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
	}
}

func TestNewFrame_ScrollUp(t *testing.T) {
	tc := []struct {
		label       string
		bgRune1     rune
		bgRune2     rune
		mixSeq      string
		extraSeq    string
		scrollSeq   string
		initialized bool
		expectSeq   string
	}{
		{
			"scroll up 5 lines", ' ', ' ', "\x1B[5;1Hscroll\r\ndown\r\nmore\r\nthan\r\n5 lines!",
			"\r\ndifferent line", "\x1B[4S", true,
			"\x1b[0m\x1b[?25l\x1b[9;1H\x1b[4S\x1b[6;1Hdifferent\x1b[1Cline\x1b[10;15H\x1b[?25h",
		},
		{
			"scroll up 6 lines", ' ', ' ', "\x1B[35;1Hscroll\r\ndown\r\nmore\r\nthan\r\n6\r\nlines!",
			"", "\x1B[34S", true,
			"\x1b[0m\r\x1b[34S\x1b[40;7H",
		},
	}

	oldE := NewEmulator3(80, 40, 40)
	newE := NewEmulator3(80, 40, 40)

	// oldE.logT.SetOutput(io.Discard)
	// newE.logT.SetOutput(io.Discard)
	defer util.Log.Restore()
	util.Log.SetOutput(io.Discard)

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

		// check the expect difference sequence
		gotSeq := d.NewFrame(v.initialized, oldE, newE)
		if gotSeq != v.expectSeq {
			t.Errorf("%q expect \n%q, got \n%q\n", v.label, v.expectSeq, gotSeq)
		}

		// fmt.Printf("OLD:\n%s", printCells(oldE.cf))
		// fmt.Printf("NEW:\n%s", printCells(newE.cf))

		// apply difference sequence to target
		oldE.HandleStream(gotSeq)

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
		initialized bool
		bell        bool
		expectSeq   string
	}{
		{"no bell", true, false, ""},
		{"has bell", true, true, "\a"},
	}
	oldE := NewEmulator3(80, 40, 40)
	newE := NewEmulator3(80, 40, 40)

	// oldE.logT.SetOutput(io.Discard)
	// newE.logT.SetOutput(io.Discard)
	defer util.Log.Restore()
	util.Log.SetOutput(io.Discard)

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
			newE.cf.ringBell()
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
		showStyle CursorStyle
		expectSeq string
	}{
		{"same blink block", CursorStyle_BlinkBlock, ""},
		{"steady block", CursorStyle_SteadyBlock, "\x1B[2 q"},
		{"blink underline", CursorStyle_BlinkUnderline, "\x1B[3 q"},
		{"steady underline", CursorStyle_SteadyUnderline, "\x1B[4 q"},
		{"blink bar", CursorStyle_BlinkBar, "\x1B[5 q"},
		{"steady bar", CursorStyle_SteadyBar, "\x1B[6 q"},
	}
	oldE := NewEmulator3(80, 40, 40)
	newE := NewEmulator3(80, 40, 40)

	// defer util.Log.Restore()
	// util.Log.SetOutput(io.Discard)

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
		initialized bool
		windowTitle string
		iconName    string
		expectSeq   string
	}{
		{"no window title and icon name", true, "", "", ""},
		{"has window title", true, "window title", "", "\x1b]1;\a\x1b]2;window title\a"},
		{"has chinese icon name", true, "", "图标名称", "\x1b]1;图标名称\a\x1b]2;\a"},
		{"has same window title & icon name", true, "中文标题", "中文标题", "\x1b]0;中文标题\a"},
	}
	oldE := NewEmulator3(80, 40, 40)
	newE := NewEmulator3(80, 40, 40)

	// oldE.logT.SetOutput(io.Discard)
	// newE.logT.SetOutput(io.Discard)
	defer util.Log.Restore()
	util.Log.SetOutput(io.Discard)

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
			newE.cf.setWindowTitle(v.windowTitle)
			newE.cf.setTitleInitialized()
		}

		if v.iconName != "" {
			newE.cf.setIconLabel(v.iconName)
			newE.cf.setTitleInitialized()
		}

		// check the expect difference sequence
		gotSeq := d.NewFrame(v.initialized, oldE, newE)
		if gotSeq != v.expectSeq {
			t.Errorf("%q expect \n%q, got \n%q\n", v.label, v.expectSeq, gotSeq)
		}
	}
}

func TestNewFrame_ReverseVideo(t *testing.T) {
	tc := []struct {
		label        string
		initialized  bool
		reverseVideo bool // determine the reverseVideo value of pair terminal
		seq          string
		expectSeq    string
	}{
		{"has reverse video", true, true, "\x1B[?5h", "\x1b[?5h"},
		{"no reverse video", true, false, "\x1B[?5h", "\x1B[?5l"},
	}
	oldE := NewEmulator3(80, 40, 40)
	newE := NewEmulator3(80, 40, 40)

	// oldE.logT.SetOutput(io.Discard)
	// newE.logT.SetOutput(io.Discard)

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

		// oldE.logT.SetOutput(io.Discard)
		// newE.logT.SetOutput(io.Discard)

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
		initialized         bool
		altScreenBufferMode bool
		seq                 string
		expectSeq           string
	}{
		{"already initialized, has altScreenBufferMode", true, true, "\x1B[?47h", "\x1B[?47h"},
		{"already initialized, no altScreenBufferMode", true, false, "\x1B[?47h", "\x1B[?47l"},
	}
	oldE := NewEmulator3(8, 4, 4)
	newE := NewEmulator3(8, 4, 4)

	// oldE.logT.SetOutput(io.Discard)
	// newE.logT.SetOutput(io.Discard)

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
		initialized bool
		margin      bool
		seq         string
		expectSeq   string
	}{
		{"already initialized, new has margin", true, true, "\x1B[2;6r", "\x1b[2;6r"},
		{"already initialized, old has margin", true, false, "\x1B[2;6r", "\x1b[r"},
	}
	oldE := NewEmulator3(8, 8, 4)
	newE := NewEmulator3(8, 8, 4)

	// oldE.logT.SetOutput(io.Discard)
	// newE.logT.SetOutput(io.Discard)

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
		initialized bool
		margin      bool
		seq         string
		expectSeq   string
	}{
		{"already initialized, new has margin", true, true, "\x1B[?69h\x1B[2;6s", "\x1b[?69h\x1b[2;6s"},
		{"already initialized, old has margin", true, false, "\x1B[?69h\x1B[2;6s", "\x1b[?69l"},
		{"already initialized, both no margin", true, false, "", ""},
	}
	oldE := NewEmulator3(8, 8, 4)
	newE := NewEmulator3(8, 8, 4)

	// oldE.logT.SetOutput(io.Discard)
	// newE.logT.SetOutput(io.Discard)

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
		initialized bool
		decsc       bool
		seq         string
		expectSeq   string
	}{
		{"already initialized, new has decsc", true, true, "\x1B7", "\x1b7"},
		{"already initialized, old has decsc", true, false, "\x1B7", "\x1b8"},
		{"already initialized, both no decsc", true, false, "", ""},
	}
	oldE := NewEmulator3(8, 8, 4)
	newE := NewEmulator3(8, 8, 4)

	// oldE.logT.SetOutput(io.Discard)
	// newE.logT.SetOutput(io.Discard)

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
		initialized bool
		scosc       bool
		seq         string
		expectSeq   string
	}{
		{"already initialized, new has scosc", true, true, "\x1B[s", "\x1b[s"},
		{"already initialized, old has scosc", true, false, "\x1B[s", "\x1b[u"},
		{"already initialized, both no scosc", true, false, "", ""},
	}
	oldE := NewEmulator3(8, 8, 4)
	newE := NewEmulator3(8, 8, 4)

	// oldE.logT.SetOutput(io.Discard)
	// newE.logT.SetOutput(io.Discard)

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
		initialized          bool
		showcursorModeForNew bool
		seq                  string
		expectSeq            string
	}{
		{"already initialized, new show no cursor", true, true, "\x1B[?25l", "\x1b[?25l"},
		{"already initialized, old show no cursor", true, false, "\x1B[?25l", "\x1b[?25h"},
		{"already initialized, both show cursor", true, false, "", ""},
	}
	oldE := NewEmulator3(8, 8, 4)
	newE := NewEmulator3(8, 8, 4)

	// oldE.logT.SetOutput(io.Discard)
	// newE.logT.SetOutput(io.Discard)

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
		initialized        bool
		bracketedPasteMode bool
		seq                string
		expectSeq          string
	}{
		{"already initialized, new has bracketedPasteMode", true, true, "\x1B[?2004h", "\x1b[?2004h"},
		{"already initialized, old has bracketedPasteMode", true, false, "\x1B[?2004h", "\x1b[?2004l"},
		{"already initialized, both no bracketedPasteMode", true, false, "", ""},
	}
	oldE := NewEmulator3(8, 8, 4)
	newE := NewEmulator3(8, 8, 4)

	// oldE.logT.SetOutput(io.Discard)
	// newE.logT.SetOutput(io.Discard)

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

	// oldE.logT.SetOutput(io.Discard)
	// newE.logT.SetOutput(io.Discard)

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
		focusEventMode bool
		seq            string
		expectSeq      string
	}{
		{"new has focusEventMode", true, "\x1B[?1004h", "\x1b[?1004h"},
		{"old has focusEventMode", false, "\x1B[?1004h", "\x1b[?1004l"},
		{"both no focusEventMode", false, "", ""},
	}
	oldE := NewEmulator3(8, 8, 4)
	newE := NewEmulator3(8, 8, 4)

	// oldE.logT.SetOutput(io.Discard)
	// newE.logT.SetOutput(io.Discard)

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

	// oldE.logT.SetOutput(io.Discard)
	// newE.logT.SetOutput(io.Discard)
	defer util.Log.Restore()
	util.Log.SetOutput(io.Discard)

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
			"\x1b[1;7H\x1bH\x1b[1;17H\x1bH\x1b[1;27H\x1bH\x1b[8;8H",
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

	// oldE.logT.SetOutput(io.Discard)
	// newE.logT.SetOutput(io.Discard)
	defer util.Log.Restore()
	util.Log.SetOutput(io.Discard)

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

	// oldE.logT.SetOutput(io.Discard)
	// newE.logT.SetOutput(io.Discard)
	defer util.Log.Restore()
	util.Log.SetOutput(io.Discard)

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

func buildSelectionDataSequence(raw string) string {
	Pd := base64.StdEncoding.EncodeToString([]byte(raw))
	// s := fmt.Sprintf("\x1B]%d;%s;%s\x1B\\", 52, "pc", Pd)
	// fmt.Printf("#test buildSelectionDataSequence() s=%q\n", s)
	// return s
	return fmt.Sprintf("\x1B]%d;%s;%s\x1B\\", 52, "pc", Pd)
}

func TestDisplayCone(t *testing.T) {
	os.Setenv("TERM", "xterm-256color")
	d, e := NewDisplay(true)
	if e != nil {
		t.Errorf("#test create display error: %s\n", e)
	}

	// clone and make some difference
	d.smcup = "clone"
	d.currentRendition.buildRendition(34)

	c := d.Clone()

	if c.smcup != d.smcup {
		t.Errorf("#test Clone() expect hasTitle %q, got %q\n", d.smcup, c.smcup)
	}

	rend := Renditions{}
	rend.buildRendition(34)
	if c.currentRendition != rend {
		t.Errorf("#test Clone() expect currentRendition %#v, got %#v\n", rend, c.currentRendition)
	}
}
