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

import (
	"fmt"
)

type UserStream struct {
	action []string
}

func (u *UserStream) subtract(prefix *UserStream) {
	fmt.Println("#UserStream subtract")
}

func (u *UserStream) diffFrom(prefix *UserStream) {
	fmt.Println("#UserStream subtract")
}

type CompleteTerminal struct {
	action []string
}

func (u *CompleteTerminal) subtract(prefix *CompleteTerminal) {
	fmt.Println("#CompleteTerminal subtract")
}

type State interface {
	UserStream | CompleteTerminal
}

var tx Transport[UserStream, CompleteTerminal]

type Transport[L State, R State] struct {
	sender        TransportSender[L]
	receivedState []TimestampedState[R]
}

func NewTransport2() *Transport[UserStream, CompleteTerminal] {
	t := Transport[UserStream, CompleteTerminal]{}
	t.receivedState[4].numEq(7)
	return &t
}

type TransportSender[S State] struct {
	sendStates   []TimestampedState[S]
	currentState S
}

func (ts *TransportSender[S]) xxx() {
	tx.sender.currentState.diffFrom(&tx.sender.sendStates[2].state)
}

func NewTransportSender2() *TransportSender[CompleteTerminal] {
	ts := TransportSender[CompleteTerminal]{}
	prefix := new(CompleteTerminal)
	ts.sendStates[3].state.subtract(prefix)
	return &ts
}

type TimestampedState[S State] struct {
	timestamp int64
	num       int64
	state     S
}

func NewTimestampedState2() *TimestampedState[UserStream] {
	ts := TimestampedState[UserStream]{}
	return &ts
}

func (t *TimestampedState[R]) numEq(v int64) bool {
	return t.num == v
}

func (t *TimestampedState[R]) numLt(v int64) bool {
	return t.num < v
}
