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
	"log"
	"os"
	"strings"
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
	showCursorMode      bool // replicated by NewFrame(), default true, ds.cursor_visible
	altScreenBufferMode bool // replicated by NewFrame(), , Alternate Screen Buffer
	autoWrapMode        bool // replicated by NewFrame(), default:true
	autoNewlineMode     bool // replicated by NewFrame(),
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

	// logger
	logE *log.Logger
	logT *log.Logger // trace
	logU *log.Logger
	logW *log.Logger
	logI *log.Logger

	/*
		CursorVisible             bool // true/false
		ReverseVideo              bool // two possible value: Reverse(true), Normal(false)
		MouseReportingMode        int  // replace it with MouseTrackingMode
		MouseFocusEvent           bool // replace it with MouseTrackingState.focusEventMode
		MouseAlternateScroll      bool // rename to altScrollMode
		MouseEncodingMode         int  // replace it with MouseTrackingEnc
		ApplicationModeCursorKeys bool // =cursorKeyMode two possible value : Application(true), ANSI(false)
	*/
}

func NewEmulator3(nCols, nRows, saveLines int) *Emulator {
	// TODO makePalette256 (palette256);

	emu := &Emulator{}
	emu.parser = NewParser()
	emu.cf, emu.marginTop, emu.marginBottom = NewFramebuffer3(nCols, nRows, saveLines)
	emu.frame_pri = *emu.cf
	emu.frame_alt = NewFramebuffer2(1, 1)

	emu.nCols = nCols
	emu.nRows = nRows

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

	// emu.savedCursor_DEC_pri = SavedCursor_DEC{}
	emu.savedCursor_DEC_alt.originMode = OriginMode_Absolute
	emu.savedCursor_DEC_pri.originMode = OriginMode_Absolute
	emu.savedCursor_DEC = &emu.savedCursor_DEC_pri
	emu.initSelectionData()
	emu.initLog()

	emu.resetTerminal()

	return emu
}

func (emu *Emulator) resize(nCols, nRows int) {
	if emu.nCols == nCols && emu.nRows == nRows {
		return
	}

	emu.hideCursor()

	if emu.altScreenBufferMode {
		// create a new frame buffer
		var alt *Framebuffer
		alt, emu.marginTop, emu.marginBottom = NewFramebuffer3(nCols, nRows, 0)
		emu.frame_alt = *alt
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
	emu.resetCharsetState()

	emu.savedCursor_SCO.isSet = false
	emu.savedCursor_DEC.isSet = false

	emu.mouseTrk = MouseTrackingState{}
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

// TODO need implementation
func (emu *Emulator) showCursor() {
	if emu.showCursorMode && emu.parser.getState() == InputState_Normal {
		emu.cf.setCursorPos(emu.posY, emu.posX)
		emu.cf.setCursorStyle(CursorStyle_FillBlock)
		// TODO set HollowBlock for no focus case
	}
}

// TODO need implementation
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
		emu.logT.Println("DECCOLM: Selected 80 columns per line")
	} else {
		emu.logT.Println("DECCOLM: Selected 132 columns per line")
	}

	emu.colMode = colMode
}

func (emu *Emulator) switchScreenBufferMode(altScreenBufferMode bool) {
	if emu.altScreenBufferMode == altScreenBufferMode {
		return
	}

	if altScreenBufferMode {
		emu.cf, emu.marginTop, emu.marginBottom = NewFramebuffer3(emu.nCols, emu.nRows, 0)
		emu.frame_alt = *emu.cf

		emu.savedCursor_DEC = &emu.savedCursor_DEC_alt
		emu.altScreenBufferMode = true
	} else {
		emu.cf = &emu.frame_pri
		emu.marginTop, emu.marginBottom = emu.cf.resize(emu.nCols, emu.nRows)
		emu.cf.expose()

		emu.savedCursor_DEC_alt.isSet = false
		emu.savedCursor_DEC = &emu.savedCursor_DEC_pri
		emu.altScreenBufferMode = false
	}
}

// set compatibility level for both parser and emulator
func (emu *Emulator) setCompatLevel(cl CompatibilityLevel) {
	if emu.compatLevel != cl {
		emu.compatLevel = cl
	}
	// emu.parser.compatLevel = cl
}

func (emu *Emulator) initSelectionData() {
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

func (emu *Emulator) initLog() {
	// init logger
	emu.logT = log.New(os.Stderr, "TRAC: ", log.Ldate|log.Ltime|log.Lshortfile)
	emu.logI = log.New(os.Stderr, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	emu.logE = log.New(os.Stderr, "ERRO: ", log.Ldate|log.Ltime|log.Lshortfile)
	emu.logW = log.New(os.Stderr, "WARN: ", log.Ldate|log.Ltime|log.Lshortfile)
	emu.logU = log.New(os.Stderr, "(Uimplemented): ", log.Ldate|log.Ltime|log.Lshortfile)
}

func (emu *Emulator) resetCharsetState() {
	// we don't use vt100 charset by default
	emu.charsetState.vtMode = false

	// default nil will fall to UTF-8
	emu.charsetState.g[0] = nil
	emu.charsetState.g[1] = nil
	emu.charsetState.g[2] = nil
	emu.charsetState.g[3] = nil

	// Locking shift states (index into g[]):
	emu.charsetState.gl = 0 // G0 in GL
	emu.charsetState.gr = 2 // G2 in GR

	// Single shift state (0 if none active):
	// 0 - not active; 2: G2 in GL; 3: G3 in GL
	emu.charsetState.ss = 0
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

// parse and handle the stream together.
func (emu *Emulator) HandleStream(seq string) (hds []*Handler) {
	hds = make([]*Handler, 0, 16)
	if len(seq) == 0 {
		return
	}

	hds = emu.parser.processStream(seq, hds)
	for _, hd := range hds {
		// fmt.Printf("#HandleStream %s, %q\n", strHandlerID[hd.id], hd.sequence)
		hd.handle(emu)
	}
	return
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
	emu.posX = posX
	emu.posY = posY
	emu.normalizeCursorPos()
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

func (emu *Emulator) GetCell(posY, posX int) Cell {
	posY, posX = emu.getCellPos(posY, posX)

	return emu.cf.getCell(posY, posX)
}

func (emu *Emulator) GetCellPtr(posY, posX int) *Cell {
	posY, posX = emu.getCellPos(posY, posX)

	return emu.cf.getCellPtr(posY, posX)
}

// convert the [posY,posX] into right position coordinates
func (emu *Emulator) getCellPos(posY, posX int) (posY2, posX2 int) {
	// in case we don't provide the row or col
	if posY < 0 || posY > emu.GetHeight() {
		posY = emu.GetCursorRow()
	}

	if posX < 0 || posX > emu.GetWidth() {
		posX = emu.GetCursorCol()
	}

	// the logic is from hdl_csi_cup()
	switch emu.originMode {
	case OriginMode_Absolute:
		posY = Max(0, Min(posY, emu.nRows))
	case OriginMode_ScrollingRegion:
		posY = Max(0, Min(posY, emu.marginBottom))
		posY += emu.marginTop
	}
	posX = Max(0, Min(posX, emu.nCols))

	posX2 = posX
	posY2 = posY
	return
}

func (emu *Emulator) GetRenditions() (rnd Renditions) {
	return emu.attrs.renditions
}

func (emu *Emulator) PrefixWindowTitle(prefix string) {
	emu.cf.prefixWindowTitle(prefix)
}

func (emu *Emulator) GetWindowTitle() string {
	return emu.cf.getWindowTitle()
}

func (emu *Emulator) GetIconName() string {
	return emu.cf.getIconName()
}

func (emu *Emulator) SetCursorVisible(visible bool) {
	if !visible {
		emu.cf.setCursorStyle(CursorStyle_Hidden)
	} else {
		emu.cf.setCursorStyle(CursorStyle_FillBlock)
		// TODO keep the old style?
	}
}
