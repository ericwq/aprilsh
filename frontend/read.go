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
	Recv() error
	DeadLiner
}

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
			msgChan <- Message{err, ""}
			continue
		} else {
			// EOF goes here
			msgChan <- Message{err, ""}
			break
		}
	}
	// fmt.Println("#ReadFromFile exit.")
}

// read data from udp socket and send the result to socketChan
func ReadFromNetwork(timeout int, msgChan chan Message, network DeadLineReceiver,
) {
	for {
		select {
		case m := <-msgChan:
			if m.Data == "shutdown" {
				return
			}
		default:
		}
		// set read time out
		network.SetReadDeadline(time.Now().Add(time.Millisecond * time.Duration(timeout)))
		// packet received from the network
		err := network.Recv()
		if err != nil {
			if errors.Is(err, os.ErrDeadlineExceeded) {
				// network read timeout
			} else {
				msgChan <- Message{err, ""}
			}
		} else {
			msgChan <- Message{nil, ""} // network.Recv() doesn't return the data
		}
	}
}
