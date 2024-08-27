// Copyright 2022 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package terminal

import (
	"fmt"
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/ericwq/aprilsh/util"
)

func TestNewFramebuffer3_Oversize(t *testing.T) {
	fb, _, _ := NewFramebuffer3(80, 40, SaveLineUpperLimit+1)
	if fb.saveLines != SaveLineUpperLimit {
		t.Errorf("#test NewFramebuffer3 oversize saveLines expect %d, got %d\n",
			SaveLineUpperLimit, fb.saveLines)
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
		rows   []Row
		w0, h0 int
		w1, h1 int
	}{
		{"shrink width and height", []Row{
			{109, 50, 'y'},
			{70, 50, 'y'},
			{69, 50, 'x'},
			{30, 50, 'x'},
			{29, 50, 'z'},
			{0, 50, 'z'},
		}, 80, 40, 50, 30},
		{"expend width and height", []Row{
			{0, 40, 'z'},
			{29, 40, 'z'},
			{40, 40, 'x'},
			{69, 40, 'x'},
			{70, 40, 'y'},
			{99, 40, 'y'},
		}, 60, 30, 80, 40},
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
		output := printCells(&fb)
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
	output := printCells(&fb)
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
		expect           string // expect cell content
		viewOffset       int    // the parameter for pageUp or pageDown
		expectViewOffset int    // the result of viewOffset
		pageType         int    // call pageUp:0 , pageDown:1 or pageToBottom:2
	}{
		{"from  0 to  1", "y", 1, 1, 0},        // y area bottom edge
		{"from  1 to 40", "y", 39, 40, 0},      // y area top edge
		{"from 40 to 41", "x", 1, 41, 0},       // x area bottom edge
		{"from 41 to 80", "x", 39, 80, 0},      // x area top edge
		{"from 80 to 41", "x", 39, 41, 1},      // x area bottom edge
		{"from 41 to 40", "y", 1, 40, 1},       // y area top edge
		{"from 40 to  1", "y", 39, 1, 1},       // y area bottom edge
		{"page to bottom", "z", 0, 0, 2},       // x area top edge
		{"page to bottom again", "z", 0, 0, 2}, // x area top edge again
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

func TestPhysicalRow_NoMargin(t *testing.T) {
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

func TestPhysicalRow_Margin(t *testing.T) {
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
	util.Logger.CreateLogger(io.Discard, true, slog.LevelDebug)
	// util.Logger.CreateLogger(os.Stderr, true, slog.LevelDebug)

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
		expect    string
		selection Rect
		snapTo    SelectSnapTo
		ok        bool
	}{
		{
			"english text", "\x1B[21;11Hfirst line  \x0D\x0C    second line   \x0D\x0C    3rd line    \x0D\x0C\x0D\x0C\x0D\x0C",
			"first line\n    second line\n    3rd line",
			Rect{Point{11, 20}, Point{79, 22}, false},
			SelectSnapTo_Word,
			true,
		},
		{
			"empty selection area", "", "",
			Rect{Point{0, 0}, Point{0, 0}, false},
			SelectSnapTo_Word,
			false,
		},
		{
			"extreme long line, selection area", "\x1B[31;61Hextreme long line will be wrapped.",
			"extreme long line will be wrapped.",
			Rect{Point{60, 30}, Point{80, 31}, false},
			SelectSnapTo_Word,
			true,
		},
		{
			"one row selection area", "\x1B[32;21Hone row selection area", "one row selection area",
			Rect{Point{14, 31}, Point{80, 31}, false},
			SelectSnapTo_Word,
			true,
		},
		{
			"rectangular selection area", "\x1B[35;21Hrectangular \x0D\x0Cselection area",
			"                    rectangular\nselection area",
			Rect{Point{0, 34}, Point{80, 35}, true},
			SelectSnapTo_Line,
			true,
		},
	}
	emu := NewEmulator3(80, 40, 0)
	// hide the log output
	util.Logger.CreateLogger(io.Discard, true, slog.LevelDebug)
	// util.Logger.CreateLogger(os.Stderr, true, slog.LevelDebug)

	for _, v := range tc {
		// print the stream to the screen
		emu.HandleStream(v.seq)

		// setup selection area
		emu.cf.selection = v.selection
		emu.cf.setSelectSnapTo(v.snapTo)

		// if i == 4 {
		// fmt.Printf("%s\n", printCells(emu.cf, 34, 35))

		// selection := emu.cf.getSnappedSelection()
		// fmt.Printf("#test getSelectedUtf8() snapTo=%d\n", emu.cf.snapTo)
		// fmt.Printf("#test getSelectedUtf8() selection=%s\n", &selection)
		// }

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
		expect string
		start  int
		count  int
	}{
		{"no intersection", "", 21, 6},
		{"copy range > damage area", "damage delta", 3, 20},
		{"copy range < damage area", "delta", 12, 17},
	}
	rawSeq := "\x1B[1;6Hdamage delta copy"

	emu := NewEmulator3(80, 40, 0)
	// hide the log output
	util.Logger.CreateLogger(io.Discard, true, slog.LevelDebug)
	// util.Logger.CreateLogger(os.Stderr, true, slog.LevelDebug)

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

func TestGetPhysicalRow(t *testing.T) {
	tc := []struct {
		label      string
		screenRow  int // screen row
		scrollHead int
		expect     int // physical row
	}{
		{"0 scrollHead, row    1", 1, 0, 1},
		{"0 scrollHead, row   42", 42, 0, 42},
		{"0 scrollHead, row   81", 81, 0, 81},
		{"0 scrollHead, row  120", 120, 0, 0},
		{"0 scrollHead, row  142", 142, 0, 22}, // if the row number is over the max, it's ring buffer
		{"0 scrollHead, row  -10", -10, 0, 110},
		{"0 scrollHead, row  -50", -50, 0, 70},
		{"0 scrollHead, row -120", -120, 0, 0},
		{"0 scrollHead, row -142", -142, 0, -22}, // if the row number is reverse over the max, it's wrong
		{"50 scrollHead,  row 12", 12, 50, 62},
		{"50 scrollHead,  row 90", 90, 50, 20},
		{"50 scrollHead  row 120", 120, 50, 50},
		{"50 scrollHead  row 130", 130, 50, 60},
		{"100 scrollHead, row 30", 30, 100, 10},
	}

	util.Logger.CreateLogger(io.Discard, true, slog.LevelDebug)
	// util.Logger.CreateLogger(os.Stderr, true, slog.LevelDebug)

	for _, v := range tc {
		t.Run(v.label, func(t *testing.T) {
			// prepare for test
			fb, _, _ := NewFramebuffer3(80, 40, 80)
			fb.scrollHead = v.scrollHead

			// test it
			got := fb.getPhysicalRow(v.screenRow)

			// validate
			if got != v.expect {
				t.Errorf("#TestGetPhysicalRow %q expect %d, got %d\n", v.label, v.expect, got)
			}
		})
	}
}

func TestASBRow(t *testing.T) {
	tc := []struct {
		label      string
		screenRow  int // screen row
		scrollHead int
		expect     int // physical row
	}{
		{"0 scrollHead, row    1", 1, 0, 1},
		{"0 scrollHead, row   10", 10, 0, 10},
		{"0 scrollHead, row   19", 19, 0, 19},
		{"0 scrollHead, row   20", 20, 0, 0},
		{"0 scrollHead, row   30", 30, 0, 10},
		{"0 scrollHead, row   40", 40, 0, 20},
		{"19 scrollHead, row   0", 00, 19, 19},
		{"19 scrollHead, row  19", 19, 19, 18},
		{"19 scrollHead, row  20", 20, 19, 19},
		{"19 scrollHead, row  21", 21, 19, 20},
		{"19 scrollHead, row  22", 22, 19, 21},
	}

	util.Logger.CreateLogger(io.Discard, true, slog.LevelDebug)
	// util.Logger.CreateLogger(os.Stderr, true, slog.LevelDebug)

	for _, v := range tc {
		t.Run(v.label, func(t *testing.T) {
			// prepare for test
			fb, _, _ := NewFramebuffer3(120, 20, 0)
			fb.scrollHead = v.scrollHead

			// test it
			got := fb.getPhysicalRow(v.screenRow)

			// validate
			if got != v.expect {
				t.Errorf("#TestGetPhysicalRow %q expect %d, got %d\n", v.label, v.expect, got)
			}
		})
	}
}

func TestFramebufferEqual(t *testing.T) {
	tc := []struct {
		label      string
		seq1, seq2 string
		expectStr  []string
		expect     bool
	}{
		{"diff size", "", "", []string{"saveLines="}, false},
		{"scroll up 5", "\x1B[5S", "", []string{"scrollHead="}, false},
		{"set cursor blink style", "\x1B[5 q", "", []string{"cursor.showStyle="}, false},
		{"change viewOffset", "", "", []string{"viewOffset="}, false},
		{"diff content", "world", "w0rld", []string{"newRow", "oldRow"}, false},
		{"diff content 0 saveLines", "world", "w0rld", []string{"newRow", "oldRow"}, false},
		{"set diff selection", "", "", []string{"selection="}, false},
	}

	var output strings.Builder

	util.Logger.CreateLogger(&output, true, slog.LevelDebug)

	for _, v := range tc {
		t.Run(v.label, func(t *testing.T) {
			var emu1, emu2 *Emulator
			if strings.Contains(v.label, "diff size") {
				emu1 = NewEmulator3(80, 40, 0)
				emu2 = NewEmulator3(80, 40, 10)
			} else if strings.Contains(v.label, "0 saveLines") {
				emu1 = NewEmulator3(80, 40, 0)
				emu2 = NewEmulator3(80, 40, 0)
			} else {
				emu1 = NewEmulator3(80, 40, 40)
				emu2 = NewEmulator3(80, 40, 40)
			}
			output.Reset()

			emu1.HandleStream(v.seq1)
			emu2.HandleStream(v.seq2)

			if strings.Contains(v.label, "viewOffset") {
				emu1.cf.scrollUp(8)
				emu1.cf.pageUp(8)
				// fmt.Printf("viewOffset=%d\n", emu1.cf.viewOffset)
			} else if strings.Contains(v.label, "diff selection") {
				emu1.cf.selection = Rect{Point{1, 1}, Point{40, 40}, true}
				// fmt.Printf("selection=%v\n", emu1.cf.selection)
			}

			// got := emu1.GetFramebuffer().equal(emu2.GetFramebuffer(), false)
			got := emu1.cf.Equal(emu2.GetFramebuffer())
			if got != v.expect {
				t.Errorf("%q expect %t, got %t\n", v.label, v.expect, got)
			}

			emu1.GetFramebuffer().equal(emu2.GetFramebuffer(), true)
			trace := output.String()
			for i := range v.expectStr {
				if !strings.Contains(trace, v.expectStr[i]) {
					t.Errorf("%q EqualTrace() expect \n%s, \ngot \n%s\n", v.label, v.expectStr[i], trace)
				}
				// t.Logf("%s\n", trace)
			}
		})
	}
}

func TestFramebufferEqual_caps(t *testing.T) {
	tc := []struct {
		label  string
		stack1 []int
		stack2 []int
		log    []string
		equal  bool
	}{
		{
			"same stack",
			[]int{},
			[]int{},
			[]string{},
			true,
		},
		{
			"different stack",
			[]int{2},
			[]int{},
			[]string{"kittyKbd.data", "kittyKbd.max"},
			false,
		},
	}

	var output strings.Builder

	util.Logger.CreateLogger(&output, true, slog.LevelDebug)
	// util.Logger.CreateLogger(os.Stdout, true, slog.LevelDebug)

	for _, v := range tc {
		fb1, _, _ := NewFramebuffer3(80, 40, 40)
		fb2, _, _ := NewFramebuffer3(80, 40, 40)
		output.Reset()

		t.Run(v.label, func(t *testing.T) {
			s1 := NewStack[int](len(v.stack1))
			for i := range v.stack1 {
				s1.Push(v.stack1[i])
			}
			fb1.kittyKbd = s1

			s2 := NewStack[int](len(v.stack2))
			for i := range v.stack2 {
				s2.Push(v.stack2[i])
			}
			fb2.kittyKbd = s2

			equal := fb1.equal(&fb2, false)
			if equal != v.equal {
				t.Errorf("%q expect %t, got %t\n", v.label, v.equal, equal)
			}

			fb1.equal(&fb2, true)
			trace := output.String()
			for i := range v.log {
				if !strings.Contains(trace, v.log[i]) {
					t.Errorf("%q equal trace expect \n%s, got \n%s\n", v.label, v.log[i], trace)
				}
			}
		})
	}
}

func TestGetRowsGap(t *testing.T) {
	tc := []struct {
		label string
		oldR  int
		newR  int
		gap   int
	}{
		{"oldR == newR", 7, 7, 0},
		{"oldR > newR", 8, 7, 79},
		{"oldR < newR", 7, 9, 2},
	}

	var output strings.Builder

	util.Logger.CreateLogger(&output, true, slog.LevelDebug)
	// util.Logger.CreateLogger(os.Stdout, true, slog.LevelDebug)

	for _, v := range tc {
		t.Run(v.label, func(t *testing.T) {
			fb, _, _ := NewFramebuffer3(80, 40, 40)
			gap := fb.getRowsGap(v.oldR, v.newR)
			if gap != v.gap {
				t.Errorf("%q expect %d, got %d\n", v.label, v.gap, gap)
			}
		})
	}
}

func TestOutputRow(t *testing.T) {
	tc := []struct {
		label  string
		seq    string
		expect string
		rowIdx int
		nCols  int
	}{
		{
			"chinese row", "中文输出",
			"[  0]中文输出                                                                        ",
			0, 80,
		},
	}

	var output strings.Builder

	util.Logger.CreateLogger(&output, true, slog.LevelDebug)
	// util.Logger.CreateLogger(os.Stdout, true, slog.LevelDebug)

	for _, v := range tc {
		t.Run(v.label, func(t *testing.T) {
			emu := NewEmulator3(80, 40, 0)
			emu.HandleStream(v.seq)

			row := emu.cf.getRow(v.rowIdx)
			got := outputRow(row, v.rowIdx, v.nCols)

			if got != v.expect {
				t.Errorf("%q expect \n%q, got \n%q\n", v.label, v.expect, got)
			}
		})
	}
}
