// Copyright 2022~2023 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package frontend

import (
	"errors"
	"fmt"
	"io"
	"sync"
	"testing"
	"time"
)

type mockDeadLineReader struct {
	round   int
	timeout []int
	data    []string
	limit   int
}

func (m *mockDeadLineReader) Read(p []byte) (n int, err error) {
	if m.round >= 0 && m.round < len(m.data) {

		// make sure we increase round
		defer func() { m.round++ }()

		// support read timeout
		time.Sleep(time.Duration(m.timeout[m.round]) * time.Millisecond)
		// if m.timeout[m.round] > m.limit {
		// 	err = os.ErrDeadlineExceeded
		//
		// 	return
		// }

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
	fmt.Printf("#mockDeadLineReader Read p=%s, n=%d, err=%s\n", p, n, err)
	return
}

func (m *mockDeadLineReader) SetReadDeadline(t time.Time) error {
	if m.round >= 0 && m.round < len(m.data) {
	}
	return nil
}

func TestReadFromFile(t *testing.T) {
	// prepare the data
	mockReader := &mockDeadLineReader{}
	mockReader.round = 0
	mockReader.timeout = []int{5, 15, 7, 3}
	mockReader.data = []string{"one", "two", "tree", "four"}
	mockReader.limit = 10

	var fileChan chan Message
	fileChan = make(chan Message, 1)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		ReadFromFile(mockReader.limit, fileChan, mockReader)
	}()

	for i := range mockReader.data {

		fileMsg := <-fileChan
		if fileMsg.Err != nil {
			fmt.Println("#test ReadFromFile read error: ", fileMsg.Err)
		} else {
			fmt.Printf("#test ReadFromFile round=%d got:=%s,%s\n", i, fileMsg.Data, fileMsg.Err)
		}
		if mockReader.data[i] != fileMsg.Data {
			t.Errorf("#ReadFromFile expect %s, got %s\n", mockReader.data[i], fileMsg.Data)
		}
	}

	last := <-fileChan
	if !errors.Is(last.Err, io.EOF) {
		t.Errorf("#test ReadFromFile last read is %s\n", last.Err)
	}
	// shutdown the goroutine
	fmt.Println("send shutdown message")
	fileChan <- Message{Err: nil, Data: "shutdown"}
	// close(fileChan)
	fmt.Println("wait for stop")
	wg.Wait()
}
