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
//
// import "fmt"
//
// type Transition struct {
// 	action    Action
// 	nextState State
// }
//
// type State interface {
// 	enter() Action
// 	exit() Action
// 	parse(rune) Transition // it's uesed to be input
// 	eventList(r rune) Transition
// }
//
// type state struct{}
//
// func (s state) enter() Action               { return &ignore{} }    // default: ignore
// func (s state) exit() Action                { return &ignore{} }    // default: ignore
// func (s state) eventList(r rune) Transition { return Transition{} } // go nothing
// func (s state) parse(r rune) Transition     { return Transition{} } // do nothing
//
// func parseInput(currentState State, r rune) Transition {
// 	// Check for immediate transitions.
// 	anywhere := anywhere(r)
//
// 	// fill in the action fields
// 	if anywhere.nextState != nil {
// 		anywhere.action.SetChar(r)
// 		anywhere.action.SetPresent(true)
// 		return anywhere
// 	}
//
// 	// Normal X.364 state machine.
// 	// Parse high Unicode codepoints like 'A'.
// 	// avatar := r
// 	// if avatar >= 0xA0 {
// 	// 	avatar = 0x41
// 	// }
// 	// ret := currentState.eventList(avatar)
// 	ret := currentState.eventList(r)
// 	ret.action.SetChar(r)
// 	ret.action.SetPresent(true)
// 	return ret
// }
//
// func anywhere(ch rune) Transition {
// 	if ch == 0x18 || ch == 0x1A || (0x80 <= ch && ch <= 0x8F) || (0x91 <= ch && ch <= 0x97) || ch == 0x99 || ch == 0x9A {
// 		return Transition{&execute{}, ground{}}
// 	} else if ch == 0x9C {
// 		return Transition{&ignore{}, ground{}}
// 	} else if ch == 0x1B {
// 		return Transition{&ignore{}, escape{}}
// 	} else if ch == 0x98 || ch == 0x9E || ch == 0x9F {
// 		return Transition{&ignore{}, sosPmApcString{}}
// 	} else if ch == 0x90 {
// 		return Transition{&ignore{}, dcsEntry{}}
// 	} else if ch == 0x9D {
// 		return Transition{&ignore{}, oscString{}}
// 	} else if ch == 0x9B {
// 		return Transition{&ignore{}, csiEntry{}}
// 	}
//
// 	// both action and nextState is nil
// 	return Transition{}
// }
//
// func c0prime(r rune) bool {
// 	// event 00-17,19,1C-1F
// 	return r <= 0x17 || r == 0x19 || (0x1C <= r && r <= 0x1F)
// }
//
// func glgr(r rune) bool {
// 	// GL or GR
// 	return (0x20 <= r && r <= 0x7F) || (0xA0 <= r && r <= 0xFF)
// }
//
// type ground struct{ state }
//
// func (st ground) String() string          { return fmt.Sprintf("[%-18s] state", "ground") }
// func (st ground) parse(r rune) Transition { return parseInput(st, r) }
// func (st ground) eventList(r rune) Transition {
// 	// C0 control
// 	if c0prime(r) {
// 		return Transition{&execute{}, nil}
// 	}
//
// 	// for unicode problem, ignore it for ground state
// 	if r == '\uFFFD' {
// 		return Transition{&ignore{}, nil}
// 	}
//
// 	// mosh treat GR the same as GL,
// 	// difference from https://vt100.net/emu/dec_ansi_parser
// 	// only event 20-7F / print
// 	if glgr(r) {
// 		return Transition{&print{}, nil}
// 	}
//
// 	return Transition{&print{}, nil}
// }
//
// type escape struct{ state }
//
// func (st escape) String() string          { return fmt.Sprintf("[%-18s] state", "escape") }
// func (st escape) parse(r rune) Transition { return parseInput(st, r) }
// func (st escape) enter() Action           { return &clear{} }
// func (st escape) eventList(r rune) Transition {
// 	// C0 control
// 	if c0prime(r) {
// 		return Transition{&execute{}, nil}
// 	}
//
// 	// goto esc intermediate
// 	if 0x20 <= r && r <= 0x2F {
// 		return Transition{&collect{}, escapeIntermediate{}}
// 	}
//
// 	// goto ground
// 	if (0x30 <= r && r <= 0x4F) || (0x51 <= r && r <= 0x57) || r == 0x59 || r == 0x5A || r == 0x5C ||
// 		(0x60 <= r && r <= 0x7E) {
// 		return Transition{&escDispatch{}, ground{}}
// 	}
//
// 	// goto csi entry
// 	if r == 0x5B {
// 		return Transition{&ignore{}, csiEntry{}}
// 	}
//
// 	// goto osc
// 	if r == 0x5D {
// 		return Transition{&ignore{}, oscString{}}
// 	}
//
// 	// goto dcs entry
// 	if r == 0x50 {
// 		return Transition{&ignore{}, dcsEntry{}}
// 	}
//
// 	// goto sos/pm/apc
// 	if r == 0x58 || r == 0x5E || r == 0x5F {
// 		return Transition{&ignore{}, sosPmApcString{}}
// 	}
//
// 	// the last one is event 7F / ignore
// 	return Transition{&ignore{}, nil}
// }
//
// type escapeIntermediate struct{ state }
//
// func (st escapeIntermediate) String() string {
// 	return fmt.Sprintf("[%-18s] state", "escapeIntermediate")
// }
// func (st escapeIntermediate) parse(r rune) Transition { return parseInput(st, r) }
// func (st escapeIntermediate) eventList(r rune) Transition {
// 	// c0 control
// 	if c0prime(r) {
// 		return Transition{&execute{}, nil}
// 	}
//
// 	// collect
// 	if 0x20 <= r && r <= 0x2F {
// 		return Transition{&collect{}, nil}
// 	}
//
// 	// goto ground
// 	if 0x30 <= r && r <= 0x7E {
// 		return Transition{&escDispatch{}, ground{}}
// 	}
//
// 	// the last one is event 7F / ignore
// 	return Transition{&ignore{}, nil}
// }
//
// type csiEntry struct{ state }
//
// func (st csiEntry) String() string          { return fmt.Sprintf("[%-18s] state", "csiEntry") }
// func (st csiEntry) parse(r rune) Transition { return parseInput(st, r) }
// func (st csiEntry) enter() Action           { return &clear{} }
// func (st csiEntry) eventList(r rune) Transition {
// 	// c0 control
// 	if c0prime(r) {
// 		return Transition{&execute{}, nil}
// 	}
//
// 	// goto ground: dispatch
// 	if 0x40 <= r && r <= 0x7E {
// 		return Transition{&csiDispatch{}, ground{}}
// 	}
//
// 	// goto csi param: param
// 	// 0~9:;
// 	if 0x30 <= r && r <= 0x3B {
// 		return Transition{&param{}, csiParam{}}
// 	}
//
// 	// goto csi para: collect
// 	// <,=,>,?
// 	if 0x3C <= r && r <= 0x3F {
// 		return Transition{&collect{}, csiParam{}}
// 	}
//
// 	// goto csi intermediate: collect
// 	// space,!,",#,$,%,&,',(,),*,+,comma,-,.,/
// 	if 0x20 <= r && r <= 0x2F {
// 		return Transition{&collect{}, csiIntermediate{}}
// 	}
//
// 	// the last one is event 7F / ignore
// 	return Transition{&ignore{}, nil}
// }
//
// type csiParam struct{ state }
//
// func (st csiParam) String() string          { return fmt.Sprintf("[%-18s] state", "csiParam") }
// func (st csiParam) parse(r rune) Transition { return parseInput(st, r) }
// func (st csiParam) eventList(r rune) Transition {
// 	// c0 control
// 	if c0prime(r) {
// 		return Transition{&execute{}, nil}
// 	}
//
// 	// csi param
// 	// 0~9:;
// 	if 0x30 <= r && r <= 0x3B {
// 		return Transition{&param{}, nil}
// 	}
//
// 	// goto csi ignore
// 	// <,=,>,?
// 	if 0x3C <= r && r <= 0x3F {
// 		return Transition{&ignore{}, csiIgnore{}}
// 	}
//
// 	// goto csi intermediate: collect
// 	// space,!,",#,$,%,&,',(,),*,+,comma,-,.,/
// 	if 0x20 <= r && r <= 0x2F {
// 		return Transition{&collect{}, csiIntermediate{}}
// 	}
//
// 	// goto ground: csi dispatch
// 	if 0x40 <= r && r <= 0x7E {
// 		return Transition{&csiDispatch{}, ground{}}
// 	}
//
// 	// the last one is event 7F / ignore
// 	return Transition{&ignore{}, nil}
// }
//
// type csiIntermediate struct{ state }
//
// func (st csiIntermediate) String() string          { return fmt.Sprintf("[%-18s] state", "csiIntermediate") }
// func (st csiIntermediate) parse(r rune) Transition { return parseInput(st, r) }
// func (st csiIntermediate) eventList(r rune) Transition {
// 	// c0 control
// 	if c0prime(r) {
// 		return Transition{&execute{}, nil}
// 	}
//
// 	// collect
// 	// space,!,",#,$,%,&,',(,),*,+,comma,-,.,/
// 	if 0x20 <= r && r <= 0x2F {
// 		return Transition{&collect{}, nil}
// 	}
//
// 	// goto ground: csi dispatch
// 	if 0x40 <= r && r <= 0x7E {
// 		return Transition{&csiDispatch{}, ground{}}
// 	}
//
// 	// goto csi ignore
// 	if 0x30 <= r && r <= 0x3F {
// 		return Transition{&ignore{}, csiIgnore{}}
// 	}
//
// 	// the last one is event 7F / ignore
// 	return Transition{&ignore{}, nil}
// }
//
// type csiIgnore struct{ state }
//
// func (st csiIgnore) String() string          { return fmt.Sprintf("[%-18s] state", "csiIgnore") }
// func (st csiIgnore) parse(r rune) Transition { return parseInput(st, r) }
// func (st csiIgnore) eventList(r rune) Transition {
// 	// c0 control
// 	if c0prime(r) {
// 		return Transition{&execute{}, nil}
// 	}
//
// 	// difference: vt100.net/emu/dec_ansi_parser
// 	// event 20-3F / ignore
//
// 	// goto ground
// 	if 0x40 <= r && r <= 0x7E {
// 		return Transition{&ignore{}, ground{}}
// 	}
//
// 	// the last one is event 7F / ignore
// 	return Transition{&ignore{}, nil}
// }
//
// type dcsEntry struct{ state }
//
// func (st dcsEntry) String() string          { return fmt.Sprintf("[%-18s] state", "dcsEntry") }
// func (st dcsEntry) parse(r rune) Transition { return parseInput(st, r) }
// func (st dcsEntry) enter() Action           { return &clear{} }
// func (st dcsEntry) eventList(r rune) Transition {
// 	// difference: vt100.net/emu/dec_ansi_parser
// 	// event 00-17,19,1C-1F / ignore
// 	// if c0prime(r) {
// 	// 	return Transition{ignore{}, nil}
// 	// }
//
// 	// goto dcs intermediate: collect
// 	if 0x20 <= r && r <= 0x2F {
// 		return Transition{&collect{}, dcsIntermediate{}}
// 	}
//
// 	// goto dcs ignore
// 	// :
// 	if r == 0x3A {
// 		return Transition{&ignore{}, dcsIgnore{}}
// 	}
//
// 	// goto dcs param: param
// 	// ;,0~9
// 	if r == 0x3B || (0x30 <= r && r <= 0x39) {
// 		return Transition{&param{}, dcsParam{}}
// 	}
//
// 	// goto dcs param: collect
// 	// <,=,>,?
// 	if 0x3C <= r && r <= 0x3F {
// 		return Transition{&collect{}, dcsParam{}}
// 	}
//
// 	// goto dcs passthrough
// 	if 0x40 <= r && r <= 0x7E {
// 		return Transition{&ignore{}, dcsPassthrough{}}
// 	}
//
// 	// the last one is event 7F / ignore
// 	// event 00-17,19,1C-1F / ignore
// 	return Transition{&ignore{}, nil}
// }
//
// type dcsParam struct{ state }
//
// func (st dcsParam) String() string          { return fmt.Sprintf("[%-18s] state", "dcsParam") }
// func (st dcsParam) parse(r rune) Transition { return parseInput(st, r) }
// func (st dcsParam) eventList(r rune) Transition {
// 	// difference: vt100.net/emu/dec_ansi_parser
// 	// event 00-17,19,1C-1F / ignore
// 	// if c0prime(r) {
// 	// 	return Transition{ignore{}, nil}
// 	// }
//
// 	// param
// 	// ;,0~9
// 	if r == 0x3B || (0x30 <= r && r <= 0x39) {
// 		return Transition{&param{}, nil}
// 	}
//
// 	// goto dcs ignore
// 	// :,<,=,>,?
// 	if r == 0x3A || (0x3C <= r && r <= 0x3F) {
// 		return Transition{&ignore{}, dcsIgnore{}}
// 	}
//
// 	// goto dcs intermediate: collect
// 	if 0x20 <= r && r <= 0x2F {
// 		return Transition{&collect{}, dcsIntermediate{}}
// 	}
//
// 	// goto dcs passthrough
// 	if 0x40 <= r && r <= 0x7E {
// 		return Transition{&ignore{}, dcsPassthrough{}}
// 	}
//
// 	// the last one is event 7F / ignore
// 	// event 00-17,19,1C-1F / ignore
// 	return Transition{&ignore{}, nil}
// }
//
// type dcsIntermediate struct{ state }
//
// func (st dcsIntermediate) String() string          { return fmt.Sprintf("[%-18s] state", "dcsIntermediate") }
// func (st dcsIntermediate) parse(r rune) Transition { return parseInput(st, r) }
// func (st dcsIntermediate) eventList(r rune) Transition {
// 	// difference: vt100.net/emu/dec_ansi_parser
// 	// event 00-17,19,1C-1F / ignore
// 	// if c0prime(r) {
// 	// 	return Transition{ignore{}, nil}
// 	// }
//
// 	// collect
// 	if 0x20 <= r && r <= 0x2F {
// 		return Transition{&collect{}, nil}
// 	}
//
// 	// goto dcs passthrough
// 	if 0x40 <= r && r <= 0x7E {
// 		return Transition{&ignore{}, dcsPassthrough{}}
// 	}
//
// 	// goto dcs ignore
// 	if 0x30 <= r && r <= 0x3F {
// 		return Transition{&ignore{}, dcsIgnore{}}
// 	}
//
// 	// the last one is event 7F / ignore
// 	// event 00-17,19,1C-1F / ignore
// 	return Transition{&ignore{}, nil}
// }
//
// type dcsPassthrough struct{ state }
//
// func (st dcsPassthrough) String() string          { return fmt.Sprintf("[%-18s] state", "dcsPassthrough") }
// func (st dcsPassthrough) parse(r rune) Transition { return parseInput(st, r) }
// func (st dcsPassthrough) enter() Action           { return &hook{} }
// func (st dcsPassthrough) exit() Action            { return &unhook{} }
// func (st dcsPassthrough) eventList(r rune) Transition {
// 	// put
// 	if c0prime(r) || (0x20 <= r && r <= 0x7E) {
// 		return Transition{&put{}, nil}
// 	}
//
// 	// finish
// 	// ST
// 	if r == 0x9C {
// 		return Transition{&ignore{}, ground{}}
// 	}
//
// 	// the last one is event 7F / ignore
// 	return Transition{&ignore{}, nil}
// }
//
// type dcsIgnore struct{ state }
//
// func (st dcsIgnore) String() string          { return fmt.Sprintf("[%-18s] state", "dcsIgnore") }
// func (st dcsIgnore) parse(r rune) Transition { return parseInput(st, r) }
// func (st dcsIgnore) eventList(r rune) Transition {
// 	// difference: vt100.net/emu/dec_ansi_parser
// 	// event 00-17,19,1C-1F,20-7F / ignore
// 	// if c0prime(r) || (0x20 <= r && r <= 0x7F) {
// 	// 	return Transition{put{}, nil}
// 	// }
//
// 	if r == 0x9C {
// 		return Transition{&ignore{}, ground{}}
// 	}
//
// 	// the lase one is
// 	// event 00-17,19,1C-1F,20-7F / ignore
// 	return Transition{&ignore{}, nil}
// }
//
// type oscString struct{ state }
//
// func (st oscString) String() string          { return fmt.Sprintf("[%-18s] state", "oscString") }
// func (st oscString) parse(r rune) Transition { return parseInput(st, r) }
// func (st oscString) enter() Action           { return &oscStart{} }
// func (st oscString) exit() Action            { return &oscEnd{} }
// func (st oscString) eventList(r rune) Transition {
// 	// difference: vt100.net/emu/dec_ansi_parser
// 	// event 00-17,19,1C-1F / ignore
// 	// if c0prime(r) {
// 	// 	return Transition{ignore{}, nil}
// 	// }
//
// 	// osc put
// 	// TODO should consider unicode title
// 	// if 0x20 <= r && r <= 0x7F {
// 	// 	return Transition{&oscPut{}, nil}
// 	// }
//
// 	// goto ground: end osc string state
// 	// 0x07 is xterm non-ANSI variant
// 	// 0x9C it's not possible for utf-8 environment
// 	// \uFFFD for osc string sequence, it's the same as 0x07
// 	if r == 0x9C || r == 0x07 || r == '\uFFFD' {
// 		return Transition{&ignore{}, ground{}}
// 	}
//
// 	// the lase one is
// 	// event 00-17,19,1C-1F / ignore
// 	return Transition{&oscPut{}, nil}
// }
//
// type sosPmApcString struct{ state }
//
// func (st sosPmApcString) String() string          { return fmt.Sprintf("[%-18s] state", "sosPmApcString") }
// func (st sosPmApcString) parse(r rune) Transition { return parseInput(st, r) }
// func (st sosPmApcString) eventList(r rune) Transition {
// 	// difference: vt100.net/emu/dec_ansi_parser
// 	// event 00-17,19,1C-1F,20-7F / ignore
// 	// if c0prime(r) || (0x20 <= r && r <= 0x7F) {
// 	// 	return Transition{ignore{}, nil}
// 	// }
//
// 	// goto ground
// 	if r == 0x9C {
// 		return Transition{&ignore{}, ground{}}
// 	}
//
// 	// the lase one is
// 	// event 00-17,19,1C-1F,20-7F / ignore
// 	return Transition{&ignore{}, nil}
// }
