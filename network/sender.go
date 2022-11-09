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

import "time"

type TransportSender[T State[T]] struct {
	// state of sender
	connection   *Connection
	currentState T
	// first element: known, acknowledged receiver sentStates
	// last element: last sent state
	sentStates []TimestampedState[T]

	// somewhere in the middle: the assumed state of the receiver
	assumedReceiverState *TimestampedState[T]

	// for fragment creation
	fragmenter *Fragmenter

	// timing state
	nextAckTime  int64
	nextSendTime int64

	verbose            uint
	shutdownInProgress bool
	shutdownTries      int
	shutdownStart      int64

	// information about receiver state
	ackNum         int64
	pendingDataAct bool
	SEND_MINDELAY  uint  // ms to collect all input
	lastHeard      int64 // last time received new state

	mindelayClock int64 // time of first pending change to current state
}

func NewTransportSender[T State[T]](connection *Connection, initialState T) *TransportSender[T] {
	ts := &TransportSender[T]{}
	ts.connection = connection
	ts.currentState = initialState
	ts.sentStates = make([]TimestampedState[T], 0)
	ts.assumedReceiverState = &ts.sentStates[0]

	ts.fragmenter = NewFragmenter()

	ts.nextAckTime = time.Now().UnixMilli()
	ts.nextSendTime = time.Now().UnixMilli()

	ts.shutdownStart = -1
	ts.SEND_MINDELAY = 8
	ts.mindelayClock = -1
	return ts
}

// Send data or an ack if necessary
func (ts *TransportSender[T]) tick() {
}

func (ts *TransportSender[T]) calculateTimers() {
	// now := time.Now().UnixMilli()
}

func (ts *TransportSender[T]) updateAssumedReceiverState() {
	// now := time.Now().UnixMilli()
	//
	// // start from what is known and give benefit of the doubt to unacknowledged states
	// // transmitted recently enough ago
	//
	// ts.assumedReceiverState = &ts.sentStates[0]
	// for i := range ts.sentStates {
	// 	// if now - ts.sentStates[i].timestamp< ts.connection.timout
	// }
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
