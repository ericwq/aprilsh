// Copyright 2022~2023 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package network

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/ericwq/aprilsh/encrypt"
	pb "github.com/ericwq/aprilsh/protobufs"
	"github.com/ericwq/aprilsh/terminal"
	"github.com/ericwq/aprilsh/util"
	"golang.org/x/exp/slog"
	// "github.com/ericwq/aprilsh/util"
)

const (
	SEND_INTERVAL_MIN    = 20    /* ms between frames */
	SEND_INTERVAL_MAX    = 250   /* ms between frames */
	ACK_INTERVAL         = 3000  /* ms between empty acks */
	ACK_DELAY            = 100   /* ms before delayed ack */
	SHUTDOWN_RETRIES     = 16    /* number of shutdown packets to send before giving up */
	ACTIVE_RETRY_TIMEOUT = 10000 /* attempt to resend at frame rate */
)

type TransportSender[T State[T]] struct {
	// state of sender
	connection   *Connection
	currentState T
	// first element: known, acknowledged receiver sentStates
	// last element: last sent state
	sentStates []TimestampedState[T]

	// somewhere in the middle: the assumed state of the receiver
	assumedReceiverState *TimestampedState[T]
	// helper function for testing, it's nil by default
	hookForTick func()

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
	ackNum         uint64
	pendingDataAck bool
	SEND_MINDELAY  uint  // ms to collect all input
	lastHeard      int64 // last time received new state

	mindelayClock int64 // time of first pending change to current state
}

func NewTransportSender[T State[T]](connection *Connection, initialState T) *TransportSender[T] {
	ts := &TransportSender[T]{}
	ts.connection = connection
	ts.currentState = initialState
	ts.sentStates = make([]TimestampedState[T], 0)

	now := time.Now().UnixMilli()
	ts.addSentState(now, 0, initialState)
	ts.assumedReceiverState = &ts.sentStates[0]

	ts.fragmenter = NewFragmenter()

	ts.nextAckTime = now
	ts.nextSendTime = now

	ts.shutdownStart = math.MaxInt64
	ts.SEND_MINDELAY = 8
	ts.mindelayClock = math.MaxInt64
	return ts
}

// update assumedReceiverState according to connection timeout and ack delay.
func (ts *TransportSender[T]) updateAssumedReceiverState() {
	now := time.Now().UnixMilli()

	// start from what is known and give benefit of the doubt to unacknowledged states
	// transmitted recently enough ago
	ts.assumedReceiverState = &ts.sentStates[0]

	timeout := ts.connection.timeout()
	for i := 1; i < len(ts.sentStates); i++ {
		// fmt.Printf("#updateAssumedReceiverState now-ts.sentStates[%2d].timestamp=%4d, ts.connection.timeout()+ACK_DELAY=%d ",
		// 	i, now-ts.sentStates[i].timestamp, ts.connection.timeout()+ACK_DELAY)
		if now-ts.sentStates[i].timestamp < timeout+ACK_DELAY {
			ts.assumedReceiverState = &ts.sentStates[i]
			// fmt.Printf("assumedReceiverState=%2d \n", i)
		} else {
			// fmt.Printf("assumedReceiverState=%2d return\n", i)
			return
		}
	}
}

// Investigate diff against known receiver state instead
// return mutated propsedDiff
func (ts *TransportSender[T]) attemptProspectiveResendOptimization(propsedDiff string) string {
	if ts.assumedReceiverState == &ts.sentStates[0] {
		return propsedDiff
	}

	resendDiff := ts.currentState.DiffFrom(ts.sentStates[0].state)

	// We do a prophylactic resend if it would make the diff shorter,
	// or if it would lengthen it by no more than 100 bytes and still be
	// less than 1000 bytes.
	rLen := len(resendDiff)
	pLen := len(propsedDiff)
	if rLen <= pLen || (rLen < 1000 && rLen-pLen < 100) {
		ts.assumedReceiverState = &ts.sentStates[0]
		propsedDiff = resendDiff
	}

	return propsedDiff
}

// clear currentState and sentStates by remove the oldest sent state from them.
// only works for UserStream state.
func (ts *TransportSender[T]) rationalizeStates() {
	knownReceiverState := ts.sentStates[0].state

	ts.currentState.Subtract(knownReceiverState)

	for i := len(ts.sentStates) - 1; i >= 0; i-- {
		ts.sentStates[i].state.Subtract(knownReceiverState)
		// fmt.Printf("#rationalizeStates after Subtract() sentStates %d =%v\n", i, ts.sentStates[i].state)
	}
}

func (ts *TransportSender[T]) sendToReceiver(diff string) error {
	var newNum uint64
	back := len(ts.sentStates) - 1
	if ts.currentState.Equal(ts.sentStates[back].state) { // previously sent
		newNum = ts.sentStates[back].num
	} else { // new state
		newNum = ts.sentStates[back].num + 1
	}

	// special case for shutdown sequence
	if ts.shutdownInProgress {
		newNum = math.MaxUint64
	}

	// now := time.Now().UnixMilli()
	if newNum == ts.sentStates[back].num {
		ts.sentStates[back].timestamp = time.Now().UnixMilli()
	} else {
		ts.addSentState(time.Now().UnixMilli(), newNum, ts.currentState)
	}

	if err := ts.sendInFragments(diff, newNum); err != nil {
		return err
	}

	/* successfully sent, probably */
	/* ("probably" because the FIRST size-exceeded datagram doesn't get an error) */
	ts.assumedReceiverState = &ts.sentStates[len(ts.sentStates)-1]
	ts.nextAckTime = time.Now().UnixMilli() + ACK_INTERVAL
	ts.nextSendTime = math.MaxInt64

	return nil
}

func (ts *TransportSender[T]) sendEmptyAck() error {
	now := time.Now().UnixMilli()
	back := len(ts.sentStates) - 1
	newNum := ts.sentStates[back].num + 1

	// special case for shutdown sequence
	if ts.shutdownInProgress {
		newNum = math.MaxUint64
	}

	ts.addSentState(now, newNum, ts.currentState)
	if err := ts.sendInFragments("", newNum); err != nil {
		return err
	}

	ts.nextAckTime = now + ACK_INTERVAL
	ts.nextSendTime = math.MaxInt64

	return nil
}

func (ts *TransportSender[T]) sendInFragments(diff string, newNum uint64) error {
	inst := pb.Instruction{}
	inst.ProtocolVersion = APRILSH_PROTOCOL_VERSION
	inst.OldNum = ts.assumedReceiverState.num
	inst.NewNum = newNum
	inst.AckNum = ts.ackNum
	inst.ThrowawayNum = ts.sentStates[0].num
	inst.Diff = []byte(diff)
	inst.Chaff = []byte(ts.makeChaff())

	var err error
	err = ts.sendFragments(&inst, newNum)

	if ts.verbose > 0 {
		var s strings.Builder
		s.WriteString("[")
		for i := range ts.sentStates {
			fmt.Fprintf(&s, "%d,", ts.sentStates[i].num)
		}
		s.WriteString("]")

		util.Log.With("sentStates", s.String()).
			With("diffLength", len(diff)).
			With("SRTT", ts.connection.getSRTT()).
			Debug("send message")
		// return ts.sendFragments(&inst, newNum)
	}
	return err
}

func (ts *TransportSender[T]) sendFragments(inst *pb.Instruction, newNum uint64) error {
	if newNum == math.MaxUint64 {
		ts.shutdownTries++
	}

	// TODO we don't use OCB, so remove the encrypt.ADDED_BYTES ?
	fragments := ts.fragmenter.makeFragments(inst, ts.connection.getMTU()-ADDED_BYTES-encrypt.ADDED_BYTES)
	for i := range fragments {
		if err := ts.connection.send(fragments[i].String()); err != nil {
			return err
		}

		if ts.verbose > 0 {
			util.Log.With("NewNum", inst.NewNum).
				With("OldNum", inst.OldNum).
				With("AckNum", inst.AckNum).
				With("ThrowawayNum", inst.ThrowawayNum).
				// With("length", len(fragments[i].contents)).
				Debug("send message")
			// util.Log.With("time", (time.Now().UnixMilli()%100000)).
			// 	With("fragmentsID", fragments[i].id).
			// 	With("fragmentNum", fragments[i].fragmentNum).
			// 	With("frameRate", 1000.0/float64(ts.sendInterval())).
			// 	With("timeout", ts.connection.timeout()).
			// 	Debug("send message")
			// util.Log.With("mindelayClock", ts.mindelayClock).
			// 	With("SEND_MINDELAY", ts.SEND_MINDELAY).
			// 	With("sendInterval", ts.sendInterval()).
			// 	With("timeout", ts.connection.timeout()).
			// 	Debug("send message")
		}
	}

	ts.pendingDataAck = false
	return nil
}

// add state into the send states list.
func (ts *TransportSender[T]) addSentState(timestamp int64, num uint64, state T) {
	s := TimestampedState[T]{timestamp, num, state.Clone()}
	ts.sentStates = append(ts.sentStates, s)

	// fmt.Printf("#addSentState No.%2d state in sendStates, %T\n", num, ts.sentStates[len(ts.sentStates)-1].state)

	if len(ts.sentStates) > 32 { // limit on state queue
		ts.sentStates = ts.sentStates[16:] // erase state from middle of queue
	}
}

// Housekeeping routine to calculate next send and ack times
// update assumed receiver state, cut out common prefix of all states
func (ts *TransportSender[T]) calculateTimers() {
	now := time.Now().UnixMilli()

	// Update assumed receiver state
	ts.updateAssumedReceiverState()

	// Cut out common prefix of all states
	ts.rationalizeStates()

	back := len(ts.sentStates) - 1

	if ts.pendingDataAck && ts.nextAckTime > now+ACK_DELAY {
		ts.nextAckTime = now + ACK_DELAY // got data from remote, send ack message later
		util.Log.With("status", "pendingDataAck").
			With("nextAckTime", ts.nextAckTime).
			Debug("calculateTimers")
	}

	if !ts.currentState.Equal(ts.sentStates[back].state) {
		// currentState is not the last sent states
		if ts.mindelayClock == math.MaxInt64 {
			ts.mindelayClock = now
		}

		// current state changed and not sent, send current state later
		ts.nextSendTime = terminal.Min(ts.mindelayClock+int64(ts.SEND_MINDELAY),
			ts.sentStates[back].timestamp+int64(ts.sendInterval()))
		// we change from Max to Min to avoid duplicat message
		// util.Log.With("status", "currentState!=lastSendStates").
		// 	With("nextSendTime", ts.nextSendTime).
		// 	Debug("calculateTimers")
	} else if !ts.currentState.Equal(ts.assumedReceiverState.state) && ts.lastHeard+ACTIVE_RETRY_TIMEOUT > now {
		// currentState is last sent state but not the assumed receiver state
		ts.nextSendTime = ts.sentStates[back].timestamp + int64(ts.sendInterval())
		if ts.mindelayClock != math.MaxInt64 {
			ts.nextSendTime = terminal.Max(ts.nextSendTime, ts.mindelayClock+int64(ts.SEND_MINDELAY))
		}
		// util.Log.With("status", "currentState==lastSendStates!=AssumedState").
		// 	With("nextSendTime", ts.nextSendTime).
		// 	With("lastHeard", ts.lastHeard).
		// 	Debug("calculateTimers")
	} else if !ts.currentState.Equal(ts.sentStates[0].state) && ts.lastHeard+ACTIVE_RETRY_TIMEOUT > now {
		// currentState is the last and assumed receiver state but not the oldest sent state
		ts.nextSendTime = ts.sentStates[back].timestamp + ts.connection.timeout() + ACK_DELAY
		// util.Log.With("status", "currentState==lastSendStates==AssumedState").
		// 	With("nextSendTime", ts.nextSendTime).
		// 	With("lastHeard", ts.lastHeard).
		// 	Debug("calculateTimers")
	} else {
		ts.nextSendTime = math.MaxInt64 // nothing need to be sent actively
		// util.Log.With("status", "others").
		// 	With("nextSendTime", ts.nextSendTime).
		// 	Debug("calculateTimers")
	}

	// speed up shutdown sequence
	if ts.shutdownInProgress || ts.ackNum == math.MaxUint64 {
		ts.nextAckTime = ts.sentStates[back].timestamp + int64(ts.sendInterval())
		// util.Log.With("status", "shutdown").
		// 	With("nextAckTime", ts.nextAckTime).
		// 	With("ackNum", ts.ackNum).
		// 	With("shutdown", ts.shutdownInProgress).
		// 	Debug("calculateTimers")
	}
	// util.Log.With("nextAckTime", ts.nextAckTime).
	// 	With("nextSendTime", ts.nextSendTime).
	// 	Debug("calculateTimers")
}

// make chaff with random length and random contents.
func (ts *TransportSender[T]) makeChaff() string {
	CHAFF_MAX := 16
	chaffLen := (int)(encrypt.PrngUint8()) % (CHAFF_MAX + 1)

	chaff := encrypt.PrngFill(chaffLen)
	return string(chaff)
}

// Send data or an ack if necessary
// before tick, currentState is updated to the latest.
func (ts *TransportSender[T]) tick() error {
	ts.calculateTimers() // updates assumed receiver state and rationalizes

	// util.Log.With("point", 100).Debug("tick")

	// skip send if there is no peer
	if !ts.connection.getHasRemoteAddr() {
		// util.Log.Debug("tick skip tick: no remote addr")
		return nil
	}

	// skip send if the interval is too short
	now := time.Now().UnixMilli()
	if now < ts.nextAckTime && now < ts.nextSendTime {
		// util.Log.With("now", now).With("now<nextAckTime", now < ts.nextAckTime).
		// 	With("now<nextSendTime", now < ts.nextSendTime).Debug("tick skip tick: time")
		return nil
	}

	// util.Log.With("point", 200).With("assumedReceiverState", ts.getAssumedReceiverStateIdx()).Debug("tick")
	// Determine if a new diff or empty ack needs to be sent
	diff := ts.currentState.DiffFrom(ts.assumedReceiverState.state)
	// util.Log.With("point", 210).Debug("tick")
	util.Log.SetLevel(slog.LevelInfo)
	diff = ts.attemptProspectiveResendOptimization(diff)
	// util.Log.With("diff", diff).With("length", len(diff)).Debug("send message")

	if ts.verbose > 0 {
		if ts.hookForTick != nil { // hook function for testing
			ts.hookForTick()
		}
		// verify diff has round-trip identity (modulo Unicode fallback rendering)
		newState := ts.assumedReceiverState.state.Clone()
		// util.Log.With("point", 410).Debug("tick")
		newState.ApplyString(diff)
		// util.Log.With("point", 420).Debug("tick")
		if !ts.currentState.Equal(newState) {
			ts.currentState.EqualTrace(newState) // TODO remove this if integration test is finished
			util.Log.Warn("#tick Warning, round-trip Instruction verification failed!")
		}

		// Also verify that both the original frame and generated frame have the same initial diff.
		// util.Log.With("point", 500).Debug("tick")
		currentDiff := ts.currentState.InitDiff()
		// util.Log.With("point", 600).Debug("tick")
		newDiff := newState.InitDiff()
		if currentDiff != newDiff {
			util.Log.Warn("#tick Warning, target state Instruction verification failed!")
		}
		// util.Log.With("point", 700).Debug("tick")
	}
	util.Log.SetLevel(slog.LevelDebug)

	// fmt.Printf("#tick send %q to receiver %s.\n", diff, ts.connection.getRemoteAddr())

	util.Log.With("nextAckTime", ts.nextAckTime).
		With("nextSendTime", ts.nextSendTime).
		With("assumedReceiverState", ts.assumedReceiverState.num).
		With("now", now).
		Debug("send message")
	if len(diff) == 0 {
		if now >= ts.nextAckTime {
			if err := ts.sendEmptyAck(); err != nil {
				return err
			}
			ts.mindelayClock = math.MaxInt64
		}

		if now >= ts.nextSendTime {
			ts.nextSendTime = math.MaxInt64
			ts.mindelayClock = math.MaxInt64
		}
	} else if now >= ts.nextSendTime || now >= ts.nextAckTime {
		util.Log.With(">nextSendTime", now >= ts.nextSendTime).
			With(">nextAckTime", now >= ts.nextAckTime).
			Debug("send message")
		// send diff or ack
		if err := ts.sendToReceiver(diff); err != nil {
			return err
		}
		ts.mindelayClock = math.MaxInt64
	}
	util.Log.With("nextAckTime", ts.nextAckTime).
		With("nextSendTime", ts.nextSendTime).
		With("now", time.Now().UnixMilli()).
		Debug("send message")

	return nil
}

// Returns the number of ms to wait until next possible event.
func (ts *TransportSender[T]) waitTime() int {
	ts.calculateTimers()

	// if nextSendTime < nextAckTime, use the nextSendTime
	nextWakeup := ts.nextAckTime
	if ts.nextSendTime < nextWakeup {
		nextWakeup = ts.nextSendTime
	}

	now := time.Now().UnixMilli()

	// util.Log.With("nextSendTime", ts.nextSendTime).
	// 	With("nextAckTime", ts.nextAckTime).
	// 	With("nextWakeup", nextWakeup).
	// 	With("now", now).
	// 	Debug("sender waitTime")
	if !ts.connection.getHasRemoteAddr() {
		return math.MaxInt
	}

	if nextWakeup > now {
		return int(nextWakeup - now)
	} else {
		return 0
	}
}

// Executed upon receipt of ack
// remove sent states whose num less than ack num.
func (ts *TransportSender[T]) processAcknowledgmentThrough(ackNum uint64) {
	// Ignore ack if we have culled the state it's acknowledging

	for i := range ts.sentStates {
		// find the first element for which its num == ackNum
		if ts.sentStates[i].numEq(ackNum) {
			// remove the element for which its num < ackNum
			ss := ts.sentStates[:0]
			for j := range ts.sentStates {
				if ts.sentStates[j].numLt(ackNum) {
					// skip this means remove this element
				} else {
					ss = append(ss, ts.sentStates[j])
				}
			}
			ts.sentStates = ss
			break
		}
	}
}

// Executed upon entry to new receiver state
func (ts *TransportSender[T]) setAckNum(ackNum uint64) {
	ts.ackNum = ackNum
}

// Accelerate reply ack
func (ts *TransportSender[T]) setDataAck() {
	ts.pendingDataAck = true
}

// Received something
func (ts *TransportSender[T]) remoteHeard(x int64) {
	ts.lastHeard = x
}

// Starts shutdown sequence
func (ts *TransportSender[T]) startShutdown() {
	if !ts.shutdownInProgress {
		ts.shutdownStart = time.Now().UnixMilli()
		ts.shutdownInProgress = true
	}
}

// Cannot modify current_state while shutdown in progress
func (ts *TransportSender[T]) getCurrentState() T {
	return ts.currentState
}

func (ts *TransportSender[T]) setCurrentState(x T) {
	ts.currentState = x
	ts.currentState.ResetInput()
}

func (ts *TransportSender[T]) setVerbose(verbose uint) {
	ts.verbose = verbose
}

func (ts *TransportSender[T]) getShutdownInProgress() bool {
	return ts.shutdownInProgress
}

func (ts *TransportSender[T]) getShutdownAcknowledged() bool {
	return ts.sentStates[0].num == math.MaxUint64
}

func (ts *TransportSender[T]) getCounterpartyShutdownAcknowledged() bool {
	return ts.fragmenter.lastAckSent() == math.MaxUint64
}

// get the first sent state timestamp
func (ts *TransportSender[T]) getSentStateAckedTimestamp() int64 {
	return ts.sentStates[0].timestamp
}

// get the first sent state num
func (ts *TransportSender[T]) getSentStateAcked() uint64 {
	return ts.sentStates[0].num
}

// get the last sent state num
func (ts *TransportSender[T]) getSentStateLast() uint64 {
	return ts.sentStates[len(ts.sentStates)-1].num
}

func (ts *TransportSender[T]) shutdonwAckTimedout() bool {
	if ts.shutdownInProgress {
		if ts.shutdownTries > SHUTDOWN_RETRIES {
			return true
		} else if time.Now().UnixMilli()-ts.shutdownStart >= ACTIVE_RETRY_TIMEOUT {
			return true
		}
	}
	return false
}

func (ts *TransportSender[T]) setSendDelay(delay uint) {
	ts.SEND_MINDELAY = delay
}

// Try to send roughly two frames per RTT, bounded by limits on frame rate
func (ts *TransportSender[T]) sendInterval() uint {
	// int SEND_INTERVAL = lrint(ceil(connection->get_SRTT() / 2.0))
	SEND_INTERVAL := math.Round(math.Ceil(ts.connection.getSRTT() / 2.0))
	// fmt.Printf("#sendInterval SEND_INTERVAL=%f, SRTT=%f\n", SEND_INTERVAL, ts.connection.getSRTT())
	if SEND_INTERVAL < SEND_INTERVAL_MIN {
		SEND_INTERVAL = SEND_INTERVAL_MIN
	} else if SEND_INTERVAL > SEND_INTERVAL_MAX {
		SEND_INTERVAL = SEND_INTERVAL_MAX
	}
	return uint(SEND_INTERVAL)
}

func (ts *TransportSender[T]) getAssumedReceiverStateIdx() int {
	idx := 0
	for j := 0; j < len(ts.sentStates); j++ {
		if ts.assumedReceiverState == &ts.sentStates[j] {
			idx = j
		}
	}
	return idx
}
