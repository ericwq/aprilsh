// Copyright 2022~2023 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package network

import (
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ericwq/aprilsh/encrypt"
	"github.com/ericwq/aprilsh/util"
	"golang.org/x/sys/unix"
)

func TestPacket(t *testing.T) {
	tc := []struct {
		name       string
		seqNonce   uint64
		mixPayload string
	}{
		{"english message", uint64(0x5223), "\x12\x23\x34\x45normal message"},
		{"chinese message", uint64(0x7226) | DIRECTION_MASK, "\x42\x23\x64\x45大端字节序就和我们平时的写法顺序一样"},
	}

	// test NewPacket2 and toMessage
	for _, v := range tc {
		m1 := encrypt.NewMessage(v.seqNonce, []byte(v.mixPayload))
		p := NewPacketFrom(m1)

		m2 := p.toMessage()

		if !reflect.DeepEqual(*m1, *m2) {
			t.Errorf("%q expect same message, got m1 and m2.\n%v\n%v\n", v.name, m1, m2)
		}
	}

	tc2 := []struct {
		name      string
		direction Direction
		ts1       uint16
		ts2       uint16
		payload   string
	}{
		{"english packet", TO_CLIENT, 1, 2, "normal message"},
		{"chinese packet", TO_SERVER, 4, 5, "大端字节序就和我们平时的写法顺序一样"},
	}

	// test NewPacket func
	for i, v := range tc2 {
		p := NewPacket(v.direction, v.ts1+timestamp16(), v.ts2+timestamp16(), []byte(v.payload))

		if v.payload != string(p.payload) {
			t.Errorf("%q expect payload %q, got %q\n", v.name, v.payload, p.payload)
		}

		if v.direction != p.direction {
			t.Errorf("%q expect direction %d, got %d\n", v.name, v.direction, p.direction)
		}

		if v.ts2-v.ts1 != p.timestampReply-p.timestamp {
			t.Errorf("%q expect ts2-ts1 %d, got %d\n", v.name, v.ts2-v.ts1, p.timestampReply-p.timestamp)
		}

		if p.seq != uint64(i+1) {
			t.Errorf("%q expect seq >0, got %d", v.name, p.seq)
		}
	}
}

func TestParsePortRange(t *testing.T) {
	tc := []struct {
		name     string
		portStr  string
		portLow  int
		portHigh int
		msg      string
	}{
		{"normal port range", "3:65534", 3, 65534, ""},
		{"outof scope number low", "-1:536", -1, -1, "#parsePort port number is outside of valid range [0..65535]"},
		{"outof scope number high", "0:65536", -1, -1, "#parsePort port number is outside of valid range [0..65535]"},
		{"invalid number", "3a", -1, -1, "#parsePort invalid port number"},
		{"port order reverse", "3:1", -1, -1, "#ParsePortRange low port is greater than high port"},
		{"solo port", "5", 5, 5, ""},
	}

	var place strings.Builder
	// output := log.New(&place, "WARN: ", log.Ldate|log.Ltime|log.Lshortfile)
	defer util.Log.Restore()
	util.Log.SetOutput(&place)

	for _, v := range tc {
		place.Reset()
		if portLow, portHigh, ok := ParsePortRange(v.portStr); ok {
			// parse ok, check the low and high value
			if v.portLow != -1 {
				if portLow != v.portLow {
					t.Errorf("%q expect port low %d, got %d\n", v.name, v.portLow, portLow)
				}
			}
			if v.portHigh != -1 {
				if portHigh != v.portHigh {
					t.Errorf("%q expect port hight %d, got %d\n", v.name, v.portHigh, portHigh)
				}
			}
		} else if !strings.Contains(place.String(), v.msg) {
			// parse failed, check the log message
			t.Errorf("%q expect \n%q\n got \n%q\n", v.name, v.msg, place.String())
		}
	}

	// restor the logFunc
	// logFunc = log.New(os.Stderr, "WARN: ", log.Ldate|log.Ltime|log.Lshortfile)
}

func TestConnection(t *testing.T) {
	tc := []struct {
		name   string
		ip     string
		port   string
		result bool
		msg    string
	}{
		{"localhost 8080", "localhost", "8080", true, ""},
		{"default range", "", "9081:9090", true, ""}, // error on macOS for ipv6
		{"invalid port", "", "4;3", false, "#parsePort invalid port number"},
		{"reverse port order", "", "4:3", false, "#ParsePortRange low port"},
		{"invalid host ", "dev.net", "403", false, "no such host"},
		{"invalid host literal", "192.158.", "403:405", false, "no such host"}, //"#tryBind error"},
	}

	// replace the logFunc
	var output strings.Builder
	// logFunc = log.New(&output, "#test", log.Ldate|log.Ltime|log.Lshortfile)
	defer util.Log.Restore()
	util.Log.SetOutput(&output)

	// test server connection creation
	for _, v := range tc {
		output.Reset()
		c := NewConnection(v.ip, v.port)

		if v.result {
			if c == nil {
				t.Errorf("%q got nil connection for %q:%q\n", v.name, v.ip, v.port)
			} else if len(c.socks) == 0 {
				t.Errorf("%q got empty connection for %q:%q\n", v.name, v.ip, v.port)
			} else {
				// t.Logf("%q close connection=%v\n", v.name, c.sock())
				c.sock().Close()
			}
		} else {
			if c != nil {
				t.Errorf("%q expect nil connection for %q:%q\n", v.name, v.ip, v.port)
				c.sock().Close()
			}
			got := output.String()
			if !strings.Contains(got, v.msg) {
				t.Errorf("%q expect \n%q, got \n%q\n", v.name, v.msg, got)
			}
		}
	}

	// restor the logFunc
	// logFunc = log.New(os.Stderr, "WARN: ", log.Ldate|log.Ltime|log.Lshortfile)
}

func TestConnectionClient(t *testing.T) {
	tc := []struct {
		name   string
		sIP    string // server ip
		sPort  string // server port
		cIP    string // client ip
		cPort  string // client port
		result bool
		msg    string
	}{
		{"localhost 8080", "localhost", "8080", "localhost", "8080", true, ""},
		{"wrong host", "", "9081:9090", "dev.net", "9081", false, "no such host"}, // error on macOS for ipv6
		{"wrong connect port", "localhost", "8080", "", "8001", true, ""},         // UDP does not connect, so different port still work.
	}

	var output strings.Builder
	defer util.Log.Restore()
	util.Log.SetOutput(&output)
	// test client connection
	for _, v := range tc {
		server := NewConnection(v.sIP, v.sPort)
		if server == nil {
			t.Errorf("%q server should not return nil.\n", v.name)
			continue
		}

		// replace the logFunc
		// logFunc = log.New(&output, "#test", log.Ldate|log.Ltime|log.Lshortfile)
		output.Reset()

		// open the client connection
		key := server.key
		client := NewConnectionClient(key.String(), v.cIP, v.cPort)

		// restor the logFunc
		// logFunc = log.New(os.Stderr, "WARN: ", log.Ldate|log.Ltime|log.Lshortfile)
		if v.result {
			if client == nil {
				t.Errorf("%q got nil connection, for %q:%q", v.name, v.cIP, v.cPort)
			} else if len(client.socks) == 0 {
				t.Errorf("%q got empty connection, for %q:%q", v.name, v.cIP, v.cPort)
			} else {
				// t.Logf("%q close connection=%v\n", v.name, client.sock())
				client.sock().Close()
				server.sock().Close()
			}
		} else {
			if client != nil {
				t.Errorf("%q expect nil connection for %q:%q\ngot %#v\n", v.name, v.cIP, v.cPort, client.sock())
				client.sock().Close()
				server.sock().Close()
			}
			got := output.String()
			if !strings.Contains(got, v.msg) {
				t.Errorf("%q expect \n%q, got \n%q\n", v.name, v.msg, got)
			}
		}
	}
}

func TestConnectionClientFail(t *testing.T) {
	title := "connection create client fail"
	ip := "localhost"
	port := "8081"

	// intercept log output
	var output strings.Builder
	defer util.Log.Restore()
	util.Log.SetOutput(&output)

	// create client
	wrongKey := "invalid key."
	NewConnectionClient(wrongKey, ip, port)

	// validate the result
	got := output.String()
	expect := "#NeNewConnectionClient build key failed"
	if !strings.Contains(got, expect) {
		t.Errorf("%q firt recv() expect \n%q, got \n%q\n", title, expect, got)
	}
}

func TestConnectionReadWrite(t *testing.T) {
	title := "connection read/write"
	ip := "localhost"
	port := "8080"

	message := []string{"a good news from udp client.", "天下风云出我辈，一入江湖岁月催。"}

	var wg sync.WaitGroup
	server := NewConnection(ip, port)

	var output strings.Builder
	// server.logW.SetOutput(&output)
	defer util.Log.Restore()
	util.Log.SetOutput(&output)

	if server == nil {
		t.Errorf("%q server should not return nil.\n", title)
		return
	}
	// fmt.Printf("#test server listen on =%s\n", server.sock().LocalAddr())

	key := server.key
	client := NewConnectionClient(key.String(), ip, port)
	// fmt.Printf("#test client=%s\n", client.sock().LocalAddr())

	for i := range message {
		sendErr := client.send(message[i], false)
		if sendErr != nil {
			t.Errorf("%q send error: %q\n", title, sendErr)
		}
	}
	defer client.sock().Close()

	wg.Add(1)
	go func() {
		defer server.sock().Close()
		for i := range message {
			output.Reset()
			// fmt.Printf("#test recv \n")
			payload, _, _ := server.Recv(1)
			// fmt.Printf("#test recv payload=%q\n", payload)
			if len(payload) == 0 || message[i] != payload {
				t.Errorf("%q expect %q, got %q\n", title, message[i], payload)
			} else {
				// t.Logf("%q expect %q, got %q\n", title, message[i], payload)
				if i == 0 {
					got := output.String()
					expect := "server now attached to client"
					if !strings.Contains(got, expect) {
						t.Errorf("%q firt recv() expect \n%q, got \n%q\n", title, expect, got)
					}
				}
			}
		}
		wg.Done()
	}()

	wg.Wait()
	// restor the logFunc
	// logFunc = log.New(os.Stderr, "WARN: ", log.Ldate|log.Ltime|log.Lshortfile)
}

func TestUDPReadWrite(t *testing.T) {
	title := "udp read/write"
	ip := "localhost"
	port := "8080"

	msg := []string{"a good news from udp client.", "天下风云出我辈，一入江湖岁月催。"}

	address := net.JoinHostPort(ip, port)
	addr, _ := net.ResolveUDPAddr(NETWORK, address)
	server, _ := net.ListenUDP("udp", addr)
	defer server.Close()

	client, _ := net.DialUDP("udp", nil, addr)
	defer client.Close()

	for i := range msg {
		_, err := client.Write([]byte(msg[i]))
		if err != nil {
			t.Fatal(err)
		}
		// fmt.Printf("%q client write size=%d, %q to %s\n", title, n, msg[i], client.RemoteAddr())
	}
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		for i := range msg {
			buf := make([]byte, 1000)
			n, addr, err := server.ReadFrom(buf)
			if err != nil {
				t.Errorf("%q server read #%d size=%d data %q from %s\n", title, i, n, buf[:n], addr)
			}
			// fmt.Printf("%q server read #%d size=%d data %q from %s\n", title, i, n, buf[:n], addr)
		}
		wg.Done()
	}()
	wg.Wait()
}

func TestSystemCallError(t *testing.T) {
	tc := []error{unix.EAGAIN, unix.EWOULDBLOCK, unix.EADDRNOTAVAIL}
	for i, v := range tc {

		e0 := v
		e1 := os.NewSyscallError("syscall", e0)
		e2 := net.OpError{Err: e1}

		switch i {
		case 0:
			if !errors.Is(&e2, unix.EAGAIN) {
				t.Errorf("#test e0=%v, e1=%v, e2=%v\n", e0, e1, e2)
			}
		case 1:
			if !errors.Is(&e2, unix.EWOULDBLOCK) {
				t.Errorf("#test e0=%v, e1=%v, e2=%v\n", e0, e1, e2)
			}
		case 2:
			if !errors.Is(&e2, unix.EADDRNOTAVAIL) {
				t.Errorf("#test e0=%v, e1=%v, e2=%v\n", e0, e1, e2)
			}
		}
	}

	err0 := errors.New("err0")
	err1 := errors.New("err0")

	if errors.Is(err0, err1) {
		t.Errorf("erros.Is() can't compare two errors. error 0=%q, error 1=%q, equal=%t\n", err0, err1, errors.Is(err0, err1))
	}
}

func TestHopPort(t *testing.T) {
	// prepare the client and server connection for the test
	title := "localhost hop port test case"
	ip := "localhost"
	port := "8080"

	server := NewConnection(ip, port)
	defer server.sock().Close()
	if server == nil {
		t.Errorf("%q server should not return nil.\n", title)
		return
	}
	// fmt.Printf("#test server listen on =%s\n", server.sock().LocalAddr())

	key := server.key
	client := NewConnectionClient(key.String(), ip, port)
	defer client.sock().Close()
	if client == nil {
		t.Errorf("%q client should not return nil.\n", title)
	}

	tc := []struct {
		name    string
		start   int  // startpoint of socket number
		maxAge  bool // exceed the max old socket age?
		maxOpen bool // exceed the max open ports
		remains int  // the remains socket
	}{
		{"over max age", 9, true, false, 1},
		{"over max open", 9, false, true, 1},
	}

	// test hopPort
	for _, v := range tc {
		// prepare the sockets
		for i := 0; i < v.start; i++ {
			time.Sleep(time.Millisecond * 5)
			client.hopPort()
		}
		// fmt.Printf("%q starts with %d sockets.\n", v.name, len(client.socks))

		if v.maxAge {
			client.lastPortChoice -= MAX_OLD_SOCKET_AGE + 10
			// hopPort will reset the lastPortChoice, so we call pruneSockets directly.
			client.pruneSockets()
		} else if v.maxOpen {
			// add more sockets
			for i := 0; i < MAX_PORTS_OPEN-v.start; i++ {
				time.Sleep(time.Millisecond * 5)
				client.hopPort()
			}
			// fmt.Printf("%q maxOpen with %d sockets.\n", v.name, len(client.socks))
		}

		if len(client.socks) != v.remains {
			t.Errorf("%q expect %d socket, got %d\n", v.name, v.remains, len(client.socks))
		}
	}

	// intercept client log
	var output strings.Builder
	// client.logW.SetOutput(&output)
	defer util.Log.Restore()
	util.Log.SetOutput(&output)

	// fake wrong remote address
	client.remoteAddr = &net.UDPAddr{Port: -80}
	client.hopPort()

	// validate the error handling
	got := output.String()
	expect := "#hopPort failed to dial"
	// fmt.Printf("#test got=%s\n", got)
	if !strings.Contains(got, expect) {
		t.Errorf("#test hopPort() expect \n%q, got \n%q\n", expect, got)
	}

	// restor the logFunc
	// logFunc = log.New(os.Stderr, "WARN: ", log.Ldate|log.Ltime|log.Lshortfile)
}

func TestTryBindFail(t *testing.T) {
	// occupy the following ports
	ports := []int{8000, 8001, 8002, 8003}
	for i := range ports {
		srv := NewConnection("", fmt.Sprintf("%d", ports[i]))
		defer srv.sock().Close()
	}

	var output strings.Builder
	defer util.Log.Restore()
	util.Log.SetOutput(&output)
	// logFunc = log.New(&output, "#test", log.Ldate|log.Ltime|log.Lshortfile)

	s := NewConnection("", "8000:8003")

	expect := "#tryBind listen"
	got := output.String()
	if s != nil || !strings.Contains(got, expect) {
		t.Errorf("#test tryBind() expect \n%q got \n%s\n", expect, got)
	}

	// restor the logFunc
	// logFunc = log.New(os.Stderr, "WARN: ", log.Ldate|log.Ltime|log.Lshortfile)
}

func TestMarkECNFail(t *testing.T) {
	tc := []struct {
		name string
		fd   int
		err  error
	}{
		{"1st error", 0, errors.New("mock error")},
		{"2nd error", 1, errors.New("mock 1st set error")},
		{"3rd error", 2, errors.New("mock 2nd set error")},
	}

	mockGetOpt := func(fd, level, opt int) (value int, err error) {
		if fd == 0 {
			return 0, tc[0].err
		}
		return
	}

	times := 0
	mockSetOpt := func(fd, level, opt int, value int) (err error) {
		switch fd {
		case 1:
			return tc[1].err
		case 2:
			if times == 0 { // return normal at 1st round invocation
				times++
			} else {
				times = 0 // return error at 2nd round invocation
				return tc[2].err
			}
		}
		return
	}

	for i := range tc {
		got := markECN(i, mockGetOpt, mockSetOpt)
		if got != tc[i].err {
			t.Errorf("#test expect error handling #%d for set socket options return %s, got %s\n", i, tc[i].err, got)
		}
	}
}

func TestNewPacket(t *testing.T) {
	// prepare the client and server connection for the test
	title := "newPacket recent received test case"
	ip := "localhost"
	port := "8080"

	server := NewConnection(ip, port)
	defer server.sock().Close()
	if server == nil {
		t.Errorf("%q server should not return nil.\n", title)
		return
	}

	// branch the condition
	server.savedTimestampReceivedAt = time.Now().UnixMilli() - 100
	pkt := server.newPacket(title)

	// validate the result
	if string(pkt.payload) != title {
		t.Errorf("%q expect payload=%q, got %q\n", title, title, pkt.payload)
	}
	if pkt.timestampReply == 0 {
		t.Errorf("%q expect timestampReply not zero, got %d\n", title, pkt.timestampReply)
	}
	if server.savedTimestampReceivedAt != 0 {
		t.Errorf("%q expect savedTimestampReceivedAt=%d, got %d\n", title, 0, server.savedTimestampReceivedAt)
	}
}

func TestSetMTU(t *testing.T) {
	// prepare the connection
	title := "#test setMTU"
	ip := "localhost"
	port := "8080"

	conn := NewConnection(ip, port)
	defer conn.sock().Close()
	if conn == nil {
		t.Errorf("%q server should not return nil.\n", title)
		return
	}

	// build ipv6 address
	udp6Addr, err := net.ResolveUDPAddr("udp6", "[::]:8080")
	if err != nil {
		t.Errorf("%q expect udp6 address, got nil", title)
	}
	conn.setMTU(udp6Addr)

	// validate the MTU
	expect := DEFAULT_IPV6_MTU - IPV6_HEADER_LEN
	if conn.mtu != expect {
		t.Errorf("%q expect MTU=%d, got %d\n", title, expect, conn.mtu)
	}
}

func TestTimestamp16(t *testing.T) {
	tc := []struct {
		name    string
		t0Delta int16
		diff    uint16
	}{
		{"t1 > t0", 500, 500},
		{"t1 = t0", 0, 0},
		{"t1 < t0", -300, 65536 - 300},
	}

	for _, v := range tc {
		t0 := timestamp16()
		t1 := int16(t0) + v.t0Delta
		got := timestampDiff(uint16(t1), t0)
		if v.diff != got {
			t.Errorf("%q expect %d, got %d\n", v.name, v.diff, got)
		}
	}
}

func TestSendFail(t *testing.T) {
	// prepare the client and server connection for the test
	title := "localhost send test case"
	ip := "localhost"
	port := "8080"

	server := NewConnection(ip, port)
	defer server.sock().Close()
	if server == nil {
		t.Errorf("%q server should not return nil.\n", title)
		return
	}
	// fmt.Printf("#test server listen on =%s\n", server.sock().LocalAddr())
	key := server.key
	client := NewConnectionClient(key.String(), ip, port)
	defer client.sock().Close()
	if client == nil {
		t.Errorf("%q client should not return nil.\n", title)
	}

	// test case
	tc := []struct {
		name          string
		hasRemoteAddr bool
		writeErr      error
		byteSend      int
	}{
		{"send without remote address", false, nil, 0},
		{"write return err", true, errors.New("#send write"), 0},
		{"write return wrong size", true, nil, 23},
	}

	// validate the failure case with mockUdpConn
	for _, v := range tc {
		if !v.hasRemoteAddr {
			client.hasRemoteAddr = false
			client.send(v.name, false)
			// there is no aciton, no error, so no validation
		} else if v.writeErr != nil {
			client.hasRemoteAddr = true
			var mock mockUdpConn
			client.socks = append(client.socks, &mock)
			err := client.send(v.name, false)
			if !strings.Contains(err.Error(), v.writeErr.Error()) {
				t.Errorf("%q expect %q, got %q\n", v.name, v.writeErr, err)
			}
		} else if v.byteSend != 0 {
			var mock mockUdpConn
			client.socks = append(client.socks, &mock)
			err := client.send(v.name, false)
			expect := "doesn't match expected data"
			if !strings.Contains(err.Error(), expect) {
				t.Errorf("%q expect %q, got %q\n", v.name, expect, err)
			}
		}
	}
}

type mockUdpConn struct {
	round int
	data  string
}

func (mc *mockUdpConn) LocalAddr() net.Addr {
	return nil
}

func (mc *mockUdpConn) RemoteAddr() net.Addr {
	return nil
}

func (c *mockUdpConn) Write(b []byte) (int, error) {
	if len(b) == 48 {
		return 0, errors.New("mock by len = 48.")
	}
	return 5, nil
}

func (mc *mockUdpConn) WriteMsgUDP(b, oob []byte, addr *net.UDPAddr) (n, oobn int, err error) {
	if len(b) == 48 {
		return 0, 0, errors.New("mock by len = 48.")
	}
	return 5, 0, nil
}

// func (mc *mockUdpConn) Write(b []byte) (int, error) {
// 	// fmt.Printf("#Write mockUdpConn len=%d\n", len(b))
// 	if len(b) == 48 {
// 		return 0, errors.New("mock by len = 48.")
// 	}
// 	return 5, nil
// }

func (mc *mockUdpConn) ReadMsgUDP(b, oob []byte) (n, oobn, flags int, addr *net.UDPAddr, err error) {
	// fmt.Printf("#mockUdpConn ReadMsgUDP() round=%d\n", mc.round)
	switch mc.round {
	case 0:
		e1 := os.NewSyscallError("syscall", unix.EWOULDBLOCK)
		err = &net.OpError{Err: e1}
	case 1:
		n = -1
	case 2:
		flags = unix.MSG_TRUNC
	case 3:
		oobStr := "malform oob message"
		copy(oob, []byte(oobStr))
		oobn = len(oobStr)
		// fmt.Printf("#mockUdpConn ReadMsgUDP() oob=%q, oobn=%d\n", oob, oobn)
	case 4:
		dataStr := "malform data: mocked by mockUdpConn."
		if mc.data != "" {
			dataStr = mc.data
		}
		copy(b, []byte(dataStr))
		n = len(dataStr)
	case 5:
		err = os.ErrDeadlineExceeded
		// fmt.Printf("#mockUdpConn ReadMsgUDP() dataStr=%q, data=%q, round=%d\n", dataStr, mc.data, mc.round)
	}

	mc.round++
	return
}

func (mc *mockUdpConn) Close() error {
	return nil
}

func (mc *mockUdpConn) WriteTo(b []byte, addr net.Addr) (len int, err error) {
	return
}

func (mc *mockUdpConn) SetReadDeadline(t time.Time) error {
	if mc.round == 5 {
		return errors.New("set read dead line error")
	}
	return nil
}

func (mc *mockUdpConn) Read(b []byte) (n int, err error) {
	return 0, nil
}

func (mc *mockUdpConn) SetDeadline(t time.Time) error {
	return nil
}

func (mc *mockUdpConn) SetWriteDeadline(t time.Time) error {
	return nil
}

func TestSendBranch(t *testing.T) {
	// prepare the client and server connection for the test
	title := "detect server detached from client"
	ip := "localhost"
	port := "8080"

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

	// client send a message to server, server receive it.
	// this will initialize server data.
	client.send(title, false)

	// we need the delay to receive the packet on server side.
	time.Sleep(time.Millisecond * 20)

	// intercept server log
	var output strings.Builder
	// server.logW.SetOutput(&output)
	defer util.Log.Restore()
	util.Log.SetOutput(&output)

	msg, _, _ := server.Recv(1)
	if msg != title {
		t.Errorf("%q client send\n%q to server, server got \n%q\n", title, title, msg)
	}

	// fake the lastHeard to meet the detach condition
	server.lastHeard = time.Now().UnixMilli() - SERVER_ASSOCIATION_TIMEOUT - 10

	err := server.send(title, false)

	// validate the send server branch
	gotLog := output.String()
	expectLog := "#send server now detached from client"
	if err != nil {
		t.Errorf("%q should return nil, got %s\n", title, err)
	} else if server.hasRemoteAddr {
		t.Errorf("%q expect hasRemoteAddr %t, got %t\n", title, false, server.hasRemoteAddr)
	} else if !strings.Contains(gotLog, expectLog) {
		t.Errorf("%q expect log \n%q, got \n%q\n", title, expectLog, gotLog)
	}

	time.Sleep(time.Millisecond * 20)
	msg, _, _ = client.Recv(1) // the msg is still the old title
	if msg != title {
		t.Errorf("%q client receive\n%q from server, client got \n%q\n", title, title, msg)
	}

	msg = "client hopPort branch"
	// set client hopPort condition
	client.lastPortChoice = time.Now().UnixMilli() - PORT_HOP_INTERVAL - 2
	client.setLastRoundtripSuccess(time.Now().UnixMilli() - PORT_HOP_INTERVAL - 2)

	client.send(msg, false)
	// hopPort will add a new socket to the list.
	if len(client.socks) != 2 {
		t.Errorf("%q expect %d socket, got %d\n", msg, 2, len(client.socks))
	}

	// restor the logFunc
	// logFunc = log.New(os.Stderr, "WARN: ", log.Ldate|log.Ltime|log.Lshortfile)
}

func TestRecvFail(t *testing.T) {
	// prepare the client and server connection for the test
	title := "localhost send test case"
	ip := "localhost"
	port := "8080"

	server := NewConnection(ip, port)
	defer server.sock().Close()
	if server == nil {
		t.Errorf("%q server should not return nil.\n", title)
		return
	}
	// fmt.Printf("#test server listen on =%s\n", server.sock().LocalAddr())
	key := server.key
	client := NewConnectionClient(key.String(), ip, port)
	defer client.sock().Close()
	if client == nil {
		t.Errorf("%q client should not return nil.\n", title)
	}

	// test case
	tc := []struct {
		name  string
		round int
		err   error
	}{
		{"receive return EWOULDBLOCK err", 0, unix.EWOULDBLOCK},
		{"receive return n<0 err", 1, ErrRecvLength},
		{"receive return MSG_TRUNC err", 2, ErrRecvOversize},
		{"receive return parse control message erro", 3, errors.New("invalid argument")},
		{"receive return parser decrypt error", 4, errors.New("cipher: message authentication failed")},
	}

	// intercept server log
	var output strings.Builder
	// client.logW.SetOutput(&output)
	defer util.Log.Restore()
	util.Log.SetOutput(&output)

	// let the mock connection as the only connection
	var mock mockUdpConn
	client.socks = append(client.socks, &mock)
	client.socks = client.socks[len(client.socks)-1:]

	// validate the failure case with mockUdpConn
	for i, v := range tc {
		_, _, err := client.Recv(1)
		if v.err != nil {
			switch i {
			case 0, 1, 2:
				if !errors.Is(err, v.err) {
					t.Errorf("%q expect err=%q, got %q\n", v.name, v.err, err)
				}
			default:
				// t.Logf("%q err != v.err = %t\n", v.name, err != v.err)
				if err.Error() != v.err.Error() {
					t.Errorf("%q expect err=%q, got %q\n", v.name, v.err, err)
				}
			}
		}
	}
	// restor the logFunc
	// logFunc = log.New(os.Stderr, "WARN: ", log.Ldate|log.Ltime|log.Lshortfile)
}

func TestRecvBranchServer(t *testing.T) {
	// prepare the client and server connection for the test
	title := "server recive branch"
	ip := "localhost"
	port := "8080"

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
	// this will initialize server data.
	client.send(msg0, false)

	// we need the delay to receive the packet on server side.
	time.Sleep(time.Millisecond * 20)

	// mock the connection: guarantee to return the msg0
	var mock mockUdpConn
	mock.round = 4
	mock.data = msg0
	server.socks = append(server.socks, &mock)
	server.socks = server.socks[len(server.socks)-1:]

	// mock the session
	server.session = &mockSession{direction: TO_CLIENT, seq: 7}

	// intercept server log
	var output strings.Builder
	// server.logW.SetOutput(&output)
	defer util.Log.Restore()
	util.Log.SetOutput(&output)

	// perform the receive
	_, _, err := server.Recv(1)
	if !errors.Is(err, ErrRecvDirection) {
		t.Errorf("%q client send\n%q to server, server got \n%q\n", title, msg0, err)
	}

	// restor the logFunc
	// logFunc = log.New(os.Stderr, "WARN: ", log.Ldate|log.Ltime|log.Lshortfile)
}

func TestRecvBranchClient(t *testing.T) {
	// prepare the client and server connection for the test
	title := "client recive branch"
	ip := "localhost"
	port := "8080"

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

	// intercept server log
	var output strings.Builder
	// server.logW.SetOutput(&output)
	defer util.Log.Restore()
	util.Log.SetOutput(&output)

	msg0 := "from client to server"
	// client send a message to server, server receive it.
	// this will initialize server remote address.
	client.send(msg0, false)
	time.Sleep(time.Millisecond * 20)
	server.Recv(1)

	// server send message to client
	msg1 := "from server to client"
	server.send(msg1, false)
	time.Sleep(time.Millisecond * 20)

	// mock the connection: guarantee to return the msg0
	var mock mockUdpConn
	mock.round = 4
	mock.data = msg1
	client.socks = append(client.socks, &mock)
	client.socks = client.socks[len(client.socks)-1:]

	// mock the session
	client.session = &mockSession{direction: TO_SERVER, seq: 7}

	// perform the receive
	_, _, err := client.Recv(1)
	if !errors.Is(err, ErrRecvDirection) {
		t.Errorf("%q client send\n%q to server, server got \n%q\n", title, msg1, err)
	}
}

type mockSession struct {
	direction Direction
	seq       uint64
}

func (m *mockSession) Decrypt(text []byte) (*encrypt.Message, error) {
	var seqNonce uint64

	// logic form Packet.toMessage()
	if m.direction == TO_CLIENT {
		// client is 1, server is 0
		seqNonce = DIRECTION_MASK | (m.seq & SEQUENCE_MASK)
	} else {
		seqNonce = m.seq & SEQUENCE_MASK
	}

	msg := encrypt.NewMessage(seqNonce, text)
	// fmt.Println("#mockSession Decrypt()")
	return msg, nil
}

func (m *mockSession) Encrypt(plainText *encrypt.Message) []byte {
	return nil
}

func TestRecvCongestionPacket(t *testing.T) {
	// prepare the client and server connection
	title := "receive congestion packet branch"
	ip := "localhost"
	port := "8080"

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

	// intercept server log
	var output strings.Builder
	// server.logW.SetOutput(&output)
	defer util.Log.Restore()
	util.Log.SetOutput(&output)

	// save old congestionFunc
	oldCF := congestionFunc
	// mock the congestion case
	congestionFunc = func(in byte) bool {
		return true
	}
	server.Recv(1)

	// validate the result
	expect := "#recvOne received explicit congestion notification"
	got := output.String()
	if !strings.Contains(got, expect) {
		t.Errorf("%q expect \n%q, got \n%q\n", title, expect, got)
	}

	// if server.savedTimestamp <= 0 {
	// 	t.Errorf("%q savedTimestamp should be greater than zero, it's %d\n", title, server.savedTimestamp)
	// }

	// restor the logFunc
	// logFunc = log.New(os.Stderr, "WARN: ", log.Ldate|log.Ltime|log.Lshortfile)
	congestionFunc = oldCF
}

func TestRecvSRTT(t *testing.T) {
	// prepare the client and server connection
	title := "receive packet to calculate SRTT/RTTVAR"
	ip := "localhost"
	port := "8080"

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

	// client send a message to server, server receive it.
	// this will initialize server remote address.

	// messages := []struct {
	// 	toServer string
	// 	toClient string
	// }{
	// 	{"1st round from client to server", "1st round from server to client"},
	// 	{"2nd round from client to server", "2nd round from server to client"},
	// 	{"3rd round from client to server", "3rd round from server to client"},
	// 	{"4th round from client to server", "4th round from server to client"},
	// }

	// intercept server log
	var output strings.Builder
	// server.logW.SetOutput(&output)
	defer util.Log.Restore()
	util.Log.SetOutput(&output)

	var got string
	maxMsg := 10

	// for i, msg := range messages {
	for i := 0; i < maxMsg; i++ {
		toServer := fmt.Sprintf("%d round from client to server", i)
		toClient := fmt.Sprintf("%d round from server to client", i)

		client.send(toServer, false)
		time.Sleep(time.Millisecond * 10)
		server.Recv(1)

		server.send(toClient, false)
		time.Sleep(time.Millisecond * 10)
		got, _, _ = client.Recv(1)

		if got != toClient {
			t.Errorf("%q expect %q, got %q\n", title, toClient, got)
		}
		// fmt.Printf("%q %d RTTHit=%t SRTT=%f, RTTVAR=%f\n", title, i, client.RTTHit, client.SRTT, client.RTTVAR)
	}
	if client.SRTT < 20 || client.SRTT > 25 {
		t.Errorf("%q RTTHit=%t SRTT=%f, RTTVAR=%f\n", title, client.RTTHit, client.SRTT, client.RTTVAR)
	}

	gotRTO := client.timeout()
	if gotRTO != MIN_RTO {
		t.Errorf("%q expect timeout %d, got %d\n", title, MIN_RTO, gotRTO)
	}

	// restor the logFunc
	// logFunc = log.New(os.Stderr, "WARN: ", log.Ldate|log.Ltime|log.Lshortfile)
}

func TestServerRecvTimeout(t *testing.T) {
	// prepare the client and server connection
	title := "receive packet to calculate SRTT/RTTVAR"
	ip := "localhost"
	port := "8080"

	server := NewConnection(ip, port)
	defer server.sock().Close()
	if server == nil {
		t.Errorf("%q server should not return nil.\n", title)
		return
	}

	// let the mock connection as the only connection
	var mock mockUdpConn
	server.socks = append(server.socks, &mock)
	server.socks = server.socks[len(server.socks)-1:]

	// validate the no error case
	if e1 := server.sock().SetReadDeadline(time.Now().Add(time.Millisecond * time.Duration(5))); e1 != nil {
		t.Errorf("#test setReadDeadline() expect nil, got %s\n", e1)
	}

	// prepare mock conditions
	mock.round = 5
	mock.data = ""

	// validate the error case
	if e1 := server.sock().SetReadDeadline(time.Now().Add(time.Millisecond * time.Duration(5))); e1 == nil {
		t.Errorf("#test setReadDeadline() expect err, got nil\n") // see mockUdpConn.SetReadDeadline
	}
	// validate the read time out case
	_, _, e2 := server.Recv(1)
	if !errors.Is(e2, os.ErrDeadlineExceeded) {
		t.Errorf("#test recv() expect err, got %s\n", e2) // see mockUdpConn.SetReadDeadline
	}
}

func TestConnectionRecvSeqFail(t *testing.T) {
	title := "connection recv seq fail"
	ip := "localhost"
	port := "60800"

	message := []string{"first message."}

	var wg sync.WaitGroup
	server := NewConnection(ip, port)

	defer util.Log.Restore()
	var output strings.Builder
	util.Log.SetLevel(slog.LevelDebug)
	// util.Log.SetOutput(os.Stdout)
	util.Log.SetOutput(&output)

	if server == nil {
		t.Errorf("%q server should not return nil.\n", title)
		return
	}

	key := server.key
	client := NewConnectionClient(key.String(), ip, port)

	for i := range message {
		sendErr := client.send(message[i], false)
		if sendErr != nil {
			t.Errorf("%q send error: %q\n", title, sendErr)
		}
	}
	defer client.Close()

	wg.Add(1)
	go func() {
		defer server.Close()
		for i := range message {
			// create the error condition
			server.expectedReceiverSeq = 75

			payload, _, _ := server.Recv(1)
			// fmt.Printf("#test recv i=%d payload=%q, expectedReceiverSeq=%d\n",
			// 	i, payload, server.expectedReceiverSeq)
			if message[i] != payload {
				t.Errorf("%q expect %q, got %q\n", title, message[i], payload)
			}

			got := output.String()
			expect := "received explicit out-of-order packets"
			if !strings.Contains(got, expect) {
				t.Errorf("%q expect \n%s, got \n%s\n", title, expect, got)
			}
		}
		wg.Done()
	}()

	wg.Wait()
}
