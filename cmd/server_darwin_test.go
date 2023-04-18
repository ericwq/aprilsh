// Copyright 2022 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

//go:build darwin

package main

import (
	"io/ioutil"
	"os"
	"testing"
)

func TestDarwinBuildConfig(t *testing.T) {
	label := "no SHELL, getShell() return empty string."
	conf0 := &Config{
		version: false, server: false, verbose: 0, desiredIP: "", desiredPort: "",
		locales: localeFlag{"LC_ALL": "en_US.UTF-8"}, color: 0,
		commandPath: "", commandArgv: []string{}, withMotd: false,
	}
	var err2 error
	err2 = nil

	// no SHELL
	shell := os.Getenv("SHELL")
	defer os.Setenv("SHELL", shell)
	os.Unsetenv("SHELL")

	// getShell() return empty string
	user := os.Getenv("USER")
	defer os.Setenv("USER", user)
	os.Unsetenv("USER")

	// save the stderr and create replaced pipe
	rescueStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// validate commandPath == _PATH_BSHELL
	err := buildConfig(conf0)
	if err != nil && conf0.commandPath != _PATH_BSHELL {
		t.Errorf("#test buildConfig %q expect %q, got %q\n", label, err2, err)
	}
	// reset the environment
	clearLocaleVariables()

	// read and restore the stderr
	w.Close()
	ioutil.ReadAll(r)
	os.Stderr = rescueStderr
}
