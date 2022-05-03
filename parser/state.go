package parser

type Transition struct {
	action    Action
	nextState State
}

type State interface {
	enter() Action
	exit() Action
	parse(rune) Transition // it's uesed to be input
	eventList(r rune) Transition
}

type state struct{}

func (s state) enter() Action               { return ignore{} }
func (s state) exit() Action                { return ignore{} }
func (s state) eventList(r rune) Transition { return Transition{} }

func (s state) parse(r rune) Transition {
	// Check for immediate transitions.
	anywhere := s.anywhere(r)

	// fill in the action fields
	if anywhere.nextState != nil {
		if access, ok := anywhere.action.(accessAction); ok {
			access.setChar(r)
			access.setPresent(true)
			return anywhere
		}
	}

	// Normal X.364 state machine.
	// Parse high Unicode codepoints like 'A'.
	// TODO verify unicode process
	if r >= 0xA0 {
		r = 0x41
	}
	ret := s.eventList(r)
	if access, ok := ret.action.(accessAction); ok {
		access.setChar(r)
		access.setPresent(true)
		return ret
	}

	return Transition{}
}

func (s state) anywhere(ch rune) Transition {
	if ch == 0x18 || ch == 0x1A || (0x80 <= ch && ch <= 0x8F) || (0x91 <= ch && ch <= 0x97) || ch == 0x99 || ch == 0x9A {
		return Transition{execute{}, ground{}}
	} else if ch == 0x9C {
		return Transition{ignore{}, ground{}}
	} else if ch == 0x1B {
		return Transition{ignore{}, escape{}}
	} else if ch == 0x98 || ch == 0x9E || ch == 0x9F {
		return Transition{ignore{}, sosPmApcString{}}
	} else if ch == 0x90 {
		return Transition{ignore{}, dcsEntry{}}
	} else if ch == 0x9D {
		return Transition{ignore{}, oscString{}}
	} else if ch == 0x9B {
		return Transition{ignore{}, csiEntry{}}
	}

	// both action and nextState is nil
	return Transition{}
}

func C0prime(r rune) bool {
	// why 0x18 0x1A 0x1B is missing?
	// covered by anywhere rule
	return ((r <= 0x17) || (r == 0x19) || ((0x1C <= r) && (r <= 0x1F)))
}

func GLGR(r rune) bool {
	// GL or GR
	return (((0x20 <= r) && (r <= 0x7F)) || ((0xA0 <= r) && (r <= 0xFF)))
}

type ground struct {
	state
}

func (g ground) eventList(r rune) Transition {
	if C0prime(r) {
		return Transition{execute{}, nil}
	}

	// treat GR the same as GL,
	// TODO verify unicode support
	if GLGR(r) {
		return Transition{print{}, nil}
	}

	return Transition{ignore{}, nil}
}

type escape struct {
	state
}

func (g escape) enter() Action { return clear{} }
func (e escape) eventList(r rune) Transition {
	// C0 control
	if C0prime(r) {
		return Transition{execute{}, nil}
	}

	// collect
	if 0x20 <= r && r <= 0x2F {
		return Transition{collect{}, escapIntermediate{}}
	}

	// esc dispatch
	if (0x30 <= r && r <= 0x4F) || (0x51 <= r && r <= 0x57) || r == 0x59 || r == 0x5A || r == 0x5C ||
		(0x60 <= r && r <= 0x7E) {
		return Transition{escDispatch{}, ground{}}
	}

	// csi entry
	if r == 0x5B {
		return Transition{nil, csiEntry{}}
	}

	// osc
	if r == 0x5D {
		return Transition{nil, oscString{}}
	}

	// dcs entry
	if r == 0x50 {
		return Transition{nil, dcsEntry{}}
	}

	// sos pm apc
	if r == 0x58 || r == 0x5E || r == 0x5F {
		return Transition{nil, sosPmApcString{}}
	}

	return Transition{ignore{}, nil}
}

type escapIntermediate struct{ state }

func (e escapIntermediate) eventList(r rune) Transition {
	// c0 control
	if C0prime(r) {
		return Transition{execute{}, nil}
	}

	// collect
	if 0x20 <= r && r <= 0x2F {
		return Transition{collect{}, nil}
	}

	// esc dispatch
	if 0x30 <= r && r <= 0x7E {
		return Transition{escDispatch{}, ground{}}
	}

	// event 7F / ignore
	// TODO verify it
	return Transition{}
}

type csiEntry struct{ state }

func (c csiEntry) enter() Action { return clear{} }
func (c csiEntry) eventList(r rune) Transition {
	// c0 control
	if C0prime(r) {
		return Transition{execute{}, nil}
	}

	// csi dispatch
	if 0x40 <= r && r <= 0x7E {
		return Transition{csiDispatch{}, ground{}}
	}

	// csi param : 0~9 and ;
	if (0x30 <= r && r <= 0x39) || r == 0x3B {
		return Transition{param{}, csiParam{}}
	}

	// csi collect param
	if 0x3C <= r && r <= 0x3F {
		return Transition{collect{}, csiParam{}}
	}

	// csi ignore
	if r == 0x3A {
		return Transition{ignore{}, csiIgnore{}}
	}

	// csi collect intermediate
	if 0x20 <= r && r <= 0x2F {
		return Transition{collect{}, csiIntermediate{}}
	}

	// event 7F / ignore
	// TODO verify it
	return Transition{}
}

type csiParam struct{ state }

func (c csiParam) eventList(r rune) Transition {
	// c0 control
	if C0prime(r) {
		return Transition{execute{}, nil}
	}

	// csi param
	if (0x30 <= r && r <= 0x39) || r == 0x3B {
		return Transition{param{}, nil}
	}

	// csi ignore
	if r == 0x3A || (0x3C <= r && r <= 0x3F) {
		return Transition{ignore{}, csiIgnore{}}
	}

	// csi intermediate collect
	if 0x20 <= r && r <= 0x2F {
		return Transition{collect{}, csiIntermediate{}}
	}

	// csi dispatch
	if 0x40 <= r && r <= 0x7E {
		return Transition{csiDispatch{}, ground{}}
	}

	// event 7F / ignore
	// TODO verify it
	return Transition{}
}

type csiIntermediate struct{ state }

func (c csiIntermediate) eventList(r rune) Transition {
	// c0 control
	if C0prime(r) {
		return Transition{execute{}, nil}
	}

	// collect action
	if 0x20 <= r && r <= 0x2F {
		return Transition{collect{}, nil}
	}

	// csi dispatch
	if 0x40 <= r && r <= 0x7E {
		return Transition{csiDispatch{}, ground{}}
	}

	// csi ignore
	if 0x30 <= r && r <= 0x3F {
		return Transition{ignore{}, csiIgnore{}}
	}

	// event 7F / ignore
	// TODO verify it
	return Transition{}
}

type csiIgnore struct{ state }

func (c csiIgnore) eventList(r rune) Transition {
	// c0 control
	if C0prime(r) {
		return Transition{execute{}, nil}
	}

	// ignore
	// event 20-3F,7F / ignore
	// TODO why change to 40-7E
	if 0x40 <= r && r <= 0x7E {
		return Transition{ignore{}, ground{}}
	}

	return Transition{}
}

type dcsEntry struct{ state }

func (d dcsEntry) enter() Action { return clear{} }
func (d dcsEntry) eventList(r rune) Transition {
	// c0 control
	// TODO we add c0, verify it
	// miss it?
	if C0prime(r) {
		return Transition{ignore{}, nil}
	}

	// goto dcs intermediate
	if 0x20 <= r && r <= 0x2F {
		return Transition{collect{}, dcsIntermediate{}}
	}

	// goto dcs ignore
	if r == 0x3A {
		return Transition{ignore{}, dcsIgnore{}}
	}

	// goto dcs param
	if r == 0x3B || (0x30 <= r && r <= 0x39) {
		return Transition{param{}, dcsParam{}}
	}

	// goto dcs param
	if 0x3C <= r && r <= 0x3F {
		return Transition{collect{}, dcsParam{}}
	}

	// goto dcs passthrough
	if 0x40 <= r && r <= 0x7E {
		return Transition{ignore{}, dcsPassthrough{}}
	}

	// event 7F / ignore
	// TODO verify it
	return Transition{}
}

type dcsParam struct{ state }

func (d dcsParam) eventList(r rune) Transition {
	// c0 control
	// TODO we add c0, verify it
	// miss it?
	if C0prime(r) {
		return Transition{ignore{}, nil}
	}

	// save param
	if r == 0x3B || (0x30 <= r && r <= 0x39) {
		return Transition{param{}, nil}
	}

	// csi ignore
	if r == 0x3A || (0x3C <= r && r <= 0x3F) {
		return Transition{ignore{}, dcsIgnore{}}
	}

	// goto dcs intermediate
	if 0x20 <= r && r <= 0x2F {
		return Transition{collect{}, dcsIntermediate{}}
	}

	// goto dcs passthrough
	if 0x40 <= r && r <= 0x7E {
		return Transition{ignore{}, dcsPassthrough{}}
	}

	// event 7F / ignore
	// TODO verify it
	if r == 0x7F {
		return Transition{ignore{}, nil}
	}
	return Transition{}
}

type dcsIntermediate struct{ state }

func (d dcsIntermediate) eventList(r rune) Transition {
	// c0 control
	// TODO we add c0, verify it
	// miss it?
	if C0prime(r) {
		return Transition{ignore{}, nil}
	}

	// collect
	if 0x20 <= r && r <= 0x2F {
		return Transition{collect{}, nil}
	}

	// goto dcs passthrough
	if 0x40 <= r && r <= 0x7E {
		return Transition{ignore{}, dcsPassthrough{}}
	}

	// goto dcs ignore
	if 0x30 <= r && r <= 0x3F {
		return Transition{ignore{}, dcsIgnore{}}
	}

	// event 7F / ignore
	// TODO verify it
	if r == 0x7F {
		return Transition{ignore{}, nil}
	}
	return Transition{}
}

type dcsPassthrough struct{ state }

func (d dcsPassthrough) enter() Action { return hook{} }
func (d dcsPassthrough) exit() Action  { return unhook{} }
func (d dcsPassthrough) eventList(r rune) Transition {
	// how does the hook works?
	// TODO verify it: should put 7E in it?
	if C0prime(r) || (0x20 <= r && r <= 0x7E) {
		return Transition{put{}, nil}
	}

	// finish
	if r == 0x9C {
		return Transition{ignore{}, ground{}}
	}

	// event 7F / ignore
	// TODO verify it
	if r == 0x7F {
		return Transition{ignore{}, nil}
	}
	return Transition{}
}

type dcsIgnore struct{ state }

func (d dcsIgnore) eventList(r rune) Transition {
	// add this according to https://vt100.net/emu/dec_ansi_parser
	// TODO verify it
	if C0prime(r) || (0x20 <= r && r <= 0x7F) {
		return Transition{put{}, nil}
	}

	if r == 0x9C {
		return Transition{ignore{}, ground{}}
	}

	return Transition{}
}

type oscString struct{ state }

func (o oscString) enter() Action { return oscStart{} }
func (o oscString) exit() Action  { return oscEnd{} }
func (o oscString) eventList(r rune) Transition {
	// add this according to https://vt100.net/emu/dec_ansi_parser
	// conflict with the following 0x07?
	// TODO  verify it
	if C0prime(r) {
		return Transition{ignore{}, nil}
	}

	if 0x20 <= r && r <= 0x7F {
		return Transition{oscPut{}, nil}
	}

	if r == 0x9C || r == 0x07 { // 0x07 is xterm non-ANSI variant
		return Transition{ignore{}, ground{}}
	}

	return Transition{}
}

type sosPmApcString struct{ state }

func (s sosPmApcString) eventList(r rune) Transition {
	// add this according to https://vt100.net/emu/dec_ansi_parser
	// TODO verify it
	if C0prime(r) || (0x20 <= r && r <= 0x7F) {
		return Transition{ignore{}, nil}
	}

	if r == 0x9C {
		return Transition{ignore{}, ground{}}
	}

	return Transition{}
}
