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

// TODO add test for other charset
func testHandleGraphicChar(t *testing.T) {
	tc := []struct {
		name  string
		raw   string
		hName string
	}{
		{"normal latin", "eng", "graphic-char"},
		{"chinese", "世界", "graphic-char"},
		{"GR char", "\xA5", "graphic-char"},
	}

	hds := make([]*Handler, 0, 16)
	p := NewParser()
	emu := NewEmulator()
	for _, v := range tc {
		for _, ch := range v.raw {
			hd := p.processInput(ch)
			if hd != nil {
				hds = append(hds, hd)
			}
		}

		for _, hd := range hds {
			hd.handle(emu)
		}
	}
}

func TestHandle_SS2_SS3(t *testing.T) {
	tc := []struct {
		name     string
		seq      string
		wantName string
		want     int
	}{
		{"SS2", "\x1BN", "c0-ss2", 2},
		{"SS3", "\x1BO", "c0-ss3", 3},
	}

	p := NewParser()
	emu := NewEmulator()
	for _, v := range tc {

		// reset the charsetState
		emu.charsetState.ss = 0

		// parse the instruction
		var hd *Handler
		for _, ch := range v.seq {
			hd = p.processInput(ch)
		}

		// call the handler
		if hd != nil {
			hd.handle(emu)

			// verify the result
			if emu.charsetState.ss != v.want || hd.name != v.wantName {
				t.Errorf("%s [%s vs %s ]expect %d, got %d\n", v.name, hd.name, v.wantName, v.want, emu.charsetState.ss)
			}

		} else {
			t.Errorf("%s got nil return\n", v.name)
		}

	}
}

func TestHandle_SO_SI(t *testing.T) {
	tc := []struct {
		name string
		r    rune
		want int
	}{
		{"SO", 0x0E, 1},
		{"SI", 0x0F, 0},
	}

	p := NewParser()
	emu := NewEmulator()
	for _, v := range tc {
		hd := p.processInput(v.r)
		if hd != nil {
			hd.handle(emu)

			if emu.charsetState.gl != v.want {
				t.Errorf("%s expect %d, got %d\n", v.name, v.want, emu.charsetState.gl)
			}

		} else {
			t.Errorf("%s got nil return\n", v.name)
		}

	}
}

func TestHandle_CUU_CUD_CUF_CUB_CUP(t *testing.T) {
	tc := []struct {
		name     string
		startX   int
		startY   int
		wantName string
		wantX    int
		wantY    int
		seq      string
	}{
		{"CSI Ps;PsH", 10, 10, "cup", 13, 23, "\x1B[24;14H"},
		{"CSI Ps;Psf", 10, 10, "cup", 41, 20, "\x1B[21;42f"},
		{"CSI Ps A  ", 10, 20, "cuu", 10, 14, "\x1B[6A"},
		{"CSI Ps B  ", 10, 10, "cud", 10, 13, "\x1B[3B"},
		{"CSI Ps C  ", 10, 10, "cuf", 12, 10, "\x1B[2C"},
		{"CSI Ps D  ", 20, 10, "cub", 12, 10, "\x1B[8D"},
		{"BS        ", 12, 12, "cub", 11, 12, "\x08"},
		{"CUB       ", 12, 12, "cub", 11, 12, "\x1B[1D"},
		{"BS agin   ", 11, 12, "cub", 10, 12, "\x08"},
	}
	p := NewParser()

	for _, v := range tc {
		var hd *Handler
		emu := NewEmulator()

		// parse the sequence
		for _, ch := range v.seq {
			hd = p.processInput(ch)
		}
		if hd != nil {

			// set the start position
			emu.framebuffer.DS.MoveRow(v.startY, false)
			emu.framebuffer.DS.MoveCol(v.startX, false, false)

			// handle the instruction
			hd.handle(emu)

			// get the result
			gotY := emu.framebuffer.DS.GetCursorRow()
			gotX := emu.framebuffer.DS.GetCursorCol()

			if gotX != v.wantX || gotY != v.wantY || hd.name != v.wantName {
				t.Errorf("%s [%s vs %s] expect cursor position (%d,%d), got (%d,%d)\n",
					v.name, v.wantName, hd.name, v.wantX, v.wantY, gotX, gotY)
			}
		} else {
			t.Errorf("%s got nil return\n", v.name)
		}

	}
}

func TestHandle_OSC_0_1_2(t *testing.T) {
	tc := []struct {
		name      string
		wantName  string
		icon      bool
		title     bool
		seq       string
		wantTitle string
	}{
		{"OSC 0;Pt BEL        ", "osc 0,1,2", true, true, "\x1B]0;ada\x07", "ada"},
		{"OSC 1;Pt 7bit ST    ", "osc 0,1,2", true, false, "\x1B]1;adas\x1B\\", "adas"},
		{"OSC 2;Pt BEL chinese", "osc 0,1,2", false, true, "\x1B]2;[道德经]\x07", "[道德经]"},
	}

	p := NewParser()
	emu := NewEmulator()
	for _, v := range tc {
		var hd *Handler

		// parse the sequence
		for _, ch := range v.seq {
			hd = p.processInput(ch)
		}

		if hd != nil {
			// handle the instruction
			hd.handle(emu)

			// get the result
			windowTitle := emu.framebuffer.windowTitle
			iconName := emu.framebuffer.iconName

			if v.title && v.icon && windowTitle == v.wantTitle && iconName == v.wantTitle &&
				hd.name == v.wantName {
				continue
			} else if v.icon && iconName == v.wantTitle && hd.name == v.wantName {
				continue
			} else if v.title && windowTitle == v.wantTitle && hd.name == v.wantName {
				continue
			} else {
				t.Errorf("%s name=%q seq=%q expect %s\n got window title=%s\n got icon name=%s\n",
					v.name, v.wantName, v.seq, v.wantTitle, windowTitle, iconName)
			}
		} else {
			t.Errorf("%s got nil return\n", v.name)
		}
	}
}

func TestHandle_BEL(t *testing.T) {
	p := NewParser()
	emu := NewEmulator()

	// process the bell sequence
	hd := p.processInput('\x07')

	if hd != nil {
		// handle the bell
		hd.handle(emu)

		// theck the handler name and bell count
		bellCount := emu.framebuffer.GetBellCount()
		if bellCount == 0 || hd.name != "c0-bel" {
			t.Errorf("BEL expect %d, got %d", 1, bellCount)
		}
	} else {
		t.Errorf("%s got nil return\n", hd.name)
	}
}

func TestHandle_HTS_TBC(t *testing.T) {
	tc := []struct {
		name     string
		position int
		setSeq   string
		clearSeq string
	}{
		{"Set/Clear tab stop 1", 18, "\x1BH", "\x1B[g"},
		{"Set/Clear tab stop 2", 38, "\x1BH", "\x1B[0g"},
		{"Set/Clear tab stop 3", 48, "\x1BH", "\x1B[3g"},
	}

	p := NewParser()
	var hd *Handler
	emu := NewEmulator()
	for _, v := range tc {

		// set the start position
		emu.framebuffer.DS.MoveRow(2, false)
		emu.framebuffer.DS.MoveCol(v.position, false, false)

		// set the tab stop position
		for _, ch := range v.setSeq {
			hd = p.processInput(ch)
		}
		if hd == nil {
			t.Errorf("%s Set got nil return\n", v.name)
			continue
		}
		hd.handle(emu)

		// verify the position is set == true
		if !emu.framebuffer.DS.tabs[v.position] {
			t.Errorf("%s expect true, got %t\n", v.name, false)
		}

		// set the tab stop position
		for _, ch := range v.clearSeq {
			hd = p.processInput(ch)
		}

		if hd == nil {
			t.Errorf("%s Clear got nil return\n", v.name)
			continue
		}
		hd.handle(emu)

		// verify the position is set == false
		if emu.framebuffer.DS.tabs[v.position] {
			t.Errorf("%s expect false, got %t\n", v.name, true)
		}
	}
}

func TestHandle_HT_CHT_CBT(t *testing.T) {
	tc := []struct {
		name     string
		startX   int
		wantName string
		wantX    int
		ctlseq   string
	}{
		{"HT 1 ", 5, "c0-ht", 8, "\x09"},
		{"HT 2 ", 9, "c0-ht", 16, "\x09"},
		{"CBT  ", 29, "csi-cbt", 8, "\x1B[3Z"},
		{"CHT  ", 2, "csi-cht", 32, "\x1B[4I"},
	}

	p := NewParser()
	var hd *Handler
	emu := NewEmulator()
	for _, v := range tc {

		for _, ch := range v.ctlseq {
			hd = p.processInput(ch)
		}
		// set the start position
		emu.framebuffer.DS.MoveRow(2, false)
		emu.framebuffer.DS.MoveCol(v.startX, false, false)

		// handle the instruction
		hd.handle(emu)

		// get the result
		if hd != nil {
			gotX := emu.framebuffer.DS.GetCursorCol()

			if gotX != v.wantX || hd.name != v.wantName {
				t.Errorf("%s [%s vs %s] expect cursor cols=%d, got %d)\n", v.name, v.wantName, hd.name, v.wantX, gotX)
			}
		} else {
			t.Errorf("%s got nil return\n", v.name)
		}

	}
}

// TODO test the HT handler
func TestHandle_CR_LF_VT_FF(t *testing.T) {
	tc := []struct {
		name     string
		startX   int
		startY   int
		wantName string
		wantX    int
		wantY    int
		ctlseq   string
	}{
		{"CR 1 ", 1, 2, "c0-cr", 0, 2, "\x0D"},
		{"CR 2 ", 9, 4, "c0-cr", 0, 4, "\x0D"},
		{"LF   ", 1, 2, "c0-lf", 1, 3, "\x0C"},
		{"VT   ", 2, 3, "c0-lf", 2, 4, "\x0B"},
		{"FF   ", 3, 4, "c0-lf", 3, 5, "\x0C"},
		{"ESC D", 4, 5, "c0-lf", 4, 6, "\x1BD"},
		//{"HT 1 ", 5, 2, "c0-ht", 15, 2, "\x09"},
		//{"HT 2 ", 3, 2, "c0-ht", 7, 2, "\x09"},
	}

	p := NewParser()
	var hd *Handler
	emu := NewEmulator()
	for _, v := range tc {

		for _, ch := range v.ctlseq {
			hd = p.processInput(ch)
		}
		// set the start position
		emu.framebuffer.DS.MoveRow(v.startY, false)
		emu.framebuffer.DS.MoveCol(v.startX, false, false)

		// handle the instruction
		hd.handle(emu)

		// get the result
		if hd != nil {
			gotY := emu.framebuffer.DS.GetCursorRow()
			gotX := emu.framebuffer.DS.GetCursorCol()

			if gotX != v.wantX || gotY != v.wantY || hd.name != v.wantName {
				t.Errorf("%s [%s vs %s] expect cursor position (%d,%d), got (%d,%d)\n",
					v.name, v.wantName, hd.name, v.startX, v.wantY, gotX, gotY)
			}
		} else {
			t.Errorf("%s got nil return\n", v.name)
		}

	}
}
