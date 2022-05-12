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
	DISPATCH_ESCAPE = iota + 1
	DISPATCH_CSI
	DISPATCH_CONTROL
)

const (
	Charset_UTF8 = iota // sync w/charCodes definition!
	Charset_DecSpec
	Charset_DecSuppl
	Charset_DecUserPref
	Charset_DecTechn
	Charset_IsoLatin1
	Charset_IsoUK
)

type Emulator interface {
	Print(act Action)
	Execute(act Action)
	Dispatch() *Dispatcher
	CSIdispatch(act Action)
	ESCdispatch(act Action)
	OSCend(act Action)
	Resize(width int, height int)
	User() *UserInput
	Framebuffer() *Framebuffer
}

type Action interface {
	ActOn(t Emulator)
	Ignore() bool
	Name() string
	SetChar(rune)
	SetPresent(bool)
	GetChar() rune
	IsPresent() bool
}

type CharsetState struct {
	// charset g0,g1,g2,g3
	g [4]int

	// Locking shift states (index into g[]):
	gl int
	gr int

	// Single shift state (0 if none active):
	// 0 - not active; 2: G2 in GL; 3: G3 in GL
	ss int
}

type emulator struct {
	dispatcher   Dispatcher
	framebuffer  Framebuffer
	charsetState CharsetState
}

func NewEmulator() *emulator {
	emu := &emulator{}
	emu.charsetState.g[0] = Charset_UTF8
	emu.charsetState.g[1] = Charset_UTF8
	emu.charsetState.g[2] = Charset_UTF8
	emu.charsetState.g[3] = Charset_UTF8

	emu.charsetState.gl = 0 // G0 in GL
	emu.charsetState.gr = 2 // G2 in GR

	emu.charsetState.ss = 0
	return emu
}

/*
func (e *emulator) CSIdispatch(act Action) {
	e.dispatcher.dispatch(DISPATCH_CSI, act, &e.framebuffer)
}

func (e *emulator) ESCdispatch(act Action) {
	ch := act.GetChar()

	// handle 7-bit ESC-encoding of C1 control characters
	if len(e.dispatcher.getDispatcherChars()) == 0 && 0x40 <= ch && ch <= 0x5F {
		// convert 7-bit esc sequence into 8-bit c1 control sequence
		// TODO consider remove 8-bit c1 control
		act2 := escDispatch{action{ch + 0x40, true}}
		e.dispatcher.dispatch(DISPATCH_CONTROL, &act2, &e.framebuffer)
	} else {
		e.dispatcher.dispatch(DISPATCH_ESCAPE, act, &e.framebuffer)
	}
}

func (e *emulator) OSCdispatch(act Action) {
	e.dispatcher.oscDispatch(act, &e.framebuffer)
}

func (e *emulator) OSCend(act Action) {
}

func (e *emulator) Resize(act Action) {
}

*/
