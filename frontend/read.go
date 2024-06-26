// Copyright 2022~2024 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package frontend

import (
	"errors"
	"io"
	"net"
	"os"
	"sync/atomic"
	"time"
)

// communication the read result with the others
type Message struct {
	RAddr net.Addr // if the message is from network, it's the remote address
	Err   error    // error if it happens
	Data  string   // payload data
}

// for easy mock
type Connection interface {
	Recv(timeout int) (payload string, rAddr net.Addr, err error)
}

// Read from the reader, set read time out for every read. The read result will be sent
// to caller via msgChan, including error info if available. doneChan channel is used to stop
// the file reader.
//
// Note the caller should consume the last read message when shutdown.
func ReadFromFile(timeout int, msgChan chan Message, doneChan chan any, fReader io.Reader) {
	var buf [16384]byte
	var err error
	var bytesRead int
	// var reading bool
	var reading int64

	for {
		timer := time.NewTimer(time.Duration(timeout) * time.Millisecond)
		// if !reading {
		if atomic.LoadInt64(&reading) == 0 {
			go func(fr io.Reader, buf []byte, ch chan Message) {
				atomic.StoreInt64(&reading, 1)
				// reading = true
				// util.Log.Debug("#read","action", "satrt")
				bytesRead, err = fr.Read(buf)
				if bytesRead > 0 {
					ch <- Message{nil, nil, string(buf[:bytesRead])}
					atomic.StoreInt64(&reading, 0)
					// reading = false
					// util.Log.Debug("#read","action", "got")
				} else {
					ch <- Message{nil, err, ""}
					// util.Log.Debug("#read","error", "EOF")
				}
			}(fReader, buf[:], msgChan)
		}

		// waiting for time out or get the shutdown message
		select {
		case <-doneChan:
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}

// Read from the network, set read time out before every read. The read result will be sent
// to caller via msgChan, including error info if available. doneChan channel is used to stop
// the network receiver.
//
// Note the caller should consume the last read message when shutdown.
// connection closed error will stop the function.
func ReadFromNetwork(timeout int, msgChan chan Message, doneChan chan any, connection Connection) {
	var err error
	var payload string
	var rAddr net.Addr

	for {
		// packet received from remote
		payload, rAddr, err = connection.Recv(timeout)
		if err != nil {
			if errors.Is(err, os.ErrDeadlineExceeded) {
				// read timeout
			} else {
				if errors.Is(err, net.ErrClosed) { // connection is closed
					// util.Log.Debug("#ReadFromNetwork","error", err)
					return
				}
				// in case of other error, notify the caller and continue.
				msgChan <- Message{nil, err, ""}
			}
		} else {
			// normal read
			msgChan <- Message{rAddr, nil, payload}
		}

		// waiting for time out or get the shutdown message
		// 2 times timeout is a experience value from trace info
		timer := time.NewTimer(time.Duration(timeout*2) * time.Millisecond)
		select {
		case <-doneChan:
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}
