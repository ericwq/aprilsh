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
	"strings"
	"testing"
)

func TestHandle_DECSC_DECRC_DECSET_1048(t *testing.T) {
	tc := []struct {
		name       string
		seq        string
		hdIDs      []int
		posY, posX int
		originMode OriginMode
	}{
		// move cursor to (8,8), set originMode scrolling, DECSC
		// move cursor to (23,13), set originMode absolute, DECRC
		{
			"ESC DECSC/DECRC",
			"\x1B[?6h\x1B[9;9H\x1B7\x1B[24;14H\x1B[?6l\x1B8",
			[]int{csi_decset, csi_cup, esc_decsc, csi_cup, csi_decrst, esc_decrc},
			8, 8, OriginMode_ScrollingRegion,
		},
		// move cursor to (9,9), set originMode absolute, DECSET 1048
		// move cursor to (21,11), set originMode scrolling, DECRST 1048
		{
			"CSI DECSET/DECRST 1048",
			"\x1B[10;10H\x1B[?6l\x1B[?1048h\x1B[22;12H\x1B[?6h\x1B[?1048l",
			[]int{csi_cup, csi_decrst, csi_decset, csi_cup, csi_decset, csi_decrst},
			9, 9, OriginMode_Absolute,
		},
	}

	p := NewParser()
	emu := NewEmulator3(80, 40, 500)
	var place strings.Builder
	emu.logT.SetOutput(&place)

	for _, v := range tc {
		// process control sequence
		hds := make([]*Handler, 0, 16)
		hds = p.processStream(v.seq, hds)

		if len(hds) == 0 {
			t.Errorf("%s got zero handlers.", v.name)
		}

		// handle the control sequence
		for j, hd := range hds {
			hd.handle(emu)
			if hd.id != v.hdIDs[j] { // validate the control sequences id
				t.Errorf("%s: seq=%q expect %s, got %s\n", v.name, v.seq, strHandlerID[v.hdIDs[j]], strHandlerID[hd.id])
			}
		}

		// validate the result
		x := emu.posX
		y := emu.posY
		mode := emu.originMode

		if x != v.posX || y != v.posY || mode != v.originMode {
			t.Errorf("%s seq=%q expect (%d,%d), got (%d,%d)\n", v.name, v.seq, v.posY, v.posX, y, x)
		}
	}
}
