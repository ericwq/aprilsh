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


const (
	ignoreName      = "Ignore"
	printName       = "Print"
	executeName     = "Execute"
	clearName       = "Clear"
	collectName     = "Collect"
	paramName       = "Param"
	escDispatchName = "ESCdispatch"
	csiDispatchName = "CSIdispatch"
	hookName        = "Hook"
	putName         = "Put"
	unhookName      = "Unhook"
	oscStartName    = "OSCstart"
	oscPutName      = "OSCput"
	oscEndName      = "OSCEnd"
	userByteName    = "UserByte"
	resizeName      = "Resize"
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
func (i ignore) Name() string { return ignoreName }

type print struct {
	action
}

// func (p print) ActOn(t Emulator) { fmt.Printf("%s: print on terminal\n", p.Name()) }
func (p print) Name() string { return printName }

type execute struct {
	action
}

// func (e execute) ActOn(t Emulator) { fmt.Printf("%s: execute on terminal\n", e.Name()) }
func (e execute) Name() string { return executeName }

type clear struct {
	action
}

// func (c clear) ActOn(t Emulator) { fmt.Printf("%s: execute on terminal\n", c.Name()) }
func (c clear) Name() string { return clearName }

type collect struct {
	action
}

// func (c collect) ActOn(t Emulator) { fmt.Printf("%s: execute on terminal\n", c.Name()) }
func (c collect) Name() string { return collectName }

type param struct {
	action
}

// func (p param) ActOn(t Emulator) { fmt.Printf("%s: execute on terminal\n", p.Name()) }
func (p param) Name() string { return paramName }

type escDispatch struct {
	action
}

// func (ed escDispatch) ActOn(t Emulator) { fmt.Printf("%s: execute on terminal\n", ed.Name()) }
func (ed escDispatch) Name() string { return escDispatchName }

type csiDispatch struct {
	action
}

// func (cd csiDispatch) ActOn(t Emulator) { fmt.Printf("%s: execute on terminal\n", cd.Name()) }
func (cd csiDispatch) Name() string { return csiDispatchName }

type hook struct {
	action
}

func (h hook) Name() string { return hookName }

type put struct {
	action
}

func (p put) Name() string { return putName }

type unhook struct {
	action
}

func (u unhook) Name() string { return unhookName }

type oscStart struct {
	action
}

// func (os oscStart) ActOn(t Emulator) { fmt.Printf("%s: execute on terminal\n", os.Name()) }
func (os oscStart) Name() string { return oscStartName }

type oscPut struct {
	action
}

// func (op oscPut) ActOn(t Emulator) { fmt.Printf("%s: execute on terminal\n", op.Name()) }
func (op oscPut) Name() string { return oscPutName }

type oscEnd struct {
	action
}

// func (oe oscEnd) ActOn(t Emulator) { fmt.Printf("%s: execute on terminal\n", oe.Name()) }
func (oe oscEnd) Name() string { return oscEndName }

type UserByte struct {
	c rune
	action
}

// func (ub UserByte) ActOn(t Emulator) { fmt.Printf("%s: execute on terminal\n", ub.Name()) }
func (ub UserByte) Name() string { return userByteName }

type Resize struct {
	width  int
	height int
	action
}

// func (r Resize) ActOn(t Emulator) { fmt.Printf("%s: execute on terminal\n", r.Name()) }
func (r Resize) Name() string { return resizeName }
