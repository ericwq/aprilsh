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
}

func NewParser() *Parser {
	p := new(Parser)
	// TODO consider remove state field
	// p.state = ground{}

	p.reset()
	return p
}

func (p *Parser) reset() {
	p.inputState = InputState_Normal
	p.ch = 0x00

	p.perror = nil

	p.maxEscOps = 16
	p.inputOps = make([]int, p.maxEscOps)
	p.nInputOps = 0
	p.argBuf.Reset()

	p.scsDst = 0x00
	p.scsMod = 0x00
}

func (p *Parser) traceNormalInput() { // TODO
}

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
			// TODO logE  "inputOp overflow!"
			p.perror = fmt.Errorf("the number is too big. %d", p.inputOps[p.nInputOps-1])
			p.setState(InputState_Normal)
		}
	} else if ch == ';' {
		isBreak = true
		if p.nInputOps < p.maxEscOps { // move to the next parameter
			p.inputOps[p.nInputOps] = 0
			p.nInputOps += 1
		} else {
			// TODO logE inputOps full, increase maxEscOps
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

// prepare parameters for the CUU
func (p *Parser) handle_CUU() (hd *Handler) {
	num := p.getPs(0, 1)

	hd = &Handler{name: "cuu", ch: p.ch}
	hd.handle = func(emu *emulator) {
		hdl_csi_cuu(emu, num)
	}

	p.setState(InputState_Normal)
	return hd
}

// prepare parameters for the CUD
func (p *Parser) handle_CUD() (hd *Handler) {
	num := p.getPs(0, 1)

	hd = &Handler{name: "cud", ch: p.ch}
	hd.handle = func(emu *emulator) {
		hdl_csi_cud(emu, num)
	}

	p.setState(InputState_Normal)
	return hd
}

// prepare parameters for the CUF
func (p *Parser) handle_CUF() (hd *Handler) {
	num := p.getPs(0, 1)

	hd = &Handler{name: "cuf", ch: p.ch}
	hd.handle = func(emu *emulator) {
		hdl_csi_cuf(emu, num)
	}

	p.setState(InputState_Normal)
	return hd
}

// prepare parameters for CUB
func (p *Parser) handle_CUB() (hd *Handler) {
	num := p.getPs(0, 1)

	hd = &Handler{name: "cub", ch: p.ch}
	hd.handle = func(emu *emulator) {
		hdl_csi_cub(emu, num)
	}

	p.setState(InputState_Normal)
	return hd
}

// prepare parameters for the CUP
func (p *Parser) handle_CUP() (hd *Handler) {
	row := p.getPs(0, 1)
	col := p.getPs(1, 1)

	hd = &Handler{name: "cup", ch: p.ch}
	hd.handle = func(emu *emulator) {
		hdl_csi_cup(emu, row, col)
	}

	// reset the state
	p.setState(InputState_Normal)
	return hd
}

func (p *Parser) handle_OSC() (hd *Handler) {
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
		// LogT "OSC: malformed command string '"
	} else {
		switch cmd {
		// create the ActOn
		case 0, 1, 2:
			hd = &Handler{name: "osc 0,1,2", ch: p.ch}
			hd.handle = func(emu *emulator) {
				hdl_osc_0(emu, cmd, arg)
			}
		case 4:
			hd = &Handler{name: "osc 4", ch: p.ch}
			hd.handle = func(emu *emulator) {
				hdl_osc_4(emu, cmd, arg)
			}
		case 52:
			hd = &Handler{name: "osc 52", ch: p.ch}
			hd.handle = func(emu *emulator) {
				hdl_osc_52(emu, cmd, arg)
			}
		case 10, 11, 12, 17, 19:
			hd = &Handler{name: "osc 10,11,12,17,19", ch: p.ch}
			hd.handle = func(emu *emulator) {
				hdl_osc_10(emu, cmd, arg)
			}
		default:
			// logU "unhandled OSC: '"
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
	return hd
}

// Line Feed
func (p *Parser) handle_IND() (hd *Handler) {
	hd = &Handler{name: "c0-lf", ch: p.ch}
	hd.handle = func(emu *emulator) {
		hdl_c0_lf(emu)
	}
	// reset the state
	p.setState(InputState_Normal)
	return hd
}

// Horizontal Tab
// move cursor position to next tab stop
func (p *Parser) handle_HT() (hd *Handler) {
	hd = &Handler{name: "c0-ht", ch: p.ch}
	hd.handle = func(emu *emulator) {
		hdl_c0_ht(emu)
	}
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

// print graphemes on screen
func (p *Parser) handle_Graphemes() (hd *Handler) {
	// fmt.Printf("handle_Graphemes got %q\n\n", p.chs)
	hd = &Handler{name: "graphemes", ch: p.ch}

	r := p.chs
	hd.handle = func(emu *emulator) {
		hdl_graphemes(emu, r...)
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
	hd = &Handler{name: "c0-hts", ch: p.ch}
	hd.handle = func(emu *emulator) {
		hdl_c0_hts(emu)
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

// ESC N Single Shift Select of G2 Character Set (SS2  is 0x8e), VT220.
func (p *Parser) handle_SS2() (hd *Handler) {
	hd = &Handler{name: "c0-ss2", ch: p.ch}
	hd.handle = func(emu *emulator) {
		hdl_c0_ss2(emu)
	}

	p.setState(InputState_Normal)
	return hd
}

// ESC O Single Shift Select of G3 Character Set (SS3  is 0x8f), VT220.
func (p *Parser) handle_SS3() (hd *Handler) {
	hd = &Handler{name: "c0-ss3", ch: p.ch}
	hd.handle = func(emu *emulator) {
		hdl_c0_ss3(emu)
	}

	p.setState(InputState_Normal)
	return hd
}

// LS1R gr = 1
func (p *Parser) handle_LS1R() (hd *Handler) {
	hd = &Handler{name: "c0-ls1r", ch: p.ch}
	hd.handle = func(emu *emulator) {
		hdl_c0_ls1r(emu)
	}

	p.setState(InputState_Normal)
	return hd
}

// LS2 gl = 2
func (p *Parser) handle_LS2() (hd *Handler) {
	hd = &Handler{name: "c0-ls2", ch: p.ch}
	hd.handle = func(emu *emulator) {
		hdl_c0_ls2(emu)
	}

	p.setState(InputState_Normal)
	return hd
}

// LS2R gr = 2
func (p *Parser) handle_LS2R() (hd *Handler) {
	hd = &Handler{name: "c0-ls2r", ch: p.ch}
	hd.handle = func(emu *emulator) {
		hdl_c0_ls2r(emu)
	}

	p.setState(InputState_Normal)
	return hd
}

// LS3 gl = 3
func (p *Parser) handle_LS3() (hd *Handler) {
	hd = &Handler{name: "c0-ls3", ch: p.ch}
	hd.handle = func(emu *emulator) {
		hdl_c0_ls3(emu)
	}

	p.setState(InputState_Normal)
	return hd
}

// LS3R gr = 3
func (p *Parser) handle_LS3R() (hd *Handler) {
	hd = &Handler{name: "c0-ls3r", ch: p.ch}
	hd.handle = func(emu *emulator) {
		hdl_c0_ls3r(emu)
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

// ESC ( C   Designate G0 Character Set, VT100, ISO 2022.
// ESC ) C   Designate G1 Character Set, ISO 2022, VT100
// ESC * C   Designate G2 Character Set, ISO 2022, VT220.
// ESC + C   Designate G3 Character Set, ISO 2022, VT220.
// ESC - C   Designate G1 Character Set, VT300.
// ESC . C   Designate G2 Character Set, VT300.
// ESC / C   Designate G3 Character Set, VT300.
// https://invisible-island.net/xterm/ctlseqs/ctlseqs.html#h3-Controls-beginning-with-ESC
func (p *Parser) handle_ESC_DCS() (hd *Handler) {
	// TODO log it
	// fmt.Printf("DEBUG Designate Character Set: destination %q ,%q, %q\n", p.scsDst, p.scsMod, p.ch)

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

	// fmt.Printf("DEBUG Designate Character Set: index= %d, charset=%d\n", index, charset)
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
				// fmt.Printf("processStream: input=%q\n", input)
				hd = p.processInput(input...)
				if hd != nil {
					hds = append(hds, hd)
				}
				_, to := graphemes.Positions()
				if to == len(str)-1 {
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
	} else { // empty chs
		return hd
	}

	// fmt.Printf("processInput got %q\n", chs)
	p.lastEscBegin = 0
	p.lastNormalBegin = 0
	p.lastStopPos = 0
	p.ch = ch

	// fmt.Printf(" ch=%q,\t nInputOps=%d, inputOps=%2d\n", ch, p.nInputOps, p.inputOps)
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
		case 'H':
			hd = p.handle_HTS()
		case 'N':
			hd = p.handle_SS2()
		case 'O':
			hd = p.handle_SS3()
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
		}
	case InputState_Esc_Pct:
		switch ch {
		case '@':
			// logT << "Select charset: default (ISO-8859-1)"
			hd = p.handle_DOCS_ISO8859_1()
		case 'G':
			// logT << "Select charset: UTF-8"
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
	case InputState_CSI:
		if p.collectNumericParameters(ch) {
			break
		}
		switch ch {
		case 'A':
			hd = p.handle_CUU()
		case 'B':
			hd = p.handle_CUD()
		case 'C':
			hd = p.handle_CUF()
		case 'D':
			hd = p.handle_CUB()
		case 'H', 'f':
			hd = p.handle_CUP()
		case 'I':
			hd = p.handle_CHT()
		case 'Z':
			hd = p.handle_CBT()
		case 'g':
			hd = p.handle_TBC()
		}
	case InputState_OSC:
		// if p.collectNumericParameters(ch) {
		// 	break
		// }
		switch ch {
		case '\x07': // final byte = BEL
			hd = p.handle_OSC()
		case '\x1B':
			p.setState(InputState_OSC_Esc)
		default:
			if p.argBuf.Len() < 4096 {
				p.argBuf.WriteRune(ch)
			} else {
				// logE "OSC argument string overflow"
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
