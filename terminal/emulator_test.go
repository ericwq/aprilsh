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
	"io"
	"strings"
	"testing"
)

func TestEmulatorResize(t *testing.T) {
	type Result struct {
		nCols, nRows      int
		hMargin, nColsEff int
	}
	tc := []struct {
		name                string
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
		emu.altScreenBufferMode = v.altScreenBufferMode
		emu.horizMarginMode = v.horizMarginMode
		emu.posY = v.posY
		emu.resize(v.nCols, v.nRows)

		if v.expect.nCols != emu.nCols || v.expect.nRows != emu.nRows ||
			v.expect.hMargin != emu.hMargin || v.expect.nColsEff != emu.nColsEff {
			t.Errorf("%q expect %v, got (%d,%d,%d,%d)\n", v.name, v.expect, emu.nCols, emu.nRows, emu.hMargin, emu.nColsEff)
		}
	}
}

func TestReadOctetsToHost(t *testing.T) {
	tc := []struct {
		name   string
		rawStr []string
		expect string
	}{
		{"one sequence", []string{"\x1B[23m"}, "\x1B[23m"},
		{"three mix sequence", []string{"\x1B[24;14H", "\x1B[3g", "长"}, "\x1B[24;14H\x1B[3g长"},
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

func TestHandleStreamEmpty(t *testing.T) {
	emu := NewEmulator3(80, 40, 0)
	hds := emu.HandleStream("")
	if len(hds) != 0 {
		t.Errorf("#test HandleStream with empty input should zero result, got %v\n", hds)
	}
}

func TestNormalizeCursorPos(t *testing.T) {
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

func TestJumpToNextTabStop(t *testing.T) {
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

func TestLookupCharset(t *testing.T) {
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

func TestPasteSelection(t *testing.T) {
	tc := []struct {
		label              string
		bracketedPasteMode bool
		selection          string
		expect             string
	}{
		{"bracketedPasteMode is false", false, "lock down", "lock down"},
		{"bracketedPasteMode is true, english ", true, "lock down", "\x1b[200~lock down\x1b[201~"},
		{"bracketedPasteMode is true, chinese ", true, "解除封控", "\x1b[200~解除封控\x1b[201~"},
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

func TestHasFocus(t *testing.T) {
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

	emu.SetLogTraceOutput(io.Discard)
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
		if strings.Contains(v.label, "origin mode") {
			// set origin mode, top/bottom margin, horizontal margin
			emu.HandleStream("\x1B[?6h\x1B[2;38r\x1B[?69h\x1B[2;68s")
			// fmt.Printf("#test originMode=%d, top=%d, bottom=%d\n", emu.originMode, emu.marginTop, emu.marginBottom)
			// fmt.Printf("#test horizMarginMode=%t, hMargin=%d, nColsEff=%d\n", emu.horizMarginMode, emu.hMargin, emu.nColsEff)
		}
		emu.MoveCursor(v.posY, v.posX)

		if emu.posY != v.expectY || emu.posX != v.expectX {
			t.Errorf("%q expect cursor position (%d,%d), got (%d,%d)\n", v.label, v.expectY, v.expectX, emu.posY, emu.posX)
		}
	}
}

func TestCompleteSetCursorVisible(t *testing.T) {
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

func TestCompletePrefixWindowTitle(t *testing.T) {
	emu := NewEmulator3(80, 40, 40)

	base := "base title"
	prefix := "前缀"
	emu.cf.setTitleInitialized()
	emu.cf.setWindowTitle(base)

	emu.PrefixWindowTitle(prefix)

	expect := prefix + base
	got := emu.cf.getWindowTitle()

	if got != expect {
		t.Errorf("#test PrefixWindowTitle() expect %q, got %q\n", expect, got)
	}
}

func TestCompleteGetCell(t *testing.T) {
	tc := []struct {
		label      string
		seq        string
		posY, posX int
		contents   string
	}{
		{"in the middle", "\x1B[11;74Houtput for normal wrap line.", 10, 73, "o"},
		{"in the last cols", "", 10, 79, " "},
		{"in the first cols", "", 11, 0, "f"},
	}

	emu := NewEmulator3(80, 40, 40)

	emu.SetLogTraceOutput(io.Discard)

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
