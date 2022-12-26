// Copyright 2022 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

//go:build linux

package main

import (
	"fmt"
	"os/exec"
	"os/user"
	"strings"
)

const (
	CODESET = 14
)

func getShell() (string, error) {
	user, err := user.Current()
	if err != nil {
		return "", err
	}

	out, err := exec.Command("getent", "passwd", user.Uid).Output()
	if err != nil {
		return "", err
	}

	ent := strings.Split(strings.TrimSuffix(string(out), "\n"), ":")
	fmt.Printf("#getShell() linux reports: %s\n", ent[6])
	return ent[6], nil
}
