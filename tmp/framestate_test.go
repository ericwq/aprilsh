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
)

func TestFrameStateNewFrameSate(t *testing.T) {
	fb := NewFramebuffer(8, 8)
	fs := NewFrameState(fb)

	if fs.lastFrame == nil {
		t.Errorf("expect lastFrame not nil, got %p\n", fs.lastFrame)
	}

	if !fs.cursorVisible {
		t.Errorf("expect cursorVisible false, got %t\n", fs.cursorVisible)
	}

	if fs.currentRendition != fb.DS.renditions {
		t.Errorf("expect currentRendition=%v, got %v\n", fb.DS.renditions, fs.currentRendition)
	}

	want := fb.DS.GetWidth() * fb.DS.GetHeight() * 4
	if fs.strBuiler.Cap() != want {
		t.Errorf("strBuiler expect size %d, got %d\n", want, fs.strBuiler.Cap())
	}
}

func TestFrameStateAppendXXX(t *testing.T) {
	fb := NewFramebuffer(8, 8)
	fs := NewFrameState(fb)

	want := "*四姑娘山----aprilsh四"
	fs.AppendByte('*')
	fs.AppendRune('四')
	fs.AppendBytes([]byte{'\xe5', '\xa7', '\x91', '\xe5', '\xa8', '\x98', '\xe5', '\xb1', '\xb1'})
	fs.AppendRepeatByte(4, '-')
	fs.AppendString("aprilsh")

	// prepare cell
	var cell Cell
	cell.Append('四')

	fs.AppendCell(&cell)

	if want != fs.strBuiler.String() {
		t.Errorf("Append... expect %s, got %s\n", want, fs.strBuiler.String())
	}
}

func TestFrameStateAppendSilentMove(t *testing.T) {
	tc := []struct {
		name string
		cy   int
		cx   int
		y    int
		x    int
		want string
	}{
		{"in scope(2,2)", 0, 0, 2, 2, "\033[?25l\033[3;3H"},
		{"in scope(9,49)", 0, 0, 9, 49, "\033[?25l\033[10;50H"},
		{"no move", 2, 2, 2, 2, ""},
		{"3backspace", 2, 5, 2, 2, "\033[?25l\b\b\b"},
		{"4backspace", 3, 5, 3, 1, "\033[?25l\b\b\b\b"},
		{"CR+4LF", 2, 2, 6, 0, "\033[?25l\r\n\n\n\n"},
		{"---4LF", 2, 0, 6, 0, "\033[?25l\n\n\n\n"},
		{"--- CR", 2, 2, 2, 0, "\033[?25l\r"},
	}

	// implicity frame szie
	maxX := 50
	maxY := 10

	for _, v := range tc {
		// prepare the FrameState
		fb := NewFramebuffer(maxX, maxY)
		fs := NewFrameState(fb)

		// prepare the cursor y,x
		fs.cursorX = v.cx
		fs.cursorY = v.cy

		// send move instruction
		fs.AppendSilentMove(v.y, v.x)

		got := fs.strBuiler.String()
		if v.want != got {
			t.Errorf("%s:\t expect %q, got %q\n", v.name, v.want, got)
		}
	}
}

func TestFrameStateUpdateRendition(t *testing.T) {
	tc := []struct {
		name  string
		r     Renditions
		other Renditions
		force bool
		want  string
	}{
		{"skip", Renditions{}, Renditions{}, false, ""},
		{"force", Renditions{}, Renditions{bgColor: ColorOlive}, true, "\033[0;43m"},
		{"other", Renditions{}, Renditions{bgColor: ColorGreen}, false, "\033[0;42m"},
	}
	// implicity frame szie
	maxX := 50
	maxY := 10

	// prepare the FrameState
	fb := NewFramebuffer(maxX, maxY)

	for _, v := range tc {
		// fresh FrameState
		fs := NewFrameState(fb)

		// pre-conditions
		fs.currentRendition = v.r

		fs.UpdateRendition(v.other, v.force)

		got := fs.strBuiler.String()
		if v.want != got {
			t.Errorf("%s:\t sequence expect %q, got %q\n", v.name, v.want, got)
		}

		if v.other != fs.currentRendition {
			t.Errorf("%s:\t renditions expect [%v], got [%v]\n", v.name, v.other, fs.currentRendition)
		}
	}
}
