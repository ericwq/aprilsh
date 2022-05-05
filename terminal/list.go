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

type actionFunc func(Emulator, Action) string

var actionList = struct {
	sync.Mutex
	list map[string]actionFunc
}{
	list: make(map[string]actionFunc),
}

func lookupActionByName(key string) (actionFunc, error) {
	actionList.Lock()
	defer actionList.Unlock()

	if function, ok := actionList.list[key]; ok {
		return function, nil
	}
	return nil, fmt.Errorf("unrecognized action name:%s", key)
}

func init() {
	actionList.Lock()

	actionList.list[ACTION_IGNORE] = ignoreAction
	actionList.list[ACTION_PRINT] = printAction
	actionList.list[ACTION_EXECUTE] = executeAction
	actionList.list[ACTION_CLEAR] = clearAction
	actionList.list[ACTION_COLLECT] = collectAction
	actionList.list[ACTION_PARAM] = paramAction
	actionList.list[ACTION_ESC_DISPATCH] = escDispatchAction
	actionList.list[ACTION_CSI_DISPATCH] = csiDispatchAction
	actionList.list[ACTION_HOOK] = hookAction
	actionList.list[ACTION_PUT] = putAction
	actionList.list[ACTION_UNHOOK] = unhookAction
	actionList.list[ACTION_OSC_START] = oscStartAction
	actionList.list[ACTION_OSC_PUT] = oscPutAction
	actionList.list[ACTION_OSC_END] = oscEndAction
	actionList.list[ACTION_USER_BYTE] = userByteAction
	actionList.list[ACTION_RESIZE] = resizeAction

	defer actionList.Unlock()
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
