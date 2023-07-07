// Copyright 2022~2023 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package frontend

import (
	"errors"
	"io"
	"os"
	"sync"
	"testing"
	"time"
)

type mockDeadLineReader struct {
	round   int
	timeout []int
	data    []string
	err     []error
	limit   int
}

func (m *mockDeadLineReader) Read(p []byte) (n int, err error) {
	if m.round >= 0 && m.round < len(m.data) {

		// make sure we increase round
		defer func() { m.round++ }()

		// support read timeout
		time.Sleep(time.Duration(m.timeout[m.round]) * time.Millisecond)
		if m.timeout[m.round] > m.limit {
			err = os.ErrDeadlineExceeded

			return
		}

		// normal read
		copy(p, []byte(m.data[m.round]))
		n = len(m.data[m.round])
		err = nil

		// fmt.Printf("#mockDeadLineReader Read p=%s, n=%d, err=%S\n", p, n, err)
		return
	}
	// m.round = 0
	n = 0
	err = io.EOF
	// fmt.Printf("#mockDeadLineReader Read p=%s, n=%d, err=%s\n", p, n, err)
	return
}

func (m *mockDeadLineReader) SetReadDeadline(t time.Time) error {
	return nil
}

func TestReadFromFile(t *testing.T) {
	// prepare the data
	mockReader := &mockDeadLineReader{}
	mockReader.round = 0
	mockReader.limit = 10
	mockReader.timeout = []int{5, 15, 7, 3, 20, 8}
	mockReader.data = []string{"zero", "one", "two", "tree", "four", "five"}
	mockReader.err = []error{nil, os.ErrDeadlineExceeded, nil, nil, os.ErrDeadlineExceeded, nil}

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
		if fileMsg.Err != nil {
			// validate the error case
			if !errors.Is(fileMsg.Err, os.ErrDeadlineExceeded) {
				t.Errorf("#test ReadFromFile expect %s, got %s\n", mockReader.err[i], fileMsg.Err)
			}
			// fmt.Printf("#test ReadFromFile round=%d read error:%s\n", i, fileMsg.Err)
		} else {
			// fmt.Printf("#test ReadFromFile round=%d read %s\n", i, fileMsg.Data)

			// validate the data field of message
			if mockReader.data[i] != fileMsg.Data {
				t.Errorf("#test ReadFromFile expect %s, got %s\n", mockReader.data[i], fileMsg.Data)
			}
		}
	}

	// consume EOF message
	last := <-fileChan
	if !errors.Is(last.Err, io.EOF) {
		t.Errorf("#test ReadFromFile last read is %s\n", last.Err)
	}
	wg.Wait()
}

func TestReadFromFile_DoneChan(t *testing.T) {
	// prepare the data
	mockReader := &mockDeadLineReader{}
	mockReader.round = 0
	mockReader.limit = 10
	mockReader.timeout = []int{5, 15, 7, 3, 20, 8}
	mockReader.data = []string{"zero+", "one+", "two+", "tree+", "four+", "five+"}
	mockReader.err = []error{nil, os.ErrDeadlineExceeded, nil, nil, os.ErrDeadlineExceeded, nil}

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
		if fileMsg.Err != nil {
			// validate the error case
			if !errors.Is(fileMsg.Err, os.ErrDeadlineExceeded) {
				t.Errorf("#test ReadFromFile expect %s, got %s\n", mockReader.err[i], fileMsg.Err)
			}
			// fmt.Printf("#test ReadFromFile round=%d read error:%s\n", i, fileMsg.Err)
		} else {
			// fmt.Printf("#test ReadFromFile round=%d read %s\n", i, fileMsg.Data)

			// validate the data field of message
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
	<-fileChan
	wg.Wait()
}

type mockDeadLineReceiver struct {
	round   int
	timeout []int
	err     []error
	limit   int
}

func (m *mockDeadLineReceiver) SetReadDeadline(t time.Time) error {
	return nil
}

func (m *mockDeadLineReceiver) Recv() (err error) {
	if m.round >= 0 && m.round < len(m.err) {
		// make sure we increase round
		defer func() { m.round++ }()

		// support read timeout
		time.Sleep(time.Duration(m.timeout[m.round]) * time.Millisecond)
		if m.timeout[m.round] > m.limit {
			err = os.ErrDeadlineExceeded
			return
		}

		// normal read
		err = nil
		return
	}
	err = io.EOF
	return
}

func TestReadFromNetwork(t *testing.T) {
	// prepare the data
	mr := &mockDeadLineReceiver{}
	mr.round = 0
	mr.limit = 10
	mr.timeout = []int{5, 15, 7, 3, 20, 8}
	mr.err = []error{nil, os.ErrDeadlineExceeded, nil, nil, os.ErrDeadlineExceeded, nil}

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
			len(mr.timeout),  len(mr.err))
	}

	// consume the data from reader
	for i := range mr.err {

		// got message from reader channel
		fileMsg := <-networkChan
		if fileMsg.Err != nil {
			// validate the error case
			if !errors.Is(fileMsg.Err, os.ErrDeadlineExceeded) {
				t.Errorf("#test ReadFromNetwork expect %s, got %s\n", mr.err[i], fileMsg.Err)
			}
			// fmt.Printf("#test ReadFromNetwork round=%d read error:%s\n", i, fileMsg.Err)
		} else {
			// fmt.Printf("#test ReadFromNetwork round=%d read %q\n", i, fileMsg.Data)

			// validate the data field of message
			if mr.err[i] != fileMsg.Err {
				t.Errorf("#test ReadFromNetwork expect %s, got %s\n", mr.err[i], fileMsg.Err)
			}
		}
	}

	// consume last message to release the reader
	<-networkChan
	wg.Wait()
}

func TestReadFromNetwork_DownChan(t *testing.T) {
	// prepare the data
	mr := &mockDeadLineReceiver{}
	mr.round = 0
	mr.limit = 10
	mr.timeout = []int{5, 15, 7, 3, 20, 8}
	mr.err = []error{nil, os.ErrDeadlineExceeded, nil, nil, os.ErrDeadlineExceeded, nil}

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
			len(mr.timeout),  len(mr.err))
	}

	// consume the data from reader
	for i := range mr.err {

		// got message from reader channel
		fileMsg := <-networkChan
		if fileMsg.Err != nil {
			// validate the error case
			if !errors.Is(fileMsg.Err, os.ErrDeadlineExceeded) {
				t.Errorf("#test ReadFromNetwork expect %s, got %s\n", mr.err[i], fileMsg.Err)
			}
			// fmt.Printf("#test ReadFromNetwork round=%d read error:%s\n", i, fileMsg.Err)
		} else {
			// fmt.Printf("#test ReadFromNetwork round=%d read %q\n", i, fileMsg.Data)

			// validate the data field of message
			if mr.err[i] != fileMsg.Err {
				t.Errorf("#test ReadFromNetwork expect %s, got %s\n", mr.err[i], fileMsg.Err)
			}
		}

		// early shutdown
		if i == 2 {
			doneChan <- "done"
			break
		}
	}

	// consume last message to release the reader
	<-networkChan
	wg.Wait()
}