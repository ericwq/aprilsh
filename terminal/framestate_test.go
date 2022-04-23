package terminal

import (
	"strings"
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

			// replace the escape char to avoid mess the screen
			a := strings.ReplaceAll(v.want, "\033", "ESC")
			a = strings.ReplaceAll(a, "\b", "\\b")
			a = strings.ReplaceAll(a, "\r", "\\r")
			a = strings.ReplaceAll(a, "\n", "\\n")

			b := strings.ReplaceAll(got, "\033", "ESC")
			b = strings.ReplaceAll(b, "\b", "\\b")
			b = strings.ReplaceAll(b, "\r", "\\r")
			b = strings.ReplaceAll(b, "\n", "\\n")

			t.Errorf("%s:\t expect [%s], got [%s]\n", v.name, a, b)

			// t.Errorf("%s:\t expect [%s], got [%s]\n", v.name, v.want, fs.strBuiler.String())
		}
	}
}
