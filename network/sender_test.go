/*

MIT License

Copyright (c) 2022~2023 wangqi

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
	"strings"
	"testing"
	"time"

	"github.com/ericwq/aprilsh/statesync"
)

func TestSenderMakeChaff(t *testing.T) {
	connection := NewConnection("localhost", "8080")
	initialState, _ := statesync.NewComplete(80, 40, 0)

	ts := NewTransportSender(connection, initialState)
	for i := 0; i < 10; i++ {
		chaff := ts.makeChaff()

		if len(chaff) <= 16 || len(chaff) >= 1 {
			// fmt.Printf("#test makeChaff() got %q, length=%d\n", chaff, len(chaff))
		} else {
			t.Errorf("#test makeChaff() exceed the size limit 1<=lenght<=16\n")
		}
	}

	connection.sock().Close()
}

func TestSenderUpdateAssumedReceiverState(t *testing.T) {
	tc := []struct {
		label            string
		fakeNetworkDelay int
		expect           int
	}{
		{"quick response", 2, 17},
		{"slow response", 65, 0},
	}

	for _, v := range tc {
		// prepare the environment
		connection := NewConnection("localhost", "8080")
		initialState, _ := statesync.NewComplete(80, 40, 0)

		ts := NewTransportSender(connection, initialState)

		// add enough state and mimic the delay between states
		for i := 0; i < 33; i++ { // addSentState require upper limit 32
			s, _ := statesync.NewComplete(80, 40, 0)
			now := time.Now().UnixMilli()
			time.Sleep(time.Millisecond * time.Duration(v.fakeNetworkDelay))
			ts.addSentState(now, int64(i+1), s)
		}

		ts.updateAssumedReceiverState()

		// validate the result
		idx := ts.getAssumedReceiverStateIdx()
		if idx != v.expect {
			t.Errorf("%q expect %d, got %d\n", v.label, v.expect, idx)
		}

		connection.sock().Close()
	}
}

func TestSenderProcessAcknowledgmentThrough(t *testing.T) {
	tc := []struct {
		label  string
		pause  int
		ackNum int64
		expect int
	}{
		{"remove first state", 50, 1, 5},
		{"keep last state", 52, 5, 1},
		{"keep all state", 51, 8, 6},
	}

	for _, v := range tc {
		connection := NewConnection("localhost", "8080")
		initialState, _ := statesync.NewComplete(80, 40, 0)

		ts := NewTransportSender(connection, initialState)
		s, _ := statesync.NewComplete(80, 40, 0)

		for i := 1; i < 6; i++ {
			now := time.Now().UnixMilli()
			time.Sleep(time.Millisecond * time.Duration(v.pause))
			ts.addSentState(now, int64(i), s)
			// fmt.Printf("%q No.%2d state in sendStates, point to %p\n", v.label, i, ts.sentStates[i].state)
		}

		ts.processAcknowledgmentThrough(v.ackNum)
		if len(ts.sentStates) != v.expect {
			t.Errorf("%q expect sentStates lengh %d, got %d\n", v.label, v.expect, len(ts.sentStates))
		}
		connection.sock().Close()
	}
}

func pushUserBytesTo(t *statesync.UserStream, raw string) {
	chs := []rune(raw)
	for i := range chs {
		t.PushBack([]rune{chs[i]})
		fmt.Printf("#pushUserBytesTo %q into state %p\n", chs[i], t)
	}
}

func TestSenderRationalizeStates(t *testing.T) {
	tc := []struct {
		label      string
		userBytes  []string
		prefix     string
		currentIdx int
		expect     []string
	}{
		{"remove first", []string{"abc", "abcde", "abcdef", "abcdefg"}, "ab", 1, []string{"", "c", "cde", "cdef", "cdefg"}},
	}

	for _, v := range tc {
		connection := NewConnection("localhost", "8080")
		initialState := &statesync.UserStream{} // first sent state
		pushUserBytesTo(initialState, v.prefix)

		ts := NewTransportSender(connection, initialState)
		// fmt.Printf("%q add state %s to 0\n", v.label, initialState)

		for i, keystroke := range v.userBytes {

			state := &statesync.UserStream{}

			pushUserBytesTo(state, keystroke)

			ts.addSentState(time.Now().UnixMilli(), int64(i+1), state)
			// fmt.Printf("%q add state %s to %2d\n", v.label, state, i+1)
		}

		ts.setCurrentState(ts.sentStates[v.currentIdx].state.Clone())
		// fmt.Printf("%q current state %d = %s\n", v.label, v.currentIdx, ts.currentState)

		ts.rationalizeStates()

		// validate the sent states
		for i := range ts.sentStates {
			// fmt.Printf("%q No.%2d state contains:%s\n", v.label, i, ts.sentStates[i].state)
			got := ts.sentStates[i].state.String()

			if !strings.Contains(got, fmt.Sprintf("\"%s\"", v.expect[i])) {
				t.Errorf("%q expect No.%d state %s, got %s\n", v.label, i, v.expect[i], got)
			}
		}

		// validate the result of current state
		currentStr := v.userBytes[v.currentIdx-1]
		expect := strings.Replace(currentStr, v.prefix, "", 1)
		if !strings.Contains(ts.getCurrentState().String(), fmt.Sprintf("\"%s\"", expect)) {
			t.Errorf("%q expct current state %q, got %q\n", v.label, expect, ts.currentState.String())
		}
		connection.sock().Close()
	}
}

func TestSenderAttemptProspectiveResendOptimization(t *testing.T) {
	tc := []struct {
		label        string
		initialState string
		states       []string
		currentIdx   int
		assumedIdx   int
		expect       string
	}{
		{"assumed receiver state is the first state", "ab", []string{"abc", "abcde", "abcdef", "abcdefg"}, 2, 0, "\n\a\x12\x05\"\x03cde"},
		{"resend length - diff length < 100", "ab", []string{"abc", "abcde", "abcdef", "abcdefg"}, 4, 1, "\n\t\x12\a\"\x05cdefg"},
	}

	for _, v := range tc {
		connection := NewConnection("localhost", "8080")
		initialState := &statesync.UserStream{} // initial state
		pushUserBytesTo(initialState, v.initialState)
		ts := NewTransportSender(connection, initialState)

		// prepare sentStates data
		for i, keystroke := range v.states {
			state := &statesync.UserStream{}
			pushUserBytesTo(state, keystroke)
			ts.addSentState(time.Now().UnixMilli(), int64(i+1), state)
		}

		// prepare currentState and assumedReceiverState
		ts.setCurrentState(ts.sentStates[v.currentIdx].state.Clone())
		ts.assumedReceiverState = &ts.sentStates[v.assumedIdx]

		diff := ts.currentState.DiffFrom(ts.assumedReceiverState.state)
		// fmt.Printf("#test attemptProspectiveResendOptimization() diff=%q\n", diff)

		got := ts.attemptProspectiveResendOptimization(diff)

		// validate the diff
		if got != v.expect {
			t.Errorf("%q expect %q, got %q\n", v.label, v.expect, got)
		}
		connection.sock().Close()
	}
}

func TestSenderCalculateTimers(t *testing.T) {
	tc := []struct {
		label              string
		initialState       string
		states             []string
		currentIdx         int
		fakeNetworkDelay   int
		lastHeard          int64
		mindelayClock      int64
		pendingDataAck     bool
		shutdown           bool
		expectNextSendTime int64
		expectNextAckTime  int64
	}{
		{
			"current !=newest", "abc",
			[]string{"abcd", "abcde", "abcdef", "abcdefg"},
			0, 2, 0, -1, false, false, 0, 0,
		},
		{
			"current = newest != assumed", "abc",
			[]string{"abcd", "abcde", "abcdef", "abcdefg"},
			4, 450, 1, 5, false, false, 0, 0,
		},
		{
			"current = newest = assumed != oldest", "abc",
			[]string{"abcd", "abcde", "abcdef", "abcdefg"},
			4, 10, 0, 5, false, false, 0, 0,
		},
		{
			"current = newest, lastHeard over due ", "abc",
			[]string{"abcd", "abcde", "abcdef", "abcdefg"},
			4, 10, -2 * ACTIVE_RETRY_TIMEOUT, 5, false, true, -1, 0,
		},
		{
			"current = newest, lastHeard over due ", "abc",
			[]string{"abcd", "abcde", "abcdef", "abcdefg"},
			4, 10, -2 * ACTIVE_RETRY_TIMEOUT, 5, true, false, -1, 0,
		},
	}

	for _, v := range tc {
		connection := NewConnection("localhost", "8080")
		initialState := &statesync.UserStream{} // initial state
		pushUserBytesTo(initialState, v.initialState)
		ts := NewTransportSender(connection, initialState)

		// prepare sentStates data
		for i, keystroke := range v.states {
			state := &statesync.UserStream{}
			pushUserBytesTo(state, keystroke)
			time.Sleep(time.Duration(v.fakeNetworkDelay) * time.Millisecond)
			// change assumedReceiverState through fakeNetworkDelay
			ts.addSentState(time.Now().UnixMilli(), int64(i+1), state)
		}

		// prepare currentState and assumedReceiverState
		ts.setCurrentState(ts.sentStates[v.currentIdx].state.Clone())
		ts.remoteHeard(time.Now().UnixMilli() + v.lastHeard)
		if v.mindelayClock != -1 {
			ts.mindelayClock = time.Now().UnixMilli()
		}

		if v.pendingDataAck {
			ts.setDataAck()
			ts.nextAckTime = time.Now().UnixMilli() + 2*ACK_DELAY
		}
		if v.shutdown { // corner case for shutdown
			ts.setAckNum(-1)
		}
		// ts.assumedReceiverState = &ts.sentStates[v.assumedIdx]

		ts.calculateTimers()

		// validate the result
		gotNextAckTime := ts.nextAckTime
		gotNextSendTime := ts.nextSendTime

		if gotNextAckTime != v.expectNextAckTime {
			if v.expectNextAckTime == -1 {
				t.Errorf("%q expect nextAckTime %d, got %d\n", v.label, v.expectNextAckTime, gotNextAckTime)
			}
		}

		if gotNextSendTime != v.expectNextSendTime {
			if v.expectNextSendTime == -1 {
				t.Errorf("%q expect nextSendTime %d, got %d\n", v.label, v.expectNextSendTime, gotNextSendTime)
			}
		}

		// fmt.Printf("#calculateTimers nextSendTime=%d, nextAckTime=%d\n", ts.nextSendTime, ts.nextAckTime)
		// ts.connection.hasRemoteAddr = true
		// waitTime := ts.waitTime()
		// fmt.Printf("#calculateTimers waitTime=%d\n", waitTime)

		connection.sock().Close()
	}
}

func TestSenderWaitTime(t *testing.T) {
	tc := []struct {
		label         string
		initialState  string
		hasRemoteAddr bool
		expect        int
	}{
		{"no remote address", "wait", false, math.MaxInt},
		{"has remote address", "wait", true, 0},
	}
	for _, v := range tc {
		connection := NewConnection("localhost", "8080")
		initialState := &statesync.UserStream{} // initial state
		pushUserBytesTo(initialState, v.initialState)
		ts := NewTransportSender(connection, initialState)

		ts.connection.hasRemoteAddr = v.hasRemoteAddr
		// fmt.Printf("%q nextAckTime=%d, nextSendTime=%d\n", v.label, ts.nextAckTime, ts.nextSendTime)
		got := ts.waitTime()
		// fmt.Printf("%q got =%d\n", v.label, got)

		if got != v.expect {
			t.Errorf("%q expect waitTime %d, got %d\n", v.label, v.expect, got)
		}

		connection.sock().Close()
	}
}
