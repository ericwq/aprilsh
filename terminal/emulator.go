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
)

// const (
// 	DISPATCH_ESCAPE = iota + 1
// 	DISPATCH_CSI
// 	DISPATCH_CONTROL
// )
//
// type Emulator interface {
// 	Print(act Action)
// 	Execute(act Action)
// 	Dispatch() *Dispatcher
// 	CSIdispatch(act Action)
// 	ESCdispatch(act Action)
// 	OSCend(act Action)
// 	Resize(width int, height int)
// 	User() *UserInput
// 	Framebuffer() *Framebuffer
// }
//
// type Action interface {
// 	ActOn(t Emulator)
// 	Ignore() bool
// 	Name() string
// 	SetChar(rune)
// 	SetPresent(bool)
// 	GetChar() rune
// 	IsPresent() bool
// }

//
// /* These tables perform translation of built-in "hard" character sets
//  * to 16-bit Unicode points. All sets are defined as 96 characters, even
//  * those originally designated by DEC as 94-character sets.
//  *
//  * These tables are referenced by Vterm::charCodes (see below).
//  */
//
// // Ref: https://en.wikipedia.org/wiki/DEC_Special_Graphics
// var uc_DecSpec = [96]rune{
// 	0x0020, 0x0021, 0x0022, 0x0023, 0x0024, 0x0025, 0x0026, 0x0027,
// 	0x0028, 0x0029, 0x002a, 0x002b, 0x002c, 0x002d, 0x002e, 0x002f,
// 	0x0030, 0x0031, 0x0032, 0x0033, 0x0034, 0x0035, 0x0036, 0x0037,
// 	0x0038, 0x0039, 0x003a, 0x003b, 0x003c, 0x003d, 0x003e, 0x003f,
//
// 	0x0040, 0x0041, 0x0042, 0x0043, 0x0044, 0x0045, 0x0046, 0x0047,
// 	0x0048, 0x0049, 0x004a, 0x004b, 0x004c, 0x004d, 0x004e, 0x004f,
// 	0x0050, 0x0051, 0x0052, 0x0053, 0x0054, 0x0055, 0x0056, 0x0057,
// 	0x0058, 0x0059, 0x005a, 0x005b, 0x005c, 0x005d, 0x005e, 0x005f,
//
// 	0x25c6, 0x2592, 0x2409, 0x240c, 0x240d, 0x240a, 0x00b0, 0x00b1,
// 	0x2424, 0x240b, 0x2518, 0x2510, 0x250c, 0x2514, 0x253c, 0x23ba,
// 	0x23bb, 0x2500, 0x23bc, 0x23bd, 0x251c, 0x2524, 0x2534, 0x252c,
// 	0x2502, 0x2264, 0x2265, 0x03c0, 0x2260, 0x00a3, 0x00b7, 0x0020,
// }
//
// // Ref: https://en.wikipedia.org/wiki/Multinational_Character_Set
// var uc_DecSuppl = [96]rune{
// 	0x0020, 0x00a1, 0x00a2, 0x00a3, 0x0024, 0x00a5, 0x0026, 0x00a7,
// 	0x00a4, 0x00a9, 0x00aa, 0x00ab, 0x002c, 0x002d, 0x002e, 0x002f,
// 	0x00b0, 0x00b1, 0x00b2, 0x00b3, 0x0034, 0x00b5, 0x00b6, 0x00b7,
// 	0x0038, 0x00b9, 0x00ba, 0x00bb, 0x00bc, 0x00bd, 0x003e, 0x00bf,
//
// 	0x00c0, 0x00c1, 0x00c2, 0x00c3, 0x00c4, 0x00c5, 0x00c6, 0x00c7,
// 	0x00c8, 0x00c9, 0x00ca, 0x00cb, 0x00cc, 0x00cd, 0x00ce, 0x00cf,
// 	0x0050, 0x00d1, 0x00d2, 0x00d3, 0x00d4, 0x00d5, 0x00d6, 0x0152,
// 	0x00d8, 0x00d9, 0x00da, 0x00db, 0x00dc, 0x0178, 0x005e, 0x00df,
//
// 	0x00e0, 0x00e1, 0x00e2, 0x00e3, 0x00e4, 0x00e5, 0x00e6, 0x00e7,
// 	0x00e8, 0x00e9, 0x00ea, 0x00eb, 0x00ec, 0x00ed, 0x00ee, 0x00ef,
// 	0x0070, 0x00f1, 0x00f2, 0x00f3, 0x00f4, 0x00f5, 0x00f6, 0x0153,
// 	0x00f8, 0x00f9, 0x00fa, 0x00fb, 0x00fc, 0x00ff, 0x007e, 0x007f,
// }
//
// // Ref: https://en.wikipedia.org/wiki/DEC_Technical_Character_Set
// var uc_DecTechn = [96]rune{
// 	0x0020, 0x23b7, 0x250c, 0x2500, 0x2320, 0x2321, 0x2502, 0x23a1,
// 	0x23a3, 0x23a4, 0x23a6, 0x239b, 0x239d, 0x239e, 0x23a0, 0x23a8,
// 	0x23ac, 0x0020, 0x0020, 0x0020, 0x0020, 0x0020, 0x0020, 0x0020,
// 	0x0020, 0x0020, 0x0020, 0x0020, 0x2264, 0x2260, 0x2265, 0x222b,
//
// 	0x2234, 0x221d, 0x221e, 0x00f7, 0x0394, 0x2207, 0x03a6, 0x0393,
// 	0x223c, 0x2243, 0x0398, 0x00d7, 0x039b, 0x21d4, 0x21d2, 0x2261,
// 	0x03a0, 0x03a8, 0x0020, 0x03a3, 0x0020, 0x0020, 0x221a, 0x03a9,
// 	0x039e, 0x03a5, 0x2282, 0x2283, 0x2229, 0x222a, 0x2227, 0x2228,
//
// 	0x00ac, 0x03b1, 0x03b2, 0x03c7, 0x03b4, 0x03b5, 0x03c6, 0x03b3,
// 	0x03b7, 0x03b9, 0x03b8, 0x03ba, 0x03bb, 0x0020, 0x03bd, 0x2202,
// 	0x03c0, 0x03c8, 0x03c1, 0x03c3, 0x03c4, 0x0020, 0x0192, 0x03c9,
// 	0x03be, 0x03c5, 0x03b6, 0x2190, 0x2191, 0x2192, 0x2193, 0x007f,
// }
//
// // Ref: https://en.wikipedia.org/wiki/ISO/IEC_8859-1
// var uc_IsoLatin1 = [96]rune{
// 	0x00a0, 0x00a1, 0x00a2, 0x00a3, 0x00a4, 0x00a5, 0x00a6, 0x00a7,
// 	0x00a8, 0x00a9, 0x00aa, 0x00ab, 0x00ac, 0x00ad, 0x00ae, 0x00af,
// 	0x00b0, 0x00b1, 0x00b2, 0x00b3, 0x00b4, 0x00b5, 0x00b6, 0x00b7,
// 	0x00b8, 0x00b9, 0x00ba, 0x00bb, 0x00bc, 0x00bd, 0x00be, 0x00bf,
//
// 	0x00c0, 0x00c1, 0x00c2, 0x00c3, 0x00c4, 0x00c5, 0x00c6, 0x00c7,
// 	0x00c8, 0x00c9, 0x00ca, 0x00cb, 0x00cc, 0x00cd, 0x00ce, 0x00cf,
// 	0x00d0, 0x00d1, 0x00d2, 0x00d3, 0x00d4, 0x00d5, 0x00d6, 0x00d7,
// 	0x00d8, 0x00d9, 0x00da, 0x00db, 0x00dc, 0x00dd, 0x00de, 0x00df,
//
// 	0x00e0, 0x00e1, 0x00e2, 0x00e3, 0x00e4, 0x00e5, 0x00e6, 0x00e7,
// 	0x00e8, 0x00e9, 0x00ea, 0x00eb, 0x00ec, 0x00ed, 0x00ee, 0x00ef,
// 	0x00f0, 0x00f1, 0x00f2, 0x00f3, 0x00f4, 0x00f5, 0x00f6, 0x00f7,
// 	0x00f8, 0x00f9, 0x00fa, 0x00fb, 0x00fc, 0x00fd, 0x00fe, 0x00ff,
// }
//
// // Same as ASCII, but with Pound sign (0x00a3 in place of 0x0023)
// var uc_IsoUK = [96]rune{
// 	0x0020, 0x0021, 0x0022, 0x00a3, 0x0024, 0x0025, 0x0026, 0x0027,
// 	0x0028, 0x0029, 0x002a, 0x002b, 0x002c, 0x002d, 0x002e, 0x002f,
// 	0x0030, 0x0031, 0x0032, 0x0033, 0x0034, 0x0035, 0x0036, 0x0037,
// 	0x0038, 0x0039, 0x003a, 0x003b, 0x003c, 0x003d, 0x003e, 0x003f,
//
// 	0x0040, 0x0041, 0x0042, 0x0043, 0x0044, 0x0045, 0x0046, 0x0047,
// 	0x0048, 0x0049, 0x004a, 0x004b, 0x004c, 0x004d, 0x004e, 0x004f,
// 	0x0050, 0x0051, 0x0052, 0x0053, 0x0054, 0x0055, 0x0056, 0x0057,
// 	0x0058, 0x0059, 0x005a, 0x005b, 0x005c, 0x005d, 0x005e, 0x005f,
//
// 	0x0060, 0x0061, 0x0062, 0x0063, 0x0064, 0x0065, 0x0066, 0x0067,
// 	0x0068, 0x0069, 0x006a, 0x006b, 0x006c, 0x006d, 0x006e, 0x006f,
// 	0x0070, 0x0071, 0x0072, 0x0073, 0x0074, 0x0075, 0x0076, 0x0077,
// 	0x0078, 0x0079, 0x007a, 0x007b, 0x007c, 0x007d, 0x007e, 0x007f,
// }
//
// const (
// 	Charset_UTF8 = iota // sync w/charCodes definition!
// 	Charset_DecSpec
// 	Charset_DecSuppl
// 	Charset_DecUserPref
// 	Charset_DecTechn
// 	Charset_IsoLatin1
// 	Charset_IsoUK
// )
//
// // Sync this with enumerators of Charset!
// var charCodes = [...]([96]rune){
// 	[96]rune{}, // Dummy slot for UTF-8 (handled differently)
// 	uc_DecSpec,
// 	uc_DecSuppl,
// 	uc_DecSuppl, // Slot for 'User-preferred supplemental'
// 	uc_DecTechn,
// 	uc_IsoLatin1,
// 	uc_IsoUK,
// }

type CharsetState struct {
	// indicate vtMode charset or not, default false
	vtMode bool

	// charset g0,g1,g2,g3
	g [4]*map[byte]rune

	// Locking shift states (index into g[]):
	gl int
	gr int

	// Single shift state (0 if none active):
	// 0 - not active; 2: G2 in GL; 3: G3 in GL
	ss int
}

type emulator struct {
	dispatcher Dispatcher

	cf             *Framebuffer // current frame buffer
	primaryFrame   Framebuffer  // normal screen buffer
	alternateFrame Framebuffer  // alternate screen buffer

	charsetState CharsetState
	user         UserInput

	// local buffer for selection data
	selectionData map[rune]string
	// logger
	logE *log.Logger
	logT *log.Logger // trace
	logU *log.Logger
	logW *log.Logger
	logI *log.Logger

	nRows int
	nCols int

	posX         int // current cursor horizontal position (on-screen)
	posY         int // current cursor vertical position (on-screen)
	marginTop    int // current margin top (copy of frame field)
	marginBottom int // current margin bottom (copy of frame field)
	lastCol      bool
	attrs        Cell // prototype cell with current attributes
	fg           Color
	bg           Color
	reverseVideo bool

	// move states from drawstate
	showCursorMode      bool
	altScreenBufferMode bool // Alternate Screen Buffer support: default false
	autoWrapMode        bool // true/false
	autoNewlineMode     bool
	keyboardLocked      bool
	insertMode          bool // true/false
	bkspSendsDel        bool // backspace send delete
	localEcho           bool
	bracketedPasteMode  bool // true/false
	altScrollMode       bool
	altSendsEscape      bool

	horizMarginMode bool // left and right margins support
	hMargin         int  // left margins
	nColsEff        int  // right margins

	tabStops []int // tab stop positions

	compatLevel   CompatibilityLevel // VT52, VT100, VT400
	cursorKeyMode CursorKeyMode
	keypadMode    KeypadMode
	originMode    OriginMode // two possiible value: ScrollingRegion(true), Absolute(false)
	colMode       ColMode    // column mode 80 or 132, just for compatibility

	savedCursor_SCO     SavedCursor_SCO // SCO console cursor state
	savedCursor_DEC_pri SavedCursor_DEC
	savedCursor_DEC_alt SavedCursor_DEC
	savedCursor_DEC     *SavedCursor_DEC

	mouseTrk MouseTrackingState

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

func NewEmulator() *emulator {
	emu := &emulator{}
	emu.resetCharsetState()

	// defalult size 80x40
	emu.primaryFrame = *NewFramebuffer(80, 40)
	emu.cf = &emu.primaryFrame

	emu.initSelectionData()
	emu.initLog()
	return emu
}

func NewEmulator3(nCols, nRows, saveLines int) *emulator {
	// TODO makePalette256 (palette256);

	emu := &emulator{}
	emu.cf, emu.marginTop, emu.marginBottom = NewFramebuffer3(nCols, nRows, saveLines)
	emu.primaryFrame = *emu.cf

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
	emu.altScrollMode = false
	emu.altSendsEscape = true

	emu.horizMarginMode = false
	emu.nColsEff = emu.nCols
	emu.hMargin = 0

	emu.posX = 0
	emu.posY = 0
	emu.lastCol = false
	emu.fg = emu.attrs.renditions.fgColor
	emu.bg = emu.attrs.renditions.bgColor

	emu.savedCursor_DEC_pri = SavedCursor_DEC{}
	emu.savedCursor_DEC = &emu.savedCursor_DEC_pri
	emu.initSelectionData()
	emu.initLog()

	emu.resetTerminal()

	return emu
}

func (emu *emulator) resetTerminal() {
	emu.resetScreen()
	emu.resetAttrs()

	emu.switchColMode(ColMode_C80)
	emu.cf.dropScrollbackHistory()
	emu.marginTop, emu.marginBottom = emu.cf.resetMargins()
	emu.clearScreen()

	emu.switchScreenBufferMode(false)
	// TODO consider how to implemnt options parameters
	// emu.altScrollMode

	emu.horizMarginMode = false
	emu.hMargin = 0
	emu.nColsEff = emu.nCols
	// TODO checking hasOSCHandler
}

func (emu *emulator) resetScreen() {
	emu.showCursorMode = true
	emu.autoWrapMode = true
	emu.autoNewlineMode = false
	emu.keyboardLocked = false
	emu.insertMode = false
	emu.bkspSendsDel = true
	emu.localEcho = false
	emu.bracketedPasteMode = false

	emu.compatLevel = CompatLevel_VT400
	emu.cursorKeyMode = CursorKeyMode_ANSI
	emu.keypadMode = KeypadMode_Normal
	emu.originMode = OriginMode_Absolute
	emu.resetCharsetState()

	emu.savedCursor_SCO.isSet = false
	emu.savedCursor_DEC.isSet = false

	emu.mouseTrk = MouseTrackingState{}
	emu.tabStops = make([]int, 0)
	emu.cf.getSelection().clear()
}

func (emu *emulator) resetAttrs() {
	emu.reverseVideo = false
	emu.fg = emu.attrs.renditions.fgColor
	emu.bg = emu.attrs.renditions.bgColor

	// reset the character attributes
	params := []int{0} // preapare parameters for SGR
	hdl_csi_sgr(emu, params)
}

func (emu *emulator) clearScreen() {
	emu.posX = 0
	emu.posY = 0
	emu.lastCol = false
	emu.fillScreen(' ')
}

func (emu *emulator) fillScreen(ch rune) {
	emu.cf.fillCells(ch, emu.attrs)
}

func (emu *emulator) initSelectionData() {
	// prepare selection data for OSC 52
	// https://invisible-island.net/xterm/ctlseqs/ctlseqs.html#h3-Operating-System-Commands
	emu.selectionData = make(map[rune]string)
	emu.selectionData['c'] = "" // clipboard
	emu.selectionData['p'] = "" // primary
	emu.selectionData['q'] = "" // secondary
	emu.selectionData['s'] = "" // select
	emu.selectionData['0'] = "" // cut-buffer 0
	emu.selectionData['1'] = "" // cut-buffer 1
	emu.selectionData['2'] = "" // cut-buffer 2
	emu.selectionData['3'] = "" // cut-buffer 3
	emu.selectionData['4'] = "" // cut-buffer 4
	emu.selectionData['5'] = "" // cut-buffer 5
	emu.selectionData['6'] = "" // cut-buffer 6
	emu.selectionData['7'] = "" // cut-buffer 7
}

func (emu *emulator) initLog() {
	// init logger
	emu.logT = log.New(os.Stderr, "TRAC: ", log.Ldate|log.Ltime|log.Lshortfile)
	emu.logI = log.New(os.Stderr, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	emu.logE = log.New(os.Stderr, "ERRO: ", log.Ldate|log.Ltime|log.Lshortfile)
	emu.logW = log.New(os.Stderr, "WARN: ", log.Ldate|log.Ltime|log.Lshortfile)
	emu.logU = log.New(os.Stderr, "(Uimplemented): ", log.Ldate|log.Ltime|log.Lshortfile)
}

func (emu *emulator) resetCharsetState() {
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

func (emu *emulator) lookupCharset(p rune) (r rune) {
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

func (emu *emulator) resize(width, height int) {
	emu.cf.Resize(width, height)
}

func (emu *emulator) switchScreenBufferMode(altScreenBufferMode bool) {
	if emu.altScreenBufferMode == altScreenBufferMode {
		return
	}

	if altScreenBufferMode {
		emu.cf, emu.marginTop, emu.marginBottom = NewFramebuffer3(emu.nCols, emu.nRows, 0)
		emu.alternateFrame = *emu.cf

		emu.savedCursor_DEC = &emu.savedCursor_DEC_alt
		emu.altScreenBufferMode = true
	} else {
		emu.cf = &emu.primaryFrame
		emu.marginTop, emu.marginBottom = emu.cf.resize(emu.nCols, emu.nRows)
		emu.cf.expose()

		emu.savedCursor_DEC_alt.isSet = false
		emu.savedCursor_DEC = &emu.savedCursor_DEC_pri
		emu.altScreenBufferMode = false

	}
}

// TODO see the comments
func (emu *emulator) switchColMode(colMode ColMode) {
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

func (emu *emulator) normalizeCursorPos() {
	if emu.nColsEff < emu.posX+1 {
		emu.posX = emu.nColsEff - 1
	}
	if emu.nRows < emu.posY+1 {
		emu.posY = emu.nRows - 1
	}
}

func (emu *emulator) isCursorInsideMargins() bool {
	return emu.posX >= emu.cf.DS.hMargin && emu.posX < emu.cf.DS.nColsEff &&
		emu.posY >= emu.marginTop && emu.posY < emu.marginBottom
}

func (emu *emulator) eraseRow(pY int) {
	emu.cf.eraseInRow(pY, emu.hMargin, emu.nColsEff-emu.hMargin, emu.attrs)
}

// erase rows at and below startY, within the scrolling area
func (emu *emulator) eraseRows(startY, count int) {
	for pY := startY; pY < startY+count; pY++ {
		emu.eraseRow(pY)
	}
}

func (emu *emulator) copyRow(dstY, srcY int) {
	emu.cf.copyRow(dstY, srcY, emu.hMargin, emu.nColsEff-emu.hMargin)
}

// insert blank rows at and below startY, within the scrolling area
func (emu *emulator) insertRows(startY, count int) {
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

// delete rows at and below startY, within the scrolling area
func (emu *emulator) deleteRows(startY, count int) {
	for pY := startY; pY < emu.marginBottom-count; pY++ {
		emu.copyRow(pY, pY+count)
	}

	for pY := emu.marginBottom - count; pY < emu.marginBottom; pY++ {
		emu.eraseRow(pY)
	}
}

// insert blank cols at and to the right of startX, within the scrolling area
func (emu *emulator) insertCols(startX, count int) {
	for r := emu.marginTop; r < emu.marginBottom; r++ {
		emu.cf.moveInRow(r, startX+count, startX, emu.nColsEff-startX-count)
		emu.cf.eraseInRow(r, startX, count, emu.attrs) // use the default renditions
	}
}

// delete cols at and to the right of startX, within the scrolling area
func (emu *emulator) deleteCols(startX, count int) {
	for r := emu.marginTop; r < emu.marginBottom; r++ {
		emu.cf.moveInRow(r, startX, startX+count, emu.nColsEff-startX-count)
		emu.cf.eraseInRow(r, emu.nColsEff-count, count, emu.attrs) // use the default renditions
	}
}

func (emu *emulator) jumpToNextTabStop() {
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
	}
	// TODO tabStops set case
	emu.lastCol = false
}

/*
func (e *emulator) CSIdispatch(act Action) {
	e.dispatcher.dispatch(DISPATCH_CSI, act, &e.framebuffer)
}

func (e *emulator) ESCdispatch(act Action) {
	ch := act.GetChar()

	// handle 7-bit ESC-encoding of C1 control characters
	if len(e.dispatcher.getDispatcherChars()) == 0 && 0x40 <= ch && ch <= 0x5F {
		// convert 7-bit esc sequence into 8-bit c1 control sequence
		// TODO consider remove 8-bit c1 control
		act2 := escDispatch{action{ch + 0x40, true}}
		e.dispatcher.dispatch(DISPATCH_CONTROL, &act2, &e.framebuffer)
	} else {
		e.dispatcher.dispatch(DISPATCH_ESCAPE, act, &e.framebuffer)
	}
}

func (e *emulator) OSCdispatch(act Action) {
	e.dispatcher.oscDispatch(act, &e.framebuffer)
}

func (e *emulator) OSCend(act Action) {
}

func (e *emulator) Resize(act Action) {
}

*/
