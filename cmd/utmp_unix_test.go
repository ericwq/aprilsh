// Copyright 2022 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

//go:build linux

package cmd

import (
	"fmt"
	"os"
	"os/user"
	"testing"

	"github.com/creack/pty"
)

const (
	PACKAGE_STRING = "aprilsh"
)

func TestCheckUnattachedRecord(t *testing.T) {
	// in the following test condition the CheckUnattachedRecord will return nil
	user, _ := user.Current()
	ignoreHost := fmt.Sprintf("%s [%d]", PACKAGE_STRING, os.Getpid())
	// t.Logf("#test CheckUnattachedRecord() user=%q, ignoreHost=%q\n", user.Username, ignoreHost)

	unatttached := CheckUnattachedRecord(user.Username, ignoreHost, PACKAGE_STRING)
	if unatttached != nil {
		t.Errorf("#test CheckUnattachedRecord() expect nil return, got %v\n", unatttached)
	}

	// open pts master and slave
	ptmx, pts, err := pty.Open()
	if err != nil {
		t.Errorf("#test CheckUnattachedRecord() open pts failed, %s", err)
	}
	defer func() {
		ptmx.Close()
		pts.Close()
	}() // Best effort.

	// add test data
	fakeHost := fmt.Sprintf("%s [%d]", PACKAGE_STRING, os.Getpid()+1)
	t.Logf("#test CheckUnattachedRecord() after add an record. fake host=%s, ignoreHost=%s\n",
		fakeHost, ignoreHost)
	ret := AddUtmpEntry(pts, fakeHost)
	t.Logf("#test CheckUnattachedRecord() AddUtmpEntry() return %t\n", ret)

	// CheckUnattachedRecord should return one record
	unatttached = CheckUnattachedRecord(user.Username, ignoreHost, PACKAGE_STRING)
	if unatttached == nil {
		t.Errorf("#test CheckUnattachedRecord() should return one record, got %v\n", unatttached)
	}

	// clean the test data
	ret = ClearUtmpEntry(pts)
	t.Logf("#test CheckUnattachedRecord() ClearUtmpEntry() return %t\n", ret)
}
