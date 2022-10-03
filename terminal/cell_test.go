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

	"github.com/rivo/uniseg"
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
		if cell.full() != c.want {
			t.Errorf("case:%s[len=%d] expected %t, got %t\n", cell.contents, len(cell.contents), c.want, cell.full())
		}
	}
}

func TestCellComparable(t *testing.T) {
	tc := []struct {
		contents   rune
		renditions Renditions
	}{
		{'A', NewRenditions(0)},
		{'b', NewRenditions(40)},
		{'\x7f', NewRenditions(41)},
		{'\u4e16', NewRenditions(42)},
		{'\u754c', NewRenditions(43)},
	}

	var c1, c2 Cell
	var base Cell

	// compare same contents and renditions
	for _, c := range tc {
		c1.Reset2(base)
		c2.Reset2(base)

		c1.Append(c.contents)
		c1.SetRenditions(c.renditions)

		c2.Append(c.contents)
		c2.SetRenditions(c1.GetRenditions())
		if c1 != c2 {
			t.Errorf("case %c c1=%v c2=%v\n", c.contents, c1, c2)
		}
	}
}

func TestCellCompare(t *testing.T) {
	tc := []struct {
		ch0          rune
		ansiBgColor0 int
		ch1          rune
		ansiBgColor1 int
		ret          bool
	}{
		{'a', 0, 'b', 0, false},
		{'i', 0, 'i', 0, true},
		{'c', 1, 'c', 1, true},
		{'中', 8, '中', 8, true},
		{'j', 0, 'j', 0, true},
		{'h', 0, 'h', 0, true},
		{'国', 3, '国', 0, false},
		{'e', 0, 'e', 7, false},
	}

	var cell0, cell1 Cell
	var base Cell

	// compare different contents and rendtions.
	for _, c := range tc {
		cell0.Reset2(base)
		cell1.Reset2(base)

		cell0.Append(c.ch0)
		r0 := Renditions{}
		r0.SetBackgroundColor(c.ansiBgColor0)
		cell0.SetRenditions(r0)

		cell1.Append(c.ch1)
		r1 := Renditions{}
		r1.SetBackgroundColor(c.ansiBgColor1)
		cell1.SetRenditions(r1)

		got := cell0 == cell1 // check compare result
		if got != c.ret {
			t.Errorf("expect %q, got %q\n", cell0, cell1)
		}
	}
}

func TestContentMatch(t *testing.T) {
	tc := []struct {
		name   string
		r0, r1 string
		result bool
	}{
		{"english", "A", "A", true},
		{"chinese", "长", "长", true},
		{"empty content", "", "", true},
		{"space content", " ", " ", true},
		{"special", "\xC2\xA0", "\xC2\xA0", true},
	}

	var c0, c1 Cell
	var base Cell

	// compare different contents
	for _, v := range tc {
		c0.Reset2(base)
		c1.Reset2(base)

		c0.contents = v.r0
		c1.contents = v.r1

		// validate ContentsMatch
		got := c0.ContentsMatch(c1)
		if got != v.result {
			t.Errorf("%q c0=%q, c1=%q\n", v.name, c0, c1)
		}

		// validate GetContents
		if c1.GetContents() != v.r1 {
			t.Errorf("%q c1=%q, r1=%q\n", v.name, c1.contents, v.r1)
		}
	}
}

func TestGetWidth(t *testing.T) {
	tc := []struct {
		name     string
		contents string
		width    int
	}{
		{"english", "A", 1},
		{"chinese", "中", 2},
		{"combing char", "n\u0308\u0308", 1},
		{"emojo", "💐", 2},
		{"emojo flag", "🇮🇹", 2},
	}

	var c, base Cell
	for _, v := range tc {
		c.Reset2(base)

		c.contents = v.contents
		c.SetDoubleWidth(uniseg.StringWidth(v.contents) == 2)

		// validate contents length
		got := c.GetWidth()
		if got != v.width {
			t.Errorf("%q expect width %d, got %d\n", v.name, v.width, got)
		}

		// validate dwidthCont case
		c.SetDoubleWidthCont(true)
		got = c.GetWidth()
		if got != 0 {
			t.Errorf("%q expect dwidthCont width %d, got %d\n", v.name, 0, got)
		}
	}
}

func TestDoubleWidth(t *testing.T) {
	var c Cell

	c.contents = "长"
	c.SetDoubleWidth(true)
	if !c.IsDoubleWidth() {
		t.Errorf("#test IsDoubleWidth() expect %t, got %t\n ", true, c.IsDoubleWidth())
	}

	if c.IsDoubleWidthCont() {
		t.Errorf("#test IsDoubleWidthCont() expect %t, got %t\n ", false, c.IsDoubleWidth())
	}
}

func TestString(t *testing.T) {
	var c Cell
	str := "江"
	c.SetContents([]rune(str))
	got := c.String()
	if got != str {
		t.Errorf("#test String() expect %q, got %q\n", str, got)
	}
}

func TestSetUnderline(t *testing.T) {
	var c Cell
	str := "Z"
	c.SetContents([]rune(str))
	c.SetUnderline(true)

	rend := c.GetRenditions()
	got := rend.underline
	if !got {
		t.Errorf("#test SetUnderline() expect %t, got %t\n", true, got)
	}
}

/*
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
		// cell.SetWide(c.wide)
		cell.PrintGrapheme(&output)
		if c.want != output.String() {
			t.Errorf("expect %s, got %s\n", c.want, output.String())
		}
		// if c.wide != cell.GetWide() {
		// 	t.Errorf("case: %s wide: expect %t, got %t\n", output.String(), c.wide, cell.GetWide())
		// }
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
*/

/*
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
	var base Cell
	for _, c := range tc {
		cell.Reset2(base)
		var output strings.Builder

		if c.ch != -1 {
			cell.Append(c.ch)
		}
		// cell.SetFallback(c.fallback)

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
*/
