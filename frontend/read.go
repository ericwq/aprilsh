// Copyright 2022~2023 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package frontend

import (
	"errors"
	"os"
	"time"

	"github.com/ericwq/aprilsh/network"
)


type Message struct {
	Err  error
	Data string
}

func ReadFromFile(timeout int, msgChan chan Message, fd *os.File) {
	var buf [16384]byte

	// set read time out
	fd.SetDeadline(time.Now().Add(time.Millisecond * time.Duration(timeout)))
	for {
		select {
		case m := <-msgChan:
			if m.Data == "shutdown" {
				return
			}
		default:
		}
		// fill buffer if possible
		bytesRead, err := fd.Read(buf[:])
		if err != nil {
			if errors.Is(err, os.ErrDeadlineExceeded) {
				// file read timeout
			} else {
				msgChan <- Message{err, ""}
				break
			}
		} else if bytesRead == 0 {
			// EOF.
			msgChan <- Message{err, ""}
			break
		} else {
			msgChan <- Message{nil, string(buf[:bytesRead])}
		}
	}
}

// read data from udp socket and send the result to socketChan
func ReadFromNetwork[S network.State[S], R network.State[R]](timeout int, msgChan chan Message,
	network *network.Transport[S, R],
) {
	// set read time out
	network.SetDeadline(time.Now().Add(time.Millisecond * time.Duration(timeout)))
	for {
		select {
		case m := <-msgChan:
			if m.Data == "shutdown" {
				return
			}
		default:
		}
		// packet received from the network
		err := network.Recv()
		if err != nil {
			if errors.Is(err, os.ErrDeadlineExceeded) {
				// network read timeout
			} else {
				msgChan <- Message{err, ""}
			}
		} else {
			msgChan <- Message{nil, ""}
		}
	}
}
