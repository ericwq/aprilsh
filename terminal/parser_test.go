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
	// "image/color"
	"io/ioutil"
	"log"
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
	p.logE.SetOutput(ioutil.Discard)
	p.logU.SetOutput(ioutil.Discard)
	p.logT.SetOutput(ioutil.Discard)

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
	p.logE.SetOutput(ioutil.Discard)
	p.logU.SetOutput(ioutil.Discard)
	p.logT.SetOutput(ioutil.Discard)

	emu := NewEmulator()
	for i, v := range tc {
		t.Run(v.name, func(t *testing.T) {
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
		})
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
		t.Run(v.name, func(t *testing.T) {
			hds := make([]*Handler, 0, 16)
			hds = p.processStream(v.raw, hds)

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

			for _, hd := range hds {
				hd.handle(emu)
			}

			row1 := emu.framebuffer.GetRow(v.y)
			row2 := emu.framebuffer.GetRow(v.y + 1)
			t.Logf("%s\n", row1.String())
			t.Logf("%s\n", row2.String())

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
		})
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
	p.logE.SetOutput(ioutil.Discard)
	p.logU.SetOutput(ioutil.Discard)
	p.logT.SetOutput(ioutil.Discard)
	p.logTrace = true

	emu := NewEmulator()
	for _, v := range tc {
		t.Run(v.name, func(t *testing.T) {
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
		})
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
	p.logE.SetOutput(ioutil.Discard)
	p.logU.SetOutput(ioutil.Discard)
	p.logT.SetOutput(ioutil.Discard)

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
		{"CSI Ps;PsH", 10, 10, "csi-cup", 13, 23, "\x1B[24;14H"},
		{"CSI Ps;Psf", 10, 10, "csi-cup", 41, 20, "\x1B[21;42f"},
		{"CSI Ps A  ", 10, 20, "csi-cuu", 10, 14, "\x1B[6A"},
		{"CSI Ps B  ", 10, 10, "csi-cud", 10, 13, "\x1B[3B"},
		{"CSI Ps C  ", 10, 10, "csi-cuf", 12, 10, "\x1B[2C"},
		{"CSI Ps D  ", 20, 10, "csi-cub", 12, 10, "\x1B[8D"},
		{"BS        ", 12, 12, "csi-cub", 11, 12, "\x08"},
		{"CUB       ", 12, 12, "csi-cub", 11, 12, "\x1B[1D"},
		{"BS agin   ", 11, 12, "csi-cub", 10, 12, "\x08"},
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
		{"OSC 0;Pt BEL        ", "osc-0,1,2", true, true, "\x1B]0;ada\x07", "ada"},
		{"OSC 1;Pt 7bit ST    ", "osc-0,1,2", true, false, "\x1B]1;adas\x1B\\", "adas"},
		{"OSC 2;Pt BEL chinese", "osc-0,1,2", false, true, "\x1B]2;[道德经]\x07", "[道德经]"},
		{"OSC 2;Pt BEL unusual", "osc-0,1,2", false, true, "\x1B]2;[neovim]\x1B78\x07", "[neovim]\x1B78"},
		{"OSC 0;Pt malform 1  ", "osc-0,1,2", true, true, "\x1B]ada\x07", ""},
		{"OSC 0;Pt malform 2  ", "osc-0,1,2", true, true, "\x1B]7fy;ada\x07", ""},
		{"OSC 0;Pt malform 2  ", "osc-0,1,2", true, true, "\x1B]7fy;ada\x07", ""},
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

	p.logE.SetOutput(ioutil.Discard)
	p.logU.SetOutput(ioutil.Discard)
	p.logT.SetOutput(ioutil.Discard)

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
		{"ENQ ", "\x05", 0, InputState_Normal},                 // ENQ - Enquiry, ignore
		{"CAN ", "\x1B\x18", 0, InputState_Normal},             // CAN and SUB interrupts ESC sequence
		{"SUB ", "\x1B\x1A", 0, InputState_Normal},             // CAN and SUB interrupts ESC sequence
		{"ESC ", "\x1B\x1B", 1, InputState_Escape},             // ESC restarts ESC sequence
		{"ESC ST ", "\x1B\\", 0, InputState_Normal},            // lone ST
		{"ESC unknow ", "\x1Bx", 0, InputState_Normal},         // unhandled ESC sequence
		{"ESC space unknow ", "\x1B x", 0, InputState_Normal},  // unhandled ESC ' 'x
		{"ESC # unknow ", "\x1B#x", 0, InputState_Normal},      // unhandled ESC '#'x
		{"CSI ESC ", "\x1B[\x1B", 0, InputState_Normal},        // CSI + ESC
		{"CSI GT unknow ", "\x1B[>5x", 0, InputState_Normal},   // CSI + > x unhandled CSI >
		{"overflow OSC string", "\x1B]", 0, InputState_Normal}, // special logic in the following test code, add 4K string
		{"CSI unknow ", "\x1B[x", 0, InputState_Normal},        // unhandled CSI sequence
		{"CSI ? unknow ", "\x1B[?x", 0, InputState_Normal},     // unhandled CSI ? sequence
		{"CSI ? ESC ", "\x1B[?\x1B", 0, InputState_Normal},     // unhandled CSI ? ESC
		{"CSI ! unknow ", "\x1B[!x", 0, InputState_Normal},     // unhandled CSI ! x
	}

	p := NewParser()
	p.logE.SetOutput(ioutil.Discard)
	p.logU.SetOutput(ioutil.Discard)
	// p.logTrace = true // open the trace
	var hd *Handler
	for _, v := range tc {
		t.Run(v.name, func(t *testing.T) {
			raw := v.seq
			// special logic for the overflow case.
			if strings.HasPrefix(v.name, "overflow OSC") {
				var b strings.Builder
				b.WriteString(v.seq)        // the header
				for i := 0; i < 1024; i++ { // just 4096
					b.WriteString("blab")
				}
				b.WriteString("#") // 4096+1
				raw = b.String()
				t.Logf("%d\n", len(raw)-2)
			}

			for _, ch := range raw {
				hd = p.processInput(ch)
			}

			if hd == nil {
				if p.inputState != v.state || p.nInputOps != v.nInputOps {
					t.Errorf("%s seq=%q expect state=%q, nInputOps=%d, got state=%q, nInputOps=%d\n",
						v.name, v.seq, strInputState[v.state], v.nInputOps, strInputState[p.inputState], p.nInputOps)
				}
			} else {
				t.Errorf("%s should get nil handler, got %s\n", v.name, hd.name)
			}
		})
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
	// p.logTrace = true // open the trace
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

func TestHandle_DECSC_DECRC_DECSET1048(t *testing.T) {
	tc := []struct {
		name         string
		seq          string
		wantName     []string
		y, x         int
		AutoWrapMode bool
		OriginMode   bool
	}{
		// between the SC/RC, move cursor to 13,23,set originMode, autoWrapMode false,
		{
			"ESC DECSC/DECRC",
			"\x1B7\x1B[24;14H\x1B[?7l\x1B[?6l\x1B8",
			[]string{"esc-decsc", "csi-cup", "csi-decrst", "csi-decrst", "esc-decrc"},
			8, 8, true, true,
		},
		// between the SC/RC, move cursor to 11,21,set originMode, autoWrapMode true,
		{
			"CSI DECSET/DECRST 1048",
			"\x1B[?1048h\x1B[22;12H\x1B[?7h\x1B[?6h\x1B[?1048l",
			[]string{"csi-decset", "csi-cup", "csi-decset", "csi-decset", "csi-decrst"},
			9, 9, false, false,
		},
	}

	p := NewParser()
	emu := NewEmulator()

	for _, v := range tc {
		// set the target state for cursor
		emu.framebuffer.DS.MoveCol(v.x, false, false)
		emu.framebuffer.DS.MoveRow(v.y, false)
		emu.framebuffer.DS.AutoWrapMode = v.AutoWrapMode
		emu.framebuffer.DS.OriginMode = v.OriginMode

		// process control sequence
		hds := make([]*Handler, 0, 16)
		hds = p.processStream(v.seq, hds)

		if len(hds) == 0 {
			t.Errorf("%s got zero handlers.", v.name)
		}

		// handle the control sequence
		for j, hd := range hds {
			hd.handle(emu)
			if hd.name != v.wantName[j] { // validate the control sequences name
				t.Errorf("%s:\t %q expect %s, got %s\n", v.name, v.seq, v.wantName[j], hd.name)
			}
		}

		// validate the result
		fb := emu.framebuffer
		if fb.DS.GetCursorCol() != v.x || fb.DS.GetCursorRow() != v.y ||
			fb.DS.AutoWrapMode != v.AutoWrapMode || fb.DS.OriginMode != v.OriginMode {
			t.Errorf("%s %q expect (%d,%d,%t,%t), got (%d,%d,%t,%t)", v.name, v.seq,
				v.y, v.x, v.AutoWrapMode, v.OriginMode,
				fb.DS.GetCursorRow(), fb.DS.GetCursorCol(), fb.DS.AutoWrapMode, fb.DS.OriginMode)
		}
	}
}

func TestHandle_DA1_DA2_DSR(t *testing.T) {
	tc := []struct {
		name     string
		seq      string
		want     string
		wantName string
	}{
		{"Primary DA  ", "\x1B[c", fmt.Sprintf("\x1B[?%s", DEVICE_ID), "csi-da1"},
		{"Secondary DA", "\x1B[>c", "\x1B[>64;0;0c", "csi-da2"},
		{"Operating Status report ", "\x1B[5n", "\x1B[0n", "csi-dsr"},
	}

	p := NewParser()
	// p.logTrace = true // open the trace
	var hd *Handler
	emu := NewEmulator()

	for _, v := range tc {
		// reset the target content
		emu.dispatcher.terminalToHost.Reset()

		t.Run(v.name, func(t *testing.T) {
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
		})
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
			t.Logf("%2d %s\n", i, row.String())

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
		t.Logf("%2d %s\n", v.startY, row.String())

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
			t.Logf("%2d %s %s\n", i, row.String(), cellIn(i))

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

func TestHandle_VPA_CHA_HPA(t *testing.T) {
	tc := []struct {
		name           string
		wantName       string
		startX, startY int
		wantX, wantY   int
		seq            string
	}{
		{"VPA move cursor to row 3-1 ", "csi-vpa", 9, 8, 9, 2, "\x1B[3d"},
		{"VPA move cursor to row 34-1", "csi-vpa", 8, 8, 8, 33, "\x1B[34d"},
		{"SHA move cursor to col 1-1 ", "csi-cha-hpa", 7, 7, 0, 7, "\x1B[G"}, // default Ps is 1
		{"SHA move cursor to col 79-1", "csi-cha-hpa", 6, 6, 78, 6, "\x1B[79G"},
		{"HPA move cursor to col 9-1", "csi-cha-hpa", 5, 5, 8, 5, "\x1B[9`"},
		{"HPA move cursor to col 49-1", "csi-cha-hpa", 4, 4, 48, 4, "\x1B[49`"},
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
		emu.framebuffer.DS.MoveRow(v.startY, false)
		emu.framebuffer.DS.MoveCol(v.startX, false, false)

		// handle the control sequence
		for _, hd := range hds {
			hd.handle(emu)
			if hd.name != v.wantName {
				t.Errorf("%s:\t %q expect %s, got %s\n", v.name, v.seq, v.wantName, hd.name)
			}
		}

		gotX := emu.framebuffer.DS.GetCursorCol()
		gotY := emu.framebuffer.DS.GetCursorRow()

		if v.wantX != gotX || v.wantY != gotY {
			t.Errorf("%s seq=%q expect (%d,%d), got (%d,%d)\n", v.name, v.seq, v.wantY, v.wantX, gotY, gotX)
		}
	}
}

/* TODO refine it
func TestHandle_SGR_RGBcolor(t *testing.T) {
	tc := []struct {
		name       string
		wantName   string
		fr, fg, fb uint32
		br, bg, bb uint32
		attr       uint32
		seq        string
	}{
		{
			"RGB Color 1", "csi-sgr",
			33, 47, 12,
			123, 24, 34,
			Bold,
			"\x1B[0;1;38;2;33;47;12;48;2;123;24;34m",
		},
		{
			"RGB Color 2", "csi-sgr",
			0, 0, 0,
			0, 0, 0,
			Italic,
			"\x1B[0;3;38:2:0:0:0;48:2:0:0:0m",
		},
		{
			"RGB Color 3", "csi-sgr",
			12, 34, 128,
			59, 190, 155,
			Underlined,
			"\x1B[0;4;38:2:12:34:128;48:2:59:190:155m",
		},
	}

	p := NewParser()
	// the default size of emu is 80x40 [colxrow]
	emu := NewEmulator()
	rend0 := new(Renditions)

	for _, v := range tc {
		// clear the attribute
		rend0.ClearAttributes()
		rend0.SetAttributes(v.attr, true)
		v.attr = rend0.attributes

		// run the test
		t.Run(v.name, func(t *testing.T) {
			// process control sequence
			hds := make([]*Handler, 0, 16)
			hds = p.processStream(v.seq, hds)

			if len(hds) == 0 {
				t.Errorf("%s got zero handlers.", v.name)
			}

			// reset the renditions
			emu.framebuffer.DS.GetRenditions().ClearAttributes()

			// handle the control sequence
			for _, hd := range hds {
				hd.handle(emu)
				if hd.name != v.wantName {
					t.Errorf("%s:\t %q expect %s, got %s\n", v.name, v.seq, v.wantName, hd.name)
				}
			}

			// validate the result
			rend := emu.framebuffer.DS.GetRenditions()
			if rend.fgColor != makeTrueColor(v.fr, v.fg, v.fb) {
				t.Errorf("%s:\t %q expect foreground r=%d,g=%d,b=%d, got %x", v.name, v.seq, v.fr, v.fg, v.fb, rend.fgColor)
			}
			if rend.bgColor != makeTrueColor(v.br, v.bg, v.bb) {
				t.Errorf("%s:\t %q expect backgournd r=%d,g=%d,b=%d, got %x", v.name, v.seq, v.br, v.bg, v.bb, rend.bgColor)
			}
			if rend.attributes != v.attr {
				t.Errorf("%s:\t %q expect atrribute %b, got %b", v.name, v.seq, v.attr, rend.attributes)
			}
		})
	}
}
func TestHandle_SGR_ANSIcolor(t *testing.T) {
	tc := []struct {
		name     string
		wantName string
		fg       uint32
		bg       uint32
		attr     uint32
		seq      string
	}{
		{
			"8 Color", "csi-sgr",
			37, 40, Bold,
			"\x1B[1;37;40m",
		},
		{
			"8 Color 2", "csi-sgr",
			31, 41, Italic,
			"\x1B[0;3;31;41m",
		},
		{
			"16 Color", "csi-sgr",
			91 - 90 + 38, 107 - 100 + 48, Underlined,
			"\x1B[4;91;107m",
		},
		{
			"256 Color 1", "csi-sgr",
			33 + 30, 47 + 40, Bold,
			"\x1B[0;1;38:5:33;48:5:47m",
		},
		{
			"256 Color 3", "csi-sgr",
			128 + 30, 155 + 40, Underlined,
			"\x1B[0;4;38:5:128;48:5:155m",
		},
	}

	p := NewParser()
	// the default size of emu is 80x40 [colxrow]
	emu := NewEmulator()
	rend0 := new(Renditions)

	for _, v := range tc {
		// clear and set the attribute according to test case attr value
		rend0.ClearAttributes()
		rend0.SetAttributes(v.attr, true)
		v.attr = rend0.attributes

		t.Run(v.name, func(t *testing.T) {
			// process control sequence
			hds := make([]*Handler, 0, 16)
			hds = p.processStream(v.seq, hds)

			if len(hds) == 0 {
				t.Errorf("%s got zero handlers.", v.name)
			}

			emu.framebuffer.DS.GetRenditions().ClearAttributes()

			// handle the control sequence
			for _, hd := range hds {
				hd.handle(emu)
				if hd.name != v.wantName {
					t.Errorf("%s:\t %q expect %s, got %s\n", v.name, v.seq, v.wantName, hd.name)
				}
			}

			// validate the result
			rend := emu.framebuffer.DS.GetRenditions()
			if rend.fgColor != v.fg {
				t.Errorf("%s:\t %q expect foreground=%d, got %d", v.name, v.seq, v.fg, rend.fgColor)
			}
			if rend.bgColor != v.bg {
				t.Errorf("%s:\t %q expect backgournd=%d, got %d", v.name, v.seq, v.bg, rend.bgColor)
			}
			if rend.attributes != v.attr {
				t.Errorf("%s:\t %q expect atrribute %b, got %b", v.name, v.seq, v.attr, rend.attributes)
			}
		})
	}
}
*/
// TODO full test for scrolling mode
func TestHandle_DSR6(t *testing.T) {
	tc := []struct {
		name           string
		startX, startY int
		originMode     bool
		seq            string
		wantResp       string
		wantName       string
	}{
		{"Report Cursor Position originMode=true ", 8, 8, true, "\x1B[6n", "\x1B[9;9R", "csi-dsr"},
		{"Report Cursor Position originMode=false", 9, 9, false, "\x1B[6n", "\x1B[10;10R", "csi-dsr"},
	}

	p := NewParser()
	// p.logTrace = true // open the trace
	var hd *Handler
	emu := NewEmulator()

	for _, v := range tc {
		// reset the target content
		emu.dispatcher.terminalToHost.Reset()

		t.Run(v.name, func(t *testing.T) {
			// parse the sequence
			for _, ch := range v.seq {
				hd = p.processInput(ch)
			}

			// set condition
			emu.framebuffer.DS.OriginMode = v.originMode
			// move to the start position
			emu.framebuffer.DS.MoveRow(v.startY, false)
			emu.framebuffer.DS.MoveCol(v.startX, false, false)

			// execute the sequence handler
			if hd != nil {
				hd.handle(emu)
				if hd.name != v.wantName {
					t.Errorf("%s:\t %q expect %s, got %s\n", v.name, v.seq, v.wantName, hd.name)
				}
			} else {
				t.Errorf("%s got nil Handler.", v.name)
			}

			// validate the response
			got := emu.dispatcher.terminalToHost.String()
			if v.wantResp != got {
				t.Errorf("%s seq:%q expect %q, got %q\n", v.name, v.seq, v.wantResp, got)
			}
		})
	}
}

func TestHistory(t *testing.T) {
	tc := []struct {
		name       string
		value      rune
		reverseIdx int
		want       rune
	}{
		{"add a", 'a', 0, 'a'},
		{"add b", 'b', 0, 'b'},
		{"add c", 'c', 0, 'c'},
		{"add d", 'd', 0, 'd'},
		{"add e", 'e', 0, 'e'},
		{"add f", 'f', 1, 'e'},
		{"add d", 'd', 1, 'f'},
		{"add e", 'e', 1, 'd'},
		{"add f", 'f', 4, 'e'},
		{"add x", 'x', 6, '\x00'},
	}

	p := NewParser()
	for _, v := range tc {
		t.Run(v.name, func(t *testing.T) {
			p.appendToHistory(v.value)
			if v.want != p.getRuneAt(v.reverseIdx) {
				t.Errorf("%s expect reverseIdx[%d] value=%q, got %q\n", v.name, v.reverseIdx, v.want, p.getRuneAt(v.reverseIdx))
			}
		})
	}
}

func TestHandle_CSI_BS_FF_VT_CR_TAB(t *testing.T) {
	tc := []struct {
		name           string
		startX, startY int
		seq            string
		wantX, wantY   int
	}{
		{"CSI backspace number    ", 0, 0, "\x1B[23;12\bH", 0, 23 - 1},       // \b is executed first, just one handler left
		{"CSI backspace semicolon ", 0, 0, "\x1B[23;\b;12H", 12 - 1, 23 - 1}, // \b is executed first, just one handler left
		{"cursor down 1+3 rows VT ", 9, 8, "\x1B[3\vB", 9, 12},               // \v handler is handled first
		{"cursor down 1+3 rows FF ", 9, 8, "\x1B[\f3B", 9, 12},               // \f handler is handled first
		{"cursor up 2 rows and CR ", 8, 7, "\x1B[\r2A", 0, 5},                // \r handler is handled first
		{"cursor up 3 rows and CR ", 6, 6, "\x1B[3\rA", 0, 3},                // \r handler is handled first
		{"cursor forward 2cols +HT", 5, 3, "\x1B[2\tC", 10, 3},               // \t handler is handed first
		{"cursor forward 1cols +HT", 2, 5, "\x1B[\t1C", 9, 5},                // \t handler is handled first
	}

	p := NewParser()
	emu := NewEmulator()
	for _, v := range tc {
		t.Run(v.name, func(t *testing.T) {
			// process control sequence
			hds := make([]*Handler, 0, 16)
			hds = p.processStream(v.seq, hds)

			if len(hds) == 0 {
				t.Errorf("%s got zero handlers.", v.name)
			}

			// reset the start position
			emu.framebuffer.DS.MoveRow(v.startY, false)
			emu.framebuffer.DS.MoveCol(v.startX, false, false)

			// handle the control sequence
			for _, hd := range hds {
				hd.handle(emu)
				// if hd.name != v.wantName {
				// 	t.Errorf("%s:\t %q expect %s, got %s\n", v.name, v.seq, v.wantName, hd.name)
				// }
			}

			// get the result
			gotY := emu.framebuffer.DS.GetCursorRow()
			gotX := emu.framebuffer.DS.GetCursorCol()

			if gotX != v.wantX || gotY != v.wantY {
				t.Errorf("%s:%q expect cursor position (%d,%d), got (%d,%d)\n",
					v.name, v.seq, v.wantX, v.wantY, gotX, gotY)
			}
		})
	}
}

type (
	ANSImode uint
	DECmode  uint
)

const (
	t_keyboardLocked ANSImode = iota
	t_InsertMode
	t_localEcho
	t_autoNewlineMode
	t_ApplicationModeCursorKeys DECmode = iota
	t_ReverseVideo
	t_OriginMode
	t_AutoWrapMode
	t_CursorVisible
	t_focusEventMode
	t_MouseAlternateScroll
	t_altSendsEscape
	t_BracketedPaste
)

func t_getANSImode(ds *DrawState, which ANSImode) bool {
	switch which {
	case t_keyboardLocked:
		return ds.keyboardLocked
	case t_InsertMode:
		return ds.InsertMode
	case t_localEcho:
		return ds.localEcho
	case t_autoNewlineMode:
		return ds.autoNewlineMode
	}
	return false
}

func t_resetANSImode(ds *DrawState, which ANSImode, value bool) {
	switch which {
	case t_keyboardLocked:
		ds.keyboardLocked = value
	case t_InsertMode:
		ds.InsertMode = value
	case t_localEcho:
		ds.localEcho = value
	case t_autoNewlineMode:
		ds.autoNewlineMode = value
	}
}

func TestHandle_SM_RM(t *testing.T) {
	tc := []struct {
		name     string
		seq      string
		which    ANSImode
		wantName string
		want     bool
	}{
		{"SM: keyboardLocked ", "\x1B[2h", t_keyboardLocked, "csi-sm", true},
		{"SM: InsertMode     ", "\x1B[4h", t_InsertMode, "csi-sm", true},
		{"SM: localEcho      ", "\x1B[12h", t_localEcho, "csi-sm", false},
		{"SM: autoNewlineMode", "\x1B[20h", t_autoNewlineMode, "csi-sm", true},

		{"RM: keyboardLocked ", "\x1B[2l", t_keyboardLocked, "csi-rm", false},
		{"RM: InsertMode     ", "\x1B[4l", t_InsertMode, "csi-rm", false},
		{"RM: localEcho      ", "\x1B[12l", t_localEcho, "csi-rm", true},
		{"RM: autoNewlineMode", "\x1B[20l", t_autoNewlineMode, "csi-rm", false},
	}

	p := NewParser()
	emu := NewEmulator()

	for _, v := range tc {
		t.Run(v.name, func(t *testing.T) {
			// process control sequence
			hds := make([]*Handler, 0, 16)
			hds = p.processStream(v.seq, hds)

			if len(hds) == 0 {
				t.Errorf("%s got zero handlers.", v.name)
			}

			// reset the target value
			t_resetANSImode(emu.framebuffer.DS, v.which, !v.want)

			// handle the control sequence
			for _, hd := range hds {
				hd.handle(emu)
				if hd.name != v.wantName {
					t.Errorf("%s:\t %q expect %s, got %s\n", v.name, v.seq, v.wantName, hd.name)
				}
			}

			if v.want != t_getANSImode(emu.framebuffer.DS, v.which) {
				t.Errorf("%s: %q\t expect %t, got %t\n", v.name, v.seq,
					v.want, t_getANSImode(emu.framebuffer.DS, v.which))
			}
		})
	}
}

func TestHandleUnknow(t *testing.T) {
	tc := []struct {
		name string
		seq  string
		want string
	}{
		{"CSI SM unknow", "\x1B[21h", "CSI SM: Ignored bogus set mode"},
		{"CSI RM unknow", "\x1B[33l", "CSI RM: Ignored bogus reset mode"},
	}

	p := NewParser()
	emu := NewEmulator()
	var place strings.Builder
	emu.logW = log.New(&place, "WARN: ", log.Ldate|log.Ltime|log.Lshortfile)

	for _, v := range tc {
		t.Run(v.name, func(t *testing.T) {
			// process control sequence
			hds := make([]*Handler, 0, 16)
			hds = p.processStream(v.seq, hds)

			if len(hds) == 0 {
				t.Errorf("%s got zero handlers.", v.name)
			}

			// handle the control sequence
			for _, hd := range hds {
				hd.handle(emu)
			}

			if !strings.Contains(place.String(), v.want) {
				t.Errorf("%s: %q\t expect %q, got %q\n", v.name, v.seq, v.want, place.String())
			}
		})
	}
}

func t_getDECmode(ds *DrawState, which DECmode) bool {
	switch which {
	case t_ApplicationModeCursorKeys:
		return ds.ApplicationModeCursorKeys
	case t_ReverseVideo:
		return ds.ReverseVideo
	case t_OriginMode:
		return ds.OriginMode
	case t_AutoWrapMode:
		return ds.AutoWrapMode
	case t_CursorVisible:
		return ds.CursorVisible
	case t_focusEventMode:
		return ds.mouseTrk.focusEventMode
	case t_MouseAlternateScroll:
		return ds.MouseAlternateScroll
	case t_altSendsEscape:
		return ds.altSendsEscape
	case t_BracketedPaste:
		return ds.BracketedPaste
	}
	return false
}

func t_resetDECmode(ds *DrawState, which DECmode, value bool) {
	switch which {
	case t_ApplicationModeCursorKeys:
		ds.ApplicationModeCursorKeys = value
	case t_ReverseVideo:
		ds.ReverseVideo = value
	case t_OriginMode:
		ds.OriginMode = value
	case t_AutoWrapMode:
		ds.AutoWrapMode = value
	case t_CursorVisible:
		ds.CursorVisible = value
	case t_focusEventMode:
		ds.mouseTrk.focusEventMode = value
	case t_MouseAlternateScroll:
		ds.MouseAlternateScroll = value
	case t_altSendsEscape:
		ds.altSendsEscape = value
	case t_BracketedPaste:
		ds.BracketedPaste = value
	}
}

func TestHandle_DECSET_DECRST_BOOL(t *testing.T) {
	tc := []struct {
		name     string
		seq      string
		which    DECmode
		wantName string
		want     bool
	}{
		{"DECSET: ApplicationModeCursorKeys", "\x1B[?1h", t_ApplicationModeCursorKeys, "csi-decset", true},
		{"DECSET: ReverseVideo", "\x1B[?5h", t_ReverseVideo, "csi-decset", true},
		{"DECSET: OriginMode", "\x1B[?6h", t_OriginMode, "csi-decset", true},
		{"DECSET: AutoWrapMode", "\x1B[?7h", t_AutoWrapMode, "csi-decset", true},
		{"DECSET: CursorVisible", "\x1B[?25h", t_CursorVisible, "csi-decset", true},
		{"DECSET: focusEventMode", "\x1B[?1004h", t_focusEventMode, "csi-decset", true},
		{"DECSET: MouseAlternateScroll", "\x1B[?1007h", t_MouseAlternateScroll, "csi-decset", true},
		{"DECSET: altSendsEscape", "\x1B[?1036h", t_altSendsEscape, "csi-decset", true},
		{"DECSET: altSendsEscape", "\x1B[?1039h", t_altSendsEscape, "csi-decset", true},
		{"DECSET: BracketedPaste", "\x1B[?2004h", t_BracketedPaste, "csi-decset", true},

		{"DECRST: ApplicationModeCursorKeys", "\x1B[?1l", t_ApplicationModeCursorKeys, "csi-decrst", false},
		{"DECRST: ReverseVideo", "\x1B[?5l", t_ReverseVideo, "csi-decrst", false},
		{"DECRST: OriginMode", "\x1B[?6l", t_OriginMode, "csi-decrst", false},
		{"DECRST: AutoWrapMode", "\x1B[?7l", t_AutoWrapMode, "csi-decrst", false},
		{"DECRST: CursorVisible", "\x1B[?25l", t_CursorVisible, "csi-decrst", false},
		{"DECRST: focusEventMode", "\x1B[?1004l", t_focusEventMode, "csi-decrst", false},
		{"DECRST: MouseAlternateScroll", "\x1B[?1007l", t_MouseAlternateScroll, "csi-decrst", false},
		{"DECRST: altSendsEscape", "\x1B[?1036l", t_altSendsEscape, "csi-decrst", false},
		{"DECRST: altSendsEscape", "\x1B[?1039l", t_altSendsEscape, "csi-decrst", false},
		{"DECRST: BracketedPaste", "\x1B[?2004l", t_BracketedPaste, "csi-decrst", false},
	}

	p := NewParser()
	emu := NewEmulator()

	for _, v := range tc {
		t.Run(v.name, func(t *testing.T) {
			// process control sequence
			hds := make([]*Handler, 0, 16)
			hds = p.processStream(v.seq, hds)

			if len(hds) == 0 {
				t.Errorf("%s got zero handlers.", v.name)
			}

			// reset the target value
			t_resetDECmode(emu.framebuffer.DS, v.which, !v.want)

			// handle the control sequence
			for _, hd := range hds {
				hd.handle(emu)
				if hd.name != v.wantName {
					t.Errorf("%s:\t %q expect %s, got %s\n", v.name, v.seq, v.wantName, hd.name)
				}
			}

			if v.want != t_getDECmode(emu.framebuffer.DS, v.which) {
				t.Errorf("%s: %q\t expect %t, got %t\n", v.name, v.seq,
					v.want, t_getDECmode(emu.framebuffer.DS, v.which))
			}
		})
	}
}

func TestHandle_DECSET_DECRST_Log(t *testing.T) {
	tc := []struct {
		name     string
		seq      string
		wantName string
		want     string
	}{
		{"DECSET:   4", "\x1B[?4h", "csi-decset", "DECSCLM: Set smooth scroll"},
		{"DECSET:   8", "\x1B[?8h", "csi-decset", "DECARM: Set auto-repeat mode"},
		{"DECSET:  12", "\x1B[?12h", "csi-decset", "Start blinking cursor"},
		{"DECSET:1001", "\x1B[?1001h", "csi-decset", "Set VT200 Highlight Mouse mode"},
		{"DECSET:unknow", "\x1B[?2022h", "csi-decset", "set priv mode"},

		{"DECRST:   4", "\x1B[?4l", "csi-decrst", "DECSCLM: Set jump scroll"},
		{"DECRST:   8", "\x1B[?8l", "csi-decrst", "DECARM: Reset auto-repeat mode"},
		{"DECRST:  12", "\x1B[?12l", "csi-decrst", "Stop blinking cursor"},
		{"DECRST:1001", "\x1B[?1001l", "csi-decrst", "Reset VT200 Highlight Mouse mode"},
		{"DECRST:unknow", "\x1B[?2022l", "csi-decrst", "reset priv mode"},
	}

	p := NewParser()
	emu := NewEmulator()

	// replcase the io.Writer for test
	var place strings.Builder
	emu.logT = log.New(&place, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	// emu.logE = log.New(&place, "ERRO: ", log.Ldate|log.Ltime|log.Lshortfile)
	// emu.logW = log.New(&place, "WARN: ", log.Ldate|log.Ltime|log.Lshortfile)
	emu.logU = log.New(&place, "(Uimplemented): ", log.Ldate|log.Ltime|log.Lshortfile)

	for _, v := range tc {
		t.Run(v.name, func(t *testing.T) {
			// process control sequence
			hds := make([]*Handler, 0, 16)
			hds = p.processStream(v.seq, hds)

			if len(hds) == 0 {
				t.Errorf("%s got zero handlers.", v.name)
			}

			// handle the control sequence
			for _, hd := range hds {
				hd.handle(emu)
				if hd.name != v.wantName {
					t.Errorf("%s:\t %q expect %s, got %s\n", v.name, v.seq, v.wantName, hd.name)
				}
			}

			if !strings.Contains(place.String(), v.want) {
				t.Errorf("%s: %q\t expect %q, got %q\n", v.name, v.seq, v.want, place.String())
			}
		})
	}
}

func TestHandle_DECSET_DECRST_MouseTrackingMode(t *testing.T) {
	tc := []struct {
		name     string
		seq      string
		wantName string
		want     MouseTrackingMode
	}{
		{"DECSET:   9", "\x1B[?9h", "csi-decset", MouseModeX10},
		{"DECSET:1000", "\x1B[?1000h", "csi-decset", MouseModeVT200},
		{"DECSET:1002", "\x1B[?1002h", "csi-decset", MouseModeButtonEvent},
		{"DECSET:1003", "\x1B[?1003h", "csi-decset", MouseModeAnyEvent},

		{"DECRST:   9", "\x1B[?9l", "csi-decrst", MouseModeNone},
		{"DECRST:1000", "\x1B[?1000l", "csi-decrst", MouseModeNone},
		{"DECRST:1002", "\x1B[?1002l", "csi-decrst", MouseModeNone},
		{"DECRST:1003", "\x1B[?1003l", "csi-decrst", MouseModeNone},
	}

	p := NewParser()
	emu := NewEmulator()

	for _, v := range tc {
		t.Run(v.name, func(t *testing.T) {
			// process control sequence
			hds := make([]*Handler, 0, 16)
			hds = p.processStream(v.seq, hds)

			if len(hds) == 0 {
				t.Errorf("%s got zero handlers.", v.name)
			}

			// handle the control sequence
			for _, hd := range hds {
				hd.handle(emu)
				if hd.name != v.wantName {
					t.Errorf("%s:\t %q expect %s, got %s\n", v.name, v.seq, v.wantName, hd.name)
				}
			}

			got := emu.framebuffer.DS.mouseTrk.mode
			if got != v.want {
				t.Errorf("%s: %q\t expect %d, got %d\n", v.name, v.seq, v.want, got)
			}
		})
	}
}

func TestHandle_DECSET_DECRST_MouseTrackingEnc(t *testing.T) {
	tc := []struct {
		name     string
		seq      string
		wantName string
		want     MouseTrackingEnc
	}{
		{"DECSET:1005", "\x1B[?1005h", "csi-decset", MouseEncUTF},
		{"DECSET:1006", "\x1B[?1006h", "csi-decset", MouseEncSGR},
		{"DECSET:1015", "\x1B[?1015h", "csi-decset", MouseEncURXVT},

		{"DECRST:1005", "\x1B[?1005l", "csi-decrst", MouseEncNone},
		{"DECRST:1006", "\x1B[?1006l", "csi-decrst", MouseEncNone},
		{"DECRST:1015", "\x1B[?1015l", "csi-decrst", MouseEncNone},
	}

	p := NewParser()
	emu := NewEmulator()

	for _, v := range tc {
		t.Run(v.name, func(t *testing.T) {
			// process control sequence
			hds := make([]*Handler, 0, 16)
			hds = p.processStream(v.seq, hds)

			if len(hds) == 0 {
				t.Errorf("%s got zero handlers.", v.name)
			}

			// handle the control sequence
			for _, hd := range hds {
				hd.handle(emu)
				if hd.name != v.wantName {
					t.Errorf("%s:\t %q expect %s, got %s\n", v.name, v.seq, v.wantName, hd.name)
				}
			}

			got := emu.framebuffer.DS.mouseTrk.enc
			if got != v.want {
				t.Errorf("%s: %q\t expect %d, got %d\n", v.name, v.seq, v.want, got)
			}
		})
	}
}

func TestHandle_DECSET_DECRST_2(t *testing.T) {
	tc := []struct {
		name                string
		seq                 string
		wantName            []string
		compatLevel         CompatibilityLevel
		isResetCharsetState bool
	}{
		{"DECSET 2", "\x1B[?2h", []string{"csi-decset"}, CompatLevelVT400, true},
		{"DECRST 2", "\x1B[?2l", []string{"csi-decrst"}, CompatLevelVT52, true},
	}

	p := NewParser()
	emu := NewEmulator()

	for _, v := range tc {

		// process control sequence
		hds := make([]*Handler, 0, 16)
		hds = p.processStream(v.seq, hds)

		if len(hds) == 0 {
			t.Errorf("%s got zero handlers.", v.name)
		}

		// handle the control sequence
		for j, hd := range hds {
			hd.handle(emu)
			if hd.name != v.wantName[j] { // validate the control sequences name
				t.Errorf("%s:\t %q expect %s, got %s\n", v.name, v.seq, v.wantName[j], hd.name)
			}
		}

		// validate the result
		got := emu.framebuffer.DS.compatLevel
		newCS := isNewCharsetState(emu.charsetState)
		if v.isResetCharsetState != newCS || v.compatLevel != got {
			t.Errorf("%s %q expect reset CharsetState and compatbility level (%t,%d), got(%t,%d)",
				v.name, v.seq, true, v.compatLevel, newCS, got)
		}
	}
}

func isNewCharsetState(cs CharsetState) (ret bool) {
	ret = true
	for _, v := range cs.g {
		if v != nil {
			return false
		}
	}

	if cs.gl != 0 || cs.gr != 2 || cs.ss != 0 {
		return false
	}

	if cs.vtMode {
		ret = false
	}
	return ret
}

func TestHandle_DECSTBM(t *testing.T) {
	tc := []struct {
		name        string
		seq         string
		wantName    []string
		top, bottom int
	}{
		{
			"DECSTBM ", "\x1B[24;14H\x1B[2;30r", // move the cursor to 24,14 first
			[]string{"csi-cup", "csi-decstbm"},
			2 - 1, 30 - 1,
		},
		{
			"DECSTBM ", "\x1B[2;6H\x1B[3;32r\x1B[32;30r", // a successful STBM follow a ignored STBM.
			[]string{"csi-cup", "csi-decstbm", "csi-decstbm"},
			3 - 1, 32 - 1,
		},
	}

	p := NewParser()
	emu := NewEmulator()

	for _, v := range tc {

		// process control sequence
		hds := make([]*Handler, 0, 16)
		hds = p.processStream(v.seq, hds)

		if len(hds) == 0 {
			t.Errorf("%s got zero handlers.", v.name)
		}

		// handle the control sequence
		for j, hd := range hds {
			hd.handle(emu)
			if hd.name != v.wantName[j] { // validate the control sequences name
				t.Errorf("%s:\t %q expect %s, got %s\n", v.name, v.seq, v.wantName[j], hd.name)
			}
		}

		// validate the result
		top := emu.framebuffer.DS.GetScrollingRegionTopRow()
		bottom := emu.framebuffer.DS.GetScrollingRegionBottomRow()

		y := emu.framebuffer.DS.GetCursorRow()
		x := emu.framebuffer.DS.GetCursorCol()

		if x != 0 || y != 0 || top != v.top || bottom != v.bottom {
			t.Logf("%s: %q expect cursor in (0,0), got (%d,%d)\n", v.name, v.seq, y, x)
			t.Errorf("%s: %q expect [top:bottom] [%d,%d], got [%d,%d]\n", v.name, v.seq,
				v.top, v.bottom, top, bottom)
		}
	}
}

// here we use control sequences to setup the pre-conditions
func TestHandle_DECSTR(t *testing.T) {
	tc := []struct {
		name     string
		seq      string
		wantName []string
		sc       SavedCursor
		rend     Renditions
	}{
		{
			"DECSTR ",
			/*
				set ture for InsertMode, OriginMode, AutoWrapMode, ApplicationModeCursorKeys
				set top/bottom region, we don't care the value, because reset.
				soft terminal reset
			*/
			"\x1B[4h\x1B[?6h\x1B[?25h\x1B[?1h\x1B[2;30r\x1B[!p",
			[]string{"csi-sm", "csi-decset", "csi-decset", "csi-decset", "csi-decstbm", "csi-decstr"},
			SavedCursor{autoWrapMode: true},
			Renditions{},
		},
	}

	p := NewParser()
	emu := NewEmulator()

	for _, v := range tc {

		// process control sequence
		hds := make([]*Handler, 0, 16)
		hds = p.processStream(v.seq, hds)

		if len(hds) == 0 {
			t.Errorf("%s got zero handlers.", v.name)
		}

		// execute the control sequence
		for j, hd := range hds {
			hd.handle(emu)
			if hd.name != v.wantName[j] { // validate the control sequences name
				t.Errorf("%s:\t %q expect %s, got %s\n", v.name, v.seq, v.wantName[j], hd.name)
			}
		}

		// validate the 4 mode, should all be false
		ds := emu.framebuffer.DS
		if ds.InsertMode || ds.OriginMode || ds.CursorVisible || ds.ApplicationModeCursorKeys {
			t.Errorf("%s: %q expect all false for InsertMode OriginMode CursorVisible ApplicationModeCursorKeys, got %t,%t,%t,%t\n",
				v.name, v.seq, ds.InsertMode, ds.OriginMode, ds.CursorVisible, ds.ApplicationModeCursorKeys)
		}

		// validate the top/bottom region
		top := ds.GetScrollingRegionTopRow()
		bottom := ds.GetScrollingRegionBottomRow()
		if top != 0 || bottom != ds.GetHeight()-1 {
			t.Errorf("%s: %q expect [top,bottom] [%d,%d], got [%d,%d]\n", v.name, v.seq, 0, ds.GetHeight()-1, top, bottom)
		}

		// validate the Renditions and SavedCursor
		if v.sc != ds.save || v.rend != *ds.GetRenditions() {
			t.Errorf("%s: %q expect Renditions=%v, SavedCursor=%v, got %v, %v\n", v.name, v.seq, v.rend, v.sc,
				*ds.GetRenditions(), ds.save)
		}
	}
}

func TestHandle_OSC_52(t *testing.T) {
	tc := []struct {
		name       string
		wantName   []string
		wantPc     string
		wantPd     string
		wantString string
		noReply    bool
		seq        string
	}{
		{
			"new selection in c",
			[]string{"osc-52"},
			"c", "YXByaWxzaAo=",
			"\x1B]52;c;YXByaWxzaAo=\x1B\\", true,
			"\x1B]52;c;YXByaWxzaAo=\x1B\\",
		},
		{
			"clear selection in cs",
			[]string{"osc-52", "osc-52"},
			"cs", "",
			"\x1B]52;cs;x\x1B\\", true, // echo "aprilsh" | base64
			"\x1B]52;cs;YXByaWxzaAo=\x1B\\\x1B]52;cs;x\x1B\\",
		},
		{
			"empty selection",
			[]string{"osc-52"},
			"s0", "5Zub5aeR5aiY5bGxCg==", // echo "四姑娘山" | base64
			"\x1B]52;s0;5Zub5aeR5aiY5bGxCg==\x1B\\", true,
			"\x1B]52;;5Zub5aeR5aiY5bGxCg==\x1B\\",
		},
		{
			"question selection",
			[]string{"osc-52", "osc-52"},
			"", "", // don't care these values
			"\x1B]52;c;5Zub5aeR5aiY5bGxCg==\x1B\\", false,
			"\x1B]52;c0;5Zub5aeR5aiY5bGxCg==\x1B\\\x1B]52;c0;?\x1B\\",
		},
	}
	p := NewParser()
	emu := NewEmulator()

	for _, v := range tc {
		emu.framebuffer.selectionData = ""
		emu.dispatcher.terminalToHost.Reset()

		t.Run(v.name, func(t *testing.T) {
			// process control sequence
			hds := make([]*Handler, 0, 16)
			hds = p.processStream(v.seq, hds)

			if len(hds) == 0 {
				t.Errorf("%s got zero handlers.", v.name)
			}

			// execute the control sequence
			for j, hd := range hds {
				hd.handle(emu)
				if hd.name != v.wantName[j] { // validate the control sequences name
					t.Errorf("%s:\t %q expect %s, got %s\n", v.name, v.seq, v.wantName[j], hd.name)
				}
			}

			if v.noReply {
				if v.wantString != emu.framebuffer.selectionData {
					t.Errorf("%s: seq=%q expect %q, got %q\n", v.name, v.seq, v.wantString, emu.framebuffer.selectionData)
				}
				for _, ch := range v.wantPc {
					if data, ok := emu.selectionData[ch]; ok && data == v.wantPd {
						continue
					} else {
						t.Errorf("%s: seq=%q, expect[%c]%q, got [%c]%q\n", v.name, v.seq, ch, v.wantPc, ch, emu.selectionData[ch])
					}
				}
			} else {
				got := emu.dispatcher.terminalToHost.String()
				if got != v.wantString {
					t.Errorf("%s: seq=%q, expect %q, got %q\n", v.name, v.seq, v.wantString, got)
				}
			}
		})
	}
}

func TestHandle_OSC_52_abort(t *testing.T) {
	tc := []struct {
		name     string
		wantName string
		wantStr  string
		seq      string
	}{
		{"malform OSC 52 ", "osc-52", "OSC 52: can't find Pc parameter.", "\x1B]52;23\x1B\\"},
		{"Pc not in range", "osc-52", "invalid Pc parameters.", "\x1B]52;se;\x1B\\"},
	}
	p := NewParser()
	emu := NewEmulator()
	var place strings.Builder
	emu.logW = log.New(&place, "WARN: ", log.Ldate|log.Ltime|log.Lshortfile)

	for _, v := range tc {
		place.Reset()
		t.Run(v.name, func(t *testing.T) {
			// process control sequence
			hds := make([]*Handler, 0, 16)
			hds = p.processStream(v.seq, hds)

			if len(hds) == 0 {
				t.Errorf("%s got zero handlers.", v.name)
			}

			// execute the control sequence
			for _, hd := range hds {
				hd.handle(emu)
				if hd.name != v.wantName { // validate the control sequences name
					t.Errorf("%s:\t %q expect %s, got %s\n", v.name, v.seq, v.wantName, hd.name)
				}
			}

			if !strings.Contains(place.String(), v.wantStr) {
				t.Errorf("%s: seq=%q expect %q, got %q\n", v.name, v.seq, v.wantStr, place.String())
			}
		})
	}
}

func TestHandle_OSC_4(t *testing.T) {
	// color.Palette.Index(c color.Color)
	tc := []struct {
		name       string
		wantName   []string
		wantString string
		warn       bool
		seq        string
	}{
		{
			"query one color number",
			[]string{"osc-4"},
			"\x1B]4;1;rgb:0080/0000/0000\x1B\\", false,
			"\x1B]4;1;?\x1B\\",
		},
		{
			"query two color number",
			[]string{"osc-4"},
			"\x1B]4;250;rgb:00bc/00bc/00bc\x1B\\\x1B]4;1;rgb:0080/0000/0000\x1B\\", false,
			"\x1B]4;250;?;1;?\x1B\\",
		},
		{
			"query 8 color number",
			[]string{"osc-4"},
			"\x1B]4;0;rgb:0000/0000/0000\x1B\\\x1B]4;1;rgb:0080/0000/0000\x1B\\\x1B]4;2;rgb:0000/0080/0000\x1B\\\x1B]4;3;rgb:0080/0080/0000\x1B\\\x1B]4;4;rgb:0000/0000/0080\x1B\\\x1B]4;5;rgb:0080/0000/0080\x1B\\\x1B]4;6;rgb:0000/0080/0080\x1B\\\x1B]4;7;rgb:00c0/00c0/00c0\x1B\\", false,
			"\x1B]4;0;?;1;?;2;?;3;?;4;?;5;?;6;?;7;?\x1B\\",
		},
		{
			"missing ';' abort",
			[]string{"osc-4"},
			"OSC 4: malformed argument, missing ';'.", true,
			"\x1B]4;1?\x1B\\",
		},
		{
			"Ps malform abort",
			[]string{"osc-4"},
			"OSC 4: can't parse c parameter.", true,
			"\x1B]4;m;?\x1B\\",
		},
	}
	p := NewParser()
	emu := NewEmulator()
	var place strings.Builder
	emu.logW = log.New(&place, "WARN: ", log.Ldate|log.Ltime|log.Lshortfile)

	for _, v := range tc {
		place.Reset()
		emu.dispatcher.terminalToHost.Reset()

		t.Run(v.name, func(t *testing.T) {
			// process control sequence
			hds := make([]*Handler, 0, 16)
			hds = p.processStream(v.seq, hds)

			if len(hds) == 0 {
				t.Errorf("%s got zero handlers.", v.name)
			}

			// execute the control sequence

			for j, hd := range hds {
				hd.handle(emu)
				if hd.name != v.wantName[j] { // validate the control sequences name
					t.Errorf("%s:\t %q expect %s, got %s\n", v.name, v.seq, v.wantName[j], hd.name)
				}
			}

			if v.warn {
				if !strings.Contains(place.String(), v.wantString) {
					t.Errorf("%s: seq=%q expect %q, got %q\n", v.name, v.seq, v.wantString, place.String())
				}
			} else {
				got := emu.dispatcher.terminalToHost.String()
				if got != v.wantString {
					t.Errorf("%s: seq=%q, \nexpect\t %q, \ngot\t\t %q\n", v.name, v.seq, v.wantString, got)
				}
			}
		})
	}
}
