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

// Check whether we got the spcified signal. If so return true, otherwise false.
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

// handle the signal by store it in Signals array.
func (s *Signals) Handler(signal os.Signal) {
	sig, ok := signal.(syscall.Signal)
	if ok && sig >= 0 && sig < MAX_SIGNAL_NUMBER {
		s[sig].Store(int32(sig))
	}
}

// Check whether we got any signal.
// This method DOES NOT consumes any signal notification.
func (s *Signals) AnySignal() (rv bool) {
	for i := range s {
		rv = rv || s[i].Load() > 0
		if rv {
			break
		}
	}
	return
}

// clear all the signals
func (s *Signals) Clear() {
	for i := range s {
		// clear signal
		s[i].Store(0)
	}
}
