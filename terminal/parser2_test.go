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
	"sort"
	"strings"
	"testing"
)

func damageArea(cf *Framebuffer, y1, x1, y2, x2 int) (start, end int) {
	start = cf.getIdx(y1, x1)
	end = cf.getIdx(y2, x2)
	return
}

// if the y,x is in the range, return true, otherwise return false
// func inRange(startY, startX, endY, endX, y, x, width int) bool {
// 	pStart := startY*width + startX
// 	pEnd := endY*width + endX
//
// 	p := y*width + x
//
// 	if pStart <= p && p <= pEnd {
// 		return true
// 	}
// 	return false
// }

// func fillRowWith(row *Row, r rune) {
// 	for i := range row.cells {
// 		row.cells[i].contents = string(r)
// 	}
// }

func isTabStop(emu *emulator, x int) bool {
	data := emu.tabStops

	i := sort.Search(len(data), func(i int) bool { return data[i] >= x })
	if i < len(data) && data[i] == x {
		return true
		// x is present at data[i]
	}
	return false
}

func TestHandle_SCOSC_SCORC(t *testing.T) {
	tc := []struct {
		name       string
		seq        string
		hdIDs      []int
		posY, posX int
		set        bool
		msg        string
	}{
		{
			"move cursor, SCOSC, check", "\x1B[22;33H\x1B[s",
			[]int{csi_cup, csi_slrm_scosc},
			22 - 1, 33 - 1, true, "",
		},
		{
			"move cursor, SCOSC, move cursor, SCORC, check", "\x1B[33;44H\x1B[s\x1B[42;35H\x1B[u",
			[]int{csi_cup, csi_slrm_scosc, csi_cup, csi_scorc},
			33 - 1, 44 - 1, false, "",
		},
		{
			"SCORC, check", "\x1B[u",
			[]int{csi_scorc},
			0, 0, false, "Asked to restore cursor (SCORC) but it has not been saved.",
		},
	}

	p := NewParser()
	emu := NewEmulator3(80, 40, 500)

	var place strings.Builder
	emu.logI.SetOutput(&place) // redirect the output to the string builder
	emu.logT.SetOutput(&place) // redirect the output to the string builder

	for i, v := range tc {
		place.Reset()

		// parse control sequence
		hds := make([]*Handler, 0, 16)
		hds = p.processStream(v.seq, hds)

		if len(hds) < 1 {
			t.Errorf("%s got %d handlers.", v.name, len(hds))
		}

		// handle the control sequence
		for j, hd := range hds {
			hd.handle(emu)
			if hd.id != v.hdIDs[j] { // validate the control sequences id
				t.Errorf("%s: seq=%q expect %s, got %s\n", v.name, v.seq, strHandlerID[v.hdIDs[j]], strHandlerID[hd.id])
			}
		}

		switch i {
		case 0, 1:
			gotCol := emu.savedCursor_SCO.posX
			gotRow := emu.savedCursor_SCO.posY
			gotSet := emu.savedCursor_SCO.isSet

			if gotCol != v.posX || gotRow != v.posY || gotSet != v.set {
				t.Errorf("%s:\t %q expect {%d,%d,%t}, got %v", v.name, v.seq, v.posY, v.posX, v.set, emu.savedCursor_SCO)
			}
		case 2:
			got := strings.Contains(place.String(), v.msg)
			if !got {
				t.Errorf("%s:\t %q expect %s, got %s\n", v.name, v.seq, v.msg, place.String())
			}
		}
	}
}

func TestHandle_DECSC_DECRC_privSM_1048(t *testing.T) {
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
			[]int{csi_privSM, csi_cup, esc_decsc, csi_cup, csi_privRM, esc_decrc},
			8, 8, OriginMode_ScrollingRegion,
		},
		// move cursor to (9,9), set originMode absolute, privSM 1048
		// move cursor to (21,11), set originMode scrolling, privRM 1048
		{
			"CSI privSM/privRM 1048",
			"\x1B[10;10H\x1B[?6l\x1B[?1048h\x1B[22;12H\x1B[?6h\x1B[?1048l",
			[]int{csi_cup, csi_privRM, csi_privSM, csi_cup, csi_privSM, csi_privRM},
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

// make sure this is a new initialized CharsetState
func isResetCharsetState(cs CharsetState) (ret bool) {
	ret = true
	for _, v := range cs.g {
		if v != nil {
			return false
		}
	}

	if cs.gl != 0 || cs.gr != 2 || cs.ss != 0 {
		return false
	}

	if cs.vtMode {
		ret = false
	}
	return ret
}

func TestHandle_DECSLRM(t *testing.T) {
	tc := []struct {
		name                    string
		seq                     string
		hdIDs                   []int
		leftMargin, rightMargin int
		posX, posY              int
	}{
		{
			"set left right margin, normal",
			"\x1B[?69h\x1B[4;70s",
			[]int{csi_privSM, csi_slrm_scosc},
			3, 70, 0, 0,
		},
		{
			"set left right margin, missing right parameter",
			"\x1B[?69h\x1B[1s",
			[]int{csi_privSM, csi_slrm_scosc},
			0, 80, 0, 0,
		},
		{
			"set left right margin, left parameter is zero",
			"\x1B[?69h\x1B[0s",
			[]int{csi_privSM, csi_slrm_scosc},
			0, 80, 0, 0,
		},
	}

	p := NewParser()
	emu := NewEmulator3(80, 40, 500)
	var place strings.Builder
	emu.logI.SetOutput(&place)
	emu.logT.SetOutput(&place)

	for _, v := range tc {

		// parse control sequence
		hds := make([]*Handler, 0, 16)
		hds = p.processStream(v.seq, hds)

		if len(hds) != 2 {
			t.Errorf("%s got %d handlers, expect 2 handlers.", v.name, len(hds))
		}

		// handle the control sequence
		for j, hd := range hds {
			hd.handle(emu)
			if hd.id != v.hdIDs[j] { // validate the control sequences id
				t.Errorf("%s:\t %q expect %s, got %s\n", v.name, v.seq, strHandlerID[v.hdIDs[j]], strHandlerID[hd.id])
			}
			switch j {
			case 0:
				gotMode := emu.horizMarginMode
				if gotMode != true {
					t.Errorf("%s:\t %q expect %t, got %t\n", v.name, v.seq, true, gotMode)
				}
			case 1:
				// validate the left/right margin
				gotLeft := emu.hMargin
				gotRight := emu.nColsEff
				if gotLeft != v.leftMargin || gotRight != v.rightMargin {
					t.Errorf("%s:\t %q expect (%d,%d), got (%d,%d)\n", v.name, v.seq, v.leftMargin, v.rightMargin, gotLeft, gotRight)
				}

				// validate the cursor row/col
				posY := emu.posY
				posXZ := emu.posX

				if posY != v.posY || posXZ != v.posX {
					t.Errorf("%s:\t %q expect (%d/%d), got (%d/%d)\n", v.name, v.seq, v.posX, v.posY, posXZ, posY)
				}
			}
		}
	}
}

func TestHandle_DECSLRM_Others(t *testing.T) {
	tc := []struct {
		name        string
		seq         string
		hdIDs       []int
		logMsg      string
		left, right int
		posY, posX  int
	}{
		{
			"DECLRMM disable", "\x1B[?69l\x1B[4;49s",
			[]int{csi_privRM, csi_slrm_scosc},
			"", 0, 0, 0, 0,
		},
		{
			"DECLRMM enable, outof range", "\x1B[?69h\x1B[4;89s",
			[]int{csi_privSM, csi_slrm_scosc},
			"Illegal arguments to SetLeftRightMargins:", 0, 0, 0, 0,
		},
		{
			"DECLRMM OriginMode_ScrollingRegion, enable", "\x1B[?6h\x1B[?69h\x1B[4;69s", // DECLRMM: Set Left and Right Margins
			[]int{csi_privSM, csi_privSM, csi_slrm_scosc},
			"", 3, 69, 0, 3,
		},
	}

	p := NewParser()
	emu := NewEmulator3(80, 40, 500)
	var place strings.Builder
	emu.logI.SetOutput(&place)
	emu.logT.SetOutput(&place)

	for i, v := range tc {

		// parse control sequence
		hds := make([]*Handler, 0, 16)
		hds = p.processStream(v.seq, hds)

		if len(hds) < 2 {
			t.Errorf("%s got %d handlers, expect at lease 2 handlers.", v.name, len(hds))
		}

		// handle the control sequence
		for j, hd := range hds {
			hd.handle(emu)
			if hd.id != v.hdIDs[j] { // validate the control sequences id
				t.Errorf("%s:\t %q expect %s, got %s\n", v.name, v.seq, strHandlerID[v.hdIDs[j]], strHandlerID[hd.id])
			}
		}

		switch i {
		case 0:
			if emu.horizMarginMode {
				t.Errorf("%s: seq=%q expect %t, got %t\n", v.name, v.seq, false, emu.horizMarginMode)
			}
		case 1:
			got := strings.Contains(place.String(), v.logMsg)
			if !got {
				t.Errorf("%s: seq=%q expect %q, got %q\n", v.name, v.seq, v.logMsg, place.String())
			}
		case 2:
			// validate the left/right margin
			left := emu.hMargin
			right := emu.nColsEff
			if left != v.left || right != v.right {
				t.Errorf("%s: seq=%q expect left/right margin (%d,%d), got (%d,%d)\n", v.name, v.seq, v.left, v.right, left, right)
			}

			// validate the cursor row/col
			posY := emu.posY
			posX := emu.posX

			if posY != v.posY || posX != v.posX {
				t.Errorf("%s: seq=%q expect cursor (%d,%d), got (%d,%d)\n", v.name, v.seq, v.posY, v.posX, posY, posX)
			}
		}
	}
}

func TestHandle_DECSTR(t *testing.T) {
	tc := []struct {
		name           string
		seq            string
		hdIDs          []int
		insertMode     bool
		originMode     OriginMode
		showCursorMode bool
		cursorKeyMode  CursorKeyMode
		reverseVideo   bool
	}{
		{
			"DECSTR ",
			/*
				set ture for insertMode=true, originMode=OriginMode_ScrollingRegion,
				showCursorMode=false, cursorKeyMode = CursorKeyMode_Application,reverseVideo = true
				set top/bottom region = [1,30)
				we don't check the response of the above sequence, we choose the opposite value on purpose
				(finally) soft terminal reset, check the opposite result for the soft reset sequence.
			*/
			"\x1B[4h\x1B[?6h\x1B[?25l\x1B[?1h\x1B[2;30r\x1B[!p",
			[]int{csi_sm, csi_privSM, csi_privRM, csi_privSM, csi_decstbm, csi_decstr},
			false, OriginMode_Absolute, true, CursorKeyMode_ANSI, false,
		},
	}

	p := NewParser()
	emu := NewEmulator3(80, 40, 500)
	var place strings.Builder
	emu.logI.SetOutput(&place)
	emu.logT.SetOutput(&place)

	for _, v := range tc {

		// process control sequence
		hds := make([]*Handler, 0, 16)
		hds = p.processStream(v.seq, hds)

		if len(hds) == 0 {
			t.Errorf("%s got zero handlers.", v.name)
		}

		// execute the control sequence
		for j, hd := range hds {
			hd.handle(emu)
			if hd.id != v.hdIDs[j] { // validate the control sequences id
				t.Errorf("%s:\t %q expect %s, got %s\n", v.name, v.seq, strHandlerID[v.hdIDs[j]], strHandlerID[hd.id])
			}
		}

		// validate the insertMode
		insertMode := emu.insertMode
		if insertMode != v.insertMode {
			t.Errorf("%s seq=%q insertMode expect %t, got %t\n", v.name, v.seq, v.insertMode, insertMode)
		}
		originMode := emu.originMode
		if originMode != v.originMode {
			t.Errorf("%s seq=%q originMode expect %d, got %d\n", v.name, v.seq, v.originMode, originMode)
		}
		showCursorMode := emu.showCursorMode
		if showCursorMode != v.showCursorMode {
			t.Errorf("%s seq=%q showCursorMode expect %t, got %t\n", v.name, v.seq, v.showCursorMode, showCursorMode)
		}
		cursorKeyMode := emu.cursorKeyMode
		if cursorKeyMode != v.cursorKeyMode {
			t.Errorf("%s seq=%q cursorKeyMode expect %d, got %d\n", v.name, v.seq, v.cursorKeyMode, cursorKeyMode)
		}
		reverseVideo := emu.reverseVideo
		if reverseVideo != v.reverseVideo {
			t.Errorf("%s seq=%q reverseVideo expect %t, got %t\n", v.name, v.seq, v.reverseVideo, reverseVideo)
		}
	}
}

func TestHandle_CR_LF_VT_FF(t *testing.T) {
	tc := []struct {
		name  string
		hdIDs []int
		posY  int
		posX  int
		seq   string
	}{
		{"CR 1 ", []int{csi_cup, c0_cr}, 2, 0, "\x1B[3;2H\x0D"},
		{"CR 2 ", []int{csi_cup, c0_cr}, 4, 0, "\x1B[5;10H\x0D"},
		{"LF   ", []int{csi_cup, esc_ind}, 3, 1, "\x1B[3;2H\x0C"},
		{"VT   ", []int{csi_cup, esc_ind}, 4, 2, "\x1B[4;3H\x0B"},
		{"FF   ", []int{csi_cup, esc_ind}, 5, 3, "\x1B[5;4H\x0C"},
		{"ESC D", []int{csi_cup, esc_ind}, 6, 4, "\x1B[6;5H\x1BD"},
		{"CHA CR", []int{csi_privSM, csi_slrm_scosc, csi_cup, c0_cr}, 4, 0, "\x1B[?69h\x1B[4;70s\x1B[5;1H\x0D"},
	}

	p := NewParser()
	emu := NewEmulator3(80, 40, 500)
	var place strings.Builder
	emu.logI.SetOutput(&place)
	emu.logT.SetOutput(&place)
	for _, v := range tc {

		// parse the sequence
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

		// get final cursor position
		gotY := emu.posY
		gotX := emu.posX

		if gotX != v.posX || gotY != v.posY {
			t.Errorf("%s seq=%q expect cursor position (%d,%d), got (%d,%d)\n", v.name, v.seq, v.posX, v.posY, gotX, gotY)
		}
	}
}

func TestHandle_CSI_BS_FF_VT_CR_TAB(t *testing.T) {
	tc := []struct {
		name         string
		hdIDs        []int
		seq          string
		wantY, wantX int
	}{
		// call CUP first to set the start position
		{"CSI backspace number    ", []int{csi_cup, csi_cup}, "\x1B[1;1H\x1B[23;12\bH", 22, 0},       // undo last character in CSI sequence
		{"CSI backspace semicolon ", []int{csi_cup, csi_cup}, "\x1B[1;1H\x1B[23;\b;12H", 22, 11},     // undo last character in CSI sequence
		{"cursor down 1+3 rows VT ", []int{csi_cup, esc_ind, csi_cud}, "\x1B[9;10H\x1B[3\vB", 12, 9}, //(8,9)->(9.9)->(12,9)
		{"cursor down 1+3 rows FF ", []int{csi_cup, esc_ind, csi_cud}, "\x1B[9;10H\x1B[\f3B", 12, 9},
		{"cursor up 2 rows and CR ", []int{csi_cup, c0_cr, csi_cuu}, "\x1B[8;9H\x1B[\r2A", 5, 0},
		{"cursor up 3 rows and CR ", []int{csi_cup, c0_cr, csi_cuu}, "\x1B[7;7H\x1B[3\rA", 3, 0},
		{"cursor forward 2cols +HT", []int{csi_cup, c0_ht, csi_cuf}, "\x1B[4;6H\x1B[2\tC", 3, 10},
		{"cursor forward 1cols +HT", []int{csi_cup, c0_ht, csi_cuf}, "\x1B[6;3H\x1B[\t1C", 5, 9},
	}

	p := NewParser()
	emu := NewEmulator3(80, 40, 500)
	var place strings.Builder
	emu.logI.SetOutput(&place)
	emu.logT.SetOutput(&place)

	for _, v := range tc {
		// parse control sequence
		hds := make([]*Handler, 0, 16)
		hds = p.processStream(v.seq, hds)

		if len(hds) == 0 {
			t.Errorf("%s got zero handlers.", v.name)
		}

		// handle the control sequence
		for j, hd := range hds {
			hd.handle(emu)
			if hd.id != v.hdIDs[j] { // validate the control sequences id
				t.Errorf("%s: seq=%q hd[%d] expect %s, got %s\n", v.name, v.seq, j, strHandlerID[v.hdIDs[j]], strHandlerID[hd.id])
			}
		}

		// get the result
		gotY := emu.posY
		gotX := emu.posX
		if gotX != v.wantX || gotY != v.wantY {
			t.Errorf("%s: seq=%q expect cursor position (%d,%d), got (%d,%d)\n", v.name, v.seq, v.wantY, v.wantX, gotY, gotX)
		}
	}
}

func TestHandle_CUU_CUD_CUF_CUB_CUP_FI_BI(t *testing.T) {
	tc := []struct {
		name  string
		hdIDs []int
		wantY int
		wantX int
		seq   string
	}{
		// call CUP first to set the start position
		{"CSI Ps A  ", []int{csi_cup, csi_cuu}, 14, 10, "\x1B[21;11H\x1B[6A"},
		{"CSI Ps B  ", []int{csi_cup, csi_cud}, 13, 10, "\x1B[11;11H\x1B[3B"},
		{"CSI Ps C  ", []int{csi_cup, csi_cuf}, 10, 12, "\x1B[11;11H\x1B[2C"},
		{"CSI Ps D  ", []int{csi_cup, csi_cub}, 10, 12, "\x1B[11;21H\x1B[8D"},
		{"BS        ", []int{csi_cup, csi_cub}, 12, 11, "\x1B[13;13H\x08"}, // \x08 calls CUB
		{"CUB       ", []int{csi_cup, csi_cub}, 12, 11, "\x1B[13;13H\x1B[1D"},
		{"BS agin   ", []int{csi_cup, csi_cub}, 12, 10, "\x1B[13;12H\x08"}, // \x08 calls CUB
		{"DECFI     ", []int{csi_cup, esc_fi}, 12, 22, "\x1B[13;22H\x1b9"},
		{"DECBI     ", []int{csi_cup, esc_bi}, 12, 20, "\x1B[13;22H\x1b6"},
		{"CUU with STBM", []int{csi_cup, csi_decstbm, csi_cuu}, 0, 0, "\x1B[2;1H\x1B[3;32r\x1B[7A"},
		// {"CUD with STBM", []int{csi_cup, csi_decstbm, csi_cud}, 3, 0, "\x1B[40;80H\x1B[3;36r\x1B[3B"},
	}

	p := NewParser()
	emu := NewEmulator3(80, 40, 40)
	var place strings.Builder
	emu.logI.SetOutput(&place)
	emu.logT.SetOutput(&place)

	for _, v := range tc {

		// parse the sequence
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

		// get final cursor position
		gotY := emu.posY
		gotX := emu.posX

		if gotX != v.wantX || gotY != v.wantY {
			t.Errorf("%s seq=%q expect cursor position (%d,%d), got (%d,%d)\n", v.name, v.seq, v.wantX, v.wantY, gotX, gotY)
		}
	}
}

func TestHandle_SU_SD(t *testing.T) {
	nCols := 8
	nRows := 5
	saveLines := 5
	tc := []struct {
		name      string
		hdIDs     []int
		emptyRows []int
		seq       string
	}{
		{"SU scroll up   2 lines", []int{csi_su}, []int{nRows - 2, nRows - 1}, "\x1B[2S"}, // bottom 2 is erased
		{"SD scroll down 3 lines", []int{csi_sd}, []int{0, 1, 2}, "\x1B[3T"},              // top three is erased.
	}

	p := NewParser()

	for _, v := range tc {
		// the terminal size is 8x5 [colxrow]
		emu := NewEmulator3(nCols, nRows, saveLines)
		var place strings.Builder
		emu.logI.SetOutput(&place)
		emu.logT.SetOutput(&place)

		hds := make([]*Handler, 0, 16)
		hds = p.processStream(v.seq, hds)

		if len(hds) == 0 {
			t.Errorf("%s got zero handlers.", v.name)
		}

		fillCells(emu.cf)
		before := printCells(emu.cf)

		// handle the control sequence
		for j, hd := range hds {
			hd.handle(emu)
			if hd.id != v.hdIDs[j] { // validate the control sequences id
				t.Errorf("%s: seq=%q expect %s, got %s\n", v.name, v.seq, strHandlerID[v.hdIDs[j]], strHandlerID[hd.id])
			}
		}

		after := printCells(emu.cf)

		if !isEmptyRows(emu.cf, v.emptyRows...) {
			t.Errorf("%s:\n", v.name)
			t.Logf("[frame] scrollHead=%d marginTop=%d marginBottom=%d [emulator] marginTop=%d marginBottom=%d\n",
				emu.cf.scrollHead, emu.cf.marginTop, emu.cf.marginBottom, emu.marginTop, emu.marginBottom)
			t.Errorf("before:\n%s", before)
			t.Errorf("after:\n%s", after)
		}
	}
}

func TestHandle_HTS_TBC(t *testing.T) {
	tc := []struct {
		name  string
		hdIDs []int
		seq   string
	}{
		{"Set/Clear tab stop 1", []int{csi_cup, esc_hts, csi_tbc}, "\x1B[21;19H\x1BH\x1B[g"}, // set tab stop; clear tab stop
		{"Set/Clear tab stop 2", []int{csi_cup, esc_hts, csi_tbc}, "\x1B[21;39H\x1BH\x1B[0g"},
		{"Set/Clear tab stop 3", []int{csi_cup, esc_hts, csi_tbc}, "\x1B[21;47H\x1BH\x1B[3g"},
	}
	// TODO test to see the HTS same position
	p := NewParser()
	emu := NewEmulator3(80, 40, 5)
	var place strings.Builder
	emu.logI.SetOutput(&place)
	emu.logT.SetOutput(&place)

	for _, v := range tc {

		hds := make([]*Handler, 0, 16)
		hds = p.processStream(v.seq, hds)

		if len(hds) != 3 {
			t.Errorf("%s expect %d handlers, got %d handlers.", v.name, 3, len(hds))
		}

		// handle the control sequence
		for j, hd := range hds {
			hd.handle(emu)
			if hd.id != v.hdIDs[j] { // validate the control sequences id
				t.Errorf("%s: seq=%q expect %s, got %s\n", v.name, v.seq, strHandlerID[v.hdIDs[j]], strHandlerID[hd.id])
			}

			gotX := emu.posX
			switch j {
			case 0:
				if isTabStop(emu, gotX) {
					t.Errorf("%s seq=%q expect position %d is not tab stop, it is\n", v.name, v.seq, gotX)
				}
			case 1:
				if !isTabStop(emu, gotX) {
					t.Errorf("%s seq=%q expect position %d is not tab stop, it is\n", v.name, v.seq, gotX)
				}
			case 2:
				if isTabStop(emu, gotX) {
					t.Errorf("%s seq=%q expect position %d is not tab stop, it is\n", v.name, v.seq, gotX)
				}
			}
		}
	}
}

func TestHandle_HT_CHT_CBT(t *testing.T) {
	tc := []struct {
		name  string
		hdIDs []int
		posX  int
		seq   string
	}{
		{"HT case 1  ", []int{csi_cup, c0_ht}, 8, "\x1B[21;6H\x09"},                 // move to the next tab stop
		{"HT case 2  ", []int{csi_cup, c0_ht}, 16, "\x1B[21;10H\x09"},               // move to the next tab stop
		{"CBT back to the 3 tab", []int{csi_cup, csi_cbt}, 8, "\x1B[21;30H\x1B[3Z"}, // move backward to the previous 3 tab stop
		{"CHT to the next 1 tab", []int{csi_cup, csi_cht}, 8, "\x1B[21;3H\x1B[I"},   // move to the next N tab stop
		{"CHT to the next 4 tab", []int{csi_cup, csi_cht}, 32, "\x1B[21;3H\x1B[4I"}, // move to the next N tab stop
		{"CHT to the right edge", []int{csi_cup, csi_cht}, 79, "\x1B[21;60H\x1B[4I"},
		{"CBT rule to left edge", []int{csi_cup, csi_cbt}, 0, "\x1B[21;3H\x1B[3Z"}, // under tab rules
		{
			"CBT tab stop to left edge",
			[]int{csi_cup, esc_hts, csi_cup, esc_hts, csi_cbt}, // set 2 tab stops, CBT 2 backwards
			0,
			"\x1B[21;4H\x1BH\x1B[21;7H\x1BH\x1B[2Z",
		},
	}

	p := NewParser()
	emu := NewEmulator3(80, 40, 5)
	var place strings.Builder
	emu.logI.SetOutput(&place)
	emu.logT.SetOutput(&place)

	for _, v := range tc {

		hds := make([]*Handler, 0, 16)
		hds = p.processStream(v.seq, hds)

		if len(hds) < 2 {
			t.Errorf("%s expect %d handlers, got %d handlers.", v.name, 2, len(hds))
		}

		// handle the control sequence
		for j, hd := range hds {
			hd.handle(emu)
			if hd.id != v.hdIDs[j] { // validate the control sequences id
				t.Errorf("%s: seq=%q expect %s, got %s\n", v.name, v.seq, strHandlerID[v.hdIDs[j]], strHandlerID[hd.id])
			}
		}

		// get the result
		gotX := emu.posX
		if gotX != v.posX {
			t.Errorf("%s seq=%q expect cursor cols: %d, got %d)\n", v.name, v.seq, v.posX, gotX)
		}
	}
}

func TestHandle_LF_ScrollUp(t *testing.T) {
	tc := []struct {
		name             string
		posY             int
		expectScrollHead int
		seq              string
	}{
		{"LF within active area", 3, 0, "\x0A\x0A\x0A"},
		{"LF outof active area", 3, 2, "\x0A\x0A\x0A\x0A\x0A"},
		{"wrap around margin bottom", 3, 1, "\n\n\n\n\n\n\n\n\n\n\n\n"},
	}

	p := NewParser()
	emu := NewEmulator3(8, 4, 4) // this is the pre-condidtion for the test case.
	var place strings.Builder
	emu.logI.SetOutput(&place)
	emu.logT.SetOutput(&place)

	for _, v := range tc {
		emu.resetTerminal()

		hds := make([]*Handler, 0, 16)
		hds = p.processStream(v.seq, hds)

		if len(hds) == 0 {
			t.Errorf("%s got %d handlers.", v.name, len(hds))
		}

		// handle the control sequence
		for _, hd := range hds {
			hd.handle(emu)
			// if i == 2 {
			// 	t.Logf("%s [frame] scrollHead=%d historyRows=%d [emulator] posY=%d\n",
			// 		v.name, emu.cf.scrollHead, emu.cf.historyRows, emu.posY)
			// }
		}

		gotY := emu.posY
		gotHead := emu.cf.scrollHead
		if gotY != v.posY || gotHead != v.expectScrollHead {
			t.Errorf("%s marginTop=%d, marginBottom=%d scrollHead=%d\n",
				v.name, emu.cf.marginTop, emu.cf.marginBottom, emu.cf.scrollHead)
			t.Errorf("%s seq=%q expect posY=%d, scrollHead=%d, got posY=%d, scrollHead=%d\n",
				v.name, v.seq, v.posY, v.expectScrollHead, gotY, gotHead)
		}
	}
}

func TestHandle_DECIC_DECDC(t *testing.T) {
	tc := []struct {
		name      string
		seq       string
		emptyCols []int
		hdIDs     []int
	}{
		// move cursor to start position, and perform insert and delete
		{"insert at left side ", "\x1B[2;1H\x1B[3'}", []int{0, 1, 2}, []int{csi_cup, csi_decic}},
		{"insert at middle    ", "\x1B[2;4H\x1B[2'}", []int{3, 4}, []int{csi_cup, csi_decic}},
		{"insert at right side", "\x1B[1;8H\x1B[2'}", []int{7}, []int{csi_cup, csi_decic}},
		{"delete at left side ", "\x1B[1;1H\x1B[3'~", []int{5, 6, 7}, []int{csi_cup, csi_decdc}},
		{"delete at middle    ", "\x1B[1;4H\x1B[2'~", []int{6, 7}, []int{csi_cup, csi_decdc}},
		{"delete at right side", "\x1B[1;8H\x1B[2'~", []int{7}, []int{csi_cup, csi_decdc}},
	}

	for _, v := range tc {
		p := NewParser()
		emu := NewEmulator3(8, 4, 4) // this is the pre-condidtion for the test case.
		var place strings.Builder
		emu.logI.SetOutput(&place)
		emu.logT.SetOutput(&place)

		fillCells(emu.cf)
		before := printCells(emu.cf)

		hds := make([]*Handler, 0, 16)
		hds = p.processStream(v.seq, hds)

		if len(hds) == 0 {
			t.Errorf("%s got %d handlers.", v.name, len(hds))
		}

		// handle the control sequence
		for j, hd := range hds {
			hd.handle(emu)
			if hd.id != v.hdIDs[j] { // validate the control sequences id
				t.Errorf("%s: seq=%q expect %s, got %s\n", v.name, v.seq, strHandlerID[v.hdIDs[j]], strHandlerID[hd.id])
			}
		}

		after := printCells(emu.cf)
		// validate the empty cell
		if !isEmptyCols(emu.cf, v.emptyCols...) {
			t.Errorf("%s:\n", v.name)
			t.Errorf("[before]\n%s", before)
			t.Errorf("[after ]\n%s", after)
		}
	}
}

func TestHandle_DECALN_RIS(t *testing.T) {
	tc := []struct {
		name  string
		seq   string
		y, x  int // check the last cell on the screen
		hdIDs []int
		want  string
	}{
		{"ESC DECLAN", "\x1B#8", 3, 7, []int{esc_decaln}, "E"}, // the whole screen is filled with 'E'
		{"ESC RIS   ", "\x1Bc", 3, 7, []int{esc_ris}, " "},     // after reset, the screen is empty
	}

	p := NewParser()
	emu := NewEmulator3(8, 4, 4) // this is the pre-condidtion for the test case.
	var place strings.Builder
	emu.logI.SetOutput(&place)
	emu.logT.SetOutput(&place)

	for _, v := range tc {
		hds := make([]*Handler, 0, 16)
		hds = p.processStream(v.seq, hds)

		if len(hds) == 0 {
			t.Errorf("%s expect %d handlers, got %d handlers.", v.name, 2, len(hds))
		}

		before := printCells(emu.cf)
		// handle the control sequence
		for j, hd := range hds {
			hd.handle(emu)
			if hd.id != v.hdIDs[j] { // validate the control sequences id
				t.Errorf("%s: seq=%q expect %s, got %s\n",
					v.name, v.seq, strHandlerID[v.hdIDs[j]], strHandlerID[hd.id])
			}
		}

		after := printCells(emu.cf)
		theCell := emu.cf.getCell(v.y, v.x)
		if v.want != theCell.contents {
			t.Errorf("%s seq=%q expect %q on position (%d,%d), got %q\n", v.name, v.seq, v.want, v.y, v.x, theCell.contents)
			t.Errorf("[before]\n%s", before)
			t.Errorf("[after ]\n%s", after)
		}
	}
}

// use DECALN to fill the screen, then call ED to erase part of it.
func TestHandle_ED_IL_DL(t *testing.T) {
	tc := []struct {
		name     string
		hdIDs    []int
		tlY, tlX int
		brY, brX int
		seq      string
		msg      string
	}{
		// use CUP to move cursor to start position, use DECALN to fill the screen, then call ED,IL or DL
		{"ED erase below @ 1,0", []int{csi_cup, esc_decaln, csi_ed}, 1, 0, 3, 7, "\x1B[2;1H\x1B#8\x1B[J", "unused"},  // Erase Below (default).
		{"ED erase below @ 3,7", []int{csi_cup, esc_decaln, csi_ed}, 3, 6, 3, 7, "\x1B[4;7H\x1B#8\x1B[0J", "unused"}, // Ps = 0  ⇒  Erase Below (default).
		{"ED erase above @ 3,6", []int{csi_cup, esc_decaln, csi_ed}, 0, 0, 3, 6, "\x1B[4;7H\x1B#8\x1B[1J", "unused"}, // Ps = 1  ⇒  Erase Above.
		{"ED erase all", []int{csi_cup, esc_decaln, csi_ed}, 0, 0, 3, 7, "\x1B[4;7H\x1B#8\x1B[2J", "unused"},         // Ps = 2  ⇒  Erase All.
		{"ED saved lines, all", []int{csi_cup, esc_decaln, csi_ed}, 0, 0, 7, 7, "\x1B[4;7H\x1B#8\x1B[3J", "unused"},  // Ps = 3  ⇒  Erase saved lines.
		{"IL 1 lines @ 2,2 mid", []int{csi_cup, esc_decaln, csi_il}, 2, 0, 3, 7, "\x1B[3;3H\x1B#8\x1B[L", "unused"},
		{"IL 2 lines @ 1,0 bottom", []int{csi_cup, esc_decaln, csi_il}, 1, 0, 3, 7, "\x1B[2;1H\x1B#8\x1B[2L", "unused"},
		{"IL 4 lines @ 0,0 top", []int{esc_decaln, csi_cup, csi_il}, 0, 0, 3, 7, "\x1B#8\x1B[1;1H\x1B[4L", "unused"},
		{"DL 2 lines @ 1,0 top", []int{esc_decaln, csi_cup, csi_dl}, 1, 0, 3, 7, "\x1B#8\x1B[2;1H\x1B[2M", "unused"},
		{"DL 1 lines @ 3,0 bottom", []int{esc_decaln, csi_cup, csi_dl}, 3, 0, 3, 7, "\x1B#8\x1B[4;1H\x1B[1M", "unused"},
		{"ED default", []int{csi_cup, esc_decaln, csi_ed}, 0, 0, 0, 0, "\x1B[4;7H\x1B#8\x1B[4J", "Erase in Display with illegal param:"}, // Unhandled case
	}

	p := NewParser()
	// the default size of emu is 80x40 [colxrow]
	emu := NewEmulator3(8, 4, 4) // this is the pre-condidtion for the test case.
	var place strings.Builder
	emu.logI.SetOutput(&place)
	emu.logT.SetOutput(&place)

	for _, v := range tc {
		place.Reset()

		hds := make([]*Handler, 0, 16)
		hds = p.processStream(v.seq, hds)

		if len(hds) == 0 {
			t.Errorf("%s got zero handlers.", v.name)
		}

		before := ""
		for j, hd := range hds {
			hd.handle(emu)
			if hd.id != v.hdIDs[j] { // validate the control sequences id
				t.Errorf("%s: seq=%q expect %s, got %s\n",
					v.name, v.seq, strHandlerID[v.hdIDs[j]], strHandlerID[hd.id])
			}
			if j == 1 {
				before = printCells(emu.cf)
				emu.cf.damage.reset()
			}
		}

		after := printCells(emu.cf)

		if v.tlX == 0 && v.tlY == 0 && v.brX == 0 && v.brY == 0 {
			if !strings.Contains(place.String(), v.msg) {

				t.Errorf("%s seq=%q\n", v.name, v.seq)
				t.Errorf("expect msg %s, got %s\n", v.msg, place.String())

			}
		} else {

			// calculate the expected dmage area
			dmg := Damage{}
			dmg.totalCells = emu.cf.damage.totalCells
			dmg.start, dmg.end = damageArea(emu.cf, v.tlY, v.tlX, v.brY, v.brX+1) // the end point is exclusive.

			if emu.cf.damage != dmg {
				t.Errorf("%s seq=%q\n", v.name, v.seq)
				t.Errorf("expect damage %v, got %v\n", dmg, emu.cf.damage)
				t.Errorf("[before]\n%s", before)
				t.Errorf("[after ]\n%s", after)
			}
		}
	}
}

func TestHandle_ICH_EL_DCH_ECH(t *testing.T) {
	tc := []struct {
		name     string
		hdIDs    []int
		tlY, tlX int // damage area top/left
		brY, brX int // damage area bottom/right
		seq      string
		emptyY   int // empty cell starting Y
		emptyX   int // empty cell starting X
		count    int // empty cells count number
		msg      string
	}{
		// use DECALN to fill the screen, use CUP to move cursor to start position, then call the sequence
		{"ICH  in middle", []int{esc_decaln, csi_cup, csi_ich}, 0, 2, 0, 7, "\x1B#8\x1B[1;3H\x1B[2@", 0, 2, 2, "unused"},
		{"ICH right side", []int{esc_decaln, csi_cup, csi_ich}, 1, 5, 1, 7, "\x1B#8\x1B[2;6H\x1B[3@", 1, 5, 3, "unused"},
		{"ICH left side ", []int{esc_decaln, csi_cup, csi_ich}, 0, 0, 0, 7, "\x1B#8\x1B[1;1H\x1B[2@", 0, 0, 2, "unused"},
		{"   EL to right", []int{esc_decaln, csi_cup, csi_el}, 3, 3, 3, 7, "\x1B#8\x1B[4;4H\x1B[0K", 3, 3, 5, "unused"},
		{"   EL  to left", []int{esc_decaln, csi_cup, csi_el}, 3, 0, 3, 3, "\x1B#8\x1B[4;4H\x1B[1K", 3, 0, 4, "unused"},
		{"   EL      all", []int{esc_decaln, csi_cup, csi_el}, 3, 0, 3, 7, "\x1B#8\x1B[4;4H\x1B[2K", 3, 0, 8, "unused"},
		{"  DCH  at left", []int{esc_decaln, csi_cup, csi_dch}, 0, 0, 0, 7, "\x1B#8\x1B[1;1H\x1B[2P", 0, 6, 2, "unused"},
		{"  DCH at right", []int{esc_decaln, csi_cup, csi_dch}, 0, 5, 0, 7, "\x1B#8\x1B[1;6H\x1B[3P", 0, 5, 3, "unused"},
		{" DCH in middle", []int{esc_decaln, csi_cup, csi_dch}, 3, 3, 3, 7, "\x1B#8\x1B[4;4H\x1B[20P", 3, 3, 5, "unused"},
		{" ECH in middle", []int{esc_decaln, csi_cup, csi_ech}, 3, 3, 3, 4, "\x1B#8\x1B[4;4H\x1B[2X", 3, 3, 2, "unused"},
		{"   ECH at left", []int{esc_decaln, csi_cup, csi_ech}, 0, 0, 0, 4, "\x1B#8\x1B[1;1H\x1B[5X", 0, 0, 5, "unused"},
		{"  ECH at right", []int{esc_decaln, csi_cup, csi_ech}, 1, 5, 1, 7, "\x1B#8\x1B[2;6H\x1B[5X", 1, 5, 3, "unused"},
		{
			"ICH right side with wrap length==0",
			[]int{csi_cup, graphemes, graphemes, graphemes, graphemes, csi_cup, csi_ich},
			1, 5, 2, 0,
			"\x1B[2;6Hwrap\x1B[2;6H\x1B[3@", 1, 5, 0, "unused",
		},
		{
			"ICH right side with wrap length!=0",
			[]int{csi_cup, graphemes, graphemes, graphemes, graphemes, csi_cup, csi_ich},
			1, 5, 2, 0,
			"\x1B[2;6Hwrap\x1B[2;6H\x1B[2@", 1, 5, 0, "unused",
		},
		{
			"   EL  default",
			[]int{esc_decaln, csi_cup, csi_el},
			0, 0, 0, 0, "\x1B#8\x1B[4;4H\x1B[3K", 3, 0, 8, "Erase in Line with illegal param:",
		},
	}
	p := NewParser()
	emu := NewEmulator3(8, 4, 4) // this is the pre-condidtion for the test case.
	var place strings.Builder
	emu.logI.SetOutput(&place)
	emu.logT.SetOutput(&place)

	for _, v := range tc {

		hds := make([]*Handler, 0, 16)
		hds = p.processStream(v.seq, hds)

		if len(hds) == 0 {
			t.Errorf("%s got zero handlers.", v.name)
		}
		before := ""

		// call the handler
		for j, hd := range hds {
			if j == 1 {
				emu.cf.damage.reset()
			}
			hd.handle(emu)
			if hd.id != v.hdIDs[j] { // validate the control sequences id
				t.Errorf("%s: seq=%q expect %s, got %s\n",
					v.name, v.seq, strHandlerID[v.hdIDs[j]], strHandlerID[hd.id])
			}
			if j == len(hds)-2 {
				before = printCells(emu.cf, v.emptyY)
			}

		}
		after := printCells(emu.cf, v.emptyY, v.emptyY+1)

		if v.tlX == 0 && v.tlY == 0 && v.brX == 0 && v.brY == 0 {
			if !strings.Contains(place.String(), v.msg) {

				t.Errorf("%s seq=%q\n", v.name, v.seq)
				t.Errorf("expect msg %s, got %s\n", v.msg, place.String())

			}
		} else {
			// calculate the expected dmage area
			dmg := Damage{}
			dmg.totalCells = emu.cf.damage.totalCells
			dmg.start, dmg.end = damageArea(emu.cf, v.tlY, v.tlX, v.brY, v.brX+1) // the end point is exclusive.

			if emu.cf.damage != dmg || !isEmptyCells(emu.cf, v.emptyY, v.emptyX, v.count) {
				t.Errorf("%s seq=%q\n", v.name, v.seq)
				t.Errorf("expect damage %v, got %v\n", dmg, emu.cf.damage)
				t.Errorf("empty cells start (%d,%d) count=%d\n", v.emptyY, v.emptyX, v.count)
				t.Errorf("[before] %s", before)
				t.Errorf("[after ] %s", after)
			}
		}
	}
}

func TestHandle_DEC_KPNM_KPAM(t *testing.T) {
	tc := []struct {
		name        string
		hdIDs       []int
		seq         string
		keypadMode0 KeypadMode
		keypadMode1 KeypadMode
	}{
		{"DEC KPNM application mode", []int{esc_deckpnm, esc_deckpam}, "\x1b>\x1b=", KeypadMode_Normal, KeypadMode_Application},
		{"DEC KPAM numeric mode", []int{esc_deckpam, esc_deckpnm}, "\x1b=\x1b>", KeypadMode_Application, KeypadMode_Normal},
	}

	p := NewParser()
	emu := NewEmulator3(8, 4, 4) // this is the pre-condidtion for the test case.

	for _, v := range tc {

		hds := make([]*Handler, 0, 16)
		hds = p.processStream(v.seq, hds)

		if len(hds) == 0 {
			t.Errorf("%s got zero handlers.", v.name)
		}

		// call the handler
		for j, hd := range hds {
			hd.handle(emu)
			if hd.id != v.hdIDs[j] { // validate the control sequences id
				t.Errorf("%s: seq=%q expect %s, got %s\n",
					v.name, v.seq, strHandlerID[v.hdIDs[j]], strHandlerID[hd.id])
			}
			got := emu.keypadMode
			switch j {
			case 0:
				if got != v.keypadMode0 {
					t.Errorf("%s seq=%q keypadmode expect %d, got %d\n", v.name, v.seq, v.keypadMode0, got)
				}
			case 1:
				if got != v.keypadMode1 {
					t.Errorf("%s seq=%q keypadmode expect %d, got %d\n", v.name, v.seq, v.keypadMode1, got)
				}
			}
		}
	}
}

func TestHandle_ESCSpaceHash_Unhandled(t *testing.T) {
	tc := []struct {
		name  string
		seq   string
		state int
		msg   string
	}{
		{"esc space F", "\x1B F", InputState_Normal, "S7C1T: Send 7-bit controls"},
		{"esc space G", "\x1B G", InputState_Normal, "S8C1T: Send 8-bit controls"},
		{"esc space L", "\x1B L", InputState_Normal, "Set ANSI conformance level 1"},
		{"esc space M", "\x1B M", InputState_Normal, "Set ANSI conformance level 2"},
		{"esc space N", "\x1B N", InputState_Normal, "Set ANSI conformance level 3"},
		{"esc space default", "\x1B O", InputState_Normal, "Unhandled input:"}, // esc space unhandle
		{"esc hash 3", "\x1B#3", InputState_Normal, "DECDHL: Double-height, top half."},
		{"esc hash 4", "\x1B#4", InputState_Normal, "DECDHL: Double-height, bottom half."},
		{"esc hash 5", "\x1B#5", InputState_Normal, "DECSWL: Single-width line."},
		{"esc hash 6", "\x1B#6", InputState_Normal, "DECDWL: Double-width line."},
		{"esc hash default", "\x1B#9", InputState_Normal, "Unhandled input:"},   // esc hash unhandle
		{"csi quote default", "\x1B['o", InputState_Normal, "Unhandled input:"}, // csi quote unhandle
		{"csi space default", "\x1B[ o", InputState_Normal, "Unhandled input:"}, // csi space unhandle
	}

	p := NewParser()
	var place strings.Builder // all the message is output to herer
	p.logU.SetOutput(&place)

	for _, v := range tc {
		place.Reset()

		hds := make([]*Handler, 0, 16)
		p.processStream(v.seq, hds)

		state := p.getState()
		if state != v.state || !strings.Contains(place.String(), v.msg) {
			t.Errorf("%s seq=%q\n", v.name, v.seq)
			t.Errorf("state expect %s, got %s\n", strInputState[v.state], strInputState[state])
			t.Errorf("msg expect %s, got %s\n", v.msg, place.String())
		}
	}
}
