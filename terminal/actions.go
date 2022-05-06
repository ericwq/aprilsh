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
	"sync"
)

type parserActionFunc func(Emulator, Action) string

var parserActions = struct {
	sync.Mutex
	actions map[string]parserActionFunc
}{
	actions: make(map[string]parserActionFunc,16),
}

func lookupActionByName(key string) (parserActionFunc, error) {
	parserActions.Lock()
	defer parserActions.Unlock()

	if function, ok := parserActions.actions[key]; ok {
		return function, nil
	}
	return nil, fmt.Errorf("unrecognized action name:%s", key)
}

func init() {
	parserActions.Lock()

	parserActions.actions[ACTION_IGNORE] = ignoreAction
	parserActions.actions[ACTION_PRINT] = printAction
	parserActions.actions[ACTION_EXECUTE] = executeAction
	parserActions.actions[ACTION_CLEAR] = clearAction
	parserActions.actions[ACTION_COLLECT] = collectAction
	parserActions.actions[ACTION_PARAM] = paramAction
	parserActions.actions[ACTION_ESC_DISPATCH] = escDispatchAction
	parserActions.actions[ACTION_CSI_DISPATCH] = csiDispatchAction
	parserActions.actions[ACTION_HOOK] = hookAction
	parserActions.actions[ACTION_PUT] = putAction
	parserActions.actions[ACTION_UNHOOK] = unhookAction
	parserActions.actions[ACTION_OSC_START] = oscStartAction
	parserActions.actions[ACTION_OSC_PUT] = oscPutAction
	parserActions.actions[ACTION_OSC_END] = oscEndAction
	parserActions.actions[ACTION_USER_BYTE] = userByteAction
	parserActions.actions[ACTION_RESIZE] = resizeAction

	defer parserActions.Unlock()
}

// Now the action function is just for testing
func ignoreAction(t Emulator, a Action) string      { return a.Name() }
func printAction(t Emulator, a Action) string       { return a.Name() }
func executeAction(t Emulator, a Action) string     { return a.Name() }
func clearAction(t Emulator, a Action) string       { return a.Name() }
func collectAction(t Emulator, a Action) string     { return a.Name() }
func paramAction(t Emulator, a Action) string       { return a.Name() }
func escDispatchAction(t Emulator, a Action) string { return a.Name() }
func csiDispatchAction(t Emulator, a Action) string { return a.Name() }
func hookAction(t Emulator, a Action) string        { return a.Name() }
func putAction(t Emulator, a Action) string         { return a.Name() }
func unhookAction(t Emulator, a Action) string      { return a.Name() }
func oscStartAction(t Emulator, a Action) string    { return a.Name() }
func oscPutAction(t Emulator, a Action) string      { return a.Name() }
func oscEndAction(t Emulator, a Action) string      { return a.Name() }
func userByteAction(t Emulator, a Action) string    { return a.Name() }
func resizeAction(t Emulator, a Action) string      { return a.Name() }
