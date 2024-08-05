// Copyright 2022 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package terminal

import (
	"fmt"
	"os"
	"strings"

	"github.com/ericwq/aprilsh/terminfo"
	"golang.org/x/exp/constraints"
)

// LookupTerminfo attempts to find a definition for the named $TERM falling
// back to attempting to parse the output from infocmp.
// func LookupTerminfo(name string) (ti *terminfo.Terminfo, e error) {
// 	ti, e = terminfo.LookupTerminfo(name)
// 	if e != nil {
// 		// ti, e = loadDynamicTerminfo(name)
// 		ti, _, e := dynamic.LoadTerminfo(name)
// 		if e != nil {
// 			return nil, e
// 		}
// 		terminfo.AddTerminfo(ti)
// 	}
//
// 	return
// }

// extract specified row from the resize screen.
// func getRowFrom(from []Cell, posY int, w int) (row []Cell) {
// 	start := posY * w
// 	end := start + w
// 	row = from[start:end]
// 	return row
// }

// func getRawRow(emu *Emulator, rowY int) (row []Cell) {
// 	start := emu.nCols * rowY
// 	end := start + emu.nCols
// 	row = emu.cf.cells[start:end]
// 	return row
// }

// return the specified row from terminal.
func getRow(emu *Emulator, posY int) (row []Cell) {
	start := emu.cf.getViewRowIdx(posY)
	end := start + emu.nCols
	row = emu.cf.cells[start:end]
	return row
}

func equalRow(a, b []Cell) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func equalSlice[T constraints.Ordered](a, b []T) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func firstScreen(oldE, newE *Emulator) bool {
	return newE.cf.scrollHead == 0 && newE.cf.scrollHead == oldE.cf.scrollHead
}

func calculateRows(oldE, newE *Emulator) int {
	// no head scroll
	if firstScreen(oldE, newE) {
		if newE.posY == oldE.posY {
			if newE.posX != oldE.posX {
				return 1
			}
			return 0
		}
		// if newE.posY == 0 { // got something like \x1b[H\x1b[2J\x1b[3J
		// 	return newE.nRows
		// }
		return newE.posY - oldE.posY + 1
	}

	// head scrolled
	if newE.cf.scrollHead > oldE.cf.scrollHead {
		// new screen head is greater than old screen head
		if newE.posY == oldE.posY && newE.posY == newE.nRows-1 {
			return newE.cf.scrollHead - oldE.cf.scrollHead + 1
		}
		gap := oldE.nRows - oldE.posY + // old screen remains rows
			(newE.cf.scrollHead - oldE.cf.scrollHead) // new screen moves rows
		return gap

	} else if newE.cf.scrollHead == oldE.cf.scrollHead {
		if newE.posY == oldE.posY && newE.posY == newE.nRows-1 {
			// just one line
			if newE.posX != oldE.posX {
				return 1
			}
			return 0
		}
		// new screen head is same as old screen head
		return newE.posY - oldE.posY
	}
	// new screen head  is smaller than old screen head (rewind)
	return oldE.cf.marginBottom - oldE.cf.scrollHead + // old screen part
		newE.cf.scrollHead + newE.posY + 1 + // new screen part
		-oldE.posY // remove old screen exist rows
}

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
	smcup string // enter and exit alternate screen mode
	rmcup string // enter and exit alternate screen mode

	// erase character is part of vt200 but not supported by tmux
	// (or by "screen" terminfo entry, which is what tmux advertises)
	hasECH       bool
	hasBCE       bool // erases result in cell filled with background color
	supportTitle bool // supports window title and icon name

	// ti           *terminfo.Terminfo

	// fields from FrameState
	// cursorX, cursorY int
	// currentRendition Renditions
	// showCursorMode   bool // mosh: cursorVisible

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
	d.supportTitle = true

	if useEnvironment {
		term := os.Getenv("TERM")

		fmt.Fprintf(os.Stderr, "query: bce, ech, smcup, rmcup\n")

		_, d.hasBCE = terminfo.LookupTerminfo("bce")
		_, d.hasECH = terminfo.LookupTerminfo("ech")
		/* Check if we can set the window title and icon name.  terminfo does not
		   have reliable information on this, so we hardcode a whitelist of
		   terminal type prefixes. */

		d.supportTitle = false
		titleTermTypes := []string{"xterm", "rxvt", "kterm", "Eterm", "alacritty", "screen", "tmux"}
		if term != "" {
			for _, tt := range titleTermTypes {
				if strings.HasPrefix(term, tt) {
					d.supportTitle = true
					break
				}
			}
		}

		d.smcup, _ = terminfo.LookupTerminfo("smcup")
		d.rmcup, _ = terminfo.LookupTerminfo("rmcup")

		fmt.Fprintf(os.Stderr, "query:\nbce=%t, ech=%t, smcup=%q, rmcup=%q\n",
			d.hasBCE, d.hasECH, d.smcup, d.rmcup)
	}

	return d, nil
}

// compare two terminals and generate mix (grapheme and control sequence) sequence
// to rebuild the new terminal from the old one.
//
// - initialized: if false, it will redraw the whole terminal, otherwise only changed part.
// - oldE: the old terminal state.
// - newE: the new terminal state.
func (d *Display) NewFrame(initialized bool, oldE, newE *Emulator) string {
	frame := new(FrameState)
	frame.cursorX = 0
	frame.cursorY = 0
	frame.currentRendition = Renditions{}
	frame.showCursorMode = oldE.showCursorMode
	frame.lastFrame = oldE
	frame.out = &strings.Builder{}
	// ti := d.ti

	// has bell been rung?
	if newE.getBellCount() != oldE.getBellCount() {
		// ti.TPuts(&b, ti.Bell)
		frame.append("\a")
	}

	// has icon name or window title changed?
	// Enhanced: has window title stack changed?
	oldWTS := oldE.windowTitleStack
	newWTS := newE.windowTitleStack
	titleAndStackBothChange := false
	if len(oldWTS) == len(newWTS) {
		if len(newWTS) == windowTitleStackMax && !equalSlice(oldWTS, newWTS) {
			// if len(newWTS) == windowTitleStackMax && !reflect.DeepEqual(oldWTS, newWTS) {
			// reach stack max with difference
			// change title first then stack
			d.titleChanged(initialized, frame, oldE, newE)
			frame.append("\x1B[22;0t")
			titleAndStackBothChange = true
		}
	} else if len(newWTS) > len(oldWTS) {
		// save title to stack
		// change title first then stack
		d.titleChanged(initialized, frame, oldE, newE)
		frame.append("\x1B[22;0t")
		titleAndStackBothChange = true
	} else {
		// restore title from stack
		// change stack first then title
		frame.append("\x1B[23;0t")
		d.titleChanged(initialized, frame, oldE, newE)
		titleAndStackBothChange = true
	}

	// has icon label or window title changed?
	if !titleAndStackBothChange {
		d.titleChanged(initialized, frame, oldE, newE)
	}

	// has clipboard changed?
	if newE.selectionData != oldE.selectionData {
		// the selectionData is in the form of "\x1B]52;%s;%s\x1B\\"
		// see hdl_osc_52() for detail
		frame.append(newE.selectionData)
	}

	// has reverse video state changed?
	if !initialized || newE.reverseVideo != oldE.reverseVideo {
		// set reverse video
		if newE.reverseVideo {
			frame.append("\x1B[?5h")
		} else {
			frame.append("\x1B[?5l")
		}
	}

	// has size changed?
	// the size of the display terminal isn't changed.
	// the size of the received terminal is changed by ApplyString()
	sizeChanged := false
	if !initialized || newE.nCols != oldE.nCols || newE.nRows != oldE.nRows {
		// TODO why reset scrolling region?
		frame.append("\x1B[r") // smgtb, reset scrolling region, reset top/bottom margin

		// ti.TPuts(&b, ti.AttrOff)  // sgr0, "\x1B[0m" turn off all attribute modes
		// ti.TPuts(&b, ti.Clear)    // clear, "\x1B[H\x1B[2J" clear screen and home cursor
		frame.append("\x1B[0m")
		frame.append("\x1B[H\x1B[2J")

		initialized = false // resize will force the initialized
		frame.cursorX = 0
		frame.cursorY = 0
		frame.currentRendition = Renditions{}
		sizeChanged = true
	} else {
		frame.cursorX = oldE.GetCursorCol()
		frame.cursorY = oldE.GetCursorRow()
		frame.currentRendition = oldE.GetRenditions()
	}

	// is cursor visibility initialized?
	if !initialized {
		// fmt.Printf("#NewFrame initialized=%t, d.showCursorMode=%t\n", initialized, d.showCursorMode)
		frame.showCursorMode = false
		// ti.TPuts(&b, ti.HideCursor) // civis, "\x1B[?25l" showCursorMode = false
		frame.append("\x1B[?25l")
	}

	// check 1049 first
	asbChanged := false
	if newE.altScreen1049 != oldE.altScreen1049 {
		asbChanged = true
		if newE.altScreen1049 {
			frame.append("\x1B[?1049h")
		} else {
			frame.append("\x1B[?1049l")
		}
		// fmt.Printf("Display.NewFrame newE.altScreen1049=%t\n", newE.altScreen1049)
	} else {
		// has the screen buffer mode changed?
		// change screen buffer is something like resize, except resize remains partial content,
		// screen buffer mode reset the whole screen.
		if !initialized || newE.altScreenBufferMode != oldE.altScreenBufferMode {
			asbChanged = true
			if newE.altScreenBufferMode {
				frame.append("\x1B[?1047h")
			} else {
				frame.append("\x1B[?1047l")
			}
		}

		// has saved cursor changed?
		// Let the target terminal decide what to save, here we just issue the control sequence.
		//
		if !initialized || newE.savedCursor_DEC.isSet != oldE.savedCursor_DEC.isSet {
			if newE.savedCursor_DEC.isSet && !oldE.savedCursor_DEC.isSet {
				frame.append("\x1B[?1048h") // decsc: VT100 use \x1B7
			} else if !newE.savedCursor_DEC.isSet && oldE.savedCursor_DEC.isSet {
				frame.append("\x1B[?1048l") // decrc: vt100 use \x1B8
			}
		}
	}

	// has the margin changed?
	if !initialized || (newE.marginTop != oldE.marginTop || newE.marginBottom != oldE.marginBottom) {
		if newE.cf.margin {
			frame.append("\x1B[%d;%dr", newE.marginTop+1, newE.marginBottom) // new margin
		} else {
			frame.append("\x1B[r") // reset margin
		}
	}

	// has the horizontal margin changed?
	if !initialized || newE.horizMarginMode != oldE.horizMarginMode {
		if newE.horizMarginMode {
			frame.append("\x1B[?69h")
			if newE.hMargin != oldE.hMargin || newE.nColsEff != oldE.nColsEff {
				// decslrm set left/right margin
				frame.append("\x1B[%d;%ds", newE.hMargin+1, newE.nColsEff)
			}
		} else {
			frame.append("\x1B[?69l")
		}
	}

	// has SCO saved cursor changed
	if !initialized || newE.savedCursor_SCO.isSet != oldE.savedCursor_SCO.isSet {
		if newE.savedCursor_SCO.isSet && !oldE.savedCursor_SCO.isSet {
			frame.append("\x1B[s") // scosc
		} else if !newE.savedCursor_SCO.isSet && oldE.savedCursor_SCO.isSet {
			frame.append("\x1B[u") // scorc
		}
	}

	d.replicateContent(initialized, oldE, newE, sizeChanged, asbChanged, frame)

	// has cursor location changed?
	if !initialized || newE.GetCursorRow() != frame.cursorY || newE.GetCursorCol() != frame.cursorX {
		// TODO using cursor position from display or cursor position from terminal?
		frame.appendMove(newE.GetCursorRow(), newE.GetCursorCol())
	}

	// has cursor visibility changed?
	// during update row, appendSilentMove() might close the cursor,
	// Here we open cursor based on the new terminal state.

	// fmt.Printf("#NewFrame newE=%t, d=%t, oldE=%t, initialized=%t\n",
	// 	newE.showCursorMode, d.showCursorMode, oldE.showCursorMode, initialized)
	if !initialized || newE.showCursorMode != frame.showCursorMode {
		if newE.showCursorMode {
			frame.append("\x1B[?25h") // cvvis
		} else {
			frame.append("\x1B[?25l") // civis
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
		frame.append("\x1B[%d q", Ps)
	}

	// has cursor color changed to default?
	if !initialized || newE.cf.cursor.color != oldE.cf.cursor.color {
		if newE.cf.cursor.color == ColorDefault {
			frame.append("\x1B]112\a")
		}
	}

	// has renditions changed?
	frame.updateRendition(newE.GetRenditions(), !initialized)

	// has bracketed paste mode changed?
	if !initialized || newE.bracketedPasteMode != oldE.bracketedPasteMode {
		if newE.bracketedPasteMode {
			frame.append("\x1B[?2004h")
		} else {
			frame.append("\x1B[?2004l")
		}
	}

	// has mouse reporting mode changed?
	if !initialized || newE.mouseTrk.mode != oldE.mouseTrk.mode {
		if newE.mouseTrk.mode == MouseTrackingMode_Disable {
			frame.append("\x1B[?1003l")
			frame.append("\x1B[?1002l")
			frame.append("\x1B[?1001l")
			frame.append("\x1B[?1000l")
		} else {
			// close old mouse reporting mode
			if oldE.mouseTrk.mode != MouseTrackingMode_Disable {
				frame.append("\x1B[?%dl", oldE.mouseTrk.mode)
			}
			// open new mouse reporting mode
			frame.append("\x1B[?%dh", newE.mouseTrk.mode)
		}
	}

	// has mouse focus mode changed?
	if !initialized || newE.mouseTrk.focusEventMode != oldE.mouseTrk.focusEventMode {
		if newE.mouseTrk.focusEventMode {
			frame.append("\x1B[?1004h")
		} else {
			frame.append("\x1B[?1004l")
		}
	}

	// has mouse encoding mode changed?
	if !initialized || newE.mouseTrk.enc != oldE.mouseTrk.enc {
		if newE.mouseTrk.enc == MouseTrackingEnc_Default {
			frame.append("\x1B[?1015l")
			frame.append("\x1B[?1006l")
			frame.append("\x1B[?1005l")
		} else {
			// close old mouse encoding mode
			if oldE.mouseTrk.enc != MouseTrackingEnc_Default {
				frame.append("\x1B[?%dl", oldE.mouseTrk.enc)
			}
			// open new mouse encoding mode
			frame.append("\x1B[?%dh", newE.mouseTrk.enc)
		}
	}

	// has auto wrap mode changed?
	if !initialized || newE.autoWrapMode != oldE.autoWrapMode {
		if newE.autoWrapMode {
			frame.append("\x1B[?7h")
		} else {
			frame.append("\x1B[?7l")
		}
	}

	// has auto newline mode changed?
	if !initialized || newE.autoNewlineMode != oldE.autoNewlineMode {
		if newE.autoNewlineMode {
			frame.append("\x1B[20h")
		} else {
			frame.append("\x1B[20l")
		}
	}

	// has keyboard action mode changed?
	if !initialized || newE.keyboardLocked != oldE.keyboardLocked {
		if newE.keyboardLocked {
			frame.append("\x1B[2h")
		} else {
			frame.append("\x1B[2l")
		}
	}

	// has insert mode changed?
	if !initialized || newE.insertMode != oldE.insertMode {
		if newE.insertMode {
			frame.append("\x1B[4h")
		} else {
			frame.append("\x1B[4l")
		}
	}

	// has local echo changed?
	if !initialized || newE.localEcho != oldE.localEcho {
		if newE.localEcho {
			frame.append("\x1B[12l") // reverse order
		} else {
			frame.append("\x1B[12h") // reverse order
		}
	}

	// has backspace send delete changed?
	if !initialized || newE.bkspSendsDel != oldE.bkspSendsDel {
		if newE.bkspSendsDel {
			frame.append("\x1B[?67l") // DECRST reverse order
		} else {
			frame.append("\x1B[?67h") // DECSET reverse order
		}
	}

	// has alt key as ESC changed?
	if !initialized || newE.altSendsEscape != oldE.altSendsEscape {
		if newE.altSendsEscape {
			frame.append("\x1B[?1036h") // DECSET
		} else {
			frame.append("\x1B[?1036l") // DECRST
		}
	}

	// has altScrollMode changed?
	if !initialized || newE.altScrollMode != oldE.altScrollMode {
		if newE.altScrollMode {
			frame.append("\x1B[?1007h") // DECSET
		} else {
			frame.append("\x1B[?1007l") // DECRST
		}
	}

	// has cursor key mode changed?
	// Note: This depends on real terminal emulator to apply cursorKeyMode.
	if !initialized || newE.cursorKeyMode != oldE.cursorKeyMode {
		switch newE.cursorKeyMode {
		case CursorKeyMode_Application:
			frame.append("\x1B[?1h") // DECSET
		case CursorKeyMode_ANSI:
			frame.append("\x1B[?1l") // DECRST
		}
	}

	// has origin mode changed?
	if !initialized || newE.originMode != oldE.originMode {
		switch newE.originMode {
		case OriginMode_ScrollingRegion:
			frame.append("\x1B[?6h") // DECSET
		case OriginMode_Absolute:
			frame.append("\x1B[?6l") // DECRST
		}
	}

	// has keypad mode changed?
	// Note: This depends on real terminal emulator to apply keypadMode.
	if !initialized || newE.keypadMode != oldE.keypadMode {
		switch newE.keypadMode {
		case KeypadMode_Application:
			frame.append("\x1B=") // DECKPAM
		case KeypadMode_Normal:
			frame.append("\x1B>") // DECKPNM
		}
	}

	// has column mode changed? the column mode is out of date.
	if !initialized || newE.colMode != oldE.colMode {
		switch newE.colMode {
		case ColMode_C132:
			frame.append("\x1B[?3h") // DECSET
		case ColMode_C80:
			frame.append("\x1B[?3l") // DECRST
		}
	}

	// has tab stop position changed?
	if !initialized || !equalSlice(newE.tabStops, oldE.tabStops) {
		// if !initialized || !reflect.DeepEqual(newE.tabStops, oldE.tabStops) {
		if len(newE.tabStops) == 0 {
			// clear tab stop if necessary
			frame.append("\x1B[3g") // TBC
		} else {
			// save the cursor position
			cursorY := frame.cursorY
			cursorX := frame.cursorX

			// rebuild the tab stop
			for _, tabStop := range newE.tabStops {
				frame.appendMove(0, tabStop) // CUP: move cursor to the tab stop position
				frame.append("\x1BH")        // HTS: set current position as tab stop
			}

			// restore the cursor position
			frame.appendMove(cursorY, cursorX)
		}
	}

	// has conformance level changed?
	if !initialized || newE.compatLevel != oldE.compatLevel {
		switch newE.compatLevel {
		case CompatLevel_VT52:
			frame.append("\x1B[?2l") // DECSET
		case CompatLevel_VT100:
			frame.append("\x1B[61\"p") // DECSCL
		case CompatLevel_VT400:
			frame.append("\x1B[64\"p") // DECSCL
		}
	}

	// has key modifier encoding level changed?
	// Note: This depends on real terminal emulator to apply modifyOtherKeys.
	if !initialized || newE.modifyOtherKeys != oldE.modifyOtherKeys {
		// the possible value for modifyOtherKeys is [0,1,2]
		// fmt.Printf("#NewFrame modifyOtherKeys newE=%d, oldE=%d, initialized=%t\n",
		// 	newE.modifyOtherKeys, oldE.modifyOtherKeys, initialized)
		frame.append("\x1B[>4;%dm", newE.modifyOtherKeys)
	}

	// TODO do we need to consider cursor selection area.
	return frame.output()
}

// func (d *Display) printFramebufferInfo(oldE, newE *Emulator) {
// 	util.Log.Debug("replicateContent",
// 		"columns   [E]:", fmt.Sprintf("%3d vs. %3d", newE.nCols, oldE.nCols))
// 	util.Log.Debug("replicateContent",
// 		"rows      [E]:", fmt.Sprintf("%3d vs. %3d", newE.nRows, oldE.nRows))
// 	util.Log.Debug("replicateContent",
// 		"position  [E]:", fmt.Sprintf("(%3d,%3d) vs. (%3d,%3d)", newE.posY, newE.posX, oldE.posY, oldE.posX))
// 	util.Log.Debug("replicateContent",
// 		"saveLines    :", fmt.Sprintf("%3d vs. %3d", newE.cf.saveLines, oldE.cf.saveLines))
// 	util.Log.Debug("replicateContent",
// 		"scrollHead   :", fmt.Sprintf("%3d vs. %3d", newE.cf.scrollHead, oldE.cf.scrollHead))
// 	util.Log.Debug("replicateContent",
// 		"marginTop    :", fmt.Sprintf("%3d vs. %3d", newE.cf.marginTop, oldE.cf.marginTop))
// 	util.Log.Debug("replicateContent",
// 		"marginBottom :", fmt.Sprintf("%3d vs. %3d", newE.cf.marginBottom, oldE.cf.marginBottom))
// 	util.Log.Debug("replicateContent",
// 		"historyRows  :", fmt.Sprintf("%3d vs. %3d", newE.cf.historyRows, oldE.cf.historyRows))
// 	util.Log.Debug("replicateContent",
// 		"viewOffset   :", fmt.Sprintf("%3d vs. %3d", newE.cf.viewOffset, oldE.cf.viewOffset))
// 	util.Log.Debug("replicateContent",
// 		"cursor       :", fmt.Sprintf("(%3d,%3d) vs. (%3d,%3d)",
// 			newE.cf.cursor.posY, newE.cf.cursor.posX, oldE.cf.cursor.posY, oldE.cf.cursor.posX))
// 	util.Log.Debug("replicateContent",
// 		"damage       :", fmt.Sprintf("(%3d,%3d) vs. (%3d,%3d)",
// 			newE.cf.damage.start, newE.cf.damage.end, oldE.cf.damage.start, oldE.cf.damage.end))
// }

// https://tomscii.sig7.se/zutty/doc/HACKING.html#Frame
func (d *Display) replicateContent(initialized bool, oldE, newE *Emulator, sizeChanged bool,
	asbChanged bool, frame *FrameState,
) {
	// d.printFramebufferInfo(oldE, newE)

	// mark := "#start"
	// prefix := frame.output()

	// util.Log.Debug("replicateContent",
	// 	"mark", mark,
	// 	"fs.cursor", fmt.Sprintf("(%02d,%02d)", frame.cursorY, frame.cursorX),
	// 	"altScreenBufferMode", newE.altScreenBufferMode,
	// 	"diff1", prefix)
	// util.Log.Debug("replicateContent",
	// 	"newSize", len(newE.cf.cells),
	// 	"oldSize", len(oldE.cf.cells),
	// 	"newMarginBottom", newE.cf.marginBottom,
	// 	"oldMarginBottom", oldE.cf.marginBottom)

	var countRows int // stream mode replicate range
	// TODO consider remove the scroll head check
	// if !newE.altScreenBufferMode && newE.cf.scrollHead > 0 {
	if !newE.altScreenBufferMode {
		var oldRow []Cell
		var newRow []Cell
		blankRow := make([]Cell, oldE.nCols)

		// mark = "stream"
		rawY := oldE.cf.getPhysicalRow(oldE.posY) // start row, it's physical row
		frameY := oldE.posY                       // screen row
		countRows = calculateRows(oldE, newE)

		// util.Log.Debug("replicateContent",
		// 	"oldHead", oldE.cf.scrollHead,
		// 	"newHead", newE.cf.scrollHead,
		// 	"oldY", oldE.posY,
		// 	"newY", newE.posY,
		// 	"oldX", oldE.posX,
		// 	"newX", newE.posX)

		// pre := frame.output()

		wrap := false
		for i := 0; i < countRows; i++ {
			// oldRow = blankRow
			if countRows == 1 {
				oldRow = oldE.cf.getRow(rawY)
			} else {
				oldRow = blankRow
			}
			newRow = newE.cf.getRow(rawY)
			wrap = d.putRow2(initialized, frame, newE, newRow, frameY, oldRow, wrap)

			// util.Log.Debug("replicateContent","old", outputRow(oldRow, rawY, oldE.nCols))
			// util.Log.Debug("replicateContent","new", outputRow(newRow, rawY, newE.nCols))
			// util.Log.Debug("replicateContent",
			// 	"fs.cursor", fmt.Sprintf("(%02d,%02d)", frame.cursorY, frame.cursorX),
			// 	"rawY", rawY,
			// 	"frameY", frameY,
			// 	"count", i,
			// 	"output", strings.TrimPrefix(frame.output(), pre))

			// pre = frame.output()

			// wrap around the end of the scrolling area
			rawY += 1
			if rawY == newE.cf.marginBottom {
				rawY = newE.cf.marginTop
			}
			// rawY = oldE.cf.getPhysicalRow(rawY + 1)

			frameY += 1
			// if frameY >= newE.GetHeight() {
			// 	frameY = newE.GetHeight()
			// 	frame.cursorY = frameY - 1
			// }
		}
	} else {
		// mark = "screen"
		d.replicateASB(initialized, oldE, newE, sizeChanged, asbChanged, frame)
	}

	// util.Log.Debug("replicateContent","diff2", strings.TrimPrefix(frame.output(), prefix))
	// util.Log.Debug("replicateContent",
	// 	"mark", mark,
	// 	"fs.cursor", fmt.Sprintf("(%02d,%02d)", frame.cursorY, frame.cursorX),
	// 	"oldHead", oldE.cf.scrollHead,
	// 	"newHead", newE.cf.scrollHead,
	// 	"lastRows", newE.lastRows,
	// 	"countRows", countRows)
}

// replicate alternate screen buffer
func (d *Display) replicateASB(initialized bool, oldE, newE *Emulator, _ bool,
	asbChanged bool, frame *FrameState,
) {
	// util.Log.Debug("replicateASB", "asbChanged", asbChanged, "sizeChanged", sizeChanged)

	/*
		var resizeScreen []Cell

		if asbChanged {
			// old is normal screen buffer, new is alternate screen buffer
			resizeScreen = make([]Cell, newE.nRows*newE.nCols)
			newE.cf.unwrapCellStorage()
		} else {
			// both screens are alternate screen buffer
			resizeScreen = oldE.cf.cells
			// newE.cf.unwrapCellStorage()
		}

		if newE.nCols != oldE.nCols || newE.nRows != oldE.nRows {
			util.Log.Warn("replicateASB","size", "changed")
			// TODO resize processing
			// resize and copy old screen
			// we copy the old screen to avoid changing the same part.

			// prepare place for the old screen
			// oldScreen := make([]Cell, oldE.nCols*oldE.nRows)
			// oldE.cf.fullCopyCells(oldScreen)

			// prepare place for the resized screen
			resizeScreen = make([]Cell, newE.nCols*newE.nRows)

			nCopyCols := Min(oldE.nCols, newE.nCols) // minimal column length
			nCopyRows := Min(oldE.nRows, newE.nRows) // minimal row length

			// copy the old screen to the new place
			for pY := 0; pY < nCopyRows; pY++ {
				srcStartIdx := pY * nCopyCols
				srcEndIdx := srcStartIdx + nCopyCols
				dstStartIdx := pY * nCopyCols
				copy(resizeScreen[dstStartIdx:], oldE.cf.cells[srcStartIdx:srcEndIdx])
			}
			// oldScreen = nil
			// resize and copy old screen
		}
	*/
	var frameY int
	var oldRow []Cell
	var newRow []Cell

	/*
		var linesScrolled int
		var scrollHeight int
		// shortcut -- has display moved up(text up, window down) by a certain number of lines?
		// NOTE: not availble for alternate screen buffer changed.
		// if initialized && !asbChanged && !newE.altScreenBufferMode {
		if initialized {

			for row := 0; row < newE.GetHeight(); row++ {
				newRow = getRow(newE, 0)
				oldRow = getRowFrom(resizeScreen, row, newE.nCols)
				// oldRow = getRow(oldE, row)

				if equalRow(newRow, oldRow) {
					// fmt.Printf("new screen row 0 is the same as old screen row %d\n", row)

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
						// oldRow = getRow(oldE, regionHeight+linesScrolled)
						if equalRow(newRow, oldRow) {
							// fmt.Printf("new screen row %d is the same as old screen row %d\n",
							// 	regionHeight, regionHeight+linesScrolled)
							scrollHeight = regionHeight + 1
						} else {
							break
						}
					}
					// fmt.Printf("new screen has %d same rows with the old screen, start from %d\n",
					// 	scrollHeight, linesScrolled)

					break
				}
			}

			if scrollHeight > 0 {
				frameY = scrollHeight

				if linesScrolled > 0 {
					// reset the renditions
					frame.updateRendition(Renditions{}, true)

					topMargin := 0
					bottomMargin := topMargin + linesScrolled + scrollHeight - 1
					// fmt.Printf("#NewFrame scrollHeight=%2d, linesScrolled=%2d, frameY=%2d, bottomMargin=%2d\n",
					// 	scrollHeight, linesScrolled, frameY, bottomMargin)

					// Common case:  if we're already on the bottom line and we're scrolling the whole
					// creen, just do a CR and LFs.
					if scrollHeight+linesScrolled == newE.GetHeight() && frame.cursorY+1 == newE.GetHeight() {
						frame.append("\r")
						frame.append("\x1B[%dS", linesScrolled)
						frame.cursorX = 0
					} else {
						// set scrolling region
						frame.append("\x1B[%d;%dr", topMargin+1, bottomMargin+1)

						// go to bottom of scrolling region
						frame.cursorY = -1
						frame.cursorX = -1
						frame.appendSilentMove(bottomMargin, 0)

						// scroll text up by <linesScrolled>
						frame.append("\x1B[%dS", linesScrolled)

						// reset scrolling region
						frame.append("\x1B[r")

						// invalidate cursor position after unsetting scrolling region
						frame.cursorY = -1
						frame.cursorX = -1
					}

					// // Now we need a proper blank row.
					// blankRow := make([]Cell, newE.nCols)
					// for i := range blankRow {
					// 	// set both contents and renditions
					// 	blankRow[i] = newE.attrs
					// }
					//
					// // do the move in our local new screen
					// for i := topMargin; i <= bottomMargin; i++ {
					// 	dstStart := i * newE.nCols
					//
					// 	if i+linesScrolled <= bottomMargin {
					// 		copy(resizeScreen[dstStart:], getRow(oldE, linesScrolled+i))
					// 		// copy(resizeScreen[dstStart:], getRowFrom(resizeScreen, linesScrolled+i, newE.nCols))
					// 	} else {
					// 		copy(resizeScreen[dstStart:], blankRow[:])
					// 		// fmt.Printf("row %d is blank\n", i)
					// 	}
					// }
					//
				}
			}
		}
	*/
	// util.Log.Debug("replicateASB",
	// 	"newHead", newE.cf.scrollHead,
	// 	"oldHead", oldE.cf.scrollHead,
	// 	"new.cursor", fmt.Sprintf("(%02d,%02d)", newE.posY, newE.posX),
	// 	"old.cursor", fmt.Sprintf("(%02d,%02d)", oldE.posY, oldE.posX),
	// 	"fs.cursor", fmt.Sprintf("(%02d,%02d)", frame.cursorY, frame.cursorX),
	// 	"frameY", frameY)

	// pre := frame.output()

	/*
		// Now update the display, row by row
		wrap := false
		// for i := 0; i < countRows; i++ {
		for ; frameY < newE.GetHeight(); frameY++ {
			// for i := 0; i < newE.GetHeight(); i++ {
			oldRow = getRowFrom(resizeScreen, frameY, oldE.nCols)
			// newRow := getRow(newE, frameY)
			newRow = newE.getRowAt(frameY)
			wrap = d.putRow2(initialized, frame, newE, newRow, frameY, oldRow, wrap)
			// fmt.Printf("#NewFrame frameY=%2d, seq=%q\n", frameY, strings.Replace(b.String(), seq, "", 1))
			// seq = b.String()

			util.Log.Debug("replicateASB","old", outputRow(oldRow, frameY, oldE.nCols))
			util.Log.Debug("replicateASB","new", outputRow(newRow, frameY, newE.nCols))
			util.Log.Debug("replicateASB",
				"fs.cursor", fmt.Sprintf("(%02d,%02d)", frame.cursorY, frame.cursorX),
				"frameY", frameY,
				"output", strings.TrimPrefix(frame.output(), pre))

			pre = frame.output()

			// frameY++
		}
	*/

	blankRow := make([]Cell, oldE.nCols)
	if asbChanged {
		frameY = oldE.nRows - 1
	} else {
		// frameY = oldE.posY
		frameY = 0
	}
	wrap := false
	for i := 0; i < newE.nRows; i++ {
		if asbChanged {
			oldRow = blankRow
		} else {
			oldRow = oldE.getRowAt(i)
		}
		newRow = newE.getRowAt(i)
		wrap = d.putRow2(initialized, frame, newE, newRow, frameY, oldRow, wrap)

		// util.Log.Debug("replicateASB","old", outputRow(oldRow, frameY, oldE.nCols))
		// util.Log.Debug("replicateASB","new", outputRow(newRow, frameY, newE.nCols))
		// util.Log.Debug("replicateASB",
		// 	"fs.cursor", fmt.Sprintf("(%02d,%02d)", frame.cursorY, frame.cursorX),
		// 	"frameY", frameY,
		// 	"row", i,
		// 	"output", strings.TrimPrefix(frame.output(), pre))

		// pre = frame.output()

		frameY++
	}
}

/*
func (d *Display) putRow(initialized bool, frame *FrameState,
	newE *Emulator, newRow []Cell, frameY int, oldRow []Cell, wrap bool) bool {
	frameX := 0
	// newRow := getRow(newE, frameY)

	// If we're forced to write the first column because of wrap, go ahead and do so.
	if wrap {
		cell := newRow[0]
		frame.updateRendition(cell.GetRenditions(), false)
		frame.appendCell(cell)

		// fmt.Printf("#putRow (%2d,%2d) is wrap-: contents=%q, renditions=%q - write wrap cell\n",
		// 	frameY, frameX, cell.contents, cell.renditions.SGR())

		frameX += cell.GetWidth()
		frame.cursorX += cell.GetWidth()
	}

	// If rows are the same object, we don't need to do anything at all.
	// if initialized && reflect.DeepEqual(newRow, oldRow) {
	if initialized && equalRow(newRow, oldRow) {
		// fmt.Printf("same row %d\n", frameY)
		return false
	}

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
			frame.updateRendition(cell.renditions, false)
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
			frame.appendSilentMove(frameY, frameX-clearCount)
			frame.updateRendition(blankRenditions, false)

			// pcell := newRow[frameX-clearCount]
			// fmt.Printf("#putRow (%2d,%2d) is empty, length=%d, cell=%q, rend=%q - write empty\n",
			// 	frameY, frameX-clearCount, clearCount, pcell.contents, pcell.renditions.SGR())

			canUseErase := d.hasBCE || frame.currentRendition == Renditions{}
			if canUseErase && d.hasECH && clearCount > 4 {
				// space is more efficient than ECH, if clearCount > 4
				frame.append("\x1B[%dX", clearCount)
			} else {
				// fmt.Printf("#putRow space=%q\n", strings.Repeat(" ", clearCount))
				frame.append(strings.Repeat(" ", clearCount))
				frame.cursorX = frameX
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

		// If we are about to print the last character in a wrapping row,
		// trash the cursor position to force explicit positioning.  We do
		// this because our input terminal state may have the cursor on
		// the autowrap column ("column 81"), but our output terminal
		// states always snap the cursor to the true last column ("column
		// 80"), and we want to be able to apply the diff to either, for
		// verification.

		if wrapThis && frameX+cellWidth >= rowWidth {
			frame.cursorX = -1
			frame.cursorY = -1
		}

		// fmt.Printf("#putRow (%2d,%2d) is diff-: contents=%q, renditions=%q - write cell\n",
		// 	frameY, frameX, cell.contents, cell.renditions.SGR())

		frame.appendSilentMove(frameY, frameX)
		frame.updateRendition(cell.GetRenditions(), false)
		frame.appendCell(cell)
		frameX += cellWidth
		frame.cursorX += cellWidth
		if frameX >= rowWidth {
			wroteLastCell = true
		}
	}
	// End of line.

	// Clear or write empty cells at EOL.
	if clearCount > 0 {
		// Move to the right position.
		frame.appendSilentMove(frameY, frameX-clearCount)
		frame.updateRendition(blankRenditions, false)

		// pcell := newRow[frameX-clearCount]
		// fmt.Printf("#putRow (%2d,%2d) is empty, length=%d, cell=%q, rend=%q - write empty at EOL\n",
		// 	frameY, frameX-clearCount, clearCount, pcell.contents, pcell.renditions.SGR())

		canUseErase := d.hasBCE || frame.currentRendition == Renditions{}
		if canUseErase && !wrapThis {
			frame.append("\x1B[K") // ti.el,  Erase in Line (EL), Erase to Right (default)
		} else {
			frame.append(strings.Repeat(" ", clearCount))
			frame.cursorX = frameX
			wroteLastCell = true
		}
	}

	if wroteLastCell && frameY < newE.nRows-1 {
		// fmt.Printf("#putRow wrapThis=%t, wroteLastCell=%t, frameY=%d\n", wrapThis, wroteLastCell, frameY)
		// To hint that a word-select should group the end of one line with the beginning of the next,
		// we let the real cursor actually wrap around in cases where it wrapped around for us.
		if wrapThis {
			// Update our cursor, and ask for wrap on the next row.
			frame.cursorX = 0
			frame.cursorY++
			return true
		} else {
			// Resort to CR/LF and update our cursor.
			frame.append("\r\n")
			frame.cursorX = 0
			frame.cursorY++
			// fmt.Printf("#putRow display cursor position (%2d,%3d)\n", d.cursorY, d.cursorX)
		}
	}
	return false
}
*/

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
func (d *Display) putRow2(initialized bool, frame *FrameState,
	newE *Emulator, newRow []Cell, frameY int, oldRow []Cell, wrap bool,
) bool {
	frameX := 0
	// newRow := newE.cf.getRow(rawY)

	// If we're forced to write the first column because of wrap, go ahead and do so.
	if wrap {
		cell := newRow[0]
		frame.updateRendition(cell.GetRenditions(), false)
		frame.appendCell(cell)

		// fmt.Printf("#putRow (%2d,%2d) is wrap-: contents=%q, renditions=%q - write wrap cell\n",
		// 	frameY, frameX, cell.contents, cell.renditions.SGR())

		frameX += cell.GetWidth()
		frame.cursorX += cell.GetWidth()
	}

	// If rows are the same object, we don't need to do anything at all.
	if initialized && equalRow(newRow, oldRow) {
		// fmt.Printf("same row %d\n", frameY)
		return false
	}

	// // shortcut for blank row.
	// var blankRow []Cell = make([]Cell, newE.nCols)
	// if initialized && equalRow(newRow, blankRow) {
	// 	util.Log.Debug("putRow2 blank row")
	// 	return false
	// }

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
			frame.updateRendition(cell.renditions, false)
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
			frame.appendSilentMove(frameY, frameX-clearCount)
			frame.updateRendition(blankRenditions, false)

			canUseErase := d.hasBCE || frame.currentRendition == Renditions{}
			if canUseErase && d.hasECH && clearCount > 4 {
				// space is more efficient than ECH, if clearCount > 4
				frame.append("\x1B[%dX", clearCount)
			} else {
				// fmt.Printf("#putRow space=%q\n", strings.Repeat(" ", clearCount))
				frame.append(strings.Repeat(" ", clearCount))
				frame.cursorX = frameX
			}

			// pcell := newRow[frameX-clearCount]
			// util.Log.Debug(fmt.Sprintf("putRow2 (%2d,%2d)", frameY, frameX-clearCount),
			// 	"fs.cursor", fmt.Sprintf("(%2d,%2d)", frame.cursorY, frame.cursorX),
			// 	"cell", 0x20,
			// 	"rend", pcell.renditions.SGR(),
			// 	"out", frame.output(),
			// 	"length", clearCount)

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
			frame.cursorX = -1
			frame.cursorY = -1
		}

		frame.appendSilentMove(frameY, frameX)
		frame.updateRendition(cell.GetRenditions(), false)
		frame.appendCell(cell)

		// util.Log.Debug(fmt.Sprintf("putRow2 (%2d,%2d)", frameY, frameX),
		// 	"fs.cursor", fmt.Sprintf("(%2d,%2d)", frame.cursorY, frame.cursorX),
		// 	"cell", cell.contents,
		// 	"rend", cell.renditions.SGR(),
		// 	"out", frame.output())

		frameX += cellWidth
		frame.cursorX += cellWidth
		if frameX >= rowWidth {
			wroteLastCell = true
		}
	}
	/* End of line. */

	// Clear or write empty cells at EOL.
	if clearCount > 0 {
		// Move to the right position.
		frame.appendSilentMove(frameY, frameX-clearCount)
		frame.updateRendition(blankRenditions, false)

		canUseErase := d.hasBCE || frame.currentRendition == Renditions{}
		if canUseErase && !wrapThis {
			frame.append("\x1B[K") // ti.el,  Erase in Line (EL), Erase to Right (default)
		} else {
			frame.append(strings.Repeat(" ", clearCount))
			frame.cursorX = frameX
			wroteLastCell = true
		}

		// pcell := newRow[frameX-clearCount]
		// util.Log.Debug(fmt.Sprintf("putRow2 (%2d,%2d)", frameY, frameX-clearCount),
		// 	"fs.cursor", fmt.Sprintf("(%2d,%2d)", frame.cursorY, frame.cursorX),
		// 	"cell", 0x20,
		// 	"rend", pcell.renditions.SGR(),
		// 	"out", frame.output(),
		// 	"length", clearCount)
	}

	// util.Log.Debug("putRow2",
	// 	"wroteLastCell", wroteLastCell,
	// 	"frameY", frameY)

	if wroteLastCell && frameY < newE.nRows-1 {
		// fmt.Printf("#putRow wrapThis=%t, wroteLastCell=%t, frameY=%d\n", wrapThis, wroteLastCell, frameY)
		// To hint that a word-select should group the end of one line with the beginning of the next,
		// we let the real cursor actually wrap around in cases where it wrapped around for us.
		if wrapThis {
			// Update our cursor, and ask for wrap on the next row.
			frame.cursorX = 0
			frame.cursorY++
			return true
		} else {
			// Resort to CR/LF and update our cursor.
			frame.append("\r\n")
			frame.cursorX = 0
			frame.cursorY++
			// fmt.Printf("#putRow display cursor position (%2d,%3d)\n", d.cursorY, d.cursorX)
		}
	}
	return false
}

func (d *Display) titleChanged(initialized bool, frame *FrameState, oldE, newE *Emulator) {
	// has icon label or window title changed?
	if d.supportTitle && newE.isTitleInitialized() && (!initialized ||
		newE.GetIconLabel() != oldE.GetIconLabel() || newE.GetWindowTitle() != oldE.GetWindowTitle()) {
		if newE.GetIconLabel() == newE.GetWindowTitle() {
			// write combined Icon label and Window Title
			frame.append("\x1B]0;%s\x07", newE.GetWindowTitle())
			// ST is more correct, but BEL is more widely supported
		} else {
			// write Icon label
			if newE.GetIconLabel() != "" {
				frame.append("\x1B]1;%s\x07", newE.GetIconLabel())
			}

			// write Window Title
			if newE.GetWindowTitle() != "" {
				frame.append("\x1B]2;%s\x07", newE.GetWindowTitle())
			}
		}
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

type FrameState struct {
	lastFrame        *Emulator
	out              *strings.Builder
	currentRendition Renditions
	cursorX          int
	cursorY          int
	showCursorMode   bool // mosh: cursorVisible
}

func (fs *FrameState) append(x string, v ...any) {
	if len(v) == 0 {
		fs.out.WriteString(x)
		// fmt.Fprint(fs.out, x)
	} else {
		// fs.out.WriteString(fmt.Sprintf(x, v...))
		fmt.Fprintf(fs.out, x, v...)
	}
}

// generate grapheme sequence to change the terminal contents.
// the generated sequence is wrote to the output stream.
func (fs *FrameState) appendCell(cell Cell) {
	// should we write space for empty contents?
	cell.printGrapheme(fs.out)
}

// turn off cursor if necessary, use appendMove to move cursor to position.
// the generated sequence is wrote to the output stream.
func (fs *FrameState) appendSilentMove(y int, x int) {
	if fs.cursorX == x && fs.cursorY == y {
		return
	}
	// fmt.Printf("#appendSilentMove (%2d,%2d) move showCursorMode=%t\n", y, x, d.showCursorMode)
	// turn off cursor if necessary before moving cursor
	if fs.showCursorMode {
		fs.append("\x1B[?25l") // ti.civis
		fs.showCursorMode = false
	}
	fs.appendMove(y, x)
}

// generate CUP sequence to move cursor, use CR/LF/BS sequence to replace CUP if possible.
// the generated sequence is wrote to the output stream.
func (fs *FrameState) appendMove(y int, x int) {
	lastX := fs.cursorX
	lastY := fs.cursorY

	fs.cursorX = x
	fs.cursorY = y

	// util.Log.Debug("appendMove","lastY", lastY,"y", y)

	// Only optimize if cursor position is known
	if lastX != -1 && lastY != -1 {
		// Can we use CR and/or LF?  They're cheap and easier to trace.
		if x == 0 && y-lastY >= 0 && y-lastY < 5 {
			// less than 5 is efficient than CUP
			if lastX != 0 {
				fs.append("\r") // CR
			}
			fs.append(strings.Repeat("\n", y-lastY)) // LF
			return
		}
		// Backspaces are good too.
		if y == lastY && x-lastX < 0 && x-lastX > -5 {
			fs.append(strings.Repeat("\b", lastX-x)) // BS
			return
		}
		// CUF is shorter than CUP
		// if y == lastY && x-lastX > 0 && x-lastX < 5 {
		if y == lastY {
			fs.append("\x1B[%dC", x-lastX) // CUF
			// fs.append(strings.Repeat(" ", x-lastX)) // use ' ' to replace CUF
			return
		}
		// More optimizations are possible.
	}

	fs.append("\x1B[%d;%dH", y+1, x+1) // ti.cup
}

// if current renditions is different from parameter renditions, generate
// SGR sequence to change the cell renditions and update the current renditions.
// the generated sequence is wrote to the output stream.
func (fs *FrameState) updateRendition(r Renditions, force bool) {
	if force || fs.currentRendition != r {
		// fmt.Printf("#updateRendition currentRendition=%q, new renditions=%q - update renditions\n",
		// 	d.currentRendition.SGR(), r.SGR())
		fs.append(r.SGR())
		fs.currentRendition = r
	}
}

func (fs *FrameState) output() string {
	return fs.out.String()
}
