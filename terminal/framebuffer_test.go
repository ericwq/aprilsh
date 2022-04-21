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
			row.cells[j].Append(rune(0x41 + i))
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
		{"in range", 0, 0, "A"},
		{"in range", 1, 1, "B"},
		{"in range", 2, 2, "C"},
		{"in range", 3, 3, "D"},
		{"in range", 4, 4, "E"},
		{"in range", 5, 5, "F"},
		{"in range", 6, 6, "G"},
		{"in range", 7, 7, "H"},
		{"out of range: col 1", 8, 8, "I"},
		{"out of range: col 2", 9, 9, "J"},
		{"out of range: row 1", 11, 9, "A"},
		{"out of range: row 2", -1, 9, "A"},
		{"out of range: both", -1, -9, "A"},
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
			t.Errorf("%s:\t expect %s, got %s\n", v.name, v.ch, cell.contents)
		}
	}
}
