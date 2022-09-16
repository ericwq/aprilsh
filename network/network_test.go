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
	"log"
	"net"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ericwq/aprilsh/encrypt"
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
		{"outof scope number low", "-1:536", -1, -1, "-1 outside valid range [0..65535]"},
		{"outof scope number high", "0:65536", -1, -1, "65536 outside valid range [0..65535]"},
		{"invalid number", "3a", -1, -1, "invalid (solo) port number"},
		{"port order reverse", "3:1", -1, -1, "greater than high port"},
		{"solo port", "5", 5, 5, ""},
	}

	var place strings.Builder
	output := log.New(&place, "WARN: ", log.Ldate|log.Ltime|log.Lshortfile)
	// output := log.New(os.Stderr, "WARN: ", log.Ldate|log.Ltime|log.Lshortfile)

	for _, v := range tc {
		place.Reset()
		if portLow, portHigh, ok := ParsePortRange(v.portStr, output); ok {
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
		{"invalid port", "", "4;3", false, ""},
		{"reverse port order", "", "4:3", false, ""},
		{"invalid host ", "dev.net", "403", false, ""},
		{"invalid host literal", "192.158.", "403:405", false, ""},
	}

	// test server connection creation
	for _, v := range tc {
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
		}
	}
}

func TestConnectionClient(t *testing.T) {
	tc := []struct {
		name   string
		sIP    string // server ip
		sPort  string // server port
		cIP    string // client ip
		cPort  string // client port
		result bool
	}{
		{"localhost 8080", "localhost", "8080", "localhost", "8080", true},
		{"wrong host", "", "9081:9090", "dev.net", "9081", false},     // error on macOS for ipv6
		{"wrong connect port", "localhost", "8080", "", "8001", true}, // UDP does not connect, so different port still work.
	}

	// test client connection
	for _, v := range tc {
		server := NewConnection(v.sIP, v.sPort)
		if server == nil {
			t.Errorf("%q server should not return nil.\n", v.name)
			continue
		}
		key := server.key
		client := NewConnectionClient(key.String(), v.cIP, v.cPort)

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
		}
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
	server.logW.SetOutput(&output)

	if server == nil {
		t.Errorf("%q server should not return nil.\n", title)
		return
	}
	// fmt.Printf("#test server listen on =%s\n", server.sock().LocalAddr())

	key := server.key
	client := NewConnectionClient(key.String(), ip, port)
	// fmt.Printf("#test client=%s\n", client.sock().LocalAddr())

	for i := range message {
		client.send(message[i])
		if client.sendError != "" {
			t.Errorf("%q send error: %q\n", title, client.sendError)
		}
	}
	defer client.sock().Close()

	wg.Add(1)
	go func() {
		defer server.sock().Close()
		for i := range message {
			output.Reset()
			payload := server.recv()
			// fmt.Printf("#test recv payload=%q\n", payload)
			if len(payload) == 0 || message[i] != payload {
				t.Errorf("%q expect %q, got %q\n", title, message[i], payload)
			} else {
				// t.Logf("%q expect %q, got %q\n", title, message[i], payload)
				if i == 0 {
					got := output.String()
					expect := "#recvOne server now attached to client at"
					if !strings.Contains(got, expect) {
						t.Errorf("%q firt recv() expect \n%q, got \n%q\n", title, expect, got)
					}
				}
			}
		}
		wg.Done()
	}()

	wg.Wait()
}

// setup EC bit for specified connection
func setupConnectionEC(u *net.UDPConn) error {
	sc, err := u.SyscallConn()
	if err != nil {
		return err
	}
	var serr error
	err = sc.Control(func(fd uintptr) {
		serr = markECN(int(fd))
	})
	if err != nil {
		return err
	}
	return serr
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
	client.logW.SetOutput(&output)

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
}

func TestTryBindFail(t *testing.T) {
	// occupy the following ports
	ports := []int{8000, 8001, 8002, 8003}
	for i := range ports {
		srv := NewConnection("", fmt.Sprintf("%d", ports[i]))
		defer srv.sock().Close()
	}

	var output strings.Builder
	oldLog := logger
	logger = log.New(&output, "#test", log.Ldate|log.Ltime|log.Lshortfile)
	defer func() {
		logger = oldLog
	}()
	s := NewConnection("", "8000:8003")

	expect := "#tryBind error"
	got := output.String()
	if s != nil || !strings.Contains(got, expect) {
		t.Errorf("#test tryBind() expect \n%q got \n%s\n", expect, got)
	}
}
