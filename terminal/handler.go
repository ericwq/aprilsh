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

	"github.com/mattn/go-runewidth"
)

/* 64 - VT420 family
 *  1 - 132 columns
 *  9 - National Replacement Character-sets
 * 15 - DEC technical set
 * 21 - horizontal scrolling
 * 22 - color
 */
const (
	DEVICE_ID = "64;1;9;15;21;22c"
)

// Handler is the parsing result. It can be used to perform control sequence
// on emulator.
type Handler struct {
	name   string              // the name of ActOn
	ch     rune                // the last byte
	handle func(emu *emulator) // handle function that will perform control sequnce on emulator
}

// In the loop, national flag's width got 1+1=2.
func runesWidth(runes []rune) (width int) {
	// quick pass for iso8859-1
	if len(runes) == 1 && runes[0] < 0x00fe {
		return 1
	}

	cond := runewidth.NewCondition()
	cond.StrictEmojiNeutral = false
	cond.EastAsianWidth = true

	// loop for multi rune
	width = 0
	for i := 0; i < len(runes); i++ {
		width += cond.RuneWidth(runes[i])
	}

	return width
}

// print the graphic char to the emulator
// https://henvic.dev/posts/go-utf8/
// https://pkg.go.dev/golang.org/x/text/encoding/charmap
// https://github.com/rivo/uniseg
func hdl_graphemes(emu *emulator, chs ...rune) {
	// fmt.Printf("hdl_graphemes got %q", chs)

	if len(chs) == 1 && emu.charsetState.vtMode {
		chs[0] = emu.lookupCharset(chs[0])
	}
	// 	fmt.Printf("   VT   : %q, %U, %x w=%d\n", chs, chs, chs, runesWidth(chs))
	// } else {
	// 	fmt.Printf("   UTF-8: %q, %U, %x w=%d\n", chs, chs, chs, runesWidth(chs))
	// }

	// get current cursor cell
	fb := emu.framebuffer
	thisCell := fb.GetCell(-1, -1)
	chWidth := runesWidth(chs)

	if fb.DS.AutoWrapMode && fb.DS.NextPrintWillWrap {
		fb.GetRow(-1).SetWrap(true)
		fb.DS.MoveCol(0, false, false)
		fb.MoveRowsAutoscroll(1)
		thisCell = nil
	} else if fb.DS.AutoWrapMode && chWidth == 2 && fb.DS.GetCursorCol() == fb.DS.GetWidth()-1 {
		// wrap 2-cell chars if no room, even without will-wrap flag
		fb.ResetCell(thisCell)
		fb.GetRow(-1).SetWrap(false)

		// There doesn't seem to be a consistent way to get the
		// downstream terminal emulator to set the wrap-around
		// copy-and-paste flag on a row that ends with an empty cell
		// because a wide char was wrapped to the next line.

		fb.DS.MoveCol(0, false, false)
		fb.MoveRowsAutoscroll(1)
		thisCell = nil
	}

	if fb.DS.InsertMode {
		for i := 0; i < chWidth; i++ {
			fb.InsertCell(fb.DS.GetCursorRow(), fb.DS.GetCursorCol())
		}
		thisCell = nil
	}

	// fmt.Printf("print@(%d,%d) chs=%q\n", emu.framebuffer.DS.GetCursorRow(), emu.framebuffer.DS.GetCursorCol(), chs)
	if thisCell == nil {
		thisCell = fb.GetCell(-1, -1)
	}

	// set the cell: content, wide and rendition
	fb.ResetCell(thisCell)
	for _, r := range chs {
		thisCell.Append(r)
	}
	if chWidth == 2 {
		thisCell.SetWide(true)
	} else {
		thisCell.SetWide(false)
	}
	fb.ApplyRenditionsToCell(thisCell)

	if chWidth == 2 { // erase overlapped cell
		if fb.DS.GetCursorCol()+1 < fb.DS.GetWidth() {
			nextCell := fb.GetCell(fb.DS.GetCursorRow(), fb.DS.GetCursorCol()+1)
			fb.ResetCell(nextCell)
		}
	}

	// move cursor to the next position
	fb.DS.MoveCol(chWidth, true, true)
}

// move cursor to the count tab position
func ht_n(fb *Framebuffer, count int) {
	col := fb.DS.GetNextTab(count)
	if col == -1 { // no tabs, go to end of line
		col = fb.DS.GetWidth() - 1
	}
	// A horizontal tab is the only operation that preserves but
	// does not set the wrap state. It also starts a new grapheme.
	wrapStateSave := fb.DS.NextPrintWillWrap
	fb.DS.MoveCol(col, false, false)
	fb.DS.NextPrintWillWrap = wrapStateSave
}

// Horizontal Tab (HTS  is Ctrl-I).
// move cursor to the next tab position
func hdl_c0_ht(emu *emulator) {
	ht_n(emu.framebuffer, 1)
}

// Bell (BEL  is Ctrl-G).
// ring the bell
func hdl_c0_bel(emu *emulator) {
	emu.framebuffer.RingBell()
}

// FF, VT same as LF
// move cursor to the next row, scroll down if necessary.
func hdl_c0_lf(emu *emulator) {
	emu.framebuffer.MoveRowsAutoscroll(1)
}

// Carriage Return (CR  is Ctrl-M).
// move cursor to the head of the same row
func hdl_c0_cr(emu *emulator) {
	emu.framebuffer.DS.MoveCol(0, false, false)
}

// SI       Switch to Standard Character Set (Ctrl-O is Shift In or LS0).
//          This invokes the G0 character set (the default) as GL.
//          VT200 and up implement LS0.
func hdl_c0_si(emu *emulator) {
	emu.charsetState.gl = 0
}

// SO       Switch to Alternate Character Set (Ctrl-N is Shift Out or
//          LS1).  This invokes the G1 character set as GL.
//          VT200 and up implement LS1.
func hdl_c0_so(emu *emulator) {
	emu.charsetState.gl = 1
}

// ESC N
//     Single Shift Select of G2 Character Set (SS2  is 0x8e), VT220.
//     This affects next character only.
func hdl_esc_ss2(emu *emulator) {
	emu.charsetState.ss = 2
}

// ESC O
//     Single Shift Select of G3 Character Set (SS3  is 0x8f), VT220.
//     This affects next character only.
func hdl_esc_ss3(emu *emulator) {
	emu.charsetState.ss = 3
}

// ESC ~     Invoke the G1 Character Set as GR (LS1R), VT100.
func hdl_esc_ls1r(emu *emulator) {
	emu.charsetState.gr = 1
}

// ESC n     Invoke the G2 Character Set as GL (LS2).
func hdl_esc_ls2(emu *emulator) {
	emu.charsetState.gl = 2
}

// ESC }     Invoke the G2 Character Set as GR (LS2R).
func hdl_esc_ls2r(emu *emulator) {
	emu.charsetState.gr = 2
}

// ESC o     Invoke the G3 Character Set as GL (LS3).
func hdl_esc_ls3(emu *emulator) {
	emu.charsetState.gl = 3
}

// ESC |     Invoke the G3 Character Set as GR (LS3R).
func hdl_esc_ls3r(emu *emulator) {
	emu.charsetState.gr = 3
}

// ESC % G   Select UTF-8 character set, ISO 2022.
// https://en.wikipedia.org/wiki/ISO/IEC_2022#Interaction_with_other_coding_systems
func hdl_esc_docs_utf8(emu *emulator) {
	emu.resetCharsetState()
}

// ESC % @   Select default character set.  That is ISO 8859-1 (ISO 2022).
// https://www.cl.cam.ac.uk/~mgk25/unicode.html#utf-8
func hdl_esc_docs_iso8859_1(emu *emulator) {
	emu.resetCharsetState()
	emu.charsetState.g[emu.charsetState.gr] = &vt_ISO_8859_1 // Charset_IsoLatin1
	emu.charsetState.vtMode = true
}

// Select G0 ~ G3 character set based on parameter
func hdl_esc_dcs(emu *emulator, index int, charset *map[byte]rune) {
	emu.charsetState.g[index] = charset
	if charset != nil {
		emu.charsetState.vtMode = true
	}
}

// horizontal tab set
// ESC H Tab Set (HTS is 0x88).
// set cursor position as tab stop position
func hdl_esc_hts(emu *emulator) {
	emu.framebuffer.DS.SetTab()
}

// ESC M  Reverse Index (RI  is 0x8d).
// reverse index -- like a backwards line feed
func hdl_esc_ri(emu *emulator) {
	emu.framebuffer.MoveRowsAutoscroll(-1)
}

// ESC E  Next Line (NEL  is 0x85).
func hdl_esc_nel(emu *emulator) {
	emu.framebuffer.DS.MoveCol(0, false, false)
	emu.framebuffer.MoveRowsAutoscroll(1)
}

// ESC c     Full Reset (RIS), VT100.
// reset the screen
func hdl_esc_ris(emu *emulator) {
	emu.framebuffer.Reset()
}

// ESC 7     Save Cursor (DECSC), VT100.
func hdl_esc_decsc(emu *emulator) {
	emu.framebuffer.DS.SaveCursor()
}

// ESC 8     Restore Cursor (DECRC), VT100.
func hdl_esc_decrc(emu *emulator) {
	emu.framebuffer.DS.RestoreCursor()
}

// ESC # 8   DEC Screen Alignment Test (DECALN), VT100.
// fill the screen with 'E'
func hdl_esc_decaln(emu *emulator) {
	fb := emu.framebuffer
	for y := 0; y < fb.DS.GetHeight(); y++ {
		for x := 0; x < fb.DS.GetWidth(); x++ {
			fb.ResetCell(fb.GetCell(y, x))
			fb.GetCell(y, x).Append('E')
		}
	}
}

// CSI Ps g  Tab Clear (TBC).
//            Ps = 0  ⇒  Clear Current Column (default).
//            Ps = 3  ⇒  Clear All.
func hdl_csi_tbc(emu *emulator, cmd int) {
	switch cmd {
	case 0: // clear this tab stop
		emu.framebuffer.DS.ClearTab(emu.framebuffer.DS.GetCursorCol())
	case 3: // clear all tab stops
		emu.framebuffer.DS.ClearDefaultTabs()
		for i := 0; i < emu.framebuffer.DS.GetWidth(); i++ {
			emu.framebuffer.DS.ClearTab(i)
		}
	}
}

// CSI Ps I  Cursor Forward Tabulation Ps tab stops (default = 1) (CHT).
func hdl_csi_cht(emu *emulator, count int) {
	ht_n(emu.framebuffer, count)
}

// CSI Ps Z  Cursor Backward Tabulation Ps tab stops (default = 1) (CBT).
func hdl_csi_cbt(emu *emulator, count int) {
	ht_n(emu.framebuffer, -count)
}

// CSI Ps @  Insert Ps (Blank) Character(s) (default = 1) (ICH).
func hdl_csi_ich(emu *emulator, count int) {
	fb := emu.framebuffer
	for i := 0; i < count; i++ {
		fb.InsertCell(fb.DS.GetCursorRow(), fb.DS.GetCursorCol())
	}
}

// CSI Ps J Erase in Display (ED), VT100.
// * Ps = 0  ⇒  Erase Below (default).
// * Ps = 1  ⇒  Erase Above.
// * Ps = 2  ⇒  Erase All.
// * Ps = 3  ⇒  Erase Saved Lines, xterm.
func hdl_csi_ed(emu *emulator, cmd int) {
	fb := emu.framebuffer
	switch cmd {
	case 0:
		// active position down to end of screen, inclusive
		clearline(fb, -1, fb.DS.GetCursorCol(), fb.DS.GetWidth()-1)
		for y := fb.DS.GetCursorRow() + 1; y < fb.DS.GetHeight(); y++ {
			fb.ResetRow(fb.GetRow(y))
		}
	case 1:
		// start of screen to active position, inclusive
		for y := 0; y < fb.DS.GetCursorRow(); y++ {
			fb.ResetRow(fb.GetRow(y))
		}
		clearline(fb, -1, 0, fb.DS.GetCursorCol())
	case 2:
		//  entire screen
		for y := 0; y < fb.DS.GetHeight(); y++ {
			fb.ResetRow(fb.GetRow(y))
		}
	}
}

// CSI Ps K Erase in Line (EL), VT100.
// * Ps = 0  ⇒  Erase to Right (default).
// * Ps = 1  ⇒  Erase to Left.
// * Ps = 2  ⇒  Erase All.
func hdl_csi_el(emu *emulator, cmd int) {
	fb := emu.framebuffer
	switch cmd {
	case 0:
		clearline(fb, -1, fb.DS.GetCursorCol(), fb.DS.GetWidth()-1)
	case 1:
		clearline(fb, -1, 0, fb.DS.GetCursorCol())
	case 2:
		fb.ResetRow(fb.GetRow(-1))
	}
}

// CSI Ps L  Insert Ps Line(s) (default = 1) (IL).
// insert N lines in cursor position
func hdl_csi_il(emu *emulator, lines int) {
	fb := emu.framebuffer
	fb.InsertLine(fb.DS.GetCursorRow(), lines)

	// vt220 manual and Ecma-48 say to move to first column */
	fb.DS.MoveCol(0, false, false)
}

// CSI Ps M  Delete Ps Line(s) (default = 1) (DL).
// delete N lines in cursor position
func hdl_csi_dl(emu *emulator, lines int) {
	fb := emu.framebuffer

	fb.DeleteLine(fb.DS.GetCursorRow(), lines)

	// vt220 manual and Ecma-48 say to move to first column */
	fb.DS.MoveCol(0, false, false)
}

// CSI Ps P  Delete Ps Character(s) (default = 1) (DCH).
func hdl_csi_dch(emu *emulator, cells int) {
	fb := emu.framebuffer

	for i := 0; i < cells; i++ {
		fb.DeleteCell(fb.DS.GetCursorRow(), fb.DS.GetCursorCol())
	}
}

// CSI Ps S  Scroll up Ps lines (default = 1) (SU), VT420, ECMA-48.
// CSI Ps T  Scroll down Ps lines (default = 1) (SD), VT420.
// SU got the -lines
func hdl_csi_su_sd(emu *emulator, lines int) {
	emu.framebuffer.Scroll(lines)
}

// erase cell from the start to end at specified row
func clearline(fb *Framebuffer, row int, start int, end int) {
	for col := start; col <= end; col++ {
		fb.ResetCell(fb.GetCell(row, col))
	}
}

// CSI Ps X  Erase Ps Character(s) (default = 1) (ECH).
func hdl_csi_ech(emu *emulator, num int) {
	fb := emu.framebuffer

	limit := fb.DS.GetCursorCol() + num - 1

	if limit >= fb.DS.GetWidth() {
		limit = fb.DS.GetWidth() - 1
	}

	clearline(fb, -1, fb.DS.GetCursorCol(), limit)
}

// CSI Ps c  Send Device Attributes (Primary DA).
// CSI ? 6 2 ; Ps c  ("VT220")
// DA response
func hdl_csi_da1(emu *emulator) {
	// mosh only reply "\x1B[?62c" plain vt220
	da1Response := fmt.Sprintf("\x1B[?%s", DEVICE_ID)
	emu.dispatcher.terminalToHost.WriteString(da1Response)
}

// CSI > Ps c Send Device Attributes (Secondary DA).
// Ps = 0  or omitted ⇒  request the terminal's identification code.
// CSI > Pp ; Pv ; Pc c
// Pp = 1  ⇒  "VT220".
// Pv is the firmware version.
// Pc indicates the ROM cartridge registration number and is always zero.
func hdl_csi_da2(emu *emulator) {
	// mosh only reply "\033[>1;10;0c" plain vt220
	da2Response := "\x1B[>64;0;0c" // VT520
	emu.dispatcher.terminalToHost.WriteString(da2Response)
}

// CSI Ps d  Line Position Absolute  [row] (default = [1,column]) (VPA).
func hdl_csi_vpa(emu *emulator, row int) {
	emu.framebuffer.DS.MoveRow(row-1, false)
}

// Move the active position to the n-th character of the active line.
// CHA—Cursor Horizontal Absolute
// CSI Ps G  Cursor Character Absolute  [column] (default = [row,1]) (CHA).
// CSI Ps `  Character Position Absolute  [column] (default = [row,1]) (HPA).
func hdl_csi_cha_hpa(emu *emulator, count int) {
	emu.framebuffer.DS.MoveCol(count-1, false, false)
}

// CSI Ps A  Cursor Up Ps Times (default = 1) (CUU).
func hdl_csi_cuu(emu *emulator, num int) {
	emu.framebuffer.DS.MoveRow(-num, true)
}

// CSI Ps B  Cursor Down Ps Times (default = 1) (CUD).
func hdl_csi_cud(emu *emulator, num int) {
	emu.framebuffer.DS.MoveRow(num, true)
}

// CSI Ps C  Cursor Forward Ps Times (default = 1) (CUF).
func hdl_csi_cuf(emu *emulator, num int) {
	emu.framebuffer.DS.MoveCol(num, true, false)
}

// CSI Ps D  Cursor Backward Ps Times (default = 1) (CUB).
func hdl_csi_cub(emu *emulator, num int) {
	emu.framebuffer.DS.MoveCol(-num, true, false)
}

// CSI Ps ; Ps H Cursor Position [row;column] (default = [1,1]) (CUP).
// CSI Ps ; Ps f Horizontal and Vertical Position [row;column] (default = [1,1]) (HVP).
func hdl_csi_cup(emu *emulator, row int, col int) {
	emu.framebuffer.DS.MoveRow(row-1, false)
	emu.framebuffer.DS.MoveCol(col-1, false, false)
}

func hdl_osc_10(_ *emulator, cmd int, arg string) {
	// TODO not finished
	fmt.Printf("handle osc dynamic cmd=%d, arg=%s\n", cmd, arg)
}

func hdl_osc_52(_ *emulator, cmd int, arg string) {
	// TODO not finished
	fmt.Printf("handle osc copy cmd=%d, arg=%s\n", cmd, arg)
}

func hdl_osc_4(_ *emulator, cmd int, arg string) {
	// TODO not finished
	fmt.Printf("handle osc palette cmd=%d, arg=%s\n", cmd, arg)
}

// OSC of the form "\x1B]X;<title>\007" where X can be:
//* 0: set icon name and window title
//* 1: set icon name
//* 2: set window title
func hdl_osc_0(emu *emulator, cmd int, arg string) {
	// set icon name / window title
	setIcon := cmd == 0 || cmd == 1
	setTitle := cmd == 0 || cmd == 2
	if setIcon || setTitle {
		emu.framebuffer.SetTitleInitialized()

		if setIcon {
			emu.framebuffer.SetIconName(arg)
		}

		if setTitle {
			emu.framebuffer.SetWindowTitle(arg)
		}
	}
}
