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

	"github.com/rivo/uniseg"
)

// disable this test
// func TestUnisegCapability(t *testing.T) {
// 	s := "Chin\u0308\u0308\u0308a 🏖 is where I live. 国旗🇳🇱Fun with Flag🇧🇷."
// 	graphemes := uniseg.NewGraphemes(s)
//
// 	for graphemes.Next() {
// 		start, end := graphemes.Positions()
// 		t.Logf("%q\t 0x%X, [%d ~ %d]\n", graphemes.Runes(), graphemes.Runes(), start, end)
// 	}
// 	if uniseg.GraphemeClusterCount(s) != 43 {
// 		t.Errorf("UTF-8 string %q expect %d, got %d\n", s, uniseg.GraphemeClusterCount(s), utf8.RuneCountInString(s))
// 	}
// }
//
// disable this test
// func testCharsetResult(t *testing.T) {
// 	s := "ABCD\xe0\xe1\xe2\xe3\xe9\x9c"
// 	want := "àáâãé"
//
// 	var ret strings.Builder
//
// 	cs := Charset_IsoLatin1
// 	for i := range s {
// 		if 160 < s[i] && s[i] < 255 {
// 			ret.WriteRune(charCodes[cs][s[i]-160])
// 			t.Logf("%c %x %d in GR", s[i], s[i], s[i])
// 		} else {
// 			t.Logf("%c %x %d not in GL", s[i], s[i], s[i])
// 		}
// 	}
// 	if want != ret.String() {
// 		t.Errorf("Charset Charset_IsoLatin1 expect %s, got %s\n", want, ret.String())
// 	}
// }

func TestRunesWidth(t *testing.T) {
	tc := []struct {
		name  string
		raw   string
		width int
	}{
		{"latin    ", "long", 4},
		{"chinese  ", "中国", 4},
		{"combining", "shangha\u0308\u0308i", 8},
		{
			"emoji 1", "🏝",
			2,
		},
		{
			"emoji 2", "🏖",
			2,
		},
		{
			"flags", "🇳🇱🇧🇷",
			4,
		},
		{
			"flag 2", "🇨🇳",
			2,
		},
	}

	for _, v := range tc {
		graphemes := uniseg.NewGraphemes(v.raw)
		width := 0
		var rs []rune
		for graphemes.Next() {
			rs = graphemes.Runes()
			width += runesWidth(rs)
		}
		if v.width != width {
			t.Logf("%s :\t %q %U\n", v.name, v.raw, rs)
			t.Errorf("%s:\t %q  expect width %d, got %d\n", v.name, v.raw, v.width, width)
		}
	}
}

// func TestCharmapCapability(t *testing.T) {
// 	invalid := "ABCD\xe0\xe1\xe2\xe3\xe9\x9c" // this is "à á â ã é" in ISO-8859-1
// 	// If we convert it from ISO8859-1 to UTF-8:
// 	dec, _ := charmap.ISO8859_1.NewDecoder().String(invalid)
// 	want := "ABCDàáâãé\u009c"
//
// 	if dec != want {
// 		t.Logf("Not UTF-8: %q (valid: %v)\n", invalid, utf8.ValidString(invalid))
// 		t.Errorf("Decoded: %q (valid UTF8: %v)\n", dec, utf8.ValidString(dec))
// 	}
// }

func TestHandleGraphemes(t *testing.T) {
	tc := []struct {
		name   string
		raw    string // data stream with control sequences
		hName  string
		want   int    // handler size: star cols as print.
		cols   []int  // expect cols for cell on screen.
		result string // data stream without control sequences
	}{
		{
			"UTF-8 plain english",
			"long long ago",
			"graphemes",
			13,
			[]int{13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25},
			"long long ago",
		},
		{
			"UTF-8 chinese, combining character and flags",
			"Chin\u0308\u0308a 🏖 i国旗🇳🇱Fun 🌈with Flag🇧🇷.s",
			"graphemes", 29,
			[]int{29, 30, 31, 32, 33, 34, 35, 37, 38, 39, 41, 43, 45, 46, 47, 48, 49, 51, 52, 53, 54, 55, 56, 57, 58, 59, 60, 62, 63},
			"Chin\u0308\u0308a 🏖 i国旗🇳🇱Fun 🌈with Flag🇧🇷.s",
		},
		{
			"VT mix UTF-8",
			"中国\x1B%@\xA5AB\xe2\xe3\xe9\x1B%GShanghai\x1B%@CD\xe0\xe1",
			"graphemes",
			23,
			[]int{23, 25, 27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38, 39, 40, 41, 42, 43, 44},
			"中国¥ABâãéShanghaiCDàá",
		},
		{
			"VT edge",
			"\x1B%@Beijing\x1B%G",
			"graphemes",
			9,
			[]int{9, 10, 11, 12, 13, 14, 15},
			"Beijing",
		},
	}

	p := NewParser()
	emu := NewEmulator()
	for i, v := range tc {
		hds := make([]*Handler, 0, 16)
		hds = p.processStream(v.raw, hds)

		if len(hds) == 0 {
			t.Errorf("%s got zero handlers.", v.name)
		}

		// move to the new row
		emu.framebuffer.DS.MoveRow(i, false)

		// move to the start col.
		emu.framebuffer.DS.MoveCol(v.want, false, false)

		for _, hd := range hds {
			// fmt.Printf("handle ... %#v\n", hd)
			hd.handle(emu)
		}
		if v.want != len(hds) {
			t.Errorf("%s expect %d handlers,got %d handlers\n", v.name, v.want, len(hds))
		}
		// else {
		// 	t.Logf("%q end %d.\n", v.name, len(hds))
		// }

		graphemes := uniseg.NewGraphemes(v.result)
		j := 0
		for graphemes.Next() {
			// the expected content
			chs := graphemes.Runes()

			// get the cell from framebuffer
			rows := i
			cols := v.cols[j]
			cell := emu.framebuffer.GetCell(rows, cols)

			if cell.contents != string(chs) {
				t.Errorf("%s:\t [row,cols]:[%2d,%2d] expect %q, got %q\n", v.name, rows, cols, string(chs), cell.contents)
			}
			j += 1
		}
	}
}

func TestCollectNumericParameters(t *testing.T) {
	tc := []struct {
		name string
		want int
		seq  string
	}{
		{"normal number     ", 65, "65;23"},
		{"too large number  ", 0, "65536;22"},
		{"over size 16      ", 0, "1;2;3;4;5;6;7;8;9;0;1;2;3;4;5;6;"},
		{"over size 17      ", 0, "1;2;3;4;5;6;7;8;9;0;1;2;3;4;5;6;7;"},
	}

	p := NewParser()
	for _, v := range tc {
		// prepare for the condition
		p.reset()
		p.nInputOps = 1
		p.inputOps[0] = 0
		p.inputState = InputState_CSI
		// parse the number
		for _, ch := range v.seq {
			p.collectNumericParameters(ch)
			if p.inputState == InputState_Normal {
				break
			}
		}
		// only test the first number
		if p.getPs(0, 0) != v.want {
			t.Errorf("%s expect %d, got %d\n", v.name, v.want, p.getPs(0, 0))
		}
	}
}

func TestHandle_ESC_DCS(t *testing.T) {
	tc := []struct {
		name        string
		seq         string
		wantName    string
		wantIndex   int
		wantCharset *map[byte]rune
	}{
		{"VT100 G0", "\x1B(A", "esc-dcs", 0, &vt_ISO_UK},
		{"VT100 G1", "\x1B)B", "esc-dcs", 1, nil},
		{"VT220 G2", "\x1B*5", "esc-dcs", 2, nil},
		{"VT220 G3", "\x1B+%5", "esc-dcs", 3, &vt_DEC_Supplement},
		{"VT300 G1", "\x1B-0", "esc-dcs", 1, &vt_DEC_Special},
		{"VT300 G2", "\x1B.<", "esc-dcs", 2, &vt_DEC_Supplement},
		{"VT300 G3", "\x1B/>", "esc-dcs", 3, &vt_DEC_Technical},
		{"VT300 G3", "\x1B/A", "esc-dcs", 3, &vt_ISO_8859_1},
		{"ISO/IEC 2022 G0 A", "\x1B,A", "esc-dcs", 0, &vt_ISO_UK},
		{"ISO/IEC 2022 G0 >", "\x1B$>", "esc-dcs", 0, &vt_DEC_Technical},
		// for other charset, just replace it with UTF-8
		{"ISO/IEC 2022 G0 None", "\x1B$%9", "esc-dcs", 0, nil},
	}

	p := NewParser()
	emu := NewEmulator()
	for _, v := range tc {

		// set different value for compare
		for i := 0; i < 4; i++ {
			emu.charsetState.g[i] = nil
		}
		// parse the instruction
		var hd *Handler
		for _, ch := range v.seq {
			hd = p.processInput(ch)
		}
		if hd != nil {
			hd.handle(emu)

			cs := emu.charsetState.g[v.wantIndex]
			if v.wantName != hd.name || cs != v.wantCharset {
				t.Errorf("%s: [%s vs %s] expect %p, got %p", v.name, hd.name, v.wantName, v.wantCharset, cs)
			}
		} else {
			t.Errorf("%s got nil return\n", v.name)
		}
	}
}

func TestHandle_DOCS(t *testing.T) {
	tc := []struct {
		name     string
		seq      string
		wantGL   int
		wantGR   int
		wantSS   int
		wantName string
	}{
		{"DOCS utf-8    ", "\x1B%G", 0, 2, 0, "esc-docs-utf-8"},
		{"DOCS iso8859-1", "\x1B%@", 0, 2, 0, "esc-docs-iso8859-1"},
	}

	p := NewParser()
	emu := NewEmulator()
	for _, v := range tc {

		// set different value
		emu.charsetState.gl = 2
		emu.charsetState.gr = 3
		emu.charsetState.ss = 2

		for i := 0; i < 4; i++ {
			emu.charsetState.g[i] = &vt_DEC_Supplement // Charset_DecSuppl
		}

		// parse the instruction
		var hd *Handler
		for _, ch := range v.seq {
			hd = p.processInput(ch)
		}

		// call the handler
		if hd != nil {
			hd.handle(emu)

			for i := 0; i < 4; i++ {
				if i == 2 {
					// skip the g[2], which is iso8859-1 for 'ESC % @'
					continue
				}
				if emu.charsetState.g[i] != nil {
					t.Errorf("%s charset g1~g4 should be utf-8.", v.name)
				}
			}
			// verify the result
			if emu.charsetState.gl != v.wantGL || emu.charsetState.gr != v.wantGR ||
				emu.charsetState.ss != v.wantSS || hd.name != v.wantName {
				t.Errorf("%s [%s vs %s] expect GL,GR,SS = %d,%d,%d, got %d,%d,%d\n", v.name, hd.name, v.wantName,
					v.wantGL, v.wantGR, v.wantSS, emu.charsetState.gl, emu.charsetState.gr, emu.charsetState.ss)
			}

		} else {
			t.Errorf("%s got nil return\n", v.name)
		}

	}
}

func TestHandle_LS2_LS3(t *testing.T) {
	tc := []struct {
		name     string
		seq      string
		wantName string
		want     int
	}{
		{"LS2", "\x1Bn", "c0-ls2", 2},
		{"LS3", "\x1Bo", "c0-ls3", 3},
	}

	p := NewParser()
	emu := NewEmulator()
	for _, v := range tc {

		// reset the charsetState
		emu.charsetState.gl = 0

		// parse the instruction
		var hd *Handler
		for _, ch := range v.seq {
			hd = p.processInput(ch)
		}

		// call the handler
		if hd != nil {
			hd.handle(emu)

			// verify the result
			if emu.charsetState.gl != v.want || hd.name != v.wantName {
				t.Errorf("%s [%s vs %s] expect %d, got %d\n", v.name, hd.name, v.wantName, v.want, emu.charsetState.gl)
			}

		} else {
			t.Errorf("%s got nil return\n", v.name)
		}

	}
}

func TestHandle_LS1R_LS2R_LS3R(t *testing.T) {
	tc := []struct {
		name     string
		seq      string
		wantName string
		want     int
	}{
		{"LS1R", "\x1B~", "c0-ls1r", 1},
		{"LS2R", "\x1B}", "c0-ls2r", 2},
		{"LS3R", "\x1B|", "c0-ls3r", 3},
	}

	p := NewParser()
	emu := NewEmulator()
	for _, v := range tc {

		// reset the charsetState
		emu.charsetState.gr = 0

		// parse the instruction
		var hd *Handler
		for _, ch := range v.seq {
			hd = p.processInput(ch)
		}

		// call the handler
		if hd != nil {
			hd.handle(emu)

			// verify the result
			if emu.charsetState.gr != v.want || hd.name != v.wantName {
				t.Errorf("%s [%s vs %s] expect %d, got %d\n", v.name, hd.name, v.wantName, v.want, emu.charsetState.gr)
			}

		} else {
			t.Errorf("%s got nil return\n", v.name)
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
				t.Errorf("%s [%s vs %s] expect %d, got %d\n", v.name, hd.name, v.wantName, v.want, emu.charsetState.ss)
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
		{"OSC 2;Pt BEL unusual", "osc 0,1,2", false, true, "\x1B]2;[neovim]\x1B78\x07", "[neovim]\x1B78"},
		{"OSC 0;Pt malform 1  ", "osc 0,1,2", true, true, "\x1B]ada\x07", ""},
		{"OSC 0;Pt malform 2  ", "osc 0,1,2", true, true, "\x1B]7fy;ada\x07", ""},
	}

	p := NewParser()
	emu := NewEmulator()
	for _, v := range tc {
		var hd *Handler
		p.reset()
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
				t.Errorf("%s name=%q seq=%q expect %q\n got window title=%q\n got icon name=%q\n",
					v.name, v.wantName, v.seq, v.wantTitle, windowTitle, iconName)
			}
		} else {
			if p.inputState == InputState_Normal && v.wantTitle == "" {
				continue
			}
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
		{"HT 1  ", 5, "c0-ht", 8, "\x09"},
		{"HT 2  ", 9, "c0-ht", 16, "\x09"},
		{"CBT   ", 29, "csi-cbt", 8, "\x1B[3Z"},
		{"CHT   ", 2, "csi-cht", 32, "\x1B[4I"},
		// reach the right edge
		{"CHT -1", 59, "csi-cht", 79, "\x1B[4I"},
		// reach the left edge
		{"CBT  0", 2, "csi-cbt", 0, "\x1B[3Z"},
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
