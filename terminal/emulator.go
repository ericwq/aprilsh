// Copyright 2022 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package terminal

import (
	"fmt"
	"maps"
	"strings"

	"github.com/ericwq/aprilsh/util"
)

// type VtModifier uint8
// TODO consider add InputSpec, InputSpecTable: mapping of a certain VtKey to a sequence of input characters
// TODO consider add RefreshHandlerFn, OscHandlerFn, BellHandlerFn

// // Terminal state - N.B.: keep resetTerminal () in sync with this!
type Emulator struct {
	parser              *Parser
	links               *linkSet
	cf                  *Framebuffer     // replicated by NewFrame(), current frame buffer
	selectionStore      map[rune]string  // local storage buffer for selection data in sequence OSC 52
	caps                map[int]string   // client terminal capability
	savedCursor_DEC     *SavedCursor_DEC // replicated by NewFrame(),
	windowTitle         string           // replicated by NewFrame()
	iconLabel           string           // replicated by NewFrame()
	selectionData       string           // replicated by NewFrame(), store the selection data for OSC 52
	terminalToHost      strings.Builder  // used for terminal write back
	tabStops            []int            // replicated by NewFrame(), tab stop positions
	windowTitleStack    []string         // for XTWINOPS
	charsetState        CharsetState     // for forward compatibility
	attrs               Cell             // replicated by NewFrame() partially, prototype cell with current attributes
	savedCursor_DEC_alt SavedCursor_DEC
	savedCursor_DEC_pri SavedCursor_DEC

	frame_alt Framebuffer // alternate screen buffer
	frame_pri Framebuffer // normal screen buffer

	savedCursor_SCO SavedCursor_SCO    // replicated by NewFrame(), SCO console cursor state
	bg              Color              // TODO: should we keep this?
	lastRows        int                // last processed rows
	bellCount       int                // replicated by NewFrame()
	posX            int                // replicated by NewFrame(), current cursor cols position (on-screen)
	posY            int                // replicated by NewFrame(), current cursor rows position (on-screen)
	nRows           int                // replicated by NewFrame(),
	nCols           int                // replicated by NewFrame(),
	marginTop       int                // replicated by NewFrame(), current margin top (screen view)
	marginBottom    int                // replicated by NewFrame(), current margin bottom (screen view)
	user            UserInput          // TODO consider how to change it.
	hMargin         int                // replicated by NewFrame(), left margins
	modifyOtherKeys uint               // replicated by NewFrame(),
	fg              Color              // TODO: should we keep this?
	nColsEff        int                // replicated by NewFrame(), right margins
	mouseTrk        MouseTrackingState // replicated by NewFrame()

	altScreenBufferMode bool          // replicated by NewFrame(), , Alternate Screen Buffer
	horizMarginMode     bool          // replicated by NewFrame(), left and right margins support
	cursorKeyMode       CursorKeyMode // replicated by NewFrame(), default:ANSI: Application(true), ANSI(false)
	keypadMode          KeypadMode    // replicated by NewFrame(), default:Normal
	originMode          OriginMode    // replicated by NewFrame(), default:Absolute, ScrollingRegion(true), Absolute(false)
	colMode             ColMode       // replicated by NewFrame(), default:80, column mode 80 or 132, just for compatibility
	showCursorMode      bool          // replicated by NewFrame(), default true, ds.cursor_visible

	hasFocus           bool               // default true
	reverseVideo       bool               // replicated by NewFrame(),
	altSendsEscape     bool               // replicated by NewFrame(), default true
	altScreen1049      bool               // DECSET and DECRST 1049, default false
	compatLevel        CompatibilityLevel // replicated by NewFrame(), VT52, VT100, VT400. default:VT400
	altScrollMode      bool               // replicated by NewFrame(),
	bracketedPasteMode bool               // replicated by NewFrame(),
	localEcho          bool               // replicated by NewFrame(),
	bkspSendsDel       bool               // replicated by NewFrame(), default:true, backspace send delete
	insertMode         bool               // replicated by NewFrame(),
	keyboardLocked     bool               // replicated by NewFrame(),
	titleInitialized   bool               // replicated by NewFrame()
	autoNewlineMode    bool               // replicated by NewFrame(), LNM
	autoWrapMode       bool               // replicated by NewFrame(), default:true
	lastCol            bool
	syncOutpuMode      bool
}

func NewEmulator3(nCols, nRows, saveLines int) *Emulator {
	// TODO makePalette256 (palette256);

	emu := &Emulator{}
	emu.parser = NewParser()
	emu.frame_pri, emu.marginTop, emu.marginBottom = NewFramebuffer3(nCols, nRows, saveLines)
	emu.cf = &emu.frame_pri
	emu.frame_alt = NewFramebuffer2(1, 1)

	emu.nCols = nCols
	emu.nRows = nRows

	emu.hasFocus = true
	emu.showCursorMode = true
	emu.altScreenBufferMode = false
	emu.autoWrapMode = true
	emu.autoNewlineMode = false
	emu.keyboardLocked = false
	emu.insertMode = false
	emu.bkspSendsDel = true
	emu.localEcho = false
	emu.bracketedPasteMode = false

	emu.horizMarginMode = false
	emu.nColsEff = emu.nCols
	emu.hMargin = 0

	emu.posX = 0
	emu.posY = 0
	emu.lastCol = false

	emu.attrs.contents = " "
	emu.attrs.renditions = Renditions{}

	emu.fg = emu.attrs.renditions.fgColor
	emu.bg = emu.attrs.renditions.bgColor

	emu.savedCursor_DEC_pri = newSavedCursor_DEC()
	emu.savedCursor_DEC_alt = newSavedCursor_DEC()
	emu.savedCursor_DEC = &emu.savedCursor_DEC_pri
	emu.initSelectionStore()
	emu.caps = make(map[int]string)

	emu.resetTerminal()
	emu.windowTitleStack = make([]string, 0)
	return emu
}

func resetCharsetState(charsetState *CharsetState) {
	// we don't use vt100 charset by default
	charsetState.vtMode = false

	// default nil will fall to UTF-8
	charsetState.g[0] = nil
	charsetState.g[1] = nil
	charsetState.g[2] = nil
	charsetState.g[3] = nil

	// Locking shift states (index into g[]):
	charsetState.gl = 0 // G0 in GL
	charsetState.gr = 2 // G2 in GR

	// Single shift state (0 if none active):
	// 0 - not active; 2: G2 in GL; 3: G3 in GL
	charsetState.ss = 0
}

func (emu *Emulator) resize(nCols, nRows int) {
	if emu.nCols == nCols && emu.nRows == nRows {
		return
	}
	// util.Log.Debug("Emulator.resize","cols", nCols,"rows", nRows)

	emu.hideCursor()

	if emu.altScreenBufferMode {
		// create a new frame buffer
		emu.frame_alt, emu.marginTop, emu.marginBottom = NewFramebuffer3(nCols, nRows, 0)
	} else {
		// adjust the cursor position if the nRow shrinked
		if nRows < emu.posY+1 {
			nScroll := emu.nRows - nRows
			emu.cf.scrollUp(nScroll)
			emu.posY -= nScroll
		}

		emu.marginTop, emu.marginBottom = emu.frame_pri.resize(nCols, nRows)

		// adjust the cursor position if the nRow expanded
		if emu.nRows < nRows {
			nScroll := min(nRows-emu.nRows, emu.cf.getHistroryRows())
			emu.cf.scrollDown(nScroll)
			emu.posY += nScroll
		}

		emu.frame_alt.freeCells()
	}

	emu.nCols = nCols
	emu.nRows = nRows

	if emu.horizMarginMode {
		emu.nColsEff = min(emu.nColsEff, emu.nCols)
		emu.hMargin = max(0, min(emu.hMargin, emu.nColsEff-2))
	} else {
		emu.nColsEff = emu.nCols
		emu.hMargin = 0
	}

	emu.normalizeCursorPos()
	emu.showCursor()
	emu.links = newLinks()

	// TODO pty resize
}

// hide the implementation of write back
func (emu *Emulator) writePty(resp string) {
	emu.terminalToHost.WriteString(resp)
}

// TODO consider to add pageUp, pageDown, mouseWheelUp, mouseWheelDown

func (emu *Emulator) setHasFocus(hasFocus bool) {
	emu.hasFocus = hasFocus
	emu.showCursor()
	// redraw()?
}

// encapsulate selection data according to bracketedPasteMode.
//
// When bracketed paste mode is set, pasted text is bracketed with control
// sequences so that the program can differentiate pasted text from typed-
// in text.  When bracketed paste mode is set, the program will receive:
//
//	ESC [ 2 0 0 ~ ,
//
// followed by the pasted text, followed by
//
//	ESC [ 2 0 1 ~ .
func (emu *Emulator) pasteSelection(utf8selection string) string {
	var b strings.Builder

	if emu.bracketedPasteMode {
		fmt.Fprint(&b, "\x1b[200~")
	}

	fmt.Fprint(&b, strings.ReplaceAll(utf8selection, "\n", "\r"))

	if emu.bracketedPasteMode {
		fmt.Fprint(&b, "\x1b[201~")
	}

	return b.String()
}

// return the terminal feedback, clean feedback buffer.
func (emu *Emulator) ReadOctetsToHost() string {
	ret := emu.terminalToHost.String()
	emu.terminalToHost.Reset()
	return ret
}

func (emu *Emulator) resetTerminal() {
	emu.parser.reset()

	emu.resetScreen()
	emu.resetAttrs()

	emu.switchColMode(ColMode_C80)
	emu.cf.dropScrollbackHistory()
	emu.resetBell()
	emu.resetTitle()
	emu.resetWindowTitleStack()
	emu.marginTop, emu.marginBottom = emu.cf.resetMargins()
	emu.clearScreen()

	emu.switchScreenBufferMode(false)
	emu.altSendsEscape = true
	emu.altScrollMode = false
	emu.modifyOtherKeys = 1
	// TODO consider how to implemnt options parameters

	emu.horizMarginMode = false
	emu.hMargin = 0
	emu.nColsEff = emu.nCols
	// TODO checking hasOSCHandler
}

func (emu *Emulator) resetAttrs() {
	emu.reverseVideo = false
	emu.fg = emu.attrs.renditions.fgColor
	emu.bg = emu.attrs.renditions.bgColor

	// reset the character attributes
	params := []int{0} // preapare parameters for SGR
	hdl_csi_sgr(emu, params)

	emu.links = newLinks()
}

func (emu *Emulator) resetScreen() {
	emu.showCursorMode = true
	emu.autoWrapMode = true
	emu.autoNewlineMode = false
	emu.keyboardLocked = false
	emu.insertMode = false
	emu.bkspSendsDel = true
	emu.localEcho = false
	emu.bracketedPasteMode = false

	emu.setCompatLevel(CompatLevel_VT400)
	emu.cursorKeyMode = CursorKeyMode_ANSI
	emu.keypadMode = KeypadMode_Normal
	emu.originMode = OriginMode_Absolute
	resetCharsetState(&emu.charsetState)

	emu.savedCursor_SCO.isSet = false
	emu.savedCursor_DEC.isSet = false

	emu.mouseTrk = newMouseTrackingState()
	emu.tabStops = make([]int, 0)
	emu.cf.getSelectionPtr().clear()
}

func (emu *Emulator) clearScreen() {
	emu.posX = 0
	emu.posY = 0
	emu.lastCol = false
	emu.fillScreen(' ') // clear the screen contents
}

func (emu *Emulator) fillScreen(ch rune) {
	emu.cf.fillCells(ch, emu.attrs)
}

func (emu *Emulator) normalizeCursorPos() {
	if emu.nColsEff < emu.posX+1 {
		emu.posX = emu.nColsEff - 1
	}
	if emu.nRows < emu.posY+1 {
		emu.posY = emu.nRows - 1
	}

	emu.lastCol = false
}

func (emu *Emulator) isCursorInsideMargins() bool {
	return emu.posX >= emu.hMargin && emu.posX < emu.nColsEff &&
		emu.posY >= emu.marginTop && emu.posY < emu.marginBottom
}

func (emu *Emulator) eraseRow(pY int) {
	emu.cf.eraseInRow(pY, emu.hMargin, emu.nColsEff-emu.hMargin, emu.attrs)
}

// erase rows at and below startY, within the scrolling area
func (emu *Emulator) eraseRows(startY, count int) {
	for pY := startY; pY < startY+count; pY++ {
		emu.eraseRow(pY)
	}
}

// copy row from src to dst.
func (emu *Emulator) copyRow(dstY, srcY int) {
	emu.cf.copyRow(dstY, srcY, emu.hMargin, emu.nColsEff-emu.hMargin)
}

// copy rows from startY to startY+count, move rows down
// insert blank rows at and below startY, within the scrolling area
func (emu *Emulator) insertRows(startY, count int) {
	for pY := emu.marginBottom - count - 1; pY >= startY; pY-- {
		emu.copyRow(pY+count, pY)
		if pY == 0 {
			break
		}
	}
	for pY := startY; pY < startY+count; pY++ {
		emu.eraseRow(pY)
	}
}

// copy rows from startY+count to startY, move rows up
// delete rows at and below startY, within the scrolling area
func (emu *Emulator) deleteRows(startY, count int) {
	for pY := startY; pY < emu.marginBottom-count; pY++ {
		emu.copyRow(pY, pY+count)
	}

	for pY := emu.marginBottom - count; pY < emu.marginBottom; pY++ {
		emu.eraseRow(pY)
	}
}

// insert count blank cols at startX, within the scrolling area
func (emu *Emulator) insertCols(startX, count int) {
	for r := emu.marginTop; r < emu.marginBottom; r++ {
		emu.cf.moveInRow(r, startX+count, startX, emu.nColsEff-startX-count)
		emu.cf.eraseInRow(r, startX, count, emu.attrs) // use the default renditions
	}
}

// delete count cols at startX, within the scrolling area
func (emu *Emulator) deleteCols(startX, count int) {
	for r := emu.marginTop; r < emu.marginBottom; r++ {
		emu.cf.moveInRow(r, startX, startX+count, emu.nColsEff-startX-count)
		emu.cf.eraseInRow(r, emu.nColsEff-count, count, emu.attrs) // use the default renditions
	}
}

func (emu *Emulator) showCursor() {
	// TODO figure out why we need the parser state?
	// if emu.showCursorMode && emu.parser.getState() == InputState_Normal {

	if emu.showCursorMode {
		emu.cf.setCursorPos(emu.posY, emu.posX)
		if emu.hasFocus {
			emu.cf.setCursorStyle(CursorStyle_FillBlock)
		} else {
			emu.cf.setCursorStyle(CursorStyle_HollowBlock)
		}
	}
}

func (emu *Emulator) hideCursor() {
	emu.cf.setCursorStyle(CursorStyle_Hidden)
}

func (emu *Emulator) jumpToNextTabStop() {
	if len(emu.tabStops) == 0 {
		margin := 0
		if emu.isCursorInsideMargins() {
			margin = emu.hMargin
		}
		// Hard default of 8 chars limited to right margin
		for ok := true; ok; ok = emu.posX < margin {
			emu.posX = ((emu.posX / 8) + 1) * 8
		}
		emu.posX = min(emu.posX, emu.nColsEff-1)
	} else {
		// Next tabstop column set, or the right margin
		nextTabIdx := LowerBound(emu.tabStops, emu.posX)
		if nextTabIdx >= len(emu.tabStops) {
			emu.posX = emu.nCols - 1
		} else {
			emu.posX = emu.tabStops[nextTabIdx]
		}
	}
	emu.lastCol = false
}

// TODO see the comments
func (emu *Emulator) switchColMode(colMode ColMode) {
	if emu.colMode == colMode {
		return
	}

	emu.resetScreen()
	emu.clearScreen()

	if colMode == ColMode_C80 {
		// emu.logT.Println("DECCOLM: Selected 80 columns per line")
		util.Logger.Debug("DECCOLM: Selected 80 columns per line")
	} else {
		// emu.logT.Println("DECCOLM: Selected 132 columns per line")
		util.Logger.Debug("DECCOLM: Selected 132 columns per line")
	}

	emu.colMode = colMode
}

func (emu *Emulator) switchScreenBufferMode(altScreenBufferMode bool) {
	if emu.altScreenBufferMode == altScreenBufferMode {
		return
	}

	if altScreenBufferMode {
		// fmt.Printf("+switchScreenBufferMode=%t marginBottom=%d, marginTop=%d, nRows=%d, nCols=%d\n",
		// 	emu.altScreenBufferMode, emu.marginBottom, emu.marginTop, emu.nRows, emu.nCols)
		emu.frame_alt, emu.marginTop, emu.marginBottom = NewFramebuffer3(emu.nCols, emu.nRows, 0)
		emu.cf = &emu.frame_alt
		emu.cf.expose()
		emu.clearScreen() // fill screen with space

		emu.savedCursor_DEC = &emu.savedCursor_DEC_alt
		emu.altScreenBufferMode = true
	} else {
		// fmt.Printf("-switchScreenBufferMode=%t marginBottom=%d, marginTop=%d, nRows=%d, nCols=%d\n",
		// 	emu.altScreenBufferMode, emu.marginBottom, emu.marginTop, emu.nRows, emu.nCols)
		emu.marginTop, emu.marginBottom = emu.frame_pri.resize(emu.nCols, emu.nRows)
		emu.cf = &emu.frame_pri
		emu.cf.expose()
		emu.frame_alt.freeCells()

		emu.savedCursor_DEC_alt.isSet = false
		emu.savedCursor_DEC = &emu.savedCursor_DEC_pri
		emu.altScreenBufferMode = false
	}

	// fmt.Printf(" switchScreenBufferMode=%t marginBottom=%d, marginTop=%d, nRows=%d, nCols=%d\n",
	// 	emu.altScreenBufferMode, emu.marginBottom, emu.marginTop, emu.nRows, emu.nCols)
	emu.links = newLinks()
}

// only set compatibility level for emulator
func (emu *Emulator) setCompatLevel(cl CompatibilityLevel) {
	if emu.compatLevel != cl {
		emu.compatLevel = cl
	}
	// emu.parser.compatLevel = cl
	// we seperate the parser compatLevel and emulator compatLevel.
}

func (emu *Emulator) initSelectionStore() {
	// prepare selection data storage for OSC 52
	// https://invisible-island.net/xterm/ctlseqs/ctlseqs.html#h3-Operating-System-Commands
	emu.selectionStore = make(map[rune]string)
	emu.selectionStore['c'] = "" // clipboard
	emu.selectionStore['p'] = "" // primary
	emu.selectionStore['q'] = "" // secondary
	emu.selectionStore['s'] = "" // select
	emu.selectionStore['0'] = "" // cut-buffer 0
	emu.selectionStore['1'] = "" // cut-buffer 1
	emu.selectionStore['2'] = "" // cut-buffer 2
	emu.selectionStore['3'] = "" // cut-buffer 3
	emu.selectionStore['4'] = "" // cut-buffer 4
	emu.selectionStore['5'] = "" // cut-buffer 5
	emu.selectionStore['6'] = "" // cut-buffer 6
	emu.selectionStore['7'] = "" // cut-buffer 7
}

func (emu *Emulator) lookupCharset(p rune) (r rune) {
	// choose the charset based on instructions before
	var cs *map[byte]rune
	if emu.charsetState.ss > 0 {
		cs = emu.charsetState.g[emu.charsetState.ss]
		emu.charsetState.ss = 0
	} else {
		if p < 0x80 {
			cs = emu.charsetState.g[emu.charsetState.gl]
		} else {
			cs = emu.charsetState.g[emu.charsetState.gr]
		}
	}

	r = lookupTable(cs, byte(p))
	return r
}

func (emu *Emulator) GetParser() *Parser {
	return emu.parser
}

func (emu *Emulator) Support(cap int) bool {
	if _, ok := emu.caps[cap]; ok {
		return true
	}
	return false
}

// check hander which will generate response. return true if excluded, otherwise false.
//
// hd: control sequence handler
//
// before: the response length before handler.
//
// after: the response length after handler.
func (emu *Emulator) excludeHandler(hd *Handler, before int, after int) bool {
	switch hd.id {
	case CSI_DSR, CSI_priDA, CSI_secDA, DCS_DECRQSS, DCS_XTGETTCAP:
		return true
	case OSC_4, OSC_10_11_12_17_19, CSI_DECRQM:
		return true
	case CSI_U_QUERY:
		return true
	case CSI_U_PUSH, CSI_U_POP, CSI_U_SET:
		if emu.Support(CSI_U_QUERY) {
			// special case: change local terminal emulator setting
			return false
		}
		return true
	case OSC_52: // special case: set OSC 52 data, then query it, the response will be updated.
		if before != after {
			return true
		}
	case VT52_ID:
		return true
	}
	return false
}

// parse and handle the stream together.
func (emu *Emulator) HandleStream(seq string) (hds []*Handler, diff string) {
	if len(seq) == 0 {
		return nil, ""
	}

	var diffB strings.Builder
	var respLen int

	hds = make([]*Handler, 0, 16)
	hds = emu.parser.processStream(seq, hds)
	for _, hd := range hds {
		respLen = emu.terminalToHost.Len()
		hd.handle(emu)

		if !emu.excludeHandler(hd, respLen, emu.terminalToHost.Len()) {
			diffB.WriteString(hd.sequence)
			// } else {
			// util.Log.Warn("HandleStream diff skip",
			// 	"hd", strHandlerID[hd.GetId()],
			// 	"sequence", hd.sequence)
		}
	}

	diff = diffB.String()
	return
}

// parse and handle the stream together. counting occupied rows, if
// ring buffer is full, pause the process and return the remains stream.
func (emu *Emulator) HandleLargeStream(seq string) (diff, remains string) {
	if len(seq) == 0 {
		// util.Log.Debug("HandleLargeStream no remains left")
		return
	}

	diff = seq
	remains = ""

	// if we reach the max rows, just return
	if emu.cf.reachMaxRows(emu.lastRows) {
		// util.Log.Debug("HandleLargeStream reach max rows, wait next time")
		remains = seq
		diff = "" // don't change diff
		return
	}

	// prepare to check occupied rows
	pos := emu.cf.getPhysicalRow(emu.posY)
	start := false
	// util.Log.Debug("rewind check",
	// 	"pos", emu.cf.getPhysicalRow(emu.posY),
	// 	"posY", emu.posY)
	var diffB strings.Builder
	var respLen int

	hds := make([]*Handler, 0, 16)
	hds = emu.parser.processStream(seq, hds)

	// util.Log.Debug("HandleLargeStream",
	// 	"point", 100,
	// 	"scrollHead", emu.cf.scrollHead,
	// 	"posY", emu.posY,
	// 	"posX", emu.posX)

	for idx, hd := range hds {
		// check rewind case
		if !emu.altScreenBufferMode && start && emu.cf.isFullFrame(emu.lastRows, pos, emu.cf.getPhysicalRow(emu.posY)) {

			// save over size content (remains) for later opportunity.
			// diff = restoreSequence(hds[:idx])
			remains = restoreSequence(hds[idx:])
			// util.Log.Warn("rewind check",
			// 	"oldPos", pos,
			// 	"posY", emu.posY,
			// 	"idx", idx,
			// 	"processed", ret)
			// util.Log.Debug("rewind check",
			// 	"newPos", emu.cf.getPhysicalRow(emu.posY),
			// 	"posY", emu.posY,
			// 	"idx", idx,
			// 	"remains", remains)
			// util.Log.Debug("rewind check",
			// 	"gap", emu.cf.getRowsGap(pos, emu.cf.getPhysicalRow(emu.posY)))
			break
		}

		respLen = emu.terminalToHost.Len()
		hd.handle(emu)

		if !emu.excludeHandler(hd, respLen, emu.terminalToHost.Len()) {
			diffB.WriteString(hd.sequence)
			// } else {
			// util.Log.Debug("HandleLargeStream diff skip",
			// 	"hd", strHandlerID[hd.GetId()],
			// 	"sequence", hd.sequence)
		}

		// start the check
		if !emu.altScreenBufferMode && emu.cf.getPhysicalRow(emu.posY) != pos {
			start = true
		}
		// util.Log.Debug("rewind check",
		// 	"newPos", emu.cf.getPhysicalRow(emu.posY),
		// 	"gap", emu.cf.getRowsGap(pos, emu.cf.getPhysicalRow(emu.posY)),
		// 	"seq", hd.sequence)
	}

	diff = diffB.String()

	if !emu.altScreenBufferMode {
		emu.lastRows += emu.cf.getRowsGap(pos, emu.cf.getPhysicalRow(emu.posY))
		// util.Log.Debug("HandleLargeStream",
		// 	"lastRows", emu.lastRows,
		// 	"once", emu.cf.getRowsGap(pos, emu.cf.getPhysicalRow(emu.posY)))
	}

	// util.Log.Debug("HandleLargeStream",
	// 	"point", 200,
	// 	"scrollHead", emu.cf.scrollHead,
	// 	"posY", emu.posY,
	// 	"posX", emu.posX,
	// 	"diff", diff)
	return
}

func (emu *Emulator) SetLastRows(x int) {
	emu.lastRows = x
}

func (emu *Emulator) GetFramebuffer() *Framebuffer {
	return emu.cf
}

/*
-----------------------------------------------------------------------------------------------------
The following methods is only used by prediction engine. The coordinate is different from the one used
by control sequence. It use the Emulator internal coordinate, starts from [0,0].
-----------------------------------------------------------------------------------------------------
*/

// move cursor to specified position
func (emu *Emulator) MoveCursor(posY, posX int) {
	// emu.posX = posX
	// emu.posY = posY
	//
	// emu.normalizeCursorPos()
	hdl_csi_cup(emu, posY+1, posX+1)
	emu.posY, emu.posX = emu.regulatePos(posY, posX)
}

// get current cursor column
func (emu *Emulator) GetCursorCol() int {
	return emu.posX
}

// get current cursor row
func (emu *Emulator) GetCursorRow() int {
	if emu.originMode == OriginMode_Absolute {
		return emu.posY
	}
	return emu.posY - emu.marginTop
}

// get active area height
func (emu *Emulator) GetHeight() int {
	return emu.marginBottom - emu.marginTop
}

// get active area width
func (emu *Emulator) GetWidth() int {
	if emu.horizMarginMode {
		return emu.nColsEff - emu.hMargin
	}

	return emu.nCols
}

func (emu *Emulator) GetSaveLines() int {
	return emu.cf.saveLines
}

func (emu *Emulator) GetCell(posY, posX int) Cell {
	posY, posX = emu.regulatePos(posY, posX)

	return emu.cf.getCell(posY, posX)
}

func (emu *Emulator) GetCellPtr(posY, posX int) *Cell {
	posY, posX = emu.regulatePos(posY, posX)

	return emu.cf.getCellPtr(posY, posX)
}

// convert the [posY,posX] into right position coordinates
func (emu *Emulator) regulatePos(posY, posX int) (posY2, posX2 int) {
	// fmt.Printf("#regulatePos convert (%d,%d)", posY, posX)
	// in case we don't provide the row or col
	if posY < 0 {
		posY = emu.GetCursorRow()
	}

	if posX < 0 {
		posX = emu.GetCursorCol()
	}

	switch emu.originMode {
	case OriginMode_Absolute:
		posY = max(0, min(posY, emu.nRows-1))
	case OriginMode_ScrollingRegion:
		posY = max(0, min(posY, emu.marginBottom-1))
		posY += emu.marginTop
	}

	if emu.horizMarginMode {
		posX = max(emu.hMargin, min(posX, emu.nColsEff-1))
	} else {
		posX = max(0, min(posX, emu.nCols-1))
	}

	posX2 = posX
	posY2 = posY

	// fmt.Printf(" into (%d,%d)\n", posY2, posX2)
	return
}

func (emu *Emulator) GetRenditions() (rnd Renditions) {
	return emu.attrs.renditions
}

func (emu *Emulator) SetCursorVisible(visible bool) {
	if !visible {
		emu.cf.setCursorStyle(CursorStyle_Hidden)
	} else {
		emu.cf.setCursorStyle(CursorStyle_FillBlock)
		// TODO keep the old style?
	}
}

func (emu *Emulator) Clone() *Emulator {
	clone := Emulator{}

	// clone regular data fields
	clone = *emu

	// clone cf, frame_alt, frame_pri
	clone.frame_alt.cells = make([]Cell, len(emu.frame_alt.cells))
	copy(clone.frame_alt.cells, emu.frame_alt.cells)

	clone.frame_pri.cells = make([]Cell, len(emu.frame_pri.cells))
	copy(clone.frame_pri.cells, emu.frame_pri.cells)

	clone.copyCaps(emu.caps)

	if emu.cf == &emu.frame_alt {
		clone.cf = &clone.frame_alt
	} else {
		clone.cf = &clone.frame_pri
	}

	clone.frame_alt.kittyKbd = emu.frame_alt.kittyKbd.Clone()
	clone.frame_pri.kittyKbd = emu.frame_pri.kittyKbd.Clone()

	// create a new parser
	clone.parser = &Parser{}
	clone.parser.reset()

	// clone tabStops
	clone.tabStops = make([]int, len(emu.tabStops))
	copy(clone.tabStops, emu.tabStops)

	// clone charsetState
	clone.charsetState.g = emu.charsetState.g

	// clone savedCursor_DEC, savedCursor_DEC_pri, savedCursor_DEC_alt
	clone.savedCursor_DEC_alt.charsetState.g = emu.savedCursor_DEC_alt.charsetState.g
	clone.savedCursor_DEC_pri.charsetState.g = emu.savedCursor_DEC_pri.charsetState.g
	if emu.savedCursor_DEC == &emu.savedCursor_DEC_alt {
		clone.savedCursor_DEC = &clone.savedCursor_DEC_alt
	} else {
		clone.savedCursor_DEC = &clone.savedCursor_DEC_pri
	}

	// init selectionStore
	clone.initSelectionStore()

	// clone windowTitleStack
	clone.windowTitleStack = make([]string, len(emu.windowTitleStack))
	copy(clone.windowTitleStack, emu.windowTitleStack)

	clone.links = emu.links.clone()

	// ignore logI,logT,logU,logW
	return &clone
}

func (emu *Emulator) PrefixWindowTitle(prefix string) { emu.prefixWindowTitle(prefix) }
func (emu *Emulator) GetWindowTitle() string          { return emu.getWindowTitle() }
func (emu *Emulator) GetIconLabel() string            { return emu.getIconLabel() }
func (emu *Emulator) setTitleInitialized()            { emu.titleInitialized = true }
func (emu *Emulator) isTitleInitialized() bool        { return emu.titleInitialized }
func (emu *Emulator) setIconLabel(iconLabel string)   { emu.iconLabel = iconLabel }
func (emu *Emulator) setWindowTitle(title string)     { emu.windowTitle = title }
func (emu *Emulator) getIconLabel() string            { return emu.iconLabel }
func (emu *Emulator) getWindowTitle() string          { return emu.windowTitle }
func (emu *Emulator) resetTitle() {
	emu.windowTitle = ""
	emu.iconLabel = ""
	emu.titleInitialized = false
}

func (emu *Emulator) prefixWindowTitle(s string) {
	if emu.iconLabel == emu.windowTitle {
		/* preserve equivalence */
		emu.iconLabel = s + emu.iconLabel
	}
	emu.windowTitle = s + emu.windowTitle
}

func (emu *Emulator) ringBell()         { emu.bellCount += 1 }
func (emu *Emulator) getBellCount() int { return emu.bellCount }
func (emu *Emulator) resetBell()        { emu.bellCount = 0 }

func (emu *Emulator) saveWindowTitleOnStack() {
	title := emu.GetWindowTitle()
	if title != "" {
		emu.windowTitleStack = append(emu.windowTitleStack, title)
		if len(emu.windowTitleStack) > windowTitleStackMax {
			emu.windowTitleStack = emu.windowTitleStack[1:]
		}
	} else {
		util.Logger.Warn("save title on stack failed: no title exist.")
	}
}

func (emu *Emulator) restoreWindowTitleOnStack() {
	if len(emu.windowTitleStack) > 0 {
		index := len(emu.windowTitleStack) - 1
		title := emu.windowTitleStack[index]
		emu.windowTitleStack = emu.windowTitleStack[:index]
		emu.setWindowTitle(title)
	} else {
		util.Logger.Warn("restore title from stack failed: empty stack.")
	}
}

func (emu *Emulator) resetWindowTitleStack() {
	emu.windowTitleStack = make([]string, 0)
}

func (emu *Emulator) Equal(x *Emulator) bool {
	return emu.equal(x, false)
}

// TODO remove this after finish test.
func (emu *Emulator) EqualTrace(x *Emulator) bool {
	return emu.equal(x, true)
}

func (emu *Emulator) equal(x *Emulator, trace bool) (ret bool) {
	ret = true
	if emu.nRows != x.nRows || emu.nCols != x.nCols {
		if trace {
			msg := fmt.Sprintf("nRows=(%d,%d), nCols=(%d,%d)", emu.nRows, x.nRows, emu.nCols, x.nCols)
			util.Logger.Warn(msg)
			ret = false
		} else {
			return false
		}
	}

	if emu.posX != x.posX || emu.posY != x.posY ||
		emu.marginTop != x.marginTop || emu.marginBottom != x.marginBottom {
		if trace {
			msg := fmt.Sprintf("posX=(%d,%d), posY=(%d,%d), marginTop=(%d,%d), marginBottom=(%d,%d)",
				emu.posX, x.posX, emu.posY, x.posY, emu.marginTop, x.marginTop, emu.marginBottom, x.marginBottom)
			util.Logger.Warn(msg)
			ret = false
		} else {
			return false
		}
	}

	if emu.lastCol != x.lastCol || emu.attrs != x.attrs ||
		emu.fg != x.fg || emu.bg != x.bg {
		if trace {
			msg := fmt.Sprintf("lastCol=(%t,%t), attrs=(%v,%v), fg=(%v,%v), bg=(%v,%v)",
				emu.lastCol, x.lastCol, emu.attrs, x.attrs, emu.fg, x.fg, emu.bg, x.bg)
			util.Logger.Warn(msg)
			ret = false
		} else {
			return false
		}
	}

	if emu.reverseVideo != x.reverseVideo || emu.hasFocus != x.hasFocus ||
		emu.showCursorMode != x.showCursorMode || emu.altScreenBufferMode != x.altScreenBufferMode {
		if trace {
			msg := fmt.Sprintf("reverseVideo=(%t,%t), hasFocus=(%t,%t), showCursorMode(%t,%t), altScreenBufferMode=(%t,%t)",
				emu.reverseVideo, x.reverseVideo, emu.hasFocus, x.hasFocus, emu.showCursorMode, x.showCursorMode,
				emu.altScreenBufferMode, x.altScreenBufferMode)
			util.Logger.Warn(msg)
			ret = false
		} else {
			return false
		}
	}

	if emu.autoWrapMode != x.autoWrapMode || emu.autoNewlineMode != x.autoNewlineMode ||
		emu.keyboardLocked != x.keyboardLocked || emu.insertMode != x.insertMode {
		if trace {
			msg := fmt.Sprintf("autoWrapMode=(%t,%t), autoNewlineMode=(%t,%t), keyboardLocked=(%t,%t), insertMode=(%t,%t)",
				emu.autoWrapMode, x.autoWrapMode, emu.autoNewlineMode, x.autoNewlineMode,
				emu.keyboardLocked, x.keyboardLocked, emu.insertMode, x.insertMode)
			util.Logger.Warn(msg)
			ret = false
		} else {
			return false
		}
	}

	if emu.bkspSendsDel != x.bkspSendsDel || emu.localEcho != x.localEcho ||
		emu.bracketedPasteMode != x.bracketedPasteMode || emu.altScrollMode != x.altScrollMode {
		if trace {
			msg := fmt.Sprintf("bkspSendsDel=(%t,%t), localEcho=(%t,%t), bracketedPasteMode=(%t,%t), altScrollMode=(%t,%t)",
				emu.bkspSendsDel, x.bkspSendsDel, emu.localEcho, x.localEcho,
				emu.bracketedPasteMode, x.bracketedPasteMode, emu.altScrollMode, x.altScrollMode)
			util.Logger.Warn(msg)
			ret = false
		} else {
			return false
		}
	}

	if emu.altSendsEscape != x.altSendsEscape || emu.modifyOtherKeys != x.modifyOtherKeys {
		if trace {
			msg := fmt.Sprintf("altSendsEscape=(%t,%t), modifyOtherKeys=(%d,%d), ",
				emu.altSendsEscape, x.altSendsEscape, emu.modifyOtherKeys, x.modifyOtherKeys)
			util.Logger.Warn(msg)
			ret = false
		} else {
			return false
		}
	}

	if emu.nColsEff != x.nColsEff || emu.hMargin != x.hMargin || emu.horizMarginMode != x.horizMarginMode {
		if trace {
			msg := fmt.Sprintf("nColsEff=(%d,%d), hMargin=(%d,%d) horizMarginMode=(%t,%t)",
				emu.nColsEff, x.nColsEff, emu.hMargin, x.hMargin,
				emu.horizMarginMode, x.horizMarginMode)
			util.Logger.Warn(msg)
			ret = false
		} else {
			return false
		}
	}

	if len(emu.tabStops) != len(x.tabStops) { // different tabStops number
		if trace {
			msg := fmt.Sprintf("tabStops length=(%d,%d)", len(emu.tabStops), len(x.tabStops))
			util.Logger.Warn(msg)
			ret = false
		} else {
			return false
		}
	} else if len(emu.tabStops) != 0 {
		for i := range emu.tabStops {
			if emu.tabStops[i] != x.tabStops[i] {
				// same tabStops number, different tabStops value
				if trace {
					msg := fmt.Sprintf("tabStops[%d]=(%d,%d)", i, emu.tabStops[i], x.tabStops[i])
					util.Logger.Warn(msg)
					ret = false
				} else {
					return false
				}
			}
		}
	}

	if !emu.charsetState.Equal(&x.charsetState) {
		if trace {
			msg := fmt.Sprintf(
				"charsetState.vtMode=(%t,%t), charsetState.gl=(%d,%d), charsetState.gr=(%d,%d), charsetState.ss=(%d,%d)",
				emu.charsetState.vtMode, x.charsetState.vtMode, emu.charsetState.gl, x.charsetState.gl,
				emu.charsetState.gr, x.charsetState.gr, emu.charsetState.ss, x.charsetState.ss)
			util.Logger.Warn(msg)
			ret = false
		} else {
			return false
		}
	}

	if emu.compatLevel != x.compatLevel || emu.cursorKeyMode != x.cursorKeyMode {
		if trace {
			msg := fmt.Sprintf("compatLevel=(%d,%d), cursorKeyMode=(%d,%d)",
				emu.compatLevel, x.compatLevel, emu.cursorKeyMode, x.cursorKeyMode)
			util.Logger.Warn(msg)
			ret = false
		} else {
			return false
		}
	}

	if emu.keypadMode != x.keypadMode || emu.originMode != x.originMode || emu.colMode != x.colMode {
		if trace {
			msg := fmt.Sprintf("keypadMode=(%d,%d), originMode=(%d,%d), colMode=(%d,%d)",
				emu.keypadMode, x.keypadMode, emu.originMode, x.originMode,
				emu.colMode, x.colMode)
			util.Logger.Warn(msg)
			ret = false
		} else {
			return false
		}
	}

	if emu.savedCursor_SCO != x.savedCursor_SCO {
		if trace {
			msg := fmt.Sprintf("savedCursor_SCO=(%v,%v)", emu.savedCursor_SCO, x.savedCursor_SCO)
			util.Logger.Warn(msg)
			ret = false
		} else {
			return false
		}
	}

	if emu.savedCursor_DEC.SavedCursor_SCO != x.savedCursor_DEC.SavedCursor_SCO ||
		emu.savedCursor_DEC.attrs != x.savedCursor_DEC.attrs ||
		emu.savedCursor_DEC.originMode != x.savedCursor_DEC.originMode ||
		!emu.savedCursor_DEC.charsetState.Equal(&x.savedCursor_DEC.charsetState) {
		if trace {
			var msg string
			if !emu.savedCursor_DEC.charsetState.Equal(&x.savedCursor_DEC.charsetState) {
				msg = fmt.Sprintf("savedCursor_DEC .charsetState .vtMode=(%t,%t), .gl=(%d,%d), .gr=(%d,%d), .ss=(%d,%d)",
					emu.savedCursor_DEC.charsetState.vtMode, x.savedCursor_DEC.charsetState.vtMode,
					emu.savedCursor_DEC.charsetState.gl, x.savedCursor_DEC.charsetState.gl,
					emu.savedCursor_DEC.charsetState.gr, x.savedCursor_DEC.charsetState.gr,
					emu.savedCursor_DEC.charsetState.ss, x.savedCursor_DEC.charsetState.ss)
			} else {
				msg = fmt.Sprintf("savedCursor_DEC .SavedCursor_SCO=(%v,%v), .attrs=(%v,%v), .originMode=(%d,%d)",
					emu.savedCursor_DEC.SavedCursor_SCO, x.savedCursor_DEC.SavedCursor_SCO,
					emu.savedCursor_DEC.attrs, x.savedCursor_DEC.attrs,
					emu.savedCursor_DEC.originMode, x.savedCursor_DEC.originMode)
			}
			util.Logger.Warn(msg)
			ret = false
		} else {
			return false
		}
	}
	if emu.mouseTrk != x.mouseTrk {
		if trace {
			msg := fmt.Sprintf("mouseTrk=(%v,%v)", emu.mouseTrk, x.mouseTrk)
			util.Logger.Warn(msg)
			ret = false
		} else {
			return false
		}
	}

	if emu.selectionData != x.selectionData {
		if trace {
			msg := fmt.Sprintf("selectionData length=(%d,%d), data=(%q,%q)",
				len(emu.selectionData), len(x.selectionData), emu.selectionData, x.selectionData)
			util.Logger.Warn(msg)
			ret = false
		} else {
			return false
		}
	}

	/* we don't compare selectionStore */
	// for k := range emu.selectionStore {
	// 	if emu.selectionStore[k] != x.selectionStore[k] {
	// 		if trace {
	// 			msg := fmt.Sprintf("selectionStore[%c]=(%q,%q)", k, emu.selectionStore[k], x.selectionStore[k])
	// 			util.Log.Warn(msg)
	// 		} else {
	// 			return false
	// 		}
	// 	}
	// }

	if emu.iconLabel != x.iconLabel || emu.windowTitle != x.windowTitle ||
		emu.bellCount != x.bellCount || emu.titleInitialized != x.titleInitialized {
		if trace {
			msg := fmt.Sprintf("iconLabel=(%s,%s), windowTitle=(%s,%s), bellCount=(%d,%d), titleInitialized=(%t,%t)",
				emu.iconLabel, x.iconLabel, emu.windowTitle, x.windowTitle,
				emu.bellCount, x.bellCount, emu.titleInitialized, x.titleInitialized)
			util.Logger.Warn(msg)
			ret = false
		} else {
			return false
		}
	}

	if len(emu.windowTitleStack) != len(x.windowTitleStack) {
		// different title stack number
		if trace {
			msg := fmt.Sprintf("windowTitleStack length=(%d,%d)", len(emu.windowTitleStack), len(x.windowTitleStack))
			util.Logger.Warn(msg)
			ret = false
		} else {
			return false
		}
	} else if len(emu.windowTitleStack) != 0 {
		for i := range emu.windowTitleStack {
			if emu.windowTitleStack[i] != x.windowTitleStack[i] {
				// same title stack number, different stack value
				if trace {
					msg := fmt.Sprintf("windowTitleStack[%d]=(%s,%s)", i, emu.windowTitleStack[i], x.windowTitleStack[i])
					util.Logger.Warn(msg)
					ret = false
				} else {
					return false
				}
			}
		}
	}

	if !maps.Equal(emu.caps, x.caps) {
		if trace {
			ret = false
			msg := fmt.Sprintf("caps=(%v,%v)", emu.caps, x.caps)
			util.Logger.Warn(msg)
		} else {
			return false
		}
	}

	// return emu.frame_pri.Equal(&x.frame_pri) && emu.frame_alt.Equal(&x.frame_alt)
	if !ret {
		if trace {
			ret = emu.cf.equal(x.cf, trace)
		}
		return ret
	}
	return emu.cf.equal(x.cf, trace)
}

// func (emu *Emulator) ResetDamage() {
// 	emu.cf.resetDamage()
// }

// func (emu *Emulator) getDamageRows() (rows int) {
// 	row1 := emu.cf.damage.start / emu.nCols
// 	row2 := emu.cf.damage.end / emu.nCols
//
// 	if row2 > row1 {
// 		return row2 - row1
// 	}
//
// 	return emu.cf.nRows + emu.cf.saveLines - (row1 - row2)
// }

// support screen row
func (emu *Emulator) getRowAt(pY int) (row []Cell) {
	start := emu.cf.getPhysRowIdx(pY)
	end := start + emu.nCols
	row = emu.cf.cells[start:end]
	return row
}

func (emu *Emulator) SetTerminalCaps(x map[int]string) {
	emu.copyCaps(x)
}

func (emu *Emulator) copyCaps(x map[int]string) {
	emu.caps = make(map[int]string)
	for k, v := range x {
		emu.caps[k] = v
	}
}

func cycleSelectSnapTo2(snapTo SelectSnapTo) SelectSnapTo {
	return SelectSnapTo((int(snapTo) + 1) % int(SelectSnapTo_COUNT))
}
