// Copyright 2022~2023 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

//go:build linux || freebsd

package main

import (
	"fmt"
	"os"
	"strings"

	utmp "github.com/ericwq/goutmp"
)

type utmpEntry struct {
	ent *utmp.UtmpEntry
}

func addUtmpEntry(ptmxName string, host string) *utmpEntry {
	usr := getCurrentUser()

	entry := utmp.Put_utmp(usr, ptmxName, host)
	return &utmpEntry{&entry}
}

func updateLastLog(ptmxName string) {
	host := fmt.Sprintf("%s [%d]", _PACKAGE_STRING, os.Getpid())
	usr := getCurrentUser()
	utmp.Put_lastlog_entry(_COMMAND_NAME, usr, ptmxName, host)
}

func clearUtmpEntry(entry *utmpEntry) {
	utmp.Unput_utmp(*(entry.ent))
}

var fp func() *utmp.Utmpx // easy for testing

func init() {
	fp = utmp.GetUtmpx
}

func checkUnattachedRecord(userName string, ignoreHost string) []string {
	var unatttached []string
	unatttached = make([]string, 0)

	r := fp()
	for r != nil {
		if r.GetType() == utmp.USER_PROCESS && r.GetUser() == userName {
			// does line show unattached session
			host := r.GetHost()
			if len(host) >= 5 && strings.HasPrefix(host, _PACKAGE_STRING) &&
				strings.HasSuffix(host, "]") && host != ignoreHost && deviceExists(r.GetLine()) {
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
