// Copyright 2022~2023 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

//go:build linux || freebsd

package main

import (
	"fmt"
	"os"

	utmp "blitter.com/go/goutmp"
)

type utmpEntry struct {
	ent *utmp.UtmpEntry
}

func addUtmpEntry(ptsName string) *utmpEntry {
	// ptsName := ptmx.Name()
	host := fmt.Sprintf("%s [%d]", PACKAGE_STRING, os.Getpid())
	usr := getCurrentUser()

	entry := utmp.Put_utmp(usr, ptsName, host)
	return &utmpEntry{&entry}
}

func updateLasLog(ptsName string) {
	host := fmt.Sprintf("%s [%d]", PACKAGE_STRING, os.Getpid())
	usr := getCurrentUser()
	utmp.Put_lastlog_entry(COMMAND_NAME, usr, ptsName, host)
}

func clearUtmpEntry(entry *utmpEntry) {
	utmp.Unput_utmp(*(entry.ent))
}
