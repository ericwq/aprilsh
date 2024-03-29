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

func GetShell() (string, error) {
	user, err := user.Current()
	if err != nil {
		return "", err
	}

	return getShell(user)
}

func GetShell4(user *user.User) (string, error) {
	return getShell(user)
}

func getShell(user *user.User) (string, error) {
	out, err := exec.Command("getent", "passwd", user.Uid).Output()
	if err != nil {
		return "", err
	}

	ent := strings.Split(strings.TrimSuffix(string(out), "\n"), ":")
	// fmt.Printf("#getShell() linux reports: %s\n", ent[6])
	return ent[6], nil
}
