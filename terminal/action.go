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
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE. */

package terminal

const (
	ACTION_IGNORE       = "Ignore"
	ACTION_PRINT        = "Print"
	ACTION_EXECUTE      = "Execute"
	ACTION_CLEAR        = "Clear"
	ACTION_COLLECT      = "Collect"
	ACTION_PARAM        = "Param"
	ACTION_ESC_DISPATCH = "ESC_Dispatch"
	ACTION_CSI_DISPATCH = "CSI_Dispatch"
	ACTION_HOOK         = "Hook"
	ACTION_PUT          = "Put"
	ACTION_UNHOOK       = "Unhook"
	ACTION_OSC_START    = "OSC_Start"
	ACTION_OSC_PUT      = "OSC_Put"
	ACTION_OSC_END      = "OSC_End"
	ACTION_USER_BYTE    = "UserByte"
	ACTION_RESIZE       = "Resize"
)

// action implement both Action interface and accessAction interface
type action struct {
	ch      rune
	present bool
}

func (a action) Ignore() bool { return false } // do not ignore us
func (a action) Name() string { return "" }

// find the action function
func (a action) ActOn(t Emulator) {
	// ignore return error
	act, _ := lookupActionByName(a.Name())

	/// call the action
	act(t, a)
}

func (a *action) SetChar(r rune)    { a.ch = r }
func (a *action) SetPresent(b bool) { a.present = b }
func (a *action) GetChar() rune     { return a.ch }
func (a *action) IsPresent() bool   { return a.present }

type ignore struct {
	action
}

func (i ignore) Ignore() bool { return true } // ignore this action
func (i ignore) Name() string { return ACTION_IGNORE }

type print struct {
	action
}

// func (p print) ActOn(t Emulator) { fmt.Printf("%s: print on terminal\n", p.Name()) }
func (p print) Name() string { return ACTION_PRINT }

type execute struct {
	action
}

// func (e execute) ActOn(t Emulator) { fmt.Printf("%s: execute on terminal\n", e.Name()) }
func (e execute) Name() string { return ACTION_EXECUTE }

type clear struct {
	action
}

// func (c clear) ActOn(t Emulator) { fmt.Printf("%s: execute on terminal\n", c.Name()) }
func (c clear) Name() string { return ACTION_CLEAR }

type collect struct {
	action
}

// func (c collect) ActOn(t Emulator) { fmt.Printf("%s: execute on terminal\n", c.Name()) }
func (c collect) Name() string { return ACTION_COLLECT }

type param struct {
	action
}

// func (p param) ActOn(t Emulator) { fmt.Printf("%s: execute on terminal\n", p.Name()) }
func (p param) Name() string { return ACTION_PARAM }

type escDispatch struct {
	action
}

// func (ed escDispatch) ActOn(t Emulator) { fmt.Printf("%s: execute on terminal\n", ed.Name()) }
func (ed escDispatch) Name() string { return ACTION_ESC_DISPATCH }

type csiDispatch struct {
	action
}

// func (cd csiDispatch) ActOn(t Emulator) { fmt.Printf("%s: execute on terminal\n", cd.Name()) }
func (cd csiDispatch) Name() string { return ACTION_CSI_DISPATCH }

type hook struct {
	action
}

func (h hook) Name() string { return ACTION_HOOK }

type put struct {
	action
}

func (p put) Name() string { return ACTION_PUT }

type unhook struct {
	action
}

func (u unhook) Name() string { return ACTION_UNHOOK }

type oscStart struct {
	action
}

// func (os oscStart) ActOn(t Emulator) { fmt.Printf("%s: execute on terminal\n", os.Name()) }
func (os oscStart) Name() string { return ACTION_OSC_START }

type oscPut struct {
	action
}

// func (op oscPut) ActOn(t Emulator) { fmt.Printf("%s: execute on terminal\n", op.Name()) }
func (op oscPut) Name() string { return ACTION_OSC_PUT }

type oscEnd struct {
	action
}

// func (oe oscEnd) ActOn(t Emulator) { fmt.Printf("%s: execute on terminal\n", oe.Name()) }
func (oe oscEnd) Name() string { return ACTION_OSC_END }

type UserByte struct {
	c rune
	action
}

// func (ub UserByte) ActOn(t Emulator) { fmt.Printf("%s: execute on terminal\n", ub.Name()) }
func (ub UserByte) Name() string { return ACTION_USER_BYTE }

type Resize struct {
	width  int
	height int
	action
}

// func (r Resize) ActOn(t Emulator) { fmt.Printf("%s: execute on terminal\n", r.Name()) }
func (r Resize) Name() string { return ACTION_RESIZE }
