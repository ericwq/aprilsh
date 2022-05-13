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
	"unicode/utf8"
)

// type handleFunc func(emu *emulator)

// Handler is the parsing result. It can be used to perform control sequence
// on emulator.
type Handler struct {
	name   string              // the name of ActOn
	ch     rune                // the last byte
	handle func(emu *emulator) // handle function that will perform control sequnce on emulator
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

// print the graphic char to the emulator
// TODO print to emulator
// TODO GR doesn't exist in UTF-8
func hdl_graphic_char(emu *emulator, r rune) {
	if utf8.RuneLen(r) > 1 {
		fmt.Printf("Unicode UTF8 print %c size=%d\n", r, utf8.RuneLen(r))
	} else if r&0x80 == 0 {
		// GL range
		cs := 0
		if emu.charsetState.ss > 0 {
			cs = emu.charsetState.g[emu.charsetState.ss]
			emu.charsetState.ss = 0
		} else {
			cs = emu.charsetState.g[emu.charsetState.gl]
		}

		if cs == Charset_UTF8 {
			fmt.Printf("GL UTF8 print %c\n", r)
		} else if r >= 32 && (cs == Charset_IsoLatin1 || r < 127) {
			ch := charCodes[cs][r-32]
			fmt.Printf("GL %d print %c\n", cs, ch)
		}
	} else {
		// GR range
		cs := emu.charsetState.g[emu.charsetState.gr]
		if cs == Charset_UTF8 {
			fmt.Printf("GR UTF8 print %c\n", r)
		} else if r >= 160 && (cs == Charset_IsoLatin1 || r < 255) {
			ch := charCodes[cs][r-160]
			fmt.Printf("GR %d print %c\n", cs, ch)
		}
	}
}

// Bell (BEL  is Ctrl-G).
// ring the bell
func hdl_c0_bel(emu *emulator) {
	emu.framebuffer.RingBell()
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
func hdl_c0_ht(emu *emulator) {
	ht_n(emu.framebuffer, 1)
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
