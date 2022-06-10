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
	"container/list"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/rivo/uniseg"
)

const (
	InputState_Normal = iota
	InputState_Escape
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
	InputState_DCS
	InputState_DCS_Esc
	InputState_OSC
	InputState_OSC_Esc
)

var strInputState = [...]string{
	"Normal",
	"Escape",
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
	"DCS",
	"DCS_Esc",
	"OSC",
	"OSC_Esc",
}

type Parser struct {
	// state State

	// parsing error
	perror error

	// big switch state machine
	inputState int
	ch         rune
	chs        []rune

	// numeric parameters
	inputOps  []int
	nInputOps int
	maxEscOps int

	// history, up to last 5 rune
	history *list.List

	// various indicators
	readPos         int
	lastEscBegin    int
	lastNormalBegin int
	lastStopPos     int

	// string parameter
	argBuf strings.Builder

	// select character set destination and mode
	scsDst rune
	scsMod rune

	// G0~G3 character set compatiable mode, default false
	vtMode bool

	// logger
	logE     *log.Logger
	logT     *log.Logger
	logU     *log.Logger
	logTrace bool
}

func NewParser() *Parser {
	p := new(Parser)

	// TODO consider to rotate the log file and limit the log file size.
	// file, err := os.OpenFile("aprish.logs.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	// if err != nil {
	//     log.Fatal(err)
	// }
	p.logT = log.New(os.Stderr, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	p.logE = log.New(os.Stderr, "ERRO: ", log.Ldate|log.Ltime|log.Lshortfile)
	p.logU = log.New(os.Stderr, "(Uimplemented): ", log.Ldate|log.Ltime|log.Lshortfile)

	p.reset()
	return p
}

// add rune to the history cache, store max 5 recent runes.
func (p *Parser) appendToHistory(r rune) {
	p.history.PushBack(r)

	if p.history.Len() > 5 {
		p.history.Remove(p.history.Front())
	}
}

// get the rune value in the specified reverse index
// return 0 if error.
func (p *Parser) getRuneAt(reverseIdx int) (r rune) {
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

func (p *Parser) reset() {
	p.inputState = InputState_Normal
	p.ch = 0x00

	p.perror = nil

	p.maxEscOps = 16
	p.inputOps = make([]int, p.maxEscOps)
	p.nInputOps = 0
	p.argBuf.Reset()

	p.history = list.New()
	p.scsDst = 0x00
	p.scsMod = 0x00
}

// trace the input if logTrace is true
func (p *Parser) traceNormalInput() {
	if p.logTrace {
		p.logT.Printf("Input:%q inputOps=%d, nInputOps=%d, argBuf=%q\n",
			p.chs, p.inputOps, p.nInputOps, p.argBuf.String())
	}
}

// log the unhandled input and reset the state to normal
func (p *Parser) unhandledInput() {
	p.logU.Printf("Unhandled input:%q state=%s, inputOps=%d, nInputOps=%d, argBuf=%q\n",
		p.ch, strInputState[p.inputState], p.inputOps, p.nInputOps, p.argBuf.String())

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
		p.lastNormalBegin = p.readPos + 1
	} else if p.inputState == InputState_Normal {
		p.traceNormalInput()
	}

	p.inputState = newState
}

// collect numeric parameter and stor them in inputOps array.
func (p *Parser) collectNumericParameters(ch rune) (isBreak bool) {
	if '0' <= ch && ch <= '9' {
		isBreak = true
		// max value for numeric parameter
		p.inputOps[p.nInputOps-1] *= 10
		p.inputOps[p.nInputOps-1] += int(ch - '0')
		if p.inputOps[p.nInputOps-1] >= 65535 {
			// TODO consider how to consume the extra rune
			p.logE.Printf("the number is too big: > 65535, %d", p.inputOps[p.nInputOps-1])
			p.perror = fmt.Errorf("the number is too big. %d", p.inputOps[p.nInputOps-1])
			p.setState(InputState_Normal)
		}
	} else if ch == ';' || ch == ':' {
		isBreak = true
		if p.nInputOps < p.maxEscOps { // move to the next parameter
			p.inputOps[p.nInputOps] = 0
			p.nInputOps += 1
		} else {
			p.logE.Printf("inputOps full, increase maxEscOps. %d", p.inputOps)
			p.perror = fmt.Errorf("the parameters count limitation is over. %d", p.maxEscOps)
			p.setState(InputState_Normal)
		}
	}
	return isBreak
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

// print graphemes on screen
func (p *Parser) handle_Graphemes() (hd *Handler) {
	hd = &Handler{name: "graphemes", ch: p.ch}

	r := p.chs
	hd.handle = func(emu *emulator) {
		hdl_graphemes(emu, r...)
	}
	return hd
}

// Cursor up by <n>
func (p *Parser) handle_CUU() (hd *Handler) {
	num := p.getPs(0, 1)

	hd = &Handler{name: "csi-cuu", ch: p.ch}
	hd.handle = func(emu *emulator) {
		hdl_csi_cuu(emu, num)
	}

	p.setState(InputState_Normal)
	return hd
}

// Cursor down by <n>
func (p *Parser) handle_CUD() (hd *Handler) {
	num := p.getPs(0, 1)

	hd = &Handler{name: "csi-cud", ch: p.ch}
	hd.handle = func(emu *emulator) {
		hdl_csi_cud(emu, num)
	}

	p.setState(InputState_Normal)
	return hd
}

// Cursor forward (Right) by <n>
func (p *Parser) handle_CUF() (hd *Handler) {
	num := p.getPs(0, 1)

	hd = &Handler{name: "csi-cuf", ch: p.ch}
	hd.handle = func(emu *emulator) {
		hdl_csi_cuf(emu, num)
	}

	p.setState(InputState_Normal)
	return hd
}

// Cursor backward (Left) by <n>
func (p *Parser) handle_CUB() (hd *Handler) {
	num := p.getPs(0, 1)

	hd = &Handler{name: "csi-cub", ch: p.ch}
	hd.handle = func(emu *emulator) {
		hdl_csi_cub(emu, num)
	}

	p.setState(InputState_Normal)
	return hd
}

// prepare parameters for the CUP
// Cursor moves to <row>; <col> coordinate within the viewport
func (p *Parser) handle_CUP() (hd *Handler) {
	row := p.getPs(0, 1)
	col := p.getPs(1, 1)

	hd = &Handler{name: "csi-cup", ch: p.ch}
	hd.handle = func(emu *emulator) {
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
// TODO rewrite the p.perror.
func (p *Parser) handle_OSC() (hd *Handler) {
	// Here we parse the parameters by ourselves.
	cmd := 0
	arg := p.getArg()

	defer p.setState(InputState_Normal)

	// get the Ps
	pos := strings.Index(arg, ";")
	if pos == -1 {
		p.perror = fmt.Errorf("OSC: no ';' exist. %q", arg)
		return
	}
	var err error
	if cmd, err = strconv.Atoi(arg[:pos]); err != nil {
		p.perror = fmt.Errorf("OSC: illegal Ps parameter. %q", arg[:pos])
		return
	}

	// get the Pt
	arg = arg[pos+1:]
	if cmd < 0 || cmd > 120 {
		p.logT.Printf("OSC: malformed command string %d %q\n", cmd, arg)
	} else {
		switch cmd {
		// create the ActOn
		case 0, 1, 2:
			hd = &Handler{name: "osc-0,1,2", ch: p.ch}
			hd.handle = func(emu *emulator) {
				hdl_osc_0_1_2(emu, cmd, arg)
			}
		case 4:
			hd = &Handler{name: "osc-4", ch: p.ch}
			hd.handle = func(emu *emulator) {
				hdl_osc_4(emu, cmd, arg)
			}
		case 52:
			hd = &Handler{name: "osc-52", ch: p.ch}
			hd.handle = func(emu *emulator) {
				hdl_osc_52(emu, cmd, arg)
			}
		case 10, 11, 12, 17, 19:
			hd = &Handler{name: "osc-10,11,12,17,19", ch: p.ch}
			hd.handle = func(emu *emulator) {
				hdl_osc_10x(emu, cmd, arg)
			}
		default:
			p.logU.Printf("unhandled OSC: %d %q\n", cmd, arg)
		}
	}

	return hd
}

// Carriage Return
func (p *Parser) handle_CR() (hd *Handler) {
	hd = &Handler{name: "c0-cr", ch: p.ch}
	hd.handle = func(emu *emulator) {
		hdl_c0_cr(emu)
	}
	// Do NOT reset the state
	return hd
}

// Line Feed
func (p *Parser) handle_IND() (hd *Handler) {
	hd = &Handler{name: "c0-lf", ch: p.ch}
	hd.handle = func(emu *emulator) {
		hdl_c0_lf(emu)
	}
	// Do NOT reset the state
	return hd
}

// Horizontal Tab
// move cursor position to next tab stop
func (p *Parser) handle_HT() (hd *Handler) {
	hd = &Handler{name: "c0-ht", ch: p.ch}
	hd.handle = func(emu *emulator) {
		hdl_c0_ht(emu)
	}
	// Do NOT reset the state
	return hd
}

// Bell
func (p *Parser) handle_BEL() (hd *Handler) {
	hd = &Handler{name: "c0-bel", ch: p.ch}
	hd.handle = func(emu *emulator) {
		hdl_c0_bel(emu)
	}
	return hd
}

// SI - switch to standard character set
func (p *Parser) handle_SI() (hd *Handler) {
	hd = &Handler{name: "c0-si", ch: p.ch}
	hd.handle = func(emu *emulator) {
		hdl_c0_si(emu)
	}
	return hd
}

// SO - switch to alternate character set
func (p *Parser) handle_SO() (hd *Handler) {
	hd = &Handler{name: "c0-so", ch: p.ch}
	hd.handle = func(emu *emulator) {
		hdl_c0_so(emu)
	}
	return hd
}

// set cursor position as tab stop position
func (p *Parser) handle_HTS() (hd *Handler) {
	hd = &Handler{name: "esc-hts", ch: p.ch}
	hd.handle = func(emu *emulator) {
		hdl_esc_hts(emu)
	}

	p.setState(InputState_Normal)
	return hd
}

// move cursor forward to the N tab stop position
func (p *Parser) handle_CHT() (hd *Handler) {
	count := p.getPs(0, 1)

	hd = &Handler{name: "csi-cht", ch: p.ch}
	hd.handle = func(emu *emulator) {
		hdl_csi_cht(emu, count)
	}

	p.setState(InputState_Normal)
	return hd
}

// move bcursor ackward to the N tab stop position
func (p *Parser) handle_CBT() (hd *Handler) {
	count := p.getPs(0, 1)

	hd = &Handler{name: "csi-cbt", ch: p.ch}
	hd.handle = func(emu *emulator) {
		hdl_csi_cbt(emu, count)
	}

	p.setState(InputState_Normal)
	return hd
}

// clear tab stop position according to cmd
func (p *Parser) handle_TBC() (hd *Handler) {
	cmd := p.getPs(0, 0)

	hd = &Handler{name: "csi-tbc", ch: p.ch}
	hd.handle = func(emu *emulator) {
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

	hd = &Handler{name: "csi-ich", ch: p.ch}
	hd.handle = func(emu *emulator) {
		hdl_csi_ich(emu, count)
	}

	p.setState(InputState_Normal)
	return hd
}

// CHA—Cursor Horizontal Absolute
// Cursor moves to <n>th position horizontally in the current line
func (p *Parser) handle_CHA_HPA() (hd *Handler) {
	count := p.getPs(0, 1)

	hd = &Handler{name: "csi-cha-hpa", ch: p.ch}
	hd.handle = func(emu *emulator) {
		hdl_csi_cha_hpa(emu, count)
	}

	p.setState(InputState_Normal)
	return hd
}

// This control function erases characters from part or all of the display.
// When you erase complete lines, they become single-height, single-width
// lines, with all visual character attributes cleared. ED works inside or
// outside the scrolling margins.
func (p *Parser) handle_ED() (hd *Handler) {
	cmd := p.getPs(0, 0)

	hd = &Handler{name: "csi-ed", ch: p.ch}
	hd.handle = func(emu *emulator) {
		hdl_csi_ed(emu, cmd)
	}

	p.setState(InputState_Normal)
	return hd
}

// This control function erases characters on the line that has the cursor.
// EL clears all character attributes from erased character positions. EL
// works inside or outside the scrolling margins.
func (p *Parser) handle_EL() (hd *Handler) {
	cmd := p.getPs(0, 0)

	hd = &Handler{name: "csi-el", ch: p.ch}
	hd.handle = func(emu *emulator) {
		hdl_csi_el(emu, cmd)
	}

	p.setState(InputState_Normal)
	return hd
}

// This control function inserts one or more blank lines, starting at the
// cursor.
func (p *Parser) handle_IL() (hd *Handler) {
	lines := p.getPs(0, 1)

	hd = &Handler{name: "csi-il", ch: p.ch}
	hd.handle = func(emu *emulator) {
		hdl_csi_il(emu, lines)
	}

	p.setState(InputState_Normal)
	return hd
}

// This control function deletes one or more lines in the scrolling region,
// starting with the line that has the cursor.
func (p *Parser) handle_DL() (hd *Handler) {
	lines := p.getPs(0, 1)

	hd = &Handler{name: "csi-dl", ch: p.ch}
	hd.handle = func(emu *emulator) {
		hdl_csi_dl(emu, lines)
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

	hd = &Handler{name: "csi-dch", ch: p.ch}
	hd.handle = func(emu *emulator) {
		hdl_csi_dch(emu, cells)
	}

	p.setState(InputState_Normal)
	return hd
}

// This control function moves the user window down a specified number
// of lines in page memory.
// SU got the +lines
// Scroll text up by <n>. Also known as pan down, new lines fill in from the bottom of the screen
func (p *Parser) handle_SU() (hd *Handler) {
	lines := p.getPs(0, 1)

	hd = &Handler{name: "csi-su-sd", ch: p.ch}
	hd.handle = func(emu *emulator) {
		hdl_csi_su_sd(emu, lines)
	}

	p.setState(InputState_Normal)
	return hd
}

// /This control function moves the user window up a specified number
// of lines in page memory.
// SD got the -lines
// Scroll down by <n>. Also known as pan up, new lines fill in from the top of the screen
func (p *Parser) handle_SD() (hd *Handler) {
	lines := p.getPs(0, 1)

	hd = &Handler{name: "csi-su-sd", ch: p.ch}
	hd.handle = func(emu *emulator) {
		hdl_csi_su_sd(emu, -lines)
	}

	p.setState(InputState_Normal)
	return hd
}

// This control function erases one or more characters, from the cursor
// position to the right. ECH clears character attributes from erased
// character positions. ECH works inside or outside the scrolling margins.
func (p *Parser) handle_ECH() (hd *Handler) {
	num := p.getPs(0, 1)

	hd = &Handler{name: "csi-ech", ch: p.ch}
	hd.handle = func(emu *emulator) {
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
func (p *Parser) handle_DA1() (hd *Handler) {
	hd = &Handler{name: "csi-da1", ch: p.ch}
	hd.handle = func(emu *emulator) {
		hdl_csi_da1(emu)
	}

	p.setState(InputState_Normal)
	return hd
}

// In this DA exchange, the host requests the terminal's identification
// code, firmware version level, and hardware options.
func (p *Parser) handle_DA2() (hd *Handler) {
	hd = &Handler{name: "csi-da2", ch: p.ch}
	hd.handle = func(emu *emulator) {
		hdl_csi_da2(emu)
	}

	p.setState(InputState_Normal)
	return hd
}

// VPA causes the active position to be moved to the corresponding horizontal position.
// Cursor moves to the <n>th position vertically in the current column
func (p *Parser) handle_VPA() (hd *Handler) {
	row := p.getPs(0, 1)

	hd = &Handler{name: "csi-vpa", ch: p.ch}
	hd.handle = func(emu *emulator) {
		hdl_csi_vpa(emu, row)
	}

	p.setState(InputState_Normal)
	return hd
}

// select graphics rendition -- e.g., bold, blinking, etc.
func (p *Parser) handle_SGR() (hd *Handler) {
	// prepare the parameters for sgr
	params := make([]int, p.nInputOps)
	copy(params, p.inputOps)

	hd = &Handler{name: "csi-sgr", ch: p.ch}
	hd.handle = func(emu *emulator) {
		hdl_csi_sgr(emu, params)
	}

	p.setState(InputState_Normal)
	return hd
}

// Device Status Reports
// Operating Status: https://www.vt100.net/docs/vt510-rm/DSR-OS.html
// Cursor Position Report: https://www.vt100.net/docs/vt510-rm/DSR-CPR.html
func (p *Parser) handle_DSR() (hd *Handler) {
	cmd := p.getPs(0, 0)

	hd = &Handler{name: "csi-dsr", ch: p.ch}
	hd.handle = func(emu *emulator) {
		hdl_csi_dsr(emu, cmd)
	}

	p.setState(InputState_Normal)
	return hd
}

// ESC N Single Shift Select of G2 Character Set (SS2  is 0x8e), VT220.
func (p *Parser) handle_SS2() (hd *Handler) {
	hd = &Handler{name: "esc-ss2", ch: p.ch}
	hd.handle = func(emu *emulator) {
		hdl_esc_ss2(emu)
	}

	p.setState(InputState_Normal)
	return hd
}

// ESC O Single Shift Select of G3 Character Set (SS3  is 0x8f), VT220.
func (p *Parser) handle_SS3() (hd *Handler) {
	hd = &Handler{name: "esc-ss3", ch: p.ch}
	hd.handle = func(emu *emulator) {
		hdl_esc_ss3(emu)
	}

	p.setState(InputState_Normal)
	return hd
}

// LS1R gr = 1
func (p *Parser) handle_LS1R() (hd *Handler) {
	hd = &Handler{name: "esc-ls1r", ch: p.ch}
	hd.handle = func(emu *emulator) {
		hdl_esc_ls1r(emu)
	}

	p.setState(InputState_Normal)
	return hd
}

// LS2 gl = 2
func (p *Parser) handle_LS2() (hd *Handler) {
	hd = &Handler{name: "esc-ls2", ch: p.ch}
	hd.handle = func(emu *emulator) {
		hdl_esc_ls2(emu)
	}

	p.setState(InputState_Normal)
	return hd
}

// LS2R gr = 2
func (p *Parser) handle_LS2R() (hd *Handler) {
	hd = &Handler{name: "esc-ls2r", ch: p.ch}
	hd.handle = func(emu *emulator) {
		hdl_esc_ls2r(emu)
	}

	p.setState(InputState_Normal)
	return hd
}

// LS3 gl = 3
func (p *Parser) handle_LS3() (hd *Handler) {
	hd = &Handler{name: "esc-ls3", ch: p.ch}
	hd.handle = func(emu *emulator) {
		hdl_esc_ls3(emu)
	}

	p.setState(InputState_Normal)
	return hd
}

// LS3R gr = 3
func (p *Parser) handle_LS3R() (hd *Handler) {
	hd = &Handler{name: "esc-ls3r", ch: p.ch}
	hd.handle = func(emu *emulator) {
		hdl_esc_ls3r(emu)
	}

	p.setState(InputState_Normal)
	return hd
}

// DOCS Select charset: UTF-8
func (p *Parser) handle_DOCS_UTF8() (hd *Handler) {
	hd = &Handler{name: "esc-docs-utf-8", ch: p.ch}
	hd.handle = func(emu *emulator) {
		hdl_esc_docs_utf8(emu)
	}

	p.vtMode = false
	p.setState(InputState_Normal)
	return hd
}

// DOCS Select charset: default (ISO-8859-1)
func (p *Parser) handle_DOCS_ISO8859_1() (hd *Handler) {
	hd = &Handler{name: "esc-docs-iso8859-1", ch: p.ch}
	hd.handle = func(emu *emulator) {
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
	hd = &Handler{name: "esc-ri", ch: p.ch}
	hd.handle = func(emu *emulator) {
		hdl_esc_ri(emu)
	}

	p.setState(InputState_Normal)
	return hd
}

// move cursor to the next row, scroll down if necessary. move cursor to row head
// Next Line
func (p *Parser) handle_NEL() (hd *Handler) {
	hd = &Handler{name: "esc-nel", ch: p.ch}
	hd.handle = func(emu *emulator) {
		hdl_esc_nel(emu)
	}

	p.setState(InputState_Normal)
	return hd
}

// reset the screen
func (p *Parser) handle_RIS() (hd *Handler) {
	hd = &Handler{name: "esc-ris", ch: p.ch}
	hd.handle = func(emu *emulator) {
		hdl_esc_ris(emu)
	}

	p.setState(InputState_Normal)
	return hd
}

// save current cursor
func (p *Parser) handle_DECSC() (hd *Handler) {
	hd = &Handler{name: "esc-decsc", ch: p.ch}
	hd.handle = func(emu *emulator) {
		hdl_esc_decsc(emu)
	}

	p.setState(InputState_Normal)
	return hd
}

// restore saved cursor
func (p *Parser) handle_DECRC() (hd *Handler) {
	hd = &Handler{name: "esc-decrc", ch: p.ch}
	hd.handle = func(emu *emulator) {
		hdl_esc_decrc(emu)
	}

	p.setState(InputState_Normal)
	return hd
}

// fill the screen with 'E'
func (p *Parser) handle_DECALN() (hd *Handler) {
	hd = &Handler{name: "esc-decaln", ch: p.ch}
	hd.handle = func(emu *emulator) {
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
	p.logT.Printf("Designate Character Set: destination %q ,%q, %q\n", p.scsDst, p.scsMod, p.ch)

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

	hd = &Handler{name: "esc-dcs", ch: p.ch}
	hd.handle = func(emu *emulator) {
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
	// prepare the parameters
	params := make([]int, p.nInputOps)
	copy(params, p.inputOps)

	hd = &Handler{name: "csi-sm", ch: p.ch}
	hd.handle = func(emu *emulator) {
		hdl_csi_sm(emu, params)
	}

	p.setState(InputState_Normal)
	return hd
}

// Reset Mode
func (p *Parser) handle_RM() (hd *Handler) {
	// prepare the parameters
	params := make([]int, p.nInputOps)
	copy(params, p.inputOps)

	hd = &Handler{name: "csi-rm", ch: p.ch}
	hd.handle = func(emu *emulator) {
		hdl_csi_rm(emu, params)
	}

	p.setState(InputState_Normal)
	return hd
}

// Set Mode (private)
// csi_privSM
func (p *Parser) handle_DECSET() (hd *Handler) {
	// prepare the parameters
	params := make([]int, p.nInputOps)
	copy(params, p.inputOps)

	hd = &Handler{name: "csi-decset", ch: p.ch}
	hd.handle = func(emu *emulator) {
		hdl_csi_decset(emu, params)
	}

	p.setState(InputState_Normal)
	return hd
}

// Reset Mode (private)
// csi_privRM
func (p *Parser) handle_DECRST() (hd *Handler) {
	// prepare the parameters
	params := make([]int, p.nInputOps)
	copy(params, p.inputOps)

	hd = &Handler{name: "csi-decrst", ch: p.ch}
	hd.handle = func(emu *emulator) {
		hdl_csi_decrst(emu, params)
	}

	p.setState(InputState_Normal)
	return hd
}

// Set Top and Bottom Margins
func (p *Parser) handle_DECSTBM() (hd *Handler) {
	top := p.getPs(0, 1)
	bottom := p.getPs(1, 1)

	hd = &Handler{name: "csi-decstbm", ch: p.ch}
	hd.handle = func(emu *emulator) {
		hdl_csi_decstbm(emu, top, bottom)
	}

	// reset the state
	p.setState(InputState_Normal)
	return hd
}

// DEC Soft Terminal Reset
func (p *Parser) handle_DECSTR() (hd *Handler) {
	hd = &Handler{name: "csi-decstr", ch: p.ch}
	hd.handle = func(emu *emulator) {
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
		hd = &Handler{name: "dcs-decrqss", ch: p.ch}
		hd.handle = func(emu *emulator) {
			hdl_dcs_decrqss(emu, arg)
		}
	} else {
		p.logU.Printf("DCS: %q", arg)
	}

	return hd
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
				hd = p.processInput(input[0])
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

				// p.logT.Printf("processStream: input=%q\n", input)
				hd = p.processInput(input...)
				if hd != nil {
					hds = append(hds, hd)
					// p.logT.Printf("add handler to list. name=%q, ch=%q", hd.name, hd.ch)
				}
				_, to := graphemes.Positions()

				// p.logT.Printf("processSTream: to=%d\n", to)
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
func (p *Parser) processInput(chs ...rune) (hd *Handler) {
	var ch rune

	// for multi runes, it should be grapheme.
	if len(chs) > 1 {
		p.chs = chs
		hd = p.handle_Graphemes()
		return hd
	} else if len(chs) == 1 { // it's either grapheme or control sequence
		p.chs = chs
		ch = chs[0]
		p.appendToHistory(ch) // save the history, max 5 runes
	} else { // empty chs
		return hd
	}

	// fmt.Printf("processInput got %q\n", chs)
	p.lastEscBegin = 0
	p.lastNormalBegin = 0
	p.lastStopPos = 0
	p.ch = ch

	// p.logT.Printf(" ch=%q,\t nInputOps=%d, inputOps=%2d\n", ch, p.nInputOps, p.inputOps)
	switch p.inputState {
	case InputState_Normal:
		switch ch {
		case '\x00': // ignore NUL
		case '\x1B':
			p.setState(InputState_Escape)
			p.inputOps[0] = 0
			p.nInputOps = 1
			p.lastEscBegin = p.readPos // TODO ???
		case '\x0D': // CR is \r
			p.traceNormalInput()
			hd = p.handle_CR()
		case '\x0C', '\x0B', '\x0A': // FF is \f, VT is \v, LF is \n, they are handled same as IND
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
			hd = p.handle_Graphemes()
		}
	case InputState_Escape:
		switch ch {
		case '\x18', '\x1A': // CAN and SUB interrupts ESC sequence
			p.setState(InputState_Normal)
		case '\x1B': // ESC restarts ESC sequence
			p.inputOps[0] = 0
			p.nInputOps = 1
			p.lastEscBegin = p.readPos // TODO ???
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
		case ',', '$': // from ISO/IEC 2022 (absorbed, treat as no-op)
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
		case '7':
			hd = p.handle_DECSC()
		case '8':
			hd = p.handle_DECRC()
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
			p.logU.Println("S7C1T: Send 7-bit controls")
			p.setState(InputState_Normal)
		case 'G':
			p.logU.Println("S8C1T: Send 8-bit controls")
			p.setState(InputState_Normal)
		case 'L':
			p.logU.Println("Set ANSI conformance level 1")
			p.setState(InputState_Normal)
		case 'M':
			p.logU.Println("Set ANSI conformance level 2")
			p.setState(InputState_Normal)
		case 'N':
			p.logU.Println("Set ANSI conformance level 3")
			p.setState(InputState_Normal)
		default:
			p.unhandledInput()
		}
	case InputState_Esc_Hash:
		switch ch {
		case '3':
			p.logU.Println("DECDHL: Double-height, top half.")
			p.setState(InputState_Normal)
		case '4':
			p.logU.Println("DECDHL: Double-height, bottom half.")
			p.setState(InputState_Normal)
		case '5':
			p.logU.Println("DECSWL: Single-width line.")
			p.setState(InputState_Normal)
		case '6':
			p.logU.Println("DECDWL: Double-width line.")
			p.setState(InputState_Normal)
		case '8':
			hd = p.handle_DECALN()
		default:
			p.unhandledInput()
		}
	case InputState_Esc_Pct:
		switch ch {
		case '@':
			p.logT.Println("Select charset: default (ISO-8859-1)")
			hd = p.handle_DOCS_ISO8859_1()
		case 'G':
			p.logT.Println("Select charset: UTF-8")
			hd = p.handle_DOCS_UTF8()
		}
	case InputState_Select_Charset:
		if ch < 0x30 {
			// save the second byte, that means there should be third byte next.
			p.scsMod = ch
		} else {
			// the second byte or the third byte
			hd = p.handle_ESC_DCS()
		}
	case InputState_CSI: // TODO CNL, CPL
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
		case 'G':
			hd = p.handle_CHA_HPA()
		case 'H', 'f':
			hd = p.handle_CUP()
		case 'I':
			hd = p.handle_CHT()
		case 'J':
			hd = p.handle_ED()
		case 'K':
			hd = p.handle_EL()
		case 'L':
			hd = p.handle_IL()
		case 'M':
			hd = p.handle_DL()
		case 'P':
			hd = p.handle_DCH()
		case 'S':
			hd = p.handle_SU()
		case 'T':
			hd = p.handle_SD()
		case 'X':
			hd = p.handle_ECH()
		case 'Z':
			hd = p.handle_CBT()
		case '@':
			hd = p.handle_ICH()
		case '`':
			hd = p.handle_CHA_HPA()
		case 'c':
			hd = p.handle_DA1()
		case 'd':
			hd = p.handle_VPA()
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
		case '!':
			p.setState(InputState_CSI_Bang)
		case '?':
			p.setState(InputState_CSI_Priv)
		case '>':
			p.setState(InputState_CSI_GT)
		case '\x07': // BEL is ignored \a in c++
		case '\x08': // BS is \b
			// undo last character in CSI sequence:
			if p.getRuneAt(1) == ';' {
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
	case InputState_CSI_Bang:
		switch ch {
		case 'p':
			hd = p.handle_DECSTR()
		default:
			p.unhandledInput()
		}
	case InputState_CSI_GT:
		if p.collectNumericParameters(ch) {
			break
		}
		switch ch {
		case 'c':
			hd = p.handle_DA2()
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
			hd = p.handle_DECSET() // csi-privSM
		case 'l':
			hd = p.handle_DECRST() // csi_privRM
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
				p.logE.Printf("DCS argument string overflow (>4095). %q\n", p.argBuf.String())
				p.setState(InputState_Normal)
			}
		}
	case InputState_DCS_Esc:
		switch ch {
		case '\\':
			hd = p.handle_DCS()
		default:
			p.argBuf.WriteRune('\x1b')
			p.argBuf.WriteRune(ch)
			p.setState(InputState_DCS)
		}
	case InputState_OSC:
		switch ch {
		case '\x07': // final byte = BEL
			hd = p.handle_OSC()
		case '\x1B':
			p.setState(InputState_OSC_Esc)
		default:
			if p.argBuf.Len() < 4095 {
				p.argBuf.WriteRune(ch)
			} else {
				p.logE.Printf("OSC argument string overflow (>4096). %q\n", p.argBuf.String())
				p.setState(InputState_Normal)
			}
		}
	case InputState_OSC_Esc:
		switch ch {
		case '\\': // ESC \ : ST
			hd = p.handle_OSC()
		default:
			p.argBuf.WriteRune('\x1b')
			p.argBuf.WriteRune(ch)
			p.setState(InputState_OSC)
		}
	}
	return hd // actions
}
