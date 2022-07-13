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
)

func TestFramebufferNewFramebuffer(t *testing.T) {
	width := 80
	height := 40
	fb := NewFramebuffer(width, height)
	if fb.DS.GetWidth() != width || fb.DS.GetHeight() != height {
		t.Errorf("DS size expect %dx%d, got %dx%d\n", width, height, fb.DS.GetWidth(), fb.DS.GetHeight())
	}
	if len(fb.rows) != height {
		t.Errorf("rows expect %d, got %d\n", height, len(fb.rows))
	}

	fb = NewFramebuffer(-1, -2)
	if fb != nil {
		t.Errorf("new expect nil, got %v\n", fb)
	}
}

var startCh int = 0x41

// fill in rows with A,B,C....
func fillinRows(fb *Framebuffer, startCh ...int) {
	x := 0x41
	if len(startCh) > 0 {
		x = startCh[0]
	}

	rows := fb.GetRows()
	for i, row := range rows {
		for j := range row.cells {
			row.cells[j].Append(rune(x + i + j))
		}
	}
}

func printRows(fb *Framebuffer) string {
	var output strings.Builder
	for _, row := range fb.rows {
		output.WriteString(row.String() + "\n")
	}

	return output.String()
}

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
	if count ==0 {
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

func TestFramebufferInsertLine(t *testing.T) {
	tc := []struct {
		name      string
		beforeRow int
		count     int
		wantCount int
		want      bool
	}{
		{"in range", 2, 2, 2, true},
		{"top edge", 0, 3, 3, true},
		{"bottom edge", 9, 1, 1, true},
		{"bottom edge, extra count", 9, 3, 1, true},
		{"outof range, bottom", 10, 3, 1, false},
		{"outof range, top", -1, 3, 1, false},
		{"outof range, zero count", 1, 0, 1, false},
		{"outof range, negative count", 1, -1, 1, false},
	}

	width := 8
	height := 10

	for _, v := range tc {
		fb := NewFramebuffer(width, height)
		// fill the contents
		fillinRows(fb)

		// save the contents: before
		before := printRows(fb)

		if fb.InsertLine(v.beforeRow, v.count) {

			// save the contents: after
			after := printRows(fb)

			// count the blank row number
			count := strings.Count(printRows(fb), strings.Repeat(" ", width))
			if count == v.wantCount {
				continue
			} else {
				t.Logf("\nBefore Insert:\n%s", before)
				t.Logf("\nAfter  Insert:\n%s", after)
				t.Errorf("%s: expect %d, got %d\n", v.name, v.wantCount, count)
			}
		} else {
			// expect return is wrong
			if v.want == true {
				t.Errorf("%s: expect %t, got %t\n", v.name, v.want, false)
			}
		}
	}
}

func TestFramebufferDeleteLine(t *testing.T) {
	tc := []struct {
		name      string
		row       int
		count     int
		wantCount int
		want      bool
	}{
		{"in range", 2, 2, 2, true},
		{"top edge", 0, 3, 3, true},
		{"bottom edge", 9, 1, 1, true},
		{"bottom edge, extra count", 9, 3, 1, true},
		{"out of range, bottom", 10, 3, 1, false},
		{"out of range, top", -1, 3, 1, false},
		{"out of range, zero count", 1, 0, 1, false},
		{"out of range, negative count", 1, -1, 1, false},
	}

	width := 8
	height := 10

	for _, v := range tc {
		fb := NewFramebuffer(width, height)
		// fill the contents
		fillinRows(fb)

		// save the contents: before
		before := printRows(fb)
		after := ""
		count := 0

		if fb.DeleteLine(v.row, v.count) {

			// save the contents: after
			after = printRows(fb)

			// count the blank row number
			count = strings.Count(printRows(fb), strings.Repeat(" ", width))
			if count == v.wantCount {
				// t.Logf("\nBefore Delete:\n%s", before)
				// t.Logf("\nAfter  Delete:\n%s", after)
				// t.Errorf("%s: expect %d, got %d\n", v.name, v.wantCount, count)
				continue
			} else {
				t.Logf("\nBefore Delete:\n%s", before)
				t.Logf("\nAfter  Delete:\n%s", after)
				t.Errorf("%s: expect %d, got %d\n", v.name, v.wantCount, count)
			}
		} else {
			// expect return is wrong
			if v.want == true {
				t.Errorf("%s: expect %t, got %t\n", v.name, v.want, false)
			}
		}
		// t.Logf("\nBefore Delete:\n%s", before)
		// t.Logf("\nAfter  Delete:\n%s", after)
		// t.Errorf("%s: expect %d, got %d\n", v.name, v.wantCount, count)
	}
}

func TestFramebufferGetCell(t *testing.T) {
	tc := []struct {
		name string
		row  int
		col  int
		ch   string
	}{
		{"in range 0", 0, 0, "A"},
		{"in range 1", 1, 1, "C"},
		{"in range 2", 2, 2, "E"},
		{"in range 3", 3, 3, "G"},
		{"in range 4", 4, 4, "I"},
		{"in range 5", 5, 5, "K"},
		{"in range 6", 6, 6, "M"},
		{"in range 7", 7, 7, "O"},
		{"out of range: col 8", 8, 8, "I"},  // 0,8
		{"out of range: col 9", 10, 7, "H"}, // 0,7
		{"out of range: row 9", 11, 9, "A"}, // 0,0
		{"out of range: row 2", -1, 9, "A"}, // 0,0
		{"out of range: both", -1, -9, "A"}, // 0,0
	}

	width := 8
	height := 10

	for _, v := range tc {
		fb := NewFramebuffer(width, height)

		// fill the contents
		fillinRows(fb)

		cell := fb.GetCell(v.row, v.col)
		// cell:= fb.rows[v.row].cells[v.col]
		if cell.contents != v.ch {
			t.Logf("\n%s\n", printRows(fb))
			t.Errorf("%s:\t expect %s, got %s\n", v.name, v.ch, cell.contents)
		}
	}
}

// func TestFramebufferResize(t *testing.T) {
// 	tc := []struct {
// 		name   string
// 		width  int
// 		height int
// 	}{
// 		{"expand both", 12, 10},
// 		{"expand height", 8, 10},
// 		{"expand width", 12, 8},
// 		{"shrink both", 6, 6},
// 		{"shrink height", 8, 2},
// 		{"shrink width", 2, 8},
// 		{"invalid both", -1, -1},
// 	}
//
// 	// initial framebuffer size
// 	width := 8
// 	height := 8
//
// 	for _, v := range tc {
// 		fb := NewFramebuffer(width, height)
// 		// fill the contents
// 		fillinRows(fb)
//
// 		// save the contents: before
// 		before := printRows(fb)
//
// 		if !fb.Resize(v.width, v.height) {
// 			continue
// 		}
//
// 		after := printRows(fb)
//
// 		if len(fb.rows) != v.height || len(fb.rows[v.height-1].cells) != v.width {
// 			t.Logf("\nBefore Delete:\n%s", before)
// 			t.Logf("\nAfter  Delete:\n%s", after)
// 			t.Errorf("%s:\t expect (%d,%d)\n", v.name, v.width, v.height)
// 		}
// 	}
// }

func TestFramebufferIconNameWindowTitle(t *testing.T) {
	windowTitle := "aprilsh"
	iconName := "四姑娘山"
	fb := NewFramebuffer(1, 1)

	if fb.GetWindowTitle() != "" || fb.GetIconName() != "" || fb.IsTitleInitialized() {
		t.Logf("expect empty windowTitle, got %s\n", fb.GetWindowTitle())
		t.Logf("expect empty iconName , got %s\n", fb.GetIconName())
		t.Errorf("expect false titleInitialized, got %t\n", fb.IsTitleInitialized())
	}

	fb.SetWindowTitle(windowTitle)
	if fb.GetWindowTitle() != windowTitle {
		t.Errorf("expect windowTitle %s, got %s\n", windowTitle, fb.GetWindowTitle())
	}

	fb.SetIconName(iconName)
	if fb.GetIconName() != iconName {
		t.Errorf("expect iconName %s, got %s\n", iconName, fb.GetIconName())
	}

	fb.SetTitleInitialized()
	if !fb.IsTitleInitialized() {
		t.Errorf("expect true titleInitialized, got %t\n", fb.IsTitleInitialized())
	}

	tc := []struct {
		name    string
		windows string
		icon    string
		prefix  string
		want    string
	}{
		{"same value", "Monterey", "Monterey", "macOS ", "macOS Monterey"},
		{"diff value", "client", "server", "aprilsh ", "aprilsh client"},
		{"chinese codepoint", "姑娘山", "姑娘山", "四", "四姑娘山"},
	}

	for _, v := range tc {
		fb.Reset()

		fb.SetWindowTitle(v.windows)

		fb.SetIconName(v.icon)

		fb.PrefixWindowTitle(v.prefix)

		if v.windows == v.icon && fb.GetIconName() != v.want {
			t.Errorf("%s expect prefix+iconName=[%s], got [%s]\n ", v.name, v.want, fb.GetIconName())
		}

		if fb.GetWindowTitle() != v.want {
			t.Errorf("%s expect prefix+windowTitle=[%s], got [%s]\n", v.name, v.want, fb.GetWindowTitle())
		}
	}
}

func TestFramebufferMoveRowsAutoscroll(t *testing.T) {
	tc := []struct {
		name       string
		rows       int
		blankCount int
		wantRow    int
	}{
		{"no scroll, in range", 3, 0, 3},
		{"scroll, over bottom", 10, 3, 7},
		{"scroll, over top", -2, 2, 0},
		{"rows out of range", 3, 0, 2}, // 3+-1==2
	}

	width := 8
	height := 8

	for _, v := range tc {
		fb := NewFramebuffer(width, height)
		// fill the contents
		fillinRows(fb)

		// save the contents: before
		// t.Logf("Before cursor at row=%d\n", fb.DS.GetCursorRow())
		before := printRows(fb)

		// special case: rows out of range
		if v.name == "rows out of range" {
			fb.DS.cursorRow = -1 // 3+-1==2
		}

		fb.MoveRowsAutoscroll(v.rows)

		// save the contents: after
		// t.Logf("After cursor at row=%d\n", fb.DS.GetCursorRow())
		after := printRows(fb)

		// count the blank row number
		count := strings.Count(printRows(fb), strings.Repeat(" ", width))
		if count != v.blankCount {
			t.Logf("Before :\n%s", before)
			t.Logf("After:\n%s", after)
			t.Errorf("%s: expect blank row: %d, got %d\n", v.name, v.blankCount, count)
		}

		// validate the cursor row
		if fb.DS.GetCursorRow() != v.wantRow {
			t.Errorf("expect row at %d, got %d\n", v.wantRow, fb.DS.GetCursorRow())
		}
	}
}

func TestFramebufferInsertCell(t *testing.T) {
	tc := []struct {
		name string
		row  int
		col  int
		want bool
	}{
		{"in range 2", 2, 2, true},
		{"in range 2", 7, 7, true},
		{"out range 9", 9, 9, false},
		{"out range -1", -1, -1, false},
	}

	width := 8
	height := 8

	for _, v := range tc {
		fb := NewFramebuffer(width, height)
		// fill the contents
		fillinRows(fb)
		before := printRows(fb)

		if fb.InsertCell(v.row, v.col) != v.want {

			// save the contents: after
			after := printRows(fb)

			// count the blank row number
			count := strings.Count(printRows(fb), " ")
			if count != 1 {
				t.Logf("Before :\n%s", before)
				t.Logf("After:\n%s", after)
				t.Errorf("%s: expect blank cell: 1, got %d\n", v.name, count)
			}
		}
	}
}

func TestFramebufferDeleteCell(t *testing.T) {
	tc := []struct {
		name string
		row  int
		col  int
		want bool
	}{
		{"in range 2", 2, 2, true},
		{"in range 2", 7, 7, true},
		{"out range 9", 9, 9, false},
		{"out range -1", -1, -1, false},
	}

	width := 8
	height := 8

	for _, v := range tc {
		fb := NewFramebuffer(width, height)
		// fill the contents
		fillinRows(fb)
		before := printRows(fb)

		if fb.DeleteCell(v.row, v.col) != v.want {

			// save the contents: after
			after := printRows(fb)

			// count the blank row number
			count := strings.Count(printRows(fb), " ")
			if count != 1 {
				t.Logf("Before :\n%s", before)
				t.Logf("After:\n%s", after)
				t.Errorf("%s: expect blank cell: 1, got %d\n", v.name, count)
			}
		}
	}
}

func TestFramebufferGetCombiningCell(t *testing.T) {
	tc := []struct {
		name string
		row  int
		col  int
		want string
	}{
		{"in range", 4, 4, "I"},
		{"out of range 8 ", 8, 8, ""},
		{"out of range -1", -1, -1, ""},
	}

	width := 8
	height := 8

	fb := NewFramebuffer(width, height)

	// fill with content
	fillinRows(fb)
	for _, v := range tc {

		// move the cursor to position
		// fb.DS.MoveRow(v.row, false)
		// fb.DS.MoveCol(v.col, false, false)
		fb.DS.combiningCharRow = v.row
		fb.DS.combiningCharCol = v.col

		cell := fb.GetCombiningCell()
		if cell == nil && v.want == "" {
			// t.Logf("%s:\t position(%d,%d) is nil\n", v.name,v.row,v.col)
			continue
		} else if cell != nil && cell.contents == v.want {
			// t.Logf("%s:\t expect cell content=%s, got content=%s\n", v.name, v.want, cell.contents)
			continue
		} else {
			t.Logf("\n%s\n", printRows(fb))
			t.Errorf("%s:\t position(%d,%d) content=%v unknow error\n", v.name, v.row, v.col, cell)
		}
	}
}

func TestFramebufferApplyRenditionsToCell(t *testing.T) {
	tc := []struct {
		name           string
		row            int
		col            int
		dsRenditions   int
		wantRenditions int
	}{
		{"nil cell", 0, 0, 41, 41},
		{"normal cell", 2, 2, 42, 42},
	}
	width := 8
	height := 8

	fb := NewFramebuffer(width, height)

	// fill with content
	fillinRows(fb)
	for _, v := range tc {
		var cell *Cell

		// find the cell
		if v.name != "nil cell" {
			fb.DS.cursorRow = v.row
			fb.DS.cursorRow = v.col
		}

		// set the target rendition
		dr := Renditions{}
		dr.SetBackgroundColor(v.dsRenditions)
		fb.DS.renditions = dr // Renditions{bgColor: v.dsRenditions}

		fb.ApplyRenditionsToCell(cell)

		want := Renditions{}
		want.SetBackgroundColor(v.wantRenditions)
		if cell != nil {
			if cell.GetRenditions() != want {
				t.Errorf("%s:\tcell renditions: expect=%v, got %v\n", v.name, v.wantRenditions, cell.GetRenditions())
			}
		} else {
			if fb.GetCell(-1, -1).GetRenditions() != want {
				t.Logf("after action, cursor row=%d, cursor col=%d\n", fb.DS.cursorRow, fb.DS.cursorCol)
				t.Errorf("%s:\t cell is nil, expect %v, got %p\n", v.name, v.wantRenditions, fb.GetCell(-1, -1))
			}
		}
	}
}

func TestFramebufferResetRow(t *testing.T) {
	fb := NewFramebuffer(8, 8)

	row := fb.GetRow(-1) // cursor row , default 0

	// prepare different renditions
	for i := range row.cells {
		r := Renditions{}
		r.SetBackgroundColor(43)
		row.cells[i].renditions = r
	}

	fb.ResetRow(row)

	// validate the result
	want := Renditions{}
	for i := range row.cells {
		if row.cells[i].renditions != want {
			t.Errorf("expect %v, got %v\n", want, row.cells[i].renditions)
		}
	}
}

func TestFramebufferResetCell(t *testing.T) {
	fb := NewFramebuffer(8, 8)

	// prepare different cell and renditions
	cell := fb.GetCell(4, 4)
	r := Renditions{}
	r.SetBackgroundColor(43)
	cell.renditions = r

	fb.ResetCell(cell)

	// validate the result
	want := Renditions{}
	if cell.renditions != want {
		t.Errorf("expect %v, got %v\n", want, cell.renditions)
	}
}

func TestFramebufferRingBell(t *testing.T) {
	fb := NewFramebuffer(8, 8)

	if fb.GetBellCount() != 0 {
		t.Errorf("initial value should be 0, got %d\n", fb.GetBellCount())
	}

	count := 5
	for i := 0; i < count; i++ {
		fb.RingBell()
	}

	if fb.GetBellCount() != count {
		t.Errorf("initial value should be 0, got %d\n", fb.GetBellCount())
	}
}

func TestFramebufferEqual(t *testing.T) {
	type parameter struct {
		width       int
		height      int
		contents    int
		windowTitle string
		bellCount   int
	}
	tc := []struct {
		name string
		p1   parameter
		p2   parameter
		want bool
	}{
		{
			"all equal",
			parameter{8, 8, 0x41, "80 million", 2},
			parameter{8, 8, 0x41, "80 million", 2},
			true,
		},
		{
			"content not equal",
			parameter{8, 8, 0x42, "80 million", 2},
			parameter{8, 8, 0x41, "80 million", 2},
			false,
		},
		{
			"size not equal",
			parameter{8, 9, 0x42, "80 million", 2},
			parameter{9, 8, 0x42, "80 million", 2},
			false,
		},
		{
			"title not equal",
			parameter{4, 4, 0x42, "80 million", 2},
			parameter{4, 4, 0x42, "90 million", 2},
			false,
		},
		{
			"bell not equal",
			parameter{4, 4, 0x42, "80 million", 2},
			parameter{4, 4, 0x42, "90 million", 9},
			false,
		},
	}

	for _, v := range tc {
		fb1 := NewFramebuffer(v.p1.width, v.p1.height)
		fillinRows(fb1, v.p1.contents)
		fb1.SetWindowTitle(v.p1.windowTitle)
		fb1.bellCount = v.p1.bellCount
		// force gen to be the same
		if v.name == "all equal" {
			for i := range fb1.rows {
				fb1.rows[i].gen = uint64(i)
			}
		}
		fb2 := NewFramebuffer(v.p2.width, v.p2.height)
		fillinRows(fb2, v.p2.contents)
		fb2.SetWindowTitle(v.p2.windowTitle)
		fb2.bellCount = v.p2.bellCount
		// force gen to be the same
		if v.name == "all equal" {
			for i := range fb2.rows {
				fb2.rows[i].gen = uint64(i)
			}
		}

		if fb1.Equal(fb2) != v.want {
			t.Errorf("%s expect %t, got %t\n", v.name, v.want, fb1.Equal(fb2))
		}
	}
}

func TestFramebufferSoftReset(t *testing.T) {
	fb := NewFramebuffer(9, 10)

	fb.DS.InsertMode = true
	fb.DS.OriginMode = true
	fb.DS.CursorVisible = true
	fb.DS.ApplicationModeCursorKeys = true
	fb.DS.SetScrollingRegion(2, 8)
	rend := Renditions{}
	rend.SetBackgroundColor(44)
	fb.DS.renditions = rend // Renditions{bgColor: uint32(44)}

	fb.SoftReset()

	if fb.DS.InsertMode || fb.DS.OriginMode || fb.DS.CursorVisible || fb.DS.ApplicationModeCursorKeys {
		t.Errorf(
			"all 4 state should be false, got InsertMode=%t, OriginMode=%t, CursorVisible=%t, ApplicationModeCursorKeys=%t\n",
			fb.DS.InsertMode, fb.DS.OriginMode, fb.DS.CursorVisible, fb.DS.ApplicationModeCursorKeys)
	}

	if fb.DS.GetScrollingRegionTopRow() != 0 || fb.DS.GetScrollingRegionBottomRow() != fb.DS.GetHeight()-1 {
		t.Errorf(
			"scrolling Region should be 0-%d, got %d-%d\n", fb.DS.GetHeight()-1,
			fb.DS.scrollingRegionTopRow, fb.DS.scrollingRegionBottomRow)
	}

	expectRend := Renditions{}
	// expectRend.SetBackgroundColor(0)
	// r := Renditions{bgColor: uint32(0)}
	if fb.DS.renditions != expectRend {
		t.Errorf("renditions expect %v, got %v\n", expectRend, fb.DS.renditions)
	}
}

func TestFramebufferMoveInRow(t *testing.T) {
	tc := []struct {
		name               string
		row, startX, count int
		// result             string
	}{
		{
			"move in head", 1, 0, 4,
			// "    CDEFKLMNOPQRSTUVWXYZABCDEFGHIJKLMNOPQRSTUVWXYZABCDEFGHIJKLMNOPQRSTUVWXYZABCD\n",
		},
		{
			"move in mid", 1, 24, 5,
			// "CDEFGHIJKLMNOPQRSTUVWXYZ     ABCDEKLMNOPQRSTUVWXYZABCDEFGHIJKLMNOPQRSTUVWXYZABCD\n",
		},
		{
			"move in end", 1, 78, 4,
			// "CDEFGHIJKLMNOPQRSTUVWXYZABCDEFGHIJKLMNOPQRSTUVWXYZABCDEFGHIJKLMNOPQRSTUVWXYZAB  \n",
			// TODO this case will move the end to the next row.
		},
	}

	fb, _, _ := NewFramebuffer3(80, 40, 0)
	// prepare the damage area
	dmg := Damage{}
	dmg.totalCells = fb.damage.totalCells

	for _, v := range tc {
		// reset the conent
		fb.damage.reset()
		fillCells(fb, v.row)

		before := printCells(fb, v.row, v.row+1)

		cell := Cell{}
		cell.contents = " "
		fb.moveInRow(v.row, v.startX+v.count, v.startX, v.count)
		fb.eraseInRow(v.row, v.startX, v.count, cell)

		after := printCells(fb, v.row, v.row+1)

		// calculate the expected dmage area
		dmg.start, dmg.end = damageArea(fb, v.row, v.startX, v.row, v.startX+v.count*2)

		if fb.damage != dmg {
			t.Errorf("%q:\n", v.name)
			t.Errorf("expect damage %v, got %v\n", dmg, fb.damage)
			t.Errorf("[before row=%d, startX=%d, count=%d]\n%s", v.row, v.startX, v.count, before)
			t.Errorf("[after  row=%d, startX=%d, count=%d]\n%s", v.row, v.startX, v.count, after)
		}
	}
}

func TestFramebufferResize(t *testing.T) {
	tc := []struct {
		name                    string
		nCols, nRows, saveLines int
		newCols, newRows        int
	}{
		// {"saveLines over limitation  ", 7, 8, 7, 50001, 8, 7},
		{
			"expand both : expand 4 cols, 3 rows",
			8, 4, 4,
			12, 7,
		},
		{
			"expand rows : expand 3 rows",
			8, 4, 4,
			8, 7,
		},
		{
			"expand cols : expand 4 cols",
			8, 4, 4,
			12, 4,
		},
		{
			"resize none : keep the size",
			8, 4, 4,
			8, 4,
		},
		{
			"shrink both : shrink 4 cols, 3 rows",
			12, 7, 4,
			8, 4,
		},
		{
			"shrink rows : shrink 3 rows",
			12, 7, 4,
			12, 4,
		},
		{
			"shrink cols : shrink 4 cols",
			12, 7, 4,
			8, 7,
		},
	}

	for _, v := range tc {
		// if j == 0 {
		// 	gotSaveLines := fb.saveLines
		// 	if gotSaveLines != SaveLineUpperLimit {
		// 		t.Errorf("%s expect saveLines limitation %d, got %d\n", v.name, SaveLineUpperLimit, gotSaveLines)
		// 	}
		// 	continue
		// }
		// this is the pre-condidtion for the test case.
		emu := NewEmulator3(v.nCols, v.nRows, v.saveLines)
		var place strings.Builder
		emu.logI.SetOutput(&place)
		emu.logT.SetOutput(&place)

		fb := emu.cf
		fb.damage.reset()
		fillCells(fb)

		place.Reset()
		// sequence: move cursor to the last row on screen
		place.WriteString(fmt.Sprintf("\x1B[%d;%dH", v.nRows, v.nCols))
		// sequence: print line number: add history row and move scrollHead down
		// historyRows = saveLines
		for r := 0; r < v.saveLines; r++ {
			place.WriteString("\n") // fmt.Sprintf("line%d:\n", r+v.nRows))
		}
		// parse the sequence
		p := NewParser()
		hds := make([]*Handler, 0, 16)

		hds = p.processStream(place.String(), hds)
		place.Reset()

		// handle the sequence
		for _, hd := range hds {
			hd.handle(emu)
		}

		before := fmt.Sprintf("%s scrollHead=%d, marginTop=%d, marginBottom=%d\n",
			printCells(fb), fb.scrollHead, fb.marginTop, fb.marginBottom)

		fb.resize(v.newCols, v.newRows)

		after := printCells(fb)
		gotCols := fb.nCols
		gotRows := fb.nRows
		if gotCols != v.newCols || gotRows != v.newRows {
			t.Errorf("%s:\n", v.name)
			t.Errorf("[before resize rows=%d, cols=%d]\n%s", v.nRows, v.nCols, before)
			t.Errorf("[after resize  rows=%d, cols=%d]\n%s scrollHead=%d, marginTop=%d, marginBottom=%d\n",
				v.newRows, v.newCols, after, fb.scrollHead, fb.marginTop, fb.marginBottom)
		}
	}
}

func testFramebufferGetPhysicalRow(t *testing.T) {
	nCols := 8
	nRows := 8
	saveLines := 4
	top := 2

	fb, _, _ := NewFramebuffer3(nCols, nRows, saveLines)
	fb.setMargins(top, top+nRows-2)
	fb.scrollHead = 6
	/*
		tc := []struct {
			name string
			pY   int
			idx  int
		}{
			{"outof active area \t\t", 0, 0}, // out of active area
			{"inside active area\t\t", 4, 7},
			{"bottom active area\t\t", nRows - 2, nRows - 2},
			{"outside top active area\t", top - 1, top - 1},
			{"outside bottom active area\t", nRows, nRows},
			{"history row case -4\t", -4, nRows},
			{"history row case -3\t", -3, nRows},
			{"history row case -2\t", -2, nRows},
			{"history row case -1\t", -1, nRows},
		}
		for _, v := range tc {
			gotIdx := fb.getPhysicalRow(v.pY)
			// if gotIdx != v.idx {
			t.Errorf("%s for row %d->%d, expect idx=%d | scrollHead=%d, marginTop=%d, marginBottom=%d\n",
				v.name, v.pY, gotIdx, v.idx, fb.scrollHead, fb.marginTop, fb.marginBottom)
			// t.Errorf("scrollHead=%d, marginTop=%d, marginBottom=%d\n", )
			// }

		}
	*/
	for i := -saveLines; i < (nRows + saveLines); i++ {
		gotIdx := fb.getPhysicalRow(i)
		t.Errorf("row %d->%d | scrollHead=%d, marginTop=%d, marginBottom=%d\n",
			i, gotIdx, fb.scrollHead, fb.marginTop, fb.marginBottom)
	}
}
