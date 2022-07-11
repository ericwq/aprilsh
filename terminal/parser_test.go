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
			[]int{csi_cup, graphemes},
			14, 0,
			[]int{13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25},
			"long long ago",
		},
		{
			"UTF-8 chinese, combining character and flags",
			"\x1B[2;30HChin\u0308\u0308a 🏖 i国旗🇳🇱Fun 🌈with Flag🇧🇷.s",
			[]int{csi_cup, graphemes},
			30, 1,
			[]int{29, 30, 31, 32, 33, 34, 35, 37, 38, 39, 41, 43, 45, 46, 47, 48, 49, 51, 52, 53, 54, 55, 56, 57, 58, 59, 60, 62, 63},
			"Chin\u0308\u0308a 🏖 i国旗🇳🇱Fun 🌈with Flag🇧🇷.s",
		},
		{
			"VT mix UTF-8",
			"\x1B[3;24H中国\x1B%@\xA5AB\xe2\xe3\xe9\x1B%GShanghai\x1B%@CD\xe0\xe1",
			[]int{
				csi_cup, graphemes, graphemes, esc_docs_iso8859_1, graphemes, graphemes, graphemes, graphemes, graphemes,
				graphemes, esc_docs_utf8, graphemes, graphemes, graphemes, graphemes, graphemes, graphemes, graphemes,
				graphemes, esc_docs_iso8859_1, graphemes, graphemes, graphemes, graphemes,
			},
			24, 2,
			[]int{23, 25, 27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38, 39, 40, 41, 42, 43, 44},
			"中国¥ABâãéShanghaiCDàá",
		},
		{
			"VT edge", "\x1B[4;10H\x1B%@Beijing\x1B%G",
			[]int{csi_cup, esc_docs_iso8859_1, graphemes, graphemes, graphemes, graphemes, graphemes, graphemes, graphemes, esc_docs_utf8},
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
	}

	p := NewParser()
	emu := NewEmulator3(80, 40, 40)
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

			// t.Logf("%s expect %s, got \n%s", v.name, v.graphemes, printCells(emu.cf, v.posY))
			// t.Logf("%s expect %s, got \n%s", v.name, v.graphemes, printCells(emu.cf, v.posY+1))

			// validate the result with data string
			graphemes := uniseg.NewGraphemes(v.graphemes)
			rows := v.posY
			index := 0
			for graphemes.Next() {
				// the expected content
				chs := graphemes.Runes()

				// get the cell from framebuffer
				cols := v.posX[index]
				if cols == 0 { // change to the next row
					rows += 1
				}
				cell := emu.cf.getCell(rows, cols)

				if cell.contents != string(chs) {
					t.Errorf("%s:\t cell [%2d,%2d] expect %q, got %q\n", v.name, rows, cols, string(chs), cell.contents)
				}

				index += 1
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
			[]int{csi_cup, graphemes, csi_rep},
			7,
			[]int{78, 79},
			"p\u0308p\u0308",
		},
		{
			"chinese even REP+wrap", "\x1B[9;79H四\x1B[5b",
			[]int{csi_cup, graphemes, csi_rep},
			8,
			[]int{78, 0, 2, 4, 6, 8},
			"四四四四四四",
		},
		{
			"chinese odd REP+wrap", "\x1B[10;79H#海\x1B[5b",
			[]int{csi_cup, graphemes, graphemes, csi_rep},
			9,
			[]int{78, 0, 2, 4, 6, 8, 10},
			"#海海海海海海",
		},
		{
			"insert REP+wrap", "\x1B[4h\x1B[11;78H#\x1B[5b",
			[]int{csi_sm, csi_cup, graphemes, csi_rep},
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

func TestHandle_ESC_DCS(t *testing.T) {
	tc := []struct {
		name        string
		seq         string
		hdIDs       []int
		wantIndex   int
		wantCharset *map[byte]rune
	}{
		{"VT100 G0", "\x1B(A", []int{esc_dcs}, 0, &vt_ISO_UK},
		{"VT100 G1", "\x1B)B", []int{esc_dcs}, 1, nil},
		{"VT220 G2", "\x1B*5", []int{esc_dcs}, 2, nil},
		{"VT220 G3", "\x1B+%5", []int{esc_dcs}, 3, &vt_DEC_Supplement},
		{"VT300 G1", "\x1B-0", []int{esc_dcs}, 1, &vt_DEC_Special},
		{"VT300 G2", "\x1B.<", []int{esc_dcs}, 2, &vt_DEC_Supplement},
		{"VT300 G3", "\x1B/>", []int{esc_dcs}, 3, &vt_DEC_Technical},
		{"VT300 G3", "\x1B/A", []int{esc_dcs}, 3, &vt_ISO_8859_1},
		{"ISO/IEC 2022 G0 A", "\x1B,A", []int{esc_dcs}, 0, &vt_ISO_UK},
		{"ISO/IEC 2022 G0 >", "\x1B$>", []int{esc_dcs}, 0, &vt_DEC_Technical},
		// for other charset, just replace it with UTF-8
		{"ISO/IEC 2022 G0 None", "\x1B$%9", []int{esc_dcs}, 0, nil},
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
				hd = p.processInput(ch)
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
		name    string
		seq     string
		wantGL  int
		wantGR  int
		wantSS  int
		hdIDs   int
		warnStr string
	}{
		{"set DOCS utf-8       ", "\x1B%G", 0, 2, 0, esc_docs_utf8, ""},
		{"set DOCS iso8859-1   ", "\x1B%@", 0, 2, 0, esc_docs_iso8859_1, ""},
		{"ESC Percent unhandled", "\x1B%H", 0, 2, 0, unused_handlerID, "Unhandled input:"},
	}

	p := NewParser()

	var place strings.Builder
	p.logT.SetOutput(&place)
	p.logU.SetOutput(&place)

	emu := NewEmulator3(8, 4, 0)
	for _, v := range tc {

		place.Reset()

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

			if hd.id != v.hdIDs {
				t.Errorf("%s: %q expect %s, got %s", v.name, v.seq, strHandlerID[v.hdIDs], strHandlerID[hd.id])
			}

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
			if emu.charsetState.gl != v.wantGL || emu.charsetState.gr != v.wantGR || emu.charsetState.ss != v.wantSS {
				t.Errorf("%s expect GL,GR,SS= %d,%d,%d, got=%d,%d,%d\n", v.name, v.wantGL, v.wantGR, v.wantSS,
					emu.charsetState.gl, emu.charsetState.gr, emu.charsetState.ss)
			}
		} else {
			if v.hdIDs == unused_handlerID {
				if !strings.Contains(place.String(), v.warnStr) {
					t.Errorf("%s:\t %q expect %q, got %s\n", v.name, v.seq, v.warnStr, place.String())
				}
			}
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
		{"LS2", "\x1Bn", []int{esc_ls2}, 2},
		{"LS3", "\x1Bo", []int{esc_ls3}, 3},
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
			hd = p.processInput(ch)
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
		{"LS1R", "\x1B~", []int{esc_ls1r}, 1},
		{"LS2R", "\x1B}", []int{esc_ls2r}, 2},
		{"LS3R", "\x1B|", []int{esc_ls3r}, 3},
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
			hd = p.processInput(ch)
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
		{"SS2", "\x1BN", []int{esc_ss2}, 2}, // G2 single shift
		{"SS3", "\x1BO", []int{esc_ss3}, 3}, // G3 single shift
	}

	p := NewParser()
	emu := NewEmulator3(8, 4, 4)
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
	}{
		{"SO", 0x0E, 1}, // G1 as GL
		{"SI", 0x0F, 0}, // G0 as GL
	}

	p := NewParser()
	emu := NewEmulator3(8, 4, 4)
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
		{"CSI Ps;PsH normal", 10, 10, csi_cup, 23, 13, "\x1B[24;14H", "Cursor positioned to"},
		{"CSI Ps;PsH default", 10, 10, csi_cup, 0, 0, "\x1B[H", "Cursor positioned to"},
		{"CSI Ps;PsH second default", 10, 10, csi_cup, 0, 0, "\x1B[1H", "Cursor positioned to"},
		{"CSI Ps;PsH outrange active area", 10, 10, csi_cup, 39, 79, "\x1B[42;89H", "Cursor positioned to"},
	}
	p := NewParser()

	emu := NewEmulator3(80, 40, 500)
	var place strings.Builder
	emu.logT.SetOutput(&place)

	for _, v := range tc {
		var hd *Handler

		// parse the sequence
		for _, ch := range v.seq {
			hd = p.processInput(ch)
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

func TestHandle_OSC_0_1_2(t *testing.T) {
	tc := []struct {
		name    string
		hdIDs   []int
		icon    bool
		title   bool
		seq     string
		wantStr string
	}{
		{"OSC 0;Pt BEL        ", []int{osc_0_1_2}, true, true, "\x1B]0;ada\x07", "ada"},
		{"OSC 1;Pt 7bit ST    ", []int{osc_0_1_2}, true, false, "\x1B]1;adas\x1B\\", "adas"},
		{"OSC 2;Pt BEL chinese", []int{osc_0_1_2}, false, true, "\x1B]2;[道德经]\x07", "[道德经]"},
		{"OSC 2;Pt BEL unusual", []int{osc_0_1_2}, false, true, "\x1B]2;[neovim]\x1B78\x07", "[neovim]\x1B78"},
	}

	p := NewParser()
	emu := NewEmulator3(8, 4, 4) // this is the pre-condidtion for the test case.

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
			windowTitle := emu.cf.windowTitle
			iconName := emu.cf.iconName

			if hd.id != v.hdIDs[0] {
				t.Errorf("%s seq=%q handler expect %q, got %q\n", v.name, v.seq, strHandlerID[v.hdIDs[0]], strHandlerID[hd.id])
			}
			if v.title && !v.icon && windowTitle != v.wantStr {
				t.Errorf("%s seq=%q only title should be set.\nexpect %q, \ngot %q\n", v.name, v.seq, v.wantStr, windowTitle)
			}
			if !v.title && v.icon && iconName != v.wantStr {
				t.Errorf("%s seq=%q only icon name should be set.\nexpect %q, \ngot %q\n", v.name, v.seq, v.wantStr, iconName)
			}
			if v.title && v.icon && (iconName != v.wantStr || windowTitle != v.wantStr) {
				t.Errorf("%s seq=%q both icon name and window title should be set.\nexpect %q, \ngot window title:%q\ngot iconName:%q\n",
					v.name, v.seq, v.wantStr, windowTitle, iconName)
			}
		} else {
			if p.inputState == InputState_Normal && v.wantStr == "" {
				continue
			}
			t.Errorf("%s got nil return\n", v.name)
		}
	}
}

func TestHandle_OSC_Abort(t *testing.T) {
	tc := []struct {
		name string
		seq  string
		want string
	}{
		{"OSC malform 1         ", "\x1B]ada\x1B\\", "OSC: no ';' exist."},
		{"OSC malform 2         ", "\x1B]7fy;ada\x1B\\", "OSC: illegal Ps parameter."},
		{"OSC Ps overflow: >120 ", "\x1B]121;home\x1B\\", "OSC: malformed command string"},
		{"OSC malform 3         ", "\x1B]7;ada\x1B\\", "unhandled OSC:"},
	}
	p := NewParser()
	var place strings.Builder
	p.logT.SetOutput(&place) // redirect the output to the string builder
	p.logU.SetOutput(&place)

	for _, v := range tc {
		// reset the out put for every test case
		place.Reset()
		var hd *Handler

		// parse the sequence
		for _, ch := range v.seq {
			hd = p.processInput(ch)
		}

		if hd != nil {
			t.Errorf("%s: seq=%q for abort case, hd should be nil. hd=%v\n", v.name, v.seq, hd)
		}

		got := place.String()
		if !strings.Contains(got, v.want) {
			t.Errorf("%s: seq=%q expect %s, got %s\n", v.name, v.seq, v.want, got)
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
	if bellCount == 0 || hds[0].id != c0_bel {
		t.Errorf("BEL expect %d, got %d\n", 1, bellCount)
		t.Errorf("BEL expect %s, got %s\n", strHandlerID[c0_bel], strHandlerID[hds[0].id])
	}
}

func TestHandle_RI_NEL(t *testing.T) {
	tc := []struct {
		name  string
		seq   string
		wantX int
		wantY int
		hdIDs []int
	}{
		{"RI ", "\x1B[11;6H\x1BM", 5, 9, []int{csi_cup, esc_ri}},   // move cursor up to the previouse row, may scroll up
		{"NEL", "\x1B[11;6H\x1BE", 0, 11, []int{csi_cup, esc_nel}}, // move cursor down to next row, may scroll down
	}

	p := NewParser()
	emu := NewEmulator3(40, 20, 0)
	var place strings.Builder
	emu.logI.SetOutput(&place) // redirect the output to the string builder
	emu.logT.SetOutput(&place) // redirect the output to the string builder

	for _, v := range tc {
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
				hd = p.processInput(ch)
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
		{"Primary DA  ", "\x1B[c", fmt.Sprintf("\x1B[?%s", DEVICE_ID), []int{csi_priDA}},
		{"Secondary DA", "\x1B[>c", "\x1B[>64;0;0c", []int{csi_secDA}},
		{"DSR device status report ", "\x1B[5n", "\x1B[0n", []int{csi_dsr}},
		// use DECSET 6 to set  originMode, use CUP to set the active position, then call DSR 6
		{"DSR OriginMode_ScrollingRegion", "\x1B[?6h\x1B[9;9H\x1B[6n", "\x1B[9;9R", []int{csi_privSM, csi_cup, csi_dsr}},
		// use DECRST 6 to set  originMode, use CUP to set the active position, then call DSR 6
		{"DSR OriginMode_Absolute", "\x1B[?6l\x1B[10;10H\x1B[6n", "\x1B[10;10R", []int{csi_privRM, csi_cup, csi_dsr}},
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

func TestHandle_VPA_VPR_CHA_HPA_HPR_CNL_CPL(t *testing.T) {
	tc := []struct {
		name         string
		hdIDs        []int
		wantY, wantX int
		seq          string
	}{
		{"VPA move cursor to row 2 ", []int{csi_cup, csi_vpa}, 2, 9, "\x1B[9;10H\x1B[3d"},
		{"VPA move cursor to row 33", []int{csi_cup, csi_vpa}, 33, 8, "\x1B[9;9H\x1B[34d"},
		{"VPR move cursor to row 12", []int{csi_cup, csi_vpr}, 9, 8, "\x1B[9;9H\x1B[e"},
		{"VPR move cursor to row 39", []int{csi_cup, csi_vpr}, 39, 8, "\x1B[9;9H\x1B[40e"},
		{"CHA move cursor to col 0 ", []int{csi_cup, csi_cha}, 7, 0, "\x1B[8;8H\x1B[G"}, // default Ps is 1
		{"CHA move cursor to col 78", []int{csi_cup, csi_cha}, 6, 78, "\x1B[7;7H\x1B[79G"},
		{"HPA move cursor to col 8 ", []int{csi_cup, csi_hpa}, 5, 8, "\x1B[6;6H\x1B[9`"},
		{"HPA move cursor to col 79", []int{csi_cup, csi_hpa}, 4, 79, "\x1B[5;5H\x1B[99`"},
		{"HPR move cursor to col 5 ", []int{csi_cup, csi_hpr}, 4, 5, "\x1B[5;5H\x1B[a"},
		{"HPR move cursor to col 39", []int{csi_cup, csi_hpr}, 4, 79, "\x1B[5;5H\x1B[79a"},
		{"CNL move cursor to (5,0) ", []int{csi_cup, csi_cnl}, 5, 0, "\x1B[5;5H\x1B[E"},
		{"CNL move cursor to (39,0)", []int{csi_cup, csi_cnl}, 39, 0, "\x1B[5;5H\x1B[79E"},
		{"CPL move cursor to (3,0) ", []int{csi_cup, csi_cpl}, 3, 0, "\x1B[5;5H\x1B[F"},
		{"CPL move cursor to (0,0) ", []int{csi_cup, csi_cpl}, 0, 0, "\x1B[5;5H\x1B[20F"},
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

func TestHandle_SGR_RGBcolor(t *testing.T) {
	tc := []struct {
		name       string
		hdIDs      []int
		fr, fg, fb int
		br, bg, bb int
		attr       charAttribute
		seq        string
	}{
		{"RGB Color 1", []int{csi_sgr}, 33, 47, 12, 123, 24, 34, Bold, "\x1B[0;1;38;2;33;47;12;48;2;123;24;34m"},
		{"RGB Color 2", []int{csi_sgr}, 0, 0, 0, 0, 0, 0, Italic, "\x1B[0;3;38:2:0:0:0;48:2:0:0:0m"},
		{"RGB Color 3", []int{csi_sgr}, 12, 34, 128, 59, 190, 155, Underlined, "\x1B[0;4;38:2:12:34:128;48:2:59:190:155m"},
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
		{"default Color", []int{csi_sgr}, ColorDefault, ColorDefault, charAttribute(38), "\x1B[200m"}, // 38,48 is empty charAttribute
		{"8 Color", []int{csi_sgr}, ColorSilver, ColorBlack, Bold, "\x1B[1;37;40m"},
		{"8 Color 2", []int{csi_sgr}, ColorMaroon, ColorMaroon, Italic, "\x1B[3;31;41m"},
		{"16 Color", []int{csi_sgr}, ColorRed, ColorWhite, Underlined, "\x1B[4;91;107m"},
		{"256 Color 1", []int{csi_sgr}, Color33, Color47, Bold, "\x1B[0;1;38:5:33;48:5:47m"},
		{"256 Color 3", []int{csi_sgr}, Color128, Color155, Underlined, "\x1B[0;4;38:5:128;48:5:155m"},
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
		{"break 38    ", []int{csi_sgr}, "\x1B[38m"},
		{"break 38;   ", []int{csi_sgr}, "\x1B[38;m"},
		{"break 38:5  ", []int{csi_sgr}, "\x1B[38;5m"},
		{"break 38:2-1", []int{csi_sgr}, "\x1B[38:2:23m"},
		{"break 38:2-2", []int{csi_sgr}, "\x1B[38:2:23:24m"},
		{"break 38:7  ", []int{csi_sgr}, "\x1B[38;7m"},
		{"break 48    ", []int{csi_sgr}, "\x1B[48m"},
		{"break 48;   ", []int{csi_sgr}, "\x1B[48;m"},
		{"break 48:5  ", []int{csi_sgr}, "\x1B[48;5m"},
		{"break 48:2-1", []int{csi_sgr}, "\x1B[48:2:23m"},
		{"break 48:2-2", []int{csi_sgr}, "\x1B[48:2:23:22m"},
		{"break 48:7  ", []int{csi_sgr}, "\x1B[48;7m"},
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

func TestHandle_DECSTBM(t *testing.T) {
	tc := []struct {
		name        string
		seq         string
		hdIDs       []int
		top, bottom int
		posX, posY  int
		logMessage  string
	}{
		{
			"DECSTBM ", "\x1B[24;14H\x1B[2;30r", // move the cursor to 23,13 first
			[]int{csi_cup, csi_decstbm}, // then set new top/bottom margin
			2 - 1, 30, 0, 0, "",
		},
		{
			"DECSTBM ", "\x1B[2;6H\x1B[3;32r\x1B[32;30r", // CUP, then a successful STBM follow an ignored STBM.
			[]int{csi_cup, csi_decstbm, csi_decstbm},
			3 - 1, 32, 0, 0, "Illegal arguments to SetTopBottomMargins:",
		},
		{"DECSTBM no parameters", "\x1B[2;6H\x1B[r", []int{csi_cup, csi_decstbm}, 0, 40, 0, 0, ""},
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

func TestHandle_OSC_52(t *testing.T) {
	tc := []struct {
		name       string
		hdIDs      []int
		wantPc     string
		wantPd     string
		wantString string
		noReply    bool
		seq        string
	}{
		{
			"new selection in c",
			[]int{osc_52},
			"c", "YXByaWxzaAo=",
			"\x1B]52;c;YXByaWxzaAo=\x1B\\", true,
			"\x1B]52;c;YXByaWxzaAo=\x1B\\",
		},
		{
			"clear selection in cs",
			[]int{osc_52, osc_52},
			"cs", "",
			"\x1B]52;cs;x\x1B\\", true, // echo "aprilsh" | base64
			"\x1B]52;cs;YXByaWxzaAo=\x1B\\\x1B]52;cs;x\x1B\\",
		},
		{
			"empty selection",
			[]int{osc_52},
			"s0", "5Zub5aeR5aiY5bGxCg==", // echo "四姑娘山" | base64
			"\x1B]52;s0;5Zub5aeR5aiY5bGxCg==\x1B\\", true,
			"\x1B]52;;5Zub5aeR5aiY5bGxCg==\x1B\\",
		},
		{
			"question selection",
			[]int{osc_52, osc_52},
			"", "", // don't care these values
			"\x1B]52;c;5Zub5aeR5aiY5bGxCg==\x1B\\", false,
			"\x1B]52;c0;5Zub5aeR5aiY5bGxCg==\x1B\\\x1B]52;c0;?\x1B\\",
		},
	}
	p := NewParser()
	emu := NewEmulator3(8, 4, 0)
	for _, v := range tc {
		emu.cf.selectionData = ""
		emu.terminalToHost.Reset()

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
				if hd.id != v.hdIDs[j] { // validate the control sequences id
					t.Errorf("%s: seq=%q expect %s, got %s\n", v.name, v.seq, strHandlerID[v.hdIDs[j]], strHandlerID[hd.id])
				}
			}

			if v.noReply {
				if v.wantString != emu.cf.selectionData {
					t.Errorf("%s: seq=%q expect %q, got %q\n", v.name, v.seq, v.wantString, emu.cf.selectionData)
				}
				for _, ch := range v.wantPc {
					if data, ok := emu.selectionData[ch]; ok && data == v.wantPd {
						continue
					} else {
						t.Errorf("%s: seq=%q, expect[%c]%q, got [%c]%q\n", v.name, v.seq, ch, v.wantPc, ch, emu.selectionData[ch])
					}
				}
			} else {
				got := emu.terminalToHost.String()
				if got != v.wantString {
					t.Errorf("%s: seq=%q, expect %q, got %q\n", v.name, v.seq, v.wantString, got)
				}
			}
		})
	}
}

func TestHandle_OSC_52_abort(t *testing.T) {
	tc := []struct {
		name    string
		hdIDs   []int
		wantStr string
		seq     string
	}{
		{"malform OSC 52 ", []int{osc_52}, "OSC 52: can't find Pc parameter.", "\x1B]52;23\x1B\\"},
		{"Pc not in range", []int{osc_52}, "invalid Pc parameters.", "\x1B]52;se;\x1B\\"},
	}
	p := NewParser()
	emu := NewEmulator3(8, 4, 0)
	var place strings.Builder
	emu.logW.SetOutput(&place)

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
			for j, hd := range hds {
				hd.handle(emu)
				if hd.id != v.hdIDs[j] { // validate the control sequences id
					t.Errorf("%s: seq=%q expect %s, got %s\n", v.name, v.seq, strHandlerID[v.hdIDs[j]], strHandlerID[hd.id])
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
		hdIDs      []int
		wantString string
		warn       bool
		seq        string
	}{
		{
			"query one color number",
			[]int{osc_4},
			"\x1B]4;1;rgb:8080/0000/0000\x1B\\", false,
			"\x1B]4;1;?\x1B\\",
		},
		{
			"query two color number",
			[]int{osc_4},
			"\x1B]4;250;rgb:bcbc/bcbc/bcbc\x1B\\\x1B]4;1;rgb:8080/0000/0000\x1B\\", false,
			"\x1B]4;250;?;1;?\x1B\\",
		},
		{
			"query 8 color number",
			[]int{osc_4},
			"\x1B]4;0;rgb:0000/0000/0000\x1B\\\x1B]4;1;rgb:8080/0000/0000\x1B\\\x1B]4;2;rgb:0000/8080/0000\x1B\\\x1B]4;3;rgb:8080/8080/0000\x1B\\\x1B]4;4;rgb:0000/0000/8080\x1B\\\x1B]4;5;rgb:8080/0000/8080\x1B\\\x1B]4;6;rgb:0000/8080/8080\x1B\\\x1B]4;7;rgb:c0c0/c0c0/c0c0\x1B\\", false,
			"\x1B]4;0;?;1;?;2;?;3;?;4;?;5;?;6;?;7;?\x1B\\",
		},
		{
			"missing ';' abort",
			[]int{osc_4},
			"OSC 4: malformed argument, missing ';'.", true,
			"\x1B]4;1?\x1B\\",
		},
		{
			"Ps malform abort",
			[]int{osc_4},
			"OSC 4: can't parse c parameter.", true,
			"\x1B]4;m;?\x1B\\",
		},
	}
	p := NewParser()
	emu := NewEmulator3(8, 4, 0)
	var place strings.Builder
	emu.logW.SetOutput(&place)

	for _, v := range tc {
		place.Reset()
		emu.terminalToHost.Reset()

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
				if hd.id != v.hdIDs[j] { // validate the control sequences id
					t.Errorf("%s: seq=%q expect %s, got %s\n", v.name, v.seq, strHandlerID[v.hdIDs[j]], strHandlerID[hd.id])
				}
			}

			if v.warn {
				if !strings.Contains(place.String(), v.wantString) {
					t.Errorf("%s: seq=%q expect %q, got %q\n", v.name, v.seq, v.wantString, place.String())
				}
			} else {
				got := emu.terminalToHost.String()
				if got != v.wantString {
					t.Errorf("%s: seq=%q, \nexpect\t %q, \ngot\t\t %q\n", v.name, v.seq, v.wantString, got)
				}
			}
		})
	}
}

// echo -ne '\e]10;?\e\\'; cat
// echo -ne '\e]4;0;?\e\\'; cat
func TestHandle_OSC_10x(t *testing.T) {
	invalidColor := NewHexColor(0xF8F8F8)
	tc := []struct {
		name        string
		fgColor     Color
		bgColor     Color
		cursorColor Color
		hdIDs       []int
		wantString  string
		warn        bool
		seq         string
	}{
		{
			"query 6 color",
			ColorWhite, ColorGreen, ColorOlive,
			[]int{osc_10_11_12_17_19},
			"\x1B]10;rgb:ffff/ffff/ffff\x1B\\\x1B]11;rgb:0000/8080/0000\x1B\\\x1B]17;rgb:0000/8080/0000\x1B\\\x1B]19;rgb:ffff/ffff/ffff\x1B\\\x1B]12;rgb:8080/8080/0000\x1B\\", false,
			"\x1B]10;?;11;?;17;?;19;?;12;?\x1B\\",
		},
		{
			"parse color parameter error",
			invalidColor, invalidColor, invalidColor,
			[]int{osc_10_11_12_17_19},
			"OSC 10x: can't parse color index.", true,
			"\x1B]10;?;m;?\x1B\\",
		},
		{
			"malform parameter",
			invalidColor, invalidColor, invalidColor,
			[]int{osc_10_11_12_17_19},
			"OSC 10x: malformed argument, missing ';'.", true,
			"\x1B]10;?;\x1B\\",
		},
		{
			"VT100 text foreground color: regular color",
			ColorWhite, invalidColor, invalidColor,
			[]int{osc_10_11_12_17_19},
			"\x1B]10;rgb:ffff/ffff/ffff\x1B\\", false,
			"\x1B]10;?\x1B\\",
		},
		{
			"VT100 text background color: default color",
			invalidColor, ColorDefault, invalidColor,
			[]int{osc_10_11_12_17_19},
			"\x1B]11;rgb:0000/0000/0000\x1B\\", false,
			"\x1B]11;?\x1B\\",
		},
		{
			"text cursor color: regular color",
			invalidColor, invalidColor, ColorGreen,
			[]int{osc_10_11_12_17_19},
			"\x1B]12;rgb:0000/8080/0000\x1B\\", false,
			"\x1B]12;?\x1B\\",
		},
		{
			"text cursor color: default color",
			invalidColor, invalidColor, ColorDefault,
			[]int{osc_10_11_12_17_19},
			"\x1B]12;rgb:0000/0000/0000\x1B\\", false,
			"\x1B]12;?\x1B\\",
		},
	}
	p := NewParser()
	emu := NewEmulator3(80, 40, 5)
	var place strings.Builder
	emu.logW.SetOutput(&place)

	for _, v := range tc {
		place.Reset()
		emu.terminalToHost.Reset()

		t.Run(v.name, func(t *testing.T) {
			// process control sequence
			hds := make([]*Handler, 0, 16)
			hds = p.processStream(v.seq, hds)

			if len(hds) == 0 {
				t.Errorf("%s got zero handlers.", v.name)
			}

			// set pre-condition
			if v.fgColor != invalidColor {
				emu.attrs.renditions.fgColor = v.fgColor
			}
			if v.bgColor != invalidColor {
				emu.attrs.renditions.bgColor = v.bgColor
			}
			if v.cursorColor != invalidColor {
				emu.cf.DS.cursorColor = v.cursorColor
			}

			// execute the control sequence
			for j, hd := range hds {
				hd.handle(emu)
				if hd.id != v.hdIDs[j] { // validate the control sequences id
					t.Errorf("%s: seq=%q expect %s, got %s\n", v.name, v.seq, strHandlerID[v.hdIDs[j]], strHandlerID[hd.id])
				}
			}

			if v.warn {
				if !strings.Contains(place.String(), v.wantString) {
					t.Errorf("%s: seq=%q expect %q, got %q\n", v.name, v.seq, v.wantString, place.String())
				}
			} else {
				got := emu.terminalToHost.String()
				if got != v.wantString {
					t.Errorf("%s: seq=%q, \nexpect\t %q, \ngot\t\t %q\n", v.name, v.seq, v.wantString, got)
				}
			}
		})
	}
}

func TestHandle_DCS(t *testing.T) {
	tc := []struct {
		name    string
		hdIDs   []int
		wantMsg string
		warn    bool
		seq     string
	}{
		{"DECRQSS normal", []int{dcs_decrqss}, "\x1BP1$r" + DEVICE_ID + "\x1B\\", false, "\x1BP$q\"p\x1B\\"},
		{"decrqss others", []int{dcs_decrqss}, "\x1BP0$rother\x1B\\", false, "\x1BP$qother\x1B\\"},
		{"DCS unimplement", []int{dcs_decrqss}, "DCS:", true, "\x1BPunimplement\x1B78\x1B\\"},
	}
	p := NewParser()
	// p.logU = log.New(&place, "(Uimplemented): ", log.Ldate|log.Ltime|log.Lshortfile)
	emu := NewEmulator3(8, 4, 0)
	var place strings.Builder
	p.logU.SetOutput(&place) // redirect the output to the string builder

	for _, v := range tc {
		place.Reset()
		emu.terminalToHost.Reset()

		t.Run(v.name, func(t *testing.T) {
			// process control sequence
			hds := make([]*Handler, 0, 16)
			hds = p.processStream(v.seq, hds)

			if !v.warn && len(hds) == 0 {
				t.Errorf("%s got zero handlers.", v.name)
			}

			// execute the control sequence
			for j, hd := range hds {
				hd.handle(emu)
				if hd.id != v.hdIDs[j] { // validate the control sequences name
					t.Errorf("%s:\t %q expect %s, got %s\n", v.name, v.seq, strHandlerID[v.hdIDs[j]], strHandlerID[hd.id])
				}
			}

			if v.warn {
				if !strings.Contains(place.String(), v.wantMsg) {
					t.Errorf("%s: seq=%q expect %q, got %q\n", v.name, v.seq, v.wantMsg, place.String())
				}
			} else {
				got := emu.terminalToHost.String()
				if got != v.wantMsg {
					t.Errorf("%s: seq=%q, \nexpect\t %q, \ngot\t\t %q\n", v.name, v.seq, v.wantMsg, got)
				}
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

func TestHandle_DECSCL(t *testing.T) {
	tc := []struct {
		name     string
		seq      string
		hdIDs    []int
		cmpLevel CompatibilityLevel
		warnStr  string
	}{
		{"CompatLevel VT100", "\x1B[61\"p", []int{csi_decscl}, CompatLevel_VT100, ""},
		{"CompatLevel VT400", "\x1B[62\"p", []int{csi_decscl}, CompatLevel_VT400, ""},
		{"CompatLevel VT400", "\x1B[63\"p", []int{csi_decscl}, CompatLevel_VT400, ""},
		{"CompatLevel VT400", "\x1B[64\"p", []int{csi_decscl}, CompatLevel_VT400, ""},
		{"CompatLevel VT400", "\x1B[65\"p", []int{csi_decscl}, CompatLevel_VT400, ""},
		{"CompatLevel VT400 DECANM", "\x1B<", []int{esc_decanm}, CompatLevel_VT400, ""},
		{"CompatLevel others", "\x1B[66\"p", []int{csi_decscl}, CompatLevel_Unused, "compatibility mode:"},
		{"CompatLevel 8-bit control", "\x1B[65;0\"p", []int{csi_decscl}, CompatLevel_Unused, "DECSCL: 8-bit controls"},
		{"CompatLevel 8-bit control", "\x1B[61;2\"p", []int{csi_decscl}, CompatLevel_Unused, "DECSCL: 8-bit controls"},
		{"CompatLevel 7-bit control", "\x1B[65;1\"p", []int{csi_decscl}, CompatLevel_Unused, "DECSCL: 7-bit controls"},
		{"CompatLevel outof range  ", "\x1B[65;3\"p", []int{csi_decscl}, CompatLevel_Unused, "DECSCL: C1 control transmission mode:"},
		{"CompatLevel unhandled", "\x1B[65;3\"q", []int{csi_decscl}, CompatLevel_Unused, "Unhandled input:"},
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

		if len(hds) != 1 {
			if v.warnStr == "" {
				t.Errorf("%s expect %d handlers.", v.name, len(hds))
			} else if v.warnStr != "" && !strings.Contains(place.String(), v.warnStr) {
				t.Errorf("%s:\t %q expect %q, got %s\n", v.name, v.seq, v.warnStr, place.String())
			} else {
				continue
			}
		}

		// handle the control sequence
		for j, hd := range hds {
			hd.handle(emu)
			if hd.id != v.hdIDs[j] { // validate the control sequences name
				t.Errorf("%s:\t %q expect %s, got %s\n", v.name, v.seq, strHandlerID[v.hdIDs[j]], strHandlerID[hd.id])
			}
			// t.Logf("%s seq=%q\n", v.name, hd.sequence)
		}

		switch i {
		case 0, 1, 2, 3, 4, 5:
			got := emu.compatLevel
			if got != v.cmpLevel {
				t.Errorf("%s:\t %q, expect %d, got %d\n", v.name, v.seq, v.cmpLevel, got)
			}
		default:
			if !strings.Contains(place.String(), v.warnStr) {
				t.Errorf("%s:\t %q expect %q, got %s\n", v.name, v.seq, v.warnStr, place.String())
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
		{"ecma48 SL 2 cols", []int{esc_decaln, csi_ecma48_SL}, 0, 0, 3, 7, "\x1B#8\x1B[2 @", []int{6, 7}},
		{"ecma48 SL 1 col ", []int{esc_decaln, csi_ecma48_SL}, 0, 0, 3, 7, "\x1B#8\x1B[ @", []int{7}},
		{"ecma48 SL all cols", []int{esc_decaln, csi_ecma48_SL}, 0, 0, 3, 7, "\x1B#8\x1B[9 @", []int{0, 1, 2, 3, 4, 5, 6, 7}},
		{"ecma48 SR 4 cols", []int{esc_decaln, csi_ecma48_SR}, 0, 0, 3, 7, "\x1B#8\x1B[4 A", []int{0, 1, 2, 3}},
		{"ecma48 SR 1 cols", []int{esc_decaln, csi_ecma48_SR}, 0, 0, 3, 7, "\x1B#8\x1B[ A", []int{0}},
		{"ecma48 SR all cols", []int{esc_decaln, csi_ecma48_SR}, 0, 0, 3, 7, "\x1B#8\x1B[9 A", []int{0, 1, 2, 3, 4, 5, 6, 7}},
		{"DECFI 1 cols", []int{esc_decaln, csi_cup, esc_fi}, 0, 0, 3, 7, "\x1B#8\x1B[4;8H\x1B9", []int{7}},
		{"DECBI 1 cols", []int{esc_decaln, csi_cup, esc_bi}, 0, 0, 3, 7, "\x1B#8\x1B[4;1H\x1B6", []int{0}},
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
		{"XTMODEKEYS 0:x    ", []int{csi_xtmodkeys}, "\x1B[>0;1m", 3, "XTMODKEYS: modifyKeyboard ="},
		{"XTMODEKEYS 0:break", []int{csi_xtmodkeys}, "\x1B[>0;0m", 3, ""},
		{"XTMODEKEYS 1:x    ", []int{csi_xtmodkeys}, "\x1B[>1;1m", 3, "XTMODKEYS: modifyCursorKeys ="},
		{"XTMODEKEYS 1:break", []int{csi_xtmodkeys}, "\x1B[>1;2m", 3, ""},
		{"XTMODEKEYS 2:x    ", []int{csi_xtmodkeys}, "\x1B[>2;1m", 3, "XTMODKEYS: modifyFunctionKeys ="},
		{"XTMODEKEYS 2:break", []int{csi_xtmodkeys}, "\x1B[>2;2m", 3, ""},
		{"XTMODEKEYS 4:x    ", []int{csi_xtmodkeys}, "\x1B[>4;2m", 2, "XTMODKEYS: modifyOtherKeys set to"},
		{"XTMODEKEYS 4:break", []int{csi_xtmodkeys}, "\x1B[>4;3m", 3, "XTMODKEYS: illegal argument for modifyOtherKeys:"},
		{"XTMODEKEYS 1 parameter", []int{csi_xtmodkeys}, "\x1B[>4m", 0, "XTMODKEYS: modifyOtherKeys set to"},
		{"XTMODEKEYS 0 parameter", []int{csi_xtmodkeys}, "\x1B[>m", 3, ""}, // no parameter
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
