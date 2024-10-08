// Copyright 2022~2024 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package terminal

import (
	"container/list"
	"strconv"
	"strings"

	"github.com/ericwq/aprilsh/util"
	"github.com/rivo/uniseg"
)

const (
	InputState_Normal = iota
	InputState_Escape
	InputState_Escape_VT52
	InputState_Esc_Space
	InputState_Esc_Hash
	InputState_Esc_Pct
	InputState_Select_Charset
	InputState_CSI
	InputState_CSI_Priv
	InputState_CSI_Quote
	InputState_CSI_DblQuote
	InputState_CSI_Bang
	InputState_CSI_SPC
	InputState_CSI_GT
	InputState_CSI_LT
	InputState_CSI_Equal
	InputState_DCS
	InputState_DCS_Esc
	InputState_OSC
	InputState_OSC_Esc
	InputState_VT52_CUP_Arg1
	InputState_VT52_CUP_Arg2
)

var strInputState = [...]string{
	"Normal",
	"Escape",
	"Escape_VT52",
	"Esc_Space",
	"Esc_Hash",
	"Esc_Pct",
	"Select_Charset",
	"CSI",
	"CSI_Priv",
	"CSI_Quote",
	"CSI_DblQuote",
	"CSI_Bang",
	"CSI_SPC",
	"CSI_GT",
	"CSI_LT",
	"CSI_Equal",
	"DCS",
	"DCS_Esc",
	"OSC",
	"OSC_Esc",
	"VT52_CUP_Arg1",
	"VT52_CUP_Arg2",
}

type Parser struct {
	// history, raw handler sequence
	history *list.List

	argBuf     strings.Builder // string parameter
	lastChs    []rune          // last graphemes
	inputOps   []int           // numeric parameters
	inputSep   []rune          // input speerator ':', ";", or empty
	chs        []rune          // current graphemes
	inputState int             // parser state
	nInputOps  int             // numeric parameter number
	maxEscOps  int
	ch         rune // currrent rune

	// select character set destination and mode
	scsDst rune
	scsMod rune

	handleReady bool               // handler is ready
	compatLevel CompatibilityLevel // independent from compatLevel in emulator

	// G0~G3 character set compatiable mode, default false
	vtMode bool

	logTrace bool
}

func NewParser() *Parser {
	p := &Parser{}

	p.reset()
	return p
}

// return the state of parser
func (p *Parser) getState() int {
	return p.inputState
}

// add rune to the history cache, store max 5 recent runes.
func (p *Parser) appendToHistory(r rune) {
	// max history = DCS/OSC buffer limitation 4095 + 2
	if p.history.Len() < 4097 {
		p.history.PushBack(r)
	} else {
		util.Logger.Error("Parser histroy string overflow (>4097)",
			"historyString", p.historyString(),
			"rune", r)
	}
}

func (p *Parser) replaceHistory(chs ...rune) {
	p.resetHistory()
	// for i := range p.inputSep {
	// 	p.inputSep[i] = 0
	// }

	for _, r := range chs {
		p.appendToHistory(r)
	}
}

// return the history string representation
func (p *Parser) historyString() string {
	var str strings.Builder
	// Iterate through list and print its contents.
	for e := p.history.Front(); e != nil; e = e.Next() {
		str.WriteRune(e.Value.(rune))
	}

	return str.String()
}

// reset the history cache
func (p *Parser) resetHistory() {
	p.history = list.New()
}

// get the rune value in the specified reverse index
// return 0 if error.
func (p *Parser) getHistoryAt(reverseIdx int) (r rune) {
	if reverseIdx >= 5 {
		return 0
	}
	x := p.history.Back()
	for i := reverseIdx; i > 0; i-- {
		x = x.Prev()
	}
	if v, ok := x.Value.(rune); ok {
		r = v
	}
	return r
}

func (p *Parser) ResetInput() {
	p.reset()
}

func (p *Parser) reset() {
	p.inputState = InputState_Normal
	p.ch = 0x00
	p.chs = nil
	p.lastChs = nil
	p.handleReady = false

	// p.perror = nil

	p.maxEscOps = 16
	p.inputOps = make([]int, p.maxEscOps)
	p.inputSep = make([]rune, p.maxEscOps)
	p.nInputOps = 0
	p.argBuf.Reset()

	p.resetHistory()
	p.scsDst = 0x00
	p.scsMod = 0x00
	p.vtMode = false
	p.logTrace = false

	p.compatLevel = CompatLevel_VT400
}

// trace the input if logTrace is true
func (p *Parser) traceNormalInput() {
	if p.logTrace {
		util.Logger.Debug("Input:",
			"input", p.chs,
			"inputOps", p.inputOps,
			"nInputOps", p.nInputOps,
			"argBuf", p.argBuf.String())
	}
}

// log the unhandled input and reset the state to normal
func (p *Parser) unhandledInput() {
	util.Logger.Warn("Unhandled input:",
		"input", p.historyString(),
		"state", strInputState[p.inputState],
		"inputOps", p.inputOps,
		"nInputOps", p.nInputOps,
		"argBuf", p.argBuf.String(),
		"unimplement", "Any")
	p.setState(InputState_Normal)
}

// set the parser new state.
// if new state is the same as old state just return.
// if new state is normal, reset the parameter buffer: inputOps[]
// if old state is normal, print the trace infomation
func (p *Parser) setState(newState int) {
	if newState == p.inputState {
		return
	}

	if newState == InputState_Normal {
		p.nInputOps = 0
		p.inputOps[0] = 0
		// p.resetHistory()
	} else if p.inputState == InputState_Normal {
		p.traceNormalInput()
	}

	p.inputState = newState
}

// collect numeric parameter and store them in inputOps array.
func (p *Parser) collectNumericParameters(ch rune) (isNumeric bool) {
	if '0' <= ch && ch <= '9' {
		isNumeric = true
		// max value for numeric parameter
		p.inputOps[p.nInputOps-1] *= 10
		p.inputOps[p.nInputOps-1] += int(ch - '0')
		if p.inputOps[p.nInputOps-1] >= 65535 {
			util.Logger.Error("the number is too big: > 65535", "lastInputOps", p.inputOps[p.nInputOps-1])
			p.setState(InputState_Normal)
		}
	} else if ch == ';' || ch == ':' {
		isNumeric = true
		if p.nInputOps < p.maxEscOps { // move to the next parameter
			p.inputSep[p.nInputOps-1] = ch
			p.inputOps[p.nInputOps] = 0
			p.nInputOps += 1
		} else {
			// p.logE.Printf("inputOps full, increase maxEscOps. %d", p.inputOps)
			util.Logger.Error("inputOps full, increase maxEscOps", "inputOps", p.inputOps)
			p.setState(InputState_Normal)
		}
	}
	return isNumeric
}

// get number n parameter from parser
// if the return parameter is zero, use the defaultVal instead
func (p *Parser) getPs(n int, defaultVal int) int {
	ret := defaultVal
	if n < p.nInputOps {
		ret = p.inputOps[n]
	}

	if ret < 1 {
		ret = defaultVal
	}
	return ret
}

// get the string parameter from parser
func (p *Parser) getArg() (arg string) {
	if p.argBuf.Len() > 0 {
		arg = p.argBuf.String()
	}

	return arg
}

// copy the numeric parameters slice
func (p *Parser) copyArgs() (args []int) {
	if p.nInputOps == 1 && p.inputOps[0] == 0 {
		args = nil
	} else {
		args = make([]int, p.nInputOps)
		copy(args, p.inputOps)
	}
	return
}

func (p *Parser) copySeps() (args []rune) {
	if p.nInputOps == 1 && p.inputSep[0] == 0 {
		args = []rune{}
	} else {
		args = make([]rune, p.nInputOps)
		copy(args, p.inputSep)
	}
	return
}

// set compatLevel if params contains value '2',or just set the compatLevel.
// only set compatLevel for parser.
func (p *Parser) setCompatLevel(cl CompatibilityLevel, params ...int) {
	if len(params) == 0 {
		if p.compatLevel != cl && cl != CompatLevel_Unused {
			p.compatLevel = cl
		}
	} else {
		for _, v := range params {
			if v == 2 {
				p.compatLevel = cl
				break
			}
		}
	}
}

// func handle_UserByte(ch rune) (hd *Handler) {
// 	u := UserByte{ch}
// 	hd = &Handler{name: "user-byte", ch: ch}
// 	hd.handle = func(emu *emulator) {
// 		hdl_userbyte(emu, u)
// 	}
// 	return hd
// }
//
// func handle_Resize(width, height int) (hd *Handler) {
// 	// resize := Resize{width, height}
//
// 	hd = &Handler{name: "resize"}
// 	hd.handle = func(emu *emulator) {
// 		hdl_resize(emu, width, height)
// 	}
// 	return hd
// }

// print graphemes on screen
func (p *Parser) handle_Graphemes() (hd *Handler) {
	hd = &Handler{id: Graphemes, ch: p.ch, sequence: p.historyString()}

	// store the last graphic character
	r := p.chs
	p.lastChs = make([]rune, len(r))
	copy(p.lastChs[0:], r[0:])

	hd.handle = func(emu *Emulator) {
		hdl_graphemes(emu, r...)
	}
	return hd
}

// Cursor up by <n>
func (p *Parser) handle_CUU() (hd *Handler) {
	num := p.getPs(0, 1)

	hd = &Handler{id: CSI_CUU, ch: p.ch, sequence: p.historyString()}
	hd.handle = func(emu *Emulator) {
		hdl_csi_cuu(emu, num)
	}

	p.setState(InputState_Normal)
	return hd
}

// Cursor down by <n>
func (p *Parser) handle_CUD() (hd *Handler) {
	num := p.getPs(0, 1)

	hd = &Handler{id: CSI_CUD, ch: p.ch, sequence: p.historyString()}
	hd.handle = func(emu *Emulator) {
		hdl_csi_cud(emu, num)
	}

	p.setState(InputState_Normal)
	return hd
}

// Cursor forward (Right) by <n>
func (p *Parser) handle_CUF() (hd *Handler) {
	num := p.getPs(0, 1)

	hd = &Handler{id: CSI_CUF, ch: p.ch, sequence: p.historyString()}
	hd.handle = func(emu *Emulator) {
		hdl_csi_cuf(emu, num)
	}

	p.setState(InputState_Normal)
	return hd
}

// Cursor backward (Left) by <n>
func (p *Parser) handle_CUB() (hd *Handler) {
	num := p.getPs(0, 1)

	hd = &Handler{id: CSI_CUB, ch: p.ch, sequence: p.historyString()}
	hd.handle = func(emu *Emulator) {
		hdl_csi_cub(emu, num)
	}

	p.setState(InputState_Normal)
	return hd
}

// prepare parameters for the CUP
// Cursor moves to <row>; <col> coordinate within the viewport
// Cursor Position
func (p *Parser) handle_CUP() (hd *Handler) {
	row := p.getPs(0, 1)
	col := p.getPs(1, 1)

	hd = &Handler{id: CSI_CUP, ch: p.ch, sequence: p.historyString()}
	hd.handle = func(emu *Emulator) {
		hdl_csi_cup(emu, row, col)
	}

	// reset the state
	p.setState(InputState_Normal)
	return hd
}

// Operating System Command
// OSC Ps ; Pt Bell
// OSC Ps ; Pt ST
// Set Text Parameters.  Some control sequences return information:
func (p *Parser) handle_OSC() (hd *Handler) {
	// Here we parse the parameters by ourselves.
	cmd := 0
	arg := p.getArg()

	defer p.setState(InputState_Normal)

	// get Ps parameter
	hasPt := true
	pos := strings.Index(arg, ";")
	if pos == -1 { // no Pt parameter
		pos = len(arg)
		hasPt = false
	}
	var err error
	if cmd, err = strconv.Atoi(arg[:pos]); err != nil {
		util.Logger.Warn("OSC: illegal Ps parameter", "arg", arg[:pos])
		return
	}

	// get Pt parameter
	if !hasPt {
		arg = ""
	} else {
		arg = arg[pos+1:]
	}
	if cmd < 0 || cmd > 120 {
		util.Logger.Warn("OSC: malformed command string", "cmd", cmd, "arg", arg)
	} else {
		switch cmd {
		// create the ActOn
		case 0, 1, 2:
			hd = &Handler{id: OSC_0_1_2, ch: p.ch, sequence: p.historyString()}
			hd.handle = func(emu *Emulator) {
				hdl_osc_0_1_2(emu, cmd, arg)
			}
		case 4:
			hd = &Handler{id: OSC_4, ch: p.ch, sequence: p.historyString()}
			hd.handle = func(emu *Emulator) {
				hdl_osc_4(emu, cmd, arg)
			}
		case 52:
			hd = &Handler{id: OSC_52, ch: p.ch, sequence: p.historyString()}
			hd.handle = func(emu *Emulator) {
				hdl_osc_52(emu, cmd, arg)
			}
		case 10, 11, 12, 17, 19:
			hd = &Handler{id: OSC_10_11_12_17_19, ch: p.ch, sequence: p.historyString()}
			hd.handle = func(emu *Emulator) {
				hdl_osc_10x(emu, cmd, arg)
			}
		case 112:
			hd = &Handler{id: OSC_112, ch: p.ch, sequence: p.historyString()}
			hd.handle = func(emu *Emulator) {
				hdl_osc_112(emu, cmd, arg)
			}
		case 8:
			hd = &Handler{id: OSC_8, ch: p.ch, sequence: p.historyString()}
			hd.handle = func(emu *Emulator) {
				hdl_osc_8(emu, cmd, arg)
			}
		default:
			util.Logger.Warn("unhandled OSC", "cmd", cmd, "arg", arg, "seq", p.historyString())
		}
	}

	return hd
}

// Carriage Return
func (p *Parser) handle_CR() (hd *Handler) {
	hd = &Handler{id: C0_CR, ch: p.ch, sequence: p.historyString()}
	hd.handle = func(emu *Emulator) {
		hdl_c0_cr(emu)
	}
	// Do NOT reset the state
	return hd
}

// Line Feed
func (p *Parser) handle_IND() (hd *Handler) {
	hd = &Handler{id: ESC_IND, ch: p.ch, sequence: p.historyString()}
	hd.handle = func(emu *Emulator) {
		hdl_esc_ind(emu)
	}
	// Do NOT reset the state
	return hd
}

// Horizontal Tab
// move cursor position to next tab stop
func (p *Parser) handle_HT() (hd *Handler) {
	hd = &Handler{id: C0_HT, ch: p.ch, sequence: p.historyString()}
	hd.handle = func(emu *Emulator) {
		hdl_c0_ht(emu)
	}
	// Do NOT reset the state
	return hd
}

// Bell
func (p *Parser) handle_BEL() (hd *Handler) {
	hd = &Handler{id: C0_BEL, ch: p.ch, sequence: p.historyString()}
	hd.handle = func(emu *Emulator) {
		hdl_c0_bel(emu)
	}
	// Do NOT reset the state
	return hd
}

// SI - switch to standard character set
func (p *Parser) handle_SI() (hd *Handler) {
	hd = &Handler{id: C0_SI, ch: p.ch, sequence: p.historyString()}
	hd.handle = func(emu *Emulator) {
		hdl_c0_si(emu)
	}
	// Do NOT reset the state
	return hd
}

// SO - switch to alternate character set
func (p *Parser) handle_SO() (hd *Handler) {
	hd = &Handler{id: C0_SO, ch: p.ch, sequence: p.historyString()}
	hd.handle = func(emu *Emulator) {
		hdl_c0_so(emu)
	}
	// Do NOT reset the state
	return hd
}

// set cursor position as tab stop position
func (p *Parser) handle_HTS() (hd *Handler) {
	hd = &Handler{id: ESC_HTS, ch: p.ch, sequence: p.historyString()}
	hd.handle = func(emu *Emulator) {
		hdl_esc_hts(emu)
	}

	p.setState(InputState_Normal)
	return hd
}

// move cursor forward to the N tab stop position
// or just got focus.
func (p *Parser) handle_CHT_Focus() (hd *Handler) {
	if len(p.historyString()) == 3 {
		// just \x1B[I
		hd = p.handle_Focus(true)
	} else {
		count := p.getPs(0, 1)

		hd = &Handler{id: CSI_CHT, ch: p.ch, sequence: p.historyString()}
		hd.handle = func(emu *Emulator) {
			hdl_csi_cht(emu, count)
		}
		p.setState(InputState_Normal)
	}
	return hd
}

// move cursor backward to the N tab stop position
func (p *Parser) handle_CBT() (hd *Handler) {
	count := p.getPs(0, 1)

	hd = &Handler{id: CSI_CBT, ch: p.ch, sequence: p.historyString()}
	hd.handle = func(emu *Emulator) {
		hdl_csi_cbt(emu, count)
	}

	p.setState(InputState_Normal)
	return hd
}

// clear tab stop position according to cmd
func (p *Parser) handle_TBC() (hd *Handler) {
	cmd := p.getPs(0, 0)

	hd = &Handler{id: CSI_TBC, ch: p.ch, sequence: p.historyString()}
	hd.handle = func(emu *Emulator) {
		hdl_csi_tbc(emu, cmd)
	}

	p.setState(InputState_Normal)
	return hd
}

// inserts one or more space (SP) characters starting at the cursor position.
// Insert <n> spaces at the current cursor position, shifting all existing text
// to the right. Text exiting the screen to the right is removed.
func (p *Parser) handle_ICH() (hd *Handler) {
	count := p.getPs(0, 1)

	hd = &Handler{id: CSI_ICH, ch: p.ch, sequence: p.historyString()}
	hd.handle = func(emu *Emulator) {
		hdl_csi_ich(emu, count)
	}

	p.setState(InputState_Normal)
	return hd
}

// Cursor moves to <n>th position horizontally in the current line
// Character Position Absolute
func (p *Parser) handle_HPA() (hd *Handler) {
	count := p.getPs(0, 1)

	hd = &Handler{id: CSI_HPA, ch: p.ch, sequence: p.historyString()}
	hd.handle = func(emu *Emulator) {
		hdl_csi_hpa(emu, count)
	}

	p.setState(InputState_Normal)
	return hd
}

// Character Position Relative
func (p *Parser) handle_HPR() (hd *Handler) {
	count := p.getPs(0, 1)

	hd = &Handler{id: CSI_HPR, ch: p.ch, sequence: p.historyString()}
	hd.handle = func(emu *Emulator) {
		hdl_csi_hpr(emu, count)
	}

	p.setState(InputState_Normal)
	return hd
}

// Cursor moves to <n>th position horizontally in the current line
// Cursor Character Absolute_
func (p *Parser) handle_CHA() (hd *Handler) {
	count := p.getPs(0, 1)

	hd = &Handler{id: CSI_CHA, ch: p.ch, sequence: p.historyString()}
	hd.handle = func(emu *Emulator) {
		hdl_csi_cha(emu, count)
	}

	p.setState(InputState_Normal)
	return hd
}

// This control function erases characters from part or all of the display.
// When you erase complete lines, they become single-height, single-width
// lines, with all visual character attributes cleared. ED works inside or
// outside the scrolling margins.
// Erase in Display
func (p *Parser) handle_ED() (hd *Handler) {
	cmd := p.getPs(0, 0)

	hd = &Handler{id: CSI_ED, ch: p.ch, sequence: p.historyString()}
	hd.handle = func(emu *Emulator) {
		hdl_csi_ed(emu, cmd)
	}

	p.setState(InputState_Normal)
	return hd
}

// This control function erases characters on the line that has the cursor.
// EL clears all character attributes from erased character positions. EL
// works inside or outside the scrolling margins.
// Erase in Line
func (p *Parser) handle_EL() (hd *Handler) {
	cmd := p.getPs(0, 0)

	hd = &Handler{id: CSI_EL, ch: p.ch, sequence: p.historyString()}
	hd.handle = func(emu *Emulator) {
		hdl_csi_el(emu, cmd)
	}

	p.setState(InputState_Normal)
	return hd
}

// This control function inserts one or more blank lines, starting at the
// cursor.
// Insert Line
func (p *Parser) handle_IL() (hd *Handler) {
	lines := p.getPs(0, 1)

	hd = &Handler{id: CSI_IL, ch: p.ch, sequence: p.historyString()}
	hd.handle = func(emu *Emulator) {
		hdl_csi_il(emu, lines)
	}

	p.setState(InputState_Normal)
	return hd
}

// This control function deletes one or more lines in the scrolling region,
// starting with the line that has the cursor.
// Delete Line
func (p *Parser) handle_DL() (hd *Handler) {
	lines := p.getPs(0, 1)

	hd = &Handler{id: CSI_DL, ch: p.ch, sequence: p.historyString()}
	hd.handle = func(emu *Emulator) {
		hdl_csi_dl(emu, lines)
	}

	p.setState(InputState_Normal)
	return hd
}

func (p *Parser) handle_Focus(hasFocus bool) (hd *Handler) {
	var id int

	if hasFocus {
		id = CSI_FocusIn
	} else {
		id = CSI_FocusOut
	}
	hd = &Handler{id: id, ch: p.ch, sequence: p.historyString()}
	hd.handle = func(emu *Emulator) {
		hdl_csi_focus(emu, hasFocus)
	}
	p.setState(InputState_Normal)
	return hd
}

// This control function deletes one or more characters from the cursor
// position to the right.
// Delete <n> characters at the current cursor position, shifting in
// space characters from the right edge of the screen.
func (p *Parser) handle_DCH() (hd *Handler) {
	cells := p.getPs(0, 1)

	hd = &Handler{id: CSI_DCH, ch: p.ch, sequence: p.historyString()}
	hd.handle = func(emu *Emulator) {
		hdl_csi_dch(emu, cells)
	}

	p.setState(InputState_Normal)
	return hd
}

// This control function moves the user window down a specified number of lines in page memory.
// Scroll text up by <n>. Also known as pan down, new lines fill in from the bottom of the screen
// text up, window down.
func (p *Parser) handle_SU() (hd *Handler) {
	lines := p.getPs(0, 1)

	hd = &Handler{id: CSI_SU, ch: p.ch, sequence: p.historyString()}
	hd.handle = func(emu *Emulator) {
		hdl_csi_su(emu, lines)
	}

	p.setState(InputState_Normal)
	return hd
}

// /This control function moves the user window up a specified number of lines in page memory.
// Scroll down by <n>. Also known as pan up, new lines fill in from the top of the screen
// text down, window up.
func (p *Parser) handle_SD() (hd *Handler) {
	lines := p.getPs(0, 1)

	hd = &Handler{id: CSI_SD, ch: p.ch, sequence: p.historyString()}
	hd.handle = func(emu *Emulator) {
		hdl_csi_sd(emu, lines)
	}

	p.setState(InputState_Normal)
	return hd
}

// This control function erases one or more characters, from the cursor
// position to the right. ECH clears character attributes from erased
// character positions. ECH works inside or outside the scrolling margins.
func (p *Parser) handle_ECH() (hd *Handler) {
	num := p.getPs(0, 1)

	hd = &Handler{id: CSI_ECH, ch: p.ch, sequence: p.historyString()}
	hd.handle = func(emu *Emulator) {
		hdl_csi_ech(emu, num)
	}

	p.setState(InputState_Normal)
	return hd
}

// In this DA exchange, the host asks for the terminal's architectural
// class and basic attributes.
//
// The terminal responds by sending its architectural class and basic
// attributes to the host. This response depends on the terminal's
// current operating VT level.
// Device Attributes (Primary)
func (p *Parser) handle_priDA() (hd *Handler) {
	hd = &Handler{id: CSI_priDA, ch: p.ch, sequence: p.historyString()}
	hd.handle = func(emu *Emulator) {
		hdl_csi_priDA(emu)
	}

	p.setState(InputState_Normal)
	return hd
}

// In this DA exchange, the host requests the terminal's identification
// code, firmware version level, and hardware options.
// Device Attributes (Secondary)
func (p *Parser) handle_secDA() (hd *Handler) {
	hd = &Handler{id: CSI_secDA, ch: p.ch, sequence: p.historyString()}
	hd.handle = func(emu *Emulator) {
		hdl_csi_secDA(emu)
	}

	p.setState(InputState_Normal)
	return hd
}

// VPA causes the active position to be moved to the corresponding horizontal position.
// Cursor moves to the <n>th position vertically in the current column
// Line Position Absolute
func (p *Parser) handle_VPA() (hd *Handler) {
	row := p.getPs(0, 1)

	hd = &Handler{id: CSI_VPA, ch: p.ch, sequence: p.historyString()}
	hd.handle = func(emu *Emulator) {
		hdl_csi_vpa(emu, row)
	}

	p.setState(InputState_Normal)
	return hd
}

// Line Position Relative
func (p *Parser) handle_VPR() (hd *Handler) {
	row := p.getPs(0, 1)

	hd = &Handler{id: CSI_VPR, ch: p.ch, sequence: p.historyString()}
	hd.handle = func(emu *Emulator) {
		hdl_csi_vpr(emu, row)
	}

	p.setState(InputState_Normal)
	return hd
}

// select graphics rendition -- e.g., bold, blinking, etc.
func (p *Parser) handle_SGR() (hd *Handler) {
	params := p.copyArgs()
	if params == nil { // default value is 0
		params = []int{0}
	}
	spes := p.copySeps()

	hd = &Handler{id: CSI_SGR, ch: p.ch, sequence: p.historyString()}
	hd.handle = func(emu *Emulator) {
		hdl_csi_sgr(emu, params, spes...)
	}

	p.setState(InputState_Normal)
	return hd
}

// Operating Status: https://www.vt100.net/docs/vt510-rm/DSR-OS.html
// Cursor Position Report: https://www.vt100.net/docs/vt510-rm/DSR-CPR.html
// Device Status Reports
func (p *Parser) handle_DSR() (hd *Handler) {
	cmd := p.getPs(0, 0)

	hd = &Handler{id: CSI_DSR, ch: p.ch, sequence: p.historyString()}
	hd.handle = func(emu *Emulator) {
		hdl_csi_dsr(emu, cmd)
	}

	p.setState(InputState_Normal)
	return hd
}

// ESC N Single Shift Select of G2 Character Set (SS2  is 0x8e), VT220.
func (p *Parser) handle_SS2() (hd *Handler) {
	hd = &Handler{id: ESC_SS2, ch: p.ch, sequence: p.historyString()}
	hd.handle = func(emu *Emulator) {
		hdl_esc_ss2(emu)
	}

	p.setState(InputState_Normal)
	return hd
}

// ESC O Single Shift Select of G3 Character Set (SS3  is 0x8f), VT220.
func (p *Parser) handle_SS3() (hd *Handler) {
	hd = &Handler{id: ESC_SS3, ch: p.ch, sequence: p.historyString()}
	hd.handle = func(emu *Emulator) {
		hdl_esc_ss3(emu)
	}

	p.setState(InputState_Normal)
	return hd
}

// LS1R gr = 1
func (p *Parser) handle_LS1R() (hd *Handler) {
	hd = &Handler{id: ESC_LS1R, ch: p.ch, sequence: p.historyString()}
	hd.handle = func(emu *Emulator) {
		hdl_esc_ls1r(emu)
	}

	p.setState(InputState_Normal)
	return hd
}

// LS2 gl = 2
func (p *Parser) handle_LS2() (hd *Handler) {
	hd = &Handler{id: ESC_LS2, ch: p.ch, sequence: p.historyString()}
	hd.handle = func(emu *Emulator) {
		hdl_esc_ls2(emu)
	}

	p.setState(InputState_Normal)
	return hd
}

// LS2R gr = 2
func (p *Parser) handle_LS2R() (hd *Handler) {
	hd = &Handler{id: ESC_LS2R, ch: p.ch, sequence: p.historyString()}
	hd.handle = func(emu *Emulator) {
		hdl_esc_ls2r(emu)
	}

	p.setState(InputState_Normal)
	return hd
}

// LS3 gl = 3
func (p *Parser) handle_LS3() (hd *Handler) {
	hd = &Handler{id: ESC_LS3, ch: p.ch, sequence: p.historyString()}
	hd.handle = func(emu *Emulator) {
		hdl_esc_ls3(emu)
	}

	p.setState(InputState_Normal)
	return hd
}

// LS3R gr = 3
func (p *Parser) handle_LS3R() (hd *Handler) {
	hd = &Handler{id: ESC_LS3R, ch: p.ch, sequence: p.historyString()}
	hd.handle = func(emu *Emulator) {
		hdl_esc_ls3r(emu)
	}

	p.setState(InputState_Normal)
	return hd
}

// DOCS Select charset: UTF-8
func (p *Parser) handle_DOCS_UTF8() (hd *Handler) {
	hd = &Handler{id: ESC_DOCS_UTF8, ch: p.ch, sequence: p.historyString()}
	hd.handle = func(emu *Emulator) {
		hdl_esc_docs_utf8(emu)
	}

	p.vtMode = false
	p.setState(InputState_Normal)
	return hd
}

// DOCS Select charset: default (ISO-8859-1)
func (p *Parser) handle_DOCS_ISO8859_1() (hd *Handler) {
	hd = &Handler{id: ESC_DOCS_ISO8859_1, ch: p.ch, sequence: p.historyString()}
	hd.handle = func(emu *Emulator) {
		hdl_esc_docs_iso8859_1(emu)
	}

	p.vtMode = true
	p.setState(InputState_Normal)
	return hd
}

// Performs the reverse operation of \n, moves cursor up one line, maintains
// horizontal position, scrolls buffer if necessary
// Reverse Index
func (p *Parser) handle_RI() (hd *Handler) {
	hd = &Handler{id: ESC_RI, ch: p.ch, sequence: p.historyString()}
	hd.handle = func(emu *Emulator) {
		hdl_esc_ri(emu)
	}

	p.setState(InputState_Normal)
	return hd
}

// move cursor to the next row, scroll down if necessary. move cursor to row head
// Next Line
func (p *Parser) handle_NEL() (hd *Handler) {
	hd = &Handler{id: ESC_NEL, ch: p.ch, sequence: p.historyString()}
	hd.handle = func(emu *Emulator) {
		hdl_esc_nel(emu)
	}

	p.setState(InputState_Normal)
	return hd
}

// reset the screen
// Reset to Initial State
func (p *Parser) handle_RIS() (hd *Handler) {
	p.setCompatLevel(CompatLevel_VT400)

	hd = &Handler{id: ESC_RIS, ch: p.ch, sequence: p.historyString()}
	hd.handle = func(emu *Emulator) {
		hdl_esc_ris(emu)
	}

	p.setState(InputState_Normal)
	return hd
}

// Save Cursor and Attributes
func (p *Parser) handle_DECSC() (hd *Handler) {
	hd = &Handler{id: ESC_DECSC, ch: p.ch, sequence: p.historyString()}
	hd.handle = func(emu *Emulator) {
		hdl_esc_decsc(emu)
	}

	p.setState(InputState_Normal)
	return hd
}

// Restore Cursor and Attributes
func (p *Parser) handle_DECRC() (hd *Handler) {
	hd = &Handler{id: ESC_DECRC, ch: p.ch, sequence: p.historyString()}
	hd.handle = func(emu *Emulator) {
		hdl_esc_decrc(emu)
	}

	p.setState(InputState_Normal)
	return hd
}

// fill the screen with 'E'
// DEC Alignment Pattern Generator
func (p *Parser) handle_DECALN() (hd *Handler) {
	hd = &Handler{id: ESC_DECALN, ch: p.ch, sequence: p.historyString()}
	hd.handle = func(emu *Emulator) {
		hdl_esc_decaln(emu)
	}

	p.setState(InputState_Normal)
	return hd
}

// ESC ( C   Designate G0 Character Set, VT100, ISO 2022.
// ESC ) C   Designate G1 Character Set, ISO 2022, VT100
// ESC * C   Designate G2 Character Set, ISO 2022, VT220.
// ESC + C   Designate G3 Character Set, ISO 2022, VT220.
// ESC - C   Designate G1 Character Set, VT300.
// ESC . C   Designate G2 Character Set, VT300.
// ESC / C   Designate G3 Character Set, VT300.
// https://invisible-island.net/xterm/ctlseqs/ctlseqs.html#h3-Controls-beginning-with-ESC
func (p *Parser) handle_ESC_DCS() (hd *Handler) {
	// util.Log.Debug("Designate Character Set",
	// 	"scsDst", p.scsDst,
	// 	"scsMod", p.scsMod,
	// 	"ch", p.ch)

	index := 0
	charset96 := false

	switch p.scsDst {
	case '(':
		index = 0
	case ')':
		index = 1
	case '*':
		index = 2
	case '+':
		index = 3
	case '-':
		index = 1
		charset96 = true
	case '.':
		index = 2
		charset96 = true
	case '/':
		index = 3
		charset96 = true
	}

	var charset *map[byte]rune = nil

	// final byte is p.ch
	switch p.ch {
	case 'A':
		if charset96 {
			charset = &vt_ISO_8859_1 // Charset_IsoLatin1
		} else {
			charset = &vt_ISO_UK // Charset_IsoUK
		}
	case 'B':
		charset = nil // Charset_UTF8
	case '0':
		charset = &vt_DEC_Special // Charset_DecSpec
	case '5':
		if p.scsMod == '%' {
			charset = &vt_DEC_Supplement // Charset_DecSuppl
		}
	case '<':
		charset = &vt_DEC_Supplement // Charset_DecUserPref
	case '>':
		charset = &vt_DEC_Technical // Charset_DecTechn
	}

	hd = &Handler{id: ESC_DCS, ch: p.ch, sequence: p.historyString()}
	hd.handle = func(emu *Emulator) {
		hdl_esc_dcs(emu, index, charset)
	}

	// if any charset is not UTF-8, go back to vt100 mode
	if charset != nil {
		p.vtMode = true
	}

	p.setState(InputState_Normal)
	return hd
}

// Set Mode
func (p *Parser) handle_SM() (hd *Handler) {
	params := p.copyArgs()

	hd = &Handler{id: CSI_SM, ch: p.ch, sequence: p.historyString()}
	hd.handle = func(emu *Emulator) {
		hdl_csi_sm(emu, params)
	}

	p.setState(InputState_Normal)
	return hd
}

// Reset Mode
func (p *Parser) handle_RM() (hd *Handler) {
	params := p.copyArgs()

	hd = &Handler{id: CSI_RM, ch: p.ch, sequence: p.historyString()}
	hd.handle = func(emu *Emulator) {
		hdl_csi_rm(emu, params)
	}

	p.setState(InputState_Normal)
	return hd
}

// Set Mode (private)
// csi_privSM
func (p *Parser) handle_privSM() (hd *Handler) {
	params := p.copyArgs()
	p.setCompatLevel(CompatLevel_VT400, params...)

	hd = &Handler{id: CSI_privSM, ch: p.ch, sequence: p.historyString()}
	hd.handle = func(emu *Emulator) {
		hdl_csi_privSM(emu, params)
	}

	p.setState(InputState_Normal)
	return hd
}

// Reset Mode (private)
// csi_privRM
func (p *Parser) handle_privRM() (hd *Handler) {
	params := p.copyArgs()
	p.setCompatLevel(CompatLevel_VT52, params...)

	hd = &Handler{id: CSI_privRM, ch: p.ch, sequence: p.historyString()}
	hd.handle = func(emu *Emulator) {
		hdl_csi_privRM(emu, params)
	}

	p.setState(InputState_Normal)
	return hd
}

func (p *Parser) handle_DECRQM() (hd *Handler) {
	params := p.copyArgs()
	if params == nil { // default value is 0
		params = []int{0}
	}

	if params[0] == 2026 {
		hd = &Handler{id: CSI_DECRQM, ch: p.ch, sequence: p.historyString()}
		hd.handle = func(emu *Emulator) {
			hdl_csi_decrqm(emu, params)
		}
	} else {
		util.Logger.Warn("unimplemented DECRQM", "seq", p.historyString())
	}

	p.setState(InputState_Normal)
	return hd
}

func (p *Parser) handle_CSI_U_query() (hd *Handler) {
	params := p.copyArgs()

	hd = &Handler{id: CSI_U_QUERY, ch: p.ch, sequence: p.historyString()}
	hd.handle = func(emu *Emulator) {
		hdl_csi_u_query(emu, params)
	}

	p.setState(InputState_Normal)
	return hd
}

func (p *Parser) handle_CSI_U_set() (hd *Handler) {
	flags := p.getPs(0, 0)
	mode := p.getPs(1, 1)

	hd = &Handler{id: CSI_U_SET, ch: p.ch, sequence: p.historyString()}
	hd.handle = func(emu *Emulator) {
		hdl_csi_u_set(emu, flags, mode)
	}

	p.setState(InputState_Normal)
	return hd
}

func (p *Parser) handle_CSI_U_push() (hd *Handler) {
	flags := p.getPs(0, 0)

	hd = &Handler{id: CSI_U_PUSH, ch: p.ch, sequence: p.historyString()}
	hd.handle = func(emu *Emulator) {
		hdl_csi_u_push(emu, flags)
	}

	p.setState(InputState_Normal)
	return hd
}

func (p *Parser) handle_CSI_U_pop() (hd *Handler) {
	count := p.getPs(0, 1)

	hd = &Handler{id: CSI_U_POP, ch: p.ch, sequence: p.historyString()}
	hd.handle = func(emu *Emulator) {
		hdl_csi_u_pop(emu, count)
	}

	p.setState(InputState_Normal)
	return hd
}

// Set Top and Bottom Margins
func (p *Parser) handle_DECSTBM() (hd *Handler) {
	params := p.copyArgs()

	hd = &Handler{id: CSI_DECSTBM, ch: p.ch, sequence: p.historyString()}
	hd.handle = func(emu *Emulator) {
		hdl_csi_decstbm(emu, params)
	}

	// reset the state
	p.setState(InputState_Normal)
	return hd
}

// DEC Soft Terminal Reset
func (p *Parser) handle_DECSTR() (hd *Handler) {
	hd = &Handler{id: CSI_DECSTR, ch: p.ch, sequence: p.historyString()}
	hd.handle = func(emu *Emulator) {
		hdl_csi_decstr(emu)
	}

	// reset the state
	p.setState(InputState_Normal)
	return hd
}

// Device Control String
func (p *Parser) handle_DCS() (hd *Handler) {
	// reset the state
	defer p.setState(InputState_Normal)

	arg := p.getArg()

	if strings.HasPrefix(arg, "$q") { // only process DECRQSS
		if arg == "$q\"p" {
			p.setCompatLevel(CompatLevel_VT400)
		}
		hd = &Handler{id: DCS_DECRQSS, ch: p.ch, sequence: p.historyString()}
		hd.handle = func(emu *Emulator) {
			hdl_dcs_decrqss(emu, arg)
		}
	} else if strings.HasPrefix(arg, "+q") {
		hd = &Handler{id: DCS_XTGETTCAP, ch: p.ch, sequence: p.historyString()}
		hd.handle = func(emu *Emulator) {
			hdl_dcs_xtgettcap(emu, arg[2:])
		}
	} else {
		util.Logger.Warn("DCS", "unimplement", "DCS", "arg", arg, "seq", p.historyString())
	}

	return hd
}

// disambiguation SLRM and SCOSC
// SLRM: Set Left and Right Margins
// SCOSC: Save Cursor Position for SCO console
func (p *Parser) handle_SLRM_SCOSC() (hd *Handler) {
	params := p.copyArgs()

	hd = &Handler{id: CSI_SLRM_SCOSC, ch: p.ch, sequence: p.historyString()}
	hd.handle = func(emu *Emulator) {
		hdl_csi_slrm_scosc(emu, params)
	}

	p.setState(InputState_Normal)
	return hd
}

// SCORC: Restore Cursor Position for SCO console
func (p *Parser) handle_SCORC() (hd *Handler) {
	hd = &Handler{id: CSI_SCORC, ch: p.ch, sequence: p.historyString()}
	hd.handle = func(emu *Emulator) {
		hdl_csi_scorc(emu)
	}

	p.setState(InputState_Normal)
	return hd
}

// DEC Set Compatibility Level
func (p *Parser) handle_DECSCL() (hd *Handler) {
	params := p.copyArgs()

	if len(params) > 0 {
		p.setCompatLevel(sclCompatLevel(params[0]))
	}
	hd = &Handler{id: CSI_DECSCL, ch: p.ch, sequence: p.historyString()}
	hd.handle = func(emu *Emulator) {
		hdl_csi_decscl(emu, params)
	}
	p.setState(InputState_Normal)
	return hd
}

// set cursor style
func (p *Parser) handle_DECSCUSR() (hd *Handler) {
	arg := p.getPs(0, 1)

	hd = &Handler{id: CSI_DECSCUSR, ch: p.ch, sequence: p.historyString()}
	hd.handle = func(emu *Emulator) {
		hdl_csi_decscusr(emu, arg)
	}

	p.setState(InputState_Normal)
	return hd
}

// Insert Column
func (p *Parser) handle_DECIC() (hd *Handler) {
	num := p.getPs(0, 1)

	hd = &Handler{id: CSI_DECIC, ch: p.ch, sequence: p.historyString()}
	hd.handle = func(emu *Emulator) {
		hdl_csi_decic(emu, num)
	}

	p.setState(InputState_Normal)
	return hd
}

// Delete Column
func (p *Parser) handle_DECDC() (hd *Handler) {
	num := p.getPs(0, 1)

	hd = &Handler{id: CSI_DECDC, ch: p.ch, sequence: p.historyString()}
	hd.handle = func(emu *Emulator) {
		hdl_csi_decdc(emu, num)
	}

	p.setState(InputState_Normal)
	return hd
}

// Shift Left
func (p *Parser) handle_ecma48_SL() (hd *Handler) {
	arg := p.getPs(0, 1)

	hd = &Handler{id: CSI_ECMA48_SL, ch: p.ch, sequence: p.historyString()}
	hd.handle = func(emu *Emulator) {
		hdl_csi_ecma48_SL(emu, arg)
	}

	p.setState(InputState_Normal)
	return hd
}

// Shift Right
func (p *Parser) handle_ecma48_SR() (hd *Handler) {
	arg := p.getPs(0, 1)

	hd = &Handler{id: CSI_ECMA48_SR, ch: p.ch, sequence: p.historyString()}
	hd.handle = func(emu *Emulator) {
		hdl_csi_ecma48_SR(emu, arg)
	}

	p.setState(InputState_Normal)
	return hd
}

// Cursor Next Line
func (p *Parser) handle_CNL() (hd *Handler) {
	arg := p.getPs(0, 1)

	hd = &Handler{id: CSI_CNL, ch: p.ch, sequence: p.historyString()}
	hd.handle = func(emu *Emulator) {
		hdl_csi_cnl(emu, arg)
	}

	p.setState(InputState_Normal)
	return hd
}

// Cursor Previous Line
func (p *Parser) handle_CPL() (hd *Handler) {
	arg := p.getPs(0, 1)

	hd = &Handler{id: CSI_CPL, ch: p.ch, sequence: p.historyString()}
	hd.handle = func(emu *Emulator) {
		hdl_csi_cpl(emu, arg)
	}

	p.setState(InputState_Normal)
	return hd
}

// Back Index
func (p *Parser) handle_BI() (hd *Handler) {
	hd = &Handler{id: ESC_BI, ch: p.ch, sequence: p.historyString()}
	hd.handle = func(emu *Emulator) {
		hdl_esc_bi(emu)
	}

	p.setState(InputState_Normal)
	return hd
}

// Forward Index
func (p *Parser) handle_FI() (hd *Handler) {
	hd = &Handler{id: ESC_FI, ch: p.ch, sequence: p.historyString()}
	hd.handle = func(emu *Emulator) {
		hdl_esc_fi(emu)
	}

	p.setState(InputState_Normal)
	return hd
}

// set keypad mode to application
func (p *Parser) handle_DECKPAM() (hd *Handler) {
	hd = &Handler{id: ESC_DECKPAM, ch: p.ch, sequence: p.historyString()}
	hd.handle = func(emu *Emulator) {
		hdl_esc_deckpam(emu)
	}

	p.setState(InputState_Normal)
	return hd
}

// set keypad mode to normal
func (p *Parser) handle_DECKPNM() (hd *Handler) {
	hd = &Handler{id: ESC_DECKPNM, ch: p.ch, sequence: p.historyString()}
	hd.handle = func(emu *Emulator) {
		hdl_esc_deckpnm(emu)
	}

	p.setState(InputState_Normal)
	return hd
}

// set compatibility level, also update the Parser.compatLevel field.
func (p *Parser) handle_DECANM(cl CompatibilityLevel) (hd *Handler) {
	p.setCompatLevel(cl)

	hd = &Handler{id: ESC_DECANM, ch: p.ch, sequence: p.historyString()}
	hd.handle = func(emu *Emulator) {
		hdl_esc_decanm(emu, cl)
	}

	p.setState(InputState_Normal)
	return hd
}

// Xterm window operations
// CSI Ps ; Ps ; Ps t
func (p *Parser) handle_XTWINOPS() (hd *Handler) {
	params := p.copyArgs()

	hd = &Handler{id: CSI_XTWINOPS, ch: p.ch, sequence: p.historyString()}
	hd.handle = func(emu *Emulator) {
		hdl_csi_xtwinops(emu, params, hd.sequence)
	}
	p.setState(InputState_Normal)
	return hd
}

// Xterm key modifier options
func (p *Parser) handle_XTMODKEYS() (hd *Handler) {
	// prepare the parameters
	params := p.copyArgs()

	hd = &Handler{id: CSI_XTMODKEYS, ch: p.ch, sequence: p.historyString()}
	hd.handle = func(emu *Emulator) {
		hdl_csi_xtmodkeys(emu, params)
	}
	p.setState(InputState_Normal)
	return hd
}

// Repeat last graphic character
func (p *Parser) handle_REP() (hd *Handler) {
	arg := p.getPs(0, 1)

	// copy the last graphic character
	chs := make([]rune, len(p.lastChs))
	copy(chs[0:], p.lastChs[0:])

	hd = &Handler{id: CSI_REP, ch: p.ch, sequence: p.historyString()}
	hd.handle = func(emu *Emulator) {
		hdl_csi_rep(emu, arg, chs)
	}

	p.setState(InputState_Normal)
	return hd
}

// VT52: Enter Graphics Mode (ESC F)
func (p *Parser) handle_EGM() (hd *Handler) {
	hd = &Handler{id: VT52_EGM, ch: p.ch, sequence: p.historyString()}
	hd.handle = func(emu *Emulator) {
		hdl_vt52_egm(emu)
	}
	p.setState(InputState_Normal)
	return hd
}

// VT52: The Identify (ESC Z)
func (p *Parser) handle_ID() (hd *Handler) {
	hd = &Handler{id: VT52_ID, ch: p.ch, sequence: p.historyString()}
	hd.handle = func(emu *Emulator) {
		hdl_vt52_id(emu)
	}
	// Do NOT reset the state
	return hd
}

func (p *Parser) handle_MouseTrack(press bool) (hd *Handler) {
	params := p.copyArgs()

	hd = &Handler{id: CSI_MOUSETRACK, ch: p.ch, sequence: p.historyString()}
	hd.handle = func(emu *Emulator) {
		hdl_csi_mousetrack(emu, press, params)
	}

	p.setState(InputState_Normal)
	return hd
}

func (p *Parser) ProcessStream(seq string) []*Handler {
	hds := make([]*Handler, 0, 16)

	if len(seq) == 0 {
		return hds
	}
	hds = p.processStream(seq, hds)
	return hds
}

// process data stream from outside. for VT mode, character set can be changed
// according to control sequences. for UTF-8 mode, no need to change character set.
// the result is a *Handler list. waiting to be executed later.
func (p *Parser) processStream(str string, hds []*Handler) []*Handler {
	var input []rune
	var hd *Handler
	end := false

	for !end {
		if p.vtMode {
			// handle raw byte for VT mode
			for i := 0; i < len(str); i++ {
				input = make([]rune, 1)
				input[0] = rune(str[i]) // note the type conversion will promote 0x9c to 0x009c
				hd = p.ProcessInput(input[0])
				if hd != nil {
					hds = append(hds, hd)
				}
				if i == len(str)-1 {
					end = true
				}
				if !p.vtMode { // switch to utf-8 mode
					str = str[i+1:]
					break
				}
			}
		} else {
			// handle multi rune for modern UTF-8
			graphemes := uniseg.NewGraphemes(str)
			for graphemes.Next() {
				input = graphemes.Runes()

				if len(input) == 2 && input[0] == '\u000D' && input[1] == '\u000A' {
					// special case for CR+LF
					hds = append(hds, p.ProcessInput(input[0]))
					hds = append(hds, p.ProcessInput(input[1]))
				} else {
					hd = p.ProcessInput(input...)
					if hd != nil {
						hds = append(hds, hd)
					}
				}

				_, to := graphemes.Positions()

				if to == len(str) {
					end = true
				}
				if p.vtMode { // switch to vt100 mode
					str = str[to:]
					break
				}
			}
		}
	}
	return hds
}

// process each rune. must apply the UTF-8 decoder to the incoming byte
// stream before interpreting any control characters.
// ref: https://docs.microsoft.com/en-us/windows/console/console-virtual-terminal-sequences
func (p *Parser) ProcessInput(chs ...rune) (hd *Handler) {
	var ch rune

	defer func() {
		if hd != nil {
			p.handleReady = true
		}
	}()

	// for multi runes, it should be grapheme.
	if len(chs) > 1 {
		p.chs = chs
		p.replaceHistory(chs...)
		hd = p.handle_Graphemes()
		return hd
	} else if len(chs) == 1 { // it's either grapheme or control sequence
		p.chs = chs
		ch = chs[0]
		if p.handleReady {
			p.resetHistory()
			p.handleReady = false
		}
		p.appendToHistory(ch)
	} else { // empty chs
		return hd
	}

	p.ch = ch

	// p.logT.Printf(" ch=%q,\t nInputOps=%d, inputOps=%2d\n", ch, p.nInputOps, p.inputOps)
	switch p.inputState {
	case InputState_Normal:
		switch ch {
		case '\x00': // ignore NUL
		case '\x1B':
			if p.compatLevel == CompatLevel_VT52 {
				p.setState(InputState_Escape_VT52)
			} else {
				p.setState(InputState_Escape)
			}
			p.inputOps[0] = 0
			p.nInputOps = 1
		case '\x0D': // CR is \r
			p.traceNormalInput()
			hd = p.handle_CR()
		case '\x0C', '\x0B', '\x0A': // FF(\f), VT(\v), LF(\n), they are handled same as IND
			p.traceNormalInput()
			hd = p.handle_IND()
		case '\x09': // HT/TAB is \t
			p.traceNormalInput()
			hd = p.handle_HT()
		case '\x08': // BS is \b
			p.traceNormalInput()
			hd = p.handle_CUB()
		case '\x07': // BEL is \a
			p.traceNormalInput()
			hd = p.handle_BEL()
		case '\x0E':
			p.traceNormalInput()
			hd = p.handle_SO()
		case '\x0F':
			p.traceNormalInput()
			hd = p.handle_SI()
		case '\x05': // ENQ - Enquiry
			p.traceNormalInput()
		default:
			// one stop https://www.cl.cam.ac.uk/~mgk25/unicode.html
			// https://harjit.moe/charsetramble.html
			// need to understand the relationship between utf-8 and  ECMA-35 charset
			p.replaceHistory(ch)
			hd = p.handle_Graphemes()
		}
	case InputState_Escape_VT52:
		// https://github.com/microsoft/terminal/blob/main/doc/specs/%23976%20-%20VT52%20escape%20sequences.md
		switch ch {
		case '\x18', '\x1A': // CAN and SUB interrupts ESC sequence
			p.setState(InputState_Normal)
		case '\x1B': // ESC restarts ESC sequence
			p.inputOps[0] = 0
			p.nInputOps = 1
		case '=':
			hd = p.handle_DECKPAM() // Enter Keypad Mode (ESC =)
		case '>':
			hd = p.handle_DECKPNM() // Exit Keypad Mode (ESC >)
		case '<':
			// Exit VT52 mode. Enter VT100 mode.
			hd = p.handle_DECANM(CompatLevel_VT100) // Enter ANSI Mode (ESC <)
		case 'A':
			hd = p.handle_CUU() // Cursor Up (ESC A)
		case 'B':
			hd = p.handle_CUD() // Cursor Down (ESC B)
		case 'C':
			hd = p.handle_CUF() // Cursor Right (ESC C)
		case 'D':
			hd = p.handle_CUB() // Cursor Left (ESC D)
		case 'F':
			hd = p.handle_EGM() // Enter Graphics Mode (ESC F)
		case 'G':
			hd = p.handle_DOCS_UTF8() //  Exit Graphics Mode (ESC G)
		case 'H':
			hd = p.handle_CUP() // Cursor Home (ESC H)
		case 'I':
			hd = p.handle_RI() // Reverse Line Feed (ESC I)
		case 'J':
			hd = p.handle_ED() // Erase to End of Display (ESC J)
		case 'K':
			hd = p.handle_EL() // Erase to End of Line (ESC K)
		case 'Y':
			p.setState(InputState_VT52_CUP_Arg1) // Direct Cursor Address (ESC Y)
		case 'Z':
			hd = p.handle_ID() // The Identify (ESC Z) should be "\e/Z";
		case 'c':
			hd = p.handle_RIS() // allow "reset" command to escape VT52
		default:
			p.unhandledInput()
		}
	case InputState_VT52_CUP_Arg1:
		// ESC Y line# column#
		// for "line#", the host send the octal code 040 to specifiy the top line of the screen.
		// for "column#", the host send the octal code 040 to specifiy the leftmost column in a line
		p.inputOps[0] = int(ch) - 31
		p.setState(InputState_VT52_CUP_Arg2)
	case InputState_VT52_CUP_Arg2:
		p.inputOps[1] = int(ch) - 31
		p.nInputOps = 2
		hd = p.handle_CUP()
	case InputState_Escape:
		switch ch {
		case '\x18', '\x1A': // CAN and SUB interrupts ESC sequence
			p.setState(InputState_Normal)
		case '\x1B': // ESC restarts ESC sequence
			p.inputOps[0] = 0
			p.nInputOps = 1
		case ' ':
			p.setState(InputState_Esc_Space)
		case '#':
			p.setState(InputState_Esc_Hash)
		case '%':
			p.setState(InputState_Esc_Pct)
		case '[':
			p.setState(InputState_CSI)
		case ']':
			p.argBuf.Reset()
			p.setState(InputState_OSC)
		case '(', ')', '*', '+', '-', '.', '/':
			fallthrough
		case ',', '$':
			// from ISO/IEC 2022 (absorbed, treat as no-op)
			// the first byte define the target character set
			p.scsDst = ch
			p.scsMod = 0x00
			p.setState(InputState_Select_Charset)
		case 'D':
			hd = p.handle_IND()
			p.setState(InputState_Normal)
		case 'M':
			hd = p.handle_RI()
		case 'E':
			hd = p.handle_NEL()
		case 'H':
			hd = p.handle_HTS()
		case 'N':
			hd = p.handle_SS2()
		case 'O':
			hd = p.handle_SS3()
		case 'P':
			p.argBuf.Reset()
			p.setState(InputState_DCS)
		case 'c':
			hd = p.handle_RIS()
		case '6':
			hd = p.handle_BI()
		case '7':
			hd = p.handle_DECSC()
		case '8':
			hd = p.handle_DECRC()
		case '9':
			hd = p.handle_FI()
		case '=':
			hd = p.handle_DECKPAM()
		case '>':
			hd = p.handle_DECKPNM()
		case '<':
			hd = p.handle_DECANM(CompatLevel_VT400)
		case '~':
			hd = p.handle_LS1R()
		case 'n':
			hd = p.handle_LS2()
		case '}':
			hd = p.handle_LS2R()
		case 'o':
			hd = p.handle_LS3()
		case '|':
			hd = p.handle_LS3R()
		case '\\': // ignore lone ST
			p.setState(InputState_Normal)
		default:
			p.unhandledInput()
		}
	case InputState_Esc_Space:
		switch ch {
		case 'F':
			util.Logger.Warn("S7C1T: Send 7-bit controls", "unimplement", "ESC space")
			p.setState(InputState_Normal)
		case 'G':
			util.Logger.Warn("S8C1T: Send 8-bit controls", "unimplement", "ESC space")
			p.setState(InputState_Normal)
		case 'L':
			util.Logger.Warn("Set ANSI conformance level 1", "unimplement", "ESC space")
			p.setState(InputState_Normal)
		case 'M':
			util.Logger.Warn("Set ANSI conformance level 2", "unimplement", "ESC space")
			p.setState(InputState_Normal)
		case 'N':
			util.Logger.Warn("Set ANSI conformance level 3", "unimplement", "ESC space")
			p.setState(InputState_Normal)
		default:
			p.unhandledInput()
		}
	case InputState_Esc_Hash:
		switch ch {
		case '3':
			util.Logger.Warn("DECDHL: Double-height, top half", "unimplement", "ESC hash")
			p.setState(InputState_Normal)
		case '4':
			util.Logger.Warn("DECDHL: Double-height, bottom half", "unimplement", "ESC hash")
			p.setState(InputState_Normal)
		case '5':
			util.Logger.Warn("DECSWL: Single-width line", "unimplement", "ESC hash")
			p.setState(InputState_Normal)
		case '6':
			util.Logger.Warn("DECDWL: Double-width line", "unimplement", "ESC hash")
			p.setState(InputState_Normal)
		case '8':
			hd = p.handle_DECALN()
		default:
			p.unhandledInput()
		}
	case InputState_Esc_Pct:
		switch ch {
		case '@':
			util.Logger.Debug("Select charset: default (ISO-8859-1)")
			hd = p.handle_DOCS_ISO8859_1()
		case 'G':
			util.Logger.Debug("Select charset: UTF-8")
			hd = p.handle_DOCS_UTF8()
		default:
			p.unhandledInput()
		}
	case InputState_Select_Charset:
		if ch < 0x30 {
			// save the second byte, that means there should be third byte next.
			p.scsMod = ch
		} else {
			// the second byte or the third byte
			hd = p.handle_ESC_DCS()
		}
	case InputState_CSI:
		if p.collectNumericParameters(ch) {
			break
		}
		switch ch {
		case '\x1B':
			p.setState(InputState_Normal)
		case 'A':
			hd = p.handle_CUU()
		case 'B':
			hd = p.handle_CUD()
		case 'C':
			hd = p.handle_CUF()
		case 'D':
			hd = p.handle_CUB()
		case 'E':
			hd = p.handle_CNL()
		case 'F':
			hd = p.handle_CPL()
		case 'G':
			hd = p.handle_CHA()
		case 'H':
			hd = p.handle_CUP()
		case 'I':
			hd = p.handle_CHT_Focus()
		case 'J':
			hd = p.handle_ED()
		case 'K':
			hd = p.handle_EL()
		case 'L':
			hd = p.handle_IL()
		case 'M':
			hd = p.handle_DL()
		case 'O':
			hd = p.handle_Focus(false)
		case 'P':
			hd = p.handle_DCH()
		case 'S':
			hd = p.handle_SU()
		case 'T', '^':
			hd = p.handle_SD() // ^ is the same as T
		case 'X':
			hd = p.handle_ECH()
		case 'Z':
			hd = p.handle_CBT()
		case '@':
			hd = p.handle_ICH()
		case '`':
			hd = p.handle_HPA()
		case 'a':
			hd = p.handle_HPR()
		case 'b':
			hd = p.handle_REP()
		case 'c':
			hd = p.handle_priDA()
		case 'd':
			hd = p.handle_VPA()
		case 'e':
			hd = p.handle_VPR()
		case 'g':
			hd = p.handle_TBC()
		case 'h':
			hd = p.handle_SM()
		case 'l':
			hd = p.handle_RM()
		case 'm':
			hd = p.handle_SGR()
		case 'n':
			hd = p.handle_DSR()
		case 'r':
			hd = p.handle_DECSTBM()
		case 's':
			hd = p.handle_SLRM_SCOSC()
		case 't':
			hd = p.handle_XTWINOPS()
		case 'u':
			hd = p.handle_SCORC()
		case '\'':
			p.setState(InputState_CSI_Quote)
		case '"':
			p.setState(InputState_CSI_DblQuote)
		case '!':
			p.setState(InputState_CSI_Bang)
		case '?':
			p.setState(InputState_CSI_Priv)
		case ' ':
			p.setState(InputState_CSI_SPC)
		case '>':
			p.setState(InputState_CSI_GT)
		case '<':
			p.setState(InputState_CSI_LT)
		case '=':
			p.setState(InputState_CSI_Equal)
		case '\x07': // BEL is ignored \a in c++
		case '\x08': // BS is \b
			// undo last character in CSI sequence:
			if p.getHistoryAt(1) == ';' {
				p.nInputOps -= 1
			} else {
				p.inputOps[p.nInputOps-1] /= 10
			}
		case '\x09': // HT/TAB is \t
			hd = p.handle_HT()
			p.setState(InputState_CSI)
		case '\x0D': // CR is \r
			hd = p.handle_CR()
			p.setState(InputState_CSI)
		case '\x0C', '\x0B': // FF is \f, VT is \v
			hd = p.handle_IND()
			p.setState(InputState_CSI)
		default:
			p.unhandledInput()
		}
	case InputState_CSI_Equal:
		if p.collectNumericParameters(ch) {
			break
		}
		switch ch {
		case 'u':
			hd = p.handle_CSI_U_set()
		default:
			p.unhandledInput()
		}
	case InputState_CSI_Bang:
		switch ch {
		case 'p':
			hd = p.handle_DECSTR()
		default:
			p.unhandledInput()
		}
	case InputState_CSI_Quote:
		switch ch {
		case '}':
			hd = p.handle_DECIC()
		case '~':
			hd = p.handle_DECDC()
		default:
			p.unhandledInput()
		}
	case InputState_CSI_DblQuote:
		switch ch {
		case 'p':
			hd = p.handle_DECSCL()
		default:
			p.unhandledInput()
		}
	case InputState_CSI_SPC:
		switch ch {
		case '@':
			hd = p.handle_ecma48_SL()
		case 'A':
			hd = p.handle_ecma48_SR()
		case 'q':
			hd = p.handle_DECSCUSR()
		default:
			p.unhandledInput()
		}
	case InputState_CSI_GT:
		if p.collectNumericParameters(ch) {
			break
		}
		switch ch {
		case 'c':
			hd = p.handle_secDA()
		case 'm':
			hd = p.handle_XTMODKEYS()
		case 'u':
			hd = p.handle_CSI_U_push()
		default:
			p.unhandledInput()
		}
	case InputState_CSI_LT:
		if p.collectNumericParameters(ch) {
			break
		}
		switch ch {
		case 'M':
			hd = p.handle_MouseTrack(true)
		case 'm':
			hd = p.handle_MouseTrack(false)
		case 'u':
			hd = p.handle_CSI_U_pop()
		default:
			p.unhandledInput()
		}
	case InputState_CSI_Priv:
		if p.collectNumericParameters(ch) {
			break
		}
		switch ch {
		case '\x1B':
			p.setState(InputState_Normal)
		case 'h':
			hd = p.handle_privSM() // DECSET
		case 'l':
			hd = p.handle_privRM() // DECRST
		case 'u':
			hd = p.handle_CSI_U_query()
		case '$':
			p.argBuf.WriteRune(ch)
		case 'p':
			p.argBuf.WriteRune(ch)
			if p.argBuf.String() == "$p" {
				hd = p.handle_DECRQM()
			} else {
				p.unhandledInput()
			}
			p.argBuf.Reset()
		default:
			p.unhandledInput()
		}
	case InputState_DCS:
		switch ch {
		case '\x1B':
			p.setState(InputState_DCS_Esc)
		default:
			if p.argBuf.Len() < 4095 {
				p.argBuf.WriteRune(ch)
			} else {
				util.Logger.Error("OSC argument string overflow (>4095)", "argBuf", p.argBuf.String()[:64])
				p.setState(InputState_Normal)
			}
		}
	case InputState_DCS_Esc:
		switch ch {
		case '\\':
			hd = p.handle_DCS()
			p.argBuf.Reset()
		default:
			p.argBuf.WriteRune('\x1B')
			p.argBuf.WriteRune(ch)
			p.setState(InputState_DCS)
		}
	case InputState_OSC:
		switch ch {
		case '\x07': // final byte = BEL
			hd = p.handle_OSC()
			p.argBuf.Reset()
		case '\x1B':
			p.setState(InputState_OSC_Esc)
		default:
			if p.argBuf.Len() < 4095 {
				p.argBuf.WriteRune(ch)
			} else {
				util.Logger.Error("OSC argument string overflow (>4095)", "argBuf", p.argBuf.String())
				p.setState(InputState_Normal)
			}
		}
	case InputState_OSC_Esc:
		switch ch {
		case '\\': // ESC \ : ST
			hd = p.handle_OSC()
			p.argBuf.Reset()
		default:
			p.argBuf.WriteRune('\x1B')
			p.argBuf.WriteRune(ch)
			p.setState(InputState_OSC)
		}
	}
	return hd
}
