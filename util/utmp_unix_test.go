// Copyright 2022 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

//go:build linux

package util

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
	// userName := "ide"
	user, _ := user.Current()
	userName := user.Username
	host := fmt.Sprintf("%s [%d]", PACKAGE_STRING, os.Getpid())

	ret := UpdateLastLog(line, userName, host)
	msg := "This test require lastlog access privilege."
	if !ret {
		if userName != "root" {
			t.Skip(msg)
		} else {
			t.Errorf("#test UpdateLastLog() failed. %s\n", msg)
		}
	}
}

func TestCheckUnattachedUtmpx(t *testing.T) {
	// in the following test condition the CheckUnattachedUtmpx will return nil
	user, _ := user.Current()
	ignoreHost := fmt.Sprintf("%s [%d]", PACKAGE_STRING, os.Getpid())
	// t.Logf("#test CheckUnattachedUtmpx() user=%q, ignoreHost=%q\n", user.Username, ignoreHost)

	unatttached := CheckUnattachedUtmpx(user.Username, ignoreHost, PACKAGE_STRING)
	if unatttached != nil {
		t.Errorf("#test CheckUnattachedUtmpx() expect nil return, got %v\n", unatttached)
	}

	// open pts master and slave
	ptmx, pts, err := pty.Open()
	if err != nil {
		t.Errorf("#test CheckUnattachedUtmpx() open pts failed, %s", err)
	}
	defer func() {
		ptmx.Close()
		pts.Close()
	}() // Best effort.

	// fmt.Printf("\n")

	// add test data
	msg := "This test require utmps privilege."
	fakeHost := fmt.Sprintf("%s [%d]", PACKAGE_STRING, os.Getpid()+1)
	// fmt.Printf("#test CheckUnattachedUtmpx() after add an record. fake host=%s, ignoreHost=%s\n",
	// 	fakeHost, ignoreHost)
	ret := AddUtmpx(pts, fakeHost) // the go test can't give the required utmps privilege
	if !ret {
		if user.Username != "root" {
			t.Skip(msg)
		} else {
			t.Errorf("#test CheckUnattachedUtmpx() AddUtmpx() return %t, %s\n", ret, msg)
		}
	}

	// CheckUnattachedUtmpx should return one record
	unatttached = CheckUnattachedUtmpx(user.Username, ignoreHost, PACKAGE_STRING)
	if unatttached == nil {
		t.Errorf("#test CheckUnattachedUtmpx() should return one record, got %v\n", unatttached)
	}

	// clean the test data
	ret = ClearUtmpx(pts)
	if !ret {
		t.Errorf("#test CheckUnattachedUtmpx() ClearUtmpx() return %t, %s\n", ret, msg)
	}
}

func TestCheckUnattachedUtmpx_Mock(t *testing.T) {
	SetFunc4GetUtmpx(mockGetUtmpx)
	defer func() {
		SetFunc4GetUtmpx(utmp.GetUtmpx)
	}()

	user, _ := user.Current()
	ignoreHost := fmt.Sprintf("%s [%d]", PACKAGE_STRING, 1223)

	unatttached := CheckUnattachedUtmpx(user.Username, ignoreHost, PACKAGE_STRING)
	expect := PACKAGE_STRING + " [1221]"
	if unatttached == nil {
		t.Errorf("#test CheckUnattachedUtmpx() expect 1 result, got nothing\n")
	}

	if unatttached != nil && unatttached[0] != expect {
		t.Errorf("#test CheckUnattachedUtmpx() expect %s, got %v\n", expect, unatttached)
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
		{utmp.USER_PROCESS, PACKAGE_STRING + " [1221]", "pts/2", "ide", 51, 1221},
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
