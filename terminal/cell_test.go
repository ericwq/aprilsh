package terminal

import (
	"os"
	"strings"
	"testing"
)

// see https://godoc.org/golang.org/x/text/width
// see http://github.com/mattn/go-runewidth
func TestFull(t *testing.T) {
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

func TestAppend(t *testing.T) {
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

func TestIsPrintISO8859_1(t *testing.T) {
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

func TestCompare(t *testing.T) {
	tc := []struct {
		ch0         rune
		renditions0 uint32
		wide0       bool
		fallback0   bool
		wrap0       bool
		ch2         rune
		renditions2 uint32
		wide2       bool
		fallback2   bool
		wrap2       bool
		want        string
		ret         bool
	}{
		{'a', 30, true, false, false, 'b', 30, true, false, false, "Graphemes:", true},
		{'i', 30, true, false, false, 'i', 30, true, false, false, "", false},
		{'c', 30, true, true, false, 'c', 30, true, true, false, "", false},
		{'g', 30, true, false, false, 'g', 30, true, false, false, "", false},
		{'j', 30, true, true, false, 'j', 30, true, false, false, "Graphemes:", true},
		{'h', 30, true, false, false, 'h', 30, true, true, false, "Graphemes:", true},
		{'d', 30, true, false, false, 'd', 30, false, false, false, "width: ", true},
		{'e', 30, true, false, false, 'e', 37, true, false, false, "renditions differ", true},
		{'f', 30, true, false, false, 'f', 30, true, false, true, "wrap: ", true},
	}
	var cell0, cell2 Cell

	o := new(strings.Builder)
	_output = o

	for _, c := range tc {
		o.Reset()
		cell0.Reset(0)
		cell2.Reset(0)
		cell0.Append(c.ch0) // prepare cell0
		cell0.SetRenditions(Renditions{bgColor: c.renditions0})
		cell0.SetWide(c.wide0)
		cell0.SetFallback(c.fallback0)
		cell0.SetWrap(c.wrap0)
		cell2.Append(c.ch2) // prepare cell2
		cell2.SetRenditions(Renditions{bgColor: c.renditions2})
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

func TestPrintGrapheme(t *testing.T) {
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
