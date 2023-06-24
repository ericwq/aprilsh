// Copyright 2022~2023 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

//go:build darwin

package cmd

import (
	"fmt"
	"os"
)

func AddUtmpx(pts *os.File, host string) bool {
	fmt.Printf("unimplement %s\n", "AddUtmpx()")
	return false
}

func ClearUtmpx(pts *os.File) bool {
	fmt.Printf("unimplement %s\n", "ClearUtmpx()")
	return false
}

func UpdateLastLog(line, userName, host string) bool {
	fmt.Printf("unimplement %s\n", "UpdateLastLog()")
	return false
}

func CheckUnattachedUtmpx(userName, ignoreHost, prefix string) []string {
	fmt.Printf("unimplement %s\n", "CheckUnattachedUtmpx()")
	return nil
}
