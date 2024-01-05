// Copyright 2022~2023 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package frontend

import (
	"errors"
	"io"
	"net"
	"os"
	"time"
)

// communication the read result with the others
type Message struct {
	Data  string   // payload data
	RAddr net.Addr // if the message is from network, it's the remote address
	Err   error    // error if it happens
}

// for easy mock
type DeadLiner interface {
	SetReadDeadline(t time.Time) error
}

// for easy mock
type DeadLineReader interface {
	io.Reader
	DeadLiner
}

// for easy mock
type DeadLineReceiver interface {
	Recv(timeout int) (payload string, rAddr net.Addr, err error)
	// DeadLiner
}

// Read from the file reader, set read time out before every read. The read result will be sent
// to caller via msgChan, including error info if available. doneChan channel is used to stop
// the file reader.
//
// Note the caller must consume the last read message after it send the shutdown message. EOF
// can also stop the file reader.
func ReadFromFile(timeout int, msgChan chan Message, doneChan chan any, fReader DeadLineReader) {
	// var buf [16384]byte
	// var err error
	// var bytesRead int
	reading := false

	for {
		// // fmt.Println("#ReadFromFile wait for shutdown message.")
		// select {
		// case <-doneChan:
		// 	// fmt.Println("#ReadFromFile got shutdown message.")
		// 	return
		// default:
		// }
		// util.Log.With("action", "satrt").Debug("#read")
		// // set read time out
		// fReader.SetReadDeadline(time.Now().Add(time.Millisecond * time.Duration(timeout)))
		//
		// // fill buffer if possible
		// bytesRead, err = fReader.Read(buf[:])
		//
		// if bytesRead > 0 {
		// 	util.Log.With("action", "got").Debug("#read")
		// 	msgChan <- Message{string(buf[:bytesRead]), nil, nil}
		// } else if errors.Is(err, os.ErrDeadlineExceeded) {
		// 	// timeout
		// 	// msgChan <- Message{err, ""}
		// 	util.Log.With("error", err).Debug("#read")
		// 	continue
		// } else {
		// 	// EOF goes here
		// 	msgChan <- Message{"", nil, err}
		// 	break
		// }

		timer := time.NewTimer(time.Duration(timeout) * time.Millisecond)
		if !reading {
			reading = true
			go func(pr DeadLineReader, ch chan Message) {
				buf := make([]byte, 16384)

				bytesRead, err := pr.Read(buf)
				if err != nil && errors.Is(err, io.EOF) {
					ch <- Message{string(buf[:bytesRead]), nil, nil}
					reading = false
				} else {
					ch <- Message{"", nil, err}
				}
			}(fReader, msgChan)
		}

		// waiting for time out or get the pipe reader send message
		select {
		case <-doneChan:
			timer.Stop()
			return
		case <-timer.C:
		}
	}
	// fmt.Println("#ReadFromFile exit.")
}

// Read from the network, set read time out before every read. The read result will be sent
// to caller via msgChan, including error info if available. doneChan channel is used to stop
// the network receiver.
//
// Note the caller must consume the last read message after it send the shutdown message.
// network read error can also stop the receiver.
func ReadFromNetwork(timeout int, msgChan chan Message, doneChan chan any, network DeadLineReceiver) {
	var err error
	var payload string
	var rAddr net.Addr

	for {
		select {
		case <-doneChan:
			return
		default:
		}
		// packet received from the network
		payload, rAddr, err = network.Recv(timeout)
		if err != nil {
			if errors.Is(err, os.ErrDeadlineExceeded) {
				// read timeout
				continue
			} else {
				// EOF goes here, in case of error retry it.
				msgChan <- Message{"", nil, err}
				continue
			}
		} else {
			// normal read
			msgChan <- Message{payload, rAddr, nil}
		}
	}
}
