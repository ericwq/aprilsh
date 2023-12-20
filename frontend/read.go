// Copyright 2022~2023 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package frontend

import (
	"errors"
	"io"
	"os"
	"time"
)

// communication the read result with the others
type Message struct {
	Err  error
	Data string
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
	// Recv(timeout int) (payload string, err error)
	Recv() (payload string, err error)
	DeadLiner
}

// Read from the file reader, set read time out before every read. The read result will be sent
// to caller via msgChan, including error info if available. doneChan channel is used to stop
// the file reader.
//
// Note the caller must consume the last read message after it send the shutdown message. EOF
// can also stop the file reader.
func ReadFromFile(timeout int, msgChan chan Message, doneChan chan any, fReader DeadLineReader) {
	var buf [16384]byte
	var err error
	var bytesRead int

	for {
		// fmt.Println("#ReadFromFile wait for shutdown message.")
		select {
		case <-doneChan:
			// fmt.Println("#ReadFromFile got shutdown message.")
			return
		default:
		}
		// set read time out
		fReader.SetReadDeadline(time.Now().Add(time.Millisecond * time.Duration(timeout)))

		// fill buffer if possible
		bytesRead, err = fReader.Read(buf[:])

		if bytesRead > 0 {
			msgChan <- Message{nil, string(buf[:bytesRead])}
		} else if errors.Is(err, os.ErrDeadlineExceeded) {
			// timeout
			// msgChan <- Message{err, ""}
			continue
		} else {
			// EOF goes here
			msgChan <- Message{err, ""}
			break
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

	for {
		select {
		case <-doneChan:
			return
		default:
		}
		// set read time out
		network.SetReadDeadline(time.Now().Add(time.Millisecond * time.Duration(timeout)))
		// packet received from the network
		// payload, err = network.Recv(timeout)
		payload, err = network.Recv()
		if err != nil {
			if errors.Is(err, os.ErrDeadlineExceeded) {
				// read timeout
				continue
			} else {
				// EOF goes here, in case of error retry it.
				msgChan <- Message{err, ""}
				continue
			}
		} else {
			// normal read
			msgChan <- Message{nil, payload}
		}
	}
	// util.Log.With("quit", true).With("err", err).Debug("ReadFromNetwork")
}
