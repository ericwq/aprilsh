// Copyright 2022 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package network

import (
	"fmt"
	"io"
	"log/slog"
	"math"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ericwq/aprilsh/statesync"
	"github.com/ericwq/aprilsh/util"
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
		fakeNetworkDelay int64
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

		// add enough state
		now := time.Now().UnixMilli()
		for i := 0; i < 33; i++ { // addSentState require upper limit 32
			s, _ := statesync.NewComplete(80, 40, 0)
			ts.addSentState(now, uint64(i+1), s)
		}

		// change the timestamp to avoid wait
		timestamp := now - int64(len(ts.sentStates))*v.fakeNetworkDelay
		for i := range ts.sentStates {
			ts.sentStates[i].timestamp = timestamp + int64(i)*v.fakeNetworkDelay
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
		ackNum uint64
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
			ts.addSentState(now, uint64(i), s)
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
		// fmt.Printf("#pushUserBytesTo %q into state %p\n", chs[i], t)
	}

	// fmt.Printf("pushUserBytesTo %p %q\n", t, t)
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

			ts.addSentState(time.Now().UnixMilli(), uint64(i+1), state)
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
			ts.addSentState(time.Now().UnixMilli(), uint64(i+1), state)
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
		initialState       string   // first sent state
		states             []string // use string to build sent states
		currentIdx         int      // current state index
		fakeNetworkDelay   int64    // delay between sent states
		lastHeard          int64    // last heard delta
		mindelayClock      int64    // -1 means use current time
		pendingDataAck     bool     // pending data ack
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
			4, 450, 1, 5, true, false, 0, 0,
		},
		{
			"current = newest = assumed != oldest", "abc",
			[]string{"abcd", "abcde", "abcdef", "abcdefg"},
			4, 10, 0, 5, false, false, 0, 0,
		},
		{
			"current = newest, lastHeard over due ", "abc",
			[]string{"abcd", "abcde", "abcdef", "abcdefg"},
			4, 10, -2 * ACTIVE_RETRY_TIMEOUT, 5, false, true, math.MaxInt64, 0,
		},
		{
			"current = newest, lastHeard over due shutdown false", "abc",
			[]string{"abcd", "abcde", "abcdef", "abcdefg"},
			4, 10, -2 * ACTIVE_RETRY_TIMEOUT, 5, true, false, math.MaxInt64, 0,
		},
	}

	waitTime := []int{0, 8, 0, 240, 100}

	for k, v := range tc {
		// for _, v := range tc {
		connection := NewConnection("localhost", "8080")
		initialState := &statesync.UserStream{} // initial state
		pushUserBytesTo(initialState, v.initialState)
		ts := NewTransportSender(connection, initialState)

		// prepare sentStates data
		now := time.Now().UnixMilli()
		for i, keystroke := range v.states {
			state := &statesync.UserStream{}
			pushUserBytesTo(state, keystroke)
			ts.addSentState(now, uint64(i+1), state)
		}

		// change the timestamp to avoid wait
		timestamp := now - int64(len(ts.sentStates))*v.fakeNetworkDelay
		for i := range ts.sentStates {
			ts.sentStates[i].timestamp = timestamp + int64(i)*v.fakeNetworkDelay
			// fmt.Printf("#test calculateTimers i=%d, timestamp=%d, delta=%d\n",
			// 	i, timestamp+int64(i)*v.fakeNetworkDelay, int64(i)*v.fakeNetworkDelay)
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
			ts.setAckNum(math.MaxUint64)
		}
		// ts.assumedReceiverState = &ts.sentStates[v.assumedIdx]

		ts.calculateTimers()

		// validate the result
		gotNextAckTime := ts.nextAckTime
		gotNextSendTime := ts.nextSendTime

		if gotNextAckTime != v.expectNextAckTime {
			if v.expectNextAckTime == math.MaxInt64 {
				t.Errorf("%q expect nextAckTime %d, got %d\n", v.label, v.expectNextAckTime, gotNextAckTime)
			}
		}

		if gotNextSendTime != v.expectNextSendTime {
			if v.expectNextSendTime == math.MaxInt64 {
				t.Errorf("%q expect nextSendTime %d, got %d\n", v.label, v.expectNextSendTime, gotNextSendTime)
			}
		}

		// fmt.Printf("#test nextAckTime=%d, nextSendTime=%d, now=%d\n", ts.nextAckTime, ts.nextSendTime, now)

		// validate waitTime() corner case
		// wait time need:
		// 1) send>now or ack>now
		// 2) hasRemoteAddr = true
		// 3) send < ack
		ts.connection.hasRemoteAddr = true
		gotWaitTime := ts.waitTime()
		if gotWaitTime-waitTime[k] > 1 {
			t.Logf("#test send<ack=%t, send>now=%t, ack>now=%t, gotWaitTiem=%d\n",
				ts.nextSendTime < ts.nextAckTime, ts.nextSendTime > now, ts.nextAckTime > now, gotWaitTime)
			t.Errorf("#test waitTime round:%d, expect %d, got %d\n", k, waitTime[k], gotWaitTime)
		}

		// validate corner case for tick()
		if now < ts.nextAckTime && now < ts.nextSendTime {
			if ts.tick() != nil {
				t.Errorf("#test for now<send, now<ack, tick() should return nil\n")
			}
		}
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

func TestSenderSendEmptyAckShutdown(t *testing.T) {
	initialStateSrv, _ := statesync.NewComplete(80, 40, 40)
	initialRemoteSrv := &statesync.UserStream{}
	desiredIp := "localhost"
	desiredPort := "6100"
	server := NewTransportServer(initialStateSrv, initialRemoteSrv, desiredIp, desiredPort)

	initialState := &statesync.UserStream{}
	initialRemote, _ := statesync.NewComplete(80, 40, 40)
	keyStr := server.connection.getKey()
	client := NewTransportClient(initialState, initialRemote, keyStr, desiredIp, desiredPort)

	// disable log
	// server.connection.logW.SetOutput(io.Discard)
	defer util.Log.Restore()
	util.Log.SetOutput(io.Discard)

	// prepare for shutdown
	client.sender.shutdownInProgress = true
	client.sender.sendEmptyAck()
	time.Sleep(time.Millisecond * 20)

	// fmt.Println("#test shutdown BEFORE.")
	// for i := range server.receivedState {
	// 	fmt.Printf("#test shutdown %d\n", server.receivedState[i].num)
	// }
	server.Recv()
	expect := client.sender.getSentStateLast()
	// got := server.getRemoteStateNum()
	// TODO shutdown send newNum (-1) to peer, with the sorted receivedState, the shutdown logic need to be checked
	got := server.receivedState[len(server.receivedState)-1].num

	if got != expect {
		t.Errorf("#test recv repeat expect %d, got %d\n", expect, got)
		t.Errorf("#test receivedState length=%d\n", len(server.receivedState))
	}

	// fmt.Println("#test shutdown AFTER.")
	// for i := range server.receivedState {
	// 	fmt.Printf("#test shutdown %d\n", server.receivedState[i].num)
	// }

	// clean the socket
	server.connection.sock().Close()
	client.connection.sock().Close()
}

func TestSenderSendEmptyAckFail(t *testing.T) {
	initialStateSrv, _ := statesync.NewComplete(80, 40, 40)
	initialRemoteSrv := &statesync.UserStream{}
	desiredIp := "localhost"
	desiredPort := "6101"
	server := NewTransportServer(initialStateSrv, initialRemoteSrv, desiredIp, desiredPort)

	initialState := &statesync.UserStream{}
	initialRemote, _ := statesync.NewComplete(80, 40, 40)
	keyStr := server.connection.getKey()
	client := NewTransportClient(initialState, initialRemote, keyStr, desiredIp, desiredPort)

	// disable log
	// server.connection.logW.SetOutput(io.Discard)
	defer util.Log.Restore()
	util.Log.SetOutput(io.Discard)

	// mockUdpConn will send with an error: send size doesn't match
	var mock mockUdpConn
	client.connection.socks = append(client.connection.socks, &mock)

	// // validate the result
	// err := client.sender.sendEmptyAck()
	// // fmt.Printf("#test sender sendEmptyAck expect %q\n", err)
	// if err == nil {
	// 	t.Errorf("#test sender sendEmptyAck expect %q\n", err)
	// }

	// validate the tick result
	err := client.Tick()
	// fmt.Printf("#test sender sendEmptyAck expect %q\n", err)
	if err == nil {
		t.Errorf("#test sender sendEmptyAck expect %q\n", err)
	}

	// clean the socket
	server.connection.sock().Close()
	client.connection.sock().Close()
}

func TestSenderSendToReceiverFail(t *testing.T) {
	initialStateSrv, _ := statesync.NewComplete(80, 40, 40)
	initialRemoteSrv := &statesync.UserStream{}
	desiredIp := "localhost"
	desiredPort := "6102"
	server := NewTransportServer(initialStateSrv, initialRemoteSrv, desiredIp, desiredPort)

	initialState := &statesync.UserStream{}
	initialRemote, _ := statesync.NewComplete(80, 40, 40)
	keyStr := server.connection.getKey()
	client := NewTransportClient(initialState, initialRemote, keyStr, desiredIp, desiredPort)

	// disable log
	// server.connection.logW.SetOutput(io.Discard)
	defer util.Log.Restore()
	util.Log.SetOutput(io.Discard)

	// mockUdpConn will send with an error: send size doesn't match
	var mock mockUdpConn
	client.connection.socks = append(client.connection.socks, &mock)

	// validate the result
	err := client.sender.sendToReceiver("send fail")
	// fmt.Printf("#test sender sendEmptyAck expect %s\n", err)
	if err == nil {
		t.Errorf("#test sender sendToReceiver expect %q\n", err)
	}

	// clean the socket
	server.connection.sock().Close()
	client.connection.sock().Close()
}

func TestSenderSendToReceiverShutdown(t *testing.T) {
	initialStateSrv, _ := statesync.NewComplete(80, 40, 40)
	initialRemoteSrv := &statesync.UserStream{}
	desiredIp := "localhost"
	desiredPort := "6103"
	server := NewTransportServer(initialStateSrv, initialRemoteSrv, desiredIp, desiredPort)

	initialState := &statesync.UserStream{}
	initialRemote, _ := statesync.NewComplete(80, 40, 40)
	keyStr := server.connection.getKey()
	client := NewTransportClient(initialState, initialRemote, keyStr, desiredIp, desiredPort)

	// disable log
	// server.connection.logW.SetOutput(io.Discard)
	defer util.Log.Restore()
	util.Log.SetOutput(io.Discard)

	// prepare for shutdown
	client.sender.startShutdown()
	client.sender.sendToReceiver("prepare for shutdown")
	time.Sleep(time.Millisecond * 20)

	// fmt.Println("#test shutdown BEFORE.")
	// for i := range server.receivedState {
	// 	fmt.Printf("#test shutdown %d\n", server.receivedState[i].num)
	// }
	server.Recv()
	expect := client.sender.getSentStateLast()
	// got := server.getRemoteStateNum()
	// TODO shutdown send newNum (-1) to peer, with the sorted receivedState, the shutdown logic need to be checked
	got := server.receivedState[len(server.receivedState)-1].num

	if got != expect {
		t.Errorf("#test recv repeat expect %d, got %d\n", expect, got)
	}

	// fmt.Println("#test shutdown AFTER.")
	// for i := range server.receivedState {
	// 	fmt.Printf("#test shutdown %d\n", server.receivedState[i].num)
	// }

	// clean the socket
	server.connection.sock().Close()
	client.connection.sock().Close()
}

func TestSenderTickNoRemoteAddr(t *testing.T) {
	connection := NewConnection("localhost", "6104")
	completeTerminal, _ := statesync.NewComplete(80, 40, 0)
	ts := NewTransportSender(connection, completeTerminal)

	// tick return quickly, nothing to validate.
	ts.tick()

	connection.sock().Close()
}

func TestSenderTickSendToReceiverFail(t *testing.T) {
	initialStateSrv, _ := statesync.NewComplete(80, 40, 40)
	initialRemoteSrv := &statesync.UserStream{}
	desiredIp := "localhost"
	desiredPort := "6104"
	server := NewTransportServer(initialStateSrv, initialRemoteSrv, desiredIp, desiredPort)

	initialState := &statesync.UserStream{}
	initialRemote, _ := statesync.NewComplete(80, 40, 40)
	keyStr := server.connection.getKey()
	client := NewTransportClient(initialState, initialRemote, keyStr, desiredIp, desiredPort)

	pushUserBytesTo(client.GetCurrentState(), "sendToReceiver failed.")

	// disable log
	// server.connection.logW.SetOutput(io.Discard)
	defer util.Log.Restore()
	util.Log.SetOutput(io.Discard)

	// mockUdpConn will send with an error: send size doesn't match
	var mock mockUdpConn
	client.connection.socks = append(client.connection.socks, &mock)

	// validate the result
	err := client.Tick()
	// fmt.Printf("#test sender sendEmptyAck expect %s\n", err)
	if err == nil {
		t.Errorf("#test sender sendToReceiver expect %q\n", err)
	}

	// clean the socket
	server.connection.sock().Close()
	client.connection.sock().Close()
}

func TestSenderTickVerify(t *testing.T) {
	initialStateSrv, _ := statesync.NewComplete(80, 40, 40)
	initialRemoteSrv := &statesync.UserStream{}
	desiredIp := "localhost"
	desiredPort := "6005"
	server := NewTransportServer(initialStateSrv, initialRemoteSrv, desiredIp, desiredPort)

	initialState := &statesync.UserStream{}
	initialRemote, _ := statesync.NewComplete(80, 40, 40)
	keyStr := server.connection.getKey() // get the key from server
	client := NewTransportClient(initialState, initialRemote, keyStr, desiredIp, desiredPort)

	pushUserBytesTo(client.GetCurrentState(), "first mock input.")

	// set verbose
	server.SetVerbose(1)

	defer util.Log.Restore()
	util.Log.SetLevel(slog.LevelDebug)
	util.Log.SetOutput(os.Stdout)
	// util.Log.SetOutput(io.Discard)

	// intercept stderr
	// swallow the tick() output to stderr
	// saveStderr := os.Stderr
	// r, w, _ := os.Pipe()
	// os.Stderr = w

	// send user stream to server
	client.Tick()
	fmt.Printf("client firt send\n")
	time.Sleep(time.Millisecond * 20)
	server.Recv()
	fmt.Printf("server firt recv\n")
	time.Sleep(time.Millisecond * 20)

	// prepare hook func to change assumedReceiverState
	server.sender.hookForTick = func() {
		// create a fake state
		state, _ := statesync.NewComplete(80, 40, 40)
		state.Act("second mock input")
		num := server.sender.getSentStateLast() + 1
		server.sender.addSentState(time.Now().UnixMilli(), num, state)

		// change the assumedReceiverState
		back := len(server.sender.sentStates) - 1
		server.sender.assumedReceiverState = &server.sender.sentStates[back]
	}

	// send complete to client
	server.Tick()
	fmt.Printf("server first send\n")
	time.Sleep(time.Millisecond * 20)
	client.Recv()
	fmt.Printf("client first recv\n")
	time.Sleep(time.Millisecond * 20)

	// restore stderr
	// w.Close()
	// io.ReadAll(r) // discard the output of stderr
	// // b, _ := ioutil.ReadAll(r)
	// os.Stderr = saveStderr
	// r.Close()

	// validate the result
	// result := string(b)

	// check the stderr output to validate the result.

	// validate client sent and server received contents
	if !server.GetLatestRemoteState().state.Equal(client.GetCurrentState()) {
		t.Errorf("#test client send %q to server, server receive %q from client\n",
			client.GetCurrentState(), server.GetLatestRemoteState().state)
	}

	server.Close()
	client.Close()
}

func TestSenderSendInterval(t *testing.T) {
	initialStateSrv, _ := statesync.NewComplete(80, 40, 40)
	initialRemoteSrv := &statesync.UserStream{}
	desiredIp := "localhost"
	desiredPort := "6005"
	server := NewTransportServer(initialStateSrv, initialRemoteSrv, desiredIp, desiredPort)

	initialState := &statesync.UserStream{}
	initialRemote, _ := statesync.NewComplete(80, 40, 40)
	keyStr := server.connection.getKey() // get the key from server
	client := NewTransportClient(initialState, initialRemote, keyStr, desiredIp, desiredPort)

	defer util.Log.Restore()
	util.Log.SetOutput(io.Discard)

	// round-trip several times to build a reasonable SRTT
	var got string
	maxMsg := 10
	// for i, msg := range messages {
	for i := 0; i < maxMsg; i++ {
		toServer := fmt.Sprintf("%d round from client to server", i)
		toClient := fmt.Sprintf("%d round from server to client", i)

		client.connection.send(toServer, false)
		time.Sleep(time.Millisecond * 5)
		server.Recv()

		server.connection.send(toClient, false)
		time.Sleep(time.Millisecond * 5)
		got, _, _ = client.connection.Recv(1)

		if got != toClient {
			t.Errorf("#test sendInterval expect %q, got %q\n", toClient, got)
		}
		// fmt.Printf("#test sendInterval %d RTTHit=%t SRTT=%f, RTTVAR=%f\n",
		// 	i, client.connection.RTTHit, client.connection.SRTT, client.connection.RTTVAR)
	}

	gotInterval := client.sender.sendInterval()
	if gotInterval != SEND_INTERVAL_MIN {
		t.Errorf("#test sendInterval expect interval %d < %d\n", gotInterval, SEND_INTERVAL_MIN)
	}

	server.connection.sock().Close()
	client.connection.sock().Close()
}

func TestSenderShutdonwAckTimedout(t *testing.T) {
	initialStateSrv, _ := statesync.NewComplete(80, 40, 40)
	initialRemoteSrv := &statesync.UserStream{}
	desiredIp := "localhost"
	desiredPort := "6105"
	ts := NewTransportServer(initialStateSrv, initialRemoteSrv, desiredIp, desiredPort)

	// validate
	if ts.ShutdownAckTimedout() {
		t.Errorf("#test ShutdownAckTimedout() expect false, got %t\n", ts.ShutdownAckTimedout())
	}

	// shutdownTries > SHUTDOWN_RETRIES
	ts.sender.shutdownTries = SHUTDOWN_RETRIES + 1
	ts.StartShutdown()
	if !ts.sender.shutdonwAckTimedout() {
		t.Errorf("#test ShutdownAckTimedout() expect true, got false, shutdownTries=%d\n",
			ts.sender.shutdownTries)
	}

	// shutdownStart >= ACTIVE_RETRY_TIMEOUT
	ts.sender.shutdownStart = time.Now().UnixMilli() - ACTIVE_RETRY_TIMEOUT - 1
	ts.sender.shutdownTries = 1
	if !ts.sender.shutdonwAckTimedout() {
		t.Errorf("#test ShutdownAckTimedout() expect true, got false, shutdownStart=%d\n",
			ts.sender.shutdownStart)
	}

	// test SetSendDelay
	ts.sender.SEND_MINDELAY = 0
	var expect uint = 12
	ts.SetSendDelay(expect)
	if ts.sender.SEND_MINDELAY != uint(expect) {
		t.Errorf("#test setSendDelay() expect %d, got %d\n", expect, ts.sender.SEND_MINDELAY)
	}

	// test hasRemoteAddr
	if ts.HasRemoteAddr() {
		t.Errorf("test HasRemoteAddr() expect false, got %t\n", ts.HasRemoteAddr())
	}
}
