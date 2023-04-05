// Copyright 2022 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

//go:build darwin

package main

import (
	"os"
	"testing"
)

// two test case for getShell()
//
// % dscl localhost -read Local/Default/Users/
// name: dsRecTypeStandard:Users
//
// % dscl localhost -read Local/Default/Users/doesnotexist
// <dscl_cmd> DS Error: -14136 (eDSRecordNotFound)
func TestGetShell(t *testing.T) {
	user := os.Getenv("USER")

	// lack of user
	os.Unsetenv("USER")
	s, e := getShell()
	if e == nil {
		t.Errorf("#test getShell() darwin empty user, expect error, got nil\n")
	}

	// user does not exist
	os.Setenv("USER", "user does not exist")
	s, e = getShell()
	if e == nil {
		t.Errorf("#test getShell() darwin expect empty string, got %q, error %q\n", s, e)
	}

	os.Setenv("USER", user)
}
