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
	"io/ioutil"
	"reflect"
	"strings"
	"testing"

	"github.com/rivo/uniseg"
)

// https://www.redhat.com/sysadmin/linux-script-command
// check script command for preparing the test data.
// apk add util-linux-misc util-linux-doc

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
			width += RunesWidth(rs)
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

	hd = p.ProcessInput(chs...)
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
		name      string
		seq       string // data stream with control sequences
		hdIDs     []int
		hdNumber  int    // expect handler number
		posY      int    // expect print row
		posX      []int  // expect print cols
		graphemes string // data string without control sequences
	}{
		// use CUP to set the active cursor position first
		{
			"UTF-8 plain english",
			"\x1B[1;14Hlong long ago",
			[]int{CSI_CUP, Graphemes},
			14, 0,
			[]int{13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25},
			"long long ago",
		},
		{
			"UTF-8 chinese, combining character and flags",
			"\x1B[2;30HChin\u0308\u0308a 🏖 i国旗🇳🇱Fun 🌈with Flag🇧🇷.s",
			[]int{CSI_CUP, Graphemes},
			30, 1,
			[]int{29, 30, 31, 32, 33, 34, 35, 37, 38, 39, 41, 43, 45, 46, 47, 48, 49, 51, 52, 53, 54, 55, 56, 57, 58, 59, 60, 62, 63},
			"Chin\u0308\u0308a 🏖 i国旗🇳🇱Fun 🌈with Flag🇧🇷.s",
		},
		{
			"VT mix UTF-8",
			"\x1B[3;24H中国\x1B%@\xA5AB\xe2\xe3\xe9\x1B%GShanghai\x1B%@CD\xe0\xe1",
			[]int{
				CSI_CUP, Graphemes, Graphemes, ESC_DOCS_ISO8859_1, Graphemes, Graphemes, Graphemes, Graphemes, Graphemes,
				Graphemes, ESC_DOCS_UTF8, Graphemes, Graphemes, Graphemes, Graphemes, Graphemes, Graphemes, Graphemes,
				Graphemes, ESC_DOCS_ISO8859_1, Graphemes, Graphemes, Graphemes, Graphemes,
			},
			24, 2,
			[]int{23, 25, 27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38, 39, 40, 41, 42, 43, 44},
			"中国¥ABâãéShanghaiCDàá",
		},
		{
			"VT edge", "\x1B[4;10H\x1B%@Beijing\x1B%G",
			[]int{CSI_CUP, ESC_DOCS_ISO8859_1, Graphemes, Graphemes, Graphemes, Graphemes, Graphemes, Graphemes, Graphemes, ESC_DOCS_UTF8},
			10, 3,
			[]int{9, 10, 11, 12, 13, 14, 15},
			"Beijing",
		},
	}

	p := NewParser()
	var place strings.Builder
	p.logE.SetOutput(&place)
	p.logU.SetOutput(&place)
	p.logT.SetOutput(&place)

	emu := NewEmulator3(80, 40, 40)
	emu.logT.SetOutput(&place)
	for _, v := range tc {
		place.Reset()

		t.Run(v.name, func(t *testing.T) {
			hds := make([]*Handler, 0, 16)
			hds = p.processStream(v.seq, hds)

			if v.hdNumber != len(hds) {
				t.Errorf("%s expect %d handlers,got %d handlers\n", v.name, v.hdNumber, len(hds))
			}

			hdID := 0
			for j, hd := range hds {
				hd.handle(emu)

				if len(v.hdIDs) > 2 {
					hdID = v.hdIDs[j]
				} else { // to avoid type more hdID in test case
					if j == 0 {
						hdID = v.hdIDs[0]
					} else {
						hdID = v.hdIDs[1]
					}
				}
				if hd.id != hdID { // validate the control sequences id
					t.Errorf("%s: seq=%q expect %s, got %s\n", v.name, v.seq, strHandlerID[hdID], strHandlerID[hd.id])
				}
			}

			// validate the result with data string
			graphemes := uniseg.NewGraphemes(v.graphemes)
			j := 0
			for graphemes.Next() {
				// the expected content
				chs := graphemes.Runes()

				// get the cell from framebuffer
				rows := v.posY
				cols := v.posX[j]
				cell := emu.cf.getCell(rows, cols)

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
		name      string
		seq       string
		posY      int    // expect print row
		posX      []int  // expect print cols
		graphemes string // data string without control sequences
	}{
		{"plain english wrap", "\x1B[8;79Hap\u0308rish", 7, []int{78, 79, 0, 1, 2, 3}, "ap\u0308rish"},
		{"chinese even wrap", "\x1B[9;79H@@四姑娘山", 8, []int{78, 79, 0, 2, 4, 6}, "@@四姑娘山"},
		{"chinese odd wrap", "\x1B[10;79H#海螺沟", 9, []int{78, 0, 2, 4, 6}, "#海螺沟"},
		{"insert wrap", "\x1B[4h\x1B[11;78H#th#", 10, []int{77, 78, 79, 0}, "#th#"},
		{"english scroll wrap", "\x1B[40;78H#th#", 39, []int{77, 78, 79, 0}, "#th#"},
	}

	p := NewParser()
	emu := NewEmulator3(80, 40, 10)
	var place strings.Builder
	emu.logT.SetOutput(&place)

	for _, v := range tc {
		place.Reset()
		t.Run(v.name, func(t *testing.T) {
			hds := make([]*Handler, 0, 16)
			hds = p.processStream(v.seq, hds)

			if len(hds) == 0 {
				t.Errorf("%s got zero handlers.", v.name)
			}

			for _, hd := range hds {
				hd.handle(emu)
			}

			// validate the result with data string
			graphemes := uniseg.NewGraphemes(v.graphemes)
			rows := v.posY
			index := 0
			// scroll will change the row number
			if v.posY == 39 {
				rows = v.posY - 1
			}
			for graphemes.Next() {
				// the expected content
				chs := graphemes.Runes()
				cols := v.posX[index]
				// change to the next row
				if cols == 0 {
					rows += 1
				}
				// get the cell from framebuffer
				cell := emu.cf.getCell(rows, cols)

				if cell.contents != string(chs) {
					t.Errorf("%s cell [%2d,%2d] expect %q, got %q\n", v.name, rows, cols, string(chs), cell.contents)
				}
				index += 1
			}

			if t.Failed() {
				t.Errorf("%s row=%d\n%s", v.name, v.posY, printCells(emu.cf, v.posY))
				t.Errorf("%s row=%d\n%s", v.name, v.posY+1, printCells(emu.cf, v.posY+1))
			}
		})
	}
}

func TestHandle_REP(t *testing.T) {
	tc := []struct {
		name      string
		seq       string
		hdIDs     []int
		posY      int    // expect print row
		posX      []int  // expect print cols
		graphemes string // data string without control sequences
	}{
		{
			"plain english REP+wrap", "\x1B[8;79Hp\u0308\x1B[b",
			[]int{CSI_CUP, Graphemes, CSI_REP},
			7,
			[]int{78, 79},
			"p\u0308p\u0308",
		},
		{
			"chinese even REP+wrap", "\x1B[9;79H四\x1B[5b",
			[]int{CSI_CUP, Graphemes, CSI_REP},
			8,
			[]int{78, 0, 2, 4, 6, 8},
			"四四四四四四",
		},
		{
			"chinese odd REP+wrap", "\x1B[10;79H#海\x1B[5b",
			[]int{CSI_CUP, Graphemes, Graphemes, CSI_REP},
			9,
			[]int{78, 0, 2, 4, 6, 8, 10},
			"#海海海海海海",
		},
		{
			"insert REP+wrap", "\x1B[4h\x1B[11;78H#\x1B[5b",
			[]int{CSI_SM, CSI_CUP, Graphemes, CSI_REP},
			10,
			[]int{77, 78, 79, 0, 1, 2},
			"######",
		},
	}

	p := NewParser()
	var place strings.Builder
	p.logE.SetOutput(&place)
	p.logU.SetOutput(&place)
	p.logT.SetOutput(&place)

	emu := NewEmulator3(80, 40, 40)
	emu.logT.SetOutput(&place)
	for _, v := range tc {
		place.Reset()

		t.Run(v.name, func(t *testing.T) {
			hds := make([]*Handler, 0, 16)
			hds = p.processStream(v.seq, hds)

			if len(v.hdIDs) != len(hds) {
				t.Errorf("%s expect %d handlers,got %d handlers\n", v.name, len(v.hdIDs), len(hds))
			}

			for j, hd := range hds {
				hd.handle(emu)

				hdID := v.hdIDs[j]
				if hd.id != hdID { // validate the control sequences id
					t.Errorf("%s seq=%q expect %s, got %s\n", v.name, v.seq, strHandlerID[hdID], strHandlerID[hd.id])
				}
				// t.Logf("%s seq=%q history=%q\n", v.name, v.seq, hd.sequence)
			}

			// t.Errorf("%s expect %s, got \n%s", v.name, v.graphemes, printCells(emu.cf, v.posY))
			// t.Errorf("%s expect %s, got \n%s", v.name, v.graphemes, printCells(emu.cf, v.posY+1))
			// validate the result with data string
			graphemes := uniseg.NewGraphemes(v.graphemes)
			rows := v.posY
			for j := 0; graphemes.Next(); j++ {
				// the expected content
				chs := graphemes.Runes()

				// get the cell from framebuffer
				cols := v.posX[j]
				if cols == 0 { // change to the next row
					rows += 1
				}
				cell := emu.cf.getCell(rows, cols)

				if cell.contents != string(chs) {
					t.Errorf("%s seq=%q", v.name, v.seq)
					t.Errorf("%s [%2d,%2d] expect %q, got %q\n", v.name, rows, cols, string(chs), cell.contents)
				}
			}
		})
	}
}

func TestHandle_SGR_RGBcolor(t *testing.T) {
	tc := []struct {
		name       string
		hdIDs      []int
		fr, fg, fb int
		br, bg, bb int
		attr       charAttribute
		seq        string
	}{
		{"RGB Color 1", []int{CSI_SGR}, 33, 47, 12, 123, 24, 34, Bold, "\x1B[0;1;38;2;33;47;12;48;2;123;24;34m"},
		{"RGB Color 2", []int{CSI_SGR}, 0, 0, 0, 0, 0, 0, Italic, "\x1B[0;3;38:2:0:0:0;48:2:0:0:0m"},
		{"RGB Color 3", []int{CSI_SGR}, 12, 34, 128, 59, 190, 155, Underlined, "\x1B[0;4;38:2:12:34:128;48:2:59:190:155m"},
	}

	p := NewParser()
	// the default size of emu is 80x40 [colxrow]
	emu := NewEmulator3(8, 4, 4) // this is the pre-condidtion for the test case.

	for _, v := range tc {
		t.Run(v.name, func(t *testing.T) {
			// process control sequence
			hds := make([]*Handler, 0, 16)
			hds = p.processStream(v.seq, hds)

			if len(hds) == 0 {
				t.Errorf("%s got zero handlers.", v.name)
			}

			// reset the renditions
			emu.attrs.renditions = Renditions{}

			// handle the control sequence
			for j, hd := range hds {
				hd.handle(emu)
				if hd.id != v.hdIDs[j] { // validate the control sequences id
					t.Errorf("%s: seq=%q expect %s, got %s\n",
						v.name, v.seq, strHandlerID[v.hdIDs[j]], strHandlerID[hd.id])
				}
			}

			// validate the result
			got := emu.attrs.renditions
			want := Renditions{}
			want.SetBgColor(v.br, v.bg, v.bb)
			want.SetFgColor(v.fr, v.fg, v.fb)
			want.SetAttributes(v.attr, true)

			if got != want {
				t.Errorf("%s:\t %q expect renditions %v, got %v", v.name, v.seq, want, got)
			}
		})
	}
}

func TestHandle_SGR_ANSIcolor(t *testing.T) {
	tc := []struct {
		name  string
		hdIDs []int
		fg    Color
		bg    Color
		attr  charAttribute
		seq   string
	}{
		// here the charAttribute(38) is an unused value, which means nothing for the result.
		{"default Color", []int{CSI_SGR}, ColorDefault, ColorDefault, charAttribute(38), "\x1B[200m"}, // 38,48 is empty charAttribute
		{"8 Color", []int{CSI_SGR}, ColorSilver, ColorBlack, Bold, "\x1B[1;37;40m"},
		{"8 Color 2", []int{CSI_SGR}, ColorMaroon, ColorMaroon, Italic, "\x1B[3;31;41m"},
		{"16 Color", []int{CSI_SGR}, ColorRed, ColorWhite, Underlined, "\x1B[4;91;107m"},
		{"256 Color 1", []int{CSI_SGR}, Color33, Color47, Bold, "\x1B[0;1;38:5:33;48:5:47m"},
		{"256 Color 3", []int{CSI_SGR}, Color128, Color155, Underlined, "\x1B[0;4;38:5:128;48:5:155m"},
	}

	p := NewParser()
	emu := NewEmulator3(8, 4, 4)
	var place strings.Builder
	// this will swallow the output from SGR, such as : attribute not supported.
	emu.logU.SetOutput(&place)

	for _, v := range tc {
		t.Run(v.name, func(t *testing.T) {
			// process control sequence
			hds := make([]*Handler, 0, 16)
			hds = p.processStream(v.seq, hds)

			if len(hds) == 0 {
				t.Errorf("%s got zero handlers.", v.name)
			}

			emu.attrs.renditions = Renditions{}

			// handle the control sequence
			for j, hd := range hds {
				hd.handle(emu)
				if hd.id != v.hdIDs[j] { // validate the control sequences id
					t.Errorf("%s: seq=%q expect %s, got %s\n",
						v.name, v.seq, strHandlerID[v.hdIDs[j]], strHandlerID[hd.id])
				}
			}

			// validate the result
			got := emu.attrs.renditions
			want := Renditions{}
			want.setAnsiForeground(v.fg)
			want.setAnsiBackground(v.bg)
			want.buildRendition(int(v.attr))

			if got != want {
				t.Errorf("%s:\t %q expect renditions %v, got %v", v.name, v.seq, want, got)
			}
		})
	}
}

func TestHandle_SGR_Break(t *testing.T) {
	tc := []struct {
		name  string
		hdIDs []int
		seq   string
	}{
		// the folloiwng test the break case for SGR
		{"break 38    ", []int{CSI_SGR}, "\x1B[38m"},
		{"break 38;   ", []int{CSI_SGR}, "\x1B[38;m"},
		{"break 38:5  ", []int{CSI_SGR}, "\x1B[38;5m"},
		{"break 38:2-1", []int{CSI_SGR}, "\x1B[38:2:23m"},
		{"break 38:2-2", []int{CSI_SGR}, "\x1B[38:2:23:24m"},
		{"break 38:7  ", []int{CSI_SGR}, "\x1B[38;7m"},
		{"break 48    ", []int{CSI_SGR}, "\x1B[48m"},
		{"break 48;   ", []int{CSI_SGR}, "\x1B[48;m"},
		{"break 48:5  ", []int{CSI_SGR}, "\x1B[48;5m"},
		{"break 48:2-1", []int{CSI_SGR}, "\x1B[48:2:23m"},
		{"break 48:2-2", []int{CSI_SGR}, "\x1B[48:2:23:22m"},
		{"break 48:7  ", []int{CSI_SGR}, "\x1B[48;7m"},
	}
	p := NewParser()
	emu := NewEmulator3(8, 4, 4)

	for _, v := range tc {
		t.Run(v.name, func(t *testing.T) {
			// process control sequence
			hds := make([]*Handler, 0, 16)
			hds = p.processStream(v.seq, hds)

			if len(hds) == 0 {
				t.Errorf("%s got zero handlers.", v.name)
			}

			// reset the renditions
			emu.attrs.renditions = Renditions{}

			// handle the control sequence
			for j, hd := range hds {
				hd.handle(emu)
				if hd.id != v.hdIDs[j] { // validate the control sequences id
					t.Errorf("%s: seq=%q expect %s, got %s\n",
						v.name, v.seq, strHandlerID[v.hdIDs[j]], strHandlerID[hd.id])
				}
			}

			// the break case should not affect the renditions, it will keep the same.
			got := emu.attrs.renditions
			want := Renditions{}

			if got != want {
				t.Errorf("%s:\t %q expect renditions \n%v, got \n%v\n", v.name, v.seq, want, got)
			}
		})
	}
}

func TestHandle_ESC_DCS(t *testing.T) {
	tc := []struct {
		name        string
		seq         string
		hdIDs       []int
		wantIndex   int
		wantCharset *map[byte]rune
	}{
		{"VT100 G0", "\x1B(A", []int{ESC_DCS}, 0, &vt_ISO_UK},
		{"VT100 G1", "\x1B)B", []int{ESC_DCS}, 1, nil},
		{"VT220 G2", "\x1B*5", []int{ESC_DCS}, 2, nil},
		{"VT220 G3", "\x1B+%5", []int{ESC_DCS}, 3, &vt_DEC_Supplement},
		{"VT300 G1", "\x1B-0", []int{ESC_DCS}, 1, &vt_DEC_Special},
		{"VT300 G2", "\x1B.<", []int{ESC_DCS}, 2, &vt_DEC_Supplement},
		{"VT300 G3", "\x1B/>", []int{ESC_DCS}, 3, &vt_DEC_Technical},
		{"VT300 G3", "\x1B/A", []int{ESC_DCS}, 3, &vt_ISO_8859_1},
		{"ISO/IEC 2022 G0 A", "\x1B,A", []int{ESC_DCS}, 0, &vt_ISO_UK},
		{"ISO/IEC 2022 G0 >", "\x1B$>", []int{ESC_DCS}, 0, &vt_DEC_Technical},
		// for other charset, just replace it with UTF-8
		{"ISO/IEC 2022 G0 None", "\x1B$%9", []int{ESC_DCS}, 0, nil},
	}

	p := NewParser()
	var place strings.Builder
	p.logT.SetOutput(&place)
	emu := NewEmulator3(8, 4, 0)

	for _, v := range tc {
		t.Run(v.name, func(t *testing.T) {
			// set different value for compare
			for i := 0; i < 4; i++ {
				emu.charsetState.g[i] = nil
			}
			// parse the instruction
			var hd *Handler
			for _, ch := range v.seq {
				hd = p.ProcessInput(ch)
			}
			if hd != nil {
				hd.handle(emu)

				cs := emu.charsetState.g[v.wantIndex]
				if v.hdIDs[0] != hd.id || cs != v.wantCharset {
					t.Errorf("%s: seq=%q handler expect %s, got %s\n", v.name, v.seq, strHandlerID[v.hdIDs[0]], strHandlerID[hd.id])
					t.Errorf("charset expect %p, got %p", v.wantCharset, cs)
				}
			} else {
				t.Errorf("%s got nil return\n", v.name)
			}
		})
	}
}

func TestHandle_DOCS(t *testing.T) {
	tc := []struct {
		name   string
		seq    string
		wantGL int
		wantGR int
		wantSS int
		hdIDs  []int
		msg    string
	}{
		{"set DOCS utf-8       ", "\x1B%G", 0, 2, 0, []int{ESC_DOCS_UTF8}, ""},
		{"set DOCS iso8859-1   ", "\x1B%@", 0, 2, 0, []int{ESC_DOCS_ISO8859_1}, ""},
		{"ESC Percent unhandled", "\x1B%H", 0, 2, 0, nil, "Unhandled input:"},
		{"VT52 ESC G", "\x1B[?2l\x1BG", 0, 2, 0, []int{CSI_privRM, ESC_DOCS_UTF8}, ""},
	}

	p := NewParser()

	var place strings.Builder
	p.logT.SetOutput(&place)
	p.logU.SetOutput(&place)

	emu := NewEmulator3(8, 4, 0)
	for _, v := range tc {
		p.reset()
		place.Reset()
		emu.resetTerminal()

		// set different value
		emu.charsetState.gl = 2
		emu.charsetState.gr = 3
		emu.charsetState.ss = 2

		for i := 0; i < 4; i++ {
			emu.charsetState.g[i] = &vt_DEC_Supplement // Charset_DecSuppl
		}

		// parse control sequence
		hds := make([]*Handler, 0, 16)
		hds = p.processStream(v.seq, hds)

		// check handler length
		if len(hds) == 0 {
			if v.msg != "" {
				if !strings.Contains(place.String(), v.msg) {
					t.Errorf("%s:\t %q expect %q, got %s\n", v.name, v.seq, v.msg, place.String())
				}
				continue
			} else {
				t.Errorf("%s got zero handlers.", v.name)
			}
		}

		// execute the control sequence
		for j, hd := range hds {
			hd.handle(emu)
			if hd.id != v.hdIDs[j] { // validate the control sequences name
				t.Errorf("%s: seq=%q expect %s, got %s\n", v.name, v.seq, strHandlerID[v.hdIDs[j]], strHandlerID[hd.id])
			}
		}

		for i := 0; i < 4; i++ {
			switch i {
			case 2:
				if v.hdIDs[len(v.hdIDs)-1] == ESC_DOCS_ISO8859_1 {
					got := emu.charsetState.g[emu.charsetState.gr]
					if !reflect.DeepEqual(got, &vt_ISO_8859_1) {
						t.Errorf("%s g[gr]= g[%d] expect ISO8859_1.\n", v.name, emu.charsetState.gr)
						t.Errorf("%v vs %v \n", vt_ISO_8859_1, *got)
					}
					break
				}
				fallthrough
			case 0, 1, 3:
				if emu.charsetState.g[i] != nil {
					t.Errorf("%s charset g[%d] should be utf-8.", v.name, i)
				}
			}
		}

		// verify the result
		if emu.charsetState.gl != v.wantGL || emu.charsetState.gr != v.wantGR || emu.charsetState.ss != v.wantSS {
			t.Errorf("%s expect GL,GR,SS= %d,%d,%d, got=%d,%d,%d\n", v.name, v.wantGL, v.wantGR, v.wantSS,
				emu.charsetState.gl, emu.charsetState.gr, emu.charsetState.ss)
		}
	}
}

func TestHandle_LS2_LS3(t *testing.T) {
	tc := []struct {
		name  string
		seq   string
		hdIDs []int
		want  int
	}{
		{"LS2", "\x1Bn", []int{ESC_LS2}, 2},
		{"LS3", "\x1Bo", []int{ESC_LS3}, 3},
	}

	p := NewParser()
	emu := NewEmulator3(80, 40, 500)
	var place strings.Builder
	emu.logT.SetOutput(&place)

	for _, v := range tc {

		// reset the charsetState
		emu.charsetState.gl = 0

		// parse the instruction
		var hd *Handler
		for _, ch := range v.seq {
			hd = p.ProcessInput(ch)
		}

		// call the handler
		if hd != nil {
			hd.handle(emu)

			// verify the result
			if emu.charsetState.gl != v.want || hd.id != v.hdIDs[0] {
				t.Errorf("%s seq=%q handler expect %s, got %s\n", v.name, v.seq, strHandlerID[v.hdIDs[0]], strHandlerID[hd.id])
				t.Errorf("%s GL expect %d, got %d\n", v.name, v.want, emu.charsetState.gl)
			}
		} else {
			t.Errorf("%s got nil return\n", v.name)
		}
	}
}

func TestHandle_LS1R_LS2R_LS3R(t *testing.T) {
	tc := []struct {
		name  string
		seq   string
		hdIDs []int
		want  int
	}{
		{"LS1R", "\x1B~", []int{ESC_LS1R}, 1},
		{"LS2R", "\x1B}", []int{ESC_LS2R}, 2},
		{"LS3R", "\x1B|", []int{ESC_LS3R}, 3},
	}

	p := NewParser()
	emu := NewEmulator3(80, 40, 500)
	var place strings.Builder
	emu.logT.SetOutput(&place)

	for _, v := range tc {

		// reset the charsetState
		emu.charsetState.gr = 0

		// parse the instruction
		var hd *Handler
		for _, ch := range v.seq {
			hd = p.ProcessInput(ch)
		}

		// call the handler
		if hd != nil {
			hd.handle(emu)

			// verify the result
			if emu.charsetState.gr != v.want || hd.id != v.hdIDs[0] {
				t.Errorf("%s seq=%q handler expect %s, got %s\n", v.name, v.seq, strHandlerID[v.hdIDs[0]], strHandlerID[hd.id])
				t.Errorf("%s GR expect %d, got %d\n", v.name, v.want, emu.charsetState.gr)
			}

		} else {
			t.Errorf("%s got nil return\n", v.name)
		}

	}
}

func TestHandle_SS2_SS3(t *testing.T) {
	tc := []struct {
		name  string
		seq   string
		hdIDs []int
		want  int
	}{
		{"SS2", "\x1BN", []int{ESC_SS2}, 2}, // G2 single shift
		{"SS3", "\x1BO", []int{ESC_SS3}, 3}, // G3 single shift
	}

	p := NewParser()
	emu := NewEmulator3(8, 4, 4)
	for _, v := range tc {

		// reset the charsetState
		emu.charsetState.ss = 0

		// parse the instruction
		var hd *Handler
		for _, ch := range v.seq {
			hd = p.ProcessInput(ch)
		}

		// call the handler
		if hd != nil {
			hd.handle(emu)

			// verify the result
			if emu.charsetState.ss != v.want || hd.id != v.hdIDs[0] {
				t.Errorf("%s seq=%q expect handler %s, got %s\n", v.name, v.seq, strHandlerID[v.hdIDs[0]], strHandlerID[hd.id])
				t.Errorf("SS expect %d, got %d\n", v.want, emu.charsetState.ss)
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
		msg  string // test traceNormalInput()
	}{
		{"SO", 0x0E, 1, "Input:['\x0e'] inputOps="}, // G1 as GL
		{"SI", 0x0F, 0, "Input:['\x0f'] inputOps="}, // G0 as GL
	}

	p := NewParser()
	var place strings.Builder // all the message is output to herer
	p.logTrace = true
	p.logT.SetOutput(&place)

	emu := NewEmulator3(8, 4, 4)
	for _, v := range tc {
		place.Reset()

		hd := p.ProcessInput(v.r)
		if hd != nil {
			hd.handle(emu)

			if emu.charsetState.gl != v.want {
				t.Errorf("%s expect %d, got %d\n", v.name, v.want, emu.charsetState.gl)
			}
			if strings.Contains(place.String(), v.msg) {
				t.Errorf("msg expect %s, got %s\n", v.msg, place.String())
			}
		} else {
			t.Errorf("%s got nil return\n", v.name)
		}

	}
}

func TestHandle_CUP(t *testing.T) {
	tc := []struct {
		name    string
		startX  int
		startY  int
		hdIDs   int
		wantY   int
		wantX   int
		seq     string
		warnStr string
	}{
		{"CSI Ps;PsH normal", 10, 10, CSI_CUP, 23, 13, "\x1B[24;14H", "Cursor positioned to"},
		{"CSI Ps;PsH default", 10, 10, CSI_CUP, 0, 0, "\x1B[H", "Cursor positioned to"},
		{"CSI Ps;PsH second default", 10, 10, CSI_CUP, 0, 0, "\x1B[1H", "Cursor positioned to"},
		{"CSI Ps;PsH outrange active area", 10, 10, CSI_CUP, 39, 79, "\x1B[42;89H", "Cursor positioned to"},
	}
	p := NewParser()

	emu := NewEmulator3(80, 40, 500)
	var place strings.Builder
	emu.logT.SetOutput(&place)

	for _, v := range tc {
		var hd *Handler

		// parse the sequence
		for _, ch := range v.seq {
			hd = p.ProcessInput(ch)
		}
		if hd == nil {
			t.Errorf("%s got nil Handler\n", v.name)
			continue
		}

		// reset the cursor position
		emu.posY = v.startY
		emu.posX = v.startX

		// handle the instruction
		hd.handle(emu)
		if hd.id != v.hdIDs {
			t.Errorf("%s seq=%q expect %s, got %s\n", v.name, v.seq, strHandlerID[v.hdIDs], strHandlerID[hd.id])
		}

		// get the result
		gotY := emu.posY
		gotX := emu.posX

		if gotX != v.wantX || gotY != v.wantY {
			t.Errorf("%s expect cursor position (%d,%d), got (%d,%d)\n",
				v.name, v.wantX, v.wantY, gotX, gotY)
		}

		if !strings.Contains(place.String(), v.warnStr) {
			t.Errorf("%s seq=%q expect %q, got %q\n", v.name, v.seq, v.warnStr, place.String())
		}
	}
}

func TestHandle_BEL(t *testing.T) {
	seq := "\x07"
	emu := NewEmulator3(8, 4, 4)

	hds := emu.handleStream(seq)

	if len(hds) == 0 {
		t.Errorf("BEL got nil for seq=%q\n", seq)
	}

	bellCount := emu.cf.GetBellCount()
	if bellCount == 0 || hds[0].id != C0_BEL {
		t.Errorf("BEL expect %d, got %d\n", 1, bellCount)
		t.Errorf("BEL expect %s, got %s\n", strHandlerID[C0_BEL], strHandlerID[hds[0].id])
	}
}

func TestHandle_RI_NEL(t *testing.T) {
	tc := []struct {
		name       string
		seq        string
		wantY      int
		wantX      int
		hdIDs      []int
		scrollHead int
	}{
		{"RI ", "\x1B[11;6H\x1BM", 9, 5, []int{CSI_CUP, ESC_RI}, 0},   // move cursor up to the previouse row
		{"RI ", "\x1B[1;6H\x1BM", 0, 5, []int{CSI_CUP, ESC_RI}, 39},   // move cursor up to the previouse row, scroll down
		{"NEL", "\x1B[11;6H\x1BE", 11, 0, []int{CSI_CUP, ESC_NEL}, 0}, // move cursor down to next row, may scroll up
		{"VT52 CUP no parameter", "\x1B[?2l\x1BH", 0, 0, []int{CSI_privRM, CSI_CUP}, 0},
		{"VT52 CUP 5,5", "\x1B[?2l\x1BY%%", 5, 5, []int{CSI_privRM, CSI_CUP}, 0}, // % is 37, check ascii table
		{"VT52 RI ", "\x1B[11;6H\x1B[?2l\x1BI", 9, 5, []int{CSI_CUP, CSI_privRM, ESC_RI}, 0},
	}

	p := NewParser()
	emu := NewEmulator3(40, 20, 20)
	var place strings.Builder
	emu.logI.SetOutput(&place) // redirect the output to the string builder
	emu.logT.SetOutput(&place) // redirect the output to the string builder

	for _, v := range tc {
		p.reset()
		emu.resetTerminal()

		// parse control sequence
		hds := make([]*Handler, 0, 16)
		hds = p.processStream(v.seq, hds)

		if len(hds) < 2 {
			t.Errorf("%s got %d handlers.", v.name, len(hds))
		}

		// handle the control sequence
		for j, hd := range hds {
			hd.handle(emu)
			if hd.id != v.hdIDs[j] { // validate the control sequences id
				t.Errorf("%s: seq=%q expect %s, got %s\n", v.name, v.seq, strHandlerID[v.hdIDs[j]], strHandlerID[hd.id])
			}
		}

		gotY := emu.posY
		gotX := emu.posX

		if gotX != v.wantX || gotY != v.wantY {
			t.Errorf("%s seq=%q expect cursor position (%d,%d), got (%d,%d)\n",
				v.name, v.seq, v.wantY, v.wantX, gotY, gotX)
		}

		scrollHead := emu.cf.scrollHead
		if scrollHead != v.scrollHead {
			t.Errorf("%s scrollHead expect %d, got %d\n", v.name, v.scrollHead, scrollHead)
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
		{"overflow DCS string", "\x1BP", 0, InputState_Normal}, // special logic in the following test code, add 4K string
		{"overflow history", "\x1BP", 0, InputState_Normal},    // special logic in the following test code, add 4K string
		{"CSI unknow ", "\x1B[x", 0, InputState_Normal},        // unhandled CSI sequence
		{"CSI ? unknow ", "\x1B[?x", 0, InputState_Normal},     // unhandled CSI ? sequence
		{"CSI ? ESC ", "\x1B[?\x1B", 0, InputState_Normal},     // unhandled CSI ? ESC
		{"CSI ! unknow ", "\x1B[!x", 0, InputState_Normal},     // unhandled CSI ! x
	}

	p := NewParser()
	var place strings.Builder

	p.logE.SetOutput(&place)
	p.logU.SetOutput(&place)
	// p.logTrace = true // open the trace
	var hd *Handler
	for _, v := range tc {
		place.Reset()

		t.Run(v.name, func(t *testing.T) {
			raw := v.seq
			// special logic for the overflow case.
			if strings.HasPrefix(v.name, "overflow") {
				var b strings.Builder
				b.WriteString(v.seq)        // the header
				for i := 0; i < 1024; i++ { // just 4096
					b.WriteString("blab")
				}
				raw = b.String()
				// t.Logf("%d\n", len(raw)-2) // OSC prefix takes two runes
			}

			for _, ch := range raw {
				hd = p.ProcessInput(ch)
			}

			if hd == nil {
				if p.inputState != v.state || p.nInputOps != v.nInputOps {
					t.Errorf("%s seq=%q expect state=%q, nInputOps=%d, got state=%q, nInputOps=%d\n",
						v.name, v.seq, strInputState[v.state], v.nInputOps, strInputState[p.inputState], p.nInputOps)
				}
				// overflow logic
				// each overflow warn message contains at least one history warn message
				if strings.HasPrefix(v.name, "overflow") && !strings.Contains(place.String(), "overflow") {
					t.Errorf("%s seq=%q should contains %q\n, got=%s\n", v.name, v.seq, "overflow", place.String())
				}
			} else {
				t.Errorf("%s should get nil handler, got %s, history=%q\n", v.name, hd.name, p.historyString())
			}
		})
	}
}

func TestHandle_priDA_secDA_DSR(t *testing.T) {
	tc := []struct {
		name     string
		seq      string
		wantResp string
		hdIDs    []int
	}{
		{"Primary DA  ", "\x1B[c", fmt.Sprintf("\x1B[?%s", DEVICE_ID), []int{CSI_priDA}},
		{"Secondary DA", "\x1B[>c", "\x1B[>64;0;0c", []int{CSI_secDA}},
		{"DSR device status report ", "\x1B[5n", "\x1B[0n", []int{CSI_DSR}},
		// use DECSET 6 to set  originMode, use CUP to set the active position, then call DSR 6
		{"DSR OriginMode_ScrollingRegion", "\x1B[?6h\x1B[9;9H\x1B[6n", "\x1B[9;9R", []int{CSI_privSM, CSI_CUP, CSI_DSR}},
		// use DECRST 6 to set  originMode, use CUP to set the active position, then call DSR 6
		{"DSR OriginMode_Absolute", "\x1B[?6l\x1B[10;10H\x1B[6n", "\x1B[10;10R", []int{CSI_privRM, CSI_CUP, CSI_DSR}},
		// TODO full test for scrolling mode
	}

	p := NewParser()
	emu := NewEmulator3(80, 40, 0)

	var place strings.Builder
	emu.logI.SetOutput(&place) // redirect the output to the string builder
	emu.logT.SetOutput(&place) // redirect the output to the string builder

	for _, v := range tc {
		// reset the target content
		emu.terminalToHost.Reset()

		t.Run(v.name, func(t *testing.T) {
			// parse control sequence
			hds := make([]*Handler, 0, 16)
			hds = p.processStream(v.seq, hds)

			if len(hds) < 1 {
				t.Errorf("%s got %d handlers.", v.name, len(hds))
			}

			// handle the control sequence
			for j, hd := range hds {
				hd.handle(emu)
				if hd.id != v.hdIDs[j] { // validate the control sequences id
					t.Errorf("%s: seq=%q expect %s, got %s\n", v.name, v.seq, strHandlerID[v.hdIDs[j]], strHandlerID[hd.id])
				}
			}

			got := emu.terminalToHost.String()
			if v.wantResp != got {
				t.Errorf("%s seq:%q expect %q, got %q\n", v.name, v.seq, v.wantResp, got)
			}
		})
	}
}

// calculate the cell content in row, based on y2,y1 value
// func getCellAtRow(y1, y2 int, row int) string {
// 	if y2 < y1 {
// 		return "_"
// 	}
//
// 	gap := y2 - y1 + 1
// 	if y1 == 0 {
// 		gap *= -1
// 	}
//
// 	ch := rune(0x30 + row + gap)
// 	return string(ch)
// }

func TestHandle_VPA_VPR_CHA_HPA_HPR_CNL_CPL(t *testing.T) {
	tc := []struct {
		name         string
		hdIDs        []int
		wantY, wantX int
		seq          string
	}{
		{"VPA move cursor to row 2 ", []int{CSI_CUP, CSI_VPA}, 2, 9, "\x1B[9;10H\x1B[3d"},
		{"VPA move cursor to row 33", []int{CSI_CUP, CSI_VPA}, 33, 8, "\x1B[9;9H\x1B[34d"},
		{"VPR move cursor to row 12", []int{CSI_CUP, CSI_VPR}, 9, 8, "\x1B[9;9H\x1B[e"},
		{"VPR move cursor to row 39", []int{CSI_CUP, CSI_VPR}, 39, 8, "\x1B[9;9H\x1B[40e"},
		{"CHA move cursor to col 0 ", []int{CSI_CUP, CSI_CHA}, 7, 0, "\x1B[8;8H\x1B[G"}, // default Ps is 1
		{"CHA move cursor to col 78", []int{CSI_CUP, CSI_CHA}, 6, 78, "\x1B[7;7H\x1B[79G"},
		{"HPA move cursor to col 8 ", []int{CSI_CUP, CSI_HPA}, 5, 8, "\x1B[6;6H\x1B[9`"},
		{"HPA move cursor to col 79", []int{CSI_CUP, CSI_HPA}, 4, 79, "\x1B[5;5H\x1B[99`"},
		{"HPR move cursor to col 5 ", []int{CSI_CUP, CSI_HPR}, 4, 5, "\x1B[5;5H\x1B[a"},
		{"HPR move cursor to col 39", []int{CSI_CUP, CSI_HPR}, 4, 79, "\x1B[5;5H\x1B[79a"},
		{"CNL move cursor to (5,0) ", []int{CSI_CUP, CSI_CNL}, 5, 0, "\x1B[5;5H\x1B[E"},
		{"CNL move cursor to (39,0)", []int{CSI_CUP, CSI_CNL}, 39, 0, "\x1B[5;5H\x1B[79E"},
		{"CPL move cursor to (3,0) ", []int{CSI_CUP, CSI_CPL}, 3, 0, "\x1B[5;5H\x1B[F"},
		{"CPL move cursor to (0,0) ", []int{CSI_CUP, CSI_CPL}, 0, 0, "\x1B[5;5H\x1B[20F"},
	}
	p := NewParser()
	emu := NewEmulator3(80, 40, 0)
	var place strings.Builder
	emu.logT.SetOutput(&place) // swallow the output

	for _, v := range tc {

		hds := make([]*Handler, 0, 16)
		hds = p.processStream(v.seq, hds)

		if len(hds) == 0 {
			t.Errorf("%s got zero handlers.", v.name)
		}

		// handle the control sequence
		for j, hd := range hds {
			hd.handle(emu)
			if hd.id != v.hdIDs[j] { // validate the control sequences id
				t.Errorf("%s: seq=%q expect %s, got %s\n", v.name, v.seq, strHandlerID[v.hdIDs[j]], strHandlerID[hd.id])
			}
		}

		gotX := emu.posX
		gotY := emu.posY

		if v.wantX != gotX || v.wantY != gotY {
			t.Errorf("%s seq=%q cursor expect (%d,%d), got (%d,%d)\n", v.name, v.seq, v.wantY, v.wantX, gotY, gotX)
		}
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
			if v.want != p.getHistoryAt(v.reverseIdx) {
				t.Errorf("%s expect reverseIdx[%d] value=%q, got %q\n", v.name, v.reverseIdx, v.want, p.getHistoryAt(v.reverseIdx))
			}
		})
	}
}

func TestHistoryReset(t *testing.T) {
	tc := []struct {
		name    string
		seq     string
		history string
	}{
		{"unhandled sequence", "\x1B[23;24i", "\x1B[23;24i"},
	}
	p := NewParser()
	var place strings.Builder
	p.logU.SetOutput(&place) // redirect the output to the string builder

	for _, v := range tc {
		// reset the output
		place.Reset()

		// process control sequence
		hds := make([]*Handler, 0, 16)
		hds = p.processStream(v.seq, hds)

		if len(hds) != 0 {
			t.Errorf("%s expect %d handlers.", v.name, len(hds))
		}

		if !strings.Contains(place.String(), fmt.Sprintf("%q", v.history)) {
			t.Errorf("%s:\t %q expect %q, got %s\n", v.name, v.seq, v.history, place.String())
		}
	}
}

func TestHandle_DECSTBM(t *testing.T) {
	tc := []struct {
		name        string
		seq         string
		hdIDs       []int
		top, bottom int
		posX, posY  int
		logMessage  string
	}{
		{ // move the cursor to 23,13 first then set new top/bottom margin
			"DECSTBM ", "\x1B[24;14H\x1B[2;30r",
			[]int{CSI_CUP, CSI_DECSTBM},
			2 - 1, 30, 0, 0, "",
		},
		{ // CUP, then a successful STBM follow an ignored STBM.
			"DECSTBM ", "\x1B[2;6H\x1B[3;32r\x1B[32;30r",
			[]int{CSI_CUP, CSI_DECSTBM, CSI_DECSTBM},
			3 - 1, 32, 0, 0, "Illegal arguments to SetTopBottomMargins:",
		},
		{
			"DECSTBM no parameters",
			"\x1B[2;6H\x1B[r",
			[]int{CSI_CUP, CSI_DECSTBM},
			0, 40, 0, 0, "",
		},
		{ // CUP, then a successful STBM follow a reset STBM
			"DECSTBM reset margin", "\x1B[2;6H\x1B[3;36r\x1B[1;40r",
			[]int{CSI_CUP, CSI_DECSTBM, CSI_DECSTBM},
			0, 40, 0, 0, "",
		},
	}

	p := NewParser()

	emu := NewEmulator3(80, 40, 500)
	var place strings.Builder
	emu.logT.SetOutput(&place)

	for k, v := range tc {
		// reset the log content
		place.Reset()

		// parse control sequence
		hds := make([]*Handler, 0, 16)
		hds = p.processStream(v.seq, hds)

		if len(hds) == 0 {
			t.Errorf("%s got zero handlers.", v.name)
		}

		// handle the control sequence
		for j, hd := range hds {
			hd.handle(emu)
			if hd.id != v.hdIDs[j] { // validate the control sequences id
				t.Errorf("%s:\t %q expect %s, got %s\n", v.name, v.seq, strHandlerID[v.hdIDs[j]], strHandlerID[hd.id])
			}
		}

		// get the new top/bottom
		top := emu.marginTop
		bottom := emu.marginBottom

		// get the new cursor position
		y := emu.posY
		x := emu.posX

		if x != v.posX || y != v.posY || top != v.top || bottom != v.bottom {
			t.Errorf("%s: %q expect cursor in (%d,%d), got (%d,%d)\n", v.name, v.seq, v.posY, v.posX, y, x)
			t.Errorf("%s: %q expect [top:bottom] [%d,%d], got [%d,%d]\n", v.name, v.seq, v.top, v.bottom, top, bottom)
		}

		switch k {
		case 1:
			if !strings.Contains(place.String(), v.logMessage) {
				t.Errorf("%s seq=%q expect output=%q, got %q\n", v.name, v.seq, v.logMessage, place.String())
			}
		default:
		}
	}
}

func TestHandle_DECSCL(t *testing.T) {
	tc := []struct {
		name     string
		seq      string
		hdIDs    []int
		cmpLevel CompatibilityLevel
		msg      string
	}{
		{"CompatLevel VT100 param 61", "\x1B[61\"p", []int{CSI_DECSCL}, CompatLevel_VT100, ""},
		{"CompatLevel VT400 param 62", "\x1B[62\"p", []int{CSI_DECSCL}, CompatLevel_VT400, ""},
		{"CompatLevel VT400 param 63", "\x1B[63\"p", []int{CSI_DECSCL}, CompatLevel_VT400, ""},
		{"CompatLevel VT400 param 64", "\x1B[64\"p", []int{CSI_DECSCL}, CompatLevel_VT400, ""},
		{"CompatLevel VT400 param 65", "\x1B[65\"p", []int{CSI_DECSCL}, CompatLevel_VT400, ""},
		{"CompatLevel VT400 DECANM  ", "\x1B<", []int{ESC_DECANM}, CompatLevel_VT400, ""},
		{"VT52 CompatLevel VT100    ", "\x1B[?2l\x1B<", []int{CSI_privRM, ESC_DECANM}, CompatLevel_VT100, ""},
		{"CompatLevel others        ", "\x1B[66\"p", []int{CSI_DECSCL}, CompatLevel_Unused, "compatibility mode:"},
		{"CompatLevel 8-bit control ", "\x1B[65;0\"p", []int{CSI_DECSCL}, CompatLevel_Unused, "DECSCL: 8-bit controls"},
		{"CompatLevel 8-bit control ", "\x1B[61;2\"p", []int{CSI_DECSCL}, CompatLevel_Unused, "DECSCL: 8-bit controls"},
		{"CompatLevel 7-bit control ", "\x1B[65;1\"p", []int{CSI_DECSCL}, CompatLevel_Unused, "DECSCL: 7-bit controls"},
		{"CompatLevel outof range   ", "\x1B[65;3\"p", []int{CSI_DECSCL}, CompatLevel_Unused, "DECSCL: C1 control transmission mode:"},
		{"CompatLevel unhandled", "\x1B[65;3\"q", []int{CSI_DECSCL}, CompatLevel_Unused, "Unhandled input:"},
	}

	emu := NewEmulator3(8, 4, 0)
	p := NewParser()
	var place strings.Builder
	// redirect the output to the string builder
	emu.logU.SetOutput(&place)
	emu.logT.SetOutput(&place)

	p.logU.SetOutput(&place)

	for i, v := range tc {
		// reset the output
		place.Reset()
		emu.compatLevel = CompatLevel_Unused

		// process control sequence
		hds := make([]*Handler, 0, 16)
		hds = p.processStream(v.seq, hds)

		// handle the control sequence
		for j, hd := range hds {
			hd.handle(emu)
			if hd.id != v.hdIDs[j] { // validate the control sequences name
				t.Errorf("%s:\t %q expect %s, got %s\n", v.name, v.seq, strHandlerID[v.hdIDs[j]], strHandlerID[hd.id])
			}
			// t.Logf("%s seq=%q\n", v.name, hd.sequence)
		}

		switch i {
		case 0, 1, 2, 3, 4, 5, 6:
			got := emu.compatLevel
			if got != v.cmpLevel {
				t.Errorf("%s:\t %q, expect %d, got %d\n", v.name, v.seq, v.cmpLevel, got)
			}
		default:
			if !strings.Contains(place.String(), v.msg) {
				t.Errorf("%s:\t %q expect %q, got %s\n", v.name, v.seq, v.msg, place.String())
			}
		}
	}
}

func TestHandle_ecma48_SL_SR_FI_BI(t *testing.T) {
	tc := []struct {
		name      string
		hdIDs     []int
		tlY, tlX  int // damage area top/left
		brY, brX  int // damage area bottom/right
		seq       string
		emptyCols []int // empty columens
	}{
		{"ecma48 SL 2 cols", []int{ESC_DECALN, CSI_ECMA48_SL}, 0, 0, 3, 7, "\x1B#8\x1B[2 @", []int{6, 7}},
		{"ecma48 SL 1 col ", []int{ESC_DECALN, CSI_ECMA48_SL}, 0, 0, 3, 7, "\x1B#8\x1B[ @", []int{7}},
		{"ecma48 SL all cols", []int{ESC_DECALN, CSI_ECMA48_SL}, 0, 0, 3, 7, "\x1B#8\x1B[9 @", []int{0, 1, 2, 3, 4, 5, 6, 7}},
		{"ecma48 SR 4 cols", []int{ESC_DECALN, CSI_ECMA48_SR}, 0, 0, 3, 7, "\x1B#8\x1B[4 A", []int{0, 1, 2, 3}},
		{"ecma48 SR 1 cols", []int{ESC_DECALN, CSI_ECMA48_SR}, 0, 0, 3, 7, "\x1B#8\x1B[ A", []int{0}},
		{"ecma48 SR all cols", []int{ESC_DECALN, CSI_ECMA48_SR}, 0, 0, 3, 7, "\x1B#8\x1B[9 A", []int{0, 1, 2, 3, 4, 5, 6, 7}},
		{"DECFI 1 cols", []int{ESC_DECALN, CSI_CUP, ESC_FI}, 0, 0, 3, 7, "\x1B#8\x1B[4;8H\x1B9", []int{7}},
		{"DECBI 1 cols", []int{ESC_DECALN, CSI_CUP, ESC_BI}, 0, 0, 3, 7, "\x1B#8\x1B[4;1H\x1B6", []int{0}},
	}

	p := NewParser()
	emu := NewEmulator3(8, 4, 4) // this is the pre-condidtion for the test case.
	var place strings.Builder
	emu.logI.SetOutput(&place)
	emu.logT.SetOutput(&place)

	for _, v := range tc {

		hds := make([]*Handler, 0, 16)
		hds = p.processStream(v.seq, hds)

		if len(hds) == 0 {
			t.Errorf("%s got zero handlers.", v.name)
		}

		before := ""

		// call the handler
		for j, hd := range hds {
			hd.handle(emu)
			if hd.id != v.hdIDs[j] { // validate the control sequences id
				t.Errorf("%s: seq=%q expect %s, got %s\n",
					v.name, v.seq, strHandlerID[v.hdIDs[j]], strHandlerID[hd.id])
			}
			if j == 0 { // print the screen before modify it.
				before = printCells(emu.cf)
				emu.cf.damage.reset()
			}
		}
		after := printCells(emu.cf)

		// calculate the expected dmage area
		dmg := Damage{}
		dmg.totalCells = emu.cf.damage.totalCells
		dmg.start, dmg.end = damageArea(emu.cf, v.tlY, v.tlX, v.brY, v.brX+1) // the end point is exclusive.

		if emu.cf.damage != dmg || !isEmptyCols(emu.cf, v.emptyCols...) {
			t.Errorf("%s seq=%q\n", v.name, v.seq)
			t.Errorf("expect damage %v, got %v\n", dmg, emu.cf.damage)
			t.Errorf("columens %v is empty = %t\n", v.emptyCols, isEmptyCols(emu.cf, v.emptyCols...))
			t.Errorf("[before]\n%s", before)
			t.Errorf("[after ]\n%s", after)
		}
	}
}

func TestHandle_XTMMODEKEYS(t *testing.T) {
	tc := []struct {
		name            string
		hdIDs           []int
		seq             string
		modifyOtherKeys uint
		msg             string
	}{
		{"XTMODEKEYS 0:x    ", []int{CSI_XTMODKEYS}, "\x1B[>0;1m", 3, "XTMODKEYS: modifyKeyboard ="},
		{"XTMODEKEYS 0:break", []int{CSI_XTMODKEYS}, "\x1B[>0;0m", 3, ""},
		{"XTMODEKEYS 1:x    ", []int{CSI_XTMODKEYS}, "\x1B[>1;1m", 3, "XTMODKEYS: modifyCursorKeys ="},
		{"XTMODEKEYS 1:break", []int{CSI_XTMODKEYS}, "\x1B[>1;2m", 3, ""},
		{"XTMODEKEYS 2:x    ", []int{CSI_XTMODKEYS}, "\x1B[>2;1m", 3, "XTMODKEYS: modifyFunctionKeys ="},
		{"XTMODEKEYS 2:break", []int{CSI_XTMODKEYS}, "\x1B[>2;2m", 3, ""},
		{"XTMODEKEYS 4:x    ", []int{CSI_XTMODKEYS}, "\x1B[>4;2m", 2, "XTMODKEYS: modifyOtherKeys set to"},
		{"XTMODEKEYS 4:break", []int{CSI_XTMODKEYS}, "\x1B[>4;3m", 3, "XTMODKEYS: illegal argument for modifyOtherKeys:"},
		{"XTMODEKEYS 1 parameter", []int{CSI_XTMODKEYS}, "\x1B[>4m", 0, "XTMODKEYS: modifyOtherKeys set to"},
		{"XTMODEKEYS 0 parameter", []int{CSI_XTMODKEYS}, "\x1B[>m", 3, ""}, // no parameter
	}

	p := NewParser()
	emu := NewEmulator3(8, 4, 4)
	var place strings.Builder // all the message is output to herer
	emu.logU.SetOutput(&place)
	emu.logT.SetOutput(&place)
	emu.logI.SetOutput(&place)

	for _, v := range tc {
		// reset the output
		place.Reset()

		hds := make([]*Handler, 0, 16)
		hds = p.processStream(v.seq, hds)

		if len(hds) == 0 {
			t.Errorf("%s got zero handlers.", v.name)
		}

		// call the handler
		for j, hd := range hds {
			hd.handle(emu)
			if hd.id != v.hdIDs[j] { // validate the control sequences id
				t.Errorf("%s: seq=%q expect %s, got %s\n",
					v.name, v.seq, strHandlerID[v.hdIDs[j]], strHandlerID[hd.id])
			}
		}

		// validate the output message
		if v.msg != "" && !strings.Contains(place.String(), v.msg) {
			t.Errorf("%s seq=%q output: expect %s, got %s\n", v.name, v.seq, v.msg, place.String())
		}

		// validate the data changed
		got := emu.modifyOtherKeys
		if v.modifyOtherKeys != 3 && got != v.modifyOtherKeys {
			t.Errorf("%s seq=%q modifyOtherKeys: expect %d, got %d\n", v.name, v.seq, v.modifyOtherKeys, got)
		}
	}
}

func TestHandle_XTWINOPS(t *testing.T) {
	seq := "\x1B[t"
	p := NewParser()
	hds := make([]*Handler, 0, 16)
	hds = p.processStream(seq, hds)

	if len(hds) != 0 {
		t.Errorf("XTWINOPS seq=%q expect zero handlers.", seq)
	}
}
