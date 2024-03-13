// Copyright 2022~2024 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package network

import (
	"strings"
	"testing"
	"time"

	"github.com/ericwq/aprilsh/util"
)

func TestRecvCongestionPacket(t *testing.T) {
	// prepare the client and server connection
	title := "receive congestion packet branch"
	ip := ""
	port := "8080"

	// intercept server log
	var output strings.Builder
	// util.Logger.CreateLogger(&output, true, slog.LevelDebug)
	util.Logger.CreateLogger(&output, false, util.LevelTrace)

	server := NewConnection(ip, port)
	defer server.sock().Close()
	if server == nil {
		t.Errorf("%q server should not return nil.\n", title)
		return
	}

	key := server.key
	client := NewConnectionClient(key.String(), ip, port)
	defer client.sock().Close()
	if client == nil {
		t.Errorf("%q client should not return nil.\n", title)
	}

	msg0 := "from client to server"
	// client send a message to server, server receive it.
	// this will initialize server remote address.
	client.send(msg0, false)
	time.Sleep(time.Millisecond * 20)

	// save old congestionFunc
	oldCF := congestionFunc
	// mock the congestion case
	congestionFunc = func(in byte) bool {
		return true
	}
	server.Recv(1)
	// restore congestionFunc
	congestionFunc = oldCF

	// validate the result
	expect := "#recvOne received explicit congestion notification"
	got := output.String()
	if !strings.Contains(got, expect) {
		t.Errorf("%q expect \n%q, got \n%s\n", title, expect, got)
	}

	// if server.savedTimestamp <= 0 {
	// 	t.Errorf("%q savedTimestamp should be greater than zero, it's %d\n", title, server.savedTimestamp)
	// }
}
