// Copyright 2022 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

//go:build linux

package util

import "testing"

func TestGetShellFail(t *testing.T) {
	// set test mark
	defer func() {
		userCurrentTest = false
	}()

	userCurrentTest = true
	r, _ := GetShell()
	if r != "" {
		t.Errorf("#test getShell() expect empty string, got %s.", r)
	}

	defer func() {
		execCmdTest = false
	}()

	execCmdTest = true
	userCurrentTest = false
	r, _ = GetShell()
	if r != "" {
		t.Errorf("#test getShell() expect empty string, got %s.", r)
	}
}

func TestGetShell(t *testing.T) {
	expect := "/bin/ash"
	r, _ := GetShell()
	if r == "" {
		t.Errorf("#test expect %s got %s\n", expect, r)
	}
}
