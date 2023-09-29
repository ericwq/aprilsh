// Copyright 2022~2023 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package network

import (
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/ericwq/aprilsh/util"
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

func NewTransportServer[S State[S], R State[R]](initialState S, initialRemote R, desiredIp, desiredPort string) *Transport[S, R] {
	ts := &Transport[S, R]{}
	ts.connection = NewConnection(desiredIp, desiredPort)
	ts.sender = NewTransportSender(ts.connection, initialState)

	ts.receivedState = make([]TimestampedState[R], 0)
	ts.receivedState = append(ts.receivedState,
		TimestampedState[R]{time.Now().UnixMilli(), 0, initialRemote.Clone()})

	ts.lastReceiverState = ts.receivedState[0].state
	ts.fragments = NewFragmentAssembly()
	return ts
}

func NewTransportClient[S State[S], R State[R]](initialState S, initialRemote R, keyStr, ip, port string) *Transport[S, R] {
	tc := &Transport[S, R]{}
	tc.connection = NewConnectionClient(keyStr, ip, port)
	tc.sender = NewTransportSender(tc.connection, initialState)

	tc.receivedState = make([]TimestampedState[R], 0)
	tc.receivedState = append(tc.receivedState,
		TimestampedState[R]{time.Now().UnixMilli(), 0, initialRemote.Clone()})

	tc.lastReceiverState = tc.receivedState[0].state
	tc.fragments = NewFragmentAssembly()
	return tc
}

// The sender uses throwawayNum to tell us the earliest received state that
// we need to keep around
func (t *Transport[S, R]) processThrowawayUntil(throwawayNum uint64) {
	rs := t.receivedState[:0]
	for i := range t.receivedState {
		if t.receivedState[i].num >= throwawayNum {
			rs = append(rs, t.receivedState[i])
			// } else {
			// 	util.Log.With("num", t.receivedState[i].num).Debug("remove num")
		} // else condition means erase this element
	}
	t.receivedState = rs
}

// Send data or an ack if necessary.
func (t *Transport[S, R]) Tick() error {
	return t.sender.tick()
}

// Returns the number of ms to wait until next possible event.
func (t *Transport[S, R]) WaitTime() int {
	return t.sender.waitTime()
}

// Blocks waiting for a packet.
func (t *Transport[S, R]) Recv() error {
	s, err := t.connection.Recv()
	if err != nil {
		return err
	}

	return t.ProcessPayload(s)
}

func (t *Transport[S, R]) GetRemoteDiff() string {
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

func (t *Transport[S, R]) SetReadDeadline(ti time.Time) error {
	return t.connection.SetReadDeadline(ti)
}

// Other side has requested shutdown and we have sent one ACK
//
//	Illegal to change current_state after this.
func (t *Transport[S, R]) StartShutdown() {
	t.sender.startShutdown()
}

// return true if shutdown is started, otherwise false.
func (t *Transport[S, R]) ShutdownInProgress() bool {
	return t.sender.getShutdownInProgress()
}

// return true if the firt sent state num is -1, otherwise false.
func (t *Transport[S, R]) ShutdownAcknowledged() bool {
	return t.sender.getShutdownAcknowledged()
}

// return true if retries reach times limit or retries time out
func (t *Transport[S, R]) ShutdownAckTimedout() bool {
	return t.sender.shutdonwAckTimedout()
}

func (t *Transport[S, R]) HasRemoteAddr() bool {
	return t.connection.getHasRemoteAddr()
}

// Other side has requested shutdown and we have sent one ACK
func (t *Transport[S, R]) CounterpartyShutdownAckSent() bool {
	return t.sender.getCounterpartyShutdownAcknowledged()
}

func (t *Transport[S, R]) GetCurrentState() S {
	return t.sender.getCurrentState()
}

func (t *Transport[S, R]) SetCurrentState(x S) {
	t.sender.setCurrentState(x)
}

func (t *Transport[S, R]) GetLatestRemoteState() TimestampedState[R] {
	last := len(t.receivedState) - 1
	return t.receivedState[last]
}

func (t *Transport[S, R]) GetRemoteStateNum() uint64 {
	last := len(t.receivedState) - 1
	return t.receivedState[last].num
}

func (t *Transport[S, R]) SetVerbose(verbose uint) {
	t.sender.setVerbose(verbose)
	t.verbose = verbose
}

func (t *Transport[S, R]) SetSendDelay(delay uint) {
	t.sender.setSendDelay(delay)
}

func (t *Transport[S, R]) GetSentStateAckedTimestamp() int64 {
	return t.sender.getSentStateAckedTimestamp()
}

func (t *Transport[S, R]) GetSentStateAcked() uint64 {
	return t.sender.getSentStateAcked()
}

func (t *Transport[S, R]) GetSentStateLast() uint64 {
	return t.sender.getSentStateLast()
}

func (t *Transport[S, R]) SentInterval() uint {
	return t.sender.sendInterval()
}

func (t *Transport[S, R]) GetRemoteAddr() net.Addr {
	return t.connection.getRemoteAddr()
}

// func (t *Transport[S, R]) Port() string {
// 	return t.connection.port()
// }

func (t *Transport[S, R]) GetKey() string {
	return t.connection.getKey()
}

func (t *Transport[S, R]) Close() {
	t.connection.sock().Close()
}

func (t *Transport[S, R]) GetConnection() *Connection {
	return t.connection
}

func (t *Transport[S, R]) ProcessPayload(s string) error {
	frag := NewFragmentFrom(s)

	if t.fragments.addFragment(frag) { // complete packet
		inst := t.fragments.getAssembly()

		// if inst.NewNum == -1 {
		// 	util.Log.Debug("got shutdown request")
		// 	for i := range t.sender.sentStates {
		// 		util.Log.With("i", i).With("num", t.sender.sentStates[i].num).Debug("sentStates")
		// 	}
		// 	for i := range t.receivedState {
		// 		util.Log.With("i", i).With("num", t.receivedState[i].num).Debug("receivedState")
		// 	}
		// }

		if inst.ProtocolVersion != APRILSH_PROTOCOL_VERSION {
			return errors.New("aprilsh protocol version mismatch.")
		}

		util.Log.With("NewNum", inst.NewNum).
			With("OldNum", inst.OldNum).
			With("AckNum", inst.AckNum).
			With("throwawayNum", inst.ThrowawayNum).
			With("diffLength", len(inst.Diff)).
			Debug("got network message")

		// remove send state for which num < AckNum
		util.Log.With("do", "before").With("sentStates", t.getSentStateList()).Debug("got network message")
		t.sender.processAcknowledgmentThrough(inst.AckNum)
		util.Log.With("do", "after-").With("sentStates", t.getSentStateList()).Debug("got network message")

		// inform network layer of roundtrip (end-to-end-to-end) connectivity
		t.connection.setLastRoundtripSuccess(t.sender.getSentStateAckedTimestamp())

		// first, make sure we don't already have the new state
		for i := range t.receivedState {
			if inst.NewNum == t.receivedState[i].num {
				util.Log.With("quit", "duplicate state").With("NewNum", inst.NewNum).Debug("got network message")
				return nil
			}
		}

		// now, make sure we do have the old state
		found := false
		var refState *TimestampedState[R]
		for i := range t.receivedState {
			if inst.OldNum == t.receivedState[i].num {
				found = true
				refState = &t.receivedState[i]
				break
			}
		}

		if !found {
			return fmt.Errorf("Ignoring out-of-order packet. Reference state %d has been "+
				"discarded or hasn't yet been received.\n", inst.OldNum)
			// this is security-sensitive and part of how we enforce idempotency
		}

		// throw away state whoes num < throwawayNum
		t.processThrowawayUntil(inst.ThrowawayNum)

		// Do not accept state if our queue is full.
		//
		// This is better than dropping states from the middle of the
		// queue (as sender does), because we don't want to ACK a state
		// and then discard it later.
		if len(t.receivedState) > 1024 { // limit on state queue
			now := time.Now().UnixMilli()
			if now < t.receiverQuenchTimer { // deny letting state grow further
				if t.verbose > 0 {
					// fmt.Fprintf(os.Stderr, "#recv [%d] Receiver queue full, discarding %d (malicious sender or "+
					// 	"long-unidirectional connectivity?)\n", now%100000, inst.NewNum)
					util.Log.With("time", now%100000).With("newNum", inst.NewNum).
						Debug("#recv Receiver queue full, discarding " +
							" (malicious sender or long-unidirectional connectivity?)")
				}
				return nil
			} else {
				t.receiverQuenchTimer = now + 15000
			}
		}

		// apply diff to reference state
		// we clone the state to avoid pollute reference state
		newState := refState.clone()
		newState.timestamp = time.Now().UnixMilli()
		newState.num = inst.NewNum
		if len(inst.Diff) > 0 {
			newState.state.ApplyString(string(inst.Diff))
		}

		// Insert new state in sorted place
		// rs := make([]TimestampedState[R], 0)
		rs := t.receivedState[:0]
		for i := range t.receivedState {
			// if /* newState.num != -1 &&  */ t.receivedState[i].num > newState.num {
			if t.receivedState[i].num > newState.num {
				// insert out-of-order new state
				rs = append(rs, newState)
				rs = append(rs, t.receivedState[i:]...)
				t.receivedState = rs

				// if newState.num == math.MaxUint64 {
				// 	//shutdown state is always out of order
				// 	// util.Log.With("ackNum", inst.AckNum).
				// 	// 	With("newNum", inst.NewNum).
				// 	// 	With("OldNum", inst.OldNum).
				// 	// 	With("throwawayNum", inst.ThrowawayNum).
				// 	// 	Debug("get shutdown state")
				// 	t.sender.setAckNum(newState.num)
				// 	t.sender.remoteHeard(newState.timestamp)
				// 	if len(inst.Diff) > 0 {
				// 		t.sender.setDataAck()
				// 	}
				// 	util.Log.With("receivedState", t.getReceivedStateList()).
				// 		With("sort", "insert shutdown state").
				// 		Debug("got network message")
				// }
				if t.verbose > 0 {
					util.Log.With("time", time.Now().UnixMilli()%100000).
						With("newNum", newState.num).
						With("ackNum", inst.AckNum).
						Debug("#recv Received OUT-OF-ORDER state x [ack y]")
				}
				return nil
			}
			rs = append(rs, t.receivedState[i])
		}

		if t.verbose > 0 { // TODO remove this duplicated log statements
			util.Log.With("time", time.Now().UnixMilli()%100000).
				With("newNum", newState.num).
				With("OldNum", inst.OldNum).
				With("AckNum", inst.AckNum).
				Debug("#recv Received state x [coming from y, ack state z]")
		}

		// fmt.Printf("#recv receive state num %d from %q got diff=%q.\n", newState.num, t.connection.remoteAddr, inst.Diff)

		t.receivedState = append(t.receivedState, newState) // insert new state
		t.sender.setAckNum(t.receivedState[len(t.receivedState)-1].num)
		util.Log.With("receivedState", t.getReceivedStateList()).Debug("got network message")

		t.sender.remoteHeard(newState.timestamp)
		if len(inst.Diff) > 0 {
			t.sender.setDataAck()
		}
	}
	return nil
}

func (t *Transport[S, R]) getReceivedStateList() string {
	var s strings.Builder
	s.WriteString("(")
	for i := range t.receivedState {
		fmt.Fprintf(&s, "%d,", t.receivedState[i].num)
	}
	s.WriteString(")")
	return s.String()
}

func (t *Transport[S, R]) getSentStateList() string {
	var s strings.Builder
	s.WriteString("[")
	for i := range t.sender.sentStates {
		fmt.Fprintf(&s, "%d,", t.sender.sentStates[i].num)
	}
	s.WriteString("]")
	return s.String()
}
