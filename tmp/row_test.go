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
	"testing"
)

func TestRowSetWrap(t *testing.T) {
	tc := []bool{false, true, false, false, false}

	row := NewRow(len(tc), 40)

	// initialize the row with different wrap for each cell
	for i := range row.cells {
		row.cells[i].SetWrap(tc[i])
	}
	// verify the last one
	if row.GetWrap() != tc[len(tc)-1] {
		t.Errorf("Last wrap: expect %t, got %t\n", tc[len(tc)-1], row.GetWrap())
	}
	// after delete the last one, verify the (new) last one
	row.DeleteCell(len(tc)-1, 0)
	if row.GetWrap() {
		t.Errorf("expect false, got %t\n", row.GetWrap())
	}
	// change the wrap for the (new) last one, and verify it
	row.SetWrap(true)
	if !row.GetWrap() {
		t.Errorf("expect ture, got %t\n", row.GetWrap())
	}
}

func TestRowString(t *testing.T) {
	width := 10
	bgColor := ColorDefault
	row := NewRow(width, bgColor)

	// filled the cell
	for i := range row.cells {
		row.cells[i].Append(rune(0x41 + i))
	}

	// gen will changed dynamiclly
	str := row.String()
	gen := fmt.Sprintf("[%2d]", row.gen)
	want := "Row" + gen + "{ABCDEFGHIJ}"

	if str != want {
		t.Errorf("expect %s, got %v\n", want, str)
	}
}

func TestRowAt(t *testing.T) {
	width := 40
	bgColor := ColorDefault

	tc := []struct {
		name string
		col  int
		want *Cell
	}{
		{"col -1", -1, nil},
		{"col 0", 0, &Cell{renditions: Renditions{}}},
		{"col 1", 1, &Cell{renditions: Renditions{}}},
		{"col w-1", width - 1, &Cell{renditions: Renditions{}}},
		{"col w", width, nil},
		{"col w+1", width + 1, nil},
	}

	row := NewRow(width, bgColor)
	for _, v := range tc {
		c1 := row.At(v.col)
		if v.want == nil && c1 == nil {
			// t.Logf("both nil, %v %v\n", v.want, row.At(v.col))
			continue
		} else if v.want != nil && c1 != nil && *c1 == *(v.want) {
			// t.Logf("%s:REAL\t expect %v, got %v\n", v.name, v.want, row.At(v.col))
			continue
		} else {
			t.Errorf("%s:\t expect %v, got %v\n", v.name, v.want, row.At(v.col))
		}
	}

	c1 := row.At(8)
	c2 := row.At(8)
	if c1 != c2 {
		t.Errorf("expect %p, got %p\n", c1, c2)
		t.Errorf("expect %v, got %v\n", c1, c2)
	}
}

func TestRowInsertCell(t *testing.T) {
	width := 3
	tc := []struct {
		col     int
		bgColor Color
	}{
		{-1, ColorGray},
		{0, ColorRed},
		{width - 2, ColorLime},
		{width - 1, ColorYellow},
		{width, ColorBlue},
		{width + 1, ColorFuchsia},
	}
	for _, c := range tc {
		row := NewRow(width, c.bgColor)

		// insert cell according to the test case col position
		if row.InsertCell(c.col, 0) {
			cell := row.cells[c.col]

			// the new cell has different bgColor
			if cell.GetRenditions().bgColor != 0 {
				t.Errorf("case %d: expect bgColor=0, got %v\n", c.col, cell.renditions)
			}
			// t.Logf("case %d,%v\n", c.col, row.cells)
		} // for our of range case, InsertCell should return false.
	}
}

func TestRowDeleteCell(t *testing.T) {
	width := 3
	tc := []struct {
		col     int
		bgColor Color
	}{
		{-1, ColorDefault},
		{0, ColorRed},
		{width - 2, ColorGreen},
		{width - 1, ColorMaroon},
		{width, ColorOlive},
		{width + 1, ColorNavy},
	}
	for _, c := range tc {
		row := NewRow(width, c.bgColor)

		// fill each cell with different grapheme
		for i := range row.cells {
			row.cells[i].Append(rune(i + 0x41))
		}

		// delete cell in different position defined by test case
		if row.DeleteCell(c.col, 0) {
			cell := row.cells[c.col]

			// the deleted cell has different grapheme
			if cell.contents == string(rune(c.col+0x41)) {
				t.Errorf("case %d, %v\n", c.col, row.cells)
			}
		} // for out of range case, return false
	}
}

func TestRowEqual(t *testing.T) {
	tc := []struct {
		width    int
		content  rune
		bgColor  Color
		wide     bool
		fallback bool
		wrap     bool
	}{
		{3, '\x41', ColorDefault, false, true, false},
		{2, '\u4e16', ColorGray, true, false, true},
	}

	// the simple case: same width, same contents
	for _, c := range tc {
		row1 := NewRow(c.width, c.bgColor)
		for i := range row1.cells {
			row1.cells[i].Append(c.content)
			rend := Renditions{}
			// rend.SetBackgroundColor(c.bgColor)
			rend.setAnsiBackground(c.bgColor)
			// row1.cells[i].SetRenditions(Renditions{bgColor: c.bgColor})
			row1.cells[i].SetRenditions(rend)
			row1.cells[i].SetWide(c.wide)
			row1.cells[i].SetFallback(c.fallback)
			row1.cells[i].SetWrap(c.wrap)
		}
		row2 := NewRow(c.width, c.bgColor)
		for i := range row2.cells {
			row2.cells[i].Append(c.content)
			rend := Renditions{}
			// rend.SetBackgroundColor(c.bgColor)
			rend.setAnsiBackground(c.bgColor)
			// row2.cells[i].SetRenditions(Renditions{bgColor: c.bgColor})
			row2.cells[i].SetRenditions(rend)
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

	// two rows with different size
	row1 := NewRow(3, 40)
	row2 := NewRow(4, 40)

	// force the gen equal
	row2.gen = row1.gen
	if row1.Equal(row2) { // compare different size row
		t.Errorf("row.width: row1=%d, row2=%d\n", len(row1.cells), len(row2.cells))
	}

	// two rows with different grapheme
	for _, c := range tc {
		row1 = NewRow(c.width, c.bgColor)
		row1.Reset(0)
		row2 = NewRow(c.width, c.bgColor)
		row2.Reset(0)
		for i := range row1.cells {
			row2.cells[i].Append(c.content)
		}

		for i := range row1.cells {
			row1.cells[i].Append(c.content + 1) // use different grapheme
		}

		// for the gen equal
		row2.gen = row1.gen
		if row1.Equal(row2) { // compare different grapheme
			t.Logf("row.width: row1=%d, row2=%d\n", len(row1.cells), len(row2.cells))
			t.Errorf("row.cells: row1=%v, row2=%v\n", row1.cells, row2.cells)
		}
	}
}
