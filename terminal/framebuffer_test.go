package terminal

import (
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

// fill in rows with A,B,C....
func fillinRows(fb *Framebuffer) {
	rows := fb.GetRows()
	for i, row := range rows {
		for j := range row.cells {
			row.cells[j].Append(rune(0x41 + i + j))
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

func TestFramebufferResize(t *testing.T) {
	tc := []struct {
		name   string
		width  int
		height int
	}{
		{"expand both", 12, 10},
		{"expand height", 8, 10},
		{"expand width", 12, 8},
		{"shrink both", 6, 6},
		{"shrink height", 8, 2},
		{"shrink width", 2, 8},
		{"invalid both", -1, -1},
	}

	// initial framebuffer size
	width := 8
	height := 8

	for _, v := range tc {
		fb := NewFramebuffer(width, height)
		// fill the contents
		fillinRows(fb)

		// save the contents: before
		before := printRows(fb)

		if !fb.Resize(v.width, v.height) {
			continue
		}

		after := printRows(fb)

		if len(fb.rows) != v.height || len(fb.rows[v.height-1].cells) != v.width {
			t.Logf("\nBefore Delete:\n%s", before)
			t.Logf("\nAfter  Delete:\n%s", after)
			t.Errorf("%s:\t expect (%d,%d)\n", v.name, v.width, v.height)
		}
	}
}

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
		dsRenditions   uint32
		wantRenditions Renditions
	}{
		{"nil cell", 0, 0, 41, Renditions{bgColor: uint32(41)}},
		//{"normal cell", 2, 2, 42, Renditions{bgColor: uint32(42)}},
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
		fb.DS.renditions = Renditions{bgColor: v.dsRenditions}

		fb.ApplyRenditionsToCell(cell)

		if cell != nil {
			if cell.GetRenditions() != v.wantRenditions {
				t.Errorf("%s:\tcell renditions: expect=%v, got %v\n", v.name, v.wantRenditions, cell.GetRenditions())
			}
		} else {
			if fb.GetCell(-1, -1).GetRenditions() != v.wantRenditions {
				t.Logf("after action, cursor row=%d, cursor col=%d\n", fb.DS.cursorRow, fb.DS.cursorCol)
				t.Errorf("%s:\t cell is nil, expect %v, got %p\n", v.name, v.wantRenditions, fb.GetCell(-1, -1))
			}
		}
	}
}
