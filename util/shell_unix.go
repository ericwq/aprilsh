// Copyright 2022 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

//go:build linux

package util

import (
	"os/exec"
	"os/user"
	"strings"
)

const (
	CODESET     = 14
	LC_CTYPE    = 0
	LC_NUMERIC  = 1
	LC_TIME     = 2
	LC_COLLATE  = 3
	LC_MONETARY = 4
	LC_MESSAGES = 5
	LC_ALL      = 6
)

var (
	userCurrentTest bool
	execCmdTest     bool
)

func GetShell() (string, error) {
	user, err := user.Current()
	if err != nil || userCurrentTest {
		return "", err
	}

	out, err := exec.Command("getent", "passwd", user.Uid).Output()
	if err != nil || execCmdTest {
		return "", err
	}

	ent := strings.Split(strings.TrimSuffix(string(out), "\n"), ":")
	// fmt.Printf("#getShell() linux reports: %s\n", ent[6])
	return ent[6], nil
}