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
	var reading bool

	for {
		timer := time.NewTimer(time.Duration(timeout) * time.Millisecond)
		if !reading {
			go func(fr io.Reader, buf []byte, ch chan Message) {
				reading = true
				// util.Log.With("action", "satrt").Debug("#read")
				bytesRead, err = fr.Read(buf)
				if bytesRead > 0 {
					ch <- Message{string(buf[:bytesRead]), nil, nil}
					reading = false
					// util.Log.With("action", "got").Debug("#read")
				} else {
					ch <- Message{"", nil, err}
					// util.Log.With("error", "EOF").Debug("#read")
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
					// util.Log.With("error", err).Debug("#ReadFromNetwork")
					return
				}
				// in case of other error, notify the caller and continue.
				msgChan <- Message{"", nil, err}
			}
		} else {
			// normal read
			msgChan <- Message{payload, rAddr, nil}
		}

		// waiting for time out or get the shutdown message
		// 5 times timeout is a experience value from debug info
		timer := time.NewTimer(time.Duration(timeout*5) * time.Millisecond)
		select {
		case <-doneChan:
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}
