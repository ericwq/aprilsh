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

func TestHandle_SCOSC_SCORC(t *testing.T) {
	tc := []struct {
		name       string
		seq        string
		hdIDs      []int
		posY, posX int
		set        bool
		logMsg     string
	}{
		{
			"move cursor, SCOSC, check", "\x1B[22;33H\x1B[s",
			[]int{csi_cup, csi_scosc},
			22 - 1, 33 - 1, true, "",
		},
		{
			"move cursor, SCOSC, move cursor, SCORC, check", "\x1B[33;44H\x1B[s\x1B[42;35H\x1B[u",
			[]int{csi_cup, csi_scosc, csi_cup, csi_scorc},
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
			got := strings.Contains(place.String(), v.logMsg)
			if !got {
				t.Errorf("%s:\t %q expect %s, got %s\n", v.name, v.seq, v.logMsg, place.String())
			}
		}
	}
}

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

func TestHandle_DECSET_DECRST_1049(t *testing.T) {
	name := "DECSET/RST 1049"
	// move cursor to 23,13
	// DECSET 1049 enable altenate screen buffer
	// move cursor to 33,23
	// DECRST 1049 disable normal screen buffer (false)
	// DECRST 1049 set normal screen buffer (again for fast return)
	seq := "\x1B[24;14H\x1B[?1049h\x1B[34;24H\x1B[?1049l\x1B[?1049l"
	hdIDs := []int{csi_cup, csi_decset, csi_cup, csi_decrst, csi_decrst}

	p := NewParser()
	emu := NewEmulator3(80, 40, 500)
	var place strings.Builder
	emu.logI.SetOutput(&place)
	emu.logT.SetOutput(&place)

	// parse the control sequence
	hds := make([]*Handler, 0, 16)
	hds = p.processStream(seq, hds)

	if len(hds) != len(hdIDs) {
		t.Errorf("%s got zero handlers.", name)
	}

	// handle the instruction
	for j, hd := range hds {
		hd.handle(emu)
		if hd.id != hdIDs[j] { // validate the control sequences id
			t.Errorf("%s:\t %q expect %s, got %s\n", name, seq, strHandlerID[hdIDs[j]], strHandlerID[hd.id])
		}

		switch j {
		case 0, 3:
			wantY := 23
			wantX := 13

			gotY := emu.posY
			gotX := emu.posX

			if gotX != wantX || gotY != wantY {
				t.Errorf("%s:\t %q expect [%d,%d], got [%d,%d]\n", name, seq, wantY, wantX, gotY, gotX)
			}

			want := false
			got := emu.altScreenBufferMode

			if got != want {
				t.Errorf("%s:\t %q expect %t, got %t\n", name, seq, want, got)
			}
		case 1:
			want := true
			got := emu.altScreenBufferMode

			if got != want {
				t.Errorf("%s:\t %q expect %t, got %t\n", name, seq, want, got)
			}
		case 2:
			wantY := 33
			wantX := 23

			gotY := emu.posY
			gotX := emu.posX

			if gotX != wantX || gotY != wantY {
				t.Errorf("%s:\t %q expect [%d,%d], got [%d,%d].\n", name, seq, wantY, wantX, gotY, gotX)
			}
		case 4:
			want := false
			got := emu.altScreenBufferMode

			if got != want {
				t.Errorf("%s:\t %q expect %t, got %t\n", name, seq, want, got)
			}

			logMsg := "Asked to restore cursor (DECRC) but it has not been saved."
			if !strings.Contains(place.String(), logMsg) {
				t.Errorf("%s seq=%q expect %q, got %q\n", name, seq, logMsg, place.String())
			}
		}
		// reset the output buffer
		place.Reset()
	}
}

func TestHandle_DECSET_DECRST_3(t *testing.T) {
	tc := []struct {
		name  string
		seq   string
		hdIDs []int
		mode  ColMode
	}{
		{"change to column Mode    132", "\x1B[?3h", []int{csi_decset}, ColMode_C132},
		{"change to column Mode     80", "\x1B[?3l", []int{csi_decrst}, ColMode_C80},
		{"change to column Mode repeat", "\x1B[?3h\x1B[?3h", []int{csi_decset, csi_decset}, ColMode_C132},
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

		// handle the control sequence
		for j, hd := range hds {
			hd.handle(emu)
			if hd.id != v.hdIDs[j] { // validate the control sequences id
				t.Errorf("%s:\t %q expect %s, got %s\n", v.name, v.seq, strHandlerID[v.hdIDs[j]], strHandlerID[hd.id])
			}
		}

		got := emu.colMode
		if got != v.mode {
			t.Errorf("%s:\t %q expect %d, got %d\n", v.name, v.seq, v.mode, got)
		}
	}
}

func TestHandle_DECSET_DECRST_2(t *testing.T) {
	tc := []struct {
		name                string
		seq                 string
		hdIDs               []int
		compatLevel         CompatibilityLevel
		isResetCharsetState bool
	}{
		{"DECSET 2", "\x1B[?2h", []int{csi_decset}, CompatLevel_VT400, true},
		{"DECRST 2", "\x1B[?2l", []int{csi_decrst}, CompatLevel_VT52, true},
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

		// handle the control sequence
		for j, hd := range hds {
			hd.handle(emu)
			if hd.id != v.hdIDs[j] { // validate the control sequences id
				t.Errorf("%s:\t %q expect %s, got %s\n", v.name, v.seq, strHandlerID[v.hdIDs[j]], strHandlerID[hd.id])
			}
		}

		// validate the result
		gotCL := emu.compatLevel
		gotRCS := isResetCharsetState(emu.charsetState)
		if v.isResetCharsetState != gotRCS || v.compatLevel != gotCL {
			t.Errorf("%s seq=%q expect reset CharsetState and compatbility level (%t,%d), got(%t,%d)",
				v.name, v.seq, v.isResetCharsetState, v.compatLevel, gotRCS, gotCL)
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

func TestHandle_DECSET_DECRST_67(t *testing.T) {
	tc := []struct {
		name         string
		seq          string
		hdIDs        []int
		bkspSendsDel bool
	}{
		{"enable DECBKM—Backarrow Key Mode", "\x1B[?67h", []int{csi_decset}, false},
		{"disable DECBKM—Backarrow Key Mode", "\x1B[?67l", []int{csi_decrst}, true},
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

		if len(hds) != 1 {
			t.Errorf("%s got %d handlers.", v.name, len(hds))
		}

		// handle the control sequence
		for j, hd := range hds {
			hd.handle(emu)
			if hd.id != v.hdIDs[j] { // validate the control sequences id
				t.Errorf("%s:\t %q expect %s, got %s\n", v.name, v.seq, strHandlerID[v.hdIDs[j]], strHandlerID[hd.id])
			}
		}

		got := emu.bkspSendsDel
		if got != v.bkspSendsDel {
			t.Errorf("%s:\t %q expect %t,got %t\n", v.name, v.seq, v.bkspSendsDel, got)
		}
	}
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
			[]int{csi_decset, csi_decslrm},
			3, 70, 0, 0,
		},
		{
			"set left right margin, missing right parameter",
			"\x1B[?69h\x1B[1s",
			[]int{csi_decset, csi_decslrm},
			0, 80, 0, 0,
		},
		{
			"set left right margin, left parameter is zero",
			"\x1B[?69h\x1B[0s",
			[]int{csi_decset, csi_decslrm},
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
			[]int{csi_decrst, csi_decslrm},
			"", 0, 0, 0, 0,
		},
		{
			"DECLRMM enable, outof range", "\x1B[?69h\x1B[4;89s",
			[]int{csi_decset, csi_decslrm},
			"Illegal arguments to SetLeftRightMargins:", 0, 0, 0, 0,
		},
		{
			"DECLRMM OriginMode_ScrollingRegion, enable", "\x1B[?6h\x1B[?69h\x1B[4;69s", // DECLRMM: Set Left and Right Margins
			[]int{csi_decset, csi_decset, csi_decslrm},
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

type (
	ANSImode uint
	DECmode  uint
)

const (
	t_keyboardLocked ANSImode = iota
	t_insertMode
	t_localEcho
	t_autoNewlineMode
)

const (
	t_cursorKeyMode DECmode = iota
	t_reverseVideo
	t_originMode
	t_autoWrapMode
	t_showCursorMode
	t_focusEventMode
	t_altScrollMode
	t_altSendsEscape
	t_bracketedPasteMode
)

func TestHandle_SM_RM(t *testing.T) {
	tc := []struct {
		name  string
		seq   string
		which ANSImode
		hdIDs []int
		want  bool
	}{
		{"SM: keyboardLocked ", "\x1B[2l\x1B[2h", t_keyboardLocked, []int{csi_rm, csi_sm}, true},
		{"SM: insertMode     ", "\x1B[4l\x1B[4h", t_insertMode, []int{csi_rm, csi_sm}, true},
		{"SM: localEcho      ", "\x1B[12l\x1B[12h", t_localEcho, []int{csi_rm, csi_sm}, false},
		{"SM: autoNewlineMode", "\x1B[20l\x1B[20h", t_autoNewlineMode, []int{csi_rm, csi_sm}, true},

		{"RM: keyboardLocked ", "\x1B[2h\x1B[2l", t_keyboardLocked, []int{csi_sm, csi_rm}, false},
		{"RM: insertMode     ", "\x1B[4h\x1B[4l", t_insertMode, []int{csi_sm, csi_rm}, false},
		{"RM: localEcho      ", "\x1B[12h\x1B[12l", t_localEcho, []int{csi_sm, csi_rm}, true},
		{"RM: autoNewlineMode", "\x1B[20h\x1B[20l", t_autoNewlineMode, []int{csi_sm, csi_rm}, false},
	}

	p := NewParser()
	emu := NewEmulator3(80, 40, 500)
	var place strings.Builder
	emu.logI.SetOutput(&place)
	emu.logT.SetOutput(&place)

	for _, v := range tc {
		t.Run(v.name, func(t *testing.T) {
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
					t.Errorf("%s: seq=%q expect %s, got %s\n", v.name, v.seq, strHandlerID[v.hdIDs[j]], strHandlerID[hd.id])
				}
			}

			if v.want != t_getANSImode(emu, v.which) {
				t.Errorf("%s: seq=%q expect %t, got %t\n", v.name, v.seq, v.want, t_getANSImode(emu, v.which))
			}
		})
	}
}

func t_getANSImode(emu *emulator, which ANSImode) bool {
	switch which {
	case t_keyboardLocked:
		return emu.keyboardLocked
	case t_insertMode:
		return emu.insertMode
	case t_localEcho:
		return emu.localEcho
	case t_autoNewlineMode:
		return emu.autoNewlineMode
	}
	return false
}

// func t_resetANSImode(emu *emulator, which ANSImode, value bool) {
// 	switch which {
// 	case t_keyboardLocked:
// 		emu.keyboardLocked = value
// 	case t_insertMode:
// 		emu.insertMode = value
// 	case t_localEcho:
// 		emu.localEcho = value
// 	case t_autoNewlineMode:
// 		emu.autoNewlineMode = value
// 	}
// }

func TestHandle_SM_RM_Unknow(t *testing.T) {
	tc := []struct {
		name string
		seq  string
		want string
	}{
		{"CSI SM unknow", "\x1B[21h", "Ignored bogus set mode"},
		{"CSI RM unknow", "\x1B[33l", "Ignored bogus reset mode"},
	}

	p := NewParser()
	emu := NewEmulator3(80, 40, 500)
	var place strings.Builder
	emu.logI.SetOutput(&place)
	emu.logW.SetOutput(&place)

	for _, v := range tc {
		t.Run(v.name, func(t *testing.T) {
			// process control sequence
			hds := make([]*Handler, 0, 16)
			hds = p.processStream(v.seq, hds)

			if len(hds) == 0 {
				t.Errorf("%s got zero handlers.", v.name)
			}

			// handle the control sequence
			for _, hd := range hds {
				hd.handle(emu)
			}

			if !strings.Contains(place.String(), v.want) {
				t.Errorf("%s: %q\t expect %q, got %q\n", v.name, v.seq, v.want, place.String())
			}
		})
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
			[]int{csi_sm, csi_decset, csi_decrst, csi_decset, csi_decstbm, csi_decstr},
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

func TestHandle_DECSET_DECRST_BOOL(t *testing.T) {
	tc := []struct {
		name  string
		seq   string
		which DECmode
		hdIDs []int
		want  bool
	}{
		{"DECSET: reverseVideo", "\x1B[?5l\x1B[?5h", t_reverseVideo, []int{csi_decrst, csi_decset}, true},
		{"DECSET: autoWrapMode", "\x1B[?7l\x1B[?7h", t_autoWrapMode, []int{csi_decrst, csi_decset}, true},
		{"DECSET: CursorVisible", "\x1B[?25l\x1B[?25h", t_showCursorMode, []int{csi_decrst, csi_decset}, true},
		{"DECSET: focusEventMode", "\x1B[?1004l\x1B[?1004h", t_focusEventMode, []int{csi_decrst, csi_decset}, true},
		{"DECSET: MouseAlternateScroll", "\x1B[?1007l\x1B[?1007h", t_altScrollMode, []int{csi_decrst, csi_decset}, true},
		{"DECSET: altSendsEscape", "\x1B[?1036l\x1B[?1036h", t_altSendsEscape, []int{csi_decrst, csi_decset}, true},
		{"DECSET: altSendsEscape", "\x1B[?1039l\x1B[?1039h", t_altSendsEscape, []int{csi_decrst, csi_decset}, true},
		{"DECSET: BracketedPaste", "\x1B[?2004l\x1B[?2004h", t_bracketedPasteMode, []int{csi_decrst, csi_decset}, true},

		{"DECRST: ReverseVideo", "\x1B[?5h\x1B[?5l", t_reverseVideo, []int{csi_decset, csi_decrst}, false},
		{"DECRST: AutoWrapMode", "\x1B[?7h\x1B[?7l", t_autoWrapMode, []int{csi_decset, csi_decrst}, false},
		{"DECRST: CursorVisible", "\x1B[?25h\x1B[?25l", t_showCursorMode, []int{csi_decset, csi_decrst}, false},
		{"DECRST: focusEventMode", "\x1B[?1004h\x1B[?1004l", t_focusEventMode, []int{csi_decset, csi_decrst}, false},
		{"DECRST: MouseAlternateScroll", "\x1B[?1007h\x1B[?1007l", t_altScrollMode, []int{csi_decset, csi_decrst}, false},
		{"DECRST: altSendsEscape", "\x1B[?1036h\x1B[?1036l", t_altSendsEscape, []int{csi_decset, csi_decrst}, false},
		{"DECRST: altSendsEscape", "\x1B[?1039h\x1B[?1039l", t_altSendsEscape, []int{csi_decset, csi_decrst}, false},
		{"DECRST: BracketedPaste", "\x1B[?2004h\x1B[?2004l", t_bracketedPasteMode, []int{csi_decset, csi_decrst}, false},
	}

	p := NewParser()
	emu := NewEmulator3(80, 40, 500)
	var place strings.Builder
	emu.logI.SetOutput(&place)
	emu.logT.SetOutput(&place)

	for _, v := range tc {
		t.Run(v.name, func(t *testing.T) {
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
					t.Errorf("%s:\t %q expect %s, got %s\n", v.name, v.seq, strHandlerID[v.hdIDs[j]], strHandlerID[hd.id])
				}
			}

			if v.want != t_getDECmode(emu, v.which) {
				t.Errorf("%s: %q\t expect %t, got %t\n", v.name, v.seq, v.want, t_getDECmode(emu, v.which))
			}
		})
	}
}

func t_getDECmode(emu *emulator, which DECmode) bool {
	switch which {
	case t_reverseVideo:
		return emu.reverseVideo
	case t_autoWrapMode:
		return emu.autoWrapMode
	case t_showCursorMode:
		return emu.showCursorMode
	case t_focusEventMode:
		return emu.mouseTrk.focusEventMode
	case t_altScrollMode:
		return emu.altScrollMode
	case t_altSendsEscape:
		return emu.altSendsEscape
	case t_bracketedPasteMode:
		return emu.bracketedPasteMode
	}
	return false
}

// func t_resetDECmode(ds *emulator, which DECmode, value bool) {
// 	switch which {
// 	case t_reverseVideo:
// 		ds.reverseVideo = value
// 	case t_autoWrapMode:
// 		ds.autoWrapMode = value
// 	case t_showCursorMode:
// 		ds.showCursorMode = value
// 	case t_focusEventMode:
// 		ds.mouseTrk.focusEventMode = value
// 	case t_altScrollMode:
// 		ds.altScrollMode = value
// 	case t_altSendsEscape:
// 		ds.altSendsEscape = value
// 	case t_bracketedPasteMode:
// 		ds.bracketedPasteMode = value
// 	}
// }

func TestHandle_DECSET_DECRST_Log(t *testing.T) {
	tc := []struct {
		name string
		seq  string
		hdID int
		want string
	}{
		{"DECSET:   4", "\x1B[?4h", csi_decset, "DECSCLM: Set smooth scroll"},
		{"DECSET:   8", "\x1B[?8h", csi_decset, "DECARM: Set auto-repeat mode"},
		{"DECSET:  12", "\x1B[?12h", csi_decset, "Start blinking cursor"},
		{"DECSET:1001", "\x1B[?1001h", csi_decset, "Set VT200 Highlight Mouse mode"},
		{"DECSET:unknow", "\x1B[?2022h", csi_decset, "set priv mode"},

		{"DECRST:   4", "\x1B[?4l", csi_decrst, "DECSCLM: Set jump scroll"},
		{"DECRST:   8", "\x1B[?8l", csi_decrst, "DECARM: Reset auto-repeat mode"},
		{"DECRST:  12", "\x1B[?12l", csi_decrst, "Stop blinking cursor"},
		{"DECRST:1001", "\x1B[?1001l", csi_decrst, "Reset VT200 Highlight Mouse mode"},
		{"DECRST:unknow", "\x1B[?2022l", csi_decrst, "reset priv mode"},
	}

	p := NewParser()
	emu := NewEmulator3(80, 40, 500)
	var place strings.Builder
	emu.logU.SetOutput(&place)
	emu.logT.SetOutput(&place)

	for _, v := range tc {
		t.Run(v.name, func(t *testing.T) {
			// process control sequence
			hds := make([]*Handler, 0, 16)
			hds = p.processStream(v.seq, hds)

			if len(hds) == 0 {
				t.Errorf("%s got zero handlers.", v.name)
			}

			// handle the control sequence
			for _, hd := range hds {
				hd.handle(emu)
				if hd.id != v.hdID { // validate the control sequences id
					t.Errorf("%s:\t %q expect %s, got %s\n", v.name, v.seq, strHandlerID[v.hdID], strHandlerID[hd.id])
				}
			}

			if !strings.Contains(place.String(), v.want) {
				t.Errorf("%s: %q\t expect %q, got %q\n", v.name, v.seq, v.want, place.String())
			}
		})
	}
}

func TestHandle_DECSET_DECRST_6(t *testing.T) {
	tc := []struct {
		name       string
		seq        string
		hdIDs      []int
		originMode OriginMode
	}{
		{"DECSET:   6", "\x1B[?6l\x1B[?6h", []int{csi_decrst, csi_decset}, OriginMode_ScrollingRegion},
		{"DECRST:   6", "\x1B[?6h\x1B[?6l", []int{csi_decset, csi_decrst}, OriginMode_Absolute},
	}

	p := NewParser()
	emu := NewEmulator3(80, 40, 500)
	var place strings.Builder
	emu.logI.SetOutput(&place)
	emu.logT.SetOutput(&place)

	for _, v := range tc {
		t.Run(v.name, func(t *testing.T) {
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
					t.Errorf("%s: seq=%q expect %s, got %s\n", v.name, v.seq, strHandlerID[v.hdIDs[j]], strHandlerID[hd.id])
				}
			}

			got := emu.originMode
			if got != v.originMode {
				t.Errorf("%s: seq=%q expect %d, got %d\n", v.name, v.seq, v.originMode, got)
			}
		})
	}
}

func TestHandle_DECSET_DECRST_1(t *testing.T) {
	tc := []struct {
		name          string
		seq           string
		hdIDs         []int
		cursorKeyMode CursorKeyMode
	}{
		{"DECSET:   1", "\x1B[?1l\x1B[?1h", []int{csi_decrst, csi_decset}, CursorKeyMode_Application},
		{"DECRST:   1", "\x1B[?1h\x1B[?1l", []int{csi_decset, csi_decrst}, CursorKeyMode_ANSI},
	}

	p := NewParser()
	emu := NewEmulator3(80, 40, 500)
	var place strings.Builder
	emu.logI.SetOutput(&place)
	emu.logT.SetOutput(&place)

	for _, v := range tc {
		t.Run(v.name, func(t *testing.T) {
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
					t.Errorf("%s: seq=%q expect %s, got %s\n", v.name, v.seq, strHandlerID[v.hdIDs[j]], strHandlerID[hd.id])
				}
			}

			got := emu.cursorKeyMode
			if got != v.cursorKeyMode {
				t.Errorf("%s: %q seq=expect %d, got %d\n", v.name, v.seq, v.cursorKeyMode, got)
			}
		})
	}
}

func TestHandle_DECSET_DECRST_MouseTrackingMode(t *testing.T) {
	tc := []struct {
		name  string
		seq   string
		hdIDs []int
		want  MouseTrackingMode
	}{
		{"DECSET:   9", "\x1B[?9l\x1B[?9h", []int{csi_decrst, csi_decset}, MouseTrackingMode_X10_Compat},
		{"DECSET:1000", "\x1B[?1000l\x1B[?1000h", []int{csi_decrst, csi_decset}, MouseTrackingMode_VT200},
		{"DECSET:1002", "\x1B[?1002l\x1B[?1002h", []int{csi_decrst, csi_decset}, MouseTrackingMode_VT200_ButtonEvent},
		{"DECSET:1003", "\x1B[?1003l\x1B[?1003h", []int{csi_decrst, csi_decset}, MouseTrackingMode_VT200_AnyEvent},

		{"DECRST:   9", "\x1B[?9h\x1B[?9l", []int{csi_decset, csi_decrst}, MouseTrackingMode_Disable},
		{"DECRST:1000", "\x1B[?1000h\x1B[?1000l", []int{csi_decset, csi_decrst}, MouseTrackingMode_Disable},
		{"DECRST:1002", "\x1B[?1002h\x1B[?1002l", []int{csi_decset, csi_decrst}, MouseTrackingMode_Disable},
		{"DECRST:1003", "\x1B[?1003h\x1B[?1003l", []int{csi_decset, csi_decrst}, MouseTrackingMode_Disable},
	}

	p := NewParser()
	emu := NewEmulator3(80, 40, 500)
	var place strings.Builder
	emu.logI.SetOutput(&place)
	emu.logT.SetOutput(&place)

	for _, v := range tc {
		t.Run(v.name, func(t *testing.T) {
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
					t.Errorf("%s:\t %q expect %s, got %s\n", v.name, v.seq, strHandlerID[v.hdIDs[j]], strHandlerID[hd.id])
				}
			}

			got := emu.mouseTrk.mode
			if got != v.want {
				t.Errorf("%s: %q\t expect %d, got %d\n", v.name, v.seq, v.want, got)
			}
		})
	}
}

func TestHandle_DECSET_DECRST_MouseTrackingEnc(t *testing.T) {
	tc := []struct {
		name  string
		seq   string
		hdIDs []int
		want  MouseTrackingEnc
	}{
		{"DECSET:1005", "\x1B[?1005l\x1B[?1005h", []int{csi_decrst, csi_decset}, MouseTrackingEnc_UTF8},
		{"DECSET:1006", "\x1B[?1006l\x1B[?1006h", []int{csi_decrst, csi_decset}, MouseTrackingEnc_SGR},
		{"DECSET:1015", "\x1B[?1015l\x1B[?1015h", []int{csi_decrst, csi_decset}, MouseTrackingEnc_URXVT},

		{"DECRST:1005", "\x1B[?1005h\x1B[?1005l", []int{csi_decset, csi_decrst}, MouseTrackingEnc_Default},
		{"DECRST:1006", "\x1B[?1006h\x1B[?1006l", []int{csi_decset, csi_decrst}, MouseTrackingEnc_Default},
		{"DECRST:1015", "\x1B[?1015h\x1B[?1015l", []int{csi_decset, csi_decrst}, MouseTrackingEnc_Default},
	}

	p := NewParser()
	emu := NewEmulator3(80, 40, 500)
	var place strings.Builder
	emu.logI.SetOutput(&place)
	emu.logT.SetOutput(&place)

	for _, v := range tc {
		t.Run(v.name, func(t *testing.T) {
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
					t.Errorf("%s:\t %q expect %s, got %s\n", v.name, v.seq, strHandlerID[v.hdIDs[j]], strHandlerID[hd.id])
				}
			}

			got := emu.mouseTrk.enc
			if got != v.want {
				t.Errorf("%s: %q\t expect %d, got %d\n", v.name, v.seq, v.want, got)
			}
		})
	}
}

func TestHandle_DECSET_DECRST_47_1047(t *testing.T) {
	tc := []struct {
		name      string
		seq       string
		hdIDs     []int
		setMode   bool
		unsetMode bool
	}{
		{"DECSET/RST 47", "\x1B[?47h\x1B[?47l", []int{csi_decset, csi_decrst}, true, false},
		{"DECSET/RST 1047", "\x1B[?1047h\x1B[?1047l", []int{csi_decset, csi_decrst}, true, false},
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

		if len(hds) != 2 {
			t.Errorf("%s got zero handlers.", v.name)
		}

		// handle the control sequence
		for j, hd := range hds {
			hd.handle(emu)
			if hd.id != v.hdIDs[j] { // validate the control sequences id
				t.Errorf("%s: seq=%q expect %s, got %s\n", v.name, v.seq, strHandlerID[v.hdIDs[j]], strHandlerID[hd.id])
			}
			got := emu.altScreenBufferMode
			switch j {
			case 0:
				if got != v.setMode {
					t.Errorf("%s: seq=%q expect %t, got %t\n", v.name, v.seq, true, got)
				}
			case 1:
				if got != v.unsetMode {
					t.Errorf("%s: seq=%q expect %t, got %t\n", v.name, v.seq, false, got)
				}
			}
		}
	}
}

func TestHandle_DECSET_DECRST_69(t *testing.T) {
	tc := []struct {
		name            string
		seq             string
		hdIDs           []int
		horizMarginMode bool
	}{
		{"DECSET/DECRST 69 combining", "\x1B[?69h\x1B[?69l", []int{csi_decset, csi_decrst}, true},
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
				t.Errorf("%s: seq=%q expect %s, got %s\n", v.name, v.seq, strHandlerID[v.hdIDs[j]], strHandlerID[hd.id])
			}
			got := emu.horizMarginMode
			switch j {
			case 0:
				if got != true {
					t.Errorf("%s:\t %q expect %t, got %t\n", v.name, v.seq, true, got)
				}
			case 1:
				if got != false {
					t.Errorf("%s:\t %q expect %t, got %t\n", v.name, v.seq, false, got)
				}
			}
		}
	}
}

func TestHandle_CR_LF_VT_FF(t *testing.T) {
	tc := []struct {
		name  string
		hdIDs []int
		wantX int
		wantY int
		seq   string
	}{
		{"CR 1 ", []int{csi_cup, c0_cr}, 0, 2, "\x1B[3;2H\x0D"},
		{"CR 2 ", []int{csi_cup, c0_cr}, 0, 4, "\x1B[5;10H\x0D"},
		{"LF   ", []int{csi_cup, c0_lf}, 1, 3, "\x1B[3;2H\x0C"},
		{"VT   ", []int{csi_cup, c0_lf}, 2, 4, "\x1B[4;3H\x0B"},
		{"FF   ", []int{csi_cup, c0_lf}, 3, 5, "\x1B[5;4H\x0C"},
		{"ESC D", []int{csi_cup, c0_lf}, 4, 6, "\x1B[6;5H\x1BD"},
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

		if len(hds) != 2 {
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

func TestHandle_CSI_BS_FF_VT_CR_TAB(t *testing.T) {
	tc := []struct {
		name         string
		hdIDs        []int
		seq          string
		wantY, wantX int
	}{
		// call CUP first to set the start position
		{"CSI backspace number    ", []int{csi_cup, csi_cup}, "\x1B[1;1H\x1B[23;12\bH", 22, 0},     // undo last character in CSI sequence
		{"CSI backspace semicolon ", []int{csi_cup, csi_cup}, "\x1B[1;1H\x1B[23;\b;12H", 22, 11},   // undo last character in CSI sequence
		{"cursor down 1+3 rows VT ", []int{csi_cup, c0_lf, csi_cud}, "\x1B[9;10H\x1B[3\vB", 12, 9}, //(8,9)->(9.9)->(12,9)
		{"cursor down 1+3 rows FF ", []int{csi_cup, c0_lf, csi_cud}, "\x1B[9;10H\x1B[\f3B", 12, 9},
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

func TestHandle_CUU_CUD_CUF_CUB_CUP(t *testing.T) {
	tc := []struct {
		name  string
		hdIDs []int
		wantX int
		wantY int
		seq   string
	}{
		// call CUP first to set the start position
		{"CSI Ps A  ", []int{csi_cup, csi_cuu}, 10, 14, "\x1B[21;11H\x1B[6A"},
		{"CSI Ps B  ", []int{csi_cup, csi_cud}, 10, 13, "\x1B[11;11H\x1B[3B"},
		{"CSI Ps C  ", []int{csi_cup, csi_cuf}, 12, 10, "\x1B[11;11H\x1B[2C"},
		{"CSI Ps D  ", []int{csi_cup, csi_cub}, 12, 10, "\x1B[11;21H\x1B[8D"},
		{"BS        ", []int{csi_cup, csi_cub}, 11, 12, "\x1B[13;13H\x08"}, // \x08 calls CUB
		{"CUB       ", []int{csi_cup, csi_cub}, 11, 12, "\x1B[13;13H\x1B[1D"},
		{"BS agin   ", []int{csi_cup, csi_cub}, 10, 12, "\x1B[13;12H\x08"}, // \x08 calls CUB
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

		if len(hds) != 2 {
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
		// the terminal size is 8x4 [colxrow]
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

func isTabStop(emu *emulator, x int) bool {
	data := emu.tabStops

	i := sort.Search(len(data), func(i int) bool { return data[i] >= x })
	if i < len(data) && data[i] == x {
		return true
		// x is present at data[i]
	}
	return false
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
		{"CHT to the next 4 tab", []int{csi_cup, csi_cht}, 32, "\x1B[21;3H\x1B[4I"}, // move to the next N tab stop
		{"CHT to the right edge", []int{csi_cup, csi_cht}, 79, "\x1B[21;60H\x1B[4I"},
		{"CBT back to the left edge", []int{csi_cup, csi_cbt}, 0, "\x1B[21;3H\x1B[3Z"},
	}

	p := NewParser()
	emu := NewEmulator3(80, 40, 5)
	var place strings.Builder
	emu.logI.SetOutput(&place)
	emu.logT.SetOutput(&place)

	for _, v := range tc {

		hds := make([]*Handler, 0, 16)
		hds = p.processStream(v.seq, hds)

		if len(hds) != 2 {
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

	for i, v := range tc {
		emu.resetTerminal()

		hds := make([]*Handler, 0, 16)
		hds = p.processStream(v.seq, hds)

		if len(hds) == 0 {
			t.Errorf("%s got %d handlers.", v.name, len(hds))
		}

		// handle the control sequence
		for _, hd := range hds {
			hd.handle(emu)
			if i == 2 {
				t.Logf("%s [frame] scrollHead=%d historyRows=%d [emulator] posY=%d\n",
					v.name, emu.cf.scrollHead, emu.cf.historyRows, emu.posY)
			}
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
