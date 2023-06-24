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
	utmp "github.com/ericwq/goutmp"
)

const (
	PACKAGE_STRING = "aprilsh"
)

func TestUpdateLastLog(t *testing.T) {
	line := "pts/9"
	userName := "ide"
	host := fmt.Sprintf("%s [%d]", PACKAGE_STRING, os.Getpid())

	ret := UpdateLastLog(line, userName, host)
	if !ret {
		t.Errorf("#test UpdateLastLog() failed.")
	}
}

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

func TestCheckUnattachedRecord_Mock(t *testing.T) {
	SetFuncForGetUtmpx(mockGetUtmpx)
	defer func() {
		SetFuncForGetUtmpx(utmp.GetUtmpx)
	}()

	user, _ := user.Current()
	ignoreHost := fmt.Sprintf("%s [%d]", PACKAGE_STRING, os.Getpid())

	unatttached := CheckUnattachedRecord(user.Username, ignoreHost, PACKAGE_STRING)
	expect := PACKAGE_STRING + " [1221]"
	if unatttached == nil && unatttached[0] != expect {
		t.Errorf("#test CheckUnattachedRecord() expect %s, got %v\n", expect, unatttached)
	}
}

var (
	index         int
	utmpxMockData []*utmp.Utmpx
)

func init() {
	data := []struct {
		xtype int
		host  string
		line  string
		usr   string
		id    int
		pid   int
	}{
		{utmp.USER_PROCESS, PACKAGE_STRING + " [1220]", "pts/0", "root", 1, 1},
		{utmp.USER_PROCESS, PACKAGE_STRING + " [1221]", "pts/1", "ide", 51, 1221},
		{utmp.DEAD_PROCESS, PACKAGE_STRING + " [1228]", "pts/3", "ide", 751, 1228},
	}

	for _, v := range data {
		u := &utmp.Utmpx{}

		u.SetType(v.xtype)
		u.SetHost(v.host)
		u.SetLine(v.line)
		u.SetUser(v.usr)
		u.SetId(v.id)
		u.SetPid(v.pid)

		utmpxMockData = append(utmpxMockData, u)
	}
}

// return utmp mock data
func mockGetUtmpx() *utmp.Utmpx {
	if 0 <= index && index < len(utmpxMockData) {
		p := utmpxMockData[index]
		index++
		return p
	}

	return nil
}
