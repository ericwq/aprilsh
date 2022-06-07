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
)

type FrameState struct {
	// the content changes frquently
	strBuiler strings.Builder

	cursorX int
	cursorY int

	currentRendition Renditions
	cursorVisible    bool

	lastFrame *Framebuffer
}

func NewFrameState(fb *Framebuffer) *FrameState {
	fs := FrameState{}
	fs.cursorVisible = fb.DS.CursorVisible
	fs.currentRendition = fb.DS.renditions
	fs.lastFrame = fb

	// Preallocate for better performance.  Make a guess-- doesn't matter for correctness
	fs.strBuiler.Grow(fs.lastFrame.DS.GetWidth() * fs.lastFrame.DS.GetHeight() * 4)
	return &fs
}

func (fs *FrameState) AppendByte(c byte)    { fs.strBuiler.WriteByte(c) }
func (fs *FrameState) AppendRune(r rune)    { fs.strBuiler.WriteRune(r) }
func (fs *FrameState) AppendBytes(b []byte) { fs.strBuiler.WriteString(string(b)) }

func (fs *FrameState) AppendRepeatByte(count int, c byte) {
	fs.strBuiler.WriteString(strings.Repeat(string(c), count))
}

func (fs *FrameState) AppendString(s string) { fs.strBuiler.WriteString(s) }
func (fs *FrameState) AppendCell(cell *Cell) { cell.PrintGrapheme(&fs.strBuiler) }

func (fs *FrameState) AppendSilentMove(y, x int) {
	if fs.cursorX == x && fs.cursorY == y {
		return
	}
	// turn off cursor if necessary before moving cursor
	if fs.cursorVisible {
		// https://invisible-island.net/xterm/ctlseqs/ctlseqs.html#h3-Functions-using-CSI-_-ordered-by-the-final-character_s_
		// DEC Private Mode Reset (DECRST).
		// Hide cursor (DECTCEM), VT220.
		fs.AppendString("\033[?25l")
		fs.cursorVisible = false
	}
	fs.AppendMove(y, x)
}

func (fs *FrameState) AppendMove(y, x int) {
	lastX := fs.cursorX
	lastY := fs.cursorY

	fs.cursorX = x
	fs.cursorY = y

	// Only optimize if cursor pos is known
	if lastX != -1 && lastY != -1 {
		// Can we use CR and/or LF?  They're cheap and easier to trace.
		// the CUP escape sequence only takes 6-8 bytes.
		// so 5 is the upper limit for cheap solution
		if x == 0 && y-lastY >= 0 && y-lastY < 5 {
			if lastX != 0 {
				fs.AppendByte('\r')
			}
			fs.AppendRepeatByte(y-lastY, '\n')
			return
		}
		// Backspaces are good too.
		if y == lastY && x-lastX < 0 && x-lastX > -5 {
			fs.AppendRepeatByte(lastX-x, '\b')
			return
		}
		// More optimizations are possible.
	}

	// https://invisible-island.net/xterm/ctlseqs/ctlseqs.html#h3-Functions-using-CSI-_-ordered-by-the-final-character_s_
	// Cursor Position [row;column] (default = [1,1]) (CUP)
	fmt.Fprintf(&fs.strBuiler, "\033[%d;%dH", y+1, x+1)
}

// change the renditions, if force is true
func (fs *FrameState) UpdateRendition(other Renditions, force bool) {
	if force || fs.currentRendition!=other {

		fs.AppendString(other.SGR())
		fs.currentRendition = other
	}
}
