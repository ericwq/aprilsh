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
	"strings"
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
	state State

	// big switch state machine
	inputState int

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
}

func NewParser() *Parser {
	p := new(Parser)
	p.state = ground{}
	p.maxEscOps = 16
	p.inputOps = make([]int, p.maxEscOps)
	return p
}

// append action to action list except ignore action
func appendTo(actions []Action, act Action) []Action {
	if !act.Ignore() {
		actions = append(actions, act)
	}
	return actions
}

// parse the input character into action and save it in action list
// it's uesed to be input
func (p *Parser) parse(actions []Action, r rune) []Action {
	// start to parse
	ts := p.state.parse(r)

	// exit action from old state
	if ts.nextState != nil {
		actions = appendTo(actions, p.state.exit())
	}

	// transition action
	actions = appendTo(actions, ts.action)
	ts.action = nil

	// enter action to new state
	if ts.nextState != nil {
		actions = appendTo(actions, ts.nextState.enter())
		// transition to next state
		p.state = ts.nextState
	}

	return actions
}

func (p *Parser) reset() {
	p.state = ground{}
}

func (p *Parser) traceNormalInput() {}
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
		if p.inputOps[p.nInputOps-1] < 65535 { // max value for numeric parameter
			p.inputOps[p.nInputOps-1] *= 10
			p.inputOps[p.nInputOps-1] += int(ch - '0')
		} else {
			// logE  "inputOp overflow!"
			p.setState(InputState_Normal)
		}
	} else if ch == ';' {
		isBreak = true
		if p.nInputOps < p.maxEscOps { // move to the next parameter
			p.inputOps[p.nInputOps] = 0
			p.nInputOps += 1
		} else {
			// logE inputOps full, increase maxEscOps
			p.setState(InputState_Normal)
		}
	}

	return isBreak
}

type Handler struct {
	name   string             // the name of ActOn
	handle func(emu emulator) // the action will take place on emulator
}

func (p *Parser) handle_CUP() *Handler {
	row := 1
	col := 1
	if p.inputOps[0] > 0 {
		row = p.inputOps[0]
	}

	if p.nInputOps > 1 && p.inputOps[1] > 0 {
		col = p.inputOps[1]
	}

	ac := Handler{}
	ac.name = "cup"
	ac.handle = func(emu emulator) {
		hdl_cup(emu, row, col)
	}

	// reset the state
	p.setState(InputState_Normal)
	return &ac
}

func hdl_cup(_ emulator, row int, col int) {
	fmt.Printf("handle osc row=%d, col=%d\n", row, col)
}

func (p *Parser) handle_OSC() *Handler {
	var hd *Handler
	cmd := 0
	arg := ""

	if p.inputOps[0] > 0 {
		cmd = p.inputOps[0]
	}

	if p.argBuf.Len() > 0 {
		arg = p.argBuf.String()
	}

	if cmd < 0 || cmd > 120 {
		// LogT "OSC: malformed command string '"
	} else {
		switch cmd {
		// create the ActOn
		case 0, 1, 2:
			hd = &Handler{}
			hd.name = "osc 0,1,2"
			hd.handle = func(emu emulator) {
				hdl_osc_0(emu, cmd, arg)
			}
		case 4:
			hd = &Handler{}
			hd.name = "osc 4"
			hd.handle = func(emu emulator) {
				hdl_osc_4(emu, cmd, arg)
			}
		case 52:
			hd = &Handler{}
			hd.name = "osc 52"
			hd.handle = func(emu emulator) {
				hdl_osc_52(emu, cmd, arg)
			}
		case 10, 11, 12, 17, 19:
			hd = &Handler{}
			hd.name = "osc 10,11,12,17,19"
			hd.handle = func(emu emulator) {
				hdl_osc_10(emu, cmd, arg)
			}
		default:
			// logU "unhandled OSC: '"
		}
	}

	// reset the state
	p.setState(InputState_Normal)
	return hd
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

func (p *Parser) processInput(ch rune) *Handler {
	var hd *Handler
	p.lastEscBegin = 0
	p.lastNormalBegin = 0
	p.lastStopPos = 0

	switch p.inputState {
	case InputState_Normal:
		switch ch {
		case '\x1B':
			p.setState(InputState_Escape)
			p.inputOps[0] = 0
			p.nInputOps = 1
			p.lastEscBegin = p.readPos // ???
		}
	case InputState_Escape:
		switch ch {
		case '[':
			p.setState(InputState_CSI)
		case ']':
			p.argBuf.Reset()
			p.setState(InputState_OSC)
		}
	case InputState_CSI:
		if p.collectNumericParameters(ch) {
			break
		}
		switch ch {
		case 'H', 'f':
			hd = p.handle_CUP()
		}

	case InputState_OSC:
		if p.collectNumericParameters(ch) {
			break
		}
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
