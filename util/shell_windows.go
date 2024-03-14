// Copyright 2022 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package util

import (
	"os"
)

// https://github.com/riywo/loginshell/blob/master/loginshell.go
func GetShell() (string, error) {
	consoleApp := os.Getenv("COMSPEC")
	if consoleApp == "" {
		consoleApp = "cmd.exe"
	}

	return consoleApp, nil
}
