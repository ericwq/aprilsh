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
	"testing"
	"time"

	"github.com/ericwq/aprilsh/statesync"
	"github.com/ericwq/aprilsh/terminal"
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

	// send user stream to server
	client.tick()
	time.Sleep(time.Millisecond * 20)
	server.recv()
	time.Sleep(time.Millisecond * 20)

	// send complete to client
	server.tick()
	time.Sleep(time.Millisecond * 20)
	client.recv()
	time.Sleep(time.Millisecond * 20)

	// validate client sent and server received contents
	if !server.getLatestRemoteState().state.Equal(client.getCurrentState()) {
		fmt.Printf("#test TickAndReceive client send %q to server, server receive %q from client\n",
			client.getCurrentState(), server.getLatestRemoteState().state)
	}

	server.connection.sock().Close()
	client.connection.sock().Close()
}

func TestTransportServerSend(t *testing.T) {
	terminalInSrv, _ := statesync.NewComplete(80, 40, 40)
	initialRemoteSrv := &statesync.UserStream{}
	desiredIp := "localhost"
	desiredPort := "6001"
	server := NewTransportServer(terminalInSrv, initialRemoteSrv, desiredIp, desiredPort)

	initialState := &statesync.UserStream{}
	initialRemote, _ := statesync.NewComplete(80, 40, 40)
	keyStr := server.connection.getKey() // get the key from server
	ip := "localhost"
	port := "6001"
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

	// apply remote diff to server current state
	us := &statesync.UserStream{}
	diff := server.getRemoteDiff()
	us.ApplyString(diff)
	terminalToHost := ""
	for i := 0; i < us.Size(); i++ {
		action := us.GetAction(i)
		switch action.(type) {
		case terminal.UserByte:
		case terminal.Resize:
			// resize the terminal
		}
		terminalToHost = terminalInSrv.ActOne(action)
	}

	fmt.Printf("#test server send: got diff %q, terminalToHost=%q\n", diff, terminalToHost)
	terminalInSrv.RegisterInputFrame(server.getRemoteStateNum(), time.Now().UnixMilli())
	server.setCurrentState(terminalInSrv)

	// send complete to client
	server.tick()
	time.Sleep(time.Millisecond * 20)
	client.recv()
	time.Sleep(time.Millisecond * 20)

	server.connection.sock().Close()
	client.connection.sock().Close()
}
