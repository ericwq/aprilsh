// Copyright 2022~2023 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package frontend

import (
	"os"
	"syscall"
	"testing"
)

/*
./signal_test.go:15:78: undefined: syscall.SIGCLD
./signal_test.go:17:76: undefined: syscall.SIGPOLL
./signal_test.go:18:28: undefined: syscall.SIGPWR
./signal_test.go:18:78: undefined: syscall.SIGSTKFLT
./signal_test.go:20:45: undefined: syscall.SIGUNUSED

*/
func TestGotSignal(t *testing.T) {
	tc := []os.Signal{
		syscall.SIGABRT, syscall.SIGALRM, syscall.SIGBUS, syscall.SIGCHLD, syscall.SIGCLD,
		syscall.SIGCONT, syscall.SIGFPE, syscall.SIGHUP, syscall.SIGILL, syscall.SIGINT,
		syscall.SIGIO, syscall.SIGIOT, syscall.SIGKILL, syscall.SIGPIPE, syscall.SIGPOLL,
		syscall.SIGPROF, syscall.SIGPWR, syscall.SIGQUIT, syscall.SIGSEGV, syscall.SIGSTKFLT,
		syscall.SIGSTOP, syscall.SIGSYS, syscall.SIGTERM, syscall.SIGTRAP, syscall.SIGTSTP,
		syscall.SIGTTIN, syscall.SIGTTOU, syscall.SIGUNUSED, syscall.SIGURG, syscall.SIGUSR1,
		syscall.SIGUSR2, syscall.SIGVTALRM, syscall.SIGWINCH, syscall.SIGXCPU, syscall.SIGXFSZ,
	}

	repeat := []bool{
		true, true, true, true, false,
		true, true, true, true, true,
		true, false, true, true, false,
		true, true, true, true, true,
		true, true, true, true, true,
		true, true, false, true, true,
		true, true, true, true, true,
	}
	var s Signals
	for i := range tc {
		s.Handler(tc[i])
	}

	for i := range tc {
		ss := tc[i].(syscall.Signal)
		got := s.GotSignal(ss)
		if got != repeat[i] {
			t.Errorf("#test GotSignal() %q %x expect %t, got %t\n", tc[i], int(ss), repeat[i], got)
		}
	}
}

func TestAnySignal(t *testing.T) {
	var s Signals

	s.Handler(syscall.Signal(-1))
	if s.AnySignal() {
		t.Errorf("#test AnySignal() expect false got %t", s.AnySignal())
	}

	s.Handler(syscall.SIGABRT)
	if !s.AnySignal() {
		t.Errorf("#test AnySignal() expect true got %t", s.AnySignal())
	}
}
