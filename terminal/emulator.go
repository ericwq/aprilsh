// Copyright 2022 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package terminal

import (
	"fmt"
	"strings"

	"github.com/ericwq/aprilsh/util"
)

// type VtModifier uint8
// TODO consider add InputSpec, InputSpecTable: mapping of a certain VtKey to a sequence of input characters
// TODO consider add RefreshHandlerFn, OscHandlerFn, BellHandlerFn

type Emulator struct {
	nRows        int          // replicated by NewFrame(),
	nCols        int          // replicated by NewFrame(),
	cf           *Framebuffer // replicated by NewFrame(), current frame buffer
	frame_pri    Framebuffer  // normal screen buffer
	frame_alt    Framebuffer  // alternate screen buffer
	posX         int          // replicated by NewFrame(), current cursor cols position (on-screen)
	posY         int          // replicated by NewFrame(), current cursor rows position (on-screen)
	marginTop    int          // replicated by NewFrame(), current margin top (screen view)
	marginBottom int          // replicated by NewFrame(), current margin bottom (screen view)
	lastCol      bool

	attrs Cell  // replicated by NewFrame() partially, prototype cell with current attributes
	fg    Color // TODO: should we keep this?
	bg    Color // TODO: should we keep this?

	parser *Parser

	// Terminal state - N.B.: keep resetTerminal () in sync with this!
	reverseVideo        bool // replicated by NewFrame(),
	hasFocus            bool // default true
	showCursorMode      bool // replicated by NewFrame(), default true, ds.cursor_visible
	altScreenBufferMode bool // replicated by NewFrame(), , Alternate Screen Buffer
	altScreen1049       bool // DECSET and DECRST 1049, default false
	autoWrapMode        bool // replicated by NewFrame(), default:true
	autoNewlineMode     bool // replicated by NewFrame(), LNM
	keyboardLocked      bool // replicated by NewFrame(),
	insertMode          bool // replicated by NewFrame(),
	bkspSendsDel        bool // replicated by NewFrame(), default:true, backspace send delete
	localEcho           bool // replicated by NewFrame(),
	bracketedPasteMode  bool // replicated by NewFrame(),
	altScrollMode       bool // replicated by NewFrame(),
	altSendsEscape      bool // replicated by NewFrame(), default true
	modifyOtherKeys     uint // replicated by NewFrame(),

	horizMarginMode bool // replicated by NewFrame(), left and right margins support
	nColsEff        int  // replicated by NewFrame(), right margins
	hMargin         int  // replicated by NewFrame(), left margins

	tabStops []int // replicated by NewFrame(), tab stop positions

	compatLevel   CompatibilityLevel // replicated by NewFrame(), VT52, VT100, VT400. default:VT400
	cursorKeyMode CursorKeyMode      // replicated by NewFrame(), default:ANSI: Application(true), ANSI(false)
	keypadMode    KeypadMode         // replicated by NewFrame(), default:Normal
	originMode    OriginMode         // replicated by NewFrame(), default:Absolute, ScrollingRegion(true), Absolute(false)
	colMode       ColMode            // replicated by NewFrame(), default:80, column mode 80 or 132, just for compatibility

	charsetState CharsetState // for forward compatibility

	savedCursor_SCO     SavedCursor_SCO // replicated by NewFrame(), SCO console cursor state
	savedCursor_DEC_pri SavedCursor_DEC
	savedCursor_DEC_alt SavedCursor_DEC
	savedCursor_DEC     *SavedCursor_DEC // replicated by NewFrame(),

	mouseTrk MouseTrackingState // replicated by NewFrame()

	terminalToHost strings.Builder // used for terminal write back
	user           UserInput       // TODO consider how to change it.
	selectionStore map[rune]string // local storage buffer for selection data in sequence OSC 52
	selectionData  string          // replicated by NewFrame(), store the selection data for OSC 52

	iconLabel        string   // replicated by NewFrame()
	windowTitle      string   // replicated by NewFrame()
	bellCount        int      // replicated by NewFrame()
	titleInitialized bool     // replicated by NewFrame()
	windowTitleStack []string // for XTWINOPS
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
	// emu.initLog()

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
	util.Log.With("cols", nCols).With("rows", nRows).Debug("Emulator.resize")
	if emu.nCols == nCols && emu.nRows == nRows {
		return
	}

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
			nScroll := Min(nRows-emu.nRows, emu.cf.getHistroryRows())
			emu.cf.scrollDown(nScroll)
			emu.posY += nScroll
		}

		emu.frame_alt.freeCells()
	}

	emu.nCols = nCols
	emu.nRows = nRows

	if emu.horizMarginMode {
		emu.nColsEff = Min(emu.nColsEff, emu.nCols)
		emu.hMargin = Max(0, Min(emu.hMargin, emu.nColsEff-2))
	} else {
		emu.nColsEff = emu.nCols
		emu.hMargin = 0
	}

	emu.normalizeCursorPos()
	emu.showCursor()

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
		emu.posX = Min(emu.posX, emu.nColsEff-1)
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
		util.Log.Debug("DECCOLM: Selected 80 columns per line")
	} else {
		// emu.logT.Println("DECCOLM: Selected 132 columns per line")
		util.Log.Debug("DECCOLM: Selected 132 columns per line")
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

// func (emu *Emulator) initLog() {
// 	// init logger
// 	emu.logT = log.New(os.Stderr, "TRAC: ", log.Ldate|log.Ltime|log.Lshortfile)
// 	emu.logI = log.New(os.Stderr, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
// 	emu.logE = log.New(os.Stderr, "ERRO: ", log.Ldate|log.Ltime|log.Lshortfile)
// 	emu.logW = log.New(os.Stderr, "WARN: ", log.Ldate|log.Ltime|log.Lshortfile)
// 	emu.logU = log.New(os.Stderr, "(Uimplemented): ", log.Ldate|log.Ltime|log.Lshortfile)
// }

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

// func (emu *Emulator) SetLogTraceOutput(w io.Writer) {
// 	emu.logT.SetOutput(w)
// }

// parse and handle the stream together.
func (emu *Emulator) HandleStream(seq string) (hds []*Handler) {
	if len(seq) == 0 {
		return nil
	}

	hds = make([]*Handler, 0, 16)
	hds = emu.parser.processStream(seq, hds)
	for _, hd := range hds {
		hd.handle(emu)
	}

	return hds
}

// parse and handle the stream together.
func (emu *Emulator) HandleLargeStream(seq string, feed chan string) (hds []*Handler, ret string) {
	if len(seq) == 0 {
		return nil, ""
	}

	// mark the rewind position
	pos := emu.cf.getPhysicalRow(emu.posY)
	start := false
	// util.Log.With("pos", emu.cf.getPhysicalRow(emu.posY)).With("posY", emu.posY).
	// 	Debug("rewind check")

	hds = make([]*Handler, 0, 16)
	hds = emu.parser.processStream(seq, hds)
	ret = ""
	for idx, hd := range hds {
		// check rewind case
		// if start && emu.cf.getPhysicalRow(emu.posY) == pos {
		if start && emu.cf.isFullFrame(pos, emu.cf.getPhysicalRow(emu.posY)) {

			// save over size content (remains) for later opportunity.
			ret = restoreSequence(hds[:idx])
			remains := restoreSequence(hds[idx:])
			util.Log.With("oldPos", pos).
				With("posY", emu.posY).With("idx", idx).
				// With("processed", strings.Split(ret, "。")[0]).
				With("processed", ret).
				Warn("rewind check")
			util.Log.With("newPos", emu.cf.getPhysicalRow(emu.posY)).
				With("posY", emu.posY).With("idx", idx).
				// With("remains", strings.Split(remains, "。")[0]).
				With("remains", remains).
				Warn("rewind check")
			util.Log.With("gap", emu.cf.getRowsGap(pos, emu.cf.getPhysicalRow(emu.posY))).
				Debug("rewind check")
			// hds = hds[:idx]
			break
		}

		hd.handle(emu)

		// start the check
		if emu.cf.getPhysicalRow(emu.posY) != pos {
			start = true
		}
		// util.Log.With("newPos", emu.cf.getPhysicalRow(emu.posY)).
		// 	With("gap", emu.cf.getRowsGap(pos, emu.cf.getPhysicalRow(emu.posY))).
		// 	With("seq", hd.sequence).Debug("rewind check")
	}

	return hds, ret
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
		posY = Max(0, Min(posY, emu.nRows-1))
	case OriginMode_ScrollingRegion:
		posY = Max(0, Min(posY, emu.marginBottom-1))
		posY += emu.marginTop
	}

	if emu.horizMarginMode {
		posX = Max(emu.hMargin, Min(posX, emu.nColsEff-1))
	} else {
		posX = Max(0, Min(posX, emu.nCols-1))
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

	if emu.cf == &emu.frame_alt {
		clone.cf = &clone.frame_alt
	} else {
		clone.cf = &clone.frame_pri
	}

	// create a new parser
	clone.parser = &Parser{}
	clone.parser.reset()

	// clone tabStops
	clone.tabStops = make([]int, len(emu.tabStops))
	copy(clone.tabStops, emu.tabStops)

	// clone charsetState
	for i := range emu.charsetState.g {
		clone.charsetState.g[i] = emu.charsetState.g[i]
	}

	// clone savedCursor_DEC, savedCursor_DEC_pri, savedCursor_DEC_alt
	for i := range emu.savedCursor_DEC_alt.charsetState.g {
		clone.savedCursor_DEC_alt.charsetState.g[i] = emu.savedCursor_DEC_alt.charsetState.g[i]
	}
	for i := range emu.savedCursor_DEC_pri.charsetState.g {
		clone.savedCursor_DEC_pri.charsetState.g[i] = emu.savedCursor_DEC_pri.charsetState.g[i]
	}
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
		util.Log.Warn("save title on stack failed: no title exist.")
	}
}

func (emu *Emulator) restoreWindowTitleOnStack() {
	if len(emu.windowTitleStack) > 0 {
		index := len(emu.windowTitleStack) - 1
		title := emu.windowTitleStack[index]
		emu.windowTitleStack = emu.windowTitleStack[:index]
		emu.setWindowTitle(title)
	} else {
		util.Log.Warn("restore title from stack failed: empty stack.")
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

func (emu *Emulator) equal(x *Emulator, trace bool) bool {
	if emu.nRows != x.nRows || emu.nCols != x.nCols {
		if trace {
			msg := fmt.Sprintf("nRows=(%d,%d), nCols=(%d,%d)", emu.nRows, x.nRows, emu.nCols, x.nCols)
			util.Log.Warn(msg)
		} else {
			return false
		}
	}

	if emu.posX != x.posX || emu.posY != x.posY ||
		emu.marginTop != x.marginTop || emu.marginBottom != x.marginBottom {
		if trace {
			msg := fmt.Sprintf("posX=(%d,%d), posY=(%d,%d), marginTop=(%d,%d), marginBottom=(%d,%d)",
				emu.posX, x.posX, emu.posY, x.posY, emu.marginTop, x.marginTop, emu.marginBottom, x.marginBottom)
			util.Log.Warn(msg)
		} else {
			return false
		}
	}

	if emu.lastCol != x.lastCol || emu.attrs != x.attrs ||
		emu.fg != x.fg || emu.bg != x.bg {
		if trace {
			msg := fmt.Sprintf("lastCol=(%t,%t), attrs=(%v,%v), fg=(%v,%v), bg=(%v,%v)",
				emu.lastCol, x.lastCol, emu.attrs, x.attrs, emu.fg, x.fg, emu.bg, x.bg)
			util.Log.Warn(msg)
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
			util.Log.Warn(msg)
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
			util.Log.Warn(msg)
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
			util.Log.Warn(msg)
		} else {
			return false
		}
	}

	if emu.altSendsEscape != x.altSendsEscape || emu.modifyOtherKeys != x.modifyOtherKeys ||
		emu.horizMarginMode != x.horizMarginMode {
		if trace {
			msg := fmt.Sprintf("altSendsEscape=(%t,%t), modifyOtherKeys=(%d,%d), horizMarginMode=(%t,%t) ",
				emu.altSendsEscape, x.altSendsEscape, emu.modifyOtherKeys, x.modifyOtherKeys,
				emu.horizMarginMode, x.horizMarginMode)
			util.Log.Warn(msg)
		} else {
			return false
		}
	}

	if emu.nColsEff != x.nColsEff || emu.hMargin != x.hMargin {
		if trace {
			msg := fmt.Sprintf("nColsEff=(%d,%d), hMargin=(%d,%d)",
				emu.nColsEff, x.nColsEff, emu.hMargin, x.hMargin)
			util.Log.Warn(msg)
		} else {
			return false
		}
	}

	if len(emu.tabStops) != len(x.tabStops) {
		if trace {
			msg := fmt.Sprintf("tabStops length=(%d,%d)", len(emu.tabStops), len(x.tabStops))
			util.Log.Warn(msg)
		} else {
			return false
		}
	}

	for i := range emu.tabStops {
		if emu.tabStops[i] != x.tabStops[i] {
			if trace {
				msg := fmt.Sprintf("tabStops[%d]=(%d,%d)", i, emu.tabStops[i], x.tabStops[i])
				util.Log.Warn(msg)
			} else {
				return false
			}
		}
	}

	if !emu.charsetState.Equal(&x.charsetState) {
		if trace {
			msg := fmt.Sprintf(
				"charsetState.vtMode=(%t,%t), charsetState.gl=(%d,%d), charsetState.gr=(%d,%d), charsetState.ss=(%d,%d)",
				emu.charsetState.vtMode, x.charsetState.vtMode, emu.charsetState.gl, x.charsetState.gl,
				emu.charsetState.gr, x.charsetState.gr, emu.charsetState.ss, x.charsetState.ss)
			util.Log.Warn(msg)
		} else {
			return false
		}
	}

	if emu.compatLevel != x.compatLevel || emu.cursorKeyMode != x.cursorKeyMode {
		if trace {
			msg := fmt.Sprintf("compatLevel=(%d,%d), cursorKeyMode=(%d,%d)",
				emu.compatLevel, x.compatLevel, emu.cursorKeyMode, x.cursorKeyMode)
			util.Log.Warn(msg)
		} else {
			return false
		}
	}

	if emu.keypadMode != x.keypadMode || emu.originMode != x.originMode || emu.colMode != x.colMode {
		if trace {
			msg := fmt.Sprintf("keypadMode=(%d,%d), originMode=(%d,%d), colMode=(%d,%d)",
				emu.keypadMode, x.keypadMode, emu.originMode, x.originMode,
				emu.colMode, x.colMode)
			util.Log.Warn(msg)
		} else {
			return false
		}
	}

	if emu.savedCursor_SCO != x.savedCursor_SCO {
		if trace {
			msg := fmt.Sprintf("savedCursor_SCO=(%v,%v)", emu.savedCursor_SCO, x.savedCursor_SCO)
			util.Log.Warn(msg)
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
			util.Log.Warn(msg)
		} else {
			return false
		}
	}
	if emu.mouseTrk != x.mouseTrk {
		if trace {
			msg := fmt.Sprintf("mouseTrk=(%v,%v)", emu.mouseTrk, x.mouseTrk)
			util.Log.Warn(msg)
		} else {
			return false
		}
	}

	if emu.selectionData != x.selectionData {
		if trace {
			msg := fmt.Sprintf("selectionData=(%q,%q)", len(emu.selectionData), len(x.selectionData))
			util.Log.Warn(msg)
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
			util.Log.Warn(msg)
		} else {
			return false
		}
	}

	if len(emu.windowTitleStack) != len(x.windowTitleStack) {
		if trace {
			msg := fmt.Sprintf("windowTitleStack length=(%d,%d)", len(emu.windowTitleStack), len(x.windowTitleStack))
			util.Log.Warn(msg)
		} else {
			return false
		}
	}
	for i := range emu.windowTitleStack {
		if emu.windowTitleStack[i] != x.windowTitleStack[i] {
			if trace {
				msg := fmt.Sprintf("windowTitleStack[%d]=(%s,%s)", i, emu.windowTitleStack[i], x.windowTitleStack[i])
				util.Log.Warn(msg)
			} else {
				return false
			}
		}
	}
	// return emu.frame_pri.Equal(&x.frame_pri) && emu.frame_alt.Equal(&x.frame_alt)
	return emu.cf.equal(x.cf, trace)
}

// func (emu *Emulator) ResetDamage() {
// 	emu.cf.resetDamage()
// }

func (emu *Emulator) getDamageRows() (rows int) {
	row1 := emu.cf.damage.start / emu.nCols
	row2 := emu.cf.damage.end / emu.nCols

	if row2 > row1 {
		return row2 - row1
	}

	return emu.cf.nRows + emu.cf.saveLines - (row1 - row2)
}

func (emu *Emulator) getRawRow(posY int) (row []Cell) {
	start := emu.cf.getPhysRowIdx(posY)
	end := start + emu.nCols
	row = emu.cf.cells[start:end]
	return row
}

func cycleSelectSnapTo2(snapTo SelectSnapTo) SelectSnapTo {
	return SelectSnapTo((int(snapTo) + 1) % int(SelectSnapTo_COUNT))
}
