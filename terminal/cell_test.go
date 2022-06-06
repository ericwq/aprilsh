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
	"os"
	"strings"
	"testing"
)

// see https://godoc.org/golang.org/x/text/width
// see http://github.com/mattn/go-runewidth
func TestCellFull(t *testing.T) {
	tc := []struct {
		base     rune
		addition rune
		repeat   int
		want     bool
	}{
		{'a', 'a', 13, false},
		{'b', '\u0304', 15, false},
		{'c', '\u0305', 16, true},
	}
	var cell Cell
	for _, c := range tc {
		cell.Clear()
		cell.Append(c.base)
		for i := 0; i < c.repeat; i++ {
			cell.Append(c.addition)
		}
		if cell.Full() != c.want {
			t.Errorf("case:%s[len=%d] expected %t, got %t\n", cell.contents, len(cell.contents), c.want, cell.Full())
		}
	}
}

func TestCellComparable(t *testing.T) {
	tc := []struct {
		contents   rune
		renditions Renditions
		wide       bool
		fallback   bool
		wrap       bool
	}{
		{'A', Renditions{bgColor: 0}, false, true, false},
		{'b', Renditions{bgColor: 40}, false, true, false},
		{'\x7f', Renditions{bgColor: 41}, false, true, false},
		{'\u4e16', Renditions{bgColor: 42}, true, true, false},
		{'\u754c', Renditions{bgColor: 43}, true, true, true},
	}
	var c1, c2 Cell
	for _, c := range tc {
		c1.Reset(0)
		c2.Reset(0)

		c1.Append(c.contents)
		c1.SetRenditions(c.renditions)
		c1.SetWide(c.wide)
		c1.SetFallback(c.fallback)
		c1.SetWrap(c.wrap)

		c2.Append(c.contents)
		c2.SetRenditions(c1.GetRenditions())
		c2.SetWide(c1.GetWide())
		c2.SetFallback(c1.GetFallback())
		c2.SetWrap(c1.GetWrap())
		if c1 != c2 {
			t.Errorf("case %c c1=%v c2=%v\n", c.contents, c1, c2)
		}
	}
}

func TestCellAppend(t *testing.T) {
	tc := []struct {
		r     rune
		wide  bool
		want  string
		width int
	}{
		{'\x41', false, "A", 1},
		{'\x4f', false, "O", 1},
		{'\u4e16', true, "世", 2},
		{'\u754c', true, "界", 2},
	}

	var output strings.Builder
	for _, c := range tc {
		var cell Cell
		output.Reset()
		cell.Append(c.r)
		cell.SetWide(c.wide)
		cell.PrintGrapheme(&output)
		if c.want != output.String() {
			t.Errorf("expect %s, got %s\n", c.want, output.String())
		}
		if c.wide != cell.GetWide() {
			t.Errorf("case: %s wide: expect %t, got %t\n", output.String(), c.wide, cell.GetWide())
		}
		if c.width != int(cell.GetWidth()) {
			t.Errorf("case: %s width: expect %d, got %d\n", output.String(), c.width, cell.GetWidth())
		}

		output.Reset()
		AppendToStr(&output, c.r)
		if c.want != output.String() {
			t.Errorf("expect %s, got %s\n", c.want, output.String())
		}
	}
}

func TestCellIsPrintISO8859_1(t *testing.T) {
	tc := []struct {
		r rune
		b bool
	}{
		{'a', true},
		{'#', true},
		{'0', true},
		{'\x20', true},
		{'\x7e', true},
		{'\xa0', true},
		{'\xff', true},
		{'\u4e16', false},
	}

	for _, c := range tc {
		d := IsPrintISO8859_1(c.r)
		if d != c.b {
			t.Errorf("for %c expect %t, got %t\n", c.r, c.b, d)
		}
	}
}

func TestCellCompare(t *testing.T) {
	tc := []struct {
		ch0         rune
		renditions0 int
		wide0       bool
		fallback0   bool
		wrap0       bool
		ch2         rune
		renditions2 int
		wide2       bool
		fallback2   bool
		wrap2       bool
		want        string
		ret         bool
	}{
		{'a', 0, true, false, false, 'b', 0, true, false, false, "Graphemes:", true},
		{'i', 0, true, false, false, 'i', 0, true, false, false, "", false},
		{'c', 0, true, true, false, 'c', 0, true, true, false, "", false},
		{'g', 0, true, false, false, 'g', 0, true, false, false, "", false},
		{'j', 0, true, true, false, 'j', 0, true, false, false, "Graphemes:", true},
		{'h', 0, true, false, false, 'h', 0, true, true, false, "Graphemes:", true},
		{'d', 0, true, false, false, 'd', 0, false, false, false, "width: ", true},
		{'e', 0, true, false, false, 'e', 7, true, false, false, "renditions differ", true},
		{'f', 0, true, false, false, 'f', 0, true, false, true, "wrap: ", true},
	}
	var cell0, cell2 Cell

	o := new(strings.Builder)
	_output = o

	for _, c := range tc {
		o.Reset()
		cell0.Reset(0)
		cell2.Reset(0)
		cell0.Append(c.ch0) // prepare cell0
		r0 := NewRenditions()
		r0.SetBackgroundColor(c.renditions0)
		cell0.SetRenditions(*r0) // Renditions{bgColor: c.renditions0})
		cell0.SetWide(c.wide0)
		cell0.SetFallback(c.fallback0)
		cell0.SetWrap(c.wrap0)
		cell2.Append(c.ch2) // prepare cell2
		r2 := NewRenditions()
		r2.SetBackgroundColor(c.renditions2)
		cell2.SetRenditions(*r2) // Renditions{bgColor: c.renditions2})
		cell2.SetWide(c.wide2)
		cell2.SetFallback(c.fallback2)
		cell2.SetWrap(c.wrap2)
		got := cell0.Compare(cell2) // check compare result
		if got != c.ret {
			t.Logf("[%s]\n", o.String())
			t.Errorf("expect %t, got %t\n", c.ret, got)
		}
		if len(c.want) > 0 && !strings.Contains(o.String(), c.want) {
			t.Logf("cell0={%s}\n", cell0.debugContents())
			t.Logf("cell2={%s}\n", cell2.debugContents())
			t.Errorf("expect '%s', got '%s'\n", c.want, o.String())
		}

	}
	_output = os.Stderr
}

func TestCellPrintGrapheme(t *testing.T) {
	tc := []struct {
		ch       rune
		fallback bool
		want     string
	}{
		{-1, true, " "},
		{'a', false, "a"},
		{'b', true, "\xC2\xA0b"},
	}
	var cell Cell
	for _, c := range tc {
		cell.Reset(0)
		var output strings.Builder

		if c.ch != -1 {
			cell.Append(c.ch)
		}
		cell.SetFallback(c.fallback)

		cell.PrintGrapheme(&output)
		if output.String() != c.want {
			t.Errorf("expect [%s], got [%s]\n", c.want, output.String())
		}
	}
}

func TestCelldebugContents(t *testing.T) {
	tc := []struct {
		name string
		ch   rune
		want string
	}{
		{"empty", -1, "'_' []"},
		{"space", '\x20', "' ' [0x20, ]"},
		{"scope", '\x7e', "'~' [0x7e, ]"},
		{"scope", '\xa0', "' ' [0xc2, , 0xa0, ]"},
		{"scope", '\xff', "'ÿ' [0xc3, , 0xbf, ]"},
		{"chinese", '\u4e16', "'世' [0xe4, , 0xb8, , 0x96, ]"},
	}

	for _, v := range tc {
		cell := Cell{}
		if v.ch != -1 {
			cell.Append(v.ch)
		}
		if v.want != cell.debugContents() {
			t.Errorf("%s:\t expect [%s], got [%s]", v.name, v.want, cell.debugContents())
		}
	}
}
