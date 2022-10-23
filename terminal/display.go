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
	"os"
	"strings"

	"github.com/ericwq/terminfo"
	_ "github.com/ericwq/terminfo/base"
	"github.com/ericwq/terminfo/dynamic"
)

/*
	 questions

	 do we need to package the the terminfo DB into application?
	    yes, mosh-server depends on ncurses-terminfo-base and  ncurses-libs
	 how to read terminfo DB? through ncurses lib or directly?
		yes the answer is read through tcell.
	 how to operate terminal? through direct escape sequence or through terminfo DB?
	 how to replace the following functions? setupterm(), tigetnum(), tigetstr(), tigetflag()
*/
type Display struct {
	// erase character is part of vt200 but not supported by tmux
	// (or by "screen" terminfo entry, which is what tmux advertises)
	hasECH   bool
	hasBCE   bool   // erases result in cell filled with background color
	hasTitle bool   // supports window title and icon name
	smcup    string // enter and exit alternate screen mode
	rmcup    string // enter and exit alternate screen mode
	ti       *terminfo.Terminfo

	// fields from FrameState
	cursorX, cursorY int
	currentRendition Renditions
	cursorVisible    bool
}

// https://github.com/gdamore/tcell the successor of termbox-go
// https://cs.opensource.google/go/x/term/+/master:README.md
// apk add mandoc man-pages ncurses-doc
// apk add ncurses-terminfo
// apk add ncurses-terminfo-base
// apk add ncurses
// https://ishuah.com/2021/03/10/build-a-terminal-emulator-in-100-lines-of-go/
//
// use TERM environment var to initialize display, if useEnvironment is true.
func NewDisplay(useEnvironment bool) (d *Display, e error) {
	d = &Display{}
	d.hasECH = true
	d.hasBCE = true
	d.hasTitle = true

	if useEnvironment {
		term := os.Getenv("TERM")
		var ti *terminfo.Terminfo

		ti, e = lookupTerminfo(term)
		if e != nil {
			return nil, e
		}

		// check for ECH
		if ti.EraseChars != "" {
			d.hasECH = true
		}

		// check for BCE
		if ti.BackColorErase {
			d.hasBCE = true
		}

		// Check if we can set the window title and icon name.  terminfo does not
		// have reliable information on this, so we hardcode a whitelist of
		// terminal type prefixes.  This is the list from Debian's default
		// screenrc, plus "screen" itself (which also covers tmux).
		d.hasTitle = false
		titleTermTypes := []string{"xterm", "rxvt", "kterm", "Eterm", "screen"}
		if term != "" {
			for _, tt := range titleTermTypes {
				if strings.HasPrefix(term, tt) {
					d.hasTitle = true
					break
				}
			}
		}

		// TODO consider use MOSH_NO_TERM_INIT to control this behavior
		d.smcup = ti.EnterCA
		d.rmcup = ti.ExitCA

		d.ti = ti
	}
	return d, nil
}

// lookupTerminfo attempts to find a definition for the named $TERM falling
// back to attempting to parse the output from infocmp.
func lookupTerminfo(name string) (ti *terminfo.Terminfo, e error) {
	ti, e = terminfo.LookupTerminfo(name)
	if e != nil {
		ti, e = loadDynamicTerminfo(name)
		if e != nil {
			return nil, e
		}
		terminfo.AddTerminfo(ti)
	}

	return
}

func loadDynamicTerminfo(term string) (*terminfo.Terminfo, error) {
	ti, _, e := dynamic.LoadTerminfo(term)
	if e != nil {
		return nil, e
	}
	return ti, nil
}

// NewFrame() compare two terminal and generate mix (grapheme and control sequence) stream
// to replicate the new terminal content and state to the existing one.
//
// - initialized: the first time is false.
// - last: the existing terminal state.
// - f: the new terminal state.
func (d *Display) NewFrame(initialized bool, oldE, newE *Emulator) string {
	var b strings.Builder
	ti := d.ti

	// has bell been rung?
	if newE.cf.getBellCount() != oldE.cf.getBellCount() {
		ti.TPuts(&b, ti.Bell)
	}

	// has icon name or window title changed?
	if d.hasTitle && newE.cf.isTitleInitialized() &&
		(!initialized || newE.cf.getIconName() != oldE.cf.getIconName() || newE.cf.getWindowTitle() != oldE.cf.getWindowTitle()) {
		if newE.cf.getIconName() == newE.cf.getWindowTitle() {
			// write combined Icon Name and Window Title
			fmt.Fprintf(&b, "\x1B]0;%s\x07", newE.cf.getWindowTitle())
			// ST is more correct, but BEL more widely supported
		} else {
			// write Icon Name
			fmt.Fprintf(&b, "\x1B]1;%s\x07", newE.cf.getIconName())

			// write Window Title
			fmt.Fprintf(&b, "\x1B]2;%s\x07", newE.cf.getWindowTitle())
		}
	}

	// has reverse video state changed?
	if !initialized || newE.reverseVideo != oldE.reverseVideo {
		// set reverse video
		if newE.reverseVideo {
			fmt.Fprintf(&b, "\x1B[?5h]")
		} else {
			fmt.Fprintf(&b, "\x1B[?5l]")
		}
	}

	// has size changed?
	// the size of the display terminal isn't changed.
	// the size of the received terminal is changed by ApplyString()
	if !initialized || newE.GetWidth() != oldE.GetWidth() || newE.GetHeight() != oldE.GetHeight() {
		// TODO why reset scrolling region?
		fmt.Fprintf(&b, "\x1B[r") // smgtb, reset scrolling region, reset top/bottom margin
		ti.TPuts(&b, ti.AttrOff)  // sgr0, "\x1B[0m" turn off all attribute modes
		ti.TPuts(&b, ti.Clear)    // clear, "\x1B[H\x1B[2J" clear screen and home cursor

		initialized = false
		d.cursorX = 0
		d.cursorY = 0
		d.currentRendition = Renditions{}
	} else {
		d.cursorX = oldE.GetCursorCol()
		d.cursorY = oldE.GetCursorRow()
		d.currentRendition = oldE.GetRenditions()
	}

	// has the screen buffer mode changed?
	if !initialized || newE.altScreenBufferMode != oldE.altScreenBufferMode {
		// change the screen buffer mode
		oldE.switchScreenBufferMode(newE.altScreenBufferMode)

		if newE.altScreenBufferMode {
			fmt.Fprint(&b, "\x1B[?47h")
		} else {
			fmt.Fprint(&b, "\x1B[?47l")
		}
	}

	// saved cursor changed?
	if !initialized || newE.savedCursor_DEC.isSet != oldE.savedCursor_DEC.isSet {
		if newE.savedCursor_DEC.isSet {
			hdl_esc_decsc(oldE)

			fmt.Fprint(&b, "\x1B[7") // sc, TODO not supported by terminfo
		} else {
			hdl_esc_decrc(oldE)

			fmt.Fprint(&b, "\x1B[8") // rc, TODO not supported by terminfo
		}
	}

	/* copy old screen and resize */
	// what the influence of margin output?
	// prepare place for the old screen
	oldScreen := make([]Cell, oldE.GetWidth()*oldE.GetHeight())
	oldE.cf.fullCopyCells(oldScreen)

	// prepare place for the new screen
	newPlace := make([]Cell, newE.GetWidth()*newE.GetHeight())

	rowLen := Min(oldE.GetWidth(), newE.GetWidth())      // minimal row length
	nCopyRows := Min(oldE.GetHeight(), newE.GetHeight()) // minimal row number

	// copy the old screen to the new place
	for pY := 0; pY < nCopyRows; pY++ {
		srcStartIdx := pY
		srcEndIdx := srcStartIdx + rowLen
		dstStartIdx := rowLen * pY
		copy(newPlace[dstStartIdx:], oldScreen[srcStartIdx:srcEndIdx])
	}
	oldScreen = nil
	/* copy old screen and resize */

	// is cursor visibility initialized?
	if !initialized {
		d.cursorVisible = false
		ti.TPuts(&b, ti.HideCursor) // civis, "\x1B[?25l]" showCursorMode = false
	}

	// f is new , last is old

	return b.String()
}

// putRow(): compare two rows to generate the stream to replicate the row.
// if wrap, write the first column
// if the rows are the same, just return (false)
// for each cell:
// - if the cells are the same, skip it.
// - if the cells are empty, counting it.
// - output the empty cells by count number.
// - re-count empty cell with different rendition.
// - output the empty cells by count number.
// - if the cells are not empty cell, output it.
// clear or write empty cells at EOL if possible.
// whether we should wrap

func (d *Display) open() string {
	var b strings.Builder
	if d.smcup != "" {
		b.WriteString(d.smcup)
	}
	fmt.Fprintf(&b, "\x1B[?1h") // DECSET: set application cursor key mode
	return b.String()
}

func (d *Display) close() string {
	var b strings.Builder
	fmt.Fprintf(&b, "\x1B[?1l")                                     // DECRST: set ANSI cursor key mode
	fmt.Fprintf(&b, "\x1B[0m")                                      // SGR: reset character attributes, foreground color and background color
	fmt.Fprintf(&b, "\x1B[?25h")                                    // DECTCEM: show cursor mode
	fmt.Fprintf(&b, "\x1B[?1003l\x1B[?1002l\x1B[?1001l\x1B[?1000l") // disable mouse tracking mode
	fmt.Fprintf(&b, "\x1B[?1015l\x1B[?1006l\x1B[?1005l")            // reset to default mouse tracking encoding
	if d.rmcup != "" {
		b.WriteString(d.rmcup)
	}
	return b.String()
}
