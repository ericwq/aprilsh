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
	"strings"
)

// const (
// 	PARAM_MAX = 65535
// )

type Dispatcher struct {
	// params         string
	// params         strings.Builder
	// parsedParams   []int
	// parsed         bool
	// dispatcherChar strings.Builder
	// oscString      strings.Builder
	terminalToHost strings.Builder
}

// func (d *Dispatcher) clear(Action) {
// 	d.params.Reset()
// 	d.dispatcherChar.Reset()
// 	d.parsed = false
// }
//
// // newParamChar() requres a &Action value
// func (d *Dispatcher) newParamChar(act Action) {
// 	if d.params.Len() < 100 {
// 		// enough for 16 five-char params plus 15 semicolons
// 		// max 16 parameter, every parameter < 65535
// 		// ensure the above rule at parseAll function
// 		d.params.WriteRune(act.GetChar())
// 	}
//
// 	d.parsed = false
// }
//
// func (d *Dispatcher) getDispatcherChars() string { return d.dispatcherChar.String() }
// func (d *Dispatcher) collect(act Action) {
// 	if d.dispatcherChar.Len() < 8 && // should never exceed 2
// 		act.GetChar() <= 0xFF { // ignore non-8-bit
// 		d.dispatcherChar.WriteRune(act.GetChar())
// 	}
// }

// parse "12;23" into []int{12, 34}
// parse "34:45" into []int{34, 45}
// corner case such as ";1;2;" will result []int{-1, 1, 2, -1}
// func (d *Dispatcher) parseAll() {
// 	// if d.parsed {
// 	// 	return
// 	// }
//
// 	// default capability is 6
// 	d.parsedParams = make([]int, 0, 6)
//
// 	// transfer :(0x3A) to ;(0x3B)
// 	params := strings.ReplaceAll(d.params.String(), ":", ";")
// 	pSlice := strings.Split(params, ";")
//
// 	value := -1
// 	for _, str := range pSlice {
//
// 		if v, err := strconv.Atoi(str); err == nil {
// 			value = v
// 			if value > PARAM_MAX {
// 				value = -1
// 			}
// 		} else {
// 			value = -1
// 		}
//
// 		d.parsedParams = append(d.parsedParams, value)
// 	}
//
// 	d.parsed = true
// }
//
// // get number n parameter from escape sequence buffer
// // if the return parameter is zero, use the defaultVal instead
// func (d *Dispatcher) getParam(n, defaultVal int) int {
// 	ret := defaultVal
// 	if !d.parsed {
// 		d.parseAll()
// 	}
//
// 	if len(d.parsedParams) > n {
// 		ret = d.parsedParams[n]
// 	}
//
// 	if ret < 1 {
// 		ret = defaultVal
// 	}
//
// 	return ret
// }
//
// func (d *Dispatcher) getParamCount() int {
// 	if !d.parsed {
// 		d.parseAll()
// 	}
//
// 	return len(d.parsedParams)
// }
//
// func (d *Dispatcher) getOSCstring() string { return d.oscString.String() }
// func (d *Dispatcher) oscPut(act Action) {
// 	if d.oscString.Len() < 256 { // should be long enough for window title
// 		d.oscString.WriteRune(act.GetChar())
// 	}
// }
//
// func (d *Dispatcher) oscStart(Action) {
// 	d.oscString.Reset()
// }
//
// func (d *Dispatcher) dispatch(funcType int, act Action, fb *Framebuffer) {
// 	key := ""
//
// 	switch funcType {
// 	case DISPATCH_ESCAPE, DISPATCH_CSI:
// 		// add final char to dispatch key
// 		act2 := collect{action{act.GetChar(), true}}
// 		d.collect(&act2)
// 		key = d.dispatcherChar.String()
// 	case DISPATCH_CONTROL:
// 		key = string(act.GetChar())
// 	}
//
// 	emuFunc := findFunctionBy(funcType, key)
// 	if emuFunc.function != nil { // nil: not find
// 		// unkown function
// 		fb.DS.NextPrintWillWrap = false
// 	} else {
// 		if emuFunc.clearsWrapState {
// 			fb.DS.NextPrintWillWrap = false
// 		}
// 		emuFunc.function(fb, d)
// 	}
// }
//
// // xterm uses an Operating System Command to set the window title
// // consider to add more useful OSC command: such as OSC 52
// // https://invisible-island.net/xterm/ctlseqs/ctlseqs.html#h3-Operating-System-Commands
// func (d *Dispatcher) oscDispatch(_ Action, fb *Framebuffer) {
// 	oscStr := d.oscString.String()
// 	if len(oscStr) >= 1 {
//
// 		offset := strings.Index(oscStr, ";")
// 		cmdNum := -1
//
// 		if offset == 0 {
// 			// OSC of the form "\033];<title>\007"
// 			cmdNum = 0
// 			offset += 1
// 		} else { // it must be 1
// 			// OSC of the form "\033]X;<title>\007" where X can be:
// 			//* 0: set icon name and window title
// 			//* 1: set icon name
// 			//* 2: set window title
// 			if i, err := strconv.Atoi(oscStr[:offset]); err == nil {
// 				if 0 <= i && i <= 2 {
// 					// only support OSC 0,1,2
// 					cmdNum = i
// 				} else {
// 					// ignore other OSC command
// 					return
// 				}
// 			}
// 			offset += 1
// 		}
// 		setIcon := cmdNum == 0 || cmdNum == 1
// 		setTitle := cmdNum == 0 || cmdNum == 2
// 		oscStr = oscStr[offset:]
//
// 		if setIcon || setTitle {
// 			fb.SetTitleInitialized()
//
// 			if setIcon {
// 				fb.SetIconName(oscStr)
// 			}
//
// 			if setTitle {
// 				fb.SetWindowTitle(oscStr)
// 			}
// 		}
// 	}
// }
