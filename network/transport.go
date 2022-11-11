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
	"errors"
	"fmt"
	"time"
)

// type S or R that must implement the State[T] interface - that is, for itself.
type Transport[S State[S], R State[R]] struct {
	// the underlying, encrypted network connection
	connection *Connection

	// sender side
	sender *TransportSender[S]

	// simple receiver
	receivedState       []TimestampedState[R]
	receiverQuenchTimer int64
	lastReceiverState   R // the state we were in when user last queried state
	fragments           *FragmentAssembly
	verbose             uint
}

func NewTransportServer[S State[S], R State[R]](initialState S, initialRemote R,
	desiredIp, desiredPort string,
) *Transport[S, R] {
	ts := &Transport[S, R]{}
	ts.connection = NewConnection(desiredIp, desiredPort)
	ts.sender = NewTransportSender(ts.connection, initialState)

	ts.receivedState = make([]TimestampedState[R], 0)
	ts.receivedState = append(ts.receivedState,
		TimestampedState[R]{time.Now().UnixMilli(), 0, initialRemote})

	ts.lastReceiverState = initialRemote
	ts.fragments = NewFragmentAssembly()
	return ts
}

func NewTransportClient[S State[S], R State[R]](initialState S, initialRemote R,
	keyStr, ip, port string,
) *Transport[S, R] {
	tc := &Transport[S, R]{}
	tc.connection = NewConnectionClient(keyStr, ip, port)
	tc.sender = NewTransportSender(tc.connection, initialState)

	tc.receivedState = make([]TimestampedState[R], 0)
	tc.receivedState = append(tc.receivedState,
		TimestampedState[R]{time.Now().UnixMilli(), 0, initialRemote})

	tc.lastReceiverState = initialRemote
	tc.fragments = NewFragmentAssembly()
	return tc
}

// The sender uses throwawayNum to tell us the earliest received state that
// we need to keep around
func (t *Transport[S, R]) processThrowawayUntil(throwawayNum int64) {
	rs := t.receivedState[:0]
	for i := range t.receivedState {
		if t.receivedState[i].num < throwawayNum {
			// skip means erase this element
		} else {
			rs = append(rs, t.receivedState[i])
		}
	}
	t.receivedState = rs
}

// Send data or an ack if necessary.
func (t *Transport[S, R]) tick() {
	t.sender.tick()
}

// Returns the number of ms to wait until next possible event.
func (t *Transport[S, R]) waitTime() int {
	return t.sender.waitTime()
}

// Blocks waiting for a packet.
func (t *Transport[S, R]) recv() error {
	s, err := t.connection.recv()
	if err != nil {
		return err
	}
	frag := NewFragmentFrom(s)

	if t.fragments.addFragment(frag) { // complete packet
		inst := t.fragments.getAssembly()

		if inst.ProtocolVersion != APRILSH_PROTOCOL_VERSION {
			return errors.New("aprilsh protocol version mismatch.")
		}

		// remove the state for which num < AckNum
		t.sender.processAcknowledgmentThrough(inst.AckNum)

		// inform network layer of roundtrip (end-to-end-to-end) connectivity
		t.connection.setLastRoundtripSuccess(t.sender.getSentStateAckedTimestamp())

		// first, make sure we don't already have the new state
		for i := range t.receivedState {
			if inst.NewNum == t.receivedState[i].num {
				return nil
			}
		}

		// now, make sure we do have the old state
		found := false
		refStateIdx := 0
		for refStateIdx = range t.receivedState {
			if inst.OldNum == t.receivedState[refStateIdx].num {
				found = true
				break
			}
		}

		if !found {
			fmt.Printf("Ignoring out-of-order packet. Reference state %d has been "+
				"discarded or hasn't yet been received.\n", inst.OldNum)
			return nil // this is security-sensitive and part of how we enforce idempotency
		}

		// Do not accept state if our queue is full.
		//
		// This is better than dropping states from the middle of the
		// queue (as sender does), because we don't want to ACK a state
		// and then discard it later.
		t.processThrowawayUntil(inst.ThrowawayNum)

		if len(t.receivedState) > 1024 { // limit on state queue
			now := time.Now().UnixMilli()
			if now < t.receiverQuenchTimer { // deny letting state grow further
				if t.verbose > 0 {
					fmt.Printf("[%d] Receiver queue full, discarding %d (malicious sender or "+
						"long-unidirectional connectivity?)\n", now%100000, inst.NewNum)
				}
				return nil
			} else {
				t.receiverQuenchTimer = now + 15000
			}
		}

		// apply diff to reference state
		newState := t.receivedState[refStateIdx] // maybe we need to clone the state
		newState.timestamp = time.Now().UnixMilli()
		newState.num = inst.NewNum
		if len(inst.Diff) > 0 {
			newState.state.ApplyString(string(inst.Diff))
		}

		// Insert new state in sorted place
		rs := t.receivedState[:0]
		for i := range t.receivedState {
			if t.receivedState[i].num > newState.num {
				// insert out-of-order new state
				rs = append(rs, newState)
				rs = append(rs, t.receivedState[i:]...)
				t.receivedState = rs

				if t.verbose > 0 {
					fmt.Printf("[%d] Received OUT-OF-ORDER state %d [ack %d]\n",
						time.Now().UnixMilli()%100000, newState.num, inst.AckNum)
				}
				return nil
			}
			rs = append(rs, t.receivedState[i])
		}
		if t.verbose > 0 {
			fmt.Printf("[%d] Received state %d [coming from %d, ack %d]\n",
				time.Now().UnixMilli()%100000, newState.num, inst.OldNum, inst.AckNum)
		}
		t.receivedState = append(t.receivedState, newState) // insert new state
		t.sender.setAckNum(t.receivedState[len(t.receivedState)-1].num)

		t.sender.remoteHeard(newState.timestamp)
		if len(inst.Diff) > 0 {
			t.sender.setDataAck()
		}
	}
	return nil
}

func (t *Transport[S, R]) getRemoteDiff() string {
	// find diff between last receiver state and current remote state, then rationalize states
	back := len(t.receivedState) - 1
	ret := t.receivedState[back].state.DiffFrom(t.lastReceiverState)

	oldestReceivedState := t.receivedState[0].state
	for i := back; i >= 0; i-- {
		t.receivedState[i].state.Subtract(oldestReceivedState)
	}

	t.lastReceiverState = t.receivedState[back].state
	return ret
}

func (t *Transport[S, R]) getCurrentState() S {
	return t.sender.getCurrentState()
}

func (t *Transport[S, R]) setCurrentState(x S) {
	t.sender.setCurrentState(x)
}

func (t *Transport[S, R]) getLatestRemoteState() R {
	last := len(t.receivedState) - 1
	return t.receivedState[last].state
}
