// Copyright 2022~2023 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package frontend

import (
	"os"
	"sync/atomic"
	"syscall"
)

const (
	MAX_SIGNAL_NUMBER = 64
)

type Signals [MAX_SIGNAL_NUMBER]atomic.Int32

// This method consumes a signal notification.
func (s *Signals) GotSignal(x syscall.Signal) (ret bool) {
	if x >= 0 && x < MAX_SIGNAL_NUMBER {
		if s[x].Load() > 0 {
			ret = true
		} else {
			// fmt.Printf("#GotSignal not found %d,%d\n", x, s[x].Load())
			ret = false
		}
		s[x].Store(0) // clear the signal
	}
	return
}

// TODO do we need to return error ?
func (s *Signals) Handler(signal os.Signal) {
	sig, ok := signal.(syscall.Signal)
	if ok && sig >= 0 && sig < MAX_SIGNAL_NUMBER {
		s[sig].Store(int32(sig))
	}
}

// This method does not consume signal notifications.
func (s *Signals) AnySignal() (rv bool) {
	for i := range s {
		rv = rv || s[i].Load() > 0
	}
	return
}
