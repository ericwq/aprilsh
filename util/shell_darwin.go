// Copyright 2022 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

//go:build darwin

package util

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
)

const (
	// https://codeberg.org/FreeBSD/freebsd-src/src/branch/main/include/langinfo.h
	CODESET = 0

	// https://codeberg.org/FreeBSD/freebsd-src/src/branch/main/include/locale.h
	LC_ALL      = 0
	LC_COLLATE  = 1
	LC_CTYPE    = 2
	LC_MONETARY = 3
	LC_NUMERIC  = 4
	LC_TIME     = 5
	LC_MESSAGES = 6
)

func GetShell() (string, error) {
	dir := "Local/Default/Users/" + os.Getenv("USER")
	out, err := exec.Command("dscl", "localhost", "-read", dir, "UserShell").Output()
	if err != nil {
		return "", err
	}

	re := regexp.MustCompile("UserShell: (/[^ ]+)\n")
	matched := re.FindStringSubmatch(string(out))
	var shell string

	if matched != nil {
		shell = matched[1]
	}
	if matched == nil || shell == "" {
		return "", errors.New(fmt.Sprintf("Invalid output: %s", string(out)))
	}

	// fmt.Printf("#getShell() darwin reports: %s\n", shell)
	return shell, nil
}
