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
// // TODO prepare to delete this file
// // append action to action list except ignore action
// func appendTo(actions []Action, act Action) []Action {
// 	if !act.Ignore() {
// 		actions = append(actions, act)
// 	}
// 	return actions
// }
//
// // parse the input character into action and save it in action list
// // it's uesed to be input
// func (p *Parser) parse(actions []Action, r rune) []Action {
// 	// start to parse
// 	ts := p.state.parse(r)
//
// 	// exit action from old state
// 	if ts.nextState != nil {
// 		actions = appendTo(actions, p.state.exit())
// 	}
//
// 	// transition action
// 	actions = appendTo(actions, ts.action)
// 	ts.action = nil
//
// 	// enter action to new state
// 	if ts.nextState != nil {
// 		actions = appendTo(actions, ts.nextState.enter())
// 		// transition to next state
// 		p.state = ts.nextState
// 	}
//
// 	return actions
// }
