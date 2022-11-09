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
		label  string
		pause  int
		expect int
	}{
		{"quick response", 55, 17},
		{"slow response", 70, 0},
	}

	for _, v := range tc {

		connection := NewConnection("localhost", "8080")
		initialState, _ := statesync.NewComplete(80, 40, 0)

		ts := NewTransportSender(connection, initialState)

		for i := 0; i < 33; i++ {
			s, _ := statesync.NewComplete(80, 40, 0)
			now := time.Now().UnixMilli()
			time.Sleep(time.Millisecond * time.Duration(v.pause))
			ts.addSendState(now, int64(i+2), s)
		}

		ts.updateAssumedReceiverState()

		if ts.assumedReceiverState != &ts.sentStates[v.expect] {
			t.Errorf("%q expect %p, got %p\n", v.label, &ts.sentStates[v.expect], ts.assumedReceiverState)
		}

		connection.sock().Close()
	}
}
