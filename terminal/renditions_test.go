package terminal

import (
	"strings"
	"testing"
)

const reset = "\033[0m"

func TestSetRendition(t *testing.T) {
	turnOn := []uint32{1, 2, 3, 4, 5, 6, 7, 8, 9}
	turnOnWant := []uint32{Bold, 0, Italic, Underlined, Blink, 0, Inverse, Invisible, 0}

	r := Renditions{}
	for i, c := range turnOn {
		r.ClearAttributes()
		// set the flag
		r.SetRendition(uint32(c))

		// check the flag and skip the undefined item
		if turnOnWant[i] > 0 && !r.GetAttributes(turnOnWant[i]) {
			t.Errorf("case [%d] expect %8b, got %8b\n", c, c, r.attributes)
		}
	}

	turnOff := []uint32{22, 23, 24, 25, 26, 27, 28, 29}
	turnOffWant := []uint32{Bold, Italic, Underlined, Blink, 0, Inverse, Invisible, 0}
	for i, c := range turnOff {
		// skip the undefined one
		if turnOffWant[i] == 0 {
			continue
		}
		r.ClearAttributes()
		// set the flag first
		r.SetAttributes(turnOffWant[i], true)
		// next action should disable the flag
		r.SetRendition(c)

		// error if the flag is not clear
		if r.GetAttributes(turnOffWant[i]) {
			t.Errorf("case [%d] expect %8b, got %8b\n", c, c, r.attributes)
		}
	}
}

func TestSetTrueColor(t *testing.T) {
	tc := []struct {
		r, g, b uint32
		want    uint32
	}{
		{2, 3, 4, makeTrueColor(2, 3, 4)},
		{200, 300, 400, makeTrueColor(200, 300, 400)},
	}

	for _, c := range tc {
		r := Renditions{}
		r.SetForegroundColor(TrueColorMask | c.r<<16 | c.g<<8 | c.b)
		r.SetBackgroundColor(TrueColorMask | c.r<<16 | c.g<<8 | c.b)
		if r.fgColor != c.want || r.bgColor != c.want {
			t.Logf("expect foreground color:%2x, got:%2x\n", c.want, r.fgColor)
			t.Errorf("expect background color:%2x, got:%2x\n", c.want, r.fgColor)
		}
	}
}

func TestSGR_RGBColor(t *testing.T) {
	tc := []struct {
		fr, fg, fb uint32
		br, bg, bb uint32
		attr       uint32
		want       string
	}{
		{33, 47, 12, 123, 24, 34, Bold, "\033[0;1;38:2:33:47:12;48:2:123:24:34m"},
		{0, 0, 0, 0, 0, 0, Italic, "\033[0;3;38:2:0:0:0;48:2:0:0:0m"},
		{12, 34, 128, 59, 190, 155, Underlined, "\033[0;4;38:2:12:34:128;48:2:59:190:155m"},
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
		{33, 47, Bold, "\033[0;1;38:5:33;48:5:47m"},
		{0, 0, Italic, "\033[0;3;30;40m"},
		{128, 155, Underlined, "\033[0;4;38:5:128;48:5:155m"},
		{205, 228, Inverse, "\033[0;7;38:5:205;48:5:228m"},
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
		{30, 47, Bold, "\033[0;1;30;47m"},
		{0, 0, Bold, "\033[0;1m"},
		{0, 0, Italic, "\033[0;3m"},
		{0, 0, Underlined, "\033[0;4m"},
		{39, 49, Invisible, "\033[0;8m"},
		{90, 107, Underlined, "\033[0;4;38:5:8;48:5:15m"},
		{37, 40, Faint, "\033[0;37;40m"},
		{97, 100, Blink, "\033[0;5;38:5:15;48:5:8m"},
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
