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

// ESC N
//     Single Shift Select of G2 Character Set (SS2  is 0x8e), VT220.
//     This affects next character only.
func hdl_c0_ss2(emu *emulator) {
	emu.charsetState.ss = 2
}

// ESC O
//     Single Shift Select of G3 Character Set (SS3  is 0x8f), VT220.
//     This affects next character only.
func hdl_c0_ss3(emu *emulator) {
	emu.charsetState.ss = 3
}

// ESC ~     Invoke the G1 Character Set as GR (LS1R), VT100.
func hdl_c0_ls1r(emu *emulator) {
	emu.charsetState.gr = 1
}

// ESC n     Invoke the G2 Character Set as GL (LS2).
func hdl_c0_ls2(emu *emulator) {
	emu.charsetState.gl = 2
}

// ESC }     Invoke the G2 Character Set as GR (LS2R).
func hdl_c0_ls2r(emu *emulator) {
	emu.charsetState.gr = 2
}

// ESC o     Invoke the G3 Character Set as GL (LS3).
func hdl_c0_ls3(emu *emulator) {
	emu.charsetState.gl = 3
}

// ESC |     Invoke the G3 Character Set as GR (LS3R).
func hdl_c0_ls3r(emu *emulator) {
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
	emu.charsetState.g[emu.charsetState.gr] = &isoLatin1SupplementalVT300 // Charset_IsoLatin1
	emu.charsetState.vt100 = true
}

// Select G0 ~ G3 character set based on parameter
func hdl_esc_dcs(emu *emulator, index int, charset *map[byte]rune) {
	emu.charsetState.g[index] = charset
	if charset != nil {
		emu.charsetState.vt100 = true
	}
}

// print the graphic char to the emulator
// TODO print to emulator
// TODO GR doesn't exist in UTF-8
// https://henvic.dev/posts/go-utf8/
// https://pkg.go.dev/golang.org/x/text/encoding/charmap
// https://github.com/rivo/uniseg
func hdl_graphemes(emu *emulator, chs ...rune) {
	// fmt.Printf("hdl_graphemes got %q", chs)

	if len(chs) == 1 {
		r := chs[0]
		if emu.charsetState.vt100 {
			r = emu.lookupCharset(r)
			fmt.Printf("   VT100: %q, %U, %x\n", r, r, r)
		} else {
			fmt.Printf("   UTF-8: %q, %U, %x\n", r, r, r)
		}
	} else if len(chs) > 1 {
		fmt.Printf("   UTF*8: %q, %U, %x\n", chs, chs, chs)
	} else {
		fmt.Printf("   UTF8 : invalid parameters\n")
	}
	/*
		if utf8.RuneLen(r) > 1 {
			fmt.Printf("Unicode UTF8 print %c size=%d\n", r, utf8.RuneLen(r))
		} else if r&0x80 == 0 {
			// GL range
			var cs *map[byte]rune
			if emu.charsetState.ss > 0 {
				cs = emu.charsetState.g[emu.charsetState.ss]
				emu.charsetState.ss = 0
			} else {
				cs = emu.charsetState.g[emu.charsetState.gl]
			}

			if cs == nil { // Charset_UTF8 {
				fmt.Printf("GL UTF8 print %c\n", r)
			} else if r >= 32 && (cs == &isoLatin1SupplementalVT300 || r < 127) { // Charset_IsoLatin1
				ch := lookupTable(cs, byte(r))
				fmt.Printf("GL %d print %c\n", cs, ch)
			}
		} else {
			// GR range
			cs := emu.charsetState.g[emu.charsetState.gr]
			if cs == nil { //} Charset_UTF8 {
				fmt.Printf("GR UTF8 print %c\n", r)
			} else if r >= 160 && (cs == &isoLatin1SupplementalVT300 || r < 255) { // Charset_IsoLatin1
				ch := lookupTable(cs, byte(r))
				fmt.Printf("GR %d print %c\n", cs, ch)
			}
		}
	*/
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

// horizontal tab set
// ESC H Tab Set (HTS is 0x88).
// set cursor position as tab stop position
func hdl_c0_hts(emu *emulator) {
	emu.framebuffer.DS.SetTab()
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
