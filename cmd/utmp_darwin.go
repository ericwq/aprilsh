// Copyright 2022~2023 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

//go:build darwin

package cmd

import (
	"fmt"
	"os"
)

func AddUtmpEntry(pts *os.File, host string) bool {
	fmt.Printf("unimplement %s\n", "addUtmpEntry()")
	return false
}

func ClearUtmpEntry(pts *os.File) bool {
	fmt.Printf("unimplement %s\n", "clearUtmpEntry()")
	return false
}

func UpdateLastLog(line, userName, host string) {
	fmt.Printf("unimplement %s\n", "updateLastLog()")
}

func CheckUnattachedRecord(userName, ignoreHost, prefix string) []string {
	fmt.Printf("unimplement %s\n", "checkUnattachedRecord()")
	return nil
}
