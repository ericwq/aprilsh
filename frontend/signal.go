// Copyright 2022~2023 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package frontend

import (
	"fmt"
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
		}
		ret = false
		s[x].Store(0) // clear the signal
	}
	return
}

// TODO do we need to return error ?
func (s *Signals) Handler(signal os.Signal) {
	if sig, ok := signal.(syscall.Signal); ok {
		if sig >= 0 && sig < MAX_SIGNAL_NUMBER { // TODO do we need this protection?
			s[sig].Store(int32(sig))
		} else {
			fmt.Printf("signal out of range: %s\n", sig)
		}
	} else {
		fmt.Printf("signal malform: %v\n", signal)
	}
}

// This method does not consume signal notifications.
func (s *Signals) AnySignal() (rv bool) {
	for i := range s {
		rv = rv || s[i].Load() > 0
	}
	return
}
