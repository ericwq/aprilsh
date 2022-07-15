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
	"reflect"
	"strings"
	"testing"
)

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

func TestHandle_privSM_privRM_67(t *testing.T) {
	tc := []struct {
		name         string
		seq          string
		hdIDs        []int
		bkspSendsDel bool
	}{
		{"enable DECBKM—Backarrow Key Mode", "\x1B[?67h", []int{csi_privSM}, false},
		{"disable DECBKM—Backarrow Key Mode", "\x1B[?67l", []int{csi_privRM}, true},
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

func TestHandle_privSM_privRM_BOOL(t *testing.T) {
	tc := []struct {
		name  string
		seq   string
		which DECmode
		hdIDs []int
		want  bool
	}{
		{"privSM: reverseVideo", "\x1B[?5l\x1B[?5h", t_reverseVideo, []int{csi_privRM, csi_privSM}, true},
		{"privSM: autoWrapMode", "\x1B[?7l\x1B[?7h", t_autoWrapMode, []int{csi_privRM, csi_privSM}, true},
		{"privSM: CursorVisible", "\x1B[?25l\x1B[?25h", t_showCursorMode, []int{csi_privRM, csi_privSM}, true},
		{"privSM: focusEventMode", "\x1B[?1004l\x1B[?1004h", t_focusEventMode, []int{csi_privRM, csi_privSM}, true},
		{"privSM: MouseAlternateScroll", "\x1B[?1007l\x1B[?1007h", t_altScrollMode, []int{csi_privRM, csi_privSM}, true},
		{"privSM: altSendsEscape", "\x1B[?1036l\x1B[?1036h", t_altSendsEscape, []int{csi_privRM, csi_privSM}, true},
		{"privSM: altSendsEscape", "\x1B[?1039l\x1B[?1039h", t_altSendsEscape, []int{csi_privRM, csi_privSM}, true},
		{"privSM: BracketedPaste", "\x1B[?2004l\x1B[?2004h", t_bracketedPasteMode, []int{csi_privRM, csi_privSM}, true},

		{"privRM: ReverseVideo", "\x1B[?5h\x1B[?5l", t_reverseVideo, []int{csi_privSM, csi_privRM}, false},
		{"privRM: AutoWrapMode", "\x1B[?7h\x1B[?7l", t_autoWrapMode, []int{csi_privSM, csi_privRM}, false},
		{"privRM: CursorVisible", "\x1B[?25h\x1B[?25l", t_showCursorMode, []int{csi_privSM, csi_privRM}, false},
		{"privRM: focusEventMode", "\x1B[?1004h\x1B[?1004l", t_focusEventMode, []int{csi_privSM, csi_privRM}, false},
		{"privRM: MouseAlternateScroll", "\x1B[?1007h\x1B[?1007l", t_altScrollMode, []int{csi_privSM, csi_privRM}, false},
		{"privRM: altSendsEscape", "\x1B[?1036h\x1B[?1036l", t_altSendsEscape, []int{csi_privSM, csi_privRM}, false},
		{"privRM: altSendsEscape", "\x1B[?1039h\x1B[?1039l", t_altSendsEscape, []int{csi_privSM, csi_privRM}, false},
		{"privRM: BracketedPaste", "\x1B[?2004h\x1B[?2004l", t_bracketedPasteMode, []int{csi_privSM, csi_privRM}, false},
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

func TestHandle_privSM_privRM_Log(t *testing.T) {
	tc := []struct {
		name string
		seq  string
		hdID int
		want string
	}{
		{"privSM:   4", "\x1B[?4h", csi_privSM, "DECSCLM: Set smooth scroll"},
		{"privSM:   8", "\x1B[?8h", csi_privSM, "DECARM: Set auto-repeat mode"},
		{"privSM:  12", "\x1B[?12h", csi_privSM, "Start blinking cursor"},
		{"privSM:1001", "\x1B[?1001h", csi_privSM, "Set VT200 Highlight Mouse mode"},
		{"privSM:unknow", "\x1B[?2022h", csi_privSM, "set priv mode"},

		{"privRM:   4", "\x1B[?4l", csi_privRM, "DECSCLM: Set jump scroll"},
		{"privRM:   8", "\x1B[?8l", csi_privRM, "DECARM: Reset auto-repeat mode"},
		{"privRM:  12", "\x1B[?12l", csi_privRM, "Stop blinking cursor"},
		{"privRM:1001", "\x1B[?1001l", csi_privRM, "Reset VT200 Highlight Mouse mode"},
		{"privRM:unknow", "\x1B[?2022l", csi_privRM, "reset priv mode"},
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

func TestHandle_privSM_privRM_6(t *testing.T) {
	tc := []struct {
		name       string
		seq        string
		hdIDs      []int
		originMode OriginMode
	}{
		{"privSM:   6", "\x1B[?6l\x1B[?6h", []int{csi_privRM, csi_privSM}, OriginMode_ScrollingRegion},
		{"privRM:   6", "\x1B[?6h\x1B[?6l", []int{csi_privSM, csi_privRM}, OriginMode_Absolute},
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

func TestHandle_privSM_privRM_1(t *testing.T) {
	tc := []struct {
		name          string
		seq           string
		hdIDs         []int
		cursorKeyMode CursorKeyMode
	}{
		{"privSM:   1", "\x1B[?1l\x1B[?1h", []int{csi_privRM, csi_privSM}, CursorKeyMode_Application},
		{"privRM:   1", "\x1B[?1h\x1B[?1l", []int{csi_privSM, csi_privRM}, CursorKeyMode_ANSI},
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

func TestHandle_privSM_privRM_MouseTrackingMode(t *testing.T) {
	tc := []struct {
		name  string
		seq   string
		hdIDs []int
		want  MouseTrackingMode
	}{
		{"privSM:   9", "\x1B[?9l\x1B[?9h", []int{csi_privRM, csi_privSM}, MouseTrackingMode_X10_Compat},
		{"privSM:1000", "\x1B[?1000l\x1B[?1000h", []int{csi_privRM, csi_privSM}, MouseTrackingMode_VT200},
		{"privSM:1002", "\x1B[?1002l\x1B[?1002h", []int{csi_privRM, csi_privSM}, MouseTrackingMode_VT200_ButtonEvent},
		{"privSM:1003", "\x1B[?1003l\x1B[?1003h", []int{csi_privRM, csi_privSM}, MouseTrackingMode_VT200_AnyEvent},

		{"privRM:   9", "\x1B[?9h\x1B[?9l", []int{csi_privSM, csi_privRM}, MouseTrackingMode_Disable},
		{"privRM:1000", "\x1B[?1000h\x1B[?1000l", []int{csi_privSM, csi_privRM}, MouseTrackingMode_Disable},
		{"privRM:1002", "\x1B[?1002h\x1B[?1002l", []int{csi_privSM, csi_privRM}, MouseTrackingMode_Disable},
		{"privRM:1003", "\x1B[?1003h\x1B[?1003l", []int{csi_privSM, csi_privRM}, MouseTrackingMode_Disable},
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

func TestHandle_privSM_privRM_MouseTrackingEnc(t *testing.T) {
	tc := []struct {
		name  string
		seq   string
		hdIDs []int
		want  MouseTrackingEnc
	}{
		{"privSM:1005", "\x1B[?1005l\x1B[?1005h", []int{csi_privRM, csi_privSM}, MouseTrackingEnc_UTF8},
		{"privSM:1006", "\x1B[?1006l\x1B[?1006h", []int{csi_privRM, csi_privSM}, MouseTrackingEnc_SGR},
		{"privSM:1015", "\x1B[?1015l\x1B[?1015h", []int{csi_privRM, csi_privSM}, MouseTrackingEnc_URXVT},

		{"privRM:1005", "\x1B[?1005h\x1B[?1005l", []int{csi_privSM, csi_privRM}, MouseTrackingEnc_Default},
		{"privRM:1006", "\x1B[?1006h\x1B[?1006l", []int{csi_privSM, csi_privRM}, MouseTrackingEnc_Default},
		{"privRM:1015", "\x1B[?1015h\x1B[?1015l", []int{csi_privSM, csi_privRM}, MouseTrackingEnc_Default},
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

func TestHandle_privSM_privRM_47_1047(t *testing.T) {
	tc := []struct {
		name      string
		seq       string
		hdIDs     []int
		setMode   bool
		unsetMode bool
	}{
		{"privSM/RST 47", "\x1B[?47h\x1B[?47l", []int{csi_privSM, csi_privRM}, true, false},
		{"privSM/RST 1047", "\x1B[?1047h\x1B[?1047l", []int{csi_privSM, csi_privRM}, true, false},
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

func TestHandle_privSM_privRM_69(t *testing.T) {
	tc := []struct {
		name            string
		seq             string
		hdIDs           []int
		horizMarginMode bool
	}{
		{"privSM/privRM 69 combining", "\x1B[?69h\x1B[?69l", []int{csi_privSM, csi_privRM}, true},
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

func TestHandle_privSM_privRM_1049(t *testing.T) {
	name := "privSM/RST 1049"
	// move cursor to 23,13
	// privSM 1049 enable altenate screen buffer
	// move cursor to 33,23
	// privRM 1049 disable normal screen buffer (false)
	// privRM 1049 set normal screen buffer (again for fast return)
	seq := "\x1B[24;14H\x1B[?1049h\x1B[34;24H\x1B[?1049l\x1B[?1049l"
	hdIDs := []int{csi_cup, csi_privSM, csi_cup, csi_privRM, csi_privRM}

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

func TestHandle_privSM_privRM_3(t *testing.T) {
	tc := []struct {
		name  string
		seq   string
		hdIDs []int
		mode  ColMode
	}{
		{"change to column Mode    132", "\x1B[?3h", []int{csi_privSM}, ColMode_C132},
		{"change to column Mode     80", "\x1B[?3l", []int{csi_privRM}, ColMode_C80},
		{"change to column Mode repeat", "\x1B[?3h\x1B[?3h", []int{csi_privSM, csi_privSM}, ColMode_C132},
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

func TestHandle_privSM_privRM_2(t *testing.T) {
	tc := []struct {
		name                string
		seq                 string
		hdIDs               []int
		compatLevel         CompatibilityLevel
		isResetCharsetState bool
	}{
		{"privSM 2", "\x1B[?2h", []int{csi_privSM}, CompatLevel_VT400, true},
		{"privRM 2", "\x1B[?2l", []int{csi_privRM}, CompatLevel_VT52, true},
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

func TestHandle_OSC_0_1_2(t *testing.T) {
	tc := []struct {
		name    string
		hdIDs   []int
		icon    bool
		title   bool
		seq     string
		wantStr string
	}{
		{"OSC 0;Pt BEL        ", []int{osc_0_1_2}, true, true, "\x1B]0;ada\x07", "ada"},
		{"OSC 1;Pt 7bit ST    ", []int{osc_0_1_2}, true, false, "\x1B]1;adas\x1B\\", "adas"},
		{"OSC 2;Pt BEL chinese", []int{osc_0_1_2}, false, true, "\x1B]2;[道德经]\x07", "[道德经]"},
		{"OSC 2;Pt BEL unusual", []int{osc_0_1_2}, false, true, "\x1B]2;[neovim]\x1B78\x07", "[neovim]\x1B78"},
	}

	p := NewParser()
	emu := NewEmulator3(8, 4, 4) // this is the pre-condidtion for the test case.

	for _, v := range tc {
		var hd *Handler
		p.reset()
		// parse the sequence
		for _, ch := range v.seq {
			hd = p.processInput(ch)
		}

		if hd != nil {
			// handle the instruction
			hd.handle(emu)

			// get the result
			windowTitle := emu.cf.windowTitle
			iconName := emu.cf.iconName

			if hd.id != v.hdIDs[0] {
				t.Errorf("%s seq=%q handler expect %q, got %q\n", v.name, v.seq, strHandlerID[v.hdIDs[0]], strHandlerID[hd.id])
			}
			if v.title && !v.icon && windowTitle != v.wantStr {
				t.Errorf("%s seq=%q only title should be set.\nexpect %q, \ngot %q\n", v.name, v.seq, v.wantStr, windowTitle)
			}
			if !v.title && v.icon && iconName != v.wantStr {
				t.Errorf("%s seq=%q only icon name should be set.\nexpect %q, \ngot %q\n", v.name, v.seq, v.wantStr, iconName)
			}
			if v.title && v.icon && (iconName != v.wantStr || windowTitle != v.wantStr) {
				t.Errorf("%s seq=%q both icon name and window title should be set.\nexpect %q, \ngot window title:%q\ngot iconName:%q\n",
					v.name, v.seq, v.wantStr, windowTitle, iconName)
			}
		} else {
			if p.inputState == InputState_Normal && v.wantStr == "" {
				continue
			}
			t.Errorf("%s got nil return\n", v.name)
		}
	}
}

func TestHandle_OSC_Abort(t *testing.T) {
	tc := []struct {
		name string
		seq  string
		want string
	}{
		{"OSC malform 1         ", "\x1B]ada\x1B\\", "OSC: no ';' exist."},
		{"OSC malform 2         ", "\x1B]7fy;ada\x1B\\", "OSC: illegal Ps parameter."},
		{"OSC Ps overflow: >120 ", "\x1B]121;home\x1B\\", "OSC: malformed command string"},
		{"OSC malform 3         ", "\x1B]7;ada\x1B\\", "unhandled OSC:"},
	}
	p := NewParser()
	var place strings.Builder
	p.logT.SetOutput(&place) // redirect the output to the string builder
	p.logU.SetOutput(&place)

	for _, v := range tc {
		// reset the out put for every test case
		place.Reset()
		var hd *Handler

		// parse the sequence
		for _, ch := range v.seq {
			hd = p.processInput(ch)
		}

		if hd != nil {
			t.Errorf("%s: seq=%q for abort case, hd should be nil. hd=%v\n", v.name, v.seq, hd)
		}

		got := place.String()
		if !strings.Contains(got, v.want) {
			t.Errorf("%s: seq=%q expect %s, got %s\n", v.name, v.seq, v.want, got)
		}
	}
}

func TestHandle_OSC_52(t *testing.T) {
	tc := []struct {
		name       string
		hdIDs      []int
		wantPc     string
		wantPd     string
		wantString string
		noReply    bool
		seq        string
	}{
		{
			"new selection in c",
			[]int{osc_52},
			"c", "YXByaWxzaAo=",
			"\x1B]52;c;YXByaWxzaAo=\x1B\\", true,
			"\x1B]52;c;YXByaWxzaAo=\x1B\\",
		},
		{
			"clear selection in cs",
			[]int{osc_52, osc_52},
			"cs", "",
			"\x1B]52;cs;x\x1B\\", true, // echo "aprilsh" | base64
			"\x1B]52;cs;YXByaWxzaAo=\x1B\\\x1B]52;cs;x\x1B\\",
		},
		{
			"empty selection",
			[]int{osc_52},
			"s0", "5Zub5aeR5aiY5bGxCg==", // echo "四姑娘山" | base64
			"\x1B]52;s0;5Zub5aeR5aiY5bGxCg==\x1B\\", true,
			"\x1B]52;;5Zub5aeR5aiY5bGxCg==\x1B\\",
		},
		{
			"question selection",
			[]int{osc_52, osc_52},
			"", "", // don't care these values
			"\x1B]52;c;5Zub5aeR5aiY5bGxCg==\x1B\\", false,
			"\x1B]52;c0;5Zub5aeR5aiY5bGxCg==\x1B\\\x1B]52;c0;?\x1B\\",
		},
	}
	p := NewParser()
	emu := NewEmulator3(8, 4, 0)
	for _, v := range tc {
		emu.cf.selectionData = ""
		emu.terminalToHost.Reset()

		t.Run(v.name, func(t *testing.T) {
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
					t.Errorf("%s: seq=%q expect %s, got %s\n", v.name, v.seq, strHandlerID[v.hdIDs[j]], strHandlerID[hd.id])
				}
			}

			if v.noReply {
				if v.wantString != emu.cf.selectionData {
					t.Errorf("%s: seq=%q expect %q, got %q\n", v.name, v.seq, v.wantString, emu.cf.selectionData)
				}
				for _, ch := range v.wantPc {
					if data, ok := emu.selectionData[ch]; ok && data == v.wantPd {
						continue
					} else {
						t.Errorf("%s: seq=%q, expect[%c]%q, got [%c]%q\n", v.name, v.seq, ch, v.wantPc, ch, emu.selectionData[ch])
					}
				}
			} else {
				got := emu.terminalToHost.String()
				if got != v.wantString {
					t.Errorf("%s: seq=%q, expect %q, got %q\n", v.name, v.seq, v.wantString, got)
				}
			}
		})
	}
}

func TestHandle_OSC_52_abort(t *testing.T) {
	tc := []struct {
		name    string
		hdIDs   []int
		wantStr string
		seq     string
	}{
		{"malform OSC 52 ", []int{osc_52}, "OSC 52: can't find Pc parameter.", "\x1B]52;23\x1B\\"},
		{"Pc not in range", []int{osc_52}, "invalid Pc parameters.", "\x1B]52;se;\x1B\\"},
	}
	p := NewParser()
	emu := NewEmulator3(8, 4, 0)
	var place strings.Builder
	emu.logW.SetOutput(&place)

	for _, v := range tc {
		place.Reset()
		t.Run(v.name, func(t *testing.T) {
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
					t.Errorf("%s: seq=%q expect %s, got %s\n", v.name, v.seq, strHandlerID[v.hdIDs[j]], strHandlerID[hd.id])
				}
			}

			if !strings.Contains(place.String(), v.wantStr) {
				t.Errorf("%s: seq=%q expect %q, got %q\n", v.name, v.seq, v.wantStr, place.String())
			}
		})
	}
}

func TestHandle_OSC_4(t *testing.T) {
	// color.Palette.Index(c color.Color)
	tc := []struct {
		name       string
		hdIDs      []int
		wantString string
		warn       bool
		seq        string
	}{
		{
			"query one color number",
			[]int{osc_4},
			"\x1B]4;1;rgb:8080/0000/0000\x1B\\", false,
			"\x1B]4;1;?\x1B\\",
		},
		{
			"query two color number",
			[]int{osc_4},
			"\x1B]4;250;rgb:bcbc/bcbc/bcbc\x1B\\\x1B]4;1;rgb:8080/0000/0000\x1B\\", false,
			"\x1B]4;250;?;1;?\x1B\\",
		},
		{
			"query 8 color number",
			[]int{osc_4},
			"\x1B]4;0;rgb:0000/0000/0000\x1B\\\x1B]4;1;rgb:8080/0000/0000\x1B\\\x1B]4;2;rgb:0000/8080/0000\x1B\\\x1B]4;3;rgb:8080/8080/0000\x1B\\\x1B]4;4;rgb:0000/0000/8080\x1B\\\x1B]4;5;rgb:8080/0000/8080\x1B\\\x1B]4;6;rgb:0000/8080/8080\x1B\\\x1B]4;7;rgb:c0c0/c0c0/c0c0\x1B\\", false,
			"\x1B]4;0;?;1;?;2;?;3;?;4;?;5;?;6;?;7;?\x1B\\",
		},
		{
			"missing ';' abort",
			[]int{osc_4},
			"OSC 4: malformed argument, missing ';'.", true,
			"\x1B]4;1?\x1B\\",
		},
		{
			"Ps malform abort",
			[]int{osc_4},
			"OSC 4: can't parse c parameter.", true,
			"\x1B]4;m;?\x1B\\",
		},
	}
	p := NewParser()
	emu := NewEmulator3(8, 4, 0)
	var place strings.Builder
	emu.logW.SetOutput(&place)

	for _, v := range tc {
		place.Reset()
		emu.terminalToHost.Reset()

		t.Run(v.name, func(t *testing.T) {
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
					t.Errorf("%s: seq=%q expect %s, got %s\n", v.name, v.seq, strHandlerID[v.hdIDs[j]], strHandlerID[hd.id])
				}
			}

			if v.warn {
				if !strings.Contains(place.String(), v.wantString) {
					t.Errorf("%s: seq=%q expect %q, got %q\n", v.name, v.seq, v.wantString, place.String())
				}
			} else {
				got := emu.terminalToHost.String()
				if got != v.wantString {
					t.Errorf("%s: seq=%q, \nexpect\t %q, \ngot\t\t %q\n", v.name, v.seq, v.wantString, got)
				}
			}
		})
	}
}

// echo -ne '\e]10;?\e\\'; cat
// echo -ne '\e]4;0;?\e\\'; cat
func TestHandle_OSC_10x(t *testing.T) {
	invalidColor := NewHexColor(0xF8F8F8)
	tc := []struct {
		name        string
		fgColor     Color
		bgColor     Color
		cursorColor Color
		hdIDs       []int
		wantString  string
		warn        bool
		seq         string
	}{
		{
			"query 6 color",
			ColorWhite, ColorGreen, ColorOlive,
			[]int{osc_10_11_12_17_19},
			"\x1B]10;rgb:ffff/ffff/ffff\x1B\\\x1B]11;rgb:0000/8080/0000\x1B\\\x1B]17;rgb:0000/8080/0000\x1B\\\x1B]19;rgb:ffff/ffff/ffff\x1B\\\x1B]12;rgb:8080/8080/0000\x1B\\", false,
			"\x1B]10;?;11;?;17;?;19;?;12;?\x1B\\",
		},
		{
			"parse color parameter error",
			invalidColor, invalidColor, invalidColor,
			[]int{osc_10_11_12_17_19},
			"OSC 10x: can't parse color index.", true,
			"\x1B]10;?;m;?\x1B\\",
		},
		{
			"malform parameter",
			invalidColor, invalidColor, invalidColor,
			[]int{osc_10_11_12_17_19},
			"OSC 10x: malformed argument, missing ';'.", true,
			"\x1B]10;?;\x1B\\",
		},
		{
			"VT100 text foreground color: regular color",
			ColorWhite, invalidColor, invalidColor,
			[]int{osc_10_11_12_17_19},
			"\x1B]10;rgb:ffff/ffff/ffff\x1B\\", false,
			"\x1B]10;?\x1B\\",
		},
		{
			"VT100 text background color: default color",
			invalidColor, ColorDefault, invalidColor,
			[]int{osc_10_11_12_17_19},
			"\x1B]11;rgb:0000/0000/0000\x1B\\", false,
			"\x1B]11;?\x1B\\",
		},
		{
			"text cursor color: regular color",
			invalidColor, invalidColor, ColorGreen,
			[]int{osc_10_11_12_17_19},
			"\x1B]12;rgb:0000/8080/0000\x1B\\", false,
			"\x1B]12;?\x1B\\",
		},
		{
			"text cursor color: default color",
			invalidColor, invalidColor, ColorDefault,
			[]int{osc_10_11_12_17_19},
			"\x1B]12;rgb:0000/0000/0000\x1B\\", false,
			"\x1B]12;?\x1B\\",
		},
	}
	p := NewParser()
	emu := NewEmulator3(80, 40, 5)
	var place strings.Builder
	emu.logW.SetOutput(&place)

	for _, v := range tc {
		place.Reset()
		emu.terminalToHost.Reset()

		t.Run(v.name, func(t *testing.T) {
			// process control sequence
			hds := make([]*Handler, 0, 16)
			hds = p.processStream(v.seq, hds)

			if len(hds) == 0 {
				t.Errorf("%s got zero handlers.", v.name)
			}

			// set pre-condition
			if v.fgColor != invalidColor {
				emu.attrs.renditions.fgColor = v.fgColor
			}
			if v.bgColor != invalidColor {
				emu.attrs.renditions.bgColor = v.bgColor
			}
			if v.cursorColor != invalidColor {
				emu.cf.DS.cursorColor = v.cursorColor
			}

			// execute the control sequence
			for j, hd := range hds {
				hd.handle(emu)
				if hd.id != v.hdIDs[j] { // validate the control sequences id
					t.Errorf("%s: seq=%q expect %s, got %s\n", v.name, v.seq, strHandlerID[v.hdIDs[j]], strHandlerID[hd.id])
				}
			}

			if v.warn {
				if !strings.Contains(place.String(), v.wantString) {
					t.Errorf("%s: seq=%q expect %q, got %q\n", v.name, v.seq, v.wantString, place.String())
				}
			} else {
				got := emu.terminalToHost.String()
				if got != v.wantString {
					t.Errorf("%s: seq=%q, \nexpect\t %q, \ngot\t\t %q\n", v.name, v.seq, v.wantString, got)
				}
			}
		})
	}
}

func TestHandle_DCS(t *testing.T) {
	tc := []struct {
		name    string
		hdIDs   []int
		wantMsg string
		warn    bool
		seq     string
	}{
		{"DECRQSS normal", []int{dcs_decrqss}, "\x1BP1$r" + DEVICE_ID + "\x1B\\", false, "\x1BP$q\"p\x1B\\"},
		{"decrqss others", []int{dcs_decrqss}, "\x1BP0$rother\x1B\\", false, "\x1BP$qother\x1B\\"},
		{"DCS unimplement", []int{dcs_decrqss}, "DCS:", true, "\x1BPunimplement\x1B78\x1B\\"},
	}
	p := NewParser()
	// p.logU = log.New(&place, "(Uimplemented): ", log.Ldate|log.Ltime|log.Lshortfile)
	emu := NewEmulator3(8, 4, 0)
	var place strings.Builder
	p.logU.SetOutput(&place) // redirect the output to the string builder

	for _, v := range tc {
		place.Reset()
		emu.terminalToHost.Reset()

		t.Run(v.name, func(t *testing.T) {
			// process control sequence
			hds := make([]*Handler, 0, 16)
			hds = p.processStream(v.seq, hds)

			if !v.warn && len(hds) == 0 {
				t.Errorf("%s got zero handlers.", v.name)
			}

			// execute the control sequence
			for j, hd := range hds {
				hd.handle(emu)
				if hd.id != v.hdIDs[j] { // validate the control sequences name
					t.Errorf("%s:\t %q expect %s, got %s\n", v.name, v.seq, strHandlerID[v.hdIDs[j]], strHandlerID[hd.id])
				}
			}

			if v.warn {
				if !strings.Contains(place.String(), v.wantMsg) {
					t.Errorf("%s: seq=%q expect %q, got %q\n", v.name, v.seq, v.wantMsg, place.String())
				}
			} else {
				got := emu.terminalToHost.String()
				if got != v.wantMsg {
					t.Errorf("%s: seq=%q, \nexpect\t %q, \ngot\t\t %q\n", v.name, v.seq, v.wantMsg, got)
				}
			}
		})
	}
}

func TestHandle_VT52_EGM_ID(t *testing.T) {
	tc := []struct {
		name      string
		seq       string
		hdIDs     []int
		charsetGL *map[byte]rune
		resp      string
	}{
		{"VT52 ESC F", "\x1B[?2l\x1BF", []int{csi_privRM, vt52_egm}, &vt_DEC_Special, ""},
		{"VT52 ESC Z", "\x1B[?2l\x1BZ", []int{csi_privRM, vt52_id}, nil, "\x1B/Z"},
	}

	p := NewParser()
	emu := NewEmulator3(8, 4, 0)
	// var place strings.Builder
	// p.logU.SetOutput(&place)

	for _, v := range tc {
		// place.Reset()
		p.reset()
		emu.terminalToHost.Reset()

		t.Run(v.name, func(t *testing.T) {
			// process control sequence
			hds := make([]*Handler, 0, 16)
			hds = p.processStream(v.seq, hds)

			if len(hds) == 0 {
				t.Errorf("%s got zero handlers.", v.name)
			}

			// execute the control sequence
			for j, hd := range hds {
				hd.handle(emu)
				if hd.id != v.hdIDs[j] { // validate the control sequences name
					t.Errorf("%s: seq=%q expect %s, got %s\n", v.name, v.seq, strHandlerID[v.hdIDs[j]], strHandlerID[hd.id])
				}
			}

			if v.resp == "" {
				got := emu.charsetState.g[emu.charsetState.gl]
				if !reflect.DeepEqual(got, v.charsetGL) {
					// if got != v.charsetGL {
					t.Errorf("%s seq=%q GL charset expect %p, got %p\n", v.name, v.seq, v.charsetGL, got)
				}
			} else {
				got := emu.terminalToHost.String()
				if got != v.resp {
					t.Errorf("%s seq=%q response expect %q, got %q\n", v.name, v.seq, v.resp, got)
				}
			}
		})
	}
}
