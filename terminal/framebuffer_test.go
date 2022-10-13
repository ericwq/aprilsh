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
	"io"
	"strings"
	"testing"
)

func TestNewFramebuffer3_Oversize(t *testing.T) {
	fb, _, _ := NewFramebuffer3(80, 40, SaveLineUpperLimit+1)
	if fb.saveLines != SaveLineUpperLimit {
		t.Errorf("#test NewFramebuffer3 oversize saveLines expect %d, got %d\n",
			SaveLineUpperLimit, fb.saveLines)
	}
}

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
	fb, _, _ := NewFramebuffer3(80, 40, 40)
	for _, v := range tc {
		fb.setWindowTitle(v.windowTitle)
		fb.setIconName(v.iconName)
		fb.setTitleInitialized()

		if !fb.isTitleInitialized() {
			t.Errorf("%q expect isTitleInitialized %t, got %t\n", v.name, true, fb.isTitleInitialized())
		}

		if fb.getIconName() != v.iconName {
			t.Errorf("%q expect IconName %q, got %q\n", v.name, v.iconName, fb.getIconName())
		}

		if fb.getWindowTitle() != v.windowTitle {
			t.Errorf("%q expect windowTitle %q, got %q\n", v.name, v.windowTitle, fb.getWindowTitle())
		}

		fb.prefixWindowTitle(v.prefix)
		if fb.getIconName() != v.expect {
			t.Errorf("%q expect prefix iconName %q, got %q\n", v.name, v.expect, fb.getIconName())
		}

	}
}

type Row struct {
	row     int
	count   int
	content rune
}

func TestResize(t *testing.T) {
	tc := []struct {
		name   string
		w0, h0 int
		w1, h1 int
		rows   []Row
	}{
		{"shrink width and height", 80, 40, 50, 30, []Row{
			{109, 50, 'y'},
			{70, 50, 'y'},
			{69, 50, 'x'},
			{30, 50, 'x'},
			{29, 50, 'z'},
			{0, 50, 'z'},
		}},
		{"expend width and height", 60, 30, 80, 40, []Row{
			{0, 40, 'z'},
			{29, 40, 'z'},
			{40, 40, 'x'},
			{69, 40, 'x'},
			{70, 40, 'y'},
			{99, 40, 'y'},
		}},
	}

	for _, v := range tc {
		fb, _, _ := NewFramebuffer3(v.w0, v.h0, v.h0*2)
		base := Cell{}
		fb.fillCells('x', base)
		// fmt.Printf("%s\n", printCells(fb))

		fb.scrollUp(v.h0)
		fb.fillCells('y', base)
		// fmt.Printf("%s\n", printCells(fb))

		fb.scrollUp(v.h0)
		fb.fillCells('z', base)
		// fmt.Printf("%s\n", printCells(fb))

		// fmt.Printf("%s\n", v.name)
		fb.resize(v.w1, v.h1)
		output := printCells(fb)
		// fmt.Printf("%s\n", output)

		for _, row := range v.rows {
			fmtStr := "[%3d] %s"
			if row.row == 0 {
				fmtStr = "[%3d]-%s"
			}
			indexStr := fmt.Sprintf(fmtStr, row.row,
				strings.Repeat(string(row.content), row.count))
			// fmt.Printf("%s\n", indexStr)

			if !strings.Contains(output, indexStr) {
				t.Errorf("%q expect %q, got empty\n", v.name, indexStr)
			}
		}
	}
}

func TestUnwrapCellStorage(t *testing.T) {
	rows := []Row{
		{0, 80, 'z'},
		{39, 80, 'z'},
		{40, 80, 'x'},
		{79, 80, 'x'},
		{80, 80, 'y'},
		{119, 80, 'y'},
	}
	name := "#test unwrapCellStorage() "

	fb, _, _ := NewFramebuffer3(80, 40, 80)
	base := Cell{}
	r := []rune{'x', 'y', 'z'}
	for i := 0; i < 3; i++ {
		fb.fillCells(r[i], base)
		if i != 2 {
			fb.scrollUp(40)
		}
	}
	// fmt.Printf("%s\n", printCells(fb))

	fb.unwrapCellStorage()
	output := printCells(fb)
	// fmt.Printf("%s\n", output)

	for _, row := range rows {
		fmtStr := "[%3d] %s"
		if row.row == 0 {
			fmtStr = "[%3d]-%s"
		}
		indexStr := fmt.Sprintf(fmtStr, row.row,
			strings.Repeat(string(row.content), row.count))
		// fmt.Printf("%s\n", indexStr)

		if !strings.Contains(output, indexStr) {
			t.Errorf("%q expect %q, got empty\n", name, indexStr)
		}
	}
}

func TestFullCopyCells(t *testing.T) {
	fb, _, _ := NewFramebuffer3(80, 40, 80)
	base := Cell{}
	r := []rune{'x', 'y', 'z'}
	for i := 0; i < 3; i++ {
		fb.fillCells(r[i], base)
		if i != 2 { // move scrollHead to row 80
			fb.scrollUp(40)
		}
	}
	// fmt.Printf("%s\n", printCells(fb))
	// move viewOffset to row 80
	fb.pageUp(80)

	// fmt.Printf("scrollHead=%d, viewOffset=%d, historyRow=%d\n",
	// 	fb.scrollHead, fb.viewOffset, fb.historyRows)

	dst := make([]Cell, fb.nCols*fb.nRows)
	fb.fullCopyCells(dst)

	// validate the result
	expect := "x"
	for _, c := range dst {
		if c.contents != expect {
			t.Errorf("#test fullCopyCells() expect %q, got %q", expect, c.contents)
			break
		}
	}
}

func TestDeltaCopyCells(t *testing.T) {
	fb, _, _ := NewFramebuffer3(80, 40, 80)
	base := Cell{}
	r := []rune{'x', 'y', 'z'}
	for i := 0; i < 3; i++ {
		fb.fillCells(r[i], base)
		if i != 2 { // move scrollHead to specified row
			fb.scrollUp(40)
		}
	}
	// fmt.Printf("%s\n", printCells(fb))
	// move viewOffset to specified row
	fb.pageUp(80)

	// fmt.Printf("scrollHead=%d, viewOffset=%d, historyRow=%d, damage=%v\n",
	// 	fb.scrollHead, fb.viewOffset, fb.historyRows, fb.damage)

	dst := make([]Cell, fb.nCols*fb.nRows)
	fb.deltaCopyCells(dst)

	// validate the result
	expect := "x"
	for _, c := range dst {
		if c.contents != expect {
			t.Errorf("#test fullCopyCells() expect %q, got %q", expect, c.contents)
			break
		}
	}
}

func TestPageUpDownBottom(t *testing.T) {
	// fill the framebuffer with 3 different content,scroll the active area.
	fb, _, _ := NewFramebuffer3(80, 40, 80)
	base := Cell{}
	r := []rune{'x', 'y', 'z'}
	for i := 0; i < 3; i++ {
		fb.fillCells(r[i], base)
		if i != 2 { // move scrollHead twice
			fb.scrollUp(40)
		}
	}

	tc := []struct {
		name             string
		viewOffset       int    // the parameter for pageUp or pageDown
		expect           string // expect cell content
		expectViewOffset int    // the result of viewOffset
		pageType         int    // call pageUp:0 , pageDown:1 or pageToBottom:2
	}{
		{"from  0 to  1", 1, "y", 1, 0},        // y area bottom edge
		{"from  1 to 40", 39, "y", 40, 0},      // y area top edge
		{"from 40 to 41", 1, "x", 41, 0},       // x area bottom edge
		{"from 41 to 80", 39, "x", 80, 0},      // x area top edge
		{"from 80 to 41", 39, "x", 41, 1},      // x area bottom edge
		{"from 41 to 40", 1, "y", 40, 1},       // y area top edge
		{"from 40 to  1", 39, "y", 1, 1},       // y area bottom edge
		{"page to bottom", 0, "z", 0, 2},       // x area top edge
		{"page to bottom again", 0, "z", 0, 2}, // x area top edge again
	}

	// fmt.Printf("%s\n", printCells(fb))

	for _, v := range tc {
		switch v.pageType {
		case 0:
			fb.pageUp(v.viewOffset)
		case 1:
			fb.pageDown(v.viewOffset)
		case 2:
			fb.pageToBottom()
		}

		// fmt.Printf("scrollHead=%2d, viewOffset=%2d, historyRow=%2d, mapping to physical row=%2d\n",
		// 	fb.scrollHead, fb.viewOffset, fb.historyRows, fb.getPhysicalRow(0-fb.viewOffset))

		if fb.viewOffset != v.expectViewOffset {
			t.Errorf("%q expect viewOffset %d, got %d\n", v.name, v.expectViewOffset, fb.viewOffset)
		}

		// validate the cell content with different viewOffset
		got := fb.cells[fb.getViewRowIdx(0)].contents
		if got != v.expect {
			t.Errorf("%q expect cell %q, got %q\n", v.name, v.expect, got)
		}
	}
}

func TestEraseInRow_Fail(t *testing.T) {
	fb, _, _ := NewFramebuffer3(80, 40, 80)
	base := Cell{}
	fb.eraseInRow(0, 0, 0, base)
}

func TestCopyRow(t *testing.T) {
	fb, _, _ := NewFramebuffer3(80, 40, 80)
	fb.copyRow(0, 0, 0, 0)
}

func TestGetPhysicalRow_NoMargin(t *testing.T) {
	fb, _, _ := NewFramebuffer3(80, 40, 40)

	// move scrollHead to specified row
	fb.scrollUp(40)

	// fill the screen
	base := Cell{}
	fb.fillCells('Z', base)
	// fmt.Printf("%s\n", printCells(fb))

	// set viewOffset to specified number
	fb.pageUp(40)

	// fmt.Printf("#test getPhysicalRow() nCols=%2d, nRows=%2d, saveLines=%2d, margin=%t\n",
	// 	fb.nCols, fb.nRows, fb.saveLines, fb.margin)
	// fmt.Printf("#test getPhysicalRow() scrollHead=%2d, marginTop=%2d, marginBottom=%2d, viewOffset=%2d, historyRow=%2d\n",
	// 	fb.scrollHead, fb.marginTop, fb.marginBottom, fb.viewOffset, fb.historyRows)

	for i := 0; i < 40; i++ {
		if i != fb.getPhysicalRow(i-fb.viewOffset) {
			t.Errorf("#test expect row %d map to physical row %d, got %d\n", i, i, fb.getPhysicalRow(i-fb.viewOffset))
			break
		}
		// fmt.Printf("#test getPhysicalRow() screen row=%2d, param=%3d, map to physical row=%2d\n",
		// 	i, i-fb.viewOffset, fb.getPhysicalRow(i-fb.viewOffset))
	}
}

func TestGetPhysicalRow_Margin(t *testing.T) {
	fb, _, _ := NewFramebuffer3(80, 40, 10)

	// set margin top/bottom
	fb.setMargins(2, 38)

	// fill the screen
	base := Cell{}
	fb.fillCells('Z', base)

	// move the scrollHead to specified row
	fb.scrollUp(10)

	tc := []struct {
		name   string
		in     int
		expect int
	}{
		{"negative max", -10, 40},          // show data in savedLines
		{"negative mini", -1, 49},          // the rest of savedLines
		{"margin top", 0, 0},               // show top margin
		{"margin top continue", 1, 1},      // show top margin
		{"scroll area top", 2, 12},         // jump to the scrollHead
		{"scroll area continue", 27, 37},   // continue to the bottom limitation
		{"scroll area wrap", 28, 2},        // wrap to the scroll area
		{"scroll area continue", 37, 11},   // continue until it reaches nRows
		{"margin bottom", 38, 38},          // show bottom margin
		{"margin bottom continue", 39, 39}, // show bottom margin
	}

	// fmt.Printf("%s\n", printCells(fb))
	// fmt.Printf("scrollHead=%d, marginTop=%d, marginBottom=%d, viewOffset=%d, historyRow=%d\n",
	// 	fb.scrollHead, fb.marginTop, fb.marginBottom, fb.viewOffset, fb.historyRows)

	for _, v := range tc {
		got := fb.getPhysicalRow(v.in)
		if got != v.expect {
			t.Errorf("%q getPhysicalRow expect %d, got %d\n", v.name, v.expect, got)
		}
	}

	// for i := -10; i < 40; i++ {
	// 	got := fb.getPhysicalRow(i)
	// 	fmt.Printf("#test getPhysicalRow in=%d, out=%d\n", i, got)
	// }
}

func TestCycleSelectSnapTo(t *testing.T) {
	fb, _, _ := NewFramebuffer3(80, 40, 10)

	tc := []SelectSnapTo{SelectSnapTo_Char, SelectSnapTo_Word, SelectSnapTo_Line, SelectSnapTo_Char}

	for i := 0; i < len(tc)-1; i++ {
		expect := tc[i+1]
		fb.cycleSelectSnapTo()
		got := fb.snapTo

		if got != expect {
			t.Errorf("#test cycleSelectSnapTo expect %d, got %d\n", expect, got)
		}
	}
}

func TestVscrollSelection(t *testing.T) {
	tc := []struct {
		label      string
		start      Rect
		vertOffset int
		expect     Rect
	}{
		{"move down", Rect{Point{0, 1}, Point{79, 3}, false}, 8, Rect{Point{0, 9}, Point{79, 11}, false}},
		{"move down oversize", Rect{Point{0, 1}, Point{79, 3}, false}, 88, Rect{Point{-1, -1}, Point{-1, -1}, false}},
		{"move up", Rect{Point{0, 37}, Point{79, 39}, false}, -4, Rect{Point{0, 33}, Point{79, 35}, false}},
		{"move up oversize", Rect{Point{0, 37}, Point{79, 39}, false}, -80, Rect{Point{-1, -1}, Point{-1, -1}, false}},
	}
	// nCols, nRows, savedLines
	fb, _, _ := NewFramebuffer3(80, 40, 40)

	for _, v := range tc {
		// the initial selection
		fb.selection = v.start

		// scroll the selection vertOffset
		fb.vscrollSelection(v.vertOffset)

		// validate the result
		if fb.selection != v.expect {
			t.Errorf("%q expect %s, got %s\n", v.label, &v.expect, &fb.selection)
		}
	}
}

func TestInvalidateSelection(t *testing.T) {
	tc := []struct {
		label     string
		selection Rect
		damage    Rect
		expect    Rect
	}{
		{
			"selection is on top of damage",
			Rect{Point{0, 1}, Point{79, 2}, true},
			Rect{Point{0, 8}, Point{79, 9}, true},
			Rect{Point{0, 1}, Point{79, 2}, true},
		},
		{
			"damage is on top of selection",
			Rect{Point{0, 8}, Point{79, 9}, true},
			Rect{Point{0, 1}, Point{79, 2}, true},
			Rect{Point{0, 8}, Point{79, 9}, true},
		},
		{
			"selection is empty",
			Rect{Point{0, 1}, Point{0, 1}, true},
			Rect{Point{0, 8}, Point{79, 9}, true},
			Rect{Point{0, 1}, Point{0, 1}, true},
		},
		{
			"selection is overlapped with damage",
			Rect{Point{0, 1}, Point{79, 4}, true},
			Rect{Point{0, 3}, Point{79, 9}, true},
			Rect{Point{-1, -1}, Point{-1, -1}, true},
		},
	}

	// nCols, nRows, savedLines
	fb, _, _ := NewFramebuffer3(80, 40, 40)

	for _, v := range tc {
		// the initial selection
		fb.selection = v.selection

		fb.invalidateSelection(&v.damage)

		// validate the result
		if fb.selection != v.expect {
			t.Errorf("%q expect %s, got %s\n", v.label, &v.expect, &fb.selection)
		}
	}
}

func TestGetSnappedSelection(t *testing.T) {
	tc := []struct {
		label     string
		seq       string
		selection Rect
		snapTo    SelectSnapTo
		expect    Rect
	}{
		{
			"english text, selection outside", "\x1B[24;14Henglish text, normal selection",
			Rect{Point{0, 23}, Point{79, 23}, false},
			SelectSnapTo_Word,
			Rect{Point{13, 23}, Point{43, 23}, false},
		},
		{
			"english text, selection inside", "\x1B[24;14Henglish text, normal selection",
			Rect{Point{15, 23}, Point{40, 23}, false},
			SelectSnapTo_Word,
			Rect{Point{13, 23}, Point{43, 23}, false},
		},
		{
			"chinese text, selection inside", "\x1B[34;14H中文字符，选择区在内",
			Rect{Point{15, 33}, Point{29, 33}, false},
			SelectSnapTo_Word,
			Rect{Point{13, 33}, Point{33, 33}, false},
		},
		{
			"chinese text, selection outside", "\x1B[34;14H中文字符，选择区在外",
			Rect{Point{5, 33}, Point{39, 33}, false},
			SelectSnapTo_Word,
			Rect{Point{13, 33}, Point{33, 33}, false},
		},
		{
			"selection is null", "",
			Rect{Point{-1, -1}, Point{-1, -1}, false},
			SelectSnapTo_Line,
			Rect{Point{-1, -1}, Point{-1, -1}, false},
		},
		{
			"selection is rectangular", "",
			Rect{Point{0, 1}, Point{79, 1}, true},
			SelectSnapTo_Line,
			Rect{Point{0, 1}, Point{79, 1}, true},
		},
		{
			"snap to char, language independent", "\x1B[41;14Hsnap to char, language independent",
			Rect{Point{15, 40}, Point{40, 40}, false},
			SelectSnapTo_Char,
			Rect{Point{15, 40}, Point{40, 40}, false},
		},
		{
			"snap to line, language independent", "\x1B[41;14Hsnap to line, language independent",
			Rect{Point{15, 40}, Point{40, 40}, false},
			SelectSnapTo_Line,
			Rect{Point{0, 40}, Point{80, 40}, false},
		},
	}

	emu := NewEmulator3(80, 40, 0)
	emu.logT.SetOutput(io.Discard) // hide the log output

	for _, v := range tc {
		// print the stream to the screen
		emu.HandleStream(v.seq)

		// fmt.Printf("%s\n", printCells(emu.cf, 33))
		// setup selection area
		emu.cf.selection = v.selection
		// setup selection state
		emu.cf.setSelectSnapTo(v.snapTo)

		got := emu.cf.getSnappedSelection()
		if v.expect != got {
			t.Errorf("%q expect %s, got %s\n", v.label, &v.expect, &got)
		}
	}
}

func TestGetSelectedUtf8(t *testing.T) {
	tc := []struct {
		label     string
		seq       string
		selection Rect
		snapTo    SelectSnapTo
		expect    string
		ok        bool
	}{
		{
			"english text", "\x1B[21;11Hfirst line  \x0D\x0C    second line   \x0D\x0C    3rd line    \x0D\x0C\x0D\x0C\x0D\x0C",
			Rect{Point{11, 20}, Point{79, 22}, false},
			SelectSnapTo_Word,
			"first line\n    second line\n    3rd line", true,
		},
		{
			"empty selection area", "",
			Rect{Point{0, 0}, Point{0, 0}, false},
			SelectSnapTo_Word,
			"", false,
		},
		{
			"extreme long line, selection area", "\x1B[31;61Hextreme long line will be wrapped.",
			Rect{Point{60, 30}, Point{80, 31}, false},
			SelectSnapTo_Word,
			"extreme long line will be wrapped.", true,
		},
		{
			"one row selection area", "\x1B[32;21Hone row selection area",
			Rect{Point{14, 31}, Point{80, 31}, false},
			SelectSnapTo_Word,
			"one row selection area", true,
		},
		{
			"rectangular selection area", "\x1B[35;21Hrectangular \x0D\x0Cselection area",
			Rect{Point{0, 34}, Point{80, 35}, true},
			SelectSnapTo_Line,
			"                    rectangular\nselection area", true,
		},
	}
	emu := NewEmulator3(80, 40, 0)
	// hide the log output
	emu.logT.SetOutput(io.Discard)

	for i, v := range tc {
		// print the stream to the screen
		emu.HandleStream(v.seq)

		// setup selection area
		emu.cf.selection = v.selection
		emu.cf.setSelectSnapTo(v.snapTo)

		if i == 4 {
			// fmt.Printf("%s\n", printCells(emu.cf, 34, 35))

			// selection := emu.cf.getSnappedSelection()
			// fmt.Printf("#test getSelectedUtf8() snapTo=%d\n", emu.cf.snapTo)
			// fmt.Printf("#test getSelectedUtf8() selection=%s\n", &selection)
		}

		ok, got := emu.cf.getSelectedUtf8()
		if v.expect != got || v.ok != ok {
			t.Errorf("%q expect %t,\n%q, got %t,\n%q\n", v.label, v.ok, v.expect, ok, got)
		}
	}
}

func TestResetDamage(t *testing.T) {
	fb, _, _ := NewFramebuffer3(80, 40, 40)

	// fill the screen
	base := Cell{}
	fb.fillCells('Z', base)

	// the expect value
	expect := fb.damage
	expect.start = 0
	expect.end = 0

	fb.resetDamage()
	got := fb.damage

	if got != expect {
		t.Errorf("#test resetDamage expect %v, got %v\n", expect, got)
	}
}

func TestGetCursor(t *testing.T) {
	fb, _, _ := NewFramebuffer3(80, 40, 40)

	// move scrollHead to specified row
	fb.scrollUp(40)

	// fill the screen
	base := Cell{}
	fb.fillCells('Z', base)

	expect := fb.cursor
	expect.posY += 40

	// set viewOffset to specified number
	fb.pageUp(40)
	got := fb.getCursor()

	if got != expect {
		t.Errorf("#test resetDamage expect %v, got %v\n", expect, got)
	}
}

func TestDamageDeltaCopy(t *testing.T) {
	tc := []struct {
		label  string
		start  int
		count  int
		expect string
	}{
		{"no intersection", 21, 6, ""},
		{"copy range > damage area", 3, 20, "damage delta"},
		{"copy range < damage area", 12, 17, "delta"},
	}
	rawSeq := "\x1B[1;6Hdamage delta copy"

	emu := NewEmulator3(80, 40, 0)
	// hide the log output
	emu.logT.SetOutput(io.Discard)

	// for easy typing
	fb := emu.cf

	// print the contents to the screen
	emu.HandleStream(rawSeq)

	// set the damage area
	fb.damage = Damage{5, 17, 3200}

	for _, v := range tc {
		dst := make([]Cell, fb.nCols*fb.nRows)
		fb.damageDeltaCopy(dst, v.start, v.count)

		// extract the result, ignore the target position
		got := extractFrom(dst)
		if v.expect != got {
			t.Errorf("%q expect %q, got %q\n", v.label, v.expect, got)
		}
	}
}

func extractFrom(cells []Cell) string {
	var b strings.Builder
	for _, v := range cells {
		if !v.dwidthCont {
			b.WriteString(v.contents)
		}
	}

	return b.String()
}
