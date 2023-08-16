// Copyright 2022 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package terminal

import (
	// "strings"
	"os"
	"testing"
)

const reset = "\033[0m"

func TestRenditionsComparable(t *testing.T) {
	tc := []struct {
		renditions   int
		fgColorIndex int
		bgColorIndex int
	}{
		{5, 30, 40},
		{0, 30, 40},
		{39, 30, 40},
		{49, 30, 40},
		{37, 30, 40}, // fg only
		{47, 30, 40}, // bg only
		{97, 30, 40},
		{107, 30, 40},
	}
	for _, c := range tc {
		r1 := NewRenditions(c.renditions)
		r1.SetForegroundColor(c.fgColorIndex)
		r1.SetBackgroundColor(c.bgColorIndex)

		r2 := NewRenditions(c.renditions)
		r2.SetForegroundColor(c.fgColorIndex)
		r2.SetBackgroundColor(c.bgColorIndex)
		if r1 != r2 {
			t.Errorf("case %d r1=%v, r2=%v\n", c.renditions, r1, r2)
		}
	}
}

func TestRenditionsSetAttributes(t *testing.T) {
	attrs := []charAttribute{Bold, Faint, Italic, Underlined, Blink, RapidBlink, Inverse, Invisible}

	r := Renditions{}
	for i, v := range attrs {
		r.ClearAttributes()
		// set the flag
		r.SetAttributes(v, true)

		// check the flag
		if v2, ok := r.GetAttributes(v); ok && !v2 {
			t.Errorf("case [%d] expect %t, got %t\n", i, true, v2)
		}
	}

	on := []int{22, 23, 24, 25, 0, 27, 28}
	attrs2 := []charAttribute{Bold, Italic, Underlined, Blink, 0, Inverse, Invisible}

	for i, v := range attrs2 {
		if on[i] == 0 {
			continue
		}
		r.ClearAttributes()
		// set the flag first
		r.SetAttributes(v, true)
		// next action should clear the flag
		r.buildRendition(on[i])

		// error if the flag is not clear
		if v2, ok := r.GetAttributes(v); ok && v2 {
			t.Errorf("case [%d] expect %t, got %t\n", i, false, v2)
		}
	}
}

func TestRenditionsGetAttributesReturnFalse(t *testing.T) {
	r := Renditions{}

	if _, ok := r.GetAttributes(charAttribute(9)); ok {
		t.Errorf("GetAttributes should return false, but get %t\n", true)
	}
}

func TestRenditionsSGR_RGBColor(t *testing.T) {
	tc := []struct {
		fr, fg, fb int
		br, bg, bb int
		attr       charAttribute
		want       string
	}{
		{33, 47, 12, 123, 24, 34, Bold, "\033[0;1;38:2:33:47:12;48:2:123:24:34m"},
		{0, 0, 0, 0, 0, 0, Italic, "\033[0;3;38:2:0:0:0;48:2:0:0:0m"},
		{12, 34, 128, 59, 190, 155, Underlined, "\033[0;4;38:2:12:34:128;48:2:59:190:155m"},
	}

	for _, c := range tc {
		r := &Renditions{}
		r.SetFgColor(c.fr, c.fg, c.fb)
		r.SetBgColor(c.br, c.bg, c.bb)
		if v, ok := r.GetAttributes(c.attr); ok && v { // Now, the attributes is not set
			t.Errorf("expect %t,ok=%t, got false", v, ok)
		}
		r.buildRendition(int(c.attr)) // set the attributes.
		got := r.SGR()
		if c.want != got {
			t.Logf("expect %q, got %q\n", c.want, got)
			t.Errorf("expect %sThis%s, got %sThis%s\n", c.want, reset, got, reset)
		}
	}
}

func TestRenditionsSGR_256color(t *testing.T) {
	tc := []struct {
		fg   Color
		bg   Color
		attr charAttribute
		want string
	}{
		{Color33, Color47, RapidBlink, "\033[0;6;38:5:33;48:5:47m"},  // 88-color
		{ColorDefault, ColorDefault, Italic, "\033[0;3m"},            // just italic
		{ColorDefault, ColorDefault, charAttribute(38), "\x1B[0m"},   // default Renditions and no charAttribute generate empty string
		{Color128, Color155, Blink, "\033[0;5;38:5:128;48:5:155m"},   // 256-color
		{Color205, Color228, Inverse, "\033[0;7;38:5:205;48:5:228m"}, // 256-color
		{ColorRed, ColorWhite, charAttribute(38), "\033[0;91;107m"},  // 16-color set
	}

	for _, c := range tc {
		// prepare SGR condition
		r := Renditions{}
		r.setAnsiForeground(c.fg)
		r.setAnsiBackground(c.bg)
		r.buildRendition(int(c.attr))

		// call SGR
		got := r.SGR()

		// validate the result
		if c.want != got {
			t.Logf("expect %q, got %q\n", c.want, got)
			t.Errorf("expect %sThis%s, got %sThis%s\n", c.want, reset, got, reset)
		}
	}
}

func TestRenditionsSGR_ANSIcolor(t *testing.T) {
	tc := []struct {
		fg   int
		bg   int
		attr charAttribute
		want string
	}{
		{30, 47, Bold, "\033[0;1;30;47m"},
		{0, 0, Bold, "\033[0;1m"},
		{0, 0, charAttribute(38), "\x1B[0m"}, // buildRendition doesn't support 38,48
		{0, 0, Italic, "\033[0;3m"},
		{0, 0, Underlined, "\033[0;4m"},
		{39, 49, Invisible, "\033[0;8m"},
		{37, 40, Faint, "\033[0;2;37;40m"},
		{90, 107, Underlined, "\033[0;4;90;107m"},
		{97, 100, Blink, "\033[0;5;97;100m"},
		{0, 34, Bold, "\033[0;1;34m"},
	}

	for _, c := range tc {
		r := Renditions{}
		r.buildRendition(c.fg)
		r.buildRendition(c.bg)
		r.buildRendition(int(c.attr))
		got := r.SGR()
		if c.want != got {
			t.Logf("expect %q, got %q\n", c.want, got)

			t.Errorf("expect %sThis%s, got %sThis%s\n", c.want, reset, got, reset)
		}
	}
}

func TestRenditionsBuildRenditions(t *testing.T) {
	r := Renditions{}
	if r.buildRendition(48) { // buildRendition doesn't support 38,48
		t.Errorf("buildRenditions expect false, got %t\n", true)
	}
}

func TestRenditionsRebuild(t *testing.T) {

	tc := []struct {
		label  string
		mix    string
		posY   []int
		posX   []int
		expect []string
	}{
		{"mix color", "\x1b[1;34mdevelop\x1b[m  ", []int{0, 0}, []int{0, 7},
			[]string{"\x1b[0;1;34m", "\x1b[0m"}},
		{"super mix", "\x1B[5;1H1st space\x1B[0K\x1b[5;21H2nd!   \x1B[1;37;40m   3rd\x1b[5;79HEOL",
			[]int{4, 4, 4, 4}, []int{0, 26, 30, 78},
			[]string{"\x1b[0m", "\x1b[0m", "\x1B[0;1;37;40m", "\x1b[0;1;37;40m"}},
	}
	emu := NewEmulator3(80, 40, 40)
	os.Setenv("TERM", "xterm-256color")
	for _, v := range tc {
		// process the mix
		emu.HandleStream(v.mix)

		for i := range v.expect {
			// get the cell
			c := emu.GetCell(v.posY[i], v.posX[i])

			// get the renditions
			r := c.GetRenditions()

			// get text representation of renditions
			got := r.SGR()

			// validate the rendition text
			if got != v.expect[i] {
				t.Errorf("%s at position (%d,%d) expect renditions %q, got %q\n",
					v.label, v.posY[i], v.posX[i], v.expect[i], got)
			}
		}
	}
}
