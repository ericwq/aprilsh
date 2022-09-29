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
)

var gen_counter uint64 = 0

type Row struct {
	cells []Cell
	// gen is a generation counter.  It can be used to quickly rule
	// out the possibility of two rows being identical; this is useful
	// in scrolling.
	gen uint64
}

func getGen() uint64 {
	gen_counter += 1
	return gen_counter
}

// TODO consider using DS rendition to create new row.
func NewRow(width int, bgColor Color) *Row {
	r := Row{}
	r.cells = make([]Cell, width)
	for i := range r.cells {
		rend := Renditions{}
		// rend.SetBackgroundColor(bgColor)
		rend.setAnsiBackground(bgColor)
		r.cells[i].SetRenditions(rend)
		// fmt.Printf("NeRow: set cell %v %d\n", c.GetRenditions(), bgColor)
	}
	r.gen = getGen()
	// fmt.Printf("NewRow: %v\n", r.cells)
	return &r
}

// return cell specified by col
func (r *Row) At(col int) *Cell {
	if col < 0 || col > len(r.cells)-1 {
		return nil
	}

	// return the pointer of slice element directly
	return &(r.cells[col])
}

func (r *Row) InsertCell(col int, bgColor uint32) bool {
	// validate the column range
	if col < 0 || col > len(r.cells)-1 {
		return false
	}

	// prepare the new cell
	cell := Cell{}
	cell.renditions = Renditions{}

	// insert cell
	r.cells = append(r.cells[:col+1], r.cells[col:]...)
	r.cells[col] = cell

	// pop the last one
	width := len(r.cells) - 1
	r.cells = r.cells[:width]
	return true
}

func (r *Row) DeleteCell(col int, bgColor uint32) bool {
	if col < 0 || col > len(r.cells)-1 {
		return false
	}

	// prepare the new cell
	cell := Cell{}
	cell.renditions = Renditions{}

	// add new cell at the end
	r.cells = append(r.cells, cell)

	// delete cell at col
	copy(r.cells[col:], r.cells[col+1:])

	// remvoe the last one
	width := len(r.cells) - 1
	r.cells = r.cells[:width]
	return true
}

func (r *Row) Reset(bgColor uint32) {
	r.gen = getGen()
	for i := range r.cells {
		r.cells[i].Reset(bgColor)
	}
}

func (r Row) GetWrap() bool {
	return r.cells[len(r.cells)-1].GetWrap()
}

func (r *Row) SetWrap(w bool) {
	r.cells[len(r.cells)-1].SetWrap(w)
}

func (r Row) Equal(other *Row) bool {
	// the easy way to compare
	if r.gen != other.gen {
		return false
	}

	// has different size?
	if len(r.cells) != len(other.cells) {
		return false
	}

	// check the content
	for i := range r.cells {
		if r.cells[i] != other.cells[i] {
			return false
		}
	}
	return true
}

func (r Row) String() string {
	var builder strings.Builder

	builder.WriteString("Row")

	fmt.Fprintf(&builder, "[%3d]{", r.gen)

	skipNext := false // skipNext will jump over the next cell for wide cell
	for _, v := range r.cells {
		if skipNext {
			skipNext = false
			continue
		}
		if v.wide {
			skipNext = true
		}
		v.PrintGrapheme(&builder)
	}
	fmt.Fprintf(&builder, "}")

	return builder.String()
}
