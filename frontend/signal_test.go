// Copyright 2022~2023 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package frontend

import (
	"syscall"
	"testing"
)

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

func TestClear(t *testing.T) {
	var s Signals

	s.Handler(syscall.SIGTERM)
	s.Handler(syscall.SIGINT)
	s.Handler(syscall.SIGHUP)

	if !s.AnySignal() {
		t.Errorf("#test AnySignal() expect true, got %t\n", s.AnySignal())
	}

	s.Clear()

	if s.AnySignal() {
		t.Errorf("#test AnySignal() expect false, got %t\n", s.AnySignal())
	}
}
