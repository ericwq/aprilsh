// Copyright 2022~2023 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

//go:build linux || freebsd

package util

import (
	"fmt"
	"os"
	"strings"
	"testing"

	utmp "github.com/ericwq/goutmp"
)

// https://blog.csdn.net/sg_knight/article/details/134373559
func AddUtmpx(pts *os.File, host string) bool {
	return utmp.UtmpxAddRecord(pts, host)
}

func ClearUtmpx(pts *os.File) bool {
	return utmp.UtmpxRemoveRecord(pts)
}

func UpdateLastLog(line, userName, host string) bool {
	return utmp.PutLastlogEntry(line, userName, host)
}

func CheckUnattachedUtmpx(userName, ignoreHost, prefix string) []string {
	var unatttached []string
	unatttached = make([]string, 0)

	r := fp()
	for r != nil {
		if r.GetType() == utmp.USER_PROCESS && r.GetUser() == userName {
			// does line show unattached session
			host := r.GetHost()
			if testing.Testing() {
				fmt.Printf("#checkUnattachedRecord() MATCH user=(%q,%q) type=(%d,%d) host=%s\n",
					r.GetUser(), userName, r.GetType(), utmp.USER_PROCESS, host)
			}
			if len(host) >= 5 && strings.HasPrefix(host, prefix) &&
				host != ignoreHost && utmp.DeviceExists(r.GetLine()) {
				unatttached = append(unatttached, host)
				if testing.Testing() {
					fmt.Printf("#checkUnattachedRecord() append host=%s, line=%q\n", host, r.GetLine())
				}
			}
		} else {
			if testing.Testing() {
				fmt.Printf("#checkUnattachedRecord() skip user=%q,%q; type=%d, line=%s, host=%s, id=%s, pid=%d\n",
					r.GetUser(), userName, r.GetType(), r.GetLine(), r.GetHost(), r.GetId(), r.GetPid())
			}
		}
		r = fp()
	}

	if len(unatttached) > 0 {
		return unatttached
	}
	return nil
}

var fp func() *utmp.Utmpx // easy for testing

func init() {
	fp = utmp.GetUtmpx
}

// easy for testing under linux
func SetFunc4GetUtmpx(f func() *utmp.Utmpx) {
	fp = f
}
