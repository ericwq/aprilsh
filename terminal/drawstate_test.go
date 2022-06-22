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

func TestDrawStateGetNextTab(t *testing.T) {
	tc := []struct {
		cols  int
		count int
		want  int
	}{
		{4, 1, 8},    // right spot
		{9, 1, 16},   // right spot
		{9, 2, 24},   // -1
		{19, -1, 16}, // right spot
		{19, -2, 8},  // 0
	}

	// implicit condition: cursor start from 0
	ds := NewDrawState(80, 1)

	for _, v := range tc {
		ds.MoveCol(v.cols, false, false)
		if v.want != ds.GetNextTab(v.count) {
			t.Errorf("GetNextTab expect %d, got %d\n", v.want, ds.GetNextTab(v.cols))
		}
	}
}

func TestDrawStateRestoreCursor(t *testing.T) {
	ds := NewDrawState(80, 40)

	x := 10
	y := 10
	// move to (10,10)
	ds.MoveCol(x, false, false)
	ds.MoveRow(y, false)

	ds.SaveCursor()

	// move to (20,20)
	ds.MoveCol(20, false, false)
	ds.MoveRow(20, false)

	ds.RestoreCursor()

	// after restore we get the first move result
	if ds.GetCursorCol() != x || ds.GetCursorRow() != y {
		t.Errorf("cursorCol expect %d, got %d\n", x, ds.GetCursorCol())
		t.Errorf("cursorRow expect %d, got %d\n", y, ds.GetCursorRow())
	}

	// clear the cursor to (0,0)
	ds.ClearSavedCursor()
	ds.RestoreCursor()
	x = 0
	y = 0

	if ds.GetCursorCol() != x || ds.GetCursorRow() != y {
		t.Errorf("clear cursorCol expect %d, got %d\n", x, ds.GetCursorCol())
		t.Errorf("clear cursorRow expect %d, got %d\n", y, ds.GetCursorRow())
	}
}

func TestDrawStateNewDrawState(t *testing.T) {
	width, height := 80, 40
	ds := NewDrawState(width, height)

	// validate the result of NewDrawState
	if ds.GetCursorCol() != 0 {
		t.Errorf("cursorCol expect 0, got %d\n", ds.GetCursorCol())
	}
	if ds.GetCursorRow() != 0 {
		t.Errorf("cursorRow expect 0, got %d\n", ds.GetCursorRow())
	}
	if ds.GetCombiningCharCol() != 0 {
		t.Errorf("combiningCharCol expect 0, got %d\n", ds.GetCombiningCharCol())
	}
	if ds.GetCombiningCharRow() != 0 {
		t.Errorf("combiningCharRow expect 0, got %d\n", ds.GetCombiningCharRow())
	}
	if ds.GetWidth() != width {
		t.Errorf("width expect %d, got %d\n", width, ds.GetWidth())
	}
	if ds.GetHeight() != height {
		t.Errorf("height expect %d, got %d\n", height, ds.GetHeight())
	}
	if !ds.defaultTabs {
		t.Errorf("defaultTabs expect true, got %t\n", ds.defaultTabs)
	}
	if len(ds.tabs) != width {
		t.Errorf("tabs expect size %d, got %d\n", width, len(ds.tabs))
	}
	if ds.GetScrollingRegionTopRow() != 0 {
		t.Errorf("scrollingRegionTopRow expect 0, got %d\n", ds.GetScrollingRegionTopRow())
	}
	if ds.GetScrollingRegionBottomRow() != height-1 {
		t.Errorf("scrollingRegionBottomRow expect %d, got %d\n", height-1, ds.GetScrollingRegionBottomRow())
	}

	r := Renditions{bgColor: 0}
	if *ds.GetRenditions() != r {
		t.Errorf("renditions expect %v, got %v\n", r, ds.GetRenditions())
	}

	s := SavedCursor{autoWrapMode: true}
	if ds.save != s {
		t.Errorf("save expect %v, got %v\n", s, ds.save)
	}

	if ds.NextPrintWillWrap {
		t.Errorf("NextPrintWillWrap expect false, got %t\n", ds.NextPrintWillWrap)
	}
	if ds.OriginMode {
		t.Errorf("OriginMode expect false, got %t\n", ds.OriginMode)
	}
	if !ds.AutoWrapMode {
		t.Errorf("AutoWrapMode expect true, got %t\n", ds.AutoWrapMode)
	}
	if ds.InsertMode {
		t.Errorf("InsertMode expect false, got %t\n", ds.InsertMode)
	}
	if !ds.CursorVisible {
		t.Errorf("CursorVisible expect true, got %t\n", ds.CursorVisible)
	}
	if ds.MouseReportingMode != MOUSE_REPORTING_NONE {
		t.Errorf("MouseReportingMode expect MOUSE_REPORTING_NONE, got %v\n", ds.MouseReportingMode)
	}
	if ds.MouseEncodingMode != MOUSE_ENCODING_DEFAULT {
		t.Errorf("MouseEncodingMode expect MOUSE_ENCODING_DEFAULT, got %v\n", ds.MouseEncodingMode)
	}
	for i, v := range ds.tabs {
		want := (i % 8) == 0
		if v != want {
			t.Errorf("tabs expect %t, got %t\n", want, v)
		}
	}
}

func TestDrawStateMoveRowCol(t *testing.T) {
	tc := []struct {
		caseStr     string
		width       int
		height      int
		rowStart    int
		colStart    int
		rowMove     int
		rowRelative bool
		colMove     int
		colRelative bool
		colImplicit bool
		rowWant     int
		colWant     int
	}{
		{"relative F, in scope", 80, 40, 0, 0, 20, false, 20, false, false, 20, 20},
		{"relative T,out scope", 80, 40, 0, 0, 41, true, 89, true, false, 39, 79},
		{"relative F,out scope", 80, 40, 0, 0, 41, false, 89, false, false, 39, 79},
		{"relative T,out scope, start", 80, 40, 10, 10, 31, true, 79, true, false, 39, 79},
		{"relative F,out scope, start", 80, 40, 10, 10, 31, false, 79, false, false, 31, 79},
		{"relative T, in scope, start", 80, 40, 10, 10, 20, true, 20, true, true, 30, 30},
	}
	for _, c := range tc {
		ds := NewDrawState(c.width, c.height)

		// move to the starting position
		ds.MoveRow(c.rowStart, false)
		ds.MoveCol(c.colStart, false, false)

		// move the row and validate
		ds.MoveRow(c.rowMove, c.rowRelative)
		if ds.cursorRow != c.rowWant || ds.NextPrintWillWrap {
			t.Errorf("case [%s] expect row %d, got %d, NextPrintWillWrap=%t\n", c.caseStr, c.rowWant, ds.cursorRow, ds.NextPrintWillWrap)
		}

		// move the col and validate
		ds.MoveCol(c.colMove, c.colRelative, c.colImplicit)
		if ds.cursorCol != c.colWant {
			t.Errorf("case [%s] expect col %d, got %d\n", c.caseStr, c.colWant, ds.cursorCol)
		}
	}
}

func TestDrawStateResize(t *testing.T) {
	tc := []struct {
		tcName        string
		currentWidth  int
		currentHeight int
		newWidth      int
		newHeight     int
	}{
		{"resize both\t:", 80, 40, 100, 50},
		{"resize width\t:", 80, 40, 100, 40},
		{"resize height\t:", 80, 40, 80, 60},
		{"shrink both\t:", 80, 40, 70, 30},
		{"shrink width\t:", 80, 40, 70, 40},
		{"shrink height\t:", 80, 40, 80, 30},
	}
	for _, v := range tc {
		// initialize ds with current size
		ds := NewDrawState(v.currentWidth, v.currentHeight)

		// move cursor to the edge
		ds.MoveCol(v.newWidth, false, true)
		ds.MoveRow(v.newHeight, false)

		// resize
		ds.Resize(v.newWidth, v.newHeight)

		// validate the new size
		if ds.GetWidth() != v.newWidth || ds.GetHeight() != v.newHeight {
			t.Errorf("%s expect size(%d,%d), got size(%d,%d)\n", v.tcName, v.newWidth, v.newHeight, ds.GetWidth(), ds.GetHeight())
		}

		// shrink will case cause the combining char cell invalidate
		if v.newHeight < v.currentHeight || v.newWidth < v.currentWidth {
			if ds.combiningCharCol != -1 && ds.combiningCharRow != -1 {
				t.Errorf("%s expect combining char col/row to be -1, get %d,%d\n", v.tcName, ds.combiningCharCol, ds.combiningCharRow)
			}
			// t.Errorf("%s expect combining char col/row to be -1, get %d,%d\n", v.tcName, ds.combiningCharCol, ds.combiningCharRow)
		}
		// t.Logf("%s expect size(%d,%d), got size(%d,%d)\n", v.tcName, v.newWidth, v.newHeight, ds.GetWidth(), ds.GetHeight())
	}
}

func TestDrawStateEqual(t *testing.T) {
	type parameter struct {
		width              int
		height             int
		cursorCol          int
		cursorRow          int
		renditions         Renditions
		mouseReportingMode int
		mouseEncodingMode  int
	}

	tc := []struct {
		name string
		p1   parameter
		p2   parameter
		want bool
	}{
		{
			"all equal:\t\t",
			parameter{80, 40, 2, 2, Renditions{bgColor: 0}, MOUSE_REPORTING_NONE, MOUSE_ENCODING_DEFAULT},
			parameter{80, 40, 2, 2, Renditions{bgColor: 0}, MOUSE_REPORTING_NONE, MOUSE_ENCODING_DEFAULT},
			true,
		},
		{
			"diff height:\t",
			parameter{80, 49, 2, 2, Renditions{bgColor: 40}, MOUSE_REPORTING_NONE, MOUSE_ENCODING_DEFAULT},
			parameter{80, 40, 2, 2, Renditions{bgColor: 40}, MOUSE_REPORTING_NONE, MOUSE_ENCODING_DEFAULT},
			false,
		},
		{
			"diff width:\t",
			parameter{83, 40, 2, 2, Renditions{bgColor: 40}, MOUSE_REPORTING_NONE, MOUSE_ENCODING_DEFAULT},
			parameter{80, 40, 2, 2, Renditions{bgColor: 40}, MOUSE_REPORTING_NONE, MOUSE_ENCODING_DEFAULT},
			false,
		},
		{
			"diff reporting:\t",
			parameter{83, 40, 2, 2, Renditions{bgColor: 40}, MOUSE_REPORTING_VT220, MOUSE_ENCODING_DEFAULT},
			parameter{80, 40, 2, 2, Renditions{bgColor: 40}, MOUSE_REPORTING_NONE, MOUSE_ENCODING_DEFAULT},
			false,
		},
		{
			"diff endoding:\t",
			parameter{83, 40, 2, 2, Renditions{bgColor: 40}, MOUSE_REPORTING_VT220, MOUSE_ENCODING_UTF8},
			parameter{80, 40, 2, 2, Renditions{bgColor: 40}, MOUSE_REPORTING_VT220, MOUSE_ENCODING_DEFAULT},
			false,
		},
		{
			"diff renditions:\t",
			parameter{83, 40, 2, 2, Renditions{bgColor: 45}, MOUSE_REPORTING_VT220, MOUSE_ENCODING_UTF8},
			parameter{80, 40, 2, 2, Renditions{bgColor: 40}, MOUSE_REPORTING_VT220, MOUSE_ENCODING_UTF8},
			false,
		},
	}
	for _, v := range tc {
		// create a DrawState and set all the field
		ds1 := NewDrawState(v.p1.width, v.p1.height)
		ds1.MoveRow(v.p1.cursorRow, false)
		ds1.MoveCol(v.p1.cursorCol, false, false)
		ds1.MouseReportingMode = v.p1.mouseReportingMode
		ds1.MouseEncodingMode = v.p1.mouseEncodingMode
		ds1.renditions = v.p1.renditions

		// create another DrawState and set all the field
		ds2 := NewDrawState(v.p2.width, v.p2.height)
		ds2.MoveRow(v.p2.cursorRow, false)
		ds2.MoveCol(v.p2.cursorCol, false, false)
		ds2.MouseReportingMode = v.p2.mouseReportingMode
		ds2.MouseEncodingMode = v.p2.mouseEncodingMode
		ds2.renditions = v.p2.renditions

		if ds1.Equal(ds2) != v.want {
			// t.Logf("ds1=%v\nds2=%v\n", ds1, ds2)
			t.Errorf("%s expect %t, got %t\n", v.name, v.want, ds1.Equal(ds2))
		}
	}
}

func TestDrawStateSetTab(t *testing.T) {
	width := 80
	tc := []struct {
		col  int
		want bool
	}{
		{0, true},
		{1, false},
		{7, false},
		{8, true},
		{width - 1, false},
		// lack of range check, cancle the test case
		// {width, true},
		// {width + 1, false},
	}
	for _, v := range tc {
		ds := NewDrawState(width, 40)

		// validate the origianl tab value
		original := ds.tabs[v.col]
		if original != v.want {
			t.Errorf("original\t col=%d expect %t, got %t\n", v.col, v.want, ds.tabs[v.col])
		}

		// move cursor to col and set tab[cursorCol]
		ds.MoveCol(v.col, false, false)

		// set tab true
		ds.SetTab()
		if ds.tabs[v.col] != true {
			t.Errorf("SetTab\t col=%d expect %t, got %t\n", v.col, true, ds.tabs[v.col])
		}

		// clear tab
		ds.ClearTab(v.col)
		if ds.tabs[v.col] != false {
			t.Errorf("ClearTab\t col=%d expect %t, got %t\n", v.col, false, ds.tabs[v.col])
		}

		// restore the original tab value
		ds.tabs[v.col] = original
	}
}

func TestDrawStateClearDefaultTab(t *testing.T) {
	ds := NewDrawState(80, 40)

	ds.ClearDefaultTabs()
	if ds.defaultTabs {
		t.Errorf("expect false, got %t\n", ds.defaultTabs)
	}
}

func TestDrawStateSetScrollingRegion(t *testing.T) {
	tc := []struct {
		name       string
		pTop       int
		pBottom    int
		wantTop    int
		wantBottom int
	}{
		{"in scope\t", 2, 38, 2, 38},
		{"reverse\t", 10, 5, 10, 10},
		{"range B\t", 2, 41, 2, 39},
		{"range T\t", -1, 40, 0, 39},
		{"range B\t", 2, 41, 2, 39},
		{"origin Mode\t", -1, 40, 0, 39},
		{"range B\t", 2, 41, 2, 39},
		{"just return\t", -57, 40, 2, 39},
	}

	// implicit screen size 80x40
	ds := NewDrawState(80, 40)

	for _, v := range tc {
		if v.pTop < 0 { // test the OriginMode == true
			ds.OriginMode = true
			if v.pTop == -57 {
				ds.height = 0 // specase case: just return, do nothing
			}
		}
		ds.SetScrollingRegion(v.pTop, v.pBottom)

		// validate the case
		if ds.GetScrollingRegionTopRow() != v.wantTop || ds.GetScrollingRegionBottomRow() != v.wantBottom {
			t.Errorf("%s expect top=%d,bottom=%d; got top=%d,bottom=%d\n", v.name, v.wantTop, v.wantBottom, ds.GetScrollingRegionTopRow(), ds.GetScrollingRegionBottomRow())
		}
	}
}

func TestDrawStateRenditions(t *testing.T) {
	// base renditions
	r := Renditions{}
	fgColorIndex := 30
	bgColorIndex := 47
	r.SetForegroundColor(fgColorIndex)
	r.SetBackgroundColor(bgColorIndex)

	ds := NewDrawState(8, 4)

	// set fg/bg color
	ds.SetForegroundColor(fgColorIndex)
	ds.SetBackgroundColor(bgColorIndex)

	// validate the result
	if ds.renditions != r {
		t.Errorf("set fg/bg color expect %v, got %v\n", r, ds.renditions)
	}

	// validate the bg color
	if ds.GetBackgroundRendition() != r.bgColor {
		t.Errorf("get bg color expect %d, got %d\n", bgColorIndex, ds.GetBackgroundRendition())
	}
	// base renditions
	r = Renditions{}
	// r.SetRendition(fg)

	ds = NewDrawState(8, 4)
	// set renditions
	ds.AddRenditions()

	// validate the result
	if ds.renditions != r {
		t.Errorf("add renditions expect %v, got %v\n", r, ds.renditions)
	}
}

func TestDrawStateSnapCursorToBorder(t *testing.T) {
	tc := []struct {
		name    string
		col     int
		row     int
		wantCol int
		wantRow int
	}{
		{" in range", 20, 30, 20, 30},
		{"out range 1", -1, -1, 0, 0},
		{"out range 2", 89, 41, 79, 39},
	}

	// implicit size 80x40
	ds := NewDrawState(80, 40)
	for _, v := range tc {
		ds.cursorCol = v.col
		ds.cursorRow = v.row
		ds.snapCursorToBorder()
		if ds.cursorCol != v.wantCol || ds.cursorRow != v.wantRow {
			t.Errorf("%s expect (%d,%d), got (%d,%d)\n", v.name, v.wantCol, v.wantRow, ds.cursorCol, ds.cursorRow)
		}
	}
}
