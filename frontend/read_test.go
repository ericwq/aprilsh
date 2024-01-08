// Copyright 2022~2023 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package frontend

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"sync"
	"testing"
	"time"
)

type mockFile struct {
	round   int
	timeout []int
	data    []string
	err     []error
	limit   int
}

func (m *mockFile) Read(p []byte) (n int, err error) {
	if m.round >= 0 && m.round < len(m.data) {

		// make sure we increase round
		defer func() { m.round++ }()

		// support read timeout
		time.Sleep(time.Duration(m.timeout[m.round]) * time.Millisecond)
		if m.timeout[m.round] > m.limit {
			err = os.ErrDeadlineExceeded
			// fmt.Printf("#mockFile Read round=%d, n=%d, err=%s\n", m.round, n, err)
			return
		} else if m.timeout[m.round] == m.limit {
			err = os.ErrPermission
			// fmt.Printf("#mockFile Read round=%d, n=%d, err=%s\n", m.round, n, err)
			return
		}

		// normal read
		copy(p, []byte(m.data[m.round]))
		n = len(m.data[m.round])
		err = nil
		// fmt.Printf("#mockFile Read round=%d, n=%d, err=%s\n", m.round, n, err)
		return
	}
	m.round = 0
	n = 0
	err = io.EOF
	// fmt.Printf("#mockFile Read round=%d, n=%d, err=%s\n", m.round, n, err)
	return
}

func TestReadFromFile(t *testing.T) {
	// prepare the data
	mockReader := &mockFile{}
	mockReader.round = 0
	mockReader.limit = 10
	mockReader.timeout = []int{5, 5, 7, 3, 8, 15}
	mockReader.data = []string{"zero", "one", "two", "tree", "four", "five"}
	mockReader.err = []error{nil, nil, nil, nil, nil, os.ErrDeadlineExceeded}

	var fileChan chan Message
	var doneChan chan any
	fileChan = make(chan Message, 0)
	doneChan = make(chan any, 1)

	// start the deal line reader
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		ReadFromFile(mockReader.limit, fileChan, doneChan, mockReader)
	}()

	// check the consistency of mock data
	if len(mockReader.data) != len(mockReader.err) || len(mockReader.err) != len(mockReader.timeout) {
		t.Errorf("#test ReadFromFile the size of data is not equeal. %d,%d,%d \n",
			len(mockReader.timeout), len(mockReader.data), len(mockReader.err))
	}

	// consume the data from reader
	for i := range mockReader.err {
		// got message from reader channel
		fileMsg := <-fileChan
		// fmt.Printf("got %s,%s\n", fileMsg.Data, fileMsg.Err)
		if mockReader.err[i] != nil {
			if !errors.Is(fileMsg.Err, mockReader.err[i]) {
				t.Errorf("#test ReadFromFile expect %s, got %s\n", mockReader.err[i], fileMsg.Err)
			}
			if "" != fileMsg.Data {
				t.Errorf("#test ReadFromFile expect %q, got %s\n", "", fileMsg.Data)
			}
		} else {
			if mockReader.data[i] != fileMsg.Data {
				t.Errorf("#test ReadFromFile expect %s, got %s\n", mockReader.data[i], fileMsg.Data)
			}
		}
	}

	doneChan <- "done"
	// consume EOF message
	select {
	case <-fileChan:
	default:
	}
	wg.Wait()
}

func TestReadFromFile_DoneChan(t *testing.T) {
	// prepare the data
	mockReader := &mockFile{}
	mockReader.round = 0
	mockReader.limit = 10
	mockReader.timeout = []int{5, 5, 7, 3, 8, 10}
	mockReader.data = []string{"zero+", "one+", "two+", "tree+", "four+", "five+"}
	mockReader.err = []error{nil, nil, nil, nil, nil, os.ErrPermission}

	var fileChan chan Message
	var doneChan chan any
	fileChan = make(chan Message, 1)
	doneChan = make(chan any, 1)

	// start the deal line reader
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		ReadFromFile(mockReader.limit, fileChan, doneChan, mockReader)
	}()

	// check the consistency of mock data
	if len(mockReader.data) != len(mockReader.err) || len(mockReader.err) != len(mockReader.timeout) {
		t.Errorf("#test ReadFromFile the size of data is not equeal. %d,%d,%d \n",
			len(mockReader.timeout), len(mockReader.data), len(mockReader.err))
	}

	// consume the data from reader
	for i := range mockReader.data {
		// got message from reader channel
		fileMsg := <-fileChan
		if mockReader.err[i] != nil {
			if !errors.Is(fileMsg.Err, mockReader.err[i]) {
				t.Errorf("#test ReadFromFile expect %s, got %s\n", mockReader.err[i], fileMsg.Err)
			}
			if "" != fileMsg.Data {
				t.Errorf("#test ReadFromFile expect %q, got %s\n", "", fileMsg.Data)
			}
		} else {
			if mockReader.data[i] != fileMsg.Data {
				t.Errorf("#test ReadFromFile expect %s, got %s\n", mockReader.data[i], fileMsg.Data)
			}
		}

		// early shutdown
		if i == 2 {
			doneChan <- "done"
			break
		}
	}

	// consume last message to release the reader
	select {
	case <-fileChan:
	default:
	}
	wg.Wait()
}

type mockConnection struct {
	round   int
	timeout []int
	data    []string
	err     []error
	limit   int
}

// func (m *mockDeadLineReceiver) SetReadDeadline(t time.Time) error {
// 	return nil
// }

func (m *mockConnection) Recv(timeout int) (payload string, rAddr net.Addr, err error) {
	// func (m *mockDeadLineReceiver) Recv() (err error) {
	if m.round >= 0 && m.round < len(m.err) {
		// make sure we increase round
		defer func() { m.round++ }()

		// support read timeout
		time.Sleep(time.Duration(m.timeout[m.round]) * time.Millisecond)
		if m.timeout[m.round] > m.limit {
			err = os.ErrDeadlineExceeded
			fmt.Printf("#mockConnection Read round=%d, data=%s, err=%s\n", m.round, payload, err)
			return
		} else if m.timeout[m.round] == m.limit {
			err = os.ErrPermission
			fmt.Printf("#mockConnection Read round=%d, data=%s, err=%s\n", m.round, payload, err)
			return
		}
		// normal read
		payload = m.data[m.round]
		err = nil
		fmt.Printf("#mockConnection Read round=%d, data=%s, err=%s\n", m.round, payload, err)
		return
	}

	m.round = 0
	// err = net.ErrClosed
	err = net.ErrClosed
	fmt.Printf("#mockConnection* Read round=%d, data=%s, err=%s\n", m.round, payload, err)
	return
}

func TestReadFromNetwork(t *testing.T) {
	// prepare the data
	mr := &mockConnection{}
	mr.round = 0
	mr.limit = 10
	mr.timeout = []int{5, 15, 7, 3, 8, 10}
	mr.data = []string{"zero>", "one>", "two>", "tree>", "four>", "five>"}
	mr.err = []error{nil, os.ErrDeadlineExceeded, nil, nil, nil, os.ErrPermission}

	var networkChan chan Message
	var doneChan chan any
	networkChan = make(chan Message, 1)
	doneChan = make(chan any, 1)

	// start the deal line reader
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		ReadFromNetwork(mr.limit, networkChan, doneChan, mr)
	}()

	// check the consistency of mock data
	if len(mr.err) != len(mr.timeout) {
		t.Errorf("#test ReadFromNetwork the size of err and timeout is not equeal. %d,%d \n",
			len(mr.timeout), len(mr.err))
	}

	// consume the data from reader
	for i := range mr.err {
		if mr.err[i] == os.ErrDeadlineExceeded {
			continue
		}

		// got message from reader channel
		netMsg := <-networkChan
		fmt.Printf("#TestReadFromNetwork got %s,%s\n", netMsg.Data, netMsg.Err)

		if mr.err[i] != nil {
			if !errors.Is(netMsg.Err, mr.err[i]) {
				t.Errorf("#test ReadFromFile expect %s, got %s\n", mr.err[i], netMsg.Err)
			}
			if "" != netMsg.Data {
				t.Errorf("#test ReadFromFile expect %q, got %s\n", "", netMsg.Data)
			}
		} else {
			if mr.data[i] != netMsg.Data {
				t.Errorf("#test ReadFromFile expect %s, got %s\n", mr.data[i], netMsg.Data)
			}
		}
	}

	//shutdown the reader
	doneChan <- "done"

	// consume last message to release the reader
	select {
	case <-networkChan:
	default:
	}
	wg.Wait()
}

func TestReadFromNetwork_ErrClosed(t *testing.T) {
	// prepare the data
	mr := &mockConnection{}
	mr.round = 0
	mr.limit = 10
	mr.timeout = []int{5, 5, 7, 3, 8, 10}
	mr.data = []string{"zero*", "one*", "two*", "tree*", "four*", "five*"}
	mr.err = []error{nil, nil, nil, nil, nil, net.ErrClosed}

	var networkChan chan Message
	var doneChan chan any
	networkChan = make(chan Message, 1)
	doneChan = make(chan any, 1)

	// start the deal line reader
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		ReadFromNetwork(mr.limit, networkChan, doneChan, mr)
	}()

	// check the consistency of mock data
	if len(mr.err) != len(mr.timeout) {
		t.Errorf("#test ReadFromNetwork the size of err and timeout is not equeal. %d,%d \n",
			len(mr.timeout), len(mr.err))
	}

	// consume the data from reader
	for i := range mr.err {
		if mr.err[i] == os.ErrDeadlineExceeded {
			continue
		}
		if mr.err[i] == net.ErrClosed {
			break
		}

		// got message from reader channel
		netMsg := <-networkChan

		fmt.Printf("got %s,%s\n", netMsg.Data, netMsg.Err)
		if mr.err[i] != nil {
			if !errors.Is(netMsg.Err, mr.err[i]) {
				t.Errorf("#test ReadFromFile expect %s, got %s\n", mr.err[i], netMsg.Err)
			}
			if "" != netMsg.Data {
				t.Errorf("#test ReadFromFile expect %q, got %s\n", "", netMsg.Data)
			}
		} else {
			if mr.data[i] != netMsg.Data {
				t.Errorf("#test ReadFromFile expect %s, got %s\n", mr.data[i], netMsg.Data)
			}
		}
	}

	wg.Wait()
}
