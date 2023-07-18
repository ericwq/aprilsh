// Copyright 2022~2023 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

//go:build linux || freebsd

package util

import (
	"os"
	"strings"

	utmp "github.com/ericwq/goutmp"
)

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
		// fmt.Printf("#checkUnattachedRecord() user=%q,%q; type=%d, line=%s, host=%s, id=%s, pid=%d\n",
		// 	r.GetUser(), userName, r.GetType(), r.GetLine(), r.GetHost(), r.GetId(), r.GetPid())
		if r.GetType() == utmp.USER_PROCESS && r.GetUser() == userName {
			// does line show unattached session
			host := r.GetHost()
			// fmt.Printf("#checkUnattachedRecord() MATCH user=%q,%q; type=%d,%d", r.GetUser(), userName, r.GetType(), utmp.USER_PROCESS)
			// fmt.Printf(" host=%s, line=%q, ignoreHost=%s\n", host, r.GetLine(), ignoreHost)
			// fmt.Printf("#checkUnattachedRecord() append 1=%t,2=%t,3=%t,4=%t,5=%t\n", len(host) >= 5,
			// 	strings.HasPrefix(host, prefix), strings.HasSuffix(host, "]"), host != ignoreHost, utmp.DeviceExists(r.GetLine()))
			if len(host) >= 5 && strings.HasPrefix(host, prefix) &&
				strings.HasSuffix(host, "]") && host != ignoreHost && utmp.DeviceExists(r.GetLine()) {
				// fmt.Printf("#checkUnattachedRecord() attached session %s\n", host)
				unatttached = append(unatttached, host)
				// } else {
				// 	fmt.Printf("#CheckUnattachedUtmpx() line:%s exist=%t ", r.GetLine(), utmp.DeviceExists(r.GetLine()))
				// 	fmt.Printf("host:%s ignoreHost=%s \n", host, ignoreHost)
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
