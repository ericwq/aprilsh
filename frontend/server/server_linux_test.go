// Copyright 2022~2023 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

//go:build linux

package main

import (
	"strings"
	"testing"

	"github.com/ericwq/aprilsh/frontend"
	"github.com/ericwq/aprilsh/util"
	utmp "github.com/ericwq/goutmp"
)

var idx = 0

var strENOTTY = "inappropriate ioctl for device"

func mockGetUtmpx() *utmp.Utmpx {
	userName := getCurrentUser()
	rs := []struct {
		Type int16
		User string
		Host string
		Line string
	}{
		{utmp.USER_PROCESS, "root", frontend.COMMAND_SERVER_NAME + " [777]", "pts/1"},
		{utmp.USER_PROCESS, userName, frontend.COMMAND_SERVER_NAME + " [888]", "pts/7"},
		{utmp.USER_PROCESS, userName, frontend.COMMAND_SERVER_NAME + " [666]", "pts/1"},
		{utmp.USER_PROCESS, userName, frontend.COMMAND_SERVER_NAME + " [999]", "pts/0"},
	}
	// the test requires the following files in /dev/pts directory
	// ls /dev/pts
	// 0     1     2     ptmx

	// if idx out of range, rewind it.
	if idx >= len(rs) {
		idx = 0
		return nil
	}

	u := utmp.Utmpx{}
	u.Type = rs[idx].Type

	b := []byte(rs[idx].User)
	for i := range u.User {
		if i >= len(b) {
			break
		}
		u.User[i] = int8(b[i])
	}

	b = []byte(rs[idx].Host)
	for i := range u.Host {
		if i >= len(b) {
			break
		}
		u.Host[i] = int8(b[i])
	}

	b = []byte(rs[idx].Line)
	for i := range u.Line {
		if i >= len(b) {
			break
		}
		u.Line[i] = int8(b[i])
	}

	// fmt.Printf("#mockGetUtmpx() rs[%d]=%v\n", idx, rs[idx])
	// increase to the next one
	idx++

	// return current one
	return &u
}

func TestWarnUnattached(t *testing.T) {
	// fp = mockGetUtmpx
	util.SetFunc4GetUtmpx(mockGetUtmpx)
	defer func() {
		// fp = utmp.GetUtmpx
		util.SetFunc4GetUtmpx(utmp.GetUtmpx)
		idx = 0
	}()

	tc := []struct {
		label      string
		ignoreHost string
		count      int
	}{
		{"one match", frontend.COMMAND_SERVER_NAME + " [999]", 1},   // 666 pts/1 exist, 888 pts/7 does not exist, only 666 remains
		{"two matches", frontend.COMMAND_SERVER_NAME + " [888]", 2}, // 666 pts1 exist, 999 pts/0 exist, so 666 and 999 remains
	}

	for _, v := range tc {
		t.Run(v.label, func(t *testing.T) {
			var out strings.Builder
			warnUnattached(&out, v.ignoreHost)
			got := out.String()
			// t.Logf("%q\n", got)
			count := strings.Count(got, "- ")
			switch count {
			case 0: // warnUnattached found one unattached session
				if strings.Index(got, "detached session on this server") != -1 && v.count != 1 {
					t.Errorf("#test warnUnattached() %q expect %d warning, got 1.\n", v.label, v.count)
				}
			default: // warnUnattached found more than one unattached session
				if count != v.count {
					t.Errorf("#test warnUnattached() %q expect %d warning, got %d.\n", v.label, v.count, count)
				}
			}
		})
	}
}

// always return nil
func mockGetUtmpx0() *utmp.Utmpx {
	return nil
}

func TestWarnUnattached0(t *testing.T) {
	// fp = mockGetUtmpx0
	util.SetFunc4GetUtmpx(mockGetUtmpx0)
	defer func() {
		util.SetFunc4GetUtmpx(utmp.GetUtmpx)
		// fp = utmp.GetUtmpx
		idx = 0
	}()
	var out strings.Builder
	warnUnattached(&out, "anything")
	got := out.String()
	if len(got) != 0 {
		t.Logf("%s\n", got)
		t.Errorf("#test warnUnattached() zero match expect 0, got %d\n", len(got))
	}
}
