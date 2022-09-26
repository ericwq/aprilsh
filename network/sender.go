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

package network

type TransportSender[T State[T]] struct {
	currentState         T
	sendStates           []TimestampedState[T]
	assumedReceiverState *TimestampedState[T]
}

func (t *TransportSender[T]) addSendState(theTimestamp int64, num int64, state T) {
}

func (t *TransportSender[T]) getCurrentState() T {
	return t.currentState
}

// TODO careful about the pointer
func (t *TransportSender[T]) setCurrentState(x T) {
	t.currentState = x
	t.currentState.ResetInput()
}

// func NewTransportSender2() *TransportSender[CompleteTerminal] {
// 	ts := TransportSender[CompleteTerminal]{}
// 	prefix := new(CompleteTerminal)
// 	ts.sendStates[3].state.subtract(prefix)
// 	return &ts
// }

// type TransportSender2 struct {
// 	currentState         State
// 	sendStates           []TimestampedState2
// 	assumedReceiverState *TimestampedState2
// }
