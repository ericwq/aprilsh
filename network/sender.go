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
	"math"
	"time"

	"github.com/ericwq/aprilsh/encrypt"
	pb "github.com/ericwq/aprilsh/protobufs"
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

	now := time.Now().UnixMilli()
	ts.addSentState(now, 0, initialState)
	ts.assumedReceiverState = &ts.sentStates[0]

	ts.fragmenter = NewFragmenter()

	ts.nextAckTime = now
	ts.nextSendTime = now

	ts.shutdownStart = -1
	ts.SEND_MINDELAY = 8
	ts.mindelayClock = -1
	return ts
}

// update assumedReceiverState according to connection timeout and ack delay.
func (ts *TransportSender[T]) updateAssumedReceiverState() {
	now := time.Now().UnixMilli()

	// start from what is known and give benefit of the doubt to unacknowledged states
	// transmitted recently enough ago
	ts.assumedReceiverState = &ts.sentStates[0]

	for i := 1; i < len(ts.sentStates); i++ {
		// fmt.Printf("#updateAssumedReceiverState now-ts.sentStates[%2d].timestamp=%4d, ts.connection.timeout()+ACK_DELAY=%d ",
		// 	i, now-ts.sentStates[i].timestamp, ts.connection.timeout()+ACK_DELAY)
		if now-ts.sentStates[i].timestamp < ts.connection.timeout()+ACK_DELAY {
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

func (ts *TransportSender[T]) rationalizeStates() {
	knownReceiverState := ts.sentStates[0].state

	ts.currentState.Subtract(knownReceiverState)
	for i := len(ts.sentStates) - 1; i >= 0; i-- {
		ts.sentStates[i].state.Subtract(knownReceiverState)
	}
}

func (ts *TransportSender[T]) sendToReceiver(diff string) {
	var newNum int64
	back := len(ts.sentStates) - 1
	if ts.currentState.Equal(ts.sentStates[back].state) { // previously sent
		newNum = ts.sentStates[back].num
	} else { // new state
		newNum = ts.sentStates[back].num + 1
	}

	// special case for shutdown sequence
	if ts.shutdownInProgress {
		newNum = -1
	}

	now := time.Now().UnixMilli()
	if newNum == ts.sentStates[back].num {
		ts.sentStates[back].timestamp = now
	} else {
		ts.addSentState(now, newNum, ts.currentState)
	}

	ts.sendInFragments(diff, newNum) // Can throw NetworkException TODO make it happens

	/* successfully sent, probably */
	/* ("probably" because the FIRST size-exceeded datagram doesn't get an error) */
	ts.assumedReceiverState = &ts.sentStates[len(ts.sentStates)-1]
	ts.nextAckTime = now + ACK_INTERVAL
	ts.nextSendTime = -1
}

func (ts *TransportSender[T]) sendInFragments(diff string, newNum int64) {
	inst := pb.Instruction{}
	inst.ProtocolVersion = APRILSH_PROTOCOL_VERSION
	inst.OldNum = ts.assumedReceiverState.num
	inst.NewNum = newNum
	inst.OldNum = ts.ackNum
	inst.ThrowawayNum = ts.sentStates[0].num
	inst.Diff = []byte(diff)
	inst.Chaff = []byte(ts.makeChaff())

	if newNum == -1 {
		ts.shutdownTries++
	}

	// TODO we don't use OCB, so remove the encrypt.ADDED_BYTES ?
	fragments := ts.fragmenter.makeFragments(&inst, ts.connection.getMTU()-ADDED_BYTES-encrypt.ADDED_BYTES)
	for i := range fragments {
		ts.connection.send(fragments[i].String())

		if ts.verbose > 0 {
			fmt.Printf("[%d] Sent [%d=>%d] id %d, frag %d ack=%d, throwaway=%d, len=%d, frame rate=%.2f, timeout=%d, srtt=%.1f\n",
				(time.Now().UnixMilli() % 100000), inst.OldNum, inst.NewNum,
				fragments[i].id, fragments[i].fragmentNum, inst.AckNum,
				inst.ThrowawayNum, len(fragments[i].contents),
				1000.0/float64(ts.sendInterval()), ts.connection.timeout(), ts.connection.getSRTT())
		}
	}

	ts.pendingDataAct = false
}

// add state into the send states list.
func (ts *TransportSender[T]) addSentState(timestamp int64, num int64, state T) {
	s := TimestampedState[T]{timestamp, num, state}
	ts.sentStates = append(ts.sentStates, s)

	if len(ts.sentStates) > 32 { // limit on state queue
		ts.sentStates = ts.sentStates[16:] // erase state from middle of queue
	}
}

func (ts *TransportSender[T]) calculateTimers() {
	// now := time.Now().UnixMilli()
}

// make chaff with random length and random contents.
func (ts *TransportSender[T]) makeChaff() string {
	CHAFF_MAX := 16
	chaffLen := (int)(encrypt.PrngUint8()) % (CHAFF_MAX + 1)

	chaff := encrypt.PrngFill(chaffLen)
	return string(chaff)
}

// Send data or an ack if necessary
func (ts *TransportSender[T]) tick() {
}

func (ts *TransportSender[T]) getCurrentState() T {
	return ts.currentState
}

// TODO careful about the pointer
func (ts *TransportSender[T]) setCurrentState(x T) {
	ts.currentState = x
	// t.currentState.ResetInput()
}

// Try to send roughly two frames per RTT, bounded by limits on frame rate
func (ts *TransportSender[T]) sendInterval() int {
	// int SEND_INTERVAL = lrint(ceil(connection->get_SRTT() / 2.0))
	SEND_INTERVAL := math.Round(math.Ceil(ts.connection.getSRTT() / 2.0))
	if SEND_INTERVAL < SEND_INTERVAL_MIN {
		SEND_INTERVAL = SEND_INTERVAL_MIN
	} else if SEND_INTERVAL > SEND_INTERVAL_MAX {
		SEND_INTERVAL = SEND_INTERVAL_MAX
	}
	return int(SEND_INTERVAL)
}
