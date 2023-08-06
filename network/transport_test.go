// Copyright 2022 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package network

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"testing"
	"time"

	pb "github.com/ericwq/aprilsh/protobufs"
	"github.com/ericwq/aprilsh/statesync"
	"github.com/ericwq/aprilsh/terminal"
	"github.com/ericwq/aprilsh/util"
	"golang.org/x/sys/unix"
)

func TestTransportClientSend(t *testing.T) {
	initialStateSrv, _ := statesync.NewComplete(80, 40, 40)
	initialRemoteSrv := &statesync.UserStream{}
	desiredIp := "localhost"
	desiredPort := "6000"
	server := NewTransportServer(initialStateSrv, initialRemoteSrv, desiredIp, desiredPort)

	initialState := &statesync.UserStream{}
	initialRemote, _ := statesync.NewComplete(80, 40, 40)
	keyStr := server.connection.getKey() // get the key from server
	ip := "localhost"
	port := "6000"
	client := NewTransportClient(initialState, initialRemote, keyStr, ip, port)

	pushUserBytesTo(client.GetCurrentState(), "Test client send and server empty ack.")
	// fmt.Printf("#test tickAndRecv currentState=%q pointer=%v, assumed=%d\n",
	// 	client.getCurrentState(), client.getCurrentState(), client.sender.getAssumedReceiverStateIdx())

	// disable log
	// server.connection.logW.SetOutput(io.Discard)
	defer util.Log.Restore()
	util.Log.SetOutput(io.Discard)

	// send user stream to server
	client.Tick()
	time.Sleep(time.Millisecond * 20)
	server.Recv()
	time.Sleep(time.Millisecond * 20)

	// validate sentStates status
	var expectNum int64
	gotNum := client.sender.getSentStateAcked()
	if gotNum != expectNum {
		t.Errorf("#test R1 client sentStates expect first num %d, got %d\n", expectNum, gotNum)
	}

	// send complete to client
	server.Tick()
	time.Sleep(time.Millisecond * 20)
	client.Recv()
	time.Sleep(time.Millisecond * 20)

	// validate client sent and server received contents
	if !server.GetLatestRemoteState().state.Equal(client.GetCurrentState()) {
		t.Errorf("#test client send %q to server, server receive %q from client\n",
			client.GetCurrentState(), server.GetLatestRemoteState().state)
	}

	// validate sentStates shrink after a server response
	expectNum = 1
	gotNum = client.sender.getSentStateAcked()
	if gotNum != expectNum {
		t.Errorf("#test client sentStates expect first num %d, got %d\n", expectNum, gotNum)
	}

	server.connection.sock().Close()
	client.connection.sock().Close()
}

func TestTransportServerSend(t *testing.T) {
	completeTerminal, _ := statesync.NewComplete(80, 5, 0)
	blank := &statesync.UserStream{}
	desiredIp := "localhost"
	desiredPort := "6010"
	server := NewTransportServer(completeTerminal, blank, desiredIp, desiredPort)

	initialState := &statesync.UserStream{}
	initialRemote, _ := statesync.NewComplete(80, 5, 0)
	keyStr := server.connection.getKey() // get the key from server
	ip := "localhost"
	port := "6010"
	client := NewTransportClient(initialState, initialRemote, keyStr, ip, port)

	pushUserBytesTo(client.GetCurrentState(), "Test server response with terminal state.")
	// fmt.Printf("#test tickAndRecv currentState=%q pointer=%v, assumed=%d\n",
	// 	client.GetCurrentState(), client.GetCurrentState(), client.sender.getAssumedReceiverStateIdx())

	// set verbose
	client.SetVerbose(1)
	server.SetVerbose(1)

	// intercept stderr
	// swallow the tick() output to stderr
	saveStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w
	// disable log
	// server.connection.logW.SetOutput(io.Discard)
	// client.connection.logW.SetOutput(io.Discard)
	defer util.Log.Restore()
	util.Log.SetOutput(io.Discard)

	// send user stream to server
	client.Tick()
	time.Sleep(time.Millisecond * 20)
	server.Recv()
	time.Sleep(time.Millisecond * 20)

	// check remote address
	if server.GetRemoteAddr() == nil {
		t.Errorf("#test server send expect remote address %v, got nil\n", server.GetRemoteAddr())
	}

	// apply remote diff to server current state
	us := &statesync.UserStream{}
	diff := server.GetRemoteDiff()
	us.ApplyString(diff)
	terminalToHost := ""
	for i := 0; i < us.Size(); i++ {
		action := us.GetAction(i)
		switch action.(type) {
		case terminal.UserByte:
			// fmt.Printf("#test process %#v\n", action)
		case terminal.Resize:
			// fmt.Printf("#test process %#v\n", action)
			// resize the terminal
		}
		terminalToHost += completeTerminal.ActOne(action)
	}

	// fmt.Printf("#test server send: got diff %q, terminalToHost=%q\n", diff, terminalToHost)
	completeTerminal.Act(terminalToHost)
	completeTerminal.RegisterInputFrame(server.GetRemoteStateNum(), time.Now().UnixMilli())
	server.SetCurrentState(completeTerminal)
	// fmt.Printf("#test currentState=%p, terminalInSrv=%p\n", server.getCurrentState(), completeTerminal)

	// send complete to client
	server.Tick()
	time.Sleep(time.Millisecond * 20)
	client.Recv()
	time.Sleep(time.Millisecond * 20)

	// restore stderr
	w.Close()
	ioutil.ReadAll(r) // discard the output of stderr
	// b, _ := ioutil.ReadAll(r)
	os.Stderr = saveStderr
	r.Close()

	// validate the result
	// fmt.Printf("#test server currentState=%p, client last remoteState=%p\n", server.getCurrentState(), client.getLatestRemoteState().state)
	if !server.GetCurrentState().Equal(client.GetLatestRemoteState().state) {
		t.Errorf("#test server send %v to client, client got %v\n ", server.GetCurrentState(), client.GetLatestRemoteState().state)
	}
	server.connection.sock().Close()
	client.connection.sock().Close()
}

func TestTransportRecvError(t *testing.T) {
	completeTerminal, _ := statesync.NewComplete(80, 5, 0)
	blank := &statesync.UserStream{}
	desiredIp := "localhost"
	desiredPort := "6011"
	server := NewTransportServer(completeTerminal, blank, desiredIp, desiredPort)

	// mockUdpConn with round=0 will return unix.EWOULDBLOCK error
	var mock mockUdpConn
	server.connection.socks = append(server.connection.socks, &mock)
	server.connection.socks = server.connection.socks[len(server.connection.socks)-1:]

	// validate
	if err := server.Recv(); err != nil {
		if !errors.Is(err, unix.EWOULDBLOCK) {
			t.Errorf("#test recv error expect err=%q, got %q\n", unix.EWOULDBLOCK, err)
		}
	}
}

func TestTransportRecvVersionError(t *testing.T) {
	initialStateSrv, _ := statesync.NewComplete(80, 40, 40)
	initialRemoteSrv := &statesync.UserStream{}
	desiredIp := "localhost"
	desiredPort := "6002"
	server := NewTransportServer(initialStateSrv, initialRemoteSrv, desiredIp, desiredPort)

	initialState := &statesync.UserStream{}
	initialRemote, _ := statesync.NewComplete(80, 40, 40)
	keyStr := server.connection.getKey() // get the key from server
	ip := "localhost"
	port := "6002"
	client := NewTransportClient(initialState, initialRemote, keyStr, ip, port)

	defer util.Log.Restore()
	util.Log.SetOutput(io.Discard)

	// send customized instruction to server
	var newNum int64 = 1
	inst := pb.Instruction{}
	inst.ProtocolVersion = APRILSH_PROTOCOL_VERSION + 1 // mock version
	inst.OldNum = client.sender.assumedReceiverState.num
	inst.NewNum = newNum
	inst.AckNum = client.sender.ackNum
	inst.ThrowawayNum = client.sender.sentStates[0].num
	inst.Diff = []byte("")
	inst.Chaff = []byte(client.sender.makeChaff())
	client.sender.sendFragments(&inst, newNum)

	time.Sleep(time.Millisecond * 20)

	err := server.Recv()
	if err != nil {
		expect := errors.New("aprilsh protocol version mismatch.")
		if err.Error() != expect.Error() {
			t.Errorf("#test recv error expect %q, got %q\n", expect, err)
		}
	}

	server.connection.sock().Close()
	client.connection.sock().Close()
}

func TestTransportRecvRepeat(t *testing.T) {
	initialStateSrv, _ := statesync.NewComplete(80, 40, 40)
	initialRemoteSrv := &statesync.UserStream{}
	desiredIp := "localhost"
	desiredPort := "6003"
	server := NewTransportServer(initialStateSrv, initialRemoteSrv, desiredIp, desiredPort)

	initialState := &statesync.UserStream{}
	initialRemote, _ := statesync.NewComplete(80, 40, 40)
	keyStr := server.connection.getKey() // get the key from server
	ip := "localhost"
	port := "6003"
	client := NewTransportClient(initialState, initialRemote, keyStr, ip, port)

	defer util.Log.Restore()
	util.Log.SetOutput(io.Discard)

	// first round
	pushUserBytesTo(client.GetCurrentState(), "first regular send")
	client.Tick()
	time.Sleep(time.Millisecond * 20)
	server.Recv()
	time.Sleep(time.Millisecond * 20)

	// second round, send repeat state
	var newNum int64 = 1
	client.sender.sendInFragments("", newNum)
	time.Sleep(time.Millisecond * 20)

	server.Recv()
	got := server.receivedState[1].num
	if got != newNum {
		t.Errorf("#test recv repeat expect %q, got %q\n", newNum, got)
	}

	// coverage for waitTime
	server.WaitTime()

	// clean the socket
	server.connection.sock().Close()
	client.connection.sock().Close()
}

func TestTransportRecvNotFoundOld(t *testing.T) {
	initialStateSrv, _ := statesync.NewComplete(80, 40, 40)
	initialRemoteSrv := &statesync.UserStream{}
	desiredIp := "localhost"
	desiredPort := "6004"
	server := NewTransportServer(initialStateSrv, initialRemoteSrv, desiredIp, desiredPort)

	initialState := &statesync.UserStream{}
	initialRemote, _ := statesync.NewComplete(80, 40, 40)
	keyStr := server.connection.getKey() // get the key from server
	ip := "localhost"
	port := "6004"
	client := NewTransportClient(initialState, initialRemote, keyStr, ip, port)

	defer util.Log.Restore()
	util.Log.SetOutput(io.Discard)

	// send customized instruction to server
	var newNum int64 = 1
	inst := pb.Instruction{}
	inst.ProtocolVersion = APRILSH_PROTOCOL_VERSION
	inst.OldNum = 3 // oldNum doesn't exist
	inst.NewNum = newNum
	inst.AckNum = client.sender.ackNum
	inst.ThrowawayNum = client.sender.sentStates[0].num
	inst.Diff = []byte("")
	inst.Chaff = []byte(client.sender.makeChaff())
	client.sender.sendFragments(&inst, newNum)

	time.Sleep(time.Millisecond * 20)

	err := server.Recv()
	expect := "Ignoring out-of-order packet. Reference state"
	if !strings.Contains(err.Error(), expect) {
		t.Errorf("#test recv expect %q, got %q\n", expect, err)
	}

	server.connection.sock().Close()
	client.connection.sock().Close()
}

func TestTransportRecvOverLimit(t *testing.T) {
	initialStateSrv, _ := statesync.NewComplete(80, 40, 40)
	initialRemoteSrv := &statesync.UserStream{}
	desiredIp := "localhost"
	desiredPort := "6005"
	server := NewTransportServer(initialStateSrv, initialRemoteSrv, desiredIp, desiredPort)

	initialState := &statesync.UserStream{}
	initialRemote, _ := statesync.NewComplete(80, 40, 40)
	keyStr := server.connection.getKey() // get the key from server
	ip := "localhost"
	port := "6005"
	client := NewTransportClient(initialState, initialRemote, keyStr, ip, port)

	// set verbose
	server.SetVerbose(1)

	// intercept stderr
	// swallow the tick() output to stderr
	saveStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	// disable log
	// server.connection.logW.SetOutput(io.Discard)
	defer util.Log.Restore()
	util.Log.SetOutput(io.Discard)

	// prepare the receivedState list
	for i := 0; i < 1024; i++ {
		server.receivedState = append(server.receivedState,
			TimestampedState[*statesync.UserStream]{time.Now().UnixMilli(), +1, initialState.Clone()})
		// time.Sleep(time.Millisecond * 2)
	}

	// send customized instruction to server
	var newNum int64 = 1024
	client.sender.sendInFragments("", newNum)
	time.Sleep(time.Millisecond * 20)

	server.Recv()
	if server.receiverQuenchTimer-time.Now().UnixMilli() > 1000 {
		// that is the expected result
		// t.Logf("#test recv over limit, receivedQuenchTimer=%d, now=%d\n", server.receiverQuenchTimer, time.Now().UnixMilli())
	} else {
		t.Errorf("#test recv over limit, receivedQuenchTimer=%d, now=%d\n", server.receiverQuenchTimer, time.Now().UnixMilli())
	}

	// restore stderr
	w.Close()
	ioutil.ReadAll(r) // discard the output of stderr
	// b, _ := ioutil.ReadAll(r)
	os.Stderr = saveStderr
	r.Close()

	server.connection.sock().Close()
	client.connection.sock().Close()
}

func TestTransportRecvOverLimit2(t *testing.T) {
	initialStateSrv, _ := statesync.NewComplete(80, 40, 40)
	initialRemoteSrv := &statesync.UserStream{}
	desiredIp := "localhost"
	desiredPort := "6005"
	server := NewTransportServer(initialStateSrv, initialRemoteSrv, desiredIp, desiredPort)

	initialState := &statesync.UserStream{}
	initialRemote, _ := statesync.NewComplete(80, 40, 40)
	keyStr := server.connection.getKey() // get the key from server
	ip := "localhost"
	port := "6005"
	client := NewTransportClient(initialState, initialRemote, keyStr, ip, port)

	// set verbose
	server.SetVerbose(1)

	// intercept stderr
	// swallow the tick() output to stderr
	saveStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	// disable log
	// server.connection.logW.SetOutput(io.Discard)
	defer util.Log.Restore()
	util.Log.SetOutput(io.Discard)

	// prepare the receivedState list
	for i := 0; i < 1024; i++ {
		server.receivedState = append(server.receivedState,
			TimestampedState[*statesync.UserStream]{time.Now().UnixMilli(), +1, initialState.Clone()})
		// time.Sleep(time.Millisecond * 2)
	}

	// send customized instruction to server
	var newNum int64 = 1024
	client.sender.sendInFragments("", newNum)
	time.Sleep(time.Millisecond * 20)

	// pre-condition for this limit branch
	server.receiverQuenchTimer = time.Now().UnixMilli() + 100

	// validate the result
	err := server.Recv()
	if err != nil {
		t.Errorf("#test recv over limit, receivedQuenchTimer=%d, now=%d\n", server.receiverQuenchTimer, time.Now().UnixMilli())
	}

	// restore stderr
	w.Close()
	ioutil.ReadAll(r) // discard the output of stderr
	// b, _ := ioutil.ReadAll(r)
	os.Stderr = saveStderr
	r.Close()

	server.connection.sock().Close()
	client.connection.sock().Close()
}

func TestTransportRecvOutOfOrder(t *testing.T) {
	initialStateSrv, _ := statesync.NewComplete(80, 40, 40)
	initialRemoteSrv := &statesync.UserStream{}
	desiredIp := "localhost"
	desiredPort := "6026"
	server := NewTransportServer(initialStateSrv, initialRemoteSrv, desiredIp, desiredPort)

	initialState := &statesync.UserStream{}
	initialRemote, _ := statesync.NewComplete(80, 40, 40)
	keyStr := server.connection.getKey() // get the key from server
	ip := "localhost"
	port := "6026"
	client := NewTransportClient(initialState, initialRemote, keyStr, ip, port)

	// set verbose
	// client.setVerbose(1)
	server.SetVerbose(1)

	// intercept stderr
	// swallow the tick() output to stderr
	saveStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	// disable log
	// server.connection.logW.SetOutput(io.Discard)
	defer util.Log.Restore()
	util.Log.SetOutput(io.Discard)

	// prepare the receivedState list
	server.receivedState = append(server.receivedState,
		TimestampedState[*statesync.UserStream]{time.Now().UnixMilli(), 1, initialState.Clone()})
	time.Sleep(time.Millisecond * 10)
	server.receivedState = append(server.receivedState,
		TimestampedState[*statesync.UserStream]{time.Now().UnixMilli(), 4, initialState.Clone()})
	time.Sleep(time.Millisecond * 10)

	// send customized instruction to server
	var newNum int64 = 3
	client.sender.sendInFragments("", newNum)
	time.Sleep(time.Millisecond * 20)

	// validate the order of state
	server.Recv()
	if server.receivedState[2].num != newNum {
		t.Errorf("#test recv expect %d, got %q\n", newNum, server.receivedState[2].num)
	}

	// restore stderr
	w.Close()
	ioutil.ReadAll(r) // discard the output of stderr
	// b, _ := ioutil.ReadAll(r)
	os.Stderr = saveStderr
	r.Close()

	server.connection.sock().Close()
	client.connection.sock().Close()
}

func TestClientShutdown(t *testing.T) {
	completeTerminal, _ := statesync.NewComplete(80, 5, 0)
	blank := &statesync.UserStream{}
	desiredIp := "localhost"
	desiredPort := "60100"
	server := NewTransportServer(completeTerminal, blank, desiredIp, desiredPort)

	// fmt.Printf("#test server initialize sentStates=%d\n",len(server.sender.sentStates))

	initialState := &statesync.UserStream{}
	initialRemote, _ := statesync.NewComplete(80, 5, 0)
	keyStr := server.connection.getKey() // get the key from server
	ip := "localhost"
	port := "60100"
	client := NewTransportClient(initialState, initialRemote, keyStr, ip, port)

	// fmt.Printf("#test client initialize sentStates=%d\n",len(client.sender.sentStates))

	// mimic user input
	label := "client shutdown"
	pushUserBytesTo(client.GetCurrentState(), label)

	// set verbose
	client.SetVerbose(1)
	server.SetVerbose(1)

	// intercept stderr
	// swallow the tick() output to stderr
	saveStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	// disable log
	defer util.Log.Restore()
	util.Log.SetOutput(io.Discard)
	// server.connection.logW.SetOutput(io.Discard)
	// client.connection.logW.SetOutput(io.Discard)

	// printClientStates(client, label)
	// printServerStates(server, label)

	// send user stream to server
	// fmt.Printf("#test --- client send.\n")
	client.StartShutdown()
	time.Sleep(time.Millisecond * 250)
	client.Tick()
	// printClientStates(client, label)

	// validate
	if !client.ShutdownInProgress() {
		t.Errorf("#test %s: ShutdownInProgress() expect true, got false\n", label)
	}

	// validate
	if client.ShutdownAcknowledged() {
		t.Errorf("#test %s: ShutdownAcknowledged() expect true, got %t\n", label, client.ShutdownAcknowledged())
	}
	// validate
	if client.ShutdownAckTimedout() {
		t.Errorf("#test %s: ShutdownAckTimedout expect false, got %t\n", label, client.ShutdownAckTimedout())
	}

	// fmt.Printf("#test --- server receive.\n")
	server.SetReadDeadline(time.Now().Add(time.Millisecond * time.Duration(5)))
	server.Recv()
	time.Sleep(time.Millisecond * 10)
	// printServerStates(server, label)

	// check remote address
	if server.GetRemoteAddr() == nil {
		t.Errorf("#test %s: GetRemoteAddr() expect remote address %v, got nil\n", label, server.GetRemoteAddr())
	}

	us := &statesync.UserStream{}
	diff := server.GetRemoteDiff()
	us.ApplyString(diff)
	terminalToHost := ""
	for i := 0; i < us.Size(); i++ {
		action := us.GetAction(i)
		switch action.(type) {
		case terminal.UserByte:
			// fmt.Printf("#test process %#v\n", action)
		case terminal.Resize:
			// fmt.Printf("#test process %#v\n", action)
			// resize the terminal
		}
		terminalToHost += completeTerminal.ActOne(action)
	}

	completeTerminal.Act(terminalToHost)
	completeTerminal.RegisterInputFrame(server.GetRemoteStateNum(), time.Now().UnixMilli())
	server.SetCurrentState(completeTerminal)

	// send complete to client
	// fmt.Printf("#test --- server send.\n")
	server.StartShutdown()
	server.Tick()
	time.Sleep(time.Millisecond * 10)
	// printServerStates(server, label)

	// fmt.Printf("#test --- client receive.\n")
	client.SetReadDeadline(time.Now().Add(time.Millisecond * time.Duration(5)))
	e23 := client.Recv()
	time.Sleep(time.Millisecond * 10)
	if e23 != nil {
		fmt.Printf("#test client receive %q.\n", e23)
	}
	// printClientStates(client, label)

	// validate
	if client.CounterpartyShutdownAckSent() {
		t.Errorf("#test %s: CounterpartyShutdownAckSent() expect %t, got %t\n",
			label, true, client.CounterpartyShutdownAckSent())
	}

	// validate the server state is the same as the client received state
	if !server.GetCurrentState().Equal(client.GetLatestRemoteState().state) {
		t.Errorf("#test %s: %v to client, client got %v\n ", label, server.GetCurrentState(), client.GetLatestRemoteState().state)
	}

	// validate
	if client.ShutdownAcknowledged() {
		t.Errorf("#test %s: ShutdownAcknowledged() expect false, got %t\n", label, client.ShutdownAcknowledged())
	}

	// restore stderr
	w.Close()
	ioutil.ReadAll(r) // discard the output of stderr
	// b, _ := ioutil.ReadAll(r)
	os.Stderr = saveStderr
	r.Close()

	server.connection.sock().Close()
	client.connection.sock().Close()
}

func printClientStates(client *Transport[*statesync.UserStream, *statesync.Complete], label string) {
	for i := range client.receivedState {
		fmt.Printf("#test %s: client receivedState[%d] num=%d\n", label, i, client.receivedState[i].num)
	}
	for i := range client.sender.sentStates {
		fmt.Printf("#test %s: client sentStates[%d] num=%d\n", label, i, client.sender.sentStates[i].num)
	}
	// fmt.Printf("#test %s: client AckNum=%d\n", label, client.sender.ackNum)
}

func printServerStates(server *Transport[*statesync.Complete, *statesync.UserStream], label string) {
	for i := range server.receivedState {
		fmt.Printf("#test %s: server receivedState[%d] num=%d\n", label, i, server.receivedState[i].num)
	}
	for i := range server.sender.sentStates {
		fmt.Printf("#test %s: server sentStates[%d] num=%d\n", label, i, server.sender.sentStates[i].num)
	}
	// fmt.Printf("#test %s: server AckNum=%d\n", label, server.sender.ackNum)
}

func TestTransportGetXXX(t *testing.T) {
	completeTerminal, _ := statesync.NewComplete(80, 5, 0)
	blank := &statesync.UserStream{}
	desiredIp := "localhost"
	desiredPort := "60101"
	s := NewTransportServer(completeTerminal, blank, desiredIp, desiredPort)

	// test GetKey
	if len(s.GetKey()) <= 0 {
		t.Errorf("#test GetKey() expect a key string, got %q\n", s.GetKey())
	}

	// test GetSentStateLast
	got := s.GetSentStateLast()
	if got != 0 {
		t.Errorf("#test GetSentStateLast() expect 0, got %d\n", got)
	}

	// test GetSentStateAcked
	if got := s.GetSentStateAcked(); got != 0 {
		t.Errorf("#test GetSentStateAcked() expect 0, got %d\n", got)
	}

	// test GetSentStateAckedTimestamp
	now := time.Now().UnixMilli()
	if got := s.GetSentStateAckedTimestamp(); got != now {
		t.Errorf("#test GetSentStateAckedTimestamp() expect %d, got %d\n", now, got)
	}

	// test SentInterval
	if got := s.SentInterval(); got != SEND_INTERVAL_MAX {
		t.Errorf("#test SentInterval() expect %d, got %d\n", SEND_INTERVAL_MAX, got)
	}

	// test Close
	s.Close()
}
