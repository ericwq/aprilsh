// Copyright 2022~2023 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package frontend

import (
	"os"
	"syscall"
	"testing"
)

func TestGotSignal(t *testing.T) {
	tc := []os.Signal{
		syscall.SIGABRT, syscall.SIGALRM, syscall.SIGBUS, syscall.SIGCHLD, syscall.SIGCLD,
		syscall.SIGCONT, syscall.SIGFPE, syscall.SIGHUP, syscall.SIGILL, syscall.SIGINT,
		syscall.SIGIO, syscall.SIGIOT, syscall.SIGKILL, syscall.SIGPIPE, syscall.SIGPOLL,
		syscall.SIGPROF, syscall.SIGPWR, syscall.SIGQUIT, syscall.SIGSEGV, syscall.SIGSTKFLT,
		syscall.SIGSTOP, syscall.SIGSYS, syscall.SIGTERM, syscall.SIGTRAP, syscall.SIGTSTP,
		syscall.SIGTTIN, syscall.SIGTTOU, syscall.SIGURG, syscall.SIGUSR1,
		syscall.SIGUSR2, syscall.SIGVTALRM, syscall.SIGWINCH, syscall.SIGXCPU, syscall.SIGXFSZ,
	}

	result := []bool{
		true, true, true, true, false,
		true, true, true, true, true,
		true, false, true, true, false,
		true, true, true, true, true,
		true, true, true, true, true,
		true, true, true, true,
		true, true, true, true, true,
	}

	/*
	   syscall.SIGPOLL and syscall.SIGIO has conflict value
	   syscall.SIGIOT and syscall.SIGABRT has conflict value
	   syscall.SIGUNUSED and syscall.SIGSYS has conflict value
	*/

	// initialize Signals array
	var s Signals
	for i := range tc {
		s.Handler(tc[i])
	}

	// validate GotSignal()
	for i := range tc {
		ss := tc[i].(syscall.Signal)
		got := s.GotSignal(ss)
		if got != result[i] {
			t.Errorf("#test GotSignal() %q %x expect %t, got %t\n", tc[i], int(ss), result[i], got)
		}
	}
}
