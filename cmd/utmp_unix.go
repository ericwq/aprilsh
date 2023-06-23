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
	// usr := getCurrentUser()
	//
	// entry := utmp.Put_utmp(usr, ptmxName, host)
	// return &utmpEntry{&entry}
	return utmp.UtmpxAddRecord(pts, host)
}

/*
	func updateLastLog(ptmxName string) {
		host := fmt.Sprintf("%s [%d]", _PACKAGE_STRING, os.Getpid())
		usr := getCurrentUser()
		utmp.PutLastlogEntry(_COMMAND_NAME, usr, ptmxName, host)
	}
*/
func UpdateLastLog(line, userName, host string) {
	// host := fmt.Sprintf("%s [%d]", _PACKAGE_STRING, os.Getpid())
	// usr := getCurrentUser()
	// utmp.PutLastlogEntry(_COMMAND_NAME, usr, ptmxName, host)
	utmp.PutLastlogEntry(line, userName, host)
}

// func clearUtmpEntry(entry *utmpEntry) {
// 	utmp.Unput_utmp(*(entry.ent))
// }

func ClearUtmpEntry(pts *os.File) bool {
	return utmp.UtmpxRemoveRecord(pts)
}

var fp func() *utmp.Utmpx // easy for testing

func init() {
	fp = utmp.GetUtmpx
	// utmpSupport = hasUtmpSupport()
}

func CheckUnattachedRecord(userName, ignoreHost, prefix string) []string {
	var unatttached []string
	unatttached = make([]string, 0)

	r := fp()
	for r != nil {
		if r.GetType() == utmp.USER_PROCESS && r.GetUser() == userName {
			// does line show unattached session
			host := r.GetHost()
			// fmt.Printf("#checkUnattachedRecord() user=%q,%q; type=%d,%d", r.GetUser(), userName, r.GetType(), utmp.USER_PROCESS)
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

// func hasUtmpSupport() bool {
// 	r := fp()
// 	if r != nil {
// 		return true
// 	}
// 	return false
// }
