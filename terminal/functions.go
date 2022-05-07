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

import "sync"

type emuFunc func(*Framebuffer, *Dispatcher)

type emuFunction struct {
	function        emuFunc
	clearsWrapState bool
}

// this is a center for register emulator function.
var emuFunctions = struct {
	sync.Mutex
	functionsESC     map[string]emuFunction
	functionsCSI     map[string]emuFunction
	functionsControl map[string]emuFunction
}{
	functionsESC:     make(map[string]emuFunction, 20),
	functionsCSI:     make(map[string]emuFunction, 20),
	functionsControl: make(map[string]emuFunction, 20),
}

func findFunctionBy(funType int, key string) emuFunction {
	emuFunctions.Lock()
	defer emuFunctions.Unlock()

	switch funType {
	case DISPATCH_CONTROL:
		if f, ok := emuFunctions.functionsControl[key]; ok {
			return f
		}
	case DISPATCH_ESCAPE:
		if f, ok := emuFunctions.functionsESC[key]; ok {
			return f
		}
	case DISPATCH_CSI:
		if f, ok := emuFunctions.functionsCSI[key]; ok {
			return f
		}
	}

	return emuFunction{}
}

func registerFunction(funType int, dispatchChar string, f emuFunc, wrap bool) {
	emuFunctions.Lock()
	defer emuFunctions.Unlock()

	switch funType {
	case DISPATCH_CONTROL:
		emuFunctions.functionsControl[dispatchChar] = emuFunction{function: f, clearsWrapState: wrap}
	case DISPATCH_ESCAPE:
		emuFunctions.functionsESC[dispatchChar] = emuFunction{function: f, clearsWrapState: wrap}
	case DISPATCH_CSI:
		emuFunctions.functionsCSI[dispatchChar] = emuFunction{function: f, clearsWrapState: wrap}
	default: // just ignore
	}
}

func init() {
	registerFunction(DISPATCH_CSI, "K", csi_el, true) // el

	registerFunction(DISPATCH_CSI, "J", csi_ed, true) // ed

	registerFunction(DISPATCH_CSI, "A", csiCursorMove, true) // cuu
	registerFunction(DISPATCH_CSI, "B", csiCursorMove, true) // cud
	registerFunction(DISPATCH_CSI, "C", csiCursorMove, true) // cuf
	registerFunction(DISPATCH_CSI, "D", csiCursorMove, true) // cub
	registerFunction(DISPATCH_CSI, "H", csiCursorMove, true) // cup
	registerFunction(DISPATCH_CSI, "f", csiCursorMove, true) // hvp

	registerFunction(DISPATCH_CSI, "c", csi_da, true)  // da request
	registerFunction(DISPATCH_CSI, ">c", csi_da, true) // sda request

	registerFunction(DISPATCH_ESCAPE, "#8", esc_decaln, true) // decaln

	registerFunction(DISPATCH_CONTROL, "\x84", ctr_lf, true)  // ind
	registerFunction(DISPATCH_CONTROL, "\x0A", ctr_lf, true)  // lf ctrl-J
	registerFunction(DISPATCH_CONTROL, "\x0B", ctr_lf, true)  // vt ctrl-K
	registerFunction(DISPATCH_CONTROL, "\x0C", ctr_lf, true)  // ff ctrl-L
	registerFunction(DISPATCH_CONTROL, "\x0D", ctr_cr, true)  // cr ctrl-M
	registerFunction(DISPATCH_CONTROL, "\x08", ctr_bs, true)  // bs ctrl-H
	registerFunction(DISPATCH_CONTROL, "\x8D", ctr_ri, true)  // ri
	registerFunction(DISPATCH_CONTROL, "\x85", ctr_nel, true) // nel
	registerFunction(DISPATCH_CONTROL, "\x09", ctr_ht, true)  // tab

	registerFunction(DISPATCH_CSI, "I", csi_cxt, true) // cht
	registerFunction(DISPATCH_CSI, "Z", csi_cxt, true) // cbt

	registerFunction(DISPATCH_CONTROL, "\x88", ctr_hts, true) // hts
	registerFunction(DISPATCH_CSI, "g", csi_tbc, true)        // tbc
}

// CSI Ps g  Tab Clear (TBC).
// *  Ps = 0  ⇒  Clear Current Column (default).
// *  Ps = 3  ⇒  Clear All.
func csi_tbc(fb *Framebuffer, d *Dispatcher) {
	param := d.getParam(0, 0)
	switch param {
	case 0:
		// clear this tab stop
		fb.DS.ClearTab(fb.DS.GetCursorCol())
	case 3:
		// clear all tab stops
		fb.DS.ClearDefaultTabs()
		for x := 0; x < fb.DS.GetWidth(); x++ {
			fb.DS.ClearTab(x)
		}
	}
}

// Tab Set (HTS  is 0x88).
// set current cursor column tab true
func ctr_hts(fb *Framebuffer, _ *Dispatcher) {
	fb.DS.SetTab()
}

// CSI Ps I  Cursor Forward Tabulation Ps tab stops (default = 1) (CHT).
// CSI Ps Z  Cursor Backward Tabulation Ps tab stops (default = 1) (CBT).
// move cursor forward/backwoard count tab position
func csi_cxt(fb *Framebuffer, d *Dispatcher) {
	param := d.getParam(0, 1)
	if d.dispatcherChar.String()[0] == 'Z' {
		param = -param
	}
	if param == 0 {
		return
	}
	ht_n(fb, param)
}

// Horizontal Tab (HTS  is Ctrl-I).
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
// #TODO should we register 0x88 also?
func ctr_ht(fb *Framebuffer, _ *Dispatcher) {
	ht_n(fb, 1)
}

// Next Line (NEL  is 0x85).
// move cursor to the next row, scroll down if necessary. move cursor to row head
func ctr_nel(fb *Framebuffer, _ *Dispatcher) {
	fb.DS.MoveCol(0, false, false)
	fb.MoveRowsAutoscroll(1)
}

// Reverse Index (RI  is 0x8d).
// move cursor to the previous row, scroll up if necessary
// reverse index -- like a backwards line feed
func ctr_ri(fb *Framebuffer, _ *Dispatcher) {
	fb.MoveRowsAutoscroll(-1)
}

// Backspace (BS  is Ctrl-H).
// bask space
func ctr_bs(fb *Framebuffer, _ *Dispatcher) {
	fb.DS.MoveCol(-1, true, false)
}

// Carriage Return (CR  is Ctrl-M).
// move cursor to the head of the same row
func ctr_cr(fb *Framebuffer, _ *Dispatcher) {
	fb.DS.MoveCol(0, false, false)
}

// IND, FF, LF, VT
// move cursor to the next row, scroll down if necessary.
func ctr_lf(fb *Framebuffer, _ *Dispatcher) {
	fb.MoveRowsAutoscroll(1)
}

// ESC # 8   DEC Screen Alignment Test (DECALN), VT100.
// fill the screen with 'E'
func esc_decaln(fb *Framebuffer, _ *Dispatcher) {
	for y := 0; y < fb.DS.GetHeight(); y++ {
		for x := 0; x < fb.DS.GetWidth(); x++ {
			fb.ResetCell(fb.GetCell(y, x))
			fb.GetCell(y, x).Append('E')
		}
	}
}

// CSI > Ps c Send Device Attributes (Secondary DA).
// Ps = 0  or omitted ⇒  request the terminal's identification code.
// CSI  > Pp ; Pv ; Pc c
// Pp = 1  ⇒  "VT220".
// Pv is the firmware version.
// Pc indicates the ROM cartridge registration number and is always zero.
func csi_sda(_ *Framebuffer, d *Dispatcher) {
	d.terminalToHost.WriteString("\033[>1;10;0c") // plain vt220
}

// CSI Ps c  Send Device Attributes (Primary DA).
// CSI ? 6 2 ; Ps c  ("VT220")
// DA response
func csi_da(_ *Framebuffer, d *Dispatcher) {
	d.terminalToHost.WriteString("\033[?62c") // plain vt220
}

// CSI Ps A  Cursor Up Ps Times (default = 1) (CUU).
// CSI Ps B  Cursor Down Ps Times (default = 1) (CUD).
// CSI Ps C  Cursor Forward Ps Times (default = 1) (CUF).
// CSI Ps D  Cursor Backward Ps Times (default = 1) (CUB).
// CSI Ps ; Ps H Cursor Position [row;column] (default = [1,1]) (CUP).
// CSI Ps ; Ps f Horizontal and Vertical Position [row;column] (default = [1,1]) (HVP).
func csiCursorMove(fb *Framebuffer, d *Dispatcher) {
	num := d.getParam(0, 1)

	switch d.getDispatcherChars()[0] {
	case 'A':
		fb.DS.MoveRow(-num, true)
	case 'B':
		fb.DS.MoveRow(num, true)
	case 'C':
		fb.DS.MoveCol(num, true, false)
	case 'D':
		fb.DS.MoveCol(-num, true, false)
	case 'H', 'f':
		x := d.getParam(0, 1)
		y := d.getParam(1, 1)
		fb.DS.MoveRow(x-1, false)
		fb.DS.MoveCol(y-1, false, false)
	}
}

// CSI Ps J Erase in Display (ED), VT100.
// * Ps = 0  ⇒  Erase Below (default).
// * Ps = 1  ⇒  Erase Above.
// * Ps = 2  ⇒  Erase All.
// * Ps = 3  ⇒  Erase Saved Lines, xterm.
func csi_ed(fb *Framebuffer, d *Dispatcher) {
	switch d.getParam(0, 0) {
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

// erase cell from the start to end at specified row
func clearline(fb *Framebuffer, row int, start int, end int) {
	for col := start; col <= end; col++ {
		fb.ResetCell(fb.GetCell(row, col))
	}
}

// CSI Ps K Erase in Line (EL), VT100.
// * Ps = 0  ⇒  Erase to Right (default).
// * Ps = 1  ⇒  Erase to Left.
// * Ps = 2  ⇒  Erase All.
func csi_el(fb *Framebuffer, d *Dispatcher) {
	switch d.getParam(0, 0) {
	case 0:
		clearline(fb, -1, fb.DS.GetCursorCol(), fb.DS.GetWidth()-1)
	case 1:
		clearline(fb, -1, 0, fb.DS.GetCursorCol())
	case 2:
		fb.ResetRow(fb.GetRow(-1))
	}
}
