// Copyright 2022 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package terminal

import (
	"fmt"
	"io"
	"log/slog"
	"maps"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/ericwq/aprilsh/util"
	"github.com/rivo/uniseg"
)

func TestEmulatorResize(t *testing.T) {
	type Result struct {
		nCols, nRows      int
		hMargin, nColsEff int
	}
	tc := []struct {
		label               string
		nCols, nRows        int
		altScreenBufferMode bool
		horizMarginMode     bool
		posY                int
		expect              Result
	}{
		// each test case will affect the result of next test case.
		{"same size", 80, 40, false, false, 0, Result{80, 40, 0, 80}},
		{"extend width", 90, 40, false, false, 0, Result{90, 40, 0, 90}},
		{"extend height", 90, 50, false, false, 0, Result{90, 50, 0, 90}},
		{"extend both", 92, 52, false, false, 0, Result{92, 52, 0, 92}},
		{"shrink height", 92, 50, false, false, 0, Result{92, 50, 0, 92}},
		{"shrink width", 90, 50, false, false, 0, Result{90, 50, 0, 90}},
		{"alt screen ", 80, 40, true, false, 0, Result{80, 40, 0, 80}},
		{"shrink height with posY is oversize", 90, 35, false, true, 39, Result{90, 35, 0, 80}},
		// before the resize operation: the posY is at 39, the previous height is 40,
		// now we shrink it to 35.
	}

	emu := NewEmulator3(80, 40, 40) // this is the initialized size.

	for _, v := range tc {
		t.Run(v.label, func(t *testing.T) {
			emu.altScreenBufferMode = v.altScreenBufferMode
			emu.horizMarginMode = v.horizMarginMode
			emu.posY = v.posY
			emu.resize(v.nCols, v.nRows)

			if v.expect.nCols != emu.nCols || v.expect.nRows != emu.nRows ||
				v.expect.hMargin != emu.hMargin || v.expect.nColsEff != emu.nColsEff {
				t.Errorf("%q expect %v, got (%d,%d,%d,%d)\n",
					v.label, v.expect, emu.nCols, emu.nRows, emu.hMargin, emu.nColsEff)
			}
		})
	}
}

func TestEmulatorReadOctetsToHost(t *testing.T) {
	tc := []struct {
		name   string
		expect string
		rawStr []string
	}{
		{"one sequence", "\x1B[23m", []string{"\x1B[23m"}},
		{"three mix sequence", "\x1B[24;14H\x1B[3g长", []string{"\x1B[24;14H", "\x1B[3g", "长"}},
	}

	emu := NewEmulator3(80, 40, 0)

	for _, v := range tc {
		// write raw string to the internal terminalToHost
		for _, raw := range v.rawStr {
			emu.writePty(raw)
		}
		got := emu.ReadOctetsToHost()
		if v.expect != got {
			t.Errorf("%q expect %q, got %q\n", v.name, v.expect, got)
		}
	}
}

func TestEmulatorHandleStreamEmpty(t *testing.T) {
	emu := NewEmulator3(80, 40, 0)
	hds, _ := emu.HandleStream("")
	if len(hds) != 0 {
		t.Errorf("#test HandleStream with empty input should zero result, got %v\n", hds)
	}
}

func TestEmulatorNormalizeCursorPos(t *testing.T) {
	type Position struct {
		posX, posY int
	}
	tc := []struct {
		name   string
		from   Position
		expect Position
	}{
		{"outof of columns", Position{80, 5}, Position{79, 5}},
		{"outof of rows", Position{5, 40}, Position{5, 39}},
	}

	emu := NewEmulator3(80, 40, 0)
	for _, v := range tc {
		emu.posX = v.from.posX
		emu.posY = v.from.posY
		emu.normalizeCursorPos()

		if emu.posX != v.expect.posX || emu.posY != v.expect.posY {
			t.Errorf("%q expect %v, got (%d,%d)\n", v.name, v.expect, emu.posY, emu.posX)
		}
	}
}

func TestEmulatorJumpToNextTabStop(t *testing.T) {
	tc := []struct {
		name       string
		setPosX    int
		fromPosX   int
		expectPosX int
	}{
		{"before tab stop position", 48, 43, 48},
		{"after tab stop position", 56, 70, 79},
	}

	emu := NewEmulator3(80, 40, 0)
	for _, v := range tc {
		emu.posX = v.setPosX
		emu.posY = 0

		// add an item in tabStops, setPosX
		hdl_esc_hts(emu)

		// set the start position
		emu.posX = v.fromPosX

		// jump to the next tab stop
		emu.jumpToNextTabStop()

		// validate the reult
		if v.expectPosX != emu.posX {
			t.Errorf("%q expect column %d, got %d\n", v.name, v.expectPosX, emu.posX)
		}
	}

	if emu.GetFramebuffer() == nil {
		t.Errorf("#test jumpToNextTabStop should never return a nil framebuffer\n")
	}
}

func TestEmulatorLookupCharset(t *testing.T) {
	emu := NewEmulator3(80, 40, 0)

	resetCharsetState(&emu.charsetState)
	// gr = 2, g[2]= DEC special
	emu.charsetState.g[emu.charsetState.gr] = &vt_DEC_Special

	str := "\x5f\x68\x7a"
	want := []rune{0x00a0, 0x2424, 0x2265}

	for i, x := range str {
		// set ss to 2
		emu.charsetState.ss = 2

		y := emu.lookupCharset(x)
		if y != want[i] {
			t.Errorf("for %x expect %U , got %U \n", x, want[i], y)
		}
	}
}

func TestEmulatorPasteSelection(t *testing.T) {
	tc := []struct {
		label              string
		selection          string
		expect             string
		bracketedPasteMode bool
	}{
		{"bracketedPasteMode is false", "lock down", "lock down", false},
		{"bracketedPasteMode is true, english ", "lock down", "\x1b[200~lock down\x1b[201~", true},
		{"bracketedPasteMode is true, chinese ", "解除封控", "\x1b[200~解除封控\x1b[201~", true},
	}

	emu := NewEmulator3(80, 40, 0)

	for _, v := range tc {
		emu.bracketedPasteMode = v.bracketedPasteMode
		got := emu.pasteSelection(v.selection)
		if got != v.expect {
			t.Errorf("%q expect %q, got %q\n", v.label, v.expect, got)
		}
	}

	if emu.GetParser() == nil {
		t.Errorf("#test pasteSelection() should return non-nil parser.\n")
	}
}

func TestEmulatorHasFocus(t *testing.T) {
	tc := []struct {
		label          string
		hasFocus       bool
		showCursorMode bool
		expect         CursorStyle
	}{
		{"hasFocus any, showCursorMode false", false, false, CursorStyle_Hidden},
		{"hasFocus false, showCursorMode true", false, true, CursorStyle_HollowBlock},
		{"hasFocus true, showCursorMode true", true, true, CursorStyle_FillBlock},
	}
	emu := NewEmulator3(80, 40, 0)

	for _, v := range tc {
		emu.setHasFocus(v.hasFocus)
		emu.showCursorMode = v.showCursorMode
		if !emu.showCursorMode {
			emu.hideCursor()
		} else {
			emu.showCursor()
		}

		got := emu.cf.cursor.style
		if got != v.expect {
			t.Errorf("%q expect cursor style %d, got %d\n", v.label, v.expect, got)
		}
	}
}

func TestEmulatorGetWidth(t *testing.T) {
	emu := NewEmulator3(80, 40, 40)

	// validate get GetWidth
	if emu.GetWidth() != 80 {
		t.Errorf("#test GetWidth() expect %d, got %d\n", 80, emu.GetWidth())
	}

	util.Logger.CreateLogger(io.Discard, true, slog.LevelDebug)
	// util.Logger.CreateLogger(os.Stdout, true, slog.LevelDebug)

	// set horizontal margin
	emu.HandleStream("\x1b[9;1Hset hMargin\x1B[?69h\x1B[2;78s")

	if emu.GetWidth() != 77 {
		t.Errorf("#test GetWidth() expect %d, got %d\n", 77, emu.GetWidth())
	}

	if emu.GetSaveLines() != 40 {
		t.Errorf("#test GetSaveLines() expect %d, got %d\n", 40, emu.GetSaveLines())
	}
}

func TestEmulatorMoveCursor(t *testing.T) {
	tc := []struct {
		label            string
		posY, posX       int
		expectY, expectX int
	}{
		{"in the top,left corner", 0, 0, 0, 0},
		{"in the middle", 20, 40, 20, 40},
		{"in the right,bottom corner", 39, 79, 39, 79},
		{"reset to top/left corner with origin mode", 0, 0, 1, 1}, // reset the cursor to top/left position
		{"out of range negative", -1, -1, 1, 1},
		{"out of range 2", 50, 90, 38, 67},
		{"out of range 3", 39, 79, 38, 67},
	}
	emu := NewEmulator3(80, 40, 40)

	for _, v := range tc {
		t.Run(v.label, func(t *testing.T) {
			if strings.Contains(v.label, "origin mode") {
				// set OriginMode_ScrollingRegion \x1B[?6
				// Set Scrolling Region [top;bottom] \x1B[2;38r
				// Set left and right margins (DECSLRM) \x1B[2;68s
				emu.HandleStream("\x1B[?6h\x1B[2;38r\x1B[?69h\x1B[2;68s")
				// fmt.Printf("#test originMode=%d, top=%d, bottom=%d\n", emu.originMode, emu.marginTop, emu.marginBottom)
				// fmt.Printf("#test horizMarginMode=%t, hMargin=%d, nColsEff=%d\n", emu.horizMarginMode, emu.hMargin, emu.nColsEff)
			}
			emu.MoveCursor(v.posY, v.posX)

			if emu.posY != v.expectY || emu.posX != v.expectX {
				t.Errorf("%q expect cursor position (%d,%d), got (%d,%d)\n", v.label, v.expectY, v.expectX, emu.posY, emu.posX)
			}
		})
	}
}

func TestEmulatorSetCursorVisible(t *testing.T) {
	emu := NewEmulator3(80, 40, 40)

	emu.SetCursorVisible(false)
	if emu.cf.cursor.style != CursorStyle_Hidden {
		t.Errorf("#test SetCursorVisible expect %d, got %d\n", CursorStyle_Hidden, emu.cf.cursor.style)
	}

	emu.SetCursorVisible(true)
	if emu.cf.cursor.style != CursorStyle_FillBlock {
		t.Errorf("#test SetCursorVisible expect %d, got %d\n", CursorStyle_FillBlock, emu.cf.cursor.style)
	}
}

func TestEmulatorPrefixWindowTitle(t *testing.T) {
	emu := NewEmulator3(80, 40, 40)

	base := "base title"
	prefix := "前缀"
	emu.setTitleInitialized()
	emu.setWindowTitle(base)

	emu.PrefixWindowTitle(prefix)

	expect := prefix + base
	got := emu.GetWindowTitle()

	if got != expect {
		t.Errorf("#test PrefixWindowTitle() expect %q, got %q\n", expect, got)
	}
}

func TestEmulatorGetCell(t *testing.T) {
	tc := []struct {
		label      string
		seq        string
		contents   string
		posY, posX int
	}{
		{"in the middle", "\x1B[11;74Houtput for normal wrap line.", "o", 10, 73},
		{"in the last cols", "", " ", 10, 79},
		{"in the first cols", "", "f", 11, 0},
	}

	emu := NewEmulator3(80, 40, 40)

	util.Logger.CreateLogger(io.Discard, true, slog.LevelDebug)
	// util.Logger.CreateLogger(os.Stderr, true, slog.LevelDebug)

	for _, v := range tc {
		emu.HandleStream(v.seq)
		c := emu.GetCell(v.posY, v.posX)

		if v.contents != c.contents {
			t.Errorf("%q expect (%d,%d) contains %q, got %q\n", v.label, v.posY, v.posX, v.contents, c.contents)
		}

		pc := emu.GetCellPtr(v.posY, v.posX)
		if v.contents != pc.contents {
			t.Errorf("%q expect (%d,%d) contains %q, got %q\n", v.label, v.posY, v.posX, v.contents, pc.contents)
		}
	}
}

func TestEmulatorClone(t *testing.T) {
	tc := []struct {
		label        string
		seq          string // mix data stream
		nRows, nCols int    // resize
	}{
		{"seq, no resize", "\x1B[11;74Houtput for normal wrap line.", 0, 0},
		{"alter screen buffer, no resize", "\x1B[?47h\x1B[11;74Houtput for normal wrap line.", 0, 0},
	}

	util.Logger.CreateLogger(io.Discard, true, slog.LevelDebug)
	// util.Logger.CreateLogger(os.Stderr, true, slog.LevelDebug)

	for _, v := range tc {
		emu := NewEmulator3(80, 40, 40)
		// emu.SetLogTraceOutput(io.Discard)

		emu.HandleStream(v.seq)
		if v.nCols != 0 && v.nRows != 0 {
			emu.resize(v.nCols, v.nRows)
		}

		got := emu.Clone()

		if !reflect.DeepEqual(emu, got) {
			if !reflect.DeepEqual(emu.cf, got.cf) {
				t.Errorf("%q cf is not equal\n", v.label)
			}
			if !reflect.DeepEqual(emu.frame_alt, got.frame_alt) {
				t.Errorf("%q frame_alt is not equal\n", v.label)
			}
			if !reflect.DeepEqual(emu.frame_pri, got.frame_pri) {
				t.Errorf("%q frame_pri is not equal\n", v.label)
			}
			if !reflect.DeepEqual(emu.savedCursor_DEC, got.savedCursor_DEC) {
				t.Errorf("%q savedCursor_DEC is not equal\n", v.label)
			}
			if !reflect.DeepEqual(emu.savedCursor_DEC_alt, got.savedCursor_DEC_alt) {
				t.Errorf("%q savedCursor_DEC_alt is not equal\n", v.label)
			}
			if !reflect.DeepEqual(emu.savedCursor_DEC_pri, got.savedCursor_DEC_pri) {
				t.Errorf("%q savedCursor_DEC_pri is not equal\n", v.label)
			}
			if !reflect.DeepEqual(emu.tabStops, got.tabStops) {
				t.Errorf("%q tabStops is not equal\n", v.label)
			}
			if !reflect.DeepEqual(emu.charsetState, got.charsetState) {
				t.Errorf("%q charsetState is not equal\n", v.label)
			}
			if !reflect.DeepEqual(emu.selectionData, got.selectionData) {
				t.Errorf("%q selectionData is not equal\n", v.label)
			}
			if !reflect.DeepEqual(emu.selectionStore, got.selectionStore) {
				t.Errorf("%q selectionStore is not equal\n", v.label)
			}
			if !reflect.DeepEqual(emu.user, got.user) {
				t.Errorf("%q user is not equal\n", v.label)
			}
			// if !reflect.DeepEqual(emu.logE, got.logE) {
			// 	t.Errorf("%q logE is not equal\n", v.label)
			// }
			if !reflect.DeepEqual(emu.parser, got.parser) {
			} else {
				t.Errorf("%q expect clone emulator is not equal with origin emulator\n", v.label)
				t.Errorf("%q parser is not equal\n", v.label)
			}
		}
	}
}

func TestHandleStream_MoveDelete(t *testing.T) {
	tc := []struct {
		label            string
		base             string // base content
		expect           string // the expect content
		row, col         int    // the start cursor position
		expectY, expectX int    // new cursor position
	}{
		{"move cursor and delete one regular graphemes", "abcde\x1B[4D\x1B[P", "acde", 0, 70, 0, 71},
		{"move cursor and delete one wide graphemes", "abc太学生\x1B[6D\x1B[2P", "abc学生", 1, 60, 1, 63},
		{"move cursor back and forth for wide graphemes", "东部战区\x1B[8D\x1B[2C\x1B[2P", "东战区", 2, 60, 2, 62},
		{"move cursor to right edge", "平潭\x1B[5C", "平潭", 3, 75, 3, 79},
		{"move cursor to left edge", "三号木\x1B[6D", "三号木", 4, 0, 4, 0},
		{"move cursor to left edge, delete 2 graphemes", "小鸡腿\x1B[6D\x1B[4P", "腿", 5, 0, 5, 0},
		{"move cursor and delete 2 graphemes", "gocto\x1B[8C\x1B[4D\x1B[2P", "gto", 6, 74, 6, 75},
		{"move cursor back and delete 4 regular graphemes", "捉鹰打goto\x1B[4D\x1B[4P鸟", "捉鹰打鸟", 7, 60, 7, 68},
	}
	emu := NewEmulator3(80, 40, 40) // TODO why we can't init emulator outside of for loop

	for _, v := range tc {
		t.Run(v.label, func(t *testing.T) {
			emu.MoveCursor(v.row, v.col)
			emu.HandleStream(v.base)
			// fmt.Printf("%s base=%q expect=%q, pos=(%d,%d)\n", v.label, v.base, v.expect, v.row, v.col)
			// printEmulatorCell(emu, v.row, v.col, v.expect, "After Base")

			graphemes := uniseg.NewGraphemes(v.expect)
			i := 0
			for graphemes.Next() {
				chs := graphemes.Runes()

				cell := emu.GetCellPtr(v.row, v.col+i)
				if cell.String() != string(chs) {
					t.Errorf("#test HandleStream() %q expect %s, got %s\n", v.label, string(chs), cell)
				}
				i += uniseg.StringWidth(string(chs))
			}

			gotY := emu.GetCursorRow()
			gotX := emu.GetCursorCol()

			if v.expectY != gotY || v.expectX != gotX {
				t.Errorf("#test HandleStream() expect cursor at (%d,%d), got (%d,%d)\n", v.expectY, v.expectX, gotY, gotX)
			}
		})
	}
}

func printEmulatorCell(emu *Emulator, row, col int, sample string, prefix string) {
	graphemes := uniseg.NewGraphemes(sample)
	i := 0
	for graphemes.Next() {
		chs := graphemes.Runes()

		cell := emu.GetCellPtr(row, col+i)
		fmt.Printf("%s # cell %p (%d,%d) is %q\n", prefix, cell, row, col+i, cell)
		i += uniseg.StringWidth(string(chs))
	}
}

/*
func testCalculateCellNum(t *testing.T) {
	emu := NewEmulator3(80, 40, 40) // TODO why we can't init emulator outside of for loop
	emu.MoveCursor(0, 79)
	// fmt.Printf("#test calculateCellNum() posX=%d, right edge=%d\n ", emu.posX, emu.nColsEff)
	got := calculateCellNum(emu, 0)
	// fmt.Printf("#test calculateCellNum() posX=%d, right edge=%d, got=%d\n ", emu.posX, emu.nColsEff, got)
	if got != 0 {
		t.Errorf("#test calculateCellNum() expect 0, got %d\n", got)
	}
}
*/

func TestIconNameWindowTitle(t *testing.T) {
	tc := []struct {
		name        string
		windowTitle string
		iconName    string
		prefix      string
		expect      string
	}{
		{"english diff string", "english window title", "english icon name", "prefix ", "english icon name"},
		{"chinese same string", "中文窗口标题", "中文窗口标题", "Aprish:", "Aprish:中文窗口标题"},
	}
	emu := NewEmulator3(80, 40, 40)

	for _, v := range tc {
		emu.setWindowTitle(v.windowTitle)
		emu.setIconLabel(v.iconName)
		emu.setTitleInitialized()

		if !emu.isTitleInitialized() {
			t.Errorf("%q expect isTitleInitialized %t, got %t\n", v.name, true, emu.isTitleInitialized())
		}

		if emu.GetIconLabel() != v.iconName {
			t.Errorf("%q expect IconName %q, got %q\n", v.name, v.iconName, emu.GetIconLabel())
		}

		if emu.GetWindowTitle() != v.windowTitle {
			t.Errorf("%q expect windowTitle %q, got %q\n", v.name, v.windowTitle, emu.GetWindowTitle())
		}

		emu.PrefixWindowTitle(v.prefix)
		if emu.GetIconLabel() != v.expect {
			t.Errorf("%q expect prefix iconName %q, got %q\n", v.name, v.expect, emu.GetIconLabel())
		}
	}
}

func TestSaveWindowTitleOnStack(t *testing.T) {
	emu := NewEmulator3(80, 40, 40)

	var out strings.Builder

	util.Logger.CreateLogger(&out, true, slog.LevelDebug)
	// util.Logger.CreateLogger(os.Stderr, true, slog.LevelDebug)

	// no title, check save stack
	emu.saveWindowTitleOnStack()
	expect := "no title exist"
	if !strings.Contains(out.String(), expect) {
		t.Errorf("TestSaveWindowTitleOnStack expect %s, got %s\n", expect, out.String())
	}

	// no title, check restore stack
	out.Reset()
	emu.restoreWindowTitleOnStack()
	expect = "empty stack"
	if !strings.Contains(out.String(), expect) {
		t.Errorf("TestSaveWindowTitleOnStack expect %s, got %s\n", expect, out.String())
	}
	out.Reset()

	// set title
	title := "our title prefix "
	emu.setWindowTitle(title)

	// push to the stack max
	for i := 0; i < 10; i++ {
		tt := fmt.Sprintf("%s%d", title, i)
		emu.setWindowTitle(tt)
		emu.saveWindowTitleOnStack()
		// fmt.Printf("i=%d, %s\n", i+1, tt)
	}

	// always got the last pushed result
	emu.restoreWindowTitleOnStack()
	expect = fmt.Sprintf("%s%d", title, 9)
	got := emu.GetWindowTitle()
	if got != expect {
		t.Errorf("windowTitle stack expect %s, got %s\n", expect, got)
	}
}

func TestEmulatorEqual(t *testing.T) {
	tc := []struct {
		label     string
		seq1      string
		seq2      string
		expectStr []string
		expect    bool
	}{
		{"size", "", "", []string{"nRows=", "nCols="}, false},
		{"same content", "Hello world", "Hello world", []string{}, true},
		{"diff cursor", "Hello world", "Hello world\x1b[7;24H", []string{"posX=", "posY="}, false},
		{
			"lastCol", "\x1b[4;76Hworld", "\x1b[4;80H",
			[]string{"lastCol=", "attrs="},
			false,
		},
		{
			"has focus", "Hello world\x1B[?1004h\x1B[I", "Hello world\x1B[?1004h\x1B[O",
			[]string{"hasFocus=", "reverseVideo="},
			false,
		},
		{
			"insert mode", "Hello world\x1B[4h", "Hello world\x1B[4l",
			[]string{"autoWrapMode=", "insertMode="},
			false,
		},
		{
			"bracketedPasteMode", "Hello world\x1B[?2004h", "Hello world\x1B[?2004l",
			[]string{"bracketedPasteMode=", "altScrollMode="},
			false,
		},
		{
			"altSendsEscape", "Hello world\x1B[?1036h", "Hello world\x1B[?1039l",
			[]string{"altSendsEscape=", "modifyOtherKeys="},
			false,
		},
		{
			"hMargin", "\x1B[?69h\x1B[2;38sworld", "world",
			[]string{"hMargin=", "nColsEff="},
			false,
		},
		{
			"diff tabStops", "world\x1BH", "world",
			[]string{"tabStops length="},
			false,
		},
		{
			"diff tabStops position", "\x1B[1;5H\x1BH", "\x1B[1;13H\x1BH\x1B[1;5H",
			[]string{"tabStops[0]="},
			false,
		},
		{
			"set charset ss", "\x1B[1;5H\x1BN", "\x1B[1;5H",
			[]string{"charsetState.ss="},
			false,
		},
		{
			"set application cursor", "\x1B[1;5H\x1B[?1h", "\x1B[1;5H",
			[]string{"cursorKeyMode="},
			false,
		},
		{
			"set application keypad", "\x1B[1;5H\x1B=", "\x1B[1;5H",
			[]string{"keypadMode="},
			false,
		},
		{
			"set cursor SCO", "\x1B[1;5H\x1B[s", "\x1B[1;5H",
			[]string{"savedCursor_SCO="},
			false,
		},
		{
			"set save cursor", "\x1B[1;5H\x1B7", "\x1B[1;5H",
			[]string{"savedCursor_DEC .SavedCursor_SCO="},
			false,
		},
		{
			"set save cursor charsetState", "\x1B[1;5H\x1BN\x1B7", "\x1B[1;5H",
			[]string{"savedCursor_DEC .charsetState .vtMode="},
			false,
		},
		{
			"set mouse tracking", "\x1B[1;5H\x1B[?1000h\x1B[?1005h", "\x1B[1;5H",
			[]string{"mouseTrk="},
			false,
		},
		{
			"set select data", "\x1B[1;5H\x1B]52;c;YXByaWxzaAo=\x1B\\", "\x1B[1;5H",
			[]string{"selectionData length="},
			false,
		},
		{
			"set window title", "\x1B[1;5H\x1B]1;adas\x1B\\", "\x1B[1;5H",
			[]string{"windowTitle="},
			false,
		},
		{
			"save window title on stack", "\x1B[1;5H\x1B]0;adas\x1B\\\x1B[22;2t", "\x1B[1;5H\x1B]0;adas\x1B\\",
			[]string{"windowTitleStack length="},
			false,
		},
		{
			"diff window title value on stack", "\x1B[1;5H\x1B]0;adas\x1B\\\x1B[22;2t", "\x1B[1;5H\x1B]0;addas\x1B\\\x1B[22;2t\x1B]0;adas\x1B\\",
			[]string{"windowTitleStack[0]="},
			false,
		},
	}

	var output strings.Builder

	util.Logger.CreateLogger(&output, true, slog.LevelDebug)
	// util.Logger.CreateLogger(os.Stdout, true, slog.LevelDebug)

	for _, v := range tc {
		t.Run(v.label, func(t *testing.T) {
			var emu1, emu2 *Emulator
			if v.label == "size" {
				emu1 = NewEmulator3(80, 45, 0)
				emu2 = NewEmulator3(80, 40, 0)
			} else {
				emu1 = NewEmulator3(80, 40, 40)
				emu2 = NewEmulator3(80, 40, 40)
			}
			output.Reset()

			emu1.HandleStream(v.seq1)
			emu2.HandleStream(v.seq2)

			got := emu1.Equal(emu2)
			if got != v.expect {
				t.Errorf("%q expect %t, got %t\n", v.label, v.expect, got)
			}

			emu1.EqualTrace(emu2)
			trace := output.String()
			for i := range v.expectStr {
				if !strings.Contains(trace, v.expectStr[i]) {
					t.Errorf("%q EqualTrace() expect \n%s, \ngot \n%s\n", v.label, v.expectStr[i], trace)
				}
				// t.Logf("%s\n", trace)
			}
		})
	}
}

func TestEmulatorEqual_Caps(t *testing.T) {
	tc := []struct {
		caps1 map[int]string
		caps2 map[int]string
		label string
		log   []string
		equal bool
	}{
		{
			map[int]string{CSI_U_QUERY: "fake"},
			map[int]string{CSI_U_QUERY: "fake"},
			"same caps",
			[]string{},
			true,
		},
		{
			map[int]string{},
			map[int]string{},
			"empty caps",
			[]string{},
			true,
		},
		{
			map[int]string{CSI_U_QUERY: "fake"},
			map[int]string{},
			"empty caps",
			[]string{"caps="},
			false,
		},
	}
	var output strings.Builder
	util.Logger.CreateLogger(&output, true, slog.LevelDebug)

	for _, v := range tc {
		emu1 := NewEmulator3(80, 40, 40)
		emu1.SetTerminalCaps(v.caps1)
		emu2 := NewEmulator3(80, 40, 40)
		emu2.SetTerminalCaps(v.caps2)
		output.Reset()

		t.Run(v.label, func(t *testing.T) {
			equal := emu1.Equal(emu2)
			if equal != v.equal {
				t.Errorf("%q expect %t, got %t\n", v.label, v.equal, equal)
			}

			emu1.EqualTrace(emu2)
			trace := output.String()
			for i := range v.log {
				if !strings.Contains(trace, v.log[i]) {
					t.Errorf("%q EqualTrace() expect \n%s, \ngot \n%s\n", v.label, v.log[i], trace)
				}
			}
		})
	}
}

func TestLargeStream(t *testing.T) {
	// 读取文件内容
	file, err := os.Open("../data/regret_poem.txt")
	if err != nil {
		panic(err)
	}

	data, err := io.ReadAll(file)
	if err != nil {
		panic(err)
	}

	// fmt.Println(string(data))

	tc := []struct {
		label         string
		seq           string
		diffSuffix    string
		remainsSuffix string
	}{
		{"normal", string(data), "梨园弟子白发新，椒房阿监青娥老。\n\n", "天长地久有时尽，此恨绵绵无绝期。\n"},
	}

	// util.Logger.CreateLogger(&output, true, slog.LevelDebug)
	util.Logger.CreateLogger(os.Stdout, true, slog.LevelDebug)

	emu := NewEmulator3(80, 40, 40)

	// test empty input
	diff, remains := emu.HandleLargeStream("")
	if diff != "" || remains != "" {
		t.Errorf("%s expect empty string result, got %s,%s\n", "HandleLargeStream", diff, remains)
	}

	// test large input
	for _, v := range tc {
		t.Run(v.label, func(t *testing.T) {
			diff, remains = emu.HandleLargeStream(v.seq)
			if !strings.HasSuffix(diff, v.diffSuffix) {
				t.Errorf("%q expect diff %q, got %q\n", v.label, v.diffSuffix, diff)
			}
			if !strings.HasSuffix(remains, v.remainsSuffix) {
				t.Errorf("%q expect remains %q, got %q\n", v.label, v.remainsSuffix, remains)
			}
		})
	}

	// test previous input reach max rows
	d2, r2 := emu.HandleLargeStream(remains)
	if d2 != "" || remains != r2 {
		t.Errorf("%s reach max rows, expect diff %s, got %s\n", "HandleLargeStream", "", d2)
	}
}

func TestExcludeHandler(t *testing.T) {
	tc := []struct {
		label     string
		seq       string // input sequence for terminal (remote)
		diff      string // pass to local terminal
		writeBack string // terminal write back
		hdsSize   int
		caps      bool
	}{
		{"DSR", "\x1B[6nworld", "world", "\x1b[1;1R", 6, false},
		{"OSC 10x", "\x1B]10;?\x1B\\world", "world", "\x1b]10;rgb:0000/0000/0000\x1b\\", 6, false},
		{"VT52_ID", "\x1B[?2l\x1BZ", "\x1B[?2l", "\x1b/Z", 2, false},
		{
			"OSC 52 query selection", "\x1B]52;c0;5Zub5aeR5aiY5bGxCg==\x1B\\\x1B]52;c0;?\x1B\\",
			"\x1b]52;c0;5Zub5aeR5aiY5bGxCg==\x1b\\", "\x1b]52;c;5Zub5aeR5aiY5bGxCg==\x1b\\", 2, false,
		},
		{"CSI u query: not support", "\x1b[?u", "", "", 1, false},
		{"CSI u query: support", "\x1b[?u", "", "\x1b[?0u", 1, true},
		{"CSI u set: not support", "\x1b[=2;2u", "", "", 1, false},
		{"CSI u push: support", "\x1b[>2u", "\x1b[>2u", "", 1, true},
	}

	util.Logger.CreateLogger(io.Discard, false, slog.LevelDebug)
	// util.Logger.CreateLogger(os.Stdout, true, slog.LevelDebug)

	for _, v := range tc {
		emu := NewEmulator3(80, 40, 40)
		if v.caps {
			caps := map[int]string{CSI_U_QUERY: "support"}
			emu.SetTerminalCaps(caps)
		}
		t.Run(v.label, func(t *testing.T) {
			hds, diff := emu.HandleStream(v.seq)

			if len(hds) != v.hdsSize {
				t.Errorf("%q expct hds size %d, got %d\n", v.label, v.hdsSize, len(hds))
			}

			if diff != v.diff {
				t.Errorf("%q expct diff %q, got %q\n", v.label, v.diff, diff)
			}

			writeBack := emu.ReadOctetsToHost()
			if writeBack != v.writeBack {
				t.Errorf("%q expect write back %q, got %q\n", v.label, v.writeBack, writeBack)
			}
		})
	}
}

func TestSetLastRows(t *testing.T) {
	util.Logger.CreateLogger(io.Discard, false, slog.LevelDebug)
	// util.Logger.CreateLogger(os.Stdout, true, slog.LevelDebug)

	emu := NewEmulator3(80, 40, 40)

	// test  SetLastRows
	expect := 51
	emu.SetLastRows(expect)
	if emu.lastRows != expect {
		t.Errorf("%q expect lastRows %d, got %d\n", "SetLastRows", expect, emu.lastRows)
	}
	emu.SetLastRows(0)

	// test GetHeight
	expect = 40
	got := emu.GetHeight()
	if got != expect {
		t.Errorf("%s expect %d, got %d\n", "GetHeight", expect, got)
	}
}

func TestTerminalCaps(t *testing.T) {
	emu := NewEmulator3(80, 40, 40)
	caps := map[int]string{1: "one", 2: "two", 3: "three"}
	emu.SetTerminalCaps(caps)

	if !maps.Equal(caps, emu.caps) {
		t.Errorf("terminal caps test: expect %v, %p, got %v, %p\n", caps, caps, emu.caps, emu.caps)
	}
}
