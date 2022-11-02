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
