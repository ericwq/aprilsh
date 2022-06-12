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
	// "fmt"
	// "sync"
)

// type emuFunc func(*Framebuffer, *Dispatcher)
//
// type emuFunction struct {
// 	function        emuFunc
// 	clearsWrapState bool
// }

// this is a center for register emulator function.
// var emuFunctions = struct {
// 	sync.Mutex
// 	functionsESC     map[string]emuFunction
// 	functionsCSI     map[string]emuFunction
// 	functionsControl map[string]emuFunction
// }{
// 	functionsESC:     make(map[string]emuFunction, 20),
// 	functionsCSI:     make(map[string]emuFunction, 20),
// 	functionsControl: make(map[string]emuFunction, 20),
// }

// func findFunctionBy(funType int, key string) emuFunction {
// 	emuFunctions.Lock()
// 	defer emuFunctions.Unlock()
//
// 	switch funType {
// 	case DISPATCH_CONTROL:
// 		if f, ok := emuFunctions.functionsControl[key]; ok {
// 			return f
// 		}
// 	case DISPATCH_ESCAPE:
// 		if f, ok := emuFunctions.functionsESC[key]; ok {
// 			return f
// 		}
// 	case DISPATCH_CSI:
// 		if f, ok := emuFunctions.functionsCSI[key]; ok {
// 			return f
// 		}
// 	}
//
// 	return emuFunction{}
// }
//
// func registerFunction(funType int, dispatchChar string, f emuFunc, wrap bool) {
// 	emuFunctions.Lock()
// 	defer emuFunctions.Unlock()
//
// 	switch funType {
// 	case DISPATCH_CONTROL:
// 		emuFunctions.functionsControl[dispatchChar] = emuFunction{function: f, clearsWrapState: wrap}
// 	case DISPATCH_ESCAPE:
// 		emuFunctions.functionsESC[dispatchChar] = emuFunction{function: f, clearsWrapState: wrap}
// 	case DISPATCH_CSI:
// 		emuFunctions.functionsCSI[dispatchChar] = emuFunction{function: f, clearsWrapState: wrap}
// 	default: // just ignore
// 	}
// }

func init() {
	// registerFunction(DISPATCH_CSI, "@", csi_ich, true) // ich
	// registerFunction(DISPATCH_CSI, "A", csi_cursor_move, true) // cuu
	// registerFunction(DISPATCH_CSI, "B", csi_cursor_move, true) // cud
	// registerFunction(DISPATCH_CSI, "C", csi_cursor_move, true) // cuf
	// registerFunction(DISPATCH_CSI, "D", csi_cursor_move, true) // cub
	// registerFunction(DISPATCH_CSI, "G", csi_hpa, true) // cha
	// registerFunction(DISPATCH_CSI, "H", csi_cursor_move, true) // cup
	// registerFunction(DISPATCH_CSI, "I", csi_cxt, true)         // cht
	// registerFunction(DISPATCH_CSI, "J", csi_ed, true)  // ed
	// registerFunction(DISPATCH_CSI, "K", csi_el, true)  // el
	// registerFunction(DISPATCH_CSI, "L", csi_il, true)  // il
	// registerFunction(DISPATCH_CSI, "M", csi_dl, true)  // dl
	// registerFunction(DISPATCH_CSI, "P", csi_dch, true) // dch
	// registerFunction(DISPATCH_CSI, "S", csi_su, true)  // SU
	// registerFunction(DISPATCH_CSI, "T", csi_sd, true)  // SD
	// registerFunction(DISPATCH_CSI, "X", csi_ech, true) // ech
	// registerFunction(DISPATCH_CSI, "Z", csi_cxt, true)         // cbt
	// registerFunction(DISPATCH_CSI, "`", csi_hpa, true) // hpa
	// registerFunction(DISPATCH_CSI, "c", csi_da, true)  // da request
	// registerFunction(DISPATCH_CSI, "d", csi_vpa, true) // vpa
	// registerFunction(DISPATCH_CSI, "f", csi_cursor_move, true) // hvp
	// registerFunction(DISPATCH_CSI, "g", csi_tbc, true)         // tbc
	// registerFunction(DISPATCH_CSI, "h", csi_sm, false)      // sm
	// registerFunction(DISPATCH_CSI, "l", csi_rm, false)      // rm
	// registerFunction(DISPATCH_CSI, "m", csi_sgr, false)     // sgr
	// registerFunction(DISPATCH_CSI, "n", csi_dsr, false)     // dsr
	// registerFunction(DISPATCH_CSI, "r", csi_decstbm, false) // decstbm
	// registerFunction(DISPATCH_CSI, "!p", csi_decstr, true)  // decstr
	// registerFunction(DISPATCH_CSI, ">c", csi_sda, true)     // sda request
	// registerFunction(DISPATCH_CSI, "?h", csi_decsm, false)  // decset
	// registerFunction(DISPATCH_CSI, "?l", csi_decrm, false)  // decrst

	// registerFunction(DISPATCH_ESCAPE, "#8", esc_decaln, true) // decaln
	// registerFunction(DISPATCH_ESCAPE, "7", esc_decsc, true)   // decsc
	// registerFunction(DISPATCH_ESCAPE, "8", esc_decrc, true)   // decrc
	// registerFunction(DISPATCH_ESCAPE, "c", esc_rts, true)     // rts

	// registerFunction(DISPATCH_CONTROL, "\x07", ctrl_bel, true) // bel ctrl-G
	// registerFunction(DISPATCH_CONTROL, "\x08", ctrl_bs, true)  // bs ctrl-H
	// registerFunction(DISPATCH_CONTROL, "\x09", ctrl_ht, true)  // tab ctrl-I
	// registerFunction(DISPATCH_CONTROL, "\x0A", ctrl_lf, true)  // lf ctrl-J
	// registerFunction(DISPATCH_CONTROL, "\x0B", ctrl_lf, true)  // vt ctrl-K
	// registerFunction(DISPATCH_CONTROL, "\x0C", ctrl_lf, true)  // ff ctrl-L
	// registerFunction(DISPATCH_CONTROL, "\x0D", ctrl_cr, true)  // cr ctrl-M
	// registerFunction(DISPATCH_CONTROL, "\x84", ctrl_lf, true)  // ind
	// registerFunction(DISPATCH_CONTROL, "\x85", ctrl_nel, true) // nel
	// registerFunction(DISPATCH_CONTROL, "\x88", ctrl_hts, true) // hts
	// registerFunction(DISPATCH_CONTROL, "\x8D", ctrl_ri, true)  // ri
}

// CSI Ps S  Scroll up Ps lines (default = 1) (SU), VT420, ECMA-48.
// TODO it seams mosh revert the SD and SU
// follow the specification
// func csi_su(fb *Framebuffer, d *Dispatcher) {
// 	fb.Scroll(d.getParam(0, 1))
// }

// CSI Ps T  Scroll down Ps lines (default = 1) (SD), VT420.
// TODO it seams mosh revert the SD and SU
// follow the specification
// func csi_sd(fb *Framebuffer, d *Dispatcher) {
// 	fb.Scroll(-d.getParam(0, 1))
// }

// CSI ! p   Soft terminal reset (DECSTR), VT220 and up.
// func csi_decstr(fb *Framebuffer, _ *Dispatcher) {
// 	fb.SoftReset()
// }

// ESC c     Full Reset (RIS), VT100.
// reset the screen
// func esc_rts(fb *Framebuffer, _ *Dispatcher) {
// 	fb.Reset()
// }

// CSI Ps X  Erase Ps Character(s) (default = 1) (ECH).
// func csi_ech(fb *Framebuffer, d *Dispatcher) {
// 	num := d.getParam(0, 1)
// 	limit := fb.DS.GetCursorCol() + num - 1
//
// 	if limit >= fb.DS.GetWidth() {
// 		limit = fb.DS.GetWidth() - 1
// 	}
//
// 	clearline(fb, -1, fb.DS.GetCursorCol(), limit)
// }

// CSI Ps G  Cursor Character Absolute  [column] (default = [row,1]) (CHA).
// CSI Ps `  Character Position Absolute  [column] (default = [row,1]) (HPA).
// func csi_hpa(fb *Framebuffer, d *Dispatcher) {
// 	col := d.getParam(0, 1)
// 	fb.DS.MoveCol(col-1, false, false)
// }

// CSI Ps d  Line Position Absolute  [row] (default = [1,column]) (VPA).
// func csi_vpa(fb *Framebuffer, d *Dispatcher) {
// 	row := d.getParam(0, 1)
// 	fb.DS.MoveRow(row-1, false)
// }

// CSI Ps P  Delete Ps Character(s) (default = 1) (DCH).
// func csi_dch(fb *Framebuffer, d *Dispatcher) {
// 	cells := d.getParam(0, 1)
//
// 	for i := 0; i < cells; i++ {
// 		fb.DeleteCell(fb.DS.GetCursorRow(), fb.DS.GetCursorCol())
// 	}
// }

// CSI Ps @  Insert Ps (Blank) Character(s) (default = 1) (ICH).
// func csi_ich(fb *Framebuffer, d *Dispatcher) {
// 	cells := d.getParam(0, 1)
//
// 	for i := 0; i < cells; i++ {
// 		fb.InsertCell(fb.DS.GetCursorRow(), fb.DS.GetCursorCol())
// 	}
// }

// CSI Ps M  Delete Ps Line(s) (default = 1) (DL).
// delete N lines in cursor position
// func csi_dl(fb *Framebuffer, d *Dispatcher) {
// 	lines := d.getParam(0, 1)
//
// 	fb.DeleteLine(fb.DS.GetCursorRow(), lines)
//
// 	// vt220 manual and Ecma-48 say to move to first column */
// 	fb.DS.MoveCol(0, false, false)
// }

// CSI Ps L  Insert Ps Line(s) (default = 1) (IL).
// insert N lines in cursor position
// func csi_il(fb *Framebuffer, d *Dispatcher) {
// 	lines := d.getParam(0, 1)
//
// 	fb.InsertLine(fb.DS.GetCursorRow(), lines)
//
// 	// vt220 manual and Ecma-48 say to move to first column */
// 	fb.DS.MoveCol(0, false, false)
// }

// CSI Ps n  Device Status Report (DSR).
//   Ps = 5  ⇒  Status Report. Result ("OK") is CSI 0 n
//   Ps = 6  ⇒  Report Cursor Position (CPR) [row;column]. Result is CSI r ; c R
// func csi_dsr(fb *Framebuffer, d *Dispatcher) {
// 	param := d.getParam(0, 0)
//
// 	switch param {
// 	case 5:
// 		// device status report requested
// 		d.terminalToHost.WriteString("\033[0n")
// 	case 6:
// 		// report of active position requested
// 		fmt.Fprintf(&d.terminalToHost, "\033[%d;%dR", fb.DS.GetCursorRow()+1, fb.DS.GetCursorCol()+1)
// 	}
// }
//
// ESC 7     Save Cursor (DECSC), VT100.
// func esc_decsc(fb *Framebuffer, _ *Dispatcher) {
// 	fb.DS.SaveCursor()
// }

// ESC 8     Restore Cursor (DECRC), VT100.
// func esc_decrc(fb *Framebuffer, _ *Dispatcher) {
// 	fb.DS.RestoreCursor()
// }

// CSI Pm m  Character Attributes (SGR).
// select graphics rendition -- e.g., bold, blinking, etc.
// support 8, 16, 256 color, RGB color.
// TODO this version doesn't suppor : as seperator
// func csi_sgr(fb *Framebuffer, d *Dispatcher) {
// 	for i := 0; i < d.getParamCount(); i++ {
// 		rendition := d.getParam(i, 0)
// 		// We need to special-case the handling of [34]8 ; 5 ; Ps,
// 		// because Ps of 0 in that case does not mean reset to default, even
// 		// though it means that otherwise (as usually renditions are applied
// 		// in order).
//
// 		if (rendition == 38 || rendition == 48) && d.getParamCount()-i >= 3 &&
// 			d.getParam(i+1, -1) == 5 {
//
// 			if rendition == 38 {
// 				fb.DS.SetForegroundColor(d.getParam(i+2, 0))
// 			} else {
// 				fb.DS.SetBackgroundColor(d.getParam(i+2, 0))
// 			}
//
// 			i += 2
// 			continue
// 		}
// 		// True color support: ESC[ ... [34]8;2;<r>;<g>;<b> ... m
// 		if (rendition == 38 || rendition == 48) && d.getParamCount()-1 >= 5 &&
// 			d.getParam(i+1, -1) == 2 {
//
// 			red := d.getParam(i+2, 0)
// 			green := d.getParam(i+3, 0)
// 			blue := d.getParam(i+4, 0)
//
// 			if rendition == 38 {
// 				fb.DS.renditions.SetFgColor(uint32(red), uint32(green), uint32(blue))
// 			} else {
// 				fb.DS.renditions.SetBgColor(uint32(red), uint32(green), uint32(blue))
// 			}
//
// 			i += 4
// 			continue
// 		}
//
// 		fb.DS.AddRenditions(uint32(rendition))
// 	}
// }

// Bell (BEL  is Ctrl-G).
// ring the bell
// func ctrl_bel(fb *Framebuffer, _ *Dispatcher) {
// 	fb.RingBell()
// }

// CSI Ps ; Ps r
// Set Scrolling Region [top;bottom] (default = full size of  window) (DECSTBM), VT100.
// set top and bottom margins
// func csi_decstbm(fb *Framebuffer, d *Dispatcher) {
// 	top := d.getParam(0, 1)
// 	bottom := d.getParam(1, fb.DS.GetHeight())
//
// 	if bottom <= top || top > fb.DS.GetHeight() || (top == 0 && bottom == 1) {
// 		return // invalid, xterm ignores
// 	}
//
// 	fb.DS.SetScrollingRegion(top-1, bottom-1)
// 	fb.DS.MoveCol(0, false, false)
// 	fb.DS.MoveRow(0, false)
// }

// func getANSImode(param int, fb *Framebuffer) *bool {
// 	switch param {
// 	case 4:
// 		// insert/replace mode
// 		return &fb.DS.InsertMode
// 	}
// 	return nil
// }

// CSI Pm h  Set Mode (SM).
// *  Ps = 2  ⇒  Keyboard Action Mode (KAM).
// *  Ps = 4  ⇒  Insert Mode (IRM).
// *  Ps = 1 2  ⇒  Send/receive (SRM).
// *  Ps = 2 0  ⇒  Automatic Newline (LNM).
// func csi_sm(fb *Framebuffer, d *Dispatcher) {
// 	for i := 0; i < d.getParamCount(); i++ {
// 		mode := getANSImode(d.getParam(i, 0), fb)
// 		if mode != nil && *mode {
// 			*mode = true
// 		}
// 	}
// }

// CSI Pm l  Reset Mode (RM).
// *  Ps = 2  ⇒  Keyboard Action Mode (KAM).
// *  Ps = 4  ⇒  Replace Mode (IRM).
// *  Ps = 1 2  ⇒  Send/receive (SRM).
// *  Ps = 2 0  ⇒  Normal Linefeed (LNM).
// func csi_rm(fb *Framebuffer, d *Dispatcher) {
// 	for i := 0; i < d.getParamCount(); i++ {
// 		mode := getANSImode(d.getParam(i, 0), fb)
// 		if mode != nil && *mode {
// 			*mode = false
// 		}
// 	}
// }

func getDECmode(param int, fb *Framebuffer) *bool {
	switch param {
	case 1:
		// cursor key mode
		return &fb.DS.ApplicationModeCursorKeys
	case 3:
		// 80/132. Ignore but clear screen.
		fb.DS.MoveCol(0, false, false)
		fb.DS.MoveRow(0, false)
		for y := 0; y < fb.DS.GetHeight(); y++ {
			fb.ResetRow(fb.GetRow(y))
		}
		return nil
	case 5:
		// reverse video
		return &fb.DS.ReverseVideo
	case 6:
		// origin
		fb.DS.MoveCol(0, false, false)
		fb.DS.MoveRow(0, false)
		return &fb.DS.OriginMode
	case 7:
		// auto wrap
		return &fb.DS.AutoWrapMode
	case 25:
		return &fb.DS.CursorVisible
	case 1004:
		// xterm mouse focus event
		return &fb.DS.MouseFocusEvent
	case 1007:
		// xterm mouse alternate scroll
		return &fb.DS.MouseAlternateScroll
	case 2004:
		// bracketed paste
		return &fb.DS.BracketedPaste
	}
	return nil
}

func setIfAvailable(mode *bool, value bool) {
	if mode != nil && *mode {
		*mode = value
	}
}

// CSI ? Pm h
// DEC Private Mode Set (DECSET).
// Ps = 9        ⇒  Send Mouse X & Y on button press.
// Ps = 1 0 0 0  ⇒  Send Mouse X & Y on button press and release.
// Ps = 1 0 0 1  ⇒  Use Hilite Mouse Tracking, xterm.
// Ps = 1 0 0 2  ⇒  Use Cell Motion Mouse Tracking, xterm.
// Ps = 1 0 0 3  ⇒  Use All Motion Mouse Tracking, xterm.
// Ps = 1 0 0 5  ⇒  Enable UTF-8 Mouse Mode, xterm.
// Ps = 1 0 0 6  ⇒  Enable SGR Mouse Mode, xterm.
// Ps = 1 0 1 5  ⇒  Enable urxvt Mouse Mode.
// set private mode
// func csi_decsm(fb *Framebuffer, d *Dispatcher) {
// 	for i := 0; i < d.getParamCount(); i++ {
// 		param := d.getParam(i, 0)
// 		if param == 9 || (1000 <= param && param <= 1003) {
// 			fb.DS.MouseReportingMode = param
// 		} else if param == 1005 || param == 1006 || param == 1015 {
// 			fb.DS.MouseEncodingMode = param
// 		} else {
// 			setIfAvailable(getDECmode(param, fb), true)
// 		}
// 	}
// }

// CSI ? Pm l
// DEC Private Mode Reset (DECRST).
// Ps = 9        ⇒  Don't send Mouse X & Y on button press, xterm.
// Ps = 1 0 0 0  ⇒  Don't send Mouse X & Y on button press and release.
// Ps = 1 0 0 1  ⇒  Don't use Hilite Mouse Tracking, xterm.
// Ps = 1 0 0 2  ⇒  Don't use Cell Motion Mouse Tracking, xterm.
// Ps = 1 0 0 3  ⇒  Don't use All Motion Mouse Tracking, xterm.
// Ps = 1 0 0 5  ⇒  Disable UTF-8 Mouse Mode, xterm.
// Ps = 1 0 0 6  ⇒  Disable SGR Mouse Mode, xterm.
// Ps = 1 0 1 5  ⇒  Disable urxvt Mouse Mode.
// clear private mode
// func csi_decrm(fb *Framebuffer, d *Dispatcher) {
// 	for i := 0; i < d.getParamCount(); i++ {
// 		param := d.getParam(i, 0)
// 		if param == 9 || (1000 <= param && param <= 1003) {
// 			fb.DS.MouseReportingMode = MOUSE_REPORTING_NONE
// 		} else if param == 1005 || param == 1006 || param == 1015 {
// 			fb.DS.MouseEncodingMode = MOUSE_ENCODING_DEFAULT
// 		} else {
// 			setIfAvailable(getDECmode(param, fb), false)
// 		}
// 	}
// }
//
// CSI Ps g  Tab Clear (TBC).
// *  Ps = 0  ⇒  Clear Current Column (default).
// *  Ps = 3  ⇒  Clear All.
// func csi_tbc(fb *Framebuffer, d *Dispatcher) {
// 	param := d.getParam(0, 0)
// 	switch param {
// 	case 0:
// 		// clear this tab stop
// 		fb.DS.ClearTab(fb.DS.GetCursorCol())
// 	case 3:
// 		// clear all tab stops
// 		fb.DS.ClearDefaultTabs()
// 		for x := 0; x < fb.DS.GetWidth(); x++ {
// 			fb.DS.ClearTab(x)
// 		}
// 	}
// }

// Tab Set (HTS  is 0x88).
// set current cursor column tab true
// func ctrl_hts(fb *Framebuffer, _ *Dispatcher) {
// 	fb.DS.SetTab()
// }

// CSI Ps I  Cursor Forward Tabulation Ps tab stops (default = 1) (CHT).
// CSI Ps Z  Cursor Backward Tabulation Ps tab stops (default = 1) (CBT).
// move cursor forward/backwoard count tab position
// func csi_cxt(fb *Framebuffer, d *Dispatcher) {
// 	param := d.getParam(0, 1)
// 	if d.dispatcherChar.String()[0] == 'Z' {
// 		param = -param
// 	}
// 	if param == 0 {
// 		return
// 	}
// 	ht_n(fb, param)
// }

// Horizontal Tab (HTS  is Ctrl-I).
// move cursor to the count tab position
// func ht_n(fb *Framebuffer, count int) {
// 	col := fb.DS.GetNextTab(count)
// 	if col == -1 { // no tabs, go to end of line
// 		col = fb.DS.GetWidth() - 1
// 	}
// 	// A horizontal tab is the only operation that preserves but
// 	// does not set the wrap state. It also starts a new grapheme.
// 	wrapStateSave := fb.DS.NextPrintWillWrap
// 	fb.DS.MoveCol(col, false, false)
// 	fb.DS.NextPrintWillWrap = wrapStateSave
// }

// Horizontal Tab (HTS  is Ctrl-I).
// move cursor to the next tab position
// func ctrl_ht(fb *Framebuffer, _ *Dispatcher) {
// 	ht_n(fb, 1)
// }

// Next Line (NEL  is 0x85).
// move cursor to the next row, scroll down if necessary. move cursor to row head
// func ctrl_nel(fb *Framebuffer, _ *Dispatcher) {
// 	fb.DS.MoveCol(0, false, false)
// 	fb.MoveRowsAutoscroll(1)
// }

// Reverse Index (RI  is 0x8d).
// move cursor to the previous row, scroll up if necessary
// reverse index -- like a backwards line feed
// func ctrl_ri(fb *Framebuffer, _ *Dispatcher) {
// 	fb.MoveRowsAutoscroll(-1)
// }

// Backspace (BS  is Ctrl-H).
// bask space
// func ctrl_bs(fb *Framebuffer, _ *Dispatcher) {
// 	fb.DS.MoveCol(-1, true, false)
// }
//
// Carriage Return (CR  is Ctrl-M).
// move cursor to the head of the same row
// func ctrl_cr(fb *Framebuffer, _ *Dispatcher) {
// 	fb.DS.MoveCol(0, false, false)
// }

// IND, FF, LF, VT
// move cursor to the next row, scroll down if necessary.
// func ctrl_lf(fb *Framebuffer, _ *Dispatcher) {
// 	fb.MoveRowsAutoscroll(1)
// }

// ESC # 8   DEC Screen Alignment Test (DECALN), VT100.
// fill the screen with 'E'
// func esc_decaln(fb *Framebuffer, _ *Dispatcher) {
// 	for y := 0; y < fb.DS.GetHeight(); y++ {
// 		for x := 0; x < fb.DS.GetWidth(); x++ {
// 			fb.ResetCell(fb.GetCell(y, x))
// 			fb.GetCell(y, x).Append('E')
// 		}
// 	}
// }

// CSI > Ps c Send Device Attributes (Secondary DA).
// Ps = 0  or omitted ⇒  request the terminal's identification code.
// CSI > Pp ; Pv ; Pc c
// Pp = 1  ⇒  "VT220".
// Pv is the firmware version.
// Pc indicates the ROM cartridge registration number and is always zero.
// func csi_sda(_ *Framebuffer, d *Dispatcher) {
// 	d.terminalToHost.WriteString("\033[>1;10;0c") // plain vt220
// }

// CSI Ps c  Send Device Attributes (Primary DA).
// CSI ? 6 2 ; Ps c  ("VT220")
// DA response
// func csi_da(_ *Framebuffer, d *Dispatcher) {
// 	d.terminalToHost.WriteString("\033[?62c") // plain vt220
// }

// CSI Ps A  Cursor Up Ps Times (default = 1) (CUU).
// CSI Ps B  Cursor Down Ps Times (default = 1) (CUD).
// CSI Ps C  Cursor Forward Ps Times (default = 1) (CUF).
// CSI Ps D  Cursor Backward Ps Times (default = 1) (CUB).
// CSI Ps ; Ps H Cursor Position [row;column] (default = [1,1]) (CUP).
// CSI Ps ; Ps f Horizontal and Vertical Position [row;column] (default = [1,1]) (HVP).
// func csi_cursor_move(fb *Framebuffer, d *Dispatcher) {
// 	num := d.getParam(0, 1)
//
// 	switch d.getDispatcherChars()[0] {
// 	case 'A':
// 		fb.DS.MoveRow(-num, true)
// 	case 'B':
// 		fb.DS.MoveRow(num, true)
// 	case 'C':
// 		fb.DS.MoveCol(num, true, false)
// 	case 'D':
// 		fb.DS.MoveCol(-num, true, false)
// 	case 'H', 'f':
// 		x := d.getParam(0, 1)
// 		y := d.getParam(1, 1)
// 		fb.DS.MoveRow(x-1, false)
// 		fb.DS.MoveCol(y-1, false, false)
// 	}
// }

// CSI Ps J Erase in Display (ED), VT100.
// * Ps = 0  ⇒  Erase Below (default).
// * Ps = 1  ⇒  Erase Above.
// * Ps = 2  ⇒  Erase All.
// * Ps = 3  ⇒  Erase Saved Lines, xterm.
// func csi_ed(fb *Framebuffer, d *Dispatcher) {
// 	switch d.getParam(0, 0) {
// 	case 0:
// 		// active position down to end of screen, inclusive
// 		clearline(fb, -1, fb.DS.GetCursorCol(), fb.DS.GetWidth()-1)
// 		for y := fb.DS.GetCursorRow() + 1; y < fb.DS.GetHeight(); y++ {
// 			fb.ResetRow(fb.GetRow(y))
// 		}
// 	case 1:
// 		// start of screen to active position, inclusive
// 		for y := 0; y < fb.DS.GetCursorRow(); y++ {
// 			fb.ResetRow(fb.GetRow(y))
// 		}
// 		clearline(fb, -1, 0, fb.DS.GetCursorCol())
// 	case 2:
// 		//  entire screen
// 		for y := 0; y < fb.DS.GetHeight(); y++ {
// 			fb.ResetRow(fb.GetRow(y))
// 		}
// 	}
// }

// erase cell from the start to end at specified row
// func clearline(fb *Framebuffer, row int, start int, end int) {
// 	for col := start; col <= end; col++ {
// 		fb.ResetCell(fb.GetCell(row, col))
// 	}
// }

// CSI Ps K Erase in Line (EL), VT100.
// * Ps = 0  ⇒  Erase to Right (default).
// * Ps = 1  ⇒  Erase to Left.
// * Ps = 2  ⇒  Erase All.
// func csi_el(fb *Framebuffer, d *Dispatcher) {
// 	switch d.getParam(0, 0) {
// 	case 0:
// 		clearline(fb, -1, fb.DS.GetCursorCol(), fb.DS.GetWidth()-1)
// 	case 1:
// 		clearline(fb, -1, 0, fb.DS.GetCursorCol())
// 	case 2:
// 		fb.ResetRow(fb.GetRow(-1))
// 	}
// }
