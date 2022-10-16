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
	"os"

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
}

// https://github.com/gdamore/tcell the successor of termbox-go
// https://cs.opensource.google/go/x/term/+/master:README.md
// apk add mandoc man-pages ncurses-doc
// apk add ncurses-terminfo
// apk add ncurses-terminfo-base
// apk add ncurses
// https://ishuah.com/2021/03/10/build-a-terminal-emulator-in-100-lines-of-go/

// use TERM environment var to initialize display, if useEnvironment is true.
func NewDisplay(useEnvironment bool) (d *Display, e error) {
	d = &Display{}
	d.hasECH = true
	d.hasBCE = true
	d.hasTitle = true

	if useEnvironment {
		d.ti, e = lookupTerminfo(os.Getenv("TERM"))
		if e != nil {
			return nil, e
		}
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

func (d *Display) NewFrame(initialized bool, oldfb, newfb *Framebuffer) string {
	return ""
}
