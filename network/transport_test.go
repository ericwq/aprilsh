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
	"io"
	"strings"
	"testing"
	"time"

	pb "github.com/ericwq/aprilsh/protobufs"
	"github.com/ericwq/aprilsh/statesync"
	"github.com/ericwq/aprilsh/terminal"
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

	pushUserBytesTo(client.getCurrentState(), "Test client send and server empty ack.")
	// fmt.Printf("#test tickAndRecv currentState=%q pointer=%v, assumed=%d\n",
	// 	client.getCurrentState(), client.getCurrentState(), client.sender.getAssumedReceiverStateIdx())

	// disable log
	server.connection.logW.SetOutput(io.Discard)

	// send user stream to server
	client.tick()
	time.Sleep(time.Millisecond * 20)
	server.recv()
	time.Sleep(time.Millisecond * 20)

	// validate sentStates status
	var expectNum int64
	gotNum := client.sender.getSentStateAcked()
	if gotNum != expectNum {
		t.Errorf("#test R1 client sentStates expect first num %d, got %d\n", expectNum, gotNum)
	}

	// send complete to client
	server.tick()
	time.Sleep(time.Millisecond * 20)
	client.recv()
	time.Sleep(time.Millisecond * 20)

	// validate client sent and server received contents
	if !server.getLatestRemoteState().state.Equal(client.getCurrentState()) {
		fmt.Printf("#test client send %q to server, server receive %q from client\n",
			client.getCurrentState(), server.getLatestRemoteState().state)
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

	pushUserBytesTo(client.getCurrentState(), "Test server response with terminal state.")
	// fmt.Printf("#test tickAndRecv currentState=%q pointer=%v, assumed=%d\n",
	// 	client.getCurrentState(), client.getCurrentState(), client.sender.getAssumedReceiverStateIdx())

	// set verbose
	client.setVerbose(1)
	server.setVerbose(1)

	// send user stream to server
	client.tick()
	time.Sleep(time.Millisecond * 20)
	server.recv()
	time.Sleep(time.Millisecond * 20)

	// check remote address
	if server.getRemoteAddr() == nil {
		t.Errorf("#test server send expect remote address %v, got nil\n", server.getRemoteAddr())
	}

	// apply remote diff to server current state
	us := &statesync.UserStream{}
	diff := server.getRemoteDiff()
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
	completeTerminal.RegisterInputFrame(server.getRemoteStateNum(), time.Now().UnixMilli())
	server.setCurrentState(completeTerminal)
	// fmt.Printf("#test currentState=%p, terminalInSrv=%p\n", server.getCurrentState(), completeTerminal)

	// send complete to client
	server.tick()
	time.Sleep(time.Millisecond * 20)
	client.recv()
	time.Sleep(time.Millisecond * 20)

	// validate the result
	// fmt.Printf("#test server currentState=%p, client last remoteState=%p\n", server.getCurrentState(), client.getLatestRemoteState().state)
	if !server.getCurrentState().Equal(client.getLatestRemoteState().state) {
		t.Errorf("#test server send %v to client, client got %v\n ", server.getCurrentState(), client.getLatestRemoteState().state)
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
	if err := server.recv(); err != nil {
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

	err := server.recv()
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

	// first round
	pushUserBytesTo(client.getCurrentState(), "first regular send")
	client.tick()
	time.Sleep(time.Millisecond * 20)
	server.recv()
	time.Sleep(time.Millisecond * 20)

	// second round, send repeat state
	var newNum int64 = 1
	client.sender.sendInFragments("", newNum)
	time.Sleep(time.Millisecond * 20)

	server.recv()
	got := server.receivedState[1].num
	if got != newNum {
		t.Errorf("#test recv repeat expect %q, got %q\n", newNum, got)
	}

	// coverage for waitTime
	server.waitTime()

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

	err := server.recv()
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
	server.setVerbose(1)

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

	server.recv()
	if server.receiverQuenchTimer-time.Now().UnixMilli() > 1000 {
		// that is the expected result
		// t.Logf("#test recv over limit, receivedQuenchTimer=%d, now=%d\n", server.receiverQuenchTimer, time.Now().UnixMilli())
	} else {
		t.Errorf("#test recv over limit, receivedQuenchTimer=%d, now=%d\n", server.receiverQuenchTimer, time.Now().UnixMilli())
	}

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
	server.setVerbose(1)

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
	err := server.recv()
	if err != nil {
		t.Errorf("#test recv over limit, receivedQuenchTimer=%d, now=%d\n", server.receiverQuenchTimer, time.Now().UnixMilli())
	}
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
	server.setVerbose(1)

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
	server.recv()
	if server.receivedState[2].num != newNum {
		t.Errorf("#test recv expect %d, got %q\n", newNum, server.receivedState[2].num)
	}

	server.connection.sock().Close()
	client.connection.sock().Close()
}
