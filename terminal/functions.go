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
	registerFunction(DISPATCH_CSI, "K", csi_el, true)        // el
	registerFunction(DISPATCH_CSI, "J", csi_ed, true)        // ed
	registerFunction(DISPATCH_CSI, "A", csiCursorMove, true) // cuu
	registerFunction(DISPATCH_CSI, "B", csiCursorMove, true) // cud
	registerFunction(DISPATCH_CSI, "C", csiCursorMove, true) // cuf
	registerFunction(DISPATCH_CSI, "D", csiCursorMove, true) // cub
	registerFunction(DISPATCH_CSI, "H", csiCursorMove, true) // cup
	registerFunction(DISPATCH_CSI, "f", csiCursorMove, true) // hvp
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
