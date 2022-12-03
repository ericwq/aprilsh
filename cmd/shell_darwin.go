// Copyright 2022 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

//go:build darwin

package main

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"errors"
)

func getShell() (string, error) {
	dir := "Local/Default/Users/" + os.Getenv("USER")
	out, err := exec.Command("dscl", "localhost", "-read", dir, "UserShell").Output()
	if err != nil {
		return "", err
	}

	re := regexp.MustCompile("UserShell: (/[^ ]+)\n")
	matched := re.FindStringSubmatch(string(out))
	shell := matched[1]
	if shell == "" {
		return "", errors.New(fmt.Sprintf("Invalid output: %s", string(out)))
	}

	fmt.Printf("#getShell() darwin reports: %s\n", shell)
	return shell, nil
}
