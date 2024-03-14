// Copyright 2022 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

//go:build darwin

package util

import (
	"os/user"
	"testing"
)

// two test case for getShell()
//
// % dscl localhost -read Local/Default/Users/
// name: dsRecTypeStandard:Users
//
// % dscl localhost -read Local/Default/Users/doesnotexist
// <dscl_cmd> DS Error: -14136 (eDSRecordNotFound)
func TestGetShellDarwin(t *testing.T) {

	// get current user
	s, e := GetShell()
	if e != nil {
		t.Errorf("#test GetShell() darwin expect no error, got %q, error %q\n", s, e)
	}

	// normal user
	u, err := user.Current()
	if err != nil {
		t.Errorf("#test darwin expect no error, got %s, error %q\n", u, e)
	}

	// get shell for this user
	s, e = GetShell4(u)
	if e != nil {
		t.Errorf("#test GetShell4() darwin expect no error, got %s, error %q\n", s, e)
	}
}
