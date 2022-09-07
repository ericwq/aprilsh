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
	"log"
	"reflect"
	"strings"
	"testing"

	"github.com/ericwq/aprilsh/encrypt"
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
		p := NewPacket2(*m1)

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
		{"invalid number", "3a", -1, -1, "Invalid (solo) port number"},
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
		name string
		ip   string
		port string
	}{
		{"localhost 8080", "localhost", "8080"},
		{"default range", "", "9081:9090"},
	}

	for _, v := range tc {
		c := NewConnection(v.ip, v.port)
		if c == nil {
			// fmt.Printf("#test got nil connection for %s:\n%s\n", v.ip, v.port)
		} else {
			// fmt.Printf("#test got right connection for %s:%s\n\tconn=%v\n", v.ip, v.port, c.sock())
			c.sock().Close()
		}
	}
}
