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
