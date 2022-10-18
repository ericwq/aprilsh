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

func (d *Display) NewFrame(initialized bool, last, f *Emulator) string {
	var b strings.Builder
	ti := d.ti

	// has bell been rung?
	if f.cf.getBellCount() != last.cf.getBellCount() {
		ti.TPuts(&b, ti.Bell)
	}

	// has icon name or window title changed?
	if d.hasTitle && f.cf.isTitleInitialized() &&
		(!initialized || f.cf.getIconName() != last.cf.getIconName() || f.cf.getWindowTitle() != last.cf.getWindowTitle()) {
		if f.cf.getIconName() == f.cf.getWindowTitle() {
			// write combined Icon Name and Window Title
			fmt.Fprintf(&b, "\x1B]0;%s\x1B\\", f.cf.getWindowTitle())
			// ST is more correct, but BEL more widely supported
			// we use ST as the ending
		} else {
			// write Icon Name
			fmt.Fprintf(&b, "\x1B]1;%s\x1B\\", f.cf.getIconName())

			// write Window Title
			fmt.Fprintf(&b, "\x1B]2;%s\x1B\\", f.cf.getWindowTitle())
		}
	}

	// has reverse video state changed?
	if !initialized || f.reverseVideo != last.reverseVideo {
		// set reverse video
		if f.reverseVideo {
			fmt.Fprintf(&b, "\x1B[?5h]")
		} else {
			fmt.Fprintf(&b, "\x1B[?5l]")
		}
	}

	// has size changed?
	if !initialized || f.GetWidth() != last.GetWidth() || f.GetHeight() != last.GetHeight() {
		fmt.Fprintf(&b, "\x1B[r") // reset scrolling region, reset top/bottom margin
		ti.TPuts(&b, ti.AttrOff)  // sgr0, turn off all attribute modes
		ti.TPuts(&b, ti.Clear)    // clear, clear screen and home cursor

		initialized = false
		d.cursorX = 0
		d.cursorY = 0
		d.currentRendition = Renditions{}
	} else {
		d.cursorX = last.GetCursorCol()
		d.cursorY = last.GetCursorRow()
		d.currentRendition = last.GetRenditions()
	}

	// is cursor visibility initialized?
	if !initialized {
		d.cursorVisible = false
		fmt.Fprintf(&b, "\x1B[?25l]")
	}

	return b.String()
}

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
