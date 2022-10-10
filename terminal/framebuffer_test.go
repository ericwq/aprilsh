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

func TestPageUpDownBottom(t *testing.T) {
	// fill the framebuffer with 3 different content,scroll the active area.
	fb, _, _ := NewFramebuffer3(80, 40, 80)
	base := Cell{}
	r := []rune{'x', 'y', 'z'}
	for i := 0; i < 3; i++ {
		fb.fillCells(r[i], base)
		if i != 2 { // move scrollHead to row 80
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
		{"from  0 to 20", 20, "y", 20, 0},
		{"from 20 to 40", 20, "y", 40, 0},
		{"from 40 to 45", 5, "x", 45, 0},
		{"from 44 to 80", 35, "x", 80, 0},
		{"from 80 to 70", 10, "x", 70, 1},
		{"from 70 to 40", 30, "y", 40, 1},
		{"from 40 to 10", 30, "y", 10, 1},
		{"page to bottom", 0, "z", 0, 2},
		{"page to bottom again", 0, "z", 0, 2},
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

		// fmt.Printf("scrollHead=%d, viewOffset=%d, historyRow=%d\n",
		// 	fb.scrollHead, fb.viewOffset, fb.historyRows)

		if fb.viewOffset != v.expectViewOffset {
			t.Errorf("%q expect viewOffset %d, got %d\n", v.name, v.expectViewOffset, fb.viewOffset)
		}

		// validate the cell content with different viewOffset
		got := fb.cells[fb.getViewRowIndex(0)].contents
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
