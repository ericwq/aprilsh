package terminal

import (
	"strings"
	"testing"
)

func TestAppend(t *testing.T) {
	tc := []struct {
		r    rune
		want string
	}{
		{'\x41', "A"},
		{'\x4f', "O"},
		{'\u4e16', "世"},
		{'\u754c', "界"},
	}

	var output strings.Builder
	for _, c := range tc {
		var cell Cell
		output.Reset()
		cell.Append(c.r)
		cell.PrintGrapheme(&output)
		if c.want != output.String() {
			t.Errorf("expect %s, got %s\n", c.want, output.String())
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
}
