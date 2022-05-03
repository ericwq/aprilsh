package parser

import "fmt"

type Terminal interface {
	Print(x Action)
}

type Action interface {
	ActOn(t Terminal)
	Ignore() bool
	Name() string
}

// action implement both Action interface and accessAction interface
type action struct {
	ch      rune
	present bool
}

func (a action) ActOn(t Terminal)   {}               // do nothing
func (a action) Ignore() bool       { return false } // do not ignore us
func (a action) Name() string       { return "" }
func (a *action) setChar(r rune)    { a.ch = r }
func (a *action) setPresent(b bool) { a.present = b }
func (a *action) getChar() rune     { return a.ch }
func (a *action) isPresent() bool   { return a.present }

// allow the interface value to access the field value
type accessAction interface {
	setChar(rune)
	setPresent(bool)
	getChar() rune
	isPresent() bool
}

type ignore struct {
	action
}

func (i ignore) Ignore() bool { return true } // ignore this action
func (i ignore) Name() string { return "Ignore" }

type print struct {
	action
}

func (p print) ActOn(t Terminal) { fmt.Printf("%s: print on terminal\n", p.Name()) }
func (p print) Name() string     { return "Print" }

type execute struct {
	action
}

func (e execute) ActOn(t Terminal) { fmt.Printf("%s: execute on terminal\n", e.Name()) }
func (e execute) Name() string     { return "Execute" }

type clear struct {
	action
}

func (c clear) ActOn(t Terminal) { fmt.Printf("%s: execute on terminal\n", c.Name()) }
func (c clear) Name() string     { return "Clear" }

type collect struct {
	action
}

func (c collect) ActOn(t Terminal) { fmt.Printf("%s: execute on terminal\n", c.Name()) }
func (c collect) Name() string     { return "Collect" }

type param struct {
	action
}

func (p param) ActOn(t Terminal) { fmt.Printf("%s: execute on terminal\n", p.Name()) }
func (p param) Name() string     { return "Param" }

type escDispatch struct {
	action
}

func (ed escDispatch) ActOn(t Terminal) { fmt.Printf("%s: execute on terminal\n", ed.Name()) }
func (ed escDispatch) Name() string     { return "ESCdispatch" }

type csiDispatch struct {
	action
}

func (cd csiDispatch) ActOn(t Terminal) { fmt.Printf("%s: execute on terminal\n", cd.Name()) }
func (cd csiDispatch) Name() string     { return "CSIdispatch" }

type hook struct {
	action
}

func (h hook) Name() string { return "Hook" }

type put struct {
	action
}

func (p put) Name() string { return "Put" }

type unhook struct {
	action
}

func (u unhook) Name() string { return "Unhook" }

type oscStart struct {
	action
}

func (os oscStart) ActOn(t Terminal) { fmt.Printf("%s: execute on terminal\n", os.Name()) }
func (os oscStart) Name() string     { return "OSCstart" }

type oscPut struct {
	action
}

func (op oscPut) ActOn(t Terminal) { fmt.Printf("%s: execute on terminal\n", op.Name()) }
func (op oscPut) Name() string     { return "OSCput" }

type oscEnd struct {
	action
}

func (oe oscEnd) ActOn(t Terminal) { fmt.Printf("%s: execute on terminal\n", oe.Name()) }
func (oe oscEnd) Name() string     { return "OSCend" }

type UserByte struct {
	c rune
	action
}

func (ub UserByte) ActOn(t Terminal) { fmt.Printf("%s: execute on terminal\n", ub.Name()) }
func (ub UserByte) Name() string     { return "UserByte" }

type Resize struct {
	width  int
	height int
	action
}

func (r Resize) ActOn(t Terminal) { fmt.Printf("%s: execute on terminal\n", r.Name()) }
func (r Resize) Name() string     { return "Resize" }
