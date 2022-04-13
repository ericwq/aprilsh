package terminal

import (
	"strings"
	"testing"
)

const reset = "\033[0m"

func TestSGR_RGBColor(t *testing.T) {
	tc := []struct {
		fr,fg,fb   uint32
		br,bg,bb   uint32
		attr uint32
		want string
	}{
		{33, 47,12, 123,24,34, Bold, "\033[0;1;38;2;33;47;12;48;2;123;24;34m"},
		{0, 0,0,0,0,0, Italic, "\033[0;3;38;2;0;0;0;48;2;0;0;0m"},
		{12,34,128, 59,190,155, Underlined, "\033[0;4;38;2;12;34;128;48;2;59;190;155m"},
	}

	for _, c := range tc {
		r := Renditions{}
		r.SetFgColor(c.fr, c.fg, c.fb)
		r.SetBgColor(c.br, c.bg, c.bb)
		if r.GetAttributes(c.attr) {
			t.Errorf("expect %t, got false", r.GetAttributes(c.attr))
		}
		r.SetAttributes(c.attr, true)
		got := r.SGR()
		if c.want != got {
			a := strings.ReplaceAll(c.want, "\033", "ESC")
			b := strings.ReplaceAll(got, "\033", "ESC")
			t.Logf("expect %s, got %s\n", a, b)

			t.Errorf("expect %sThis%s, got %sThis%s\n", c.want, reset, got, reset)
		}
	}
}

func TestSGR_256color(t *testing.T) {
	tc := []struct {
		fg   uint32
		bg   uint32
		attr uint32
		want string
	}{
		{33, 47, Bold, "\033[0;1;38;5;33;48;5;47m"},
		{0, 0, Italic, "\033[0;3;30;40m"},
		{128, 155, Underlined, "\033[0;4;38;5;128;48;5;155m"},
		{205, 228, Inverse, "\033[0;7;38;5;205;48;5;228m"},
	}

	for _, c := range tc {
		r := Renditions{}
		r.SetForegroundColor(c.fg)
		r.SetBackgroundColor(c.bg)
		r.SetAttributes(c.attr, true)
		got := r.SGR()
		if c.want != got {
			a := strings.ReplaceAll(c.want, "\033", "ESC")
			b := strings.ReplaceAll(got, "\033", "ESC")
			t.Logf("expect %s, got %s\n", a, b)

			t.Errorf("expect %sThis%s, got %sThis%s\n", c.want, reset, got, reset)
		}
	}
}

func TestSGR_ANSIcolor(t *testing.T) {
	tc := []struct {
		fg   uint32
		bg   uint32
		attr uint32
		want string
	}{
		{31, 47, Bold, "\033[0;1;31;47m"},
		{0, 0, Italic, "\033[0;3m"},
		{39, 49, Italic, "\033[0;3m"},
		{90, 107, Underlined, "\033[0;4;38;5;8;48;5;15m"},
		{31, 106, Faint, "\033[0;31;48;5;14m"},
	}

	for _, c := range tc {
		r := Renditions{}
		r.SetRendition(c.fg)
		r.SetRendition(c.bg)
		r.SetAttributes(c.attr, true)
		got := r.SGR()
		if c.want != got {
			a := strings.ReplaceAll(c.want, "\033", "ESC")
			b := strings.ReplaceAll(got, "\033", "ESC")
			t.Logf("expect %s, got %s\n", a, b)

			t.Errorf("expect %sThis%s, got %sThis%s\n", c.want, reset, got, reset)
		}
	}
}
