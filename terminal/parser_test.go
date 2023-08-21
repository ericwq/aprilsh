// Copyright 2022 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package terminal

import (
	"fmt"
	"io"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/ericwq/aprilsh/util"
	"github.com/rivo/uniseg"
	"golang.org/x/exp/slog"
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

// func printRows(fb *Framebuffer) string {
// 	var output strings.Builder
// 	for _, row := range fb.rows {
// 		output.WriteString(row.String() + "\n")
// 	}
//
// 	return output.String()
// }

// check if the row in the specified row list. if so, return true
// otherwise return false. for empty row list, return false.
func inScope(rows []int, row int) bool {
	if len(rows) == 0 {
		return false
	}
	for _, v := range rows {
		if v == row {
			return true
		}
	}
	return false
}

// fill the specified rows on the screen with rotating A~Z.
// if row list is empty, fill the whole screen.
func fillCells(fb *Framebuffer, rows ...int) {
	A := 0x41

	for r := 0; r < fb.nRows; r++ {
		if len(rows) == 0 || inScope(rows, r) {
			start := fb.nCols * r // fb.getIdx(r, 0)
			end := start + fb.nCols
			for k := start; k < end; k++ {
				ch := rune(A + (k % 26))
				fb.cells[k].contents = string(ch)
			}
		}
	}
}

// print the screen with specified rows. if the row list is empty, print the whole screen.
func printCells(fb *Framebuffer, rows ...int) string {
	var output strings.Builder

	fmt.Fprintf(&output, "[RULE]")
	for r := 0; r < fb.nCols; r++ {
		fmt.Fprintf(&output, "%d", r%10)
	}
	fmt.Fprintf(&output, "\n")

	for r := 0; r < fb.nRows; r++ {
		if len(rows) == 0 || inScope(rows, r) {
			start := fb.nCols * r // fb.getIdx(r, 0)
			end := start + fb.nCols
			printRowAt(r, start, end, fb, &output)
		}
	}
	// print the saveLines if it has
	if fb.saveLines > 0 {
		for r := fb.nRows; r < fb.nRows+fb.saveLines; r++ {
			if len(rows) == 0 || inScope(rows, r) {
				start := r*fb.nCols + 0
				end := start + fb.nCols
				printRowAt(r, start, end, fb, &output)
			}
		}
	}
	return output.String()
}

func printRowAt(r int, start int, end int, fb *Framebuffer, output *strings.Builder) {
	if fb.scrollHead == r {
		fmt.Fprintf(output, "[%3d]-", r)
	} else {
		fmt.Fprintf(output, "[%3d] ", r)
	}
	for k := start; k < end; k++ {
		switch fb.cells[k].contents {
		case " ":
			if !fb.cells[k].dwidthCont {
				output.WriteString(".")
			}
		case "":
			if !fb.cells[k].dwidthCont {
				output.WriteString("*")
			}
		default:
			output.WriteString(fb.cells[k].contents)
		}
	}
	output.WriteString("\n")
}

// check the specified rows is empty, if so return true, otherwise return false.
func isEmptyRows(fb *Framebuffer, rows ...int) bool {
	if len(rows) == 0 {
		return false
	}

	for _, r := range rows {
		for c := 0; c < fb.nCols; c++ {
			idx := fb.getIdx(r, c)
			if fb.cells[idx].contents != " " {
				return false
			}
		}
	}
	return true
}

// check the specified cols is empty, if so return true, otherwise return false.
func isEmptyCols(fb *Framebuffer, cols ...int) bool {
	if len(cols) == 0 {
		return false
	}
	for _, c := range cols {
		for r := 0; r < fb.nRows; r++ {
			idx := fb.getIdx(r, c)
			if fb.cells[idx].contents != " " {
				// fmt.Printf("isEmptyCols() row=%d col=%d is %s\n", r, c, fb.cells[idx].contents)
				return false
			}
		}
	}
	return true
}

// check the specified cells is empty, if so return true, otherwise return false.
// the cells start at (pY,pX), counting sucessive number .
func isEmptyCells(fb *Framebuffer, pY, pX, count int) bool {
	if count == 0 {
		return true
	}
	for i := 0; i < count; i++ {
		idx := fb.getIdx(pY, pX+i)
		if fb.cells[idx].contents != " " {
			return false
		}
	}
	return true
}

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
			1,
		},
		{
			"emoji 2", "🗻",
			2,
		},
		{
			"emoji 3", "🏖",
			1,
		},
		{
			"flags", "🇳🇱🇧🇷i",
			5,
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
			width += uniseg.StringWidth(string(rs))
			// fmt.Printf("%c %q %U w=%d\n", rs, rs, rs, uniseg.StringWidth(string(rs)))
		}

		if v.width != width {
			t.Logf("%s :\t %q %U\n", v.name, v.raw, rs)
			t.Errorf("%s:\t %q  expect width %d, got %d\n", v.name, v.raw, v.width, width)
		}
	}
}

func TestUnisegStringWidth(t *testing.T) {
	raw := "\x1B[2;30HChin\u0308\u0308a 🏖 i国旗🇳🇱Fun 🌈with Flag🇧🇷.s"

	gs := uniseg.NewGraphemes(raw)
	for gs.Next() {
		rs := gs.Runes()
		w1 := uniseg.StringWidth(string(rs))
		w2 := gs.Width()
		if w2 != w1 {
			t.Errorf("%q %U width expect %d, got %d\n", rs, rs, w1, w2)
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
	// p.logE.SetOutput(ioutil.Discard)
	// p.logU.SetOutput(ioutil.Discard)
	// p.logT.SetOutput(ioutil.Discard)
	defer util.Log.Restore()
	util.Log.SetOutput(io.Discard)

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
		t.Errorf("processInput expect empty, got %s\n", strHandlerID[hd.id])
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
		posX      int    // expect starting cols
		graphemes string // data string without control sequences
	}{
		// use CUP to set the active cursor position first
		{
			"UTF-8 plain english",
			"\x1B[1;14Hlong long ago",
			[]int{CSI_CUP, Graphemes},
			14, 0, 13,
			"long long ago",
		},
		{
			"UTF-8 chinese, combining character and flags",
			"\x1B[2;30HChin\u0308\u0308a 🏖 i国旗🇳🇱Fun 🌈with Flag🇧🇷.s",
			[]int{CSI_CUP, Graphemes},
			30, 1, 29,
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
			24, 2, 23,
			"中国¥ABâãéShanghaiCDàá",
		},
		{
			"VT edge", "\x1B[4;10H\x1B%@Beijing\x1B%G",
			[]int{CSI_CUP, ESC_DOCS_ISO8859_1, Graphemes, Graphemes, Graphemes, Graphemes, Graphemes, Graphemes, Graphemes, ESC_DOCS_UTF8},
			10, 3, 9,
			"Beijing",
		},
	}

	p := NewParser()
	var place strings.Builder
	// p.logE.SetOutput(&place)
	// p.logU.SetOutput(&place)
	// p.logT.SetOutput(&place)
	defer util.Log.Restore()
	util.Log.SetOutput(&place)

	emu := NewEmulator3(80, 40, 40)
	// emu.logT.SetOutput(&place)
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
			j := 0        // the nth grapheme
			pos := v.posX // the position counting width
			for graphemes.Next() {
				// the expected content
				chs := graphemes.Runes()

				// get the cell from framebuffer
				rows := v.posY
				cols := pos
				cell := emu.cf.getCell(rows, cols)

				w := uniseg.StringWidth(string(chs))
				// fmt.Printf("%c %q %x x=%d\n", chs, chs, chs, w)
				if cell.contents != string(chs) {
					t.Errorf("%s:\t [row,cols]:[%2d,%2d] expect %q, got %q\n", v.name, rows, cols, string(chs), cell.contents)
				}
				j += 1
				pos += w
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
	// emu.logT.SetOutput(&place)
	defer util.Log.Restore()
	util.Log.SetOutput(&place)

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
	// p.logE.SetOutput(&place)
	// p.logU.SetOutput(&place)
	// p.logT.SetOutput(&place)
	defer util.Log.Restore()
	util.Log.SetOutput(&place)

	emu := NewEmulator3(80, 40, 40)
	// emu.logT.SetOutput(&place)
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
				if hd.GetId() != hdID { // validate the control sequences id
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
	// emu.logU.SetOutput(&place)
	defer util.Log.Restore()
	util.Log.SetOutput(&place)

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
		{"sgr0        ", []int{CSI_SGR}, "\x1B[m"},
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

func TestSequenceWithColor(t *testing.T) {
	tc := []struct {
		label string
		seq   string
		rends []Renditions
	}{
		{"sequence with text and changed color", "\x1b[1;34mdevelop\x1b[m  \x1b[1;34mproj\x1b[m",
			[]Renditions{
				{fgColor: ColorNavy, bold: true}, {fgColor: ColorNavy, bold: true}, {fgColor: ColorNavy, bold: true},
				{fgColor: ColorNavy, bold: true}, {fgColor: ColorNavy, bold: true}, {fgColor: ColorNavy, bold: true},
				{fgColor: ColorNavy, bold: true}, {}, {}, {fgColor: ColorNavy, bold: true},
				{fgColor: ColorNavy, bold: true}, {fgColor: ColorNavy, bold: true}, {fgColor: ColorNavy, bold: true}}},
	}
	p := NewParser()
	emu := NewEmulator3(80, 40, 40)

	for _, v := range tc {
		t.Run(v.label, func(t *testing.T) {
			// process control sequence
			hds := make([]*Handler, 0, 16)
			hds = p.processStream(v.seq, hds)

			if len(hds) != 17 {
				t.Errorf("%s got zero handlers.", v.label)
			}

			// handle the control sequence
			for _, hd := range hds {
				hd.handle(emu)
			}

			// the break case should not affect the renditions, it will keep the same.
			rows := 0
			for pos := range v.rends {
				cols := pos
				cell := emu.cf.getCell(rows, cols)
				got := cell.GetRenditions()
				if got != v.rends[pos] {
					t.Errorf("%s: pos %d expect renditions:\n%v %s, got \n%v\n", v.label, pos, v.rends[pos], cell, got)
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
	// p.logT.SetOutput(&place)
	defer util.Log.Restore()
	util.Log.SetOutput(&place)

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
		{"ESC Percent unhandled", "\x1B%H", 0, 2, 0, nil, "Unhandled input"},
		{"VT52 ESC G", "\x1B[?2l\x1BG", 0, 2, 0, []int{CSI_privRM, ESC_DOCS_UTF8}, ""},
	}

	p := NewParser()

	var place strings.Builder
	// p.logT.SetOutput(&place)
	// p.logU.SetOutput(&place)
	defer util.Log.Restore()
	util.Log.SetOutput(&place)
	util.Log.SetLevel(slog.LevelDebug)

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
				t.Errorf("%s: seq=%q expect %s, got %s\n",
					v.name, v.seq, strHandlerID[v.hdIDs[j]], strHandlerID[hd.id])
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
	// emu.logT.SetOutput(&place)
	defer util.Log.Restore()
	util.Log.SetOutput(&place)
	util.Log.SetLevel(slog.LevelDebug)

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
	// emu.logT.SetOutput(&place)
	defer util.Log.Restore()
	util.Log.SetOutput(&place)
	util.Log.SetLevel(slog.LevelDebug)

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
	// p.logT.SetOutput(&place)
	defer util.Log.Restore()
	util.Log.SetOutput(&place)
	util.Log.SetLevel(slog.LevelDebug)

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
		name   string
		startX int
		startY int
		hdIDs  int
		wantY  int
		wantX  int
		seq    string
	}{
		{"CSI Ps;PsH normal", 10, 10, CSI_CUP, 23, 13, "\x1B[24;14H"},
		{"CSI Ps;PsH default", 10, 10, CSI_CUP, 0, 0, "\x1B[H"},
		{"CSI Ps;PsH second default", 10, 10, CSI_CUP, 0, 0, "\x1B[1H"},
		{"CSI Ps;PsH outrange active area", 10, 10, CSI_CUP, 39, 79, "\x1B[42;89H"},
	}
	p := NewParser()

	emu := NewEmulator3(80, 40, 500)

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
	}
}

func TestHandle_BEL(t *testing.T) {
	seq := "\x07"
	emu := NewEmulator3(8, 4, 4)

	hds := emu.HandleStream(seq)

	if len(hds) == 0 {
		t.Errorf("BEL got nil for seq=%q\n", seq)
	}

	bellCount := emu.cf.getBellCount()
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
	// emu.logI.SetOutput(&place) // redirect the output to the string builder
	// emu.logT.SetOutput(&place) // redirect the output to the string builder
	defer util.Log.Restore()
	util.Log.SetOutput(&place)
	util.Log.SetLevel(slog.LevelDebug)

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

	// p.logE.SetOutput(&place)
	// p.logU.SetOutput(&place)
	defer util.Log.Restore()
	util.Log.SetOutput(&place)
	util.Log.SetLevel(slog.LevelDebug)

	p.logTrace = true // open the trace
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
				t.Errorf("%s should get nil handler, got %s, history=%q\n", v.name, strHandlerID[hd.id], p.historyString())
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
	// emu.logI.SetOutput(&place) // redirect the output to the string builder
	// emu.logT.SetOutput(&place) // redirect the output to the string builder
	defer util.Log.Restore()
	util.Log.SetOutput(&place)

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
	// emu.logT.SetOutput(&place) // swallow the output
	defer util.Log.Restore()
	util.Log.SetOutput(&place)

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
	// p.logU.SetOutput(&place) // redirect the output to the string builder
	defer util.Log.Restore()
	util.Log.SetOutput(&place)

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
			3 - 1, 32, 0, 0, "Illegal arguments to SetTopBottomMargins",
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
	// emu.logT.SetOutput(&place)
	defer util.Log.Restore()
	util.Log.SetOutput(&place)
	util.Log.SetLevel(slog.LevelDebug)

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
		{"CompatLevel others        ", "\x1B[66\"p", []int{CSI_DECSCL}, CompatLevel_Unused, "compatibility mode"},
		{"CompatLevel 8-bit control ", "\x1B[65;0\"p", []int{CSI_DECSCL}, CompatLevel_Unused, "DECSCL: 8-bit controls"},
		{"CompatLevel 8-bit control ", "\x1B[61;2\"p", []int{CSI_DECSCL}, CompatLevel_Unused, "DECSCL: 8-bit controls"},
		{"CompatLevel 7-bit control ", "\x1B[65;1\"p", []int{CSI_DECSCL}, CompatLevel_Unused, "DECSCL: 7-bit controls"},
		{"CompatLevel outof range   ", "\x1B[65;3\"p", []int{CSI_DECSCL}, CompatLevel_Unused, "DECSCL: C1 control transmission mode"},
		{"CompatLevel unhandled", "\x1B[65;3\"q", []int{CSI_DECSCL}, CompatLevel_Unused, "Unhandled input"},
	}

	emu := NewEmulator3(8, 4, 0)
	p := NewParser()
	var place strings.Builder
	// redirect the output to the string builder
	// emu.logU.SetOutput(&place)
	// emu.logT.SetOutput(&place)
	//
	// p.logU.SetOutput(&place)
	defer util.Log.Restore()
	util.Log.SetOutput(&place)
	util.Log.SetLevel(slog.LevelDebug)

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
	// emu.logI.SetOutput(&place)
	// emu.logT.SetOutput(&place)
	defer util.Log.Restore()
	util.Log.SetOutput(&place)

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
		{"XTMODEKEYS 0:x    ", []int{CSI_XTMODKEYS}, "\x1B[>0;1m", 3, "XTMODKEYS: modifyKeyboard"},
		{"XTMODEKEYS 0:break", []int{CSI_XTMODKEYS}, "\x1B[>0;0m", 3, ""},
		{"XTMODEKEYS 1:x    ", []int{CSI_XTMODKEYS}, "\x1B[>1;1m", 3, "XTMODKEYS: modifyCursorKeys"},
		{"XTMODEKEYS 1:break", []int{CSI_XTMODKEYS}, "\x1B[>1;2m", 3, ""},
		{"XTMODEKEYS 2:x    ", []int{CSI_XTMODKEYS}, "\x1B[>2;1m", 3, "XTMODKEYS: modifyFunctionKeys"},
		{"XTMODEKEYS 2:break", []int{CSI_XTMODKEYS}, "\x1B[>2;2m", 3, ""},
		{"XTMODEKEYS 4:x    ", []int{CSI_XTMODKEYS}, "\x1B[>4;2m", 2, "XTMODKEYS: modifyOtherKeys set to"},
		{"XTMODEKEYS 4:break", []int{CSI_XTMODKEYS}, "\x1B[>4;3m", 3, "XTMODKEYS: illegal argument for modifyOtherKeys"},
		{"XTMODEKEYS 1 parameter", []int{CSI_XTMODKEYS}, "\x1B[>4m", 0, "XTMODKEYS: modifyOtherKeys set to"},
		{"XTMODEKEYS 0 parameter", []int{CSI_XTMODKEYS}, "\x1B[>m", 3, ""}, // no parameter
	}

	p := NewParser()
	emu := NewEmulator3(8, 4, 4)
	var place strings.Builder // all the message is output to herer
	// emu.logU.SetOutput(&place)
	// emu.logT.SetOutput(&place)
	// emu.logI.SetOutput(&place)
	defer util.Log.Restore()
	util.Log.SetOutput(&place)
	util.Log.SetLevel(slog.LevelDebug)

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

func damageArea(cf *Framebuffer, y1, x1, y2, x2 int) (start, end int) {
	start = cf.getIdx(y1, x1)
	end = cf.getIdx(y2, x2)
	return
}

// if the y,x is in the range, return true, otherwise return false
// func inRange(startY, startX, endY, endX, y, x, width int) bool {
// 	pStart := startY*width + startX
// 	pEnd := endY*width + endX
//
// 	p := y*width + x
//
// 	if pStart <= p && p <= pEnd {
// 		return true
// 	}
// 	return false
// }

// func fillRowWith(row *Row, r rune) {
// 	for i := range row.cells {
// 		row.cells[i].contents = string(r)
// 	}
// }

func isTabStop(emu *Emulator, x int) bool {
	data := emu.tabStops

	i := sort.Search(len(data), func(i int) bool { return data[i] >= x })
	if i < len(data) && data[i] == x {
		return true
		// x is present at data[i]
	}
	return false
}

func TestHandle_SCOSC_SCORC(t *testing.T) {
	tc := []struct {
		name       string
		seq        string
		hdIDs      []int
		posY, posX int
		set        bool
		msg        string
	}{
		{
			"move cursor, SCOSC, check", "\x1B[22;33H\x1B[s",
			[]int{CSI_CUP, CSI_SLRM_SCOSC},
			22 - 1, 33 - 1, true, "",
		},
		{
			"move cursor, SCOSC, move cursor, SCORC, check", "\x1B[33;44H\x1B[s\x1B[42;35H\x1B[u",
			[]int{CSI_CUP, CSI_SLRM_SCOSC, CSI_CUP, CSI_SCORC},
			33 - 1, 44 - 1, false, "",
		},
		{
			"SCORC, check", "\x1B[u",
			[]int{CSI_SCORC},
			0, 0, false, "Asked to restore cursor (SCORC) but it has not been saved",
		},
	}

	p := NewParser()
	emu := NewEmulator3(80, 40, 500)

	var place strings.Builder
	// emu.logI.SetOutput(&place) // redirect the output to the string builder
	// emu.logT.SetOutput(&place) // redirect the output to the string builder
	defer util.Log.Restore()
	util.Log.SetOutput(&place)
	util.Log.SetLevel(slog.LevelDebug)

	for i, v := range tc {
		place.Reset()

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

		switch i {
		case 0, 1:
			gotCol := emu.savedCursor_SCO.posX
			gotRow := emu.savedCursor_SCO.posY
			gotSet := emu.savedCursor_SCO.isSet

			if gotCol != v.posX || gotRow != v.posY || gotSet != v.set {
				t.Errorf("%s:\t %q expect {%d,%d,%t}, got %v", v.name, v.seq, v.posY, v.posX, v.set, emu.savedCursor_SCO)
			}
		case 2:
			got := strings.Contains(place.String(), v.msg)
			if !got {
				t.Errorf("%s:\t %q expect %s, got %s\n", v.name, v.seq, v.msg, place.String())
			}
		}
	}
}

func TestHandle_DECSC_DECRC_privSM_1048(t *testing.T) {
	tc := []struct {
		name       string
		seq        string
		hdIDs      []int
		posY, posX int
		originMode OriginMode
	}{
		// move cursor to (8,8), set originMode scrolling, DECSC
		// move cursor to (23,13), set originMode absolute, DECRC
		{
			"ESC DECSC/DECRC",
			"\x1B[?6h\x1B[9;9H\x1B7\x1B[24;14H\x1B[?6l\x1B8",
			[]int{CSI_privSM, CSI_CUP, ESC_DECSC, CSI_CUP, CSI_privRM, ESC_DECRC},
			8, 8, OriginMode_ScrollingRegion,
		},
		// move cursor to (9,9), set originMode absolute, privSM 1048
		// move cursor to (21,11), set originMode scrolling, privRM 1048
		{
			"CSI privSM/privRM 1048",
			"\x1B[10;10H\x1B[?6l\x1B[?1048h\x1B[22;12H\x1B[?6h\x1B[?1048l",
			[]int{CSI_CUP, CSI_privRM, CSI_privSM, CSI_CUP, CSI_privSM, CSI_privRM},
			9, 9, OriginMode_Absolute,
		},
	}

	p := NewParser()
	emu := NewEmulator3(80, 40, 500)
	var place strings.Builder
	// emu.logT.SetOutput(&place)
	defer util.Log.Restore()
	util.Log.SetOutput(&place)

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
			if hd.id != v.hdIDs[j] { // validate the control sequences id
				t.Errorf("%s: seq=%q expect %s, got %s\n", v.name, v.seq, strHandlerID[v.hdIDs[j]], strHandlerID[hd.id])
			}
		}

		// validate the result
		x := emu.posX
		y := emu.posY
		mode := emu.originMode

		if x != v.posX || y != v.posY || mode != v.originMode {
			t.Errorf("%s seq=%q expect (%d,%d), got (%d,%d)\n", v.name, v.seq, v.posY, v.posX, y, x)
		}
	}
}

// make sure this is a new initialized CharsetState
func isResetCharsetState(cs CharsetState) (ret bool) {
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

func TestHandle_DECSLRM(t *testing.T) {
	tc := []struct {
		name                    string
		seq                     string
		hdIDs                   []int
		leftMargin, rightMargin int
		posX, posY              int
	}{
		{
			"set left right margin, normal",
			"\x1B[?69h\x1B[4;70s",
			[]int{CSI_privSM, CSI_SLRM_SCOSC},
			3, 70, 0, 0,
		},
		{
			"set left right margin, missing right parameter",
			"\x1B[?69h\x1B[1s",
			[]int{CSI_privSM, CSI_SLRM_SCOSC},
			0, 80, 0, 0,
		},
		{
			"set left right margin, left parameter is zero",
			"\x1B[?69h\x1B[0s",
			[]int{CSI_privSM, CSI_SLRM_SCOSC},
			0, 80, 0, 0,
		},
	}

	p := NewParser()
	emu := NewEmulator3(80, 40, 500)
	var place strings.Builder
	// emu.logI.SetOutput(&place)
	// emu.logT.SetOutput(&place)
	defer util.Log.Restore()
	util.Log.SetOutput(&place)

	for _, v := range tc {

		// parse control sequence
		hds := make([]*Handler, 0, 16)
		hds = p.processStream(v.seq, hds)

		if len(hds) != 2 {
			t.Errorf("%s got %d handlers, expect 2 handlers.", v.name, len(hds))
		}

		// handle the control sequence
		for j, hd := range hds {
			hd.handle(emu)
			if hd.id != v.hdIDs[j] { // validate the control sequences id
				t.Errorf("%s:\t %q expect %s, got %s\n", v.name, v.seq, strHandlerID[v.hdIDs[j]], strHandlerID[hd.id])
			}
			switch j {
			case 0:
				gotMode := emu.horizMarginMode
				if gotMode != true {
					t.Errorf("%s:\t %q expect %t, got %t\n", v.name, v.seq, true, gotMode)
				}
			case 1:
				// validate the left/right margin
				gotLeft := emu.hMargin
				gotRight := emu.nColsEff
				if gotLeft != v.leftMargin || gotRight != v.rightMargin {
					t.Errorf("%s:\t %q expect (%d,%d), got (%d,%d)\n", v.name, v.seq, v.leftMargin, v.rightMargin, gotLeft, gotRight)
				}

				// validate the cursor row/col
				posY := emu.posY
				posXZ := emu.posX

				if posY != v.posY || posXZ != v.posX {
					t.Errorf("%s:\t %q expect (%d/%d), got (%d/%d)\n", v.name, v.seq, v.posX, v.posY, posXZ, posY)
				}
			}
		}
	}
}

func TestHandle_DECSLRM_Others(t *testing.T) {
	tc := []struct {
		name        string
		seq         string
		hdIDs       []int
		logMsg      string
		left, right int
		posY, posX  int
	}{
		{
			"DECLRMM disable", "\x1B[?69l\x1B[4;49s",
			[]int{CSI_privRM, CSI_SLRM_SCOSC},
			"", 0, 0, 0, 0,
		},
		{
			"DECLRMM enable, outof range", "\x1B[?69h\x1B[4;89s",
			[]int{CSI_privSM, CSI_SLRM_SCOSC},
			"Illegal arguments to SetLeftRightMargins", 0, 0, 0, 0,
		},
		{
			"DECLRMM OriginMode_ScrollingRegion, enable", "\x1B[?6h\x1B[?69h\x1B[4;69s", // DECLRMM: Set Left and Right Margins
			[]int{CSI_privSM, CSI_privSM, CSI_SLRM_SCOSC},
			"", 3, 69, 0, 3,
		},
	}

	p := NewParser()
	emu := NewEmulator3(80, 40, 500)
	var place strings.Builder
	// emu.logI.SetOutput(&place)
	// emu.logT.SetOutput(&place)
	defer util.Log.Restore()
	util.Log.SetOutput(&place)
	util.Log.SetLevel(slog.LevelDebug)

	for i, v := range tc {

		// parse control sequence
		hds := make([]*Handler, 0, 16)
		hds = p.processStream(v.seq, hds)

		if len(hds) < 2 {
			t.Errorf("%s got %d handlers, expect at lease 2 handlers.", v.name, len(hds))
		}

		// handle the control sequence
		for j, hd := range hds {
			hd.handle(emu)
			if hd.id != v.hdIDs[j] { // validate the control sequences id
				t.Errorf("%s:\t %q expect %s, got %s\n", v.name, v.seq, strHandlerID[v.hdIDs[j]], strHandlerID[hd.id])
			}
		}

		switch i {
		case 0:
			if emu.horizMarginMode {
				t.Errorf("%s: seq=%q expect %t, got %t\n", v.name, v.seq, false, emu.horizMarginMode)
			}
		case 1:
			got := strings.Contains(place.String(), v.logMsg)
			if !got {
				t.Errorf("%s: seq=%q expect %q, got %q\n", v.name, v.seq, v.logMsg, place.String())
			}
		case 2:
			// validate the left/right margin
			left := emu.hMargin
			right := emu.nColsEff
			if left != v.left || right != v.right {
				t.Errorf("%s: seq=%q expect left/right margin (%d,%d), got (%d,%d)\n", v.name, v.seq, v.left, v.right, left, right)
			}

			// validate the cursor row/col
			posY := emu.posY
			posX := emu.posX

			if posY != v.posY || posX != v.posX {
				t.Errorf("%s: seq=%q expect cursor (%d,%d), got (%d,%d)\n", v.name, v.seq, v.posY, v.posX, posY, posX)
			}
		}
	}
}

func TestHandle_DECSTR(t *testing.T) {
	tc := []struct {
		name           string
		seq            string
		hdIDs          []int
		insertMode     bool
		originMode     OriginMode
		showCursorMode bool
		cursorKeyMode  CursorKeyMode
		reverseVideo   bool
	}{
		{
			"DECSTR ",
			/*
				set ture for insertMode=true, originMode=OriginMode_ScrollingRegion,
				showCursorMode=false, cursorKeyMode = CursorKeyMode_Application,reverseVideo = true
				set top/bottom region = [1,30)
				we don't check the response of the above sequence, we choose the opposite value on purpose
				(finally) soft terminal reset, check the opposite result for the soft reset sequence.
			*/
			"\x1B[4h\x1B[?6h\x1B[?25l\x1B[?1h\x1B[2;30r\x1B[!p",
			[]int{CSI_SM, CSI_privSM, CSI_privRM, CSI_privSM, CSI_DECSTBM, CSI_DECSTR},
			false, OriginMode_Absolute, true, CursorKeyMode_ANSI, false,
		},
	}

	p := NewParser()
	emu := NewEmulator3(80, 40, 500)
	var place strings.Builder
	// emu.logI.SetOutput(&place)
	// emu.logT.SetOutput(&place)
	defer util.Log.Restore()
	util.Log.SetOutput(&place)

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
			if hd.id != v.hdIDs[j] { // validate the control sequences id
				t.Errorf("%s:\t %q expect %s, got %s\n", v.name, v.seq, strHandlerID[v.hdIDs[j]], strHandlerID[hd.id])
			}
		}

		// validate the insertMode
		insertMode := emu.insertMode
		if insertMode != v.insertMode {
			t.Errorf("%s seq=%q insertMode expect %t, got %t\n", v.name, v.seq, v.insertMode, insertMode)
		}
		originMode := emu.originMode
		if originMode != v.originMode {
			t.Errorf("%s seq=%q originMode expect %d, got %d\n", v.name, v.seq, v.originMode, originMode)
		}
		showCursorMode := emu.showCursorMode
		if showCursorMode != v.showCursorMode {
			t.Errorf("%s seq=%q showCursorMode expect %t, got %t\n", v.name, v.seq, v.showCursorMode, showCursorMode)
		}
		cursorKeyMode := emu.cursorKeyMode
		if cursorKeyMode != v.cursorKeyMode {
			t.Errorf("%s seq=%q cursorKeyMode expect %d, got %d\n", v.name, v.seq, v.cursorKeyMode, cursorKeyMode)
		}
		reverseVideo := emu.reverseVideo
		if reverseVideo != v.reverseVideo {
			t.Errorf("%s seq=%q reverseVideo expect %t, got %t\n", v.name, v.seq, v.reverseVideo, reverseVideo)
		}
	}
}

func TestHandle_CR_LF_VT_FF(t *testing.T) {
	tc := []struct {
		name  string
		hdIDs []int
		posY  int
		posX  int
		seq   string
	}{
		{"CR 1 ", []int{CSI_CUP, C0_CR}, 2, 0, "\x1B[3;2H\x0D"},
		{"CR 2 ", []int{CSI_CUP, C0_CR}, 4, 0, "\x1B[5;10H\x0D"},
		{"LF   ", []int{CSI_CUP, ESC_IND}, 3, 1, "\x1B[3;2H\x0C"},
		{"VT   ", []int{CSI_CUP, ESC_IND}, 4, 2, "\x1B[4;3H\x0B"},
		{"FF   ", []int{CSI_CUP, ESC_IND}, 5, 3, "\x1B[5;4H\x0C"},
		{"ESC D", []int{CSI_CUP, ESC_IND}, 6, 4, "\x1B[6;5H\x1BD"},
		{"CHA CR", []int{CSI_privSM, CSI_SLRM_SCOSC, CSI_CUP, C0_CR}, 4, 0, "\x1B[?69h\x1B[4;70s\x1B[5;1H\x0D"},
	}

	p := NewParser()
	emu := NewEmulator3(80, 40, 500)
	var place strings.Builder
	// emu.logI.SetOutput(&place)
	// emu.logT.SetOutput(&place)
	defer util.Log.Restore()
	util.Log.SetOutput(&place)

	for _, v := range tc {

		// parse the sequence
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

		// get final cursor position
		gotY := emu.posY
		gotX := emu.posX

		if gotX != v.posX || gotY != v.posY {
			t.Errorf("%s seq=%q expect cursor position (%d,%d), got (%d,%d)\n", v.name, v.seq, v.posX, v.posY, gotX, gotY)
		}
	}
}

func TestHandle_CSI_BS_FF_VT_CR_TAB(t *testing.T) {
	tc := []struct {
		name         string
		hdIDs        []int
		seq          string
		wantY, wantX int
	}{
		// call CUP first to set the start position
		{"CSI backspace number    ", []int{CSI_CUP, CSI_CUP}, "\x1B[1;1H\x1B[23;12\bH", 22, 0},       // undo last character in CSI sequence
		{"CSI backspace semicolon ", []int{CSI_CUP, CSI_CUP}, "\x1B[1;1H\x1B[23;\b;12H", 22, 11},     // undo last character in CSI sequence
		{"cursor down 1+3 rows VT ", []int{CSI_CUP, ESC_IND, CSI_CUD}, "\x1B[9;10H\x1B[3\vB", 12, 9}, //(8,9)->(9.9)->(12,9)
		{"cursor down 1+3 rows FF ", []int{CSI_CUP, ESC_IND, CSI_CUD}, "\x1B[9;10H\x1B[\f3B", 12, 9},
		{"cursor up 2 rows and CR ", []int{CSI_CUP, C0_CR, CSI_CUU}, "\x1B[8;9H\x1B[\r2A", 5, 0},
		{"cursor up 3 rows and CR ", []int{CSI_CUP, C0_CR, CSI_CUU}, "\x1B[7;7H\x1B[3\rA", 3, 0},
		{"cursor forward 2cols +HT", []int{CSI_CUP, C0_HT, CSI_CUF}, "\x1B[4;6H\x1B[2\tC", 3, 10},
		{"cursor forward 1cols +HT", []int{CSI_CUP, C0_HT, CSI_CUF}, "\x1B[6;3H\x1B[\t1C", 5, 9},
	}

	p := NewParser()
	emu := NewEmulator3(80, 40, 500)
	var place strings.Builder
	// emu.logI.SetOutput(&place)
	// emu.logT.SetOutput(&place)
	defer util.Log.Restore()
	util.Log.SetOutput(&place)

	for _, v := range tc {
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
				t.Errorf("%s: seq=%q hd[%d] expect %s, got %s\n", v.name, v.seq, j, strHandlerID[v.hdIDs[j]], strHandlerID[hd.id])
			}
		}

		// get the result
		gotY := emu.posY
		gotX := emu.posX
		if gotX != v.wantX || gotY != v.wantY {
			t.Errorf("%s: seq=%q expect cursor position (%d,%d), got (%d,%d)\n", v.name, v.seq, v.wantY, v.wantX, gotY, gotX)
		}
	}
}

func TestHandle_CUU_CUD_CUF_CUB_CUP_FI_BI(t *testing.T) {
	tc := []struct {
		name  string
		hdIDs []int
		wantY int
		wantX int
		seq   string
	}{
		// call CUP first to set the start position
		{"CSI Ps A  ", []int{CSI_CUP, CSI_CUU}, 14, 10, "\x1B[21;11H\x1B[6A"},
		{"CSI Ps B  ", []int{CSI_CUP, CSI_CUD}, 13, 10, "\x1B[11;11H\x1B[3B"},
		{"CSI Ps C  ", []int{CSI_CUP, CSI_CUF}, 10, 12, "\x1B[11;11H\x1B[2C"},
		{"CSI Ps D  ", []int{CSI_CUP, CSI_CUB}, 10, 12, "\x1B[11;21H\x1B[8D"},
		{"BS        ", []int{CSI_CUP, CSI_CUB}, 12, 11, "\x1B[13;13H\x08"}, // \x08 calls CUB
		{"CUB       ", []int{CSI_CUP, CSI_CUB}, 12, 11, "\x1B[13;13H\x1B[1D"},
		{"BS agin   ", []int{CSI_CUP, CSI_CUB}, 12, 10, "\x1B[13;12H\x08"}, // \x08 calls CUB
		{"DECFI     ", []int{CSI_CUP, ESC_FI}, 12, 22, "\x1B[13;22H\x1b9"},
		{"DECBI     ", []int{CSI_CUP, ESC_BI}, 12, 20, "\x1B[13;22H\x1b6"},
		{"CUU with STBM", []int{CSI_DECSTBM, CSI_CUP, CSI_CUU}, 0, 0, "\x1B[3;32r\x1B[2;1H\x1B[7A"},
		{"CUD with STBM", []int{CSI_DECSTBM, CSI_CUP, CSI_CUD}, 39, 79, "\x1B[3;36r\x1B[40;80H\x1B[3B"},
		{"CUB SLRM left", []int{CSI_privSM, CSI_SLRM_SCOSC, CSI_CUP, CSI_CUB}, 0, 0, "\x1B[?69h\x1B[3;76s\x1B[1;1H\x1B[5D"},
		{"CUB with right", []int{CSI_privSM, CSI_SLRM_SCOSC, CSI_CUP, CSI_CUB}, 39, 71, "\x1B[?69h\x1B[3;76s\x1B[40;77H\x1B[4D"},
		{"VT52 CUU", []int{CSI_CUP, CSI_privRM, CSI_CUU}, 19, 10, "\x1B[21;11H\x1B[?2l\x1BA"},
		{"VT52 CUD", []int{CSI_CUP, CSI_privRM, CSI_CUD}, 21, 10, "\x1B[21;11H\x1B[?2l\x1BB"},
		{"VT52 CUF", []int{CSI_CUP, CSI_privRM, CSI_CUF}, 20, 11, "\x1B[21;11H\x1B[?2l\x1BC"},
		{"VT52 CUB", []int{CSI_CUP, CSI_privRM, CSI_CUB}, 20, 9, "\x1B[21;11H\x1B[?2l\x1BD"},
	}

	p := NewParser()
	emu := NewEmulator3(80, 40, 40)
	var place strings.Builder
	// emu.logI.SetOutput(&place)
	// emu.logT.SetOutput(&place)
	defer util.Log.Restore()
	util.Log.SetOutput(&place)

	for _, v := range tc {
		p.reset()
		emu.resetTerminal()

		// parse the sequence
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

		// get final cursor position
		gotY := emu.posY
		gotX := emu.posX

		if gotX != v.wantX || gotY != v.wantY {
			t.Errorf("%s seq=%q expect cursor position (%d,%d), got (%d,%d)\n", v.name, v.seq, v.wantY, v.wantX, gotY, gotX)
		}
	}
}

func TestHandle_SU_SD(t *testing.T) {
	nCols := 8
	nRows := 5
	saveLines := 5
	tc := []struct {
		name      string
		hdIDs     []int
		emptyRows []int
		seq       string
	}{
		{"SU scroll up   2 lines", []int{CSI_SU}, []int{nRows - 2, nRows - 1}, "\x1B[2S"}, // bottom 2 is erased
		{"SD scroll down 3 lines", []int{CSI_SD}, []int{0, 1, 2}, "\x1B[3T"},              // top three is erased.
		{
			"SU scroll up 2 with SLRM",
			[]int{CSI_privSM, CSI_SLRM_SCOSC, CSI_SU},
			[]int{nRows - 2, nRows - 1},
			"\x1B[?69h\x1B[3;76s\x1B[2S",
		}, // bottom 2 is erased
		{
			"SD scroll down 3 with SLRM",
			[]int{CSI_privSM, CSI_SLRM_SCOSC, CSI_SD},
			[]int{0, 1, 2},
			"\x1B[?69h\x1B[3;76s\x1B[3T",
		}, // top three is erased.
	}

	p := NewParser()

	for _, v := range tc {
		// the terminal size is 8x5 [colxrow]
		emu := NewEmulator3(nCols, nRows, saveLines)
		var place strings.Builder
		// emu.logI.SetOutput(&place)
		// emu.logT.SetOutput(&place)
		defer util.Log.Restore()
		util.Log.SetOutput(&place)

		hds := make([]*Handler, 0, 16)
		hds = p.processStream(v.seq, hds)

		if len(hds) == 0 {
			t.Errorf("%s got zero handlers.", v.name)
		}

		fillCells(emu.cf)
		before := printCells(emu.cf)

		// handle the control sequence
		for j, hd := range hds {
			hd.handle(emu)
			if hd.id != v.hdIDs[j] { // validate the control sequences id
				t.Errorf("%s: seq=%q expect %s, got %s\n", v.name, v.seq, strHandlerID[v.hdIDs[j]], strHandlerID[hd.id])
			}
		}

		after := printCells(emu.cf)

		if !isEmptyRows(emu.cf, v.emptyRows...) {
			t.Errorf("%s:\n", v.name)
			t.Logf("[frame] scrollHead=%d marginTop=%d marginBottom=%d [emulator] marginTop=%d marginBottom=%d\n",
				emu.cf.scrollHead, emu.cf.marginTop, emu.cf.marginBottom, emu.marginTop, emu.marginBottom)
			t.Errorf("before:\n%s", before)
			t.Errorf("after:\n%s", after)
		}
	}
}

func TestHandle_HTS_TBC(t *testing.T) {
	tc := []struct {
		name  string
		hdIDs []int
		seq   string
	}{
		{"Set/Clear tab stop 1", []int{CSI_CUP, ESC_HTS, CSI_TBC}, "\x1B[21;19H\x1BH\x1B[g"}, // set tab stop; clear tab stop
		{"Set/Clear tab stop 2", []int{CSI_CUP, ESC_HTS, CSI_TBC}, "\x1B[21;39H\x1BH\x1B[0g"},
		{"Set/Clear tab stop 3", []int{CSI_CUP, ESC_HTS, CSI_TBC}, "\x1B[21;47H\x1BH\x1B[3g"},
	}
	// TODO test to see the HTS same position
	p := NewParser()
	emu := NewEmulator3(80, 40, 5)
	var place strings.Builder
	// emu.logI.SetOutput(&place)
	// emu.logT.SetOutput(&place)
	defer util.Log.Restore()
	util.Log.SetOutput(&place)

	for _, v := range tc {

		hds := make([]*Handler, 0, 16)
		hds = p.processStream(v.seq, hds)

		if len(hds) != 3 {
			t.Errorf("%s expect %d handlers, got %d handlers.", v.name, 3, len(hds))
		}

		// handle the control sequence
		for j, hd := range hds {
			hd.handle(emu)
			if hd.id != v.hdIDs[j] { // validate the control sequences id
				t.Errorf("%s: seq=%q expect %s, got %s\n", v.name, v.seq, strHandlerID[v.hdIDs[j]], strHandlerID[hd.id])
			}

			gotX := emu.posX
			switch j {
			case 0:
				if isTabStop(emu, gotX) {
					t.Errorf("%s seq=%q expect position %d is not tab stop, it is\n", v.name, v.seq, gotX)
				}
			case 1:
				if !isTabStop(emu, gotX) {
					t.Errorf("%s seq=%q expect position %d is not tab stop, it is\n", v.name, v.seq, gotX)
				}
			case 2:
				if isTabStop(emu, gotX) {
					t.Errorf("%s seq=%q expect position %d is not tab stop, it is\n", v.name, v.seq, gotX)
				}
			}
		}
	}
}

func TestHandle_HT_CHT_CBT(t *testing.T) {
	tc := []struct {
		name  string
		hdIDs []int
		posX  int
		seq   string
	}{
		{"HT case 1  ", []int{CSI_CUP, C0_HT}, 8, "\x1B[21;6H\x09"},                 // move to the next tab stop
		{"HT case 2  ", []int{CSI_CUP, C0_HT}, 16, "\x1B[21;10H\x09"},               // move to the next tab stop
		{"CBT back to the 3 tab", []int{CSI_CUP, CSI_CBT}, 8, "\x1B[21;30H\x1B[3Z"}, // move backward to the previous 3 tab stop
		{"CHT to the next 1 tab", []int{CSI_CUP, CSI_CHT}, 8, "\x1B[21;3H\x1B[I"},   // move to the next N tab stop
		{"CHT to the next 4 tab", []int{CSI_CUP, CSI_CHT}, 32, "\x1B[21;3H\x1B[4I"}, // move to the next N tab stop
		{"CHT to the right edge", []int{CSI_CUP, CSI_CHT}, 79, "\x1B[21;60H\x1B[4I"},
		{"CBT rule to left edge", []int{CSI_CUP, CSI_CBT}, 0, "\x1B[21;3H\x1B[3Z"}, // under tab rules
		{
			"CBT tab stop to left edge",
			[]int{CSI_CUP, ESC_HTS, CSI_CUP, ESC_HTS, CSI_CBT}, // set 2 tab stops, CBT 2 backwards
			0,
			"\x1B[21;4H\x1BH\x1B[21;7H\x1BH\x1B[2Z",
		},
	}

	p := NewParser()
	emu := NewEmulator3(80, 40, 5)
	var place strings.Builder
	// emu.logI.SetOutput(&place)
	// emu.logT.SetOutput(&place)
	defer util.Log.Restore()
	util.Log.SetOutput(&place)

	for _, v := range tc {

		hds := make([]*Handler, 0, 16)
		hds = p.processStream(v.seq, hds)

		if len(hds) < 2 {
			t.Errorf("%s expect %d handlers, got %d handlers.", v.name, 2, len(hds))
		}

		// handle the control sequence
		for j, hd := range hds {
			hd.handle(emu)
			if hd.id != v.hdIDs[j] { // validate the control sequences id
				t.Errorf("%s: seq=%q expect %s, got %s\n", v.name, v.seq, strHandlerID[v.hdIDs[j]], strHandlerID[hd.id])
			}
		}

		// get the result
		gotX := emu.posX
		if gotX != v.posX {
			t.Errorf("%s seq=%q expect cursor cols: %d, got %d)\n", v.name, v.seq, v.posX, gotX)
		}
	}
}

func TestHandle_LF_ScrollUp(t *testing.T) {
	tc := []struct {
		name             string
		posY             int
		expectScrollHead int
		seq              string
	}{
		{"LF within active area", 3, 0, "\x0A\x0A\x0A"},
		{"LF outof active area", 3, 2, "\x0A\x0A\x0A\x0A\x0A"},
		{"wrap around margin bottom", 3, 1, "\n\n\n\n\n\n\n\n\n\n\n\n"},
	}

	p := NewParser()
	emu := NewEmulator3(8, 4, 4) // this is the pre-condidtion for the test case.
	var place strings.Builder
	// emu.logI.SetOutput(&place)
	// emu.logT.SetOutput(&place)
	defer util.Log.Restore()
	util.Log.SetOutput(&place)

	for _, v := range tc {
		emu.resetTerminal()

		hds := make([]*Handler, 0, 16)
		hds = p.processStream(v.seq, hds)

		if len(hds) == 0 {
			t.Errorf("%s got %d handlers.", v.name, len(hds))
		}

		// handle the control sequence
		for _, hd := range hds {
			hd.handle(emu)
			// if i == 2 {
			// 	t.Logf("%s [frame] scrollHead=%d historyRows=%d [emulator] posY=%d\n",
			// 		v.name, emu.cf.scrollHead, emu.cf.historyRows, emu.posY)
			// }
		}

		gotY := emu.posY
		gotHead := emu.cf.scrollHead
		if gotY != v.posY || gotHead != v.expectScrollHead {
			t.Errorf("%s marginTop=%d, marginBottom=%d scrollHead=%d\n",
				v.name, emu.cf.marginTop, emu.cf.marginBottom, emu.cf.scrollHead)
			t.Errorf("%s seq=%q expect posY=%d, scrollHead=%d, got posY=%d, scrollHead=%d\n",
				v.name, v.seq, v.posY, v.expectScrollHead, gotY, gotHead)
		}
	}
}

func TestHandle_DECIC_DECDC(t *testing.T) {
	tc := []struct {
		name      string
		seq       string
		emptyCols []int
		hdIDs     []int
	}{
		// move cursor to start position, and perform insert and delete
		{"insert at left side ", "\x1B[2;1H\x1B[3'}", []int{0, 1, 2}, []int{CSI_CUP, CSI_DECIC}},
		{"insert at middle    ", "\x1B[2;4H\x1B[2'}", []int{3, 4}, []int{CSI_CUP, CSI_DECIC}},
		{"insert at right side", "\x1B[1;8H\x1B[2'}", []int{7}, []int{CSI_CUP, CSI_DECIC}},
		{"delete at left side ", "\x1B[1;1H\x1B[3'~", []int{5, 6, 7}, []int{CSI_CUP, CSI_DECDC}},
		{"delete at middle    ", "\x1B[1;4H\x1B[2'~", []int{6, 7}, []int{CSI_CUP, CSI_DECDC}},
		{"delete at right side", "\x1B[1;8H\x1B[2'~", []int{7}, []int{CSI_CUP, CSI_DECDC}},
	}

	for _, v := range tc {
		p := NewParser()
		emu := NewEmulator3(8, 4, 4) // this is the pre-condidtion for the test case.
		var place strings.Builder
		// emu.logI.SetOutput(&place)
		// emu.logT.SetOutput(&place)
		defer util.Log.Restore()
		util.Log.SetOutput(&place)

		fillCells(emu.cf)
		before := printCells(emu.cf)

		hds := make([]*Handler, 0, 16)
		hds = p.processStream(v.seq, hds)

		if len(hds) == 0 {
			t.Errorf("%s got %d handlers.", v.name, len(hds))
		}

		// handle the control sequence
		for j, hd := range hds {
			hd.handle(emu)
			if hd.id != v.hdIDs[j] { // validate the control sequences id
				t.Errorf("%s: seq=%q expect %s, got %s\n", v.name, v.seq, strHandlerID[v.hdIDs[j]], strHandlerID[hd.id])
			}
		}

		after := printCells(emu.cf)
		// validate the empty cell
		if !isEmptyCols(emu.cf, v.emptyCols...) {
			t.Errorf("%s:\n", v.name)
			t.Errorf("[before]\n%s", before)
			t.Errorf("[after ]\n%s", after)
		}
	}
}

func TestHandle_DECALN_RIS(t *testing.T) {
	tc := []struct {
		name  string
		seq   string
		y, x  int // check the last cell on the screen
		hdIDs []int
		want  string
	}{
		{"ESC DECLAN", "\x1B#8", 3, 7, []int{ESC_DECALN}, "E"},                 // the whole screen is filled with 'E'
		{"ESC RIS   ", "\x1Bc", 3, 7, []int{ESC_RIS}, " "},                     // after reset, the screen is empty
		{"ESC DECLAN", "\x1B#8", 3, 7, []int{ESC_DECALN}, "E"},                 // the whole screen is filled with 'E'
		{"VT52 ESC c", "\x1B[?2l\x1Bc", 3, 7, []int{CSI_privRM, ESC_RIS}, " "}, // after reset, the screen is empty
	}

	p := NewParser()
	emu := NewEmulator3(8, 4, 4) // this is the pre-condidtion for the test case.
	var place strings.Builder
	// emu.logI.SetOutput(&place)
	// emu.logT.SetOutput(&place)
	defer util.Log.Restore()
	util.Log.SetOutput(&place)

	for _, v := range tc {
		emu.resetTerminal()
		hds := make([]*Handler, 0, 16)
		hds = p.processStream(v.seq, hds)

		if len(hds) == 0 {
			t.Errorf("%s expect %d handlers, got %d handlers.", v.name, 2, len(hds))
		}

		before := printCells(emu.cf)
		// handle the control sequence
		for j, hd := range hds {
			hd.handle(emu)
			if hd.id != v.hdIDs[j] { // validate the control sequences id
				t.Errorf("%s: seq=%q expect %s, got %s\n",
					v.name, v.seq, strHandlerID[v.hdIDs[j]], strHandlerID[hd.id])
			}
		}

		after := printCells(emu.cf)
		theCell := emu.cf.getCell(v.y, v.x)
		if v.want != theCell.contents {
			t.Errorf("%s seq=%q expect %q on position (%d,%d), got %q\n", v.name, v.seq, v.want, v.y, v.x, theCell.contents)
			t.Errorf("[before]\n%s", before)
			t.Errorf("[after ]\n%s", after)
		}
	}
}

// use DECALN to fill the screen, then call ED to erase part of it.
func TestHandle_ED_IL_DL(t *testing.T) {
	tc := []struct {
		name     string
		hdIDs    []int
		tlY, tlX int
		brY, brX int
		seq      string
		msg      string
	}{
		// use CUP to move cursor to start position, use DECALN to fill the screen, then call ED,IL or DL
		{"ED erase below @ 1,0", []int{CSI_CUP, ESC_DECALN, CSI_ED}, 1, 0, 3, 7, "\x1B[2;1H\x1B#8\x1B[J", "unused"}, // Erase Below (default).
		{
			"VT52 ED erase below @ 1,0",
			[]int{CSI_CUP, ESC_DECALN, CSI_privRM, CSI_ED},
			1, 0, 3, 7, "\x1B[2;1H\x1B#8\x1B[?2l\x1BJ", "unused",
		}, // Erase Below (default).
		{"ED erase below @ 3,7", []int{CSI_CUP, ESC_DECALN, CSI_ED}, 3, 6, 3, 7, "\x1B[4;7H\x1B#8\x1B[0J", "unused"}, // Ps = 0  ⇒  Erase Below (default).
		{"ED erase above @ 3,6", []int{CSI_CUP, ESC_DECALN, CSI_ED}, 0, 0, 3, 6, "\x1B[4;7H\x1B#8\x1B[1J", "unused"}, // Ps = 1  ⇒  Erase Above.
		{"ED erase all", []int{CSI_CUP, ESC_DECALN, CSI_ED}, 0, 0, 3, 7, "\x1B[4;7H\x1B#8\x1B[2J", "unused"},         // Ps = 2  ⇒  Erase All.
		{"ED saved lines, all", []int{CSI_CUP, ESC_DECALN, CSI_ED}, 0, 0, 7, 7, "\x1B[4;7H\x1B#8\x1B[3J", "unused"},  // Ps = 3  ⇒  Erase saved lines.
		{"IL 1 lines @ 2,2 mid", []int{CSI_CUP, ESC_DECALN, CSI_IL}, 2, 0, 3, 7, "\x1B[3;3H\x1B#8\x1B[L", "unused"},
		{"IL 2 lines @ 1,0 bottom", []int{CSI_CUP, ESC_DECALN, CSI_IL}, 1, 0, 3, 7, "\x1B[2;1H\x1B#8\x1B[2L", "unused"},
		{"IL 4 lines @ 0,0 top", []int{ESC_DECALN, CSI_CUP, CSI_IL}, 0, 0, 3, 7, "\x1B#8\x1B[1;1H\x1B[4L", "unused"},
		{"DL 2 lines @ 1,0 top", []int{ESC_DECALN, CSI_CUP, CSI_DL}, 1, 0, 3, 7, "\x1B#8\x1B[2;1H\x1B[2M", "unused"},
		{"DL 1 lines @ 3,0 bottom", []int{ESC_DECALN, CSI_CUP, CSI_DL}, 3, 0, 3, 7, "\x1B#8\x1B[4;1H\x1B[1M", "unused"},
		{"ED default", []int{CSI_CUP, ESC_DECALN, CSI_ED}, 0, 0, 0, 0, "\x1B[4;7H\x1B#8\x1B[4J", "Erase in Display with illegal param"}, // Unhandled case
	}

	p := NewParser()
	// the default size of emu is 80x40 [colxrow]
	emu := NewEmulator3(8, 4, 4) // this is the pre-condidtion for the test case.
	var place strings.Builder
	// emu.logI.SetOutput(&place)
	// emu.logT.SetOutput(&place)
	defer util.Log.Restore()
	util.Log.SetOutput(&place)
	util.Log.SetLevel(slog.LevelDebug)

	for _, v := range tc {
		place.Reset()
		p.reset()
		emu.resetTerminal()

		hds := make([]*Handler, 0, 16)
		hds = p.processStream(v.seq, hds)

		if len(hds) == 0 {
			t.Errorf("%s got zero handlers.", v.name)
		}

		before := ""
		for j, hd := range hds {
			hd.handle(emu)
			if hd.id != v.hdIDs[j] { // validate the control sequences id
				t.Errorf("%s: seq=%q expect %s, got %s\n",
					v.name, v.seq, strHandlerID[v.hdIDs[j]], strHandlerID[hd.id])
			}
			if j == 1 {
				before = printCells(emu.cf)
				emu.cf.damage.reset()
			}
		}

		after := printCells(emu.cf)

		if v.tlX == 0 && v.tlY == 0 && v.brX == 0 && v.brY == 0 {
			if !strings.Contains(place.String(), v.msg) {

				t.Errorf("%s seq=%q\n", v.name, v.seq)
				t.Errorf("expect msg %s, got %s\n", v.msg, place.String())

			}
		} else {

			// calculate the expected dmage area
			dmg := Damage{}
			dmg.totalCells = emu.cf.damage.totalCells
			dmg.start, dmg.end = damageArea(emu.cf, v.tlY, v.tlX, v.brY, v.brX+1) // the end point is exclusive.

			if emu.cf.damage != dmg {
				t.Errorf("%s seq=%q\n", v.name, v.seq)
				t.Errorf("expect damage %v, got %v\n", dmg, emu.cf.damage)
				t.Errorf("[before]\n%s", before)
				t.Errorf("[after ]\n%s", after)
			}
		}
	}
}

func TestHandle_ICH2(t *testing.T) {
	tc := []struct {
		name     string
		hdIDs    []int
		tlY, tlX int // damage area top/left
		brY, brX int // damage area bottom/right
		seq      string
		emptyY   int // empty cell starting Y
		emptyX   int // empty cell starting X
		count    int // empty cells count number
		msg      string
	}{
		// use DECALN to fill the screen, use CUP to move cursor to start position, then call the sequence
		{
			"ICH right side with wrap length==0",
			[]int{CSI_CUP, Graphemes, Graphemes, Graphemes, Graphemes, CSI_CUP, CSI_ICH},
			1, 77, 2, 0,
			"\x1B[2;78Hwrap\x1B[2;78H\x1B[3@", 1, 77, 3, "unused",
		},
		{
			"ICH right side with wrap length!=0",
			[]int{CSI_CUP, Graphemes, Graphemes, Graphemes, Graphemes, CSI_CUP, CSI_ICH},
			1, 77, 2, 0,
			"\x1B[2;78Hwrap\x1B[2;78H\x1B[2@", 1, 77, 0, "unused",
		}, //"\033[2;78Hwrap\033[2;78H\033[3@"
	}
	p := NewParser()
	emu := NewEmulator3(80, 40, 40) // this is the pre-condidtion for the test case.
	var place strings.Builder
	// emu.logI.SetOutput(&place)
	// emu.logT.SetOutput(&place)
	defer util.Log.Restore()
	util.Log.SetOutput(&place)

	for _, v := range tc {

		hds := make([]*Handler, 0, 16)
		hds = p.processStream(v.seq, hds)

		if len(hds) == 0 {
			t.Errorf("%s got zero handlers.", v.name)
		}
		before := ""

		// call the handler
		for j, hd := range hds {
			if j == 1 {
				emu.cf.damage.reset()
			}
			hd.handle(emu)
			if hd.id != v.hdIDs[j] { // validate the control sequences id
				t.Errorf("%s: seq=%q expect %s, got %s\n",
					v.name, v.seq, strHandlerID[v.hdIDs[j]], strHandlerID[hd.id])
			}
			if j == len(hds)-2 {
				before = printCells(emu.cf, v.emptyY)
			}

		}
		after := printCells(emu.cf, v.emptyY, v.emptyY+1)

		if v.tlX == 0 && v.tlY == 0 && v.brX == 0 && v.brY == 0 {
			if !strings.Contains(place.String(), v.msg) {

				t.Errorf("%s seq=%q\n", v.name, v.seq)
				t.Errorf("expect msg %s, got %s\n", v.msg, place.String())

			}
		} else {
			// calculate the expected dmage area
			dmg := Damage{}
			dmg.totalCells = emu.cf.damage.totalCells
			dmg.start, dmg.end = damageArea(emu.cf, v.tlY, v.tlX, v.brY, v.brX+1) // the end point is exclusive.

			if emu.cf.damage != dmg || !isEmptyCells(emu.cf, v.emptyY, v.emptyX, v.count) {
				t.Errorf("%s seq=%q\n", v.name, v.seq)
				t.Errorf("expect damage %v, got %v\n", dmg, emu.cf.damage)
				t.Errorf("empty cells start (%d,%d) count=%d\n", v.emptyY, v.emptyX, v.count)
				t.Errorf("[before] %s", before)
				t.Errorf("[after ] %s", after)
			}
		}
	}
}

func TestHandle_ICH_EL_DCH_ECH(t *testing.T) {
	tc := []struct {
		name     string
		hdIDs    []int
		tlY, tlX int // damage area top/left
		brY, brX int // damage area bottom/right
		seq      string
		emptyY   int // empty cell starting Y
		emptyX   int // empty cell starting X
		count    int // empty cells count number
		msg      string
	}{
		// use DECALN to fill the screen, use CUP to move cursor to start position, then call the sequence
		{"ICH  in middle", []int{ESC_DECALN, CSI_CUP, CSI_ICH}, 0, 2, 0, 7, "\x1B#8\x1B[1;3H\x1B[2@", 0, 2, 2, "unused"},
		{"ICH right side", []int{ESC_DECALN, CSI_CUP, CSI_ICH}, 1, 5, 1, 7, "\x1B#8\x1B[2;6H\x1B[3@", 1, 5, 3, "unused"},
		{"ICH left side ", []int{ESC_DECALN, CSI_CUP, CSI_ICH}, 0, 0, 0, 7, "\x1B#8\x1B[1;1H\x1B[2@", 0, 0, 2, "unused"},
		{"   EL to right", []int{ESC_DECALN, CSI_CUP, CSI_EL}, 3, 3, 3, 7, "\x1B#8\x1B[4;4H\x1B[0K", 3, 3, 5, "unused"},
		{"   EL  to left", []int{ESC_DECALN, CSI_CUP, CSI_EL}, 3, 0, 3, 3, "\x1B#8\x1B[4;4H\x1B[1K", 3, 0, 4, "unused"},
		{"   EL      all", []int{ESC_DECALN, CSI_CUP, CSI_EL}, 3, 0, 3, 7, "\x1B#8\x1B[4;4H\x1B[2K", 3, 0, 8, "unused"},
		{"  DCH  at left", []int{ESC_DECALN, CSI_CUP, CSI_DCH}, 0, 0, 0, 7, "\x1B#8\x1B[1;1H\x1B[2P", 0, 6, 2, "unused"},
		{"  DCH at right", []int{ESC_DECALN, CSI_CUP, CSI_DCH}, 0, 5, 0, 7, "\x1B#8\x1B[1;6H\x1B[3P", 0, 5, 3, "unused"},
		{" DCH in middle", []int{ESC_DECALN, CSI_CUP, CSI_DCH}, 3, 3, 3, 7, "\x1B#8\x1B[4;4H\x1B[20P", 3, 3, 5, "unused"},
		{" ECH in middle", []int{ESC_DECALN, CSI_CUP, CSI_ECH}, 3, 3, 3, 4, "\x1B#8\x1B[4;4H\x1B[2X", 3, 3, 2, "unused"},
		{"   ECH at left", []int{ESC_DECALN, CSI_CUP, CSI_ECH}, 0, 0, 0, 4, "\x1B#8\x1B[1;1H\x1B[5X", 0, 0, 5, "unused"},
		{"  ECH at right", []int{ESC_DECALN, CSI_CUP, CSI_ECH}, 1, 5, 1, 7, "\x1B#8\x1B[2;6H\x1B[5X", 1, 5, 3, "unused"},
		{
			"ICH right side with wrap length==0",
			[]int{CSI_CUP, Graphemes, Graphemes, Graphemes, Graphemes, CSI_CUP, CSI_ICH},
			1, 5, 2, 0,
			"\x1B[2;6Hwrap\x1B[2;6H\x1B[3@", 1, 5, 0, "unused",
		},
		{
			"ICH right side with wrap length!=0",
			[]int{CSI_CUP, Graphemes, Graphemes, Graphemes, Graphemes, CSI_CUP, CSI_ICH},
			1, 5, 2, 0,
			"\x1B[2;6Hwrap\x1B[2;6H\x1B[2@", 1, 5, 0, "unused",
		},
		{
			"   EL  default",
			[]int{ESC_DECALN, CSI_CUP, CSI_EL},
			0, 0, 0, 0, "\x1B#8\x1B[4;4H\x1B[3K", 3, 0, 8, "Erase in Line with illegal param",
		},
		{
			"VT52 EL to right",
			[]int{ESC_DECALN, CSI_CUP, CSI_privRM, CSI_EL},
			3, 3, 3, 7,
			"\x1B#8\x1B[4;4H\x1B[?2l\x1BK", 3, 3, 5, "unused",
		},
	}
	p := NewParser()
	emu := NewEmulator3(8, 4, 4) // this is the pre-condidtion for the test case.
	var place strings.Builder
	// emu.logI.SetOutput(&place)
	// emu.logT.SetOutput(&place)
	defer util.Log.Restore()
	util.Log.SetOutput(&place)
	util.Log.SetLevel(slog.LevelDebug)

	for _, v := range tc {

		hds := make([]*Handler, 0, 16)
		hds = p.processStream(v.seq, hds)

		if len(hds) == 0 {
			t.Errorf("%s got zero handlers.", v.name)
		}
		before := ""

		// call the handler
		for j, hd := range hds {
			if j == 1 {
				emu.cf.damage.reset()
			}
			hd.handle(emu)
			if hd.id != v.hdIDs[j] { // validate the control sequences id
				t.Errorf("%s: seq=%q expect %s, got %s\n",
					v.name, v.seq, strHandlerID[v.hdIDs[j]], strHandlerID[hd.id])
			}
			if j == len(hds)-2 {
				before = printCells(emu.cf, v.emptyY)
			}

		}
		after := printCells(emu.cf, v.emptyY, v.emptyY+1)

		if v.tlX == 0 && v.tlY == 0 && v.brX == 0 && v.brY == 0 {
			if !strings.Contains(place.String(), v.msg) {

				t.Errorf("%s seq=%q\n", v.name, v.seq)
				t.Errorf("expect msg %s, got %s\n", v.msg, place.String())

			}
		} else {
			// calculate the expected dmage area
			dmg := Damage{}
			dmg.totalCells = emu.cf.damage.totalCells
			dmg.start, dmg.end = damageArea(emu.cf, v.tlY, v.tlX, v.brY, v.brX+1) // the end point is exclusive.

			if emu.cf.damage != dmg || !isEmptyCells(emu.cf, v.emptyY, v.emptyX, v.count) {
				t.Errorf("%s seq=%q\n", v.name, v.seq)
				t.Errorf("expect damage %v, got %v\n", dmg, emu.cf.damage)
				t.Errorf("empty cells start (%d,%d) count=%d\n", v.emptyY, v.emptyX, v.count)
				t.Errorf("[before] %s", before)
				t.Errorf("[after ] %s", after)
			}
		}
	}
}

func TestHandle_DEC_KPNM_KPAM(t *testing.T) {
	tc := []struct {
		name        string
		hdIDs       []int
		seq         string
		keypadMode0 KeypadMode
		keypadMode1 KeypadMode
	}{
		{
			"DEC KPNM application mode",
			[]int{ESC_DECKPNM, ESC_DECKPAM},
			"\x1b>\x1b=", KeypadMode_Normal, KeypadMode_Application,
		},
		{
			"DEC KPAM numeric mode",
			[]int{ESC_DECKPAM, ESC_DECKPNM},
			"\x1b=\x1b>", KeypadMode_Application, KeypadMode_Normal,
		},
		{
			"VT52 DEC KPAM KPAM KPNM",
			[]int{CSI_privRM, ESC_DECKPAM, ESC_DECKPNM},
			"\x1B[?2l\x1b=\x1b>", KeypadMode_Application, KeypadMode_Normal,
		},
	}

	p := NewParser()
	emu := NewEmulator3(8, 4, 4) // this is the pre-condidtion for the test case.

	for _, v := range tc {
		p.reset()
		emu.resetTerminal()

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
			got := emu.keypadMode
			switch j {
			case len(hds) - 2:
				if got != v.keypadMode0 {
					t.Errorf("%s seq=%q keypadmode expect %d, got %d\n", v.name, v.seq, v.keypadMode0, got)
				}
			case len(hds) - 1:
				if got != v.keypadMode1 {
					t.Errorf("%s seq=%q keypadmode expect %d, got %d\n", v.name, v.seq, v.keypadMode1, got)
				}
			}
		}
	}
}

func TestHandle_ESCSpaceHash_Unhandled(t *testing.T) {
	tc := []struct {
		name  string
		seq   string
		state int
		msg   string
	}{
		{"esc space F", "\x1B F", InputState_Normal, "S7C1T: Send 7-bit controls"},
		{"esc space G", "\x1B G", InputState_Normal, "S8C1T: Send 8-bit controls"},
		{"esc space L", "\x1B L", InputState_Normal, "Set ANSI conformance level 1"},
		{"esc space M", "\x1B M", InputState_Normal, "Set ANSI conformance level 2"},
		{"esc space N", "\x1B N", InputState_Normal, "Set ANSI conformance level 3"},
		{"esc space default", "\x1B O", InputState_Normal, "Unhandled input"}, // esc space unhandle
		{"esc hash 3", "\x1B#3", InputState_Normal, "DECDHL: Double-height, top half"},
		{"esc hash 4", "\x1B#4", InputState_Normal, "DECDHL: Double-height, bottom half"},
		{"esc hash 5", "\x1B#5", InputState_Normal, "DECSWL: Single-width line"},
		{"esc hash 6", "\x1B#6", InputState_Normal, "DECDWL: Double-width line"},
		{"esc hash default", "\x1B#9", InputState_Normal, "Unhandled input:"},        // esc hash unhandle
		{"csi quote default", "\x1B['o", InputState_Normal, "Unhandled input:"},      // csi quote unhandle
		{"csi space default", "\x1B[ o", InputState_Normal, "Unhandled input:"},      // csi space unhandle
		{"VT52 default", "\x1B[?2l\x1B\x1Bd", InputState_Normal, "Unhandled input:"}, // vt52 unhandle
		{"VT52 CAN SUB", "\x1B[?2l\x1B\x18\x1B\x1A", InputState_Normal, ""},
	}

	p := NewParser()
	var place strings.Builder // all the message is output to herer
	// p.logU.SetOutput(&place)
	defer util.Log.Restore()
	util.Log.SetOutput(&place)
	util.Log.SetLevel(slog.LevelDebug)

	for _, v := range tc {
		place.Reset()

		hds := make([]*Handler, 0, 16)
		p.processStream(v.seq, hds)

		state := p.getState()
		if state != v.state || !strings.Contains(place.String(), v.msg) {
			t.Errorf("%s seq=%q\n", v.name, v.seq)
			t.Errorf("state expect %s, got %s\n", strInputState[v.state], strInputState[state])
			t.Errorf("msg expect %s, got %s\n", v.msg, place.String())
		}
	}
}

type (
	ANSImode uint
	DECmode  uint
)

const (
	t_keyboardLocked ANSImode = iota
	t_insertMode
	t_localEcho
	t_autoNewlineMode
)

const (
	t_cursorKeyMode DECmode = iota
	t_reverseVideo
	t_originMode
	t_autoWrapMode
	t_showCursorMode
	t_focusEventMode
	t_altScrollMode
	t_altSendsEscape
	t_bracketedPasteMode
)

func t_getDECmode(emu *Emulator, which DECmode) bool {
	switch which {
	case t_reverseVideo:
		return emu.reverseVideo
	case t_autoWrapMode:
		return emu.autoWrapMode
	case t_showCursorMode:
		return emu.showCursorMode
	case t_focusEventMode:
		return emu.mouseTrk.focusEventMode
	case t_altScrollMode:
		return emu.altScrollMode
	case t_altSendsEscape:
		return emu.altSendsEscape
	case t_bracketedPasteMode:
		return emu.bracketedPasteMode
	}
	return false
}

// func t_resetDECmode(ds *emulator, which DECmode, value bool) {
// 	switch which {
// 	case t_reverseVideo:
// 		ds.reverseVideo = value
// 	case t_autoWrapMode:
// 		ds.autoWrapMode = value
// 	case t_showCursorMode:
// 		ds.showCursorMode = value
// 	case t_focusEventMode:
// 		ds.mouseTrk.focusEventMode = value
// 	case t_altScrollMode:
// 		ds.altScrollMode = value
// 	case t_altSendsEscape:
// 		ds.altSendsEscape = value
// 	case t_bracketedPasteMode:
// 		ds.bracketedPasteMode = value
// 	}
// }

func t_getANSImode(emu *Emulator, which ANSImode) bool {
	switch which {
	case t_keyboardLocked:
		return emu.keyboardLocked
	case t_insertMode:
		return emu.insertMode
	case t_localEcho:
		return emu.localEcho
	case t_autoNewlineMode:
		return emu.autoNewlineMode
	}
	return false
}

// func t_resetANSImode(emu *emulator, which ANSImode, value bool) {
// 	switch which {
// 	case t_keyboardLocked:
// 		emu.keyboardLocked = value
// 	case t_insertMode:
// 		emu.insertMode = value
// 	case t_localEcho:
// 		emu.localEcho = value
// 	case t_autoNewlineMode:
// 		emu.autoNewlineMode = value
// 	}
// }

func TestHandle_SM_RM(t *testing.T) {
	tc := []struct {
		name  string
		seq   string
		which ANSImode
		hdIDs []int
		want  bool
	}{
		{"SM: keyboardLocked ", "\x1B[2l\x1B[2h", t_keyboardLocked, []int{CSI_RM, CSI_SM}, true},
		{"SM: insertMode     ", "\x1B[4l\x1B[4h", t_insertMode, []int{CSI_RM, CSI_SM}, true},
		{"SM: localEcho      ", "\x1B[12l\x1B[12h", t_localEcho, []int{CSI_RM, CSI_SM}, false},
		{"SM: autoNewlineMode", "\x1B[20l\x1B[20h", t_autoNewlineMode, []int{CSI_RM, CSI_SM}, true},

		{"RM: keyboardLocked ", "\x1B[2h\x1B[2l", t_keyboardLocked, []int{CSI_SM, CSI_RM}, false},
		{"RM: insertMode     ", "\x1B[4h\x1B[4l", t_insertMode, []int{CSI_SM, CSI_RM}, false},
		{"RM: localEcho      ", "\x1B[12h\x1B[12l", t_localEcho, []int{CSI_SM, CSI_RM}, true},
		{"RM: autoNewlineMode", "\x1B[20h\x1B[20l", t_autoNewlineMode, []int{CSI_SM, CSI_RM}, false},
	}

	p := NewParser()
	emu := NewEmulator3(80, 40, 500)
	var place strings.Builder
	// emu.logI.SetOutput(&place)
	// emu.logT.SetOutput(&place)
	defer util.Log.Restore()
	util.Log.SetOutput(&place)

	for _, v := range tc {
		t.Run(v.name, func(t *testing.T) {
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
					t.Errorf("%s: seq=%q expect %s, got %s\n", v.name, v.seq, strHandlerID[v.hdIDs[j]], strHandlerID[hd.id])
				}
			}

			if v.want != t_getANSImode(emu, v.which) {
				t.Errorf("%s: seq=%q expect %t, got %t\n", v.name, v.seq, v.want, t_getANSImode(emu, v.which))
			}
		})
	}
}

func TestHandle_SM_RM_Unknow(t *testing.T) {
	tc := []struct {
		name string
		seq  string
		want string
	}{
		{"CSI SM unknow", "\x1B[21h", "Ignored bogus set mode"},
		{"CSI RM unknow", "\x1B[33l", "Ignored bogus reset mode"},
	}

	p := NewParser()
	emu := NewEmulator3(80, 40, 500)
	var place strings.Builder
	// emu.logI.SetOutput(&place)
	// emu.logW.SetOutput(&place)
	defer util.Log.Restore()
	util.Log.SetOutput(&place)

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

func TestHandle_privSM_privRM_67(t *testing.T) {
	tc := []struct {
		name         string
		seq          string
		hdIDs        []int
		bkspSendsDel bool
	}{
		{"enable DECBKM—Backarrow Key Mode", "\x1B[?67h", []int{CSI_privSM}, false},
		{"disable DECBKM—Backarrow Key Mode", "\x1B[?67l", []int{CSI_privRM}, true},
	}

	p := NewParser()
	emu := NewEmulator3(80, 40, 500)
	var place strings.Builder
	// emu.logI.SetOutput(&place)
	// emu.logT.SetOutput(&place)
	defer util.Log.Restore()
	util.Log.SetOutput(&place)

	for _, v := range tc {

		// process control sequence
		hds := make([]*Handler, 0, 16)
		hds = p.processStream(v.seq, hds)

		if len(hds) != 1 {
			t.Errorf("%s got %d handlers.", v.name, len(hds))
		}

		// handle the control sequence
		for j, hd := range hds {
			hd.handle(emu)
			if hd.id != v.hdIDs[j] { // validate the control sequences id
				t.Errorf("%s:\t %q expect %s, got %s\n", v.name, v.seq, strHandlerID[v.hdIDs[j]], strHandlerID[hd.id])
			}
		}

		got := emu.bkspSendsDel
		if got != v.bkspSendsDel {
			t.Errorf("%s:\t %q expect %t,got %t\n", v.name, v.seq, v.bkspSendsDel, got)
		}
	}
}

func TestHandle_privSM_privRM_BOOL(t *testing.T) {
	tc := []struct {
		name  string
		seq   string
		which DECmode
		hdIDs []int
		want  bool
	}{
		{"privSM: reverseVideo", "\x1B[?5l\x1B[?5h", t_reverseVideo, []int{CSI_privRM, CSI_privSM}, true},
		{"privSM: autoWrapMode", "\x1B[?7l\x1B[?7h", t_autoWrapMode, []int{CSI_privRM, CSI_privSM}, true},
		{"privSM: CursorVisible", "\x1B[?25l\x1B[?25h", t_showCursorMode, []int{CSI_privRM, CSI_privSM}, true},
		{"privSM: focusEventMode", "\x1B[?1004l\x1B[?1004h", t_focusEventMode, []int{CSI_privRM, CSI_privSM}, true},
		{"privSM: MouseAlternateScroll", "\x1B[?1007l\x1B[?1007h", t_altScrollMode, []int{CSI_privRM, CSI_privSM}, true},
		{"privSM: altSendsEscape", "\x1B[?1036l\x1B[?1036h", t_altSendsEscape, []int{CSI_privRM, CSI_privSM}, true},
		{"privSM: altSendsEscape", "\x1B[?1039l\x1B[?1039h", t_altSendsEscape, []int{CSI_privRM, CSI_privSM}, true},
		{"privSM: BracketedPaste", "\x1B[?2004l\x1B[?2004h", t_bracketedPasteMode, []int{CSI_privRM, CSI_privSM}, true},

		{"privRM: ReverseVideo", "\x1B[?5h\x1B[?5l", t_reverseVideo, []int{CSI_privSM, CSI_privRM}, false},
		{"privRM: AutoWrapMode", "\x1B[?7h\x1B[?7l", t_autoWrapMode, []int{CSI_privSM, CSI_privRM}, false},
		{"privRM: CursorVisible", "\x1B[?25h\x1B[?25l", t_showCursorMode, []int{CSI_privSM, CSI_privRM}, false},
		{"privRM: focusEventMode", "\x1B[?1004h\x1B[?1004l", t_focusEventMode, []int{CSI_privSM, CSI_privRM}, false},
		{"privRM: MouseAlternateScroll", "\x1B[?1007h\x1B[?1007l", t_altScrollMode, []int{CSI_privSM, CSI_privRM}, false},
		{"privRM: altSendsEscape", "\x1B[?1036h\x1B[?1036l", t_altSendsEscape, []int{CSI_privSM, CSI_privRM}, false},
		{"privRM: altSendsEscape", "\x1B[?1039h\x1B[?1039l", t_altSendsEscape, []int{CSI_privSM, CSI_privRM}, false},
		{"privRM: BracketedPaste", "\x1B[?2004h\x1B[?2004l", t_bracketedPasteMode, []int{CSI_privSM, CSI_privRM}, false},
	}

	p := NewParser()
	emu := NewEmulator3(80, 40, 500)
	var place strings.Builder
	// emu.logI.SetOutput(&place)
	// emu.logT.SetOutput(&place)
	defer util.Log.Restore()
	util.Log.SetOutput(&place)

	for _, v := range tc {
		t.Run(v.name, func(t *testing.T) {
			// process control sequence
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

			if v.want != t_getDECmode(emu, v.which) {
				t.Errorf("%s: %q\t expect %t, got %t\n", v.name, v.seq, v.want, t_getDECmode(emu, v.which))
			}
		})
	}
}

func TestHandle_privSM_privRM_Log(t *testing.T) {
	tc := []struct {
		name string
		seq  string
		hdID int
		want string
	}{
		{"privSM:   4", "\x1B[?4h", CSI_privSM, "DECSCLM: Set smooth scroll"},
		{"privSM:   8", "\x1B[?8h", CSI_privSM, "DECARM: Set auto-repeat mode"},
		{"privSM:  12", "\x1B[?12h", CSI_privSM, "Start blinking cursor"},
		// {"privSM:1001", "\x1B[?1001h", CSI_privSM, "Set VT200 Highlight Mouse mode"},
		{"privSM:unknow", "\x1B[?2022h", CSI_privSM, "set priv mode"},

		{"privRM:   4", "\x1B[?4l", CSI_privRM, "DECSCLM: Set jump scroll"},
		{"privRM:   8", "\x1B[?8l", CSI_privRM, "DECARM: Reset auto-repeat mode"},
		{"privRM:  12", "\x1B[?12l", CSI_privRM, "Stop blinking cursor"},
		// {"privRM:1001", "\x1B[?1001l", CSI_privRM, "Reset VT200 Highlight Mouse mode"},
		{"privRM:unknow", "\x1B[?2022l", CSI_privRM, "reset priv mode"},
	}

	p := NewParser()
	emu := NewEmulator3(80, 40, 500)
	var place strings.Builder
	// emu.logU.SetOutput(&place)
	// emu.logT.SetOutput(&place)
	defer util.Log.Restore()
	util.Log.SetOutput(&place)

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
				if hd.id != v.hdID { // validate the control sequences id
					t.Errorf("%s:\t %q expect %s, got %s\n", v.name, v.seq, strHandlerID[v.hdID], strHandlerID[hd.id])
				}
			}

			if !strings.Contains(place.String(), v.want) {
				t.Errorf("%s: %q\t expect %q, got %q\n", v.name, v.seq, v.want, place.String())
			}
		})
	}
}

func TestHandle_privSM_privRM_6(t *testing.T) {
	tc := []struct {
		name       string
		seq        string
		hdIDs      []int
		originMode OriginMode
	}{
		{"privSM:   6", "\x1B[?6l\x1B[?6h", []int{CSI_privRM, CSI_privSM}, OriginMode_ScrollingRegion},
		{"privRM:   6", "\x1B[?6h\x1B[?6l", []int{CSI_privSM, CSI_privRM}, OriginMode_Absolute},
	}

	p := NewParser()
	emu := NewEmulator3(80, 40, 500)
	var place strings.Builder
	// emu.logI.SetOutput(&place)
	// emu.logT.SetOutput(&place)
	defer util.Log.Restore()
	util.Log.SetOutput(&place)

	for _, v := range tc {
		t.Run(v.name, func(t *testing.T) {
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
					t.Errorf("%s: seq=%q expect %s, got %s\n", v.name, v.seq, strHandlerID[v.hdIDs[j]], strHandlerID[hd.id])
				}
			}

			got := emu.originMode
			if got != v.originMode {
				t.Errorf("%s: seq=%q expect %d, got %d\n", v.name, v.seq, v.originMode, got)
			}
		})
	}
}

func TestHandle_privSM_privRM_1(t *testing.T) {
	tc := []struct {
		name          string
		seq           string
		hdIDs         []int
		cursorKeyMode CursorKeyMode
	}{
		{"privSM:   1", "\x1B[?1l\x1B[?1h", []int{CSI_privRM, CSI_privSM}, CursorKeyMode_Application},
		{"privRM:   1", "\x1B[?1h\x1B[?1l", []int{CSI_privSM, CSI_privRM}, CursorKeyMode_ANSI},
	}

	p := NewParser()
	emu := NewEmulator3(80, 40, 500)
	var place strings.Builder
	// emu.logI.SetOutput(&place)
	// emu.logT.SetOutput(&place)
	defer util.Log.Restore()
	util.Log.SetOutput(&place)

	for _, v := range tc {
		t.Run(v.name, func(t *testing.T) {
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
					t.Errorf("%s: seq=%q expect %s, got %s\n", v.name, v.seq, strHandlerID[v.hdIDs[j]], strHandlerID[hd.id])
				}
			}

			got := emu.cursorKeyMode
			if got != v.cursorKeyMode {
				t.Errorf("%s: %q seq=expect %d, got %d\n", v.name, v.seq, v.cursorKeyMode, got)
			}
		})
	}
}

func TestHandle_privSM_privRM_MouseTrackingMode(t *testing.T) {
	tc := []struct {
		name  string
		seq   string
		hdIDs []int
		want  MouseTrackingMode
	}{
		{"privSM:   9", "\x1B[?9l\x1B[?9h", []int{CSI_privRM, CSI_privSM}, MouseTrackingMode_X10_Compat},
		{"privSM:1000", "\x1B[?1000l\x1B[?1000h", []int{CSI_privRM, CSI_privSM}, MouseTrackingMode_VT200},
		{"privSM:1001", "\x1B[?1001l\x1B[?1001h", []int{CSI_privRM, CSI_privSM}, MouseTrackingMode_VT200_HighLight},
		{"privSM:1002", "\x1B[?1002l\x1B[?1002h", []int{CSI_privRM, CSI_privSM}, MouseTrackingMode_VT200_ButtonEvent},
		{"privSM:1003", "\x1B[?1003l\x1B[?1003h", []int{CSI_privRM, CSI_privSM}, MouseTrackingMode_VT200_AnyEvent},

		{"privRM:   9", "\x1B[?9h\x1B[?9l", []int{CSI_privSM, CSI_privRM}, MouseTrackingMode_Disable},
		{"privRM:1000", "\x1B[?1000h\x1B[?1000l", []int{CSI_privSM, CSI_privRM}, MouseTrackingMode_Disable},
		{"privRM:1001", "\x1B[?1001h\x1B[?1001l", []int{CSI_privSM, CSI_privRM}, MouseTrackingMode_Disable},
		{"privRM:1002", "\x1B[?1002h\x1B[?1002l", []int{CSI_privSM, CSI_privRM}, MouseTrackingMode_Disable},
		{"privRM:1003", "\x1B[?1003h\x1B[?1003l", []int{CSI_privSM, CSI_privRM}, MouseTrackingMode_Disable},
	}

	p := NewParser()
	emu := NewEmulator3(80, 40, 500)
	var place strings.Builder
	// emu.logI.SetOutput(&place)
	// emu.logT.SetOutput(&place)
	defer util.Log.Restore()
	util.Log.SetOutput(&place)

	for _, v := range tc {
		t.Run(v.name, func(t *testing.T) {
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

			got := emu.mouseTrk.mode
			if got != v.want {
				t.Errorf("%s: %q\t expect %d, got %d\n", v.name, v.seq, v.want, got)
			}
		})
	}
}

func TestHandle_privSM_privRM_MouseTrackingEnc(t *testing.T) {
	tc := []struct {
		name  string
		seq   string
		hdIDs []int
		want  MouseTrackingEnc
	}{
		{"privSM:1005", "\x1B[?1005l\x1B[?1005h", []int{CSI_privRM, CSI_privSM}, MouseTrackingEnc_UTF8},
		{"privSM:1006", "\x1B[?1006l\x1B[?1006h", []int{CSI_privRM, CSI_privSM}, MouseTrackingEnc_SGR},
		{"privSM:1015", "\x1B[?1015l\x1B[?1015h", []int{CSI_privRM, CSI_privSM}, MouseTrackingEnc_URXVT},

		{"privRM:1005", "\x1B[?1005h\x1B[?1005l", []int{CSI_privSM, CSI_privRM}, MouseTrackingEnc_Default},
		{"privRM:1006", "\x1B[?1006h\x1B[?1006l", []int{CSI_privSM, CSI_privRM}, MouseTrackingEnc_Default},
		{"privRM:1015", "\x1B[?1015h\x1B[?1015l", []int{CSI_privSM, CSI_privRM}, MouseTrackingEnc_Default},
	}

	p := NewParser()
	emu := NewEmulator3(80, 40, 500)
	var place strings.Builder
	// emu.logI.SetOutput(&place)
	// emu.logT.SetOutput(&place)
	defer util.Log.Restore()
	util.Log.SetOutput(&place)

	for _, v := range tc {
		t.Run(v.name, func(t *testing.T) {
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

			got := emu.mouseTrk.enc
			if got != v.want {
				t.Errorf("%s: %q\t expect %d, got %d\n", v.name, v.seq, v.want, got)
			}
		})
	}
}

func TestHandle_privSM_privRM_47_1047(t *testing.T) {
	tc := []struct {
		name      string
		seq       string
		hdIDs     []int
		setMode   bool
		unsetMode bool
	}{
		{"privSM/RST 47", "\x1B[?47h\x1B[?47l", []int{CSI_privSM, CSI_privRM}, true, false},
		{"privSM/RST 1047", "\x1B[?1047h\x1B[?1047l", []int{CSI_privSM, CSI_privRM}, true, false},
	}

	p := NewParser()
	emu := NewEmulator3(80, 40, 500)
	var place strings.Builder
	// emu.logI.SetOutput(&place)
	// emu.logT.SetOutput(&place)
	defer util.Log.Restore()
	util.Log.SetOutput(&place)

	for _, v := range tc {

		// process control sequence
		hds := make([]*Handler, 0, 16)
		hds = p.processStream(v.seq, hds)

		if len(hds) != 2 {
			t.Errorf("%s got zero handlers.", v.name)
		}

		// handle the control sequence
		for j, hd := range hds {
			hd.handle(emu)
			if hd.id != v.hdIDs[j] { // validate the control sequences id
				t.Errorf("%s: seq=%q expect %s, got %s\n", v.name, v.seq, strHandlerID[v.hdIDs[j]], strHandlerID[hd.id])
			}
			got := emu.altScreenBufferMode
			switch j {
			case 0:
				if got != v.setMode {
					t.Errorf("%s: seq=%q expect %t, got %t\n", v.name, v.seq, true, got)
				}
			case 1:
				if got != v.unsetMode {
					t.Errorf("%s: seq=%q expect %t, got %t\n", v.name, v.seq, false, got)
				}
			}
		}
	}
}

func TestHandle_privSM_privRM_69(t *testing.T) {
	tc := []struct {
		name            string
		seq             string
		hdIDs           []int
		horizMarginMode bool
	}{
		{"privSM/privRM 69 combining", "\x1B[?69h\x1B[?69l", []int{CSI_privSM, CSI_privRM}, true},
	}

	p := NewParser()
	emu := NewEmulator3(80, 40, 500)
	var place strings.Builder
	// emu.logI.SetOutput(&place)
	// emu.logT.SetOutput(&place)
	defer util.Log.Restore()
	util.Log.SetOutput(&place)

	for _, v := range tc {

		// parse control sequence
		hds := make([]*Handler, 0, 16)
		hds = p.processStream(v.seq, hds)

		if len(hds) != 2 {
			t.Errorf("%s got %d handlers, expect 2 handlers.", v.name, len(hds))
		}

		// handle the control sequence
		for j, hd := range hds {
			hd.handle(emu)
			if hd.id != v.hdIDs[j] { // validate the control sequences id
				t.Errorf("%s: seq=%q expect %s, got %s\n", v.name, v.seq, strHandlerID[v.hdIDs[j]], strHandlerID[hd.id])
			}
			got := emu.horizMarginMode
			switch j {
			case 0:
				if got != true {
					t.Errorf("%s:\t %q expect %t, got %t\n", v.name, v.seq, true, got)
				}
			case 1:
				if got != false {
					t.Errorf("%s:\t %q expect %t, got %t\n", v.name, v.seq, false, got)
				}
			}
		}
	}
}

func TestHandle_privSM_privRM_1049(t *testing.T) {
	name := "privSM/RST 1049"
	// move cursor to 23,13
	// privSM 1049 enable altenate screen buffer
	// move cursor to 33,23
	// privRM 1049 disable normal screen buffer (false)
	// privRM 1049 set normal screen buffer (again for fast return)
	seq := "\x1B[24;14H\x1B[?1049h\x1B[34;24H\x1B[?1049l\x1B[?1049l"
	hdIDs := []int{CSI_CUP, CSI_privSM, CSI_CUP, CSI_privRM, CSI_privRM}

	p := NewParser()
	emu := NewEmulator3(80, 40, 500)
	var place strings.Builder
	// emu.logI.SetOutput(&place)
	// emu.logT.SetOutput(&place)
	defer util.Log.Restore()
	util.Log.SetOutput(&place)

	// parse the control sequence
	hds := make([]*Handler, 0, 16)
	hds = p.processStream(seq, hds)

	if len(hds) != len(hdIDs) {
		t.Errorf("%s got zero handlers.", name)
	}

	// handle the instruction
	for j, hd := range hds {
		hd.handle(emu)
		if hd.id != hdIDs[j] { // validate the control sequences id
			t.Errorf("%s:\t %q expect %s, got %s\n", name, seq, strHandlerID[hdIDs[j]], strHandlerID[hd.id])
		}

		switch j {
		case 0, 3:
			wantY := 23
			wantX := 13

			gotY := emu.posY
			gotX := emu.posX

			if gotX != wantX || gotY != wantY {
				t.Errorf("%s:\t %q expect [%d,%d], got [%d,%d]\n", name, seq, wantY, wantX, gotY, gotX)
			}

			want := false
			got := emu.altScreenBufferMode

			if got != want {
				t.Errorf("%s:\t %q expect %t, got %t\n", name, seq, want, got)
			}
		case 1:
			want := true
			got := emu.altScreenBufferMode

			if got != want {
				t.Errorf("%s:\t %q expect %t, got %t\n", name, seq, want, got)
			}
		case 2:
			wantY := 33
			wantX := 23

			gotY := emu.posY
			gotX := emu.posX

			if gotX != wantX || gotY != wantY {
				t.Errorf("%s:\t %q expect [%d,%d], got [%d,%d].\n", name, seq, wantY, wantX, gotY, gotX)
			}
		case 4:
			want := false
			got := emu.altScreenBufferMode

			if got != want {
				t.Errorf("%s:\t %q expect %t, got %t\n", name, seq, want, got)
			}

			logMsg := "Asked to restore cursor (DECRC) but it has not been saved"
			if !strings.Contains(place.String(), logMsg) {
				t.Errorf("%s seq=%q expect %q, got %q\n", name, seq, logMsg, place.String())
			}
		}
		// reset the output buffer
		place.Reset()
	}
}

func TestHandle_privSM_privRM_3(t *testing.T) {
	tc := []struct {
		name  string
		seq   string
		hdIDs []int
		mode  ColMode
	}{
		{"change to column Mode    132", "\x1B[?3h", []int{CSI_privSM}, ColMode_C132},
		{"change to column Mode     80", "\x1B[?3l", []int{CSI_privRM}, ColMode_C80},
		{"change to column Mode repeat", "\x1B[?3h\x1B[?3h", []int{CSI_privSM, CSI_privSM}, ColMode_C132},
	}

	p := NewParser()
	emu := NewEmulator3(80, 40, 500)
	var place strings.Builder
	// emu.logI.SetOutput(&place)
	// emu.logT.SetOutput(&place)
	defer util.Log.Restore()
	util.Log.SetOutput(&place)

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
			if hd.id != v.hdIDs[j] { // validate the control sequences id
				t.Errorf("%s:\t %q expect %s, got %s\n", v.name, v.seq, strHandlerID[v.hdIDs[j]], strHandlerID[hd.id])
			}
		}

		got := emu.colMode
		if got != v.mode {
			t.Errorf("%s:\t %q expect %d, got %d\n", v.name, v.seq, v.mode, got)
		}
	}
}

func TestHandle_privSM_privRM_2(t *testing.T) {
	tc := []struct {
		name                string
		seq                 string
		hdIDs               []int
		compatLevel         CompatibilityLevel
		isResetCharsetState bool
	}{
		{"privSM 2", "\x1B[?2h", []int{CSI_privSM}, CompatLevel_VT400, true},
		{"privRM 2", "\x1B[?2l", []int{CSI_privRM}, CompatLevel_VT52, true},
	}

	p := NewParser()
	emu := NewEmulator3(80, 40, 500)
	var place strings.Builder
	// emu.logI.SetOutput(&place)
	// emu.logT.SetOutput(&place)
	defer util.Log.Restore()
	util.Log.SetOutput(&place)

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
			if hd.id != v.hdIDs[j] { // validate the control sequences id
				t.Errorf("%s:\t %q expect %s, got %s\n", v.name, v.seq, strHandlerID[v.hdIDs[j]], strHandlerID[hd.id])
			}
		}

		// validate the result
		gotCL := emu.compatLevel
		gotRCS := isResetCharsetState(emu.charsetState)
		if v.isResetCharsetState != gotRCS || v.compatLevel != gotCL {
			t.Errorf("%s seq=%q expect reset CharsetState and compatbility level (%t,%d), got(%t,%d)",
				v.name, v.seq, v.isResetCharsetState, v.compatLevel, gotRCS, gotCL)
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
		{"OSC 0;Pt BEL        ", []int{OSC_0_1_2}, true, true, "\x1B]0;ada\x07", "ada"},
		{"OSC 1;Pt 7bit ST    ", []int{OSC_0_1_2}, true, false, "\x1B]1;adas\x1B\\", "adas"},
		{"OSC 2;Pt BEL chinese", []int{OSC_0_1_2}, false, true, "\x1B]2;[道德经]\x07", "[道德经]"},
		{"OSC 2;Pt BEL unusual", []int{OSC_0_1_2}, false, true, "\x1B]2;[neovim]\x1B78\x07", "[neovim]\x1B78"},
	}

	p := NewParser()
	emu := NewEmulator3(8, 4, 4) // this is the pre-condidtion for the test case.

	for _, v := range tc {
		var hd *Handler
		p.reset()
		// parse the sequence
		for _, ch := range v.seq {
			hd = p.ProcessInput(ch)
		}

		if hd != nil {
			// handle the instruction
			hd.handle(emu)

			// get the result
			windowTitle := emu.cf.windowTitle
			iconName := emu.cf.iconLabel

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
		{"OSC malform 1         ", "\x1B]ada\x1B\\", "OSC: no ';' exist"},
		{"OSC malform 2         ", "\x1B]7fy;ada\x1B\\", "OSC: illegal Ps parameter"},
		{"OSC Ps overflow: >120 ", "\x1B]121;home\x1B\\", "OSC: malformed command string"},
		{"OSC malform 3         ", "\x1B]7;ada\x1B\\", "unhandled OSC"},
	}
	p := NewParser()
	var place strings.Builder
	// p.logT.SetOutput(&place) // redirect the output to the string builder
	// p.logU.SetOutput(&place)
	defer util.Log.Restore()
	util.Log.SetOutput(&place)
	util.Log.SetLevel(slog.LevelDebug)

	for _, v := range tc {
		// reset the out put for every test case
		place.Reset()
		var hd *Handler

		// parse the sequence
		for _, ch := range v.seq {
			hd = p.ProcessInput(ch)
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

func TestHandle_OSC_52(t *testing.T) {
	tc := []struct {
		label      string
		hdIDs      []int
		wantPc     string
		wantPd     string
		wantString string
		noReply    bool
		seq        string
	}{
		{
			"new selection in c",
			[]int{OSC_52},
			"c", "YXByaWxzaAo=",
			"\x1B]52;c;YXByaWxzaAo=\x1B\\", true,
			"\x1B]52;c;YXByaWxzaAo=\x1B\\",
		},
		{
			"clear selection in cs",
			[]int{OSC_52, OSC_52},
			"cs", "",
			"\x1B]52;cs;x\x1B\\", true, // echo "aprilsh" | base64
			"\x1B]52;cs;YXByaWxzaAo=\x1B\\\x1B]52;cs;x\x1B\\",
		},
		{
			"empty selection",
			[]int{OSC_52},
			"pc", "5Zub5aeR5aiY5bGxCg==", // echo "四姑娘山" | base64
			"\x1B]52;pc;5Zub5aeR5aiY5bGxCg==\x1B\\", true,
			"\x1B]52;;5Zub5aeR5aiY5bGxCg==\x1B\\",
		},
		{
			"question selection",
			[]int{OSC_52, OSC_52},
			"", "", // don't care these values
			"\x1B]52;c;5Zub5aeR5aiY5bGxCg==\x1B\\", false,
			"\x1B]52;c0;5Zub5aeR5aiY5bGxCg==\x1B\\\x1B]52;c0;?\x1B\\",
		},
	}
	p := NewParser()
	emu := NewEmulator3(8, 4, 0)
	for _, v := range tc {
		emu.selectionData = ""
		emu.terminalToHost.Reset()

		t.Run(v.label, func(t *testing.T) {
			// process control sequence
			hds := make([]*Handler, 0, 16)
			hds = p.processStream(v.seq, hds)

			if len(hds) == 0 {
				t.Errorf("%s got zero handlers.", v.label)
			}

			// execute the control sequence
			for j, hd := range hds {
				hd.handle(emu)
				if hd.id != v.hdIDs[j] { // validate the control sequences id
					t.Errorf("%s: seq=%q expect %s, got %s\n", v.label, v.seq, strHandlerID[v.hdIDs[j]], strHandlerID[hd.id])
				}
			}

			if v.noReply {
				if v.wantString != emu.selectionData {
					t.Errorf("%s: seq=%q expect %q, got %q\n", v.label, v.seq, v.wantString, emu.selectionData)
				}
				for _, ch := range v.wantPc {
					if data, ok := emu.selectionStore[ch]; ok && data == v.wantPd {
						continue
					} else {
						t.Errorf("%s: seq=%q, expect[%c]%q, got [%c]%q\n", v.label, v.seq, ch, v.wantPc, ch, emu.selectionStore[ch])
					}
				}
			} else {
				got := emu.terminalToHost.String()
				if got != v.wantString {
					t.Errorf("%s: seq=%q, expect %q, got %q\n", v.label, v.seq, v.wantString, got)
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
		{"malform OSC 52 ", []int{OSC_52}, "OSC 52: can't find Pc parameter", "\x1B]52;23\x1B\\"},
		{"Pc not in range", []int{OSC_52}, "invalid Pc parameters", "\x1B]52;se;\x1B\\"},
	}
	p := NewParser()
	emu := NewEmulator3(8, 4, 0)
	var place strings.Builder
	// emu.logW.SetOutput(&place)
	defer util.Log.Restore()
	util.Log.SetOutput(&place)
	util.Log.SetLevel(slog.LevelDebug)

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
			[]int{OSC_4},
			"\x1B]4;1;rgb:8080/0000/0000\x1B\\", false,
			"\x1B]4;1;?\x1B\\",
		},
		{
			"query two color number",
			[]int{OSC_4},
			"\x1B]4;250;rgb:bcbc/bcbc/bcbc\x1B\\\x1B]4;1;rgb:8080/0000/0000\x1B\\", false,
			"\x1B]4;250;?;1;?\x1B\\",
		},
		{
			"query 8 color number",
			[]int{OSC_4},
			"\x1B]4;0;rgb:0000/0000/0000\x1B\\\x1B]4;1;rgb:8080/0000/0000\x1B\\\x1B]4;2;rgb:0000/8080/0000\x1B\\\x1B]4;3;rgb:8080/8080/0000\x1B\\\x1B]4;4;rgb:0000/0000/8080\x1B\\\x1B]4;5;rgb:8080/0000/8080\x1B\\\x1B]4;6;rgb:0000/8080/8080\x1B\\\x1B]4;7;rgb:c0c0/c0c0/c0c0\x1B\\", false,
			"\x1B]4;0;?;1;?;2;?;3;?;4;?;5;?;6;?;7;?\x1B\\",
		},
		{
			"missing ';' abort",
			[]int{OSC_4},
			"OSC 4: malformed argument, missing ';'", true,
			"\x1B]4;1?\x1B\\",
		},
		{
			"Ps malform abort",
			[]int{OSC_4},
			"OSC 4: can't parse c parameter", true,
			"\x1B]4;m;?\x1B\\",
		},
	}
	p := NewParser()
	emu := NewEmulator3(8, 4, 0)
	var place strings.Builder
	// emu.logW.SetOutput(&place)
	defer util.Log.Restore()
	util.Log.SetOutput(&place)
	util.Log.SetLevel(slog.LevelDebug)

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
			[]int{OSC_10_11_12_17_19},
			"\x1B]10;rgb:ffff/ffff/ffff\x1B\\\x1B]11;rgb:0000/8080/0000\x1B\\\x1B]17;rgb:0000/8080/0000\x1B\\\x1B]19;rgb:ffff/ffff/ffff\x1B\\\x1B]12;rgb:8080/8080/0000\x1B\\", false,
			"\x1B]10;?;11;?;17;?;19;?;12;?\x1B\\",
		},
		{
			"parse color parameter error",
			invalidColor, invalidColor, invalidColor,
			[]int{OSC_10_11_12_17_19},
			"OSC 10x: can't parse color index", true,
			"\x1B]10;?;m;?\x1B\\",
		},
		{
			"malform parameter",
			invalidColor, invalidColor, invalidColor,
			[]int{OSC_10_11_12_17_19},
			"OSC 10x: malformed argument, missing ';'", true,
			"\x1B]10;?;\x1B\\",
		},
		{
			"VT100 text foreground color: regular color",
			ColorWhite, invalidColor, invalidColor,
			[]int{OSC_10_11_12_17_19},
			"\x1B]10;rgb:ffff/ffff/ffff\x1B\\", false,
			"\x1B]10;?\x1B\\",
		},
		{
			"VT100 text background color: default color",
			invalidColor, ColorDefault, invalidColor,
			[]int{OSC_10_11_12_17_19},
			"\x1B]11;rgb:0000/0000/0000\x1B\\", false,
			"\x1B]11;?\x1B\\",
		},
		{
			"text cursor color: regular color",
			invalidColor, invalidColor, ColorGreen,
			[]int{OSC_10_11_12_17_19},
			"\x1B]12;rgb:0000/8080/0000\x1B\\", false,
			"\x1B]12;?\x1B\\",
		},
		{
			"text cursor color: default color",
			invalidColor, invalidColor, ColorDefault,
			[]int{OSC_10_11_12_17_19},
			"\x1B]12;rgb:0000/0000/0000\x1B\\", false,
			"\x1B]12;?\x1B\\",
		},
	}
	p := NewParser()
	emu := NewEmulator3(80, 40, 5)
	var place strings.Builder
	// emu.logW.SetOutput(&place)
	defer util.Log.Restore()
	util.Log.SetOutput(&place)
	util.Log.SetLevel(slog.LevelDebug)

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
				emu.cf.cursor.color = v.cursorColor
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
		{"DECRQSS normal", []int{DCS_DECRQSS}, "\x1BP1$r" + DEVICE_ID + "\x1B\\", false, "\x1BP$q\"p\x1B\\"},
		{"decrqss others", []int{DCS_DECRQSS}, "\x1BP0$rother\x1B\\", false, "\x1BP$qother\x1B\\"},
		{"DCS unimplement", []int{DCS_DECRQSS}, "DCS", true, "\x1BPunimplement\x1B78\x1B\\"},
	}
	p := NewParser()
	// p.logU = log.New(&place, "(Uimplemented): ", log.Ldate|log.Ltime|log.Lshortfile)
	emu := NewEmulator3(8, 4, 0)
	var place strings.Builder
	// p.logU.SetOutput(&place) // redirect the output to the string builder
	defer util.Log.Restore()
	util.Log.SetOutput(&place)

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

func TestHandle_VT52_EGM_ID(t *testing.T) {
	tc := []struct {
		name      string
		seq       string
		hdIDs     []int
		charsetGL *map[byte]rune
		resp      string
	}{
		{"VT52 ESC F", "\x1B[?2l\x1BF", []int{CSI_privRM, VT52_EGM}, &vt_DEC_Special, ""},
		{"VT52 ESC Z", "\x1B[?2l\x1BZ", []int{CSI_privRM, VT52_ID}, nil, "\x1B/Z"},
	}

	p := NewParser()
	emu := NewEmulator3(8, 4, 0)
	// var place strings.Builder
	// p.logU.SetOutput(&place)

	for _, v := range tc {
		// place.Reset()
		p.reset()
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
				if hd.id != v.hdIDs[j] { // validate the control sequences name
					t.Errorf("%s: seq=%q expect %s, got %s\n", v.name, v.seq, strHandlerID[v.hdIDs[j]], strHandlerID[hd.id])
				}
			}

			if v.resp == "" {
				got := emu.charsetState.g[emu.charsetState.gl]
				if !reflect.DeepEqual(got, v.charsetGL) {
					// if got != v.charsetGL {
					t.Errorf("%s seq=%q GL charset expect %p, got %p\n", v.name, v.seq, v.charsetGL, got)
				}
			} else {
				got := emu.terminalToHost.String()
				if got != v.resp {
					t.Errorf("%s seq=%q response expect %q, got %q\n", v.name, v.seq, v.resp, got)
				}
			}
		})
	}
}

func TestHandler(t *testing.T) {
	tc := []struct {
		name     string
		raw      string
		id       int
		sequence string
		ch       rune
	}{
		{"CUP", "\x1B[24;14H", CSI_CUP, "\x1B[24;14H", 'H'},
		{"TBC", "\x1B[3g", CSI_TBC, "\x1B[3g", 'g'},
	}

	p := NewParser()

	for _, v := range tc {
		p.ResetInput()

		hds := make([]*Handler, 0, 16)
		hds = p.processStream(v.raw, hds)

		if len(hds) != 1 {
			t.Errorf("%s should get 1 handler. got %d handlers\n", v.name, len(hds))
		}

		id := hds[0].GetId()
		if v.id != id {
			t.Errorf("%q expect ID %s, got %s\n", v.name, strHandlerID[v.id], strHandlerID[id])
		}

		sequence := hds[0].sequence
		if v.sequence != sequence {
			t.Errorf("%q expect sequence %q, got %q\n", v.name, v.sequence, sequence)
		}

		ch := hds[0].GetCh()
		if v.ch != ch {
			t.Errorf("%q expect ch %q, got %q\n", v.name, v.ch, ch)
		}
	}
}

func TestMixSequence(t *testing.T) {
	tc := []struct {
		name     string
		seq      string // data stream with control sequences
		hdNumber int    // expect handler number
	}{
		// CSI t
		// https://github.com/JetBrains/jediterm/commit/931243fe40f6c167e2a45c56d61d521d41e53e91
		// https://github.com/kovidgoyal/kitty/discussions/3636
		// CSI u
		// https://sw.kovidgoyal.net/kitty/keyboard-protocol/#functional-key-definitions
		{"vi sample", "\x1b[?1049h\x1b[22;0;0t\x1b[22;0t\x1b[?1h\x1b=\x1b[H\x1b[2J\x1b]11;?\a\x1b[?2004h\x1b[?u\x1b[c\x1b[?25h",
			10},
		{
			"vi sample 2", "\x1b[?25l\x1b(B\x1b[m\x1b[H\x1b[2J\x1b[>4;2m\x1b]112\a\x1b[2 q\x1b[?1002h\x1b[?1006h\x1b[38;2;233;233;244m\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[J\x1b[H",
			72},
		// {"vi output", "\x1b]11;rgb:0000/0000/0000\x1b\\\x1b[?64;1;9;15;21;22c",
		// 	[]int{}, 2},
	}
	p := NewParser()
	emu := NewEmulator3(8, 4, 0)

	// var place strings.Builder
	// defer util.Log.Restore()
	// util.Log.SetOutput(&place)

	for _, v := range tc {
		t.Run(v.name, func(t *testing.T) {
			hds := make([]*Handler, 0, 16)
			hds = p.processStream(v.seq, hds)

			if v.hdNumber != len(hds) {
				t.Errorf("%s expect %d handlers, got %d handlers\n", v.name, v.hdNumber, len(hds))
				for _, hd := range hds {
					hd.handle(emu)
					escCount := strings.Count(hd.sequence, "\x1b")
					if escCount > 1 {
						t.Logf("%s: id=%s seq=%q warn=ture\n", v.name, strHandlerID[hd.id], hd.sequence)
					} else {
						t.Logf("%s: id=%s seq=%q\n", v.name, strHandlerID[hd.id], hd.sequence)
					}
				}
			}
		})
	}
}
