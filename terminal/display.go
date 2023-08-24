// Copyright 2022 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package terminal

import (
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"

	"github.com/ericwq/terminfo"
	_ "github.com/ericwq/terminfo/base"
	"github.com/ericwq/terminfo/dynamic"
)

/*
 *	 The following are some interesting questions I asked several month ago. As I know terminfo
 *	 and terminal better, the answter is more clear and confident than before.
 *
 *	 do we need to package the the terminfo DB into application?
 *	 - yes, mosh-server depends on ncurses-terminfo-base and  ncurses-libs
 *	 how to read terminfo DB? through ncurses lib or directly?
 *	 - yes the answer is read through tcell.
 *	 how to operate terminal? through direct escape sequence or through terminfo DB?
 *	 how to replace the following functions? setupterm(), tigetnum(), tigetstr(), tigetflag()
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
	showCursorMode   bool // mosh: cursorVisible

	// logW *log.Logger
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

	// d.logW = log.New(os.Stderr, "WARN: ", log.Ldate|log.Ltime|log.Lshortfile)

	if useEnvironment {
		term := os.Getenv("TERM")
		var ti *terminfo.Terminfo

		ti, e = LookupTerminfo(term)
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

// LookupTerminfo attempts to find a definition for the named $TERM falling
// back to attempting to parse the output from infocmp.
func LookupTerminfo(name string) (ti *terminfo.Terminfo, e error) {
	ti, e = terminfo.LookupTerminfo(name)
	if e != nil {
		// ti, e = loadDynamicTerminfo(name)
		ti, _, e := dynamic.LoadTerminfo(name)
		if e != nil {
			return nil, e
		}
		terminfo.AddTerminfo(ti)
	}

	return
}

/*
func loadDynamicTerminfo(term string) (*terminfo.Terminfo, error) {
	ti, _, e := dynamic.LoadTerminfo(term)
	if e != nil {
		return nil, e
	}
	return ti, nil
}
*/

// return the specified row from the new terminal.
func getRow(newE *Emulator, posY int) (row []Cell) {
	start := newE.cf.getViewRowIdx(posY)
	end := start + newE.nCols
	row = newE.cf.cells[start:end]
	return row
}

// extract specified row from the resize screen.
func getRowFrom(from []Cell, posY int, w int) (row []Cell) {
	start := posY * w
	end := start + w
	row = from[start:end]
	return row
}

// compare two terminals and generate mix (grapheme and control sequence) sequence
// to rebuild the new terminal from the old one.
//
// - initialized: if false, it will redraw the whole terminal, otherwise only changed part.
// - oldE: the old terminal state.
// - newE: the new terminal state.
func (d *Display) NewFrame(initialized bool, oldE, newE *Emulator) string {
	var b strings.Builder
	// ti := d.ti

	// has bell been rung?
	if newE.cf.getBellCount() != oldE.cf.getBellCount() {
		// ti.TPuts(&b, ti.Bell)
		fmt.Fprint(&b, "\a")
	}

	// has icon label or window title changed?
	if d.hasTitle && newE.isTitleInitialized() &&
		(!initialized || newE.GetIconLabel() != oldE.GetIconLabel() || newE.GetWindowTitle() != oldE.GetWindowTitle()) {
		if newE.GetIconLabel() == newE.GetWindowTitle() {
			// write combined Icon label and Window Title
			fmt.Fprintf(&b, "\x1B]0;%s\x07", newE.GetWindowTitle())
			// ST is more correct, but BEL more widely supported
		} else {
			// write Icon label
			fmt.Fprintf(&b, "\x1B]1;%s\x07", newE.GetIconLabel())

			// write Window Title
			fmt.Fprintf(&b, "\x1B]2;%s\x07", newE.GetWindowTitle())
		}
	}

	// has reverse video state changed?
	if !initialized || newE.reverseVideo != oldE.reverseVideo {
		// set reverse video
		if newE.reverseVideo {
			fmt.Fprintf(&b, "\x1B[?5h")
		} else {
			fmt.Fprintf(&b, "\x1B[?5l")
		}
	}

	// has size changed?
	// the size of the display terminal isn't changed.
	// the size of the received terminal is changed by ApplyString()
	if !initialized || newE.nCols != oldE.nCols || newE.nRows != oldE.nRows {
		// TODO why reset scrolling region?
		fmt.Fprintf(&b, "\x1B[r") // smgtb, reset scrolling region, reset top/bottom margin

		// ti.TPuts(&b, ti.AttrOff)  // sgr0, "\x1B[0m" turn off all attribute modes
		// ti.TPuts(&b, ti.Clear)    // clear, "\x1B[H\x1B[2J" clear screen and home cursor
		fmt.Fprint(&b, "\x1B[0m")
		fmt.Fprint(&b, "\x1B[H\x1B[2J")

		initialized = false // resize will force the initialized
		d.cursorX = 0
		d.cursorY = 0
		d.currentRendition = Renditions{}
	} else {
		d.cursorX = oldE.GetCursorCol()
		d.cursorY = oldE.GetCursorRow()
		d.currentRendition = oldE.GetRenditions()
	}

	// init showCursorMode from old screen
	d.showCursorMode = oldE.showCursorMode

	// is cursor visibility initialized?
	if !initialized {
		// fmt.Printf("#NewFrame initialized=%t, d.showCursorMode=%t\n", initialized, d.showCursorMode)
		d.showCursorMode = false
		// ti.TPuts(&b, ti.HideCursor) // civis, "\x1B[?25l" showCursorMode = false
		fmt.Fprint(&b, "\x1B[?25l")
	}

	// has the screen buffer mode changed?
	// change screen buffer is something like resize, except resize remains partial content,
	// screen buffer mode reset the whole screen.
	if !initialized || newE.altScreenBufferMode != oldE.altScreenBufferMode {
		if newE.altScreenBufferMode {
			fmt.Fprint(&b, "\x1B[?47h")
		} else {
			fmt.Fprint(&b, "\x1B[?47l")
		}
	}

	// has the margin changed?
	if !initialized || (newE.marginTop != oldE.marginTop || newE.marginBottom != oldE.marginBottom) {
		if newE.cf.margin {
			fmt.Fprintf(&b, "\x1B[%d;%dr", newE.marginTop+1, newE.marginBottom) // new margin
		} else {
			fmt.Fprint(&b, "\x1B[r") // reset margin
		}
	}

	// has the horizontal margin changed?
	if !initialized || newE.horizMarginMode != oldE.horizMarginMode {
		if newE.horizMarginMode {
			fmt.Fprint(&b, "\x1B[?69h")
			if newE.hMargin != oldE.hMargin || newE.nColsEff != oldE.nColsEff {
				// decslrm set left/right margin
				fmt.Fprintf(&b, "\x1B[%d;%ds", newE.hMargin+1, newE.nColsEff)
			}
		} else {
			fmt.Fprint(&b, "\x1B[?69l")
		}
	}

	// has saved cursor changed?
	// Let the target terminal decide what to save, here we just issue the control sequence.
	//
	// Saves the following items in the terminal's memory:
	//
	// Cursor position
	// Character attributes set by the SGR command
	// Character sets (G0, G1, G2, or G3) currently in GL and GR
	// Wrap flag (autowrap or no autowrap)
	// State of origin mode (DECOM)
	// Selective erase attribute
	// Any single shift 2 (SS2) or single shift 3 (SS3) functions sent
	if !initialized || newE.savedCursor_DEC.isSet != oldE.savedCursor_DEC.isSet {
		if newE.savedCursor_DEC.isSet && !oldE.savedCursor_DEC.isSet {
			fmt.Fprint(&b, "\x1B7") // decsc
		} else if !newE.savedCursor_DEC.isSet && oldE.savedCursor_DEC.isSet {
			fmt.Fprint(&b, "\x1B8") // decrc
		}
	}

	// has SCO saved cursor changed
	if !initialized || newE.savedCursor_SCO.isSet != oldE.savedCursor_SCO.isSet {
		if newE.savedCursor_SCO.isSet && !oldE.savedCursor_SCO.isSet {
			fmt.Fprint(&b, "\x1B[s") // scosc
		} else if !newE.savedCursor_SCO.isSet && oldE.savedCursor_SCO.isSet {
			fmt.Fprint(&b, "\x1B[u") // scorc
		}
	}

	/* resize and copy old screen */
	// we copy the old screen to avoid changing the existing terminal state.

	// prepare place for the old screen
	oldScreen := make([]Cell, oldE.nCols*oldE.nRows)
	oldE.cf.fullCopyCells(oldScreen)

	// prepare place for the new screen
	resizeScreen := make([]Cell, newE.nCols*newE.nRows)

	nCopyCols := Min(oldE.nCols, newE.nCols) // minimal column length
	nCopyRows := Min(oldE.nRows, newE.nRows) // minimal row length

	// copy the old screen to the new place
	for pY := 0; pY < nCopyRows; pY++ {
		srcStartIdx := pY * nCopyCols
		srcEndIdx := srcStartIdx + nCopyCols
		dstStartIdx := pY * nCopyCols
		copy(resizeScreen[dstStartIdx:], oldScreen[srcStartIdx:srcEndIdx])
	}
	oldScreen = nil
	/* resize and copy old screen */

	var frameY int
	var oldRow []Cell
	var newRow []Cell

	// shortcut -- has display moved up(text up, window down) by a certain number of lines?
	if initialized {
		var linesScrolled int
		var scrollHeight int

		for row := 0; row < newE.GetHeight(); row++ {
			newRow = getRow(newE, 0)
			oldRow = getRowFrom(resizeScreen, row, newE.nCols)

			if reflect.DeepEqual(newRow, oldRow) {
				// if row 0, we're looking at ourselves and probably didn't scroll
				if row == 0 {
					break
				}

				// found a scroll: text up, window down
				linesScrolled = row
				scrollHeight = 1

				// how big is the region that was scrolled?
				for regionHeight := 1; linesScrolled+regionHeight < newE.GetHeight(); regionHeight++ {
					newRow = getRow(newE, regionHeight)
					oldRow = getRowFrom(resizeScreen, linesScrolled+regionHeight, newE.nCols)
					if reflect.DeepEqual(newRow, oldRow) {
						scrollHeight = regionHeight + 1
					} else {
						break
					}
				}

				break
			}
		}

		if scrollHeight > 0 {
			frameY = scrollHeight

			if linesScrolled > 0 {
				// reset the renditions
				d.updateRendition(&b, Renditions{}, true)

				topMargin := 0
				bottomMargin := topMargin + linesScrolled + scrollHeight - 1
				// fmt.Printf("#NewFrame scrollHeight=%2d, linesScrolled=%2d, frameY=%2d, bottomMargin=%2d\n",
				// 	scrollHeight, linesScrolled, frameY, bottomMargin)

				// Common case:  if we're already on the bottom line and we're scrolling the whole
				// creen, just do a CR and LFs.
				if scrollHeight+linesScrolled == newE.GetHeight() && d.cursorY+1 == newE.GetHeight() {
					fmt.Fprint(&b, "\r")
					// fmt.Fprint(&b, strings.Repeat("\n", linesScrolled)) // ind
					fmt.Fprintf(&b, "\x1B[%dS", linesScrolled)
					d.cursorX = 0
				} else {
					// set scrolling region
					// fmt.Fprintf(&b, "\x1B[%d;%dr", topMargin+1, bottomMargin+1)

					// go to bottom of scrolling region
					d.cursorY = -1
					d.cursorX = -1
					d.appendSilentMove(&b, bottomMargin, 0)

					// scroll text up by <linesScrolled>
					fmt.Fprintf(&b, "\x1B[%dS", linesScrolled)
					// fmt.Fprint(&b, strings.Repeat("\r", linesScrolled)) // ind

					// reset scrolling region
					// fmt.Fprint(&b, "\x1B[r")

					// invalidate cursor position after unsetting scrolling region
					d.cursorY = -1
					d.cursorX = -1
				}

				// Now we need a proper blank row.
				blankRow := make([]Cell, newE.nCols)
				for i := range blankRow {
					// set both contents and renditions
					blankRow[i] = newE.attrs
				}

				// do the move in our local new screen
				for i := topMargin; i <= bottomMargin; i++ {
					dstStart := i * newE.nCols

					if i+linesScrolled <= bottomMargin {
						copy(resizeScreen[dstStart:], getRowFrom(resizeScreen, linesScrolled+i, newE.nCols))
					} else {
						copy(resizeScreen[dstStart:], blankRow[:])
					}
				}
			}
		}
	}
	// Now update the display, row by row
	wrap := false
	for ; frameY < newE.GetHeight(); frameY++ {
		oldRow = getRowFrom(resizeScreen, frameY, newE.nCols)
		wrap = d.putRow(&b, initialized, oldE, newE, frameY, oldRow, wrap)
	}

	// fmt.Printf("#NewFrame d.cursorY=%d,d.cursorX=%d newE (%d,%d)\n", d.cursorY, d.cursorX, newE.GetCursorRow(), newE.GetCursorCol())
	// has cursor location changed?
	if !initialized || newE.GetCursorRow() != d.cursorY || newE.GetCursorCol() != d.cursorX {
		// TODO using cursor position from display or cursor position from terminal?
		d.appendMove(&b, newE.GetCursorRow(), newE.GetCursorCol())
	}

	// has cursor visibility changed?
	// during update row, appendSilentMove() might close the cursor,
	// Here we open cursor based on the new terminal state.

	// fmt.Printf("#NewFrame newE=%t, d=%t, oldE=%t, initialized=%t\n",
	// 	newE.showCursorMode, d.showCursorMode, oldE.showCursorMode, initialized)
	if !initialized || newE.showCursorMode != d.showCursorMode {
		if newE.showCursorMode {
			fmt.Fprint(&b, "\x1B[?25h") // cvvis
		} else {
			fmt.Fprint(&b, "\x1B[?25l") // civis
		}
	}

	// has cursor style changed?
	if !initialized || newE.cf.cursor.showStyle != oldE.cf.cursor.showStyle {
		Ps := 1 // default is blinking block
		switch newE.cf.cursor.showStyle {
		case CursorStyle_BlinkBlock:
			Ps = 1
		case CursorStyle_SteadyBlock:
			Ps = 2
		case CursorStyle_BlinkUnderline:
			Ps = 3
		case CursorStyle_SteadyUnderline:
			Ps = 4
		case CursorStyle_BlinkBar:
			Ps = 5
		case CursorStyle_SteadyBar:
			Ps = 6
		}
		fmt.Fprintf(&b, "\x1B[%d q", Ps)
	}

	// has cursor color changed to default?
	if !initialized || newE.cf.cursor.color != oldE.cf.cursor.color {
		if newE.cf.cursor.color == ColorDefault {
			fmt.Fprintf(&b, "\x1B]112\a")
		}
	}

	// has renditions changed?
	d.updateRendition(&b, newE.GetRenditions(), !initialized)

	// has bracketed paste mode changed?
	if !initialized || newE.bracketedPasteMode != oldE.bracketedPasteMode {
		if newE.bracketedPasteMode {
			fmt.Fprint(&b, "\x1B[?2004h")
		} else {
			fmt.Fprint(&b, "\x1B[?2004l")
		}
	}

	// has mouse reporting mode changed?
	if !initialized || newE.mouseTrk.mode != oldE.mouseTrk.mode {
		if newE.mouseTrk.mode == MouseTrackingMode_Disable {
			fmt.Fprint(&b, "\x1B[?1003l")
			fmt.Fprint(&b, "\x1B[?1002l")
			fmt.Fprint(&b, "\x1B[?1001l")
			fmt.Fprint(&b, "\x1B[?1000l")
		} else {
			// close old mouse reporting mode
			if oldE.mouseTrk.mode != MouseTrackingMode_Disable {
				fmt.Fprintf(&b, "\x1B[?%dl", oldE.mouseTrk.mode)
			}
			// open new mouse reporting mode
			fmt.Fprintf(&b, "\x1B[?%dh", newE.mouseTrk.mode)
		}
	}

	// has mouse focus mode changed?
	if !initialized || newE.mouseTrk.focusEventMode != oldE.mouseTrk.focusEventMode {
		if newE.mouseTrk.focusEventMode {
			fmt.Fprint(&b, "\x1B[?1004h")
		} else {
			fmt.Fprint(&b, "\x1B[?1004l")
		}
	}

	// has mouse encoding mode changed?
	if !initialized || newE.mouseTrk.enc != oldE.mouseTrk.enc {
		if newE.mouseTrk.enc == MouseTrackingEnc_Default {
			fmt.Fprint(&b, "\x1B[?1015l")
			fmt.Fprint(&b, "\x1B[?1006l")
			fmt.Fprint(&b, "\x1B[?1005l")
		} else {
			// close old mouse encoding mode
			if oldE.mouseTrk.enc != MouseTrackingEnc_Default {
				fmt.Fprintf(&b, "\x1B[?%dl", oldE.mouseTrk.enc)
			}
			// open new mouse encoding mode
			fmt.Fprintf(&b, "\x1B[?%dh", newE.mouseTrk.enc)
		}
	}

	// has auto wrap mode changed?
	if !initialized || newE.autoWrapMode != oldE.autoWrapMode {
		if newE.autoWrapMode {
			fmt.Fprint(&b, "\x1B[?7h")
		} else {
			fmt.Fprint(&b, "\x1B[?7l")
		}
	}

	// has auto newline mode changed?
	if !initialized || newE.autoNewlineMode != oldE.autoNewlineMode {
		if newE.autoNewlineMode {
			fmt.Fprint(&b, "\x1B[20h")
		} else {
			fmt.Fprint(&b, "\x1B[20l")
		}
	}

	// has keyboard action mode changed?
	if !initialized || newE.keyboardLocked != oldE.keyboardLocked {
		if newE.keyboardLocked {
			fmt.Fprint(&b, "\x1B[2h")
		} else {
			fmt.Fprint(&b, "\x1B[2l")
		}
	}

	// has insert mode changed?
	if !initialized || newE.insertMode != oldE.insertMode {
		if newE.insertMode {
			fmt.Fprint(&b, "\x1B[4h")
		} else {
			fmt.Fprint(&b, "\x1B[4l")
		}
	}

	// has local echo changed?
	if !initialized || newE.localEcho != oldE.localEcho {
		if newE.localEcho {
			fmt.Fprint(&b, "\x1B[12l") // reverse order
		} else {
			fmt.Fprint(&b, "\x1B[12h") // reverse order
		}
	}

	// has backspace send delete changed?
	if !initialized || newE.bkspSendsDel != oldE.bkspSendsDel {
		if newE.bkspSendsDel {
			fmt.Fprint(&b, "\x1B[?67l") // DECRST reverse order
		} else {
			fmt.Fprint(&b, "\x1B[?67h") // DECSET reverse order
		}
	}

	// has alt key as ESC changed?
	if !initialized || newE.altSendsEscape != oldE.altSendsEscape {
		if newE.altSendsEscape {
			fmt.Fprint(&b, "\x1B[?1036h") // DECSET
		} else {
			fmt.Fprint(&b, "\x1B[?1036l") // DECRST
		}
	}

	// has altScrollMode changed?
	if !initialized || newE.altScrollMode != oldE.altScrollMode {
		if newE.altScrollMode {
			fmt.Fprint(&b, "\x1B[?1007h") // DECSET
		} else {
			fmt.Fprint(&b, "\x1B[?1007l") // DECRST
		}
	}

	// has cursor key mode changed?
	// Note: This depends on real terminal emulator to apply cursorKeyMode.
	if !initialized || newE.cursorKeyMode != oldE.cursorKeyMode {
		switch newE.cursorKeyMode {
		case CursorKeyMode_Application:
			fmt.Fprint(&b, "\x1B[?1h") // DECSET
		case CursorKeyMode_ANSI:
			fmt.Fprint(&b, "\x1B[?1l") // DECRST
		}
	}

	// has origin mode changed?
	if !initialized || newE.originMode != oldE.originMode {
		switch newE.originMode {
		case OriginMode_ScrollingRegion:
			fmt.Fprint(&b, "\x1B[?6h") // DECSET
		case OriginMode_Absolute:
			fmt.Fprint(&b, "\x1B[?6l") // DECRST
		}
	}

	// has keypad mode changed?
	// Note: This depends on real terminal emulator to apply keypadMode.
	if !initialized || newE.keypadMode != oldE.keypadMode {
		switch newE.keypadMode {
		case KeypadMode_Application:
			fmt.Fprint(&b, "\x1B=") // DECKPAM
		case KeypadMode_Normal:
			fmt.Fprint(&b, "\x1B>") // DECKPNM
		}
	}

	// has column mode changed? the column mode is out of date.
	if !initialized || newE.colMode != oldE.colMode {
		switch newE.colMode {
		case ColMode_C132:
			fmt.Fprint(&b, "\x1B[?3h") // DECSET
		case ColMode_C80:
			fmt.Fprint(&b, "\x1B[?3l") // DECRST
		}
	}

	// has tab stop position changed?
	if !initialized || !reflect.DeepEqual(newE.tabStops, oldE.tabStops) {
		if len(newE.tabStops) == 0 {
			// clear tab stop if necessary
			fmt.Fprint(&b, "\x1B[3g") // TBC
		} else {
			// save the cursor position
			cursorY := d.cursorY
			cursorX := d.cursorX

			// rebuild the tab stop
			for _, tabStop := range newE.tabStops {
				d.appendMove(&b, 0, tabStop) // CUP: move cursor to the tab stop position
				fmt.Fprint(&b, "\x1BH")      // HTS: set current position as tab stop
			}

			// restore the cursor position
			d.appendMove(&b, cursorY, cursorX)
		}
	}

	// has conformance level changed?
	if !initialized || newE.compatLevel != oldE.compatLevel {
		switch newE.compatLevel {
		case CompatLevel_VT52:
			fmt.Fprint(&b, "\x1B[?2l") // DECSET
		case CompatLevel_VT100:
			fmt.Fprint(&b, "\x1B[61\"p") // DECSCL
		case CompatLevel_VT400:
			fmt.Fprint(&b, "\x1B[64\"p") // DECSCL
		}
	}

	// has key modifier encoding level changed?
	// Note: This depends on real terminal emulator to apply modifyOtherKeys.
	if !initialized || newE.modifyOtherKeys != oldE.modifyOtherKeys {
		// the possible value for modifyOtherKeys is [0,1,2]
		// fmt.Printf("#NewFrame modifyOtherKeys newE=%d, oldE=%d, initialized=%t\n",
		// 	newE.modifyOtherKeys, oldE.modifyOtherKeys, initialized)
		fmt.Fprintf(&b, "\x1B[>4;%dm", newE.modifyOtherKeys)
	}

	// has OSC 52 selection data changed?
	if !initialized || newE.selectionData != oldE.selectionData {
		// the selectionData is in the form of "\x1B]52;%s;%s\x1B\\"
		// see hdl_osc_52() for detail
		fmt.Fprint(&b, newE.selectionData)
	}

	// TODO do we need to consider cursor selection area.
	return b.String()
}

// compare new row with old row to generate the mix stream to rebuild the new row
// from the old one.
//
// if the previous row is wrapped, write the first column.
//
// if the two rows are the same (both cell and renditions), just return (false)
//
// for each cell:
//
// - if the cells are the same, skip it. change renditions if possible.
//
// - if the cells are empty, counting it.
//
// - output the empty cells with counting number.
//
// - re-count empty cell with different rendition.
//
// - output the empty cells by count number.
//
// - if the cells are not empty cell, output it.
//
// clear or write empty cells at EOL if possible. whether we should wrap
func (d *Display) putRow(out io.Writer, initialized bool, oldE *Emulator,
	newE *Emulator, frameY int, oldRow []Cell, wrap bool) bool {
	frameX := 0
	newRow := getRow(newE, frameY)

	// If we're forced to write the first column because of wrap, go ahead and do so.
	if wrap {
		cell := newRow[0]
		d.updateRendition(out, cell.GetRenditions(), false)
		d.appendCell(out, cell)

		// fmt.Printf("#putRow (%2d,%2d) is wrap-: contents=%q, renditions=%q - write wrap cell\n",
		// 	frameY, frameX, cell.contents, cell.renditions.SGR())

		frameX += cell.GetWidth()
		d.cursorX += cell.GetWidth()
	}

	// If rows are the same object, we don't need to do anything at all.
	if initialized && reflect.DeepEqual(newRow, oldRow) {
		return false
	}

	// this row should be wrapped. TODO: need to consider double width cell
	wrapThis := newRow[len(newRow)-1].wrap
	// fmt.Printf("#putRow row=%d, wrapThis=%t\n", frameY, wrapThis)
	rowWidth := newE.nCols
	clearCount := 0
	wroteLastCell := false
	blankRenditions := Renditions{}

	// iterate for every cell
	for frameX < rowWidth {
		cell := newRow[frameX]
		// fmt.Printf("#putRow pos=(%d,%d) cell=%q renditions=%q\n", frameY, frameX, cell, cell.renditions.SGR())

		// Does cell need to be drawn?  Skip all this.
		if initialized && clearCount == 0 && cell == oldRow[frameX] {
			// the new cell is the same as the old cell
			// don't do anything except move column counting.

			// fmt.Printf("#putRow (%2d,%2d) is same-: contents=%q, renditions=%q - skip cell\n",
			// 	frameY, frameX, cell.contents, cell.renditions.SGR())

			// check the renditions if it's changed.
			d.updateRendition(out, cell.renditions, false)
			frameX += cell.GetWidth()
			continue
		}

		// Slurp up all the empty cells
		if cell.IsBlank() {
			// it's empty cell
			// fmt.Printf("#putRow (%2d,%2d) is blank: %q\n", frameY, frameX, cell.contents)
			if cell.IsEarlyWrap() { // skip the early wrap cell. for double width cell
				frameX++
				continue
			}

			if clearCount == 0 {
				// remember the renditions of first empty cell
				blankRenditions = cell.GetRenditions()
			}
			if cell.GetRenditions() == blankRenditions {
				// Remember run of blank cells
				// counting the number of empty cells with same renditions
				clearCount++
				frameX++
				continue
			}
		}

		// Clear or write empty cells within the row (not to end).
		if clearCount > 0 { // draw empty cells previously counting
			// Move to the right(correct) position.
			d.appendSilentMove(out, frameY, frameX-clearCount)
			d.updateRendition(out, blankRenditions, false)

			// pcell := newRow[frameX-clearCount]
			// fmt.Printf("#putRow (%2d,%2d) is empty, length=%d, cell=%q, rend=%q - write empty\n",
			// 	frameY, frameX-clearCount, clearCount, pcell.contents, pcell.renditions.SGR())

			canUseErase := d.hasBCE || d.currentRendition == Renditions{}
			if canUseErase && d.hasECH && clearCount > 4 {
				// space is more efficient than ECH, if clearCount > 4
				fmt.Fprintf(out, "\x1B[%dX", clearCount)
			} else {
				// fmt.Printf("#putRow space=%q\n", strings.Repeat(" ", clearCount))
				fmt.Fprint(out, strings.Repeat(" ", clearCount))
				d.cursorX = frameX
			}

			// If the current character is *another* empty cell in a different rendition,
			// we restart counting and continue here
			clearCount = 0
			if cell.IsBlank() {
				blankRenditions = cell.GetRenditions()
				clearCount = 1
				frameX++
				continue
			}
		}

		// Now draw a character cell.
		// Move to the right position.
		cellWidth := cell.GetWidth()
		/*
			If we are about to print the last character in a wrapping row,
			trash the cursor position to force explicit positioning.  We do
			this because our input terminal state may have the cursor on
			the autowrap column ("column 81"), but our output terminal
			states always snap the cursor to the true last column ("column
			80"), and we want to be able to apply the diff to either, for
			verification.
		*/
		if wrapThis && frameX+cellWidth >= rowWidth {
			d.cursorX = -1
			d.cursorY = -1
		}

		// fmt.Printf("#putRow (%2d,%2d) is diff-: contents=%q, renditions=%q - write cell\n",
		// 	frameY, frameX, cell.contents, cell.renditions.SGR())

		d.appendSilentMove(out, frameY, frameX)
		d.updateRendition(out, cell.GetRenditions(), false)
		d.appendCell(out, cell)
		frameX += cellWidth
		d.cursorX += cellWidth
		if frameX >= rowWidth {
			wroteLastCell = true
		}
	}
	/* End of line. */

	// Clear or write empty cells at EOL.
	if clearCount > 0 {
		// Move to the right position.
		d.appendSilentMove(out, frameY, frameX-clearCount)
		d.updateRendition(out, blankRenditions, false)

		// pcell := newRow[frameX-clearCount]
		// fmt.Printf("#putRow (%2d,%2d) is empty, length=%d, cell=%q, rend=%q - write empty at EOL\n",
		// 	frameY, frameX-clearCount, clearCount, pcell.contents, pcell.renditions.SGR())

		canUseErase := d.hasBCE || d.currentRendition == Renditions{}
		if canUseErase && !wrapThis {
			fmt.Fprint(out, "\x1B[K") // ti.el,  Erase in Line (EL), Erase to Right (default)
		} else {
			fmt.Fprint(out, strings.Repeat(" ", clearCount))
			d.cursorX = frameX
			wroteLastCell = true
		}
	}

	if wroteLastCell && frameY < newE.nRows-1 {
		// fmt.Printf("#putRow wrapThis=%t, wroteLastCell=%t, frameY=%d\n", wrapThis, wroteLastCell, frameY)
		// To hint that a word-select should group the end of one line with the beginning of the next,
		// we let the real cursor actually wrap around in cases where it wrapped around for us.
		if wrapThis {
			// Update our cursor, and ask for wrap on the next row.
			d.cursorX = 0
			d.cursorY++
			return true
		} else {
			// Resort to CR/LF and update our cursor.
			fmt.Fprint(out, "\r\n")
			d.cursorX = 0
			d.cursorY++
			// fmt.Printf("#putRow display cursor position (%2d,%3d)\n", d.cursorY, d.cursorX)
		}
	}
	return false
}

// generate grapheme sequence to change the terminal contents.
// the generated sequence is wrote to the output stream.
func (d *Display) appendCell(out io.Writer, cell Cell) {
	// should we write space for empty contents?
	cell.printGrapheme(out)
}

// turn off cursor if necessary, use appendMove to move cursor to position.
// the generated sequence is wrote to the output stream.
func (d *Display) appendSilentMove(out io.Writer, y int, x int) {
	if d.cursorX == x && d.cursorY == y {
		return
	}
	// fmt.Printf("#appendSilentMove (%2d,%2d) move showCursorMode=%t\n", y, x, d.showCursorMode)
	// turn off cursor if necessary before moving cursor
	if d.showCursorMode {
		fmt.Fprint(out, "\x1B[?25l") // ti.civis
		d.showCursorMode = false
	}
	d.appendMove(out, y, x)
}

// generate CUP sequence to move cursor, use CR/LF/BS sequence to replace CUP if possible.
// the generated sequence is wrote to the output stream.
func (d *Display) appendMove(out io.Writer, y int, x int) {
	lastX := d.cursorX
	lastY := d.cursorY

	d.cursorX = x
	d.cursorY = y

	// fmt.Printf("#appendMove display change to (%2d,%2d)\n", d.cursorY, d.cursorX)
	// Only optimize if cursor position is known
	if lastX != -1 && lastY != -1 {
		// Can we use CR and/or LF?  They're cheap and easier to trace.
		if x == 0 && y-lastY >= 0 && y-lastY < 5 {
			// less than 5 is efficient than CUP
			if lastX != 0 {
				fmt.Fprint(out, "\r") // CR
			}
			fmt.Fprint(out, strings.Repeat("\n", y-lastY)) // LF
			return
		}
		// Backspaces are good too.
		if y == lastY && x-lastX < 0 && x-lastX > -5 {
			fmt.Fprint(out, strings.Repeat("\b", lastX-x)) // BS
			return
		}
		// CUF is shorter than CUP
		if y == lastY && x-lastX > 0 && x-lastX < 5 {
			fmt.Fprintf(out, "\x1B[%dC", x-lastX) // CUF
			return
		}
		// More optimizations are possible.
	}

	fmt.Fprintf(out, "\x1B[%d;%dH", y+1, x+1) // ti.cup
}

// if current renditions is different from parameter renditions, generate
// SGR sequence to change the cell renditions and update the current renditions.
// the generated sequence is wrote to the output stream.
func (d *Display) updateRendition(out io.Writer, r Renditions, force bool) {
	if force || d.currentRendition != r {
		// fmt.Printf("#updateRendition currentRendition=%q, new renditions=%q - update renditions\n",
		// 	d.currentRendition.SGR(), r.SGR())
		out.Write([]byte(r.SGR()))
		d.currentRendition = r
	}
}

func (d *Display) Open() string {
	var b strings.Builder
	if d.smcup != "" {
		b.WriteString(d.smcup)
	}
	// DECSET: set application cursor key mode
	fmt.Fprintf(&b, "\x1B[?1h")
	return b.String()
}

func (d *Display) Close() string {
	var b strings.Builder
	// DECRST: set ANSI cursor key mode
	fmt.Fprintf(&b, "\x1B[?1l")
	// SGR: reset character attributes, foreground color and background color
	fmt.Fprintf(&b, "\x1B[0m")
	// DECTCEM: show cursor mode
	fmt.Fprintf(&b, "\x1B[?25h")
	// disable mouse tracking mode
	fmt.Fprintf(&b, "\x1B[?1003l\x1B[?1002l\x1B[?1001l\x1B[?1000l")
	// reset to default mouse tracking encoding
	fmt.Fprintf(&b, "\x1B[?1015l\x1B[?1006l\x1B[?1005l")
	if d.rmcup != "" {
		b.WriteString(d.rmcup)
	}
	return b.String()
}

func (d *Display) Clone() *Display {
	clone := Display{}
	// clone regular data fields
	clone = *d

	// ignore logW
	// ignore terminfo
	return &clone
}
