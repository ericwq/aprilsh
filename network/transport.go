// Copyright 2022~2024 wangqi. All rights reserved.
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

const (
	_ACTIVE_GAP = 5100
)

// type S or R that must implement the State[T] interface - that is, for itself.
type Transport[S State[S], R State[R]] struct {
	lastReceiverState R // the state we were in when user last queried state

	// the underlying, encrypted network connection
	connection *Connection

	// sender side
	sender *TransportSender[S]

	fragments *FragmentAssembly
	port      string // server port

	// simple receiver
	receivedState       []TimestampedState[R]
	receiverQuenchTimer int64
	verbose             uint
}

func NewTransportServer[S State[S], R State[R]](initialState S, initialRemote R,
	desiredIp, desiredPort string,
) *Transport[S, R] {
	ts := &Transport[S, R]{}
	ts.connection = NewConnection(desiredIp, desiredPort)
	ts.sender = NewTransportSender(ts.connection, initialState)
	ts.port = desiredPort

	ts.receivedState = make([]TimestampedState[R], 0)
	ts.receivedState = append(ts.receivedState,
		TimestampedState[R]{initialRemote.Clone(), time.Now().UnixMilli(), 0})

	ts.lastReceiverState = ts.receivedState[0].state
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
		TimestampedState[R]{initialRemote.Clone(), time.Now().UnixMilli(), 0})

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
			// 	util.Log.Debug("remove num","num", t.receivedState[i].num)
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
	s, _, err := t.connection.Recv(1)
	if err != nil {
		return err
	}

	// t.remoteAddr = rAddr
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

// func (t *Transport[S, R]) SetReadDeadline(ti time.Time) error {
// 	return t.connection.sock().SetReadDeadline(ti)
// }

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

func (t *Transport[S, R]) GetSentStateLastTimestamp() int64 {
	return t.sender.getSentStateLastTimestamp()
}

func (t *Transport[S, R]) SentInterval() uint {
	return t.sender.sendInterval()
}

func (t *Transport[S, R]) GetRemoteAddr() net.Addr {
	return t.connection.getRemoteAddr()
}

func (t *Transport[S, R]) GetKey() string {
	return t.connection.getKey()
}

func (t *Transport[S, R]) Close() {
	t.connection.Close()
}

func (t *Transport[S, R]) GetConnection() *Connection {
	return t.connection
}

func (t *Transport[S, R]) GetServerPort() string {
	return t.port
}

func (t *Transport[S, R]) ProcessPayload(s string) error {
	frag := NewFragmentFrom(s)

	if t.fragments.addFragment(frag) { // complete packet
		inst := t.fragments.getAssembly()

		// if inst.NewNum == -1 {
		// 	util.Log.Debug("got shutdown request")
		// 	for i := range t.sender.sentStates {
		// 		util.Log.Debug("sentStates","i", i,"num", t.sender.sentStates[i].num)
		// 	}
		// 	for i := range t.receivedState {
		// 		util.Log.Debug("receivedState","i", i,"num", t.receivedState[i].num)
		// 	}
		// }

		if inst.ProtocolVersion != APRILSH_PROTOCOL_VERSION {
			return errors.New("aprilsh protocol version mismatch")
		}

		util.Logger.Trace("got message",
			"NewNum", inst.NewNum,
			"OldNum", inst.OldNum,
			"AckNum", inst.AckNum,
			"throwawayNum", inst.ThrowawayNum,
			"port", t.port)

		// remove send state for which num < AckNum
		// util.Log.Debug("got message","do", "before","sentStates", t.sender.getSentStateList())
		t.sender.processAcknowledgmentThrough(inst.AckNum)
		// util.Log.Debug("got message","do", "after-","sentStates", t.sender.getSentStateList())

		// inform network layer of roundtrip (end-to-end-to-end) connectivity
		t.connection.setLastRoundtripSuccess(t.sender.getSentStateAckedTimestamp())

		// first, make sure we don't already have the new state
		for i := range t.receivedState {
			if inst.NewNum == t.receivedState[i].num {
				util.Logger.Warn("got message", "quit", "duplicate state", "NewNum", inst.NewNum)
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
			return fmt.Errorf("ignoring out-of-order packet, reference state %d has been "+
				"discarded or hasn't yet been received", inst.OldNum)
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
					msg := "#recv Receiver queue full, discarding (malicious sender or long-unidirectional connectivity?)"
					util.Logger.Warn(msg, "time", now%100000, "newNum", inst.NewNum)
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
			util.Logger.Trace("got message", "applyString", "start")
			newState.state.ApplyString(string(inst.Diff))
			util.Logger.Trace("got message", "applyString", "end")
		}

		// Insert new state in sorted place
		rs := t.receivedState[:0]
		for i := range t.receivedState {
			// if /* newState.num != -1 &&  */ t.receivedState[i].num > newState.num {
			if t.receivedState[i].num > newState.num {
				// insert out-of-order new state
				rs = append(rs, newState)
				rs = append(rs, t.receivedState[i:]...)
				t.receivedState = rs

				if t.verbose > 0 {
					util.Logger.Warn("#recv Received OUT-OF-ORDER state x [ack y]",
						"time", time.Now().UnixMilli()%100000,
						"newNum", newState.num,
						"ackNum", inst.AckNum)
				}
				return nil
			}
			rs = append(rs, t.receivedState[i])
		}

		t.receivedState = append(t.receivedState, newState) // insert new state
		t.sender.setAckNum(t.receivedState[len(t.receivedState)-1].num)

		t.sender.remoteHeard(newState.timestamp)
		if len(inst.Diff) > 0 {
			t.sender.setDataAck()
		}

		util.Logger.Trace("got message",
			"receivedState", t.getReceivedStateList(),
			"AckNum", t.receivedState[len(t.receivedState)-1].num,
			"pendingDataAck", t.sender.pendingDataAck,
			"diffLength", len(inst.Diff))

		// util.Log.Debug("got message",
		// 	"nextAckTime", t.sender.nextAckTime,
		// 	"nextSendTime", t.sender.nextSendTime,
		// 	"time", newState.GetTimestamp()%10000)
	} else {
		util.Logger.Debug("addFragment return false")
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

func (t *Transport[S, R]) InitSize(nCols, nRows int) {
	s := t.sender.sentStates[0].GetState()
	s.InitSize(nCols, nRows)
}

// detect computer awaken from hibernate based on receivedState and sentStates.
func (t *Transport[S, R]) Awaken(now int64) (ret bool) {
	_, recvStatus := awaken(t.receivedState, now)
	_, sendStatus := t.sender.Awaken(now)

	if sendStatus == _KEEP_ALIVE {
		ret = false
	} else if (sendStatus == _ONE_AWAKEN || sendStatus == _JUST_AWAKEN) &&
		(recvStatus == _ONE_AWAKEN || recvStatus == _JUST_AWAKEN) {
		ret = true
	} else if sendStatus == _LACK_STATE || recvStatus == _LACK_STATE {
		ret = false
	} else {
		ret = false
	}

	/*
	              | keep live | just awaken | one awaken | lack state | no response | send status
	   keep live  | false     | x (false)   | (x) false  | false      | x           |
	   just awaken| false     | true        | true       | false      | false       |
	   one awaken | false     | true        | true       | false      | x			  |
	   lack state | false     | false       | false      | false      | x		     |
	  no response | x         | x           | x          | x          | x           |
	   recv status
	*/

	defer func() {
		util.Logger.Debug("Awaken",
			"recvStatus", recvStatus,
			"sendStatus", sendStatus,
			"ret", ret,
			"now", now,
			"port", t.GetServerPort())
		back := len(t.receivedState)
		if back >= 2 {
			util.Logger.Debug("Awaken",
				"recvPrev", t.receivedState[back-2].GetTimestamp(),
				"recvLast", t.receivedState[back-1].GetTimestamp())
		}
		back = len(t.sender.sentStates)
		if back >= 2 {
			util.Logger.Debug("Awaken",
				"sendPrev", t.sender.sentStates[back-2].GetTimestamp(),
				"sendLast", t.sender.sentStates[back-1].GetTimestamp())
		}
	}()

	return
}

const (
	_KEEP_ALIVE  = 0 // keep send, no awaken
	_JUST_AWAKEN = 1 // just awaken, not finish send/recv
	_ONE_AWAKEN  = 2 // awaken, finish one send/recv
	_LACK_STATE  = 3 // only one state available
	_NO_RESPONSE = 4 // no response for a long time
)

// return wake up status and awaken result
// if the last state is resent, check the previous state, if the previous state
// is not recent, found awaken.
// if the last state is not resent, found awaken.
// otherwise no hibernate
func awaken[R State[R]](states []TimestampedState[R], now int64) (ret bool, ak int) {
	i := len(states) - 1
	// is last state recent?
	if now-states[i].GetTimestamp() < _ACTIVE_GAP {
		if len(states) >= 2 {
			// check the previous state (before the last)
			i = len(states) - 2
			if now-states[i].GetTimestamp() > _ACTIVE_GAP*2 {
				ak = _ONE_AWAKEN
			} else {
				// the previous state is recent
				ak = _KEEP_ALIVE
			}
		} else {
			ak = _LACK_STATE
		}
	} else {
		ak = _JUST_AWAKEN
		if len(states) >= 14 {
			ak = _NO_RESPONSE
		}
	}

	switch ak {
	case _KEEP_ALIVE, _NO_RESPONSE:
		ret = false
	case _JUST_AWAKEN:
		ret = true
	case _ONE_AWAKEN:
		ret = true
	case _LACK_STATE:
		ret = false
	}
	return
}
