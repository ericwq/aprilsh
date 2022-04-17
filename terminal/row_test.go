package terminal

import (
	"testing"
)

func TestSetWrap(t *testing.T) {
	tc := []bool{false, true, false, false, false}

	row := NewRow(len(tc), 40)

	for i := range row.cells {
		row.cells[i].SetWrap(tc[i])
	}
	if row.GetWrap() != tc[len(tc)-1] {
		t.Errorf("Last wrap: expect %t, got %t\n", tc[len(tc)-1], row.GetWrap())
	}
	row.DeleteCell(len(tc)-1, 0)
	if row.GetWrap() {
		t.Errorf("expect false, got %t\n", row.GetWrap())
	}
	row.SetWrap(true)
	if !row.GetWrap() {
		t.Errorf("expect ture, got %t\n", row.GetWrap())
	}

}

func TestInsertCell(t *testing.T) {
	width := 3
	tc := []struct {
		col     int
		bgColor uint32
	}{
		{-1, 40},
		{0, 41},
		{width - 2, 42},
		{width - 1, 43},
		{width, 44},
		{width + 1, 45},
	}
	for _, c := range tc {
		row := NewRow(width, c.bgColor)
		if row.InsertCell(c.col, 0) {
			cell := row.cells[c.col]
			if cell.GetRenditions().bgColor != 0 {
				t.Errorf("case %d: expect bgColor=0, got %v\n", c.col, cell.renditions)
			}
			// t.Logf("case %d,%v\n", c.col, row.cells)
		}
	}
}

func TestDeleteCell(t *testing.T) {
	width := 3
	tc := []struct {
		col     int
		bgColor uint32
	}{
		{-1, 40},
		{0, 41},
		{width - 2, 42},
		{width - 1, 43},
		{width, 44},
		{width + 1, 45},
	}
	for _, c := range tc {
		row := NewRow(width, c.bgColor)
		for i := range row.cells {
			row.cells[i].Append(rune(i + 0x41))
		}
		if row.DeleteCell(c.col, 0) {
			cell := row.cells[c.col]
			if cell.contents == string(rune(c.col+0x41)) {
				t.Errorf("case %d, %v\n", c.col, row.cells)
			}
		}
	}
}

func TestEqual(t *testing.T) {
	tc := []struct {
		width    int
		content  rune
		bgColor  uint32
		wide     bool
		fallback bool
		wrap     bool
	}{
		{3, '\x41', 41, false, true, false},
		{2, '\u4e16', 42, true, false, true},
	}

	// the simple case: same contents
	for _, c := range tc {
		row1 := NewRow(c.width, c.bgColor)
		for i := range row1.cells {
			row1.cells[i].Append(c.content)
			row1.cells[i].SetRenditions(Renditions{bgColor: c.bgColor})
			row1.cells[i].SetWide(c.wide)
			row1.cells[i].SetFallback(c.fallback)
			row1.cells[i].SetWrap(c.wrap)
		}
		row2 := NewRow(c.width, c.bgColor)
		for i := range row2.cells {
			row2.cells[i].Append(c.content)
			row2.cells[i].SetRenditions(Renditions{bgColor: c.bgColor})
			row2.cells[i].SetWide(c.wide)
			row2.cells[i].SetFallback(c.fallback)
			row2.cells[i].SetWrap(c.wrap)
		}
		if row1.Equal(row2) {
			t.Errorf("row.gen should be different: row1 %d, row2 %d\n", row1.gen, row2.gen)
		}
		row2.gen = row1.gen
		if !row1.Equal(row2) {
			t.Logf("row.width: row1=%d, row2=%d\n", len(row1.cells), len(row2.cells))
			t.Errorf("row.cells: row1=%v, row2=%v\n", row1.cells, row2.cells)
		}
	}

	// test rows with different width
	row1 := NewRow(3, 40)
	row2 := NewRow(4, 40)

	// force the gen equal
	row2.gen = row1.gen
	if row1.Equal(row2) { // compare different size row
		t.Errorf("row.width: row1=%d, row2=%d\n", len(row1.cells), len(row2.cells))
	}

	// test rows with different content
	for _, c := range tc {
		row1 = NewRow(c.width, c.bgColor)
		row1.Reset(0)
		row2 = NewRow(c.width, c.bgColor)
		row2.Reset(0)
		for i := range row1.cells {
			row2.cells[i].Append(c.content)
		}

		for i := range row1.cells {
			row1.cells[i].Append(c.content + 1) // use different content
		}
		row2.gen = row1.gen
		if row1.Equal(row2) {
			t.Logf("row.width: row1=%d, row2=%d\n", len(row1.cells), len(row2.cells))
			t.Errorf("row.cells: row1=%v, row2=%v\n", row1.cells, row2.cells)
		}
	}
}
