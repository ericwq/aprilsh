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
	"fmt"
	"strings"
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

func TestProcessInputEmpty(t *testing.T) {
	p := NewParser()
	var hd *Handler
	var chs []rune

	hd = p.processInput(chs...)
	if hd != nil {
		t.Errorf("processInput expect empty, got %s\n", hd.name)
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

func TestHandle_Graphemes(t *testing.T) {
	tc := []struct {
		name   string
		raw    string // data stream with control sequences
		hName  string
		want   int    // handler size: start cols as print. it's also used as start column.
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

func TestHandle_Graphemes_Wrap(t *testing.T) {
	tc := []struct {
		name string
		raw  string
		y, x int
		cols []int
	}{
		{
			"plain english wrap",
			"ap\u0308rish",
			7, 78,
			[]int{78, 79, 0, 1, 2, 3},
		},
		{
			"chinese even wrap",
			"@@四姑娘山",
			8, 78,
			[]int{78, 79, 0, 2, 4, 6},
		},
		{
			"chinese odd wrap",
			"#海螺沟",
			9, 78,
			[]int{78, 0, 2, 4, 6},
		},
		{
			"insert wrap",
			"#th#",
			7, 77,
			[]int{77, 78, 79, 0},
		},
	}

	p := NewParser()
	p.logTrace = true
	emu := NewEmulator()
	for _, v := range tc {
		// p.logT.Println("A0")
		hds := make([]*Handler, 0, 16)
		hds = p.processStream(v.raw, hds)

		// p.logT.Println("A1")
		if len(hds) == 0 {
			t.Errorf("%s got zero handlers.", v.name)
		}

		// move to the start row/col
		emu.framebuffer.DS.MoveRow(v.y, false)
		emu.framebuffer.DS.MoveCol(v.x, false, false)

		// enable insert mode for this test case
		if strings.HasPrefix(v.name, "insert") {
			emu.framebuffer.DS.InsertMode = true
		}

		// p.logT.Println("A2")
		for _, hd := range hds {
			hd.handle(emu)
		}

		// p.logT.Println("A3")
		row1 := emu.framebuffer.GetRow(v.y)
		row2 := emu.framebuffer.GetRow(v.y + 1)
		p.logT.Printf("%s\n", row1.String())
		p.logT.Printf("%s\n", row2.String())

		graphemes := uniseg.NewGraphemes(v.raw)
		rows := v.y
		index := 0
		for graphemes.Next() {
			// the expected content
			chs := graphemes.Runes()

			// get the cell from framebuffer
			cols := v.cols[index]
			if cols == 0 { // wrap
				rows += 1
			}
			cell := emu.framebuffer.GetCell(rows, cols)

			if cell.contents != string(chs) {
				t.Errorf("%s:\t [row,cols]:[%2d,%2d] expect %q, got %q\n", v.name, rows, cols, string(chs), cell.contents)
			}

			index += 1
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
		{"LS2", "\x1Bn", "esc-ls2", 2},
		{"LS3", "\x1Bo", "esc-ls3", 3},
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
		{"LS1R", "\x1B~", "esc-ls1r", 1},
		{"LS2R", "\x1B}", "esc-ls2r", 2},
		{"LS3R", "\x1B|", "esc-ls3r", 3},
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
		{"SS2", "\x1BN", "esc-ss2", 2}, // G2 single shift
		{"SS3", "\x1BO", "esc-ss3", 3}, // G3 single shift
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
		{"SO", 0x0E, 1}, // G1 as GL
		{"SI", 0x0F, 0}, // G0 as GL
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

func TestHandle_OSC_0_Overflow(t *testing.T) {
	p := NewParser()
	// parameter Ps overflow: >120
	seq := "\x1B]121;home\x07"
	var hd *Handler

	// parse the sequence
	for _, ch := range seq {
		hd = p.processInput(ch)
	}

	if hd != nil {
		t.Errorf("OSC overflow: %q should return nil.", seq)
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

func TestHandle_RI_NEL(t *testing.T) {
	tc := []struct {
		name     string
		startY   int // startX is always 5
		seq      string
		wantX    int
		wantY    int
		wantName string
	}{
		{"RI ", 10, "\x1BM", 5, 9, "esc-ri"},   // move cursor up to the previouse row, may scroll up
		{"NEL", 10, "\x1BE", 0, 11, "esc-nel"}, // move cursor down to next row, may scroll down
	}

	p := NewParser()
	var hd *Handler
	emu := NewEmulator()
	for _, v := range tc {

		for _, ch := range v.seq {
			hd = p.processInput(ch)
		}
		// set the start position
		emu.framebuffer.DS.MoveRow(v.startY, false)
		emu.framebuffer.DS.MoveCol(5, false, false)

		// get the result
		if hd != nil {
			// handle the instruction
			hd.handle(emu)

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

func TestHandle_HTS_TBC(t *testing.T) {
	tc := []struct {
		name     string
		position int
		setSeq   string
		clearSeq string
	}{
		{"Set/Clear tab stop 1", 18, "\x1BH", "\x1B[g"}, // set tab stop; clear tab stop
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
		{"HT 1  ", 5, "c0-ht", 8, "\x09"},       // move to the next tab stop
		{"HT 2  ", 9, "c0-ht", 16, "\x09"},      // move to the next tab stop
		{"CBT   ", 29, "csi-cbt", 8, "\x1B[3Z"}, // move backward to the previous N tab stop
		{"CHT   ", 2, "csi-cht", 32, "\x1B[4I"}, // move to the next N tab stop
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

		// get the result
		if hd != nil {
			// handle the instruction
			hd.handle(emu)
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

func TestHandle_ENQ_CAN_SUB_ESC(t *testing.T) {
	tc := []struct {
		name      string
		seq       string
		nInputOps int
		state     int
	}{
		{"ENQ ", "\x05", 0, InputState_Normal},         // ENQ - Enquiry, ignore
		{"CAN ", "\x1B\x18", 0, InputState_Normal},     // CAN and SUB interrupts ESC sequence
		{"SUB ", "\x1B\x1A", 0, InputState_Normal},     // CAN and SUB interrupts ESC sequence
		{"ESC ", "\x1B\x1B", 1, InputState_Escape},     // ESC restarts ESC sequence
		{"ESC ST ", "\x1B\\", 0, InputState_Normal},    // lone ST
		{"ESC unknow ", "\x1Bx", 0, InputState_Normal}, // unhandled ESC sequence
	}

	p := NewParser()
	p.logTrace = true // open the trace
	var hd *Handler
	for _, v := range tc {
		for _, ch := range v.seq {
			hd = p.processInput(ch)
		}

		if hd == nil {
			if p.inputState != v.state || p.nInputOps != v.nInputOps {
				t.Errorf("%s seq=%q expect state=%s, nInputOps=%d, got state=%s, nInputOps=%d\n",
					v.name, v.seq, strInputState[v.state], v.nInputOps, strInputState[p.inputState], p.nInputOps)
			}
		} else {
			t.Errorf("%s should get nil handler\n", v.name)
		}
	}
}

func TestHandle_DECALN_RIS(t *testing.T) {
	tc := []struct {
		name     string
		seq      string
		y, x     int
		wantName string
		want     string
	}{
		{"ESC DECLAN", "\x1B#8", 10, 10, "esc-decaln", "E"}, // the whole screen is filled with 'E'
		{"ESC RIS   ", "\x1Bc", 10, 10, "esc-ris", ""},      // after reset, the screen is empty
	}

	p := NewParser()
	p.logTrace = true // open the trace
	var hd *Handler
	emu := NewEmulator()
	for _, v := range tc {
		for _, ch := range v.seq {
			hd = p.processInput(ch)
		}

		if hd != nil {
			hd.handle(emu)

			theCell := emu.framebuffer.GetCell(v.y, v.x)
			if v.want != theCell.contents || hd.name != v.wantName {
				t.Errorf("%s:\t [%s vs %s] expect (10,10) %q, got %q",
					v.name, v.wantName, hd.name, v.want, theCell.contents)
			}
		} else {
			t.Errorf("%s expect valid Handler, got nil", v.name)
		}
	}
}

func TestHandle_DECSC_DECRC(t *testing.T) {
	tc := []struct {
		name         string
		seq          string
		wantName     string
		y, x         int
		autoWrapMode bool
		originMode   bool
	}{
		{"ESC DECSC", "\x1B7", "esc-decsc", 8, 8, false, true},
		{"ESC DECRC", "\x1B8", "esc-decrc", 18, 18, true, false},
	}

	p := NewParser()
	p.logTrace = true // open the trace
	var hd *Handler
	emu := NewEmulator()

	for i, v := range tc {
		// 1st round: set the inital position for cursor
		// 2nd round: set the different position for cursor
		emu.framebuffer.DS.MoveCol(v.x, false, false)
		emu.framebuffer.DS.MoveRow(v.y, false)

		// 1st round: set the mode value for emulator
		// 2nd round: set the different mode value for emulator
		emu.framebuffer.DS.AutoWrapMode = v.autoWrapMode
		emu.framebuffer.DS.OriginMode = v.originMode

		// save the cursor data
		for _, ch := range v.seq {
			hd = p.processInput(ch)
		}

		if hd != nil {
			hd.handle(emu)
			if hd.name != v.wantName {
				t.Errorf("%s:\t %q expect %s, got %s\n", v.name, v.seq, v.wantName, hd.name)
			}
		} else {
			t.Errorf("%s got nil Handler.", v.name)
		}

		fb := emu.framebuffer
		v = tc[0]
		// we only validate the result after the second instruction.
		if i == 1 && (fb.DS.GetCursorCol() != v.x || fb.DS.GetCursorRow() != v.y ||
			fb.DS.AutoWrapMode != v.autoWrapMode || fb.DS.OriginMode != v.originMode) {
			t.Errorf("%s %q expect (%d,%d,%t,%t), got (%d,%d,%t,%t)", v.name, v.seq,
				v.y, v.x, v.autoWrapMode, v.originMode,
				fb.DS.GetCursorRow(), fb.DS.GetCursorCol(), fb.DS.AutoWrapMode, fb.DS.OriginMode)
		}
	}
}

func TestHandle_DA1_DA2(t *testing.T) {
	tc := []struct {
		name     string
		seq      string
		want     string
		wantName string
	}{
		{"Primary DA  ", "\x1B[c", fmt.Sprintf("\x1B[?%s", DEVICE_ID), "csi-da1"},
		{"Secondary DA", "\x1B[>c", "\x1B[>64;0;0c", "csi-da2"},
	}

	p := NewParser()
	p.logTrace = true // open the trace
	var hd *Handler
	emu := NewEmulator()

	for _, v := range tc {
		// reset the target content
		emu.dispatcher.terminalToHost.Reset()

		// parse the sequence
		for _, ch := range v.seq {
			hd = p.processInput(ch)
		}

		// execute the sequence handler
		if hd != nil {
			hd.handle(emu)
			if hd.name != v.wantName {
				t.Errorf("%s:\t %q expect %s, got %s\n", v.name, v.seq, v.wantName, hd.name)
			}
		} else {
			t.Errorf("%s got nil Handler.", v.name)
		}

		got := emu.dispatcher.terminalToHost.String()
		if v.want != got {
			t.Errorf("%s seq:%q expect %q, got %q\n", v.name, v.seq, v.want, got)
		}
	}
}

// use DECALN to fill the screen, then call ED to erase part of it.
func TestHandle_ED_IL_DL(t *testing.T) {
	tc := []struct {
		name             string
		wantName         string
		activeY, activeX int
		emptyY1, emptyX1 int
		emptyY2, emptyX2 int
		seq              string
	}{
		{"ED erase below @ 20,10  ", "csi-ed", 20, 10, 20, 10, 39, 79, "\x1B#8\x1B[J"},  // Erase Below (default).
		{"ED erase below @ 35,20  ", "csi-ed", 35, 20, 35, 20, 39, 79, "\x1B#8\x1B[0J"}, // Ps = 0  ⇒  Erase Below (default).
		{"ED erase above @ 12,5   ", "csi-ed", 12, 5, 0, 0, 12, 5, "\x1B#8\x1B[1J"},     // Ps = 1  ⇒  Erase Above.
		{"ED erase all            ", "csi-ed", 42, 5, 0, 0, 39, 79, "\x1B#8\x1B[2J"},    // Ps = 2  ⇒  Erase All.
		{"IL 1 lines @ 34,2 mid   ", "csi-il", 34, 2, 34, 0, 34, 79, "\x1B#8\x1B[L"},
		{"IL 2 lines @ 39,2 bottom", "csi-il", 39, 2, 39, 0, 39, 79, "\x1B#8\x1B[2L"},
		{"IL 5 lines @ 0,2 top    ", "csi-il", 0, 2, 0, 0, 4, 79, "\x1B#8\x1B[5L"},
		{"DL 5 lines @ 5,2 top    ", "csi-dl", 5, 2, 35, 0, 39, 79, "\x1B#8\x1B[5M"},
		{"DL 5 lines @ 36,2 bottom", "csi-dl", 36, 2, 36, 0, 39, 79, "\x1B#8\x1B[5M"},
	}

	p := NewParser()
	// the default size of emu is 80x40 [colxrow]
	emu := NewEmulator()
	for _, v := range tc {

		hds := make([]*Handler, 0, 16)
		hds = p.processStream(v.seq, hds)

		if len(hds) == 0 {
			t.Errorf("%s got zero handlers.", v.name)
		}

		// move cursor to the active row
		emu.framebuffer.DS.MoveRow(v.activeY, false)
		emu.framebuffer.DS.MoveCol(v.activeX, false, false)
		for i, hd := range hds {
			hd.handle(emu)
			if i == 1 && hd.name != v.wantName {
				t.Errorf("%s:\t %q expect %s, got %s\n", v.name, v.seq, v.wantName, hd.name)
			}
		}

		// prepare the validate tools
		ds := emu.framebuffer.DS
		isEmpty := func(row, col int) bool {
			return inRange(v.emptyY1, v.emptyX1, v.emptyY2, v.emptyX2, row, col, 80)
		}

		// validate the whole screen.
		for i := 0; i < ds.GetHeight(); i++ {
			// print the row
			row := emu.framebuffer.GetRow(i)
			p.logT.Printf("%2d %s\n", i, row.String())

			// validate the cell should be empty
			for j := 0; j < ds.GetWidth(); j++ {
				cell := emu.framebuffer.GetCell(i, j)
				if isEmpty(i, j) && cell.contents == "E" {
					t.Errorf("%s seq=%q expect empty cell at (%d,%d), got %q.\n", v.name, v.seq, i, j, cell.contents)
				} else if !isEmpty(i, j) && cell.contents == "" {
					t.Errorf("%s seq=%q expect 'E' cell at (%d,%d), got empty.\n", v.name, v.seq, i, j)
				}
			}
		}
	}
}

// if the y,x is in the range, return true, otherwise return false
func inRange(startY, startX, endY, endX, y, x, width int) bool {
	pStart := startY*width + startX
	pEnd := endY*width + endX

	p := y*width + x

	if pStart <= p && p <= pEnd {
		return true
	}
	return false
}

func fillRowWith(row *Row, r rune) {
	for i := range row.cells {
		row.cells[i].contents = string(r)
	}
}

func TestHandle_ICH_EL_DCH_ECH(t *testing.T) {
	tc := []struct {
		name       string
		wantName   string
		seq        string
		startY     int // start Y
		startX     int // start X
		blankStart int // count number
		blankEnd   int // count number
	}{
		{"ICH  left side", "csi-ich", "\x1B[2@", 7, 0, 0, 1},      // insert 2 cell at col 0~1
		{"ICH right side", "csi-ich", "\x1B[3@", 8, 78, 78, 79},   // insert 3 cell at col 78
		{"ICH in  middle", "csi-ich", "\x1B[10@", 9, 40, 40, 49},  // insert 10 cell at col 40
		{"   EL to right", "csi-el", "\x1B[0K", 10, 9, 9, 79},     // erase to right from col 9
		{"   EL  to left", "csi-el", "\x1B[1K", 11, 9, 0, 9},      // erase to left from col 9
		{"   EL      all", "csi-el", "\x1B[2K", 12, 9, 0, 79},     // erase the how line
		{"  DCH  at left", "csi-dch", "\x1B[2P", 20, 9, 78, 79},   // delete 2 cell at 9 col
		{"  DCH at right", "csi-dch", "\x1B[3P", 21, 77, 77, 79},  // delete 3 cell at 77 col
		{" DCH in middle", "csi-dch", "\x1B[20P", 22, 40, 60, 79}, // delete 20 cell at 40 col
		{" ECH in middle", "csi-ech", "\x1B[2X", 30, 40, 40, 41},  // erase 2 cell at col 40
		{"   ECH at left", "csi-ech", "\x1B[5X", 30, 1, 1, 5},     // erase 5 cell at col 1
		{"  ECH at right", "csi-ech", "\x1B[5X", 30, 76, 76, 79},  // erase 5 cell at col 76
	}
	p := NewParser()
	// the default size of emu is 80x40 [colxrow]
	emu := NewEmulator()
	for _, v := range tc {

		hds := make([]*Handler, 0, 16)
		hds = p.processStream(v.seq, hds)

		if len(hds) == 0 {
			t.Errorf("%s got zero handlers.", v.name)
		}

		// fill the row with content
		row := emu.framebuffer.GetRow(v.startY)
		fillRowWith(row, 'H')

		// move cursor to the active row
		emu.framebuffer.DS.MoveRow(v.startY, false)
		emu.framebuffer.DS.MoveCol(v.startX, false, false)

		// call the handler
		for _, hd := range hds {
			hd.handle(emu)
			if hd.name != v.wantName {
				t.Errorf("%s:\t %q expect %s, got %s\n", v.name, v.seq, v.wantName, hd.name)
			}
		}

		// print the row
		p.logT.Printf("%2d %s\n", v.startY, row.String())

		// prepare the validate tool
		isEmpty := func(col int) bool {
			return inRange(v.startY, v.blankStart, v.startY, v.blankEnd, v.startY, col, 80)
		}

		// validate the result
		for col := 0; col < emu.framebuffer.DS.width; col++ {
			cell := emu.framebuffer.GetCell(v.startY, col)
			if isEmpty(col) && cell.contents == "H" {
				t.Errorf("%s seq=%q cols=%d expect empty cell, got 'H' cell\n", v.name, v.seq, col)
			} else if !isEmpty(col) && cell.contents == "" {
				t.Errorf("%s seq=%q cols=%d expect 'H' cell, got empty cell\n", v.name, v.seq, col)
			}
		}
	}
}

func TestHandle_SU_SD(t *testing.T) {
	tc := []struct {
		name             string
		wantName         string
		activeY, activeX int
		emptyY1, emptyX1 int
		emptyY2, emptyX2 int
		seq              string
	}{
		{"SU scroll up   2 lines", "csi-su-sd", 5, 0, 38, 0, 39, 79, "\x1B[2S"},
		{"SD scroll down 3 lines", "csi-su-sd", 5, 0, 0, 0, 2, 79, "\x1B[3T"},
	}

	p := NewParser()
	// the default size of emu is 80x40 [colxrow]
	emu := NewEmulator()
	for _, v := range tc {

		hds := make([]*Handler, 0, 16)
		hds = p.processStream(v.seq, hds)

		if len(hds) == 0 {
			t.Errorf("%s got zero handlers.", v.name)
		}

		// fill the screen with different rune for each row
		ds := emu.framebuffer.DS
		for i := 0; i < ds.GetHeight(); i++ {
			row := emu.framebuffer.GetRow(i)
			fillRowWith(row, rune(0x0030+i))
		}

		// handle the control sequence
		for _, hd := range hds {
			hd.handle(emu)
			if hd.name != v.wantName {
				t.Errorf("%s:\t %q expect %s, got %s\n", v.name, v.seq, v.wantName, hd.name)
			}
		}

		// prepare the validate tools
		isEmpty := func(row, col int) bool {
			return inRange(v.emptyY1, v.emptyX1, v.emptyY2, v.emptyX2, row, col, 80)
		}

		cellIn := func(row int) string {
			return getCellAtRow(v.emptyY1, v.emptyY2, row)
		}
		// validate the whole screen.
		for i := 0; i < ds.GetHeight(); i++ {
			// print the row
			row := emu.framebuffer.GetRow(i)
			p.logT.Printf("%2d %s %s\n", i, row.String(), cellIn(i))

			// validate the cell should be empty
			for j := 0; j < ds.GetWidth(); j++ {
				cell := emu.framebuffer.GetCell(i, j)
				if isEmpty(i, j) {
					if cell.contents != "" {
						t.Errorf("%s seq=%q expect empty cell at (%d,%d), got %q.\n", v.name, v.seq, i, j, cell.contents)
					}
				} else {
					if cell.contents != cellIn(i) {
						t.Errorf("%s seq=%q expect none empty cell at (%d,%d), got %s.\n", v.name, v.seq, i, j, cellIn(i))
					}
				}
			}
		}
	}
}

// calculate the cell content in row, based on y2,y1 value
func getCellAtRow(y1, y2 int, row int) string {
	if y2 < y1 {
		return "_"
	}

	gap := y2 - y1 + 1
	if y1 == 0 {
		gap *= -1
	}

	ch := rune(0x30 + row + gap)
	return string(ch)
}
