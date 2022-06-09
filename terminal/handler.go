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
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

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
	name   string              // the name of Handler
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

// CSI Ps n  Device Status Report (DSR).
//   Ps = 5  ⇒  Status Report. Result ("OK") is CSI 0 n
//   Ps = 6  ⇒  Report Cursor Position (CPR) [row;column]. Result is CSI r ; c R
func hdl_csi_dsr(emu *emulator, cmd int) {
	switch cmd {
	case 5:
		// device status report requested
		emu.dispatcher.terminalToHost.WriteString("\x1B[0n") // device OK
	case 6:
		resp := ""
		// report of active position requested
		if emu.framebuffer.DS.OriginMode { // original mode
			resp = fmt.Sprintf("\x1B[%d;%dR", emu.framebuffer.DS.GetCursorRow()+1,
				emu.framebuffer.DS.GetCursorCol()+1)
		} else { // scrolling region mode
			resp = fmt.Sprintf("\x1B[%d;%dR", emu.framebuffer.DS.GetCursorRow()-emu.framebuffer.DS.GetScrollingRegionTopRow()+1,
				emu.framebuffer.DS.GetCursorCol()+1)
		}
		emu.dispatcher.terminalToHost.WriteString(resp)
	default:
	}
}

// CSI Pm m  Character Attributes (SGR).
// select graphics rendition -- e.g., bold, blinking, etc.
// support 8, 16, 256 color, RGB color.
func hdl_csi_sgr(emu *emulator, params []int) {
	// we need to change the field, get the field pointer
	rend := &emu.framebuffer.DS.renditions
	for k := 0; k < len(params); k++ {
		attr := params[k]

		// process the 8-color set, 16-color set and default color
		if rend.buildRendition(attr) {
			continue
		}
		switch attr {
		case 38:
			if k >= len(params)-1 {
				break
			}
			switch params[k+1] {
			case 5:
				if k+1 >= len(params)-1 {
					k += len(params) - 1 - k
					break
				}
				rend.SetForegroundColor(params[k+2])
				k += 2
			case 2:
				if k+1 >= len(params)-3 {
					k += len(params) - 1 - k
					break
				}
				red := params[k+2]
				green := params[k+3]
				blue := params[k+4]
				rend.SetFgColor(red, green, blue)
				k += 4
			default:
				k += len(params) - 1 - k
			}
		case 48:
			if k >= len(params)-1 {
				break
			}
			switch params[k+1] {
			case 5:
				if k+1 >= len(params)-1 {
					k += len(params) - 1 - k
					break
				}
				rend.SetBackgroundColor(params[k+2])
				k += 2
			case 2:
				if k+1 >= len(params)-3 {
					k += len(params) - 1 - k
					break
				}
				red := params[k+2]
				green := params[k+3]
				blue := params[k+4]
				rend.SetBgColor(red, green, blue)
				k += 4
			default:
				k += len(params) - 1 - k
			}
		default:
			emu.logU.Printf("attribute not supported. %d \n", attr)
		}
	}
}

/*

OSC Ps ; Pt ST
          The 10 colors (below) which may be set or queried using 1 0
          through 1 9  are denoted dynamic colors, since the
          corresponding control sequences were the first means for
          setting xterm's colors dynamically, i.e., after it was
          started.  They are not the same as the ANSI colors (however,
          the dynamic text foreground and background colors are used
          when ANSI colors are reset using SGR 3 9  and 4 9 ,
          respectively).  These controls may be disabled using the
          allowColorOps resource.  At least one parameter is expected
          for Pt.  Each successive parameter changes the next color in
          the list.  The value of Ps tells the starting point in the
          list.  The colors are specified by name or RGB specification
          as per XParseColor.

          If a "?" is given rather than a name or RGB specification,
          xterm replies with a control sequence of the same form which
          can be used to set the corresponding dynamic color.  Because
          more than one pair of color number and specification can be
          given in one control sequence, xterm can make more than one
          reply.

            Ps = 1 0  ⇒  Change VT100 text foreground color to Pt.
            Ps = 1 1  ⇒  Change VT100 text background color to Pt.
            Ps = 1 2  ⇒  Change text cursor color to Pt.
            Ps = 1 3  ⇒  Change pointer foreground color to Pt.
            Ps = 1 4  ⇒  Change pointer background color to Pt.
            Ps = 1 5  ⇒  Change Tektronix foreground color to Pt.
            Ps = 1 6  ⇒  Change Tektronix background color to Pt.
            Ps = 1 7  ⇒  Change highlight background color to Pt.
            Ps = 1 8  ⇒  Change Tektronix cursor color to Pt.
            Ps = 1 9  ⇒  Change highlight foreground color to Pt.
*/
func hdl_osc_10x(emu *emulator, cmd int, arg string) {
	arg = fmt.Sprintf("%d;%s", cmd, arg) // add the cmd back to the arg
	count := strings.Count(arg, ";")
	if count > 0 && count%2 == 1 { // color pair has 2n-1 ';'
		pairs := (count + 1) / 2 // count the color pair
		if pairs >= 5 {          // limit the color pair up to 5
			pairs = 5
		}

		args := strings.Split(arg, ";")
		idx := 0
		for i := 0; i < pairs; i++ {

			idx = i * 2
			color := args[idx]
			action := args[idx+1]

			// we only support query for the time being.
			if action == "?" {
				colorIdx, err := strconv.Atoi(color)
				if err != nil {
					emu.logW.Printf("OSC 10x: can't parse color index. %q\n", arg)
					return
				}

				color := ColorDefault
				switch colorIdx {
				case 11, 17: // 11: VT100 text background color  17: highlight background color
					color = emu.framebuffer.DS.renditions.bgColor
				case 10, 19: // 10: VT100 text foreground color; 19: highlight foreground color
					color = emu.framebuffer.DS.renditions.fgColor
				case 12: // 12: text cursor color
					color = emu.framebuffer.DS.cursorColor
				}
				response := fmt.Sprintf("\x1B]%d;%s\x1B\\", colorIdx, color) // the String() method of Color will be called.
				emu.dispatcher.terminalToHost.WriteString(response)
			}
		}
	} else {
		emu.logW.Printf("OSC 10x: malformed argument, missing ';'. %q\n", arg)
		return
	}
}

/*
OSC Ps ; Pt ST
Ps = 5 2  ⇒ Manipulate Selection Data. The parameter Pt is parsed as
			Pc ; Pd
1. use one of the following commands to encode the original text
and set the system clipboard
	% echo -e "\033]52;c;$(base64 <<< hello)\a"
	% echo "Hello Russia!" | base64
	SGVsbG8gUnVzc2lhIQo=
	% echo  -e "\033]52;p;SGVsbG8gUnVzc2lhIQo=\a"
2. press the paste hot-eky, you will see the original text in step 1.
	% hello
	% Hello Russia!
*/
func hdl_osc_52(emu *emulator, cmd int, arg string) {
	// parse Pc:Pd
	pos := strings.Index(arg, ";")
	if pos == -1 {
		emu.logW.Printf("OSC 52: can't find Pc parameter. %q\n", arg)
		return
	}
	Pc := arg[:pos]
	Pd := arg[pos+1:]

	/*
		If the parameter is empty, xterm uses s 0 , to specify the configurable
		primary/clipboard selection and cut-buffer 0.
	*/
	if Pc == "" {
		Pc = "s0"
	}

	// validate Pc
	if !osc52InRange(Pc) {
		emu.logW.Printf("OSC 52: invalid Pc parameters. %q\n", Pc)
		return
	}

	if Pd == "?" {
		/*
			If the second parameter is a ? , xterm replies to the host
			with the selection data encoded using the same protocol.  It
			uses the first selection found by asking successively for each
			item from the list of selection parameters.
		*/
		for _, ch := range Pc {
			if data, ok := emu.selectionData[ch]; ok && data != "" {
				// response to the host
				response := fmt.Sprintf("\x1B]%d;%c;%s\x1B\\", cmd, ch, data)
				emu.dispatcher.terminalToHost.WriteString(response)
				break
			}
		}
	} else {
		// TODO please consider the race condition
		_, err := base64.StdEncoding.DecodeString(Pd)
		set := false
		if err == nil { // it's a base64 string
			/*
			   The second parameter, Pd, gives the selection data.  Normally
			   this is a string encoded in base64 (RFC-4648).  The data
			   becomes the new selection, which is then available for pasting
			   by other applications.
			*/
			for _, ch := range Pc {
				if _, ok := emu.selectionData[ch]; ok { // make sure Pc exist
					// update the new selection in local cache
					emu.selectionData[ch] = Pd
					set = true
				}
			}
			if set {
				// save the selection in framebuffer, later it will be sent to terminal.
				emu.framebuffer.selectionData += fmt.Sprintf("\x1B]%d;%s;%s\x1B\\", cmd, Pc, Pd)
			}
		} else {
			/*
			   If the second parameter is neither a base64 string nor ? ,
			   then the selection is cleared.
			*/
			for _, ch := range Pc {
				if _, ok := emu.selectionData[ch]; ok { // make sure Pc exist
					// clear the selection in local cache
					emu.selectionData[ch] = ""
					set = true
				}
			}
			if set {
				// save the selection in framebuffer, later it will be sent to terminal.
				emu.framebuffer.selectionData = fmt.Sprintf("\x1B]%d;%s;%s\x1B\\", cmd, Pc, Pd)
			}
		}
	}
}

// validate hte Pc content is in the specified set
func osc52InRange(Pc string) (ret bool) {
	specSet := "cpqs01234567"
	for _, ch := range Pc {
		if !strings.Contains(specSet, string(ch)) {
			return false
		}
	}

	return true
}

/*
Ps = 4 ; c ; spec ⇒  Change Color Number c to the color specified by spec.

The spec can be a name or RGB specification as per
XParseColor.  Any number of c/spec pairs may be given.  The
color numbers correspond to the ANSI colors 0-7, their bright
versions 8-15, and if supported, the remainder of the 88-color
or 256-color table.

If a "?" is given rather than a name or RGB specification,
xterm replies with a control sequence of the same form which
can be used to set the corresponding color.  Because more than
one pair of color number and specification can be given in one
control sequence, xterm can make more than one reply.

string names for colors: https://tronche.com/gui/x/xlib/color/strings/
xterm 256 color protocol: https://unix.stackexchange.com/questions/105568/how-can-i-list-the-available-color-names
color and formatting: https://misc.flogisoft.com/bash/tip_colors_and_formatting

*/
func hdl_osc_4(emu *emulator, cmd int, arg string) {
	count := strings.Count(arg, ";")
	if count > 0 && count%2 == 1 { // c/spec pair has 2n-1 ';'
		pairs := (count + 1) / 2 // count the c/spec pair
		if pairs >= 8 {          // limit the c/spec pair up to 8
			pairs = 8
		}

		args := strings.Split(arg, ";")
		idx := 0
		for i := 0; i < pairs; i++ {

			idx = i * 2
			c := args[idx]
			spec := args[idx+1]

			// we only support query for the time being.
			if spec == "?" {
				colorIdx, err := strconv.Atoi(c)
				if err != nil {
					emu.logW.Printf("OSC 4: can't parse c parameter. %q\n", arg)
					return
				}
				color := PaletteColor(colorIdx)
				response := fmt.Sprintf("\x1B]%d;%d;%s\x1B\\", cmd, colorIdx, color)
				emu.dispatcher.terminalToHost.WriteString(response)
			}
		}
	} else {
		emu.logW.Printf("OSC 4: malformed argument, missing ';'. %q\n", arg)
		return
	}
}

// OSC of the form "\x1B]X;<title>\007" where X can be:
//* 0: set icon name and window title
//* 1: set icon name
//* 2: set window title
func hdl_osc_0_1_2(emu *emulator, cmd int, arg string) {
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

// CSI Pm h  Set Mode (SM).
// *  Ps = 2  ⇒  Keyboard Action Mode (KAM).
// *  Ps = 4  ⇒  Insert Mode (IRM).
// *  Ps = 1 2  ⇒  Send/receive (SRM).
// *  Ps = 2 0  ⇒  Automatic Newline (LNM).
func hdl_csi_sm(emu *emulator, params []int) {
	for _, param := range params {
		switch param {
		case 2:
			emu.framebuffer.DS.keyboardLocked = true
		case 4:
			emu.framebuffer.DS.InsertMode = true // zutty:insertMode
		case 12:
			emu.framebuffer.DS.localEcho = false
		case 20:
			emu.framebuffer.DS.autoNewlineMode = true
		default:
			emu.logW.Printf("CSI SM: Ignored bogus set mode %d.\n", param)
		}
	}
}

// CSI Pm l  Reset Mode (RM).
// *  Ps = 2  ⇒  Keyboard Action Mode (KAM).
// *  Ps = 4  ⇒  Replace Mode (IRM).
// *  Ps = 1 2  ⇒  Send/receive (SRM).
// *  Ps = 2 0  ⇒  Normal Linefeed (LNM).
func hdl_csi_rm(emu *emulator, params []int) {
	for _, param := range params {
		switch param {
		case 2:
			emu.framebuffer.DS.keyboardLocked = false
		case 4:
			emu.framebuffer.DS.InsertMode = false // zutty:insertMode
		case 12:
			emu.framebuffer.DS.localEcho = true
		case 20:
			emu.framebuffer.DS.autoNewlineMode = false
		default:
			emu.logW.Printf("CSI RM: Ignored bogus reset mode %d.\n", param)
		}
	}
}

// CSI ? Pm h
// DEC Private Mode Set (DECSET).
func hdl_csi_decset(emu *emulator, params []int) {
	for _, param := range params {
		switch param {
		case 1:
			emu.framebuffer.DS.ApplicationModeCursorKeys = true // DECCKM Apllication zutty:cursorKeyMode
		case 2:
			emu.resetCharsetState()
			emu.framebuffer.DS.compatLevel = CompatLevelVT400
		case 3:
			emu.logU.Println("TODO switchColMode(ColMode::C132) zutty vterm.icc line 1427") // mosh terminalfunctions.cc line 256
		case 4:
			emu.logT.Println("DECSCLM: Set smooth scroll")
		case 5:
			emu.framebuffer.DS.ReverseVideo = true // DECSCNM Reverse
		case 6:
			emu.framebuffer.DS.OriginMode = true // DECOM ScrollingRegion zutty:originMode
		case 7:
			emu.framebuffer.DS.AutoWrapMode = true // DECAWM zutty:autoWrapMode
		case 8:
			emu.logU.Println("DECARM: Set auto-repeat mode")
		case 9:
			emu.framebuffer.DS.mouseTrk.mode = MouseModeX10
		case 12:
			emu.logU.Println("Start blinking cursor")
		case 25:
			emu.framebuffer.DS.CursorVisible = true // DECTCEM zutty:showCursorMode
		case 47:
			emu.logU.Println("TODO switchScreenBufferMode(true) zutty vterm.icc line 1436")
		case 67:
			emu.logU.Println("TODO zutty vterm.icc line 1437")
		case 69:
			emu.logU.Println("TODO zutty vterm.icc line 1438")
		case 1000:
			emu.framebuffer.DS.mouseTrk.mode = MouseModeVT200
		case 1001:
			emu.logU.Println("Set VT200 Highlight Mouse mode")
		case 1002:
			emu.framebuffer.DS.mouseTrk.mode = MouseModeButtonEvent
		case 1003:
			emu.framebuffer.DS.mouseTrk.mode = MouseModeAnyEvent
		case 1004:
			// TODO replace MouseFocusEvent with mouseTrk.focusEventMode
			emu.framebuffer.DS.MouseFocusEvent = true // xterm zutty:mouseTrk.focusEventMode
			emu.framebuffer.DS.mouseTrk.focusEventMode = true
		case 1005:
			emu.framebuffer.DS.mouseTrk.enc = MouseEncUTF
		case 1006:
			emu.framebuffer.DS.mouseTrk.enc = MouseEncSGR
		case 1007:
			emu.framebuffer.DS.MouseAlternateScroll = true // xterm zutty:altScrollMode
		case 1015:
			emu.framebuffer.DS.mouseTrk.enc = MouseEncURXVT
		case 1036, 1039:
			emu.framebuffer.DS.altSendsEscape = true
		case 1047:
			emu.logU.Println("TODO switchScreenBufferMode(true) zutty vterm.icc line 1449")
		case 1048:
			hdl_esc_decsc(emu)
		case 1049:
			hdl_esc_decsc(emu)
			emu.logU.Println("TODO switchScreenBufferMode(true) zutty vterm.icc line 1451")
		case 2004:
			emu.framebuffer.DS.BracketedPaste = true // xterm zutty:bracketedPasteMode
		default:
			emu.logU.Printf("set priv mode %d\n", param)
		}
	}
}

// CSI ? Pm l
// DEC Private Mode Reset (DECRST).
func hdl_csi_decrst(emu *emulator, params []int) {
	for _, param := range params {
		switch param {
		case 1:
			emu.framebuffer.DS.ApplicationModeCursorKeys = false // ANSI
		case 2:
			emu.resetCharsetState()
			emu.framebuffer.DS.compatLevel = CompatLevelVT52
		case 3:
			emu.logU.Println("TODO switchColMode(ColMode::C80) zutty vterm.icc line 1476") // mosh terminalfunctions.cc line 256
		case 4:
			emu.logT.Println("DECSCLM: Set jump scroll")
		case 5:
			emu.framebuffer.DS.ReverseVideo = false // Normal
		case 6:
			emu.framebuffer.DS.OriginMode = false // Absolute
		case 7:
			emu.framebuffer.DS.AutoWrapMode = false
		case 8:
			emu.logU.Println("DECARM: Reset auto-repeat mode")
		case 9, 1000, 1002, 1003:
			emu.framebuffer.DS.mouseTrk.mode = MouseModeNone
		case 12:
			emu.logU.Println("Stop blinking cursor")
		case 25:
			emu.framebuffer.DS.CursorVisible = false
		case 47:
			emu.logU.Println("TODO switchScreenBufferMode(false) zutty vterm.icc line 1486")
		case 67:
			emu.logU.Println("TODO zutty vterm.icc line 1487")
		case 69:
			emu.logU.Println("TODO zutty vterm.icc line 1488")
		case 1001:
			emu.logU.Println("Reset VT200 Highlight Mouse mode")
		case 1004:
			// TODO replace MouseFocusEvent with mouseTrk.focusEventMode
			emu.framebuffer.DS.MouseFocusEvent = false
			emu.framebuffer.DS.mouseTrk.focusEventMode = false
		case 1005, 1006, 1015:
			emu.framebuffer.DS.mouseTrk.enc = MouseEncNone
		case 1007:
			emu.framebuffer.DS.MouseAlternateScroll = false
		case 1036, 1039:
			emu.framebuffer.DS.altSendsEscape = false
		case 1047:
			emu.logU.Println("TODO switchScreenBufferMode(false) zutty vterm.icc line 1495")
		case 1048:
			hdl_esc_decrc(emu)
		case 1049:
			emu.logU.Println("TODO switchScreenBufferMode(false) zutty vterm.icc line 1497")
			hdl_esc_decrc(emu)
		case 2004:
			emu.framebuffer.DS.BracketedPaste = false
		default:
			emu.logU.Printf("reset priv mode %d\n", param)
		}
	}
}

// CSI Ps ; Ps r
// Set Scrolling Region [top;bottom] (default = full size of  window) (DECSTBM), VT100.
func hdl_csi_decstbm(emu *emulator, top, bottom int) {
	// TODO consider the originMode, zutty vterm.icc 1310~1343
	fb := emu.framebuffer
	if bottom <= top || top > fb.DS.GetHeight() || (top == 0 && bottom == 1) {
		return // invalid, xterm ignores
	}

	fb.DS.SetScrollingRegion(top-1, bottom-1)
	fb.DS.MoveCol(0, false, false)
	fb.DS.MoveRow(0, false)
}

// CSI ! p   Soft terminal reset (DECSTR), VT220 and up.
func hdl_csi_decstr(emu *emulator) {
	// TODO consider the implementation, zutty csi_DECSTR vterm.icc 1748~1749
	emu.framebuffer.SoftReset()
}

// DCS $ q Pt ST
//           Request Status String (DECRQSS), VT420 and up.
//           The string following the "q" is one of the following:
//             m       ⇒  SGR
//             " p     ⇒  DECSCL
//             SP q    ⇒  DECSCUSR
//             " q     ⇒  DECSCA
//             r       ⇒  DECSTBM
//             s       ⇒  DECSLRM
//             t       ⇒  DECSLPP
//             $ |     ⇒  DECSCPP
//             $ }     ⇒  DECSASD
//             $ ~     ⇒  DECSSDT
//             * |     ⇒  DECSNLS
//           xterm responds with DCS 1 $ r Pt ST for valid requests,
//           replacing the Pt with the corresponding CSI string, or DCS 0 $
//           r Pt ST for invalid requests.
//
// DECRQSS—Request Selection or Setting
func hdl_dcs_decrqss(emu *emulator, arg string) {
	// only response to DECSCL
	if arg == "$q\"p" {
		emu.framebuffer.DS.compatLevel = CompatLevelVT400
		response := fmt.Sprintf("\x1BP1$r%s\x1B\\", DEVICE_ID)
		emu.dispatcher.terminalToHost.WriteString(response)
	} else {
		response := fmt.Sprintf("\x1BP0$r%s\x1B\\", arg[2:])
		emu.dispatcher.terminalToHost.WriteString(response)
	}
}
