// Copyright 2022~2023 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

//go:build linux || freebsd

package cmd

import (
	"os"
	"strings"

	utmp "github.com/ericwq/goutmp"
)

func AddUtmpEntry(pts *os.File, host string) bool {
	return utmp.UtmpxAddRecord(pts, host)
}

func ClearUtmpEntry(pts *os.File) bool {
	return utmp.UtmpxRemoveRecord(pts)
}

func UpdateLastLog(line, userName, host string) bool {
	return utmp.PutLastlogEntry(line, userName, host)
}

func CheckUnattachedRecord(userName, ignoreHost, prefix string) []string {
	var unatttached []string
	unatttached = make([]string, 0)

	r := fp()
	for r != nil {
		// fmt.Printf("#checkUnattachedRecord() user=%q,%q; type=%d, line=%s, host=%s\n",
		// 	r.GetUser(), userName, r.GetType(), r.GetLine(), r.GetHost())
		if r.GetType() == utmp.USER_PROCESS && r.GetUser() == userName {
			// does line show unattached session
			host := r.GetHost()
			// fmt.Printf("#checkUnattachedRecord() MATCH user=%q,%q; type=%d,%d", r.GetUser(), userName, r.GetType(), utmp.USER_PROCESS)
			// fmt.Printf(" host=%s, line=%q, ignoreHost=%s\n", host, r.GetLine(), ignoreHost)
			if len(host) >= 5 && strings.HasPrefix(host, prefix) &&
				strings.HasSuffix(host, "]") && host != ignoreHost && utmp.DeviceExists(r.GetLine()) {
				// fmt.Printf("#checkUnattachedRecord() attached session %s\n", host)
				unatttached = append(unatttached, host)
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
func SetFuncForGetUtmpx(f func() *utmp.Utmpx) {
	fp = f
}
