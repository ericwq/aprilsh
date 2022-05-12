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

import "fmt"

// Handler is the parsing result. It can be used to perform control sequence
// on emulator.
type Handler struct {
	name   string             // the name of ActOn
	ch     rune               // the last byte
	handle func(emu emulator) // handle the action on emulator
}

// CSI Ps A  Cursor Up Ps Times (default = 1) (CUU).
// CSI Ps B  Cursor Down Ps Times (default = 1) (CUD).
// CSI Ps C  Cursor Forward Ps Times (default = 1) (CUF).
// CSI Ps D  Cursor Backward Ps Times (default = 1) (CUB).
func hd_cursor_move(emu emulator, ch rune, num int) {
	switch ch {
	case 'A':
		emu.framebuffer.DS.MoveRow(-num, true)
	case 'B':
		emu.framebuffer.DS.MoveRow(num, true)
	case 'C':
		emu.framebuffer.DS.MoveCol(num, true, false)
	case 'D':
		emu.framebuffer.DS.MoveCol(-num, true, false)
	}
}

// CSI Ps ; Ps H Cursor Position [row;column] (default = [1,1]) (CUP).
// CSI Ps ; Ps f Horizontal and Vertical Position [row;column] (default = [1,1]) (HVP).
func hdl_cup(emu emulator, row int, col int) {
	emu.framebuffer.DS.MoveRow(row-1, false)
	emu.framebuffer.DS.MoveCol(col-1, false, false)
}

func hdl_osc_10(_ emulator, cmd int, arg string) {
	fmt.Printf("handle osc dynamic cmd=%d, arg=%s\n", cmd, arg)
}

func hdl_osc_52(_ emulator, cmd int, arg string) {
	fmt.Printf("handle osc copy cmd=%d, arg=%s\n", cmd, arg)
}

func hdl_osc_4(_ emulator, cmd int, arg string) {
	fmt.Printf("handle osc palette cmd=%d, arg=%s\n", cmd, arg)
}

func hdl_osc_0(emu emulator, cmd int, arg string) {
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
