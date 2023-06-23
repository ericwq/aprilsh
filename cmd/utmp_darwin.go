// Copyright 2022~2023 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

//go:build darwin

package cmd

import "os"

type utmpEntry struct{}

func AddUtmpEntry(pts *os.File, host string) bool {
	logW.Printf("unimplement %s\n", "addUtmpEntry()")
	return false
}

func UpdateLastLog(line, userName, host string) {
	logW.Printf("unimplement %s\n", "updateLastLog()")
}

func ClearUtmpEntry(pts *os.File) bool {
	logW.Printf("unimplement %s\n", "clearUtmpEntry()")
	return false
}

func CheckUnattachedRecord(userName, ignoreHost, prefix string) []string {
	logW.Printf("unimplement %s\n", "checkUnattachedRecord()")
	return nil
}

// func hasUtempter() bool {
// 	logW.Printf("unimplement %s\n", "isUtmpSupport()")
// 	return false
// }
