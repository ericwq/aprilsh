// Copyright 2022 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

//go:build darwin

package main

import (
	"errors"
	"io/ioutil"
	"log"
	"os"
	"reflect"
	"strings"
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

func TestBuildConfigDarwin(t *testing.T) {
	tc := []struct {
		label string
		conf0 Config
		conf2 Config
		err   error
	}{
		{
			"non UTF-8 locale",
			Config{
				version: false, server: false, verbose: 0, desiredIP: "", desiredPort: "",
				locales: localeFlag{"LC_ALL": "zh_CN.GB2312", "LANG": "zh_CN.GB2312"}, color: 0,
				commandPath: "", commandArgv: []string{"/bin/sh", "-sh"}, withMotd: false,
			}, // TODO GB2312 is not available in apline linux
			Config{
				version: false, server: false, verbose: 0, desiredIP: "", desiredPort: "",
				locales: localeFlag{}, color: 0,
				commandPath: "/bin/sh", commandArgv: []string{"*sh"}, withMotd: false,
			},
			errors.New("UTF-8 locale fail."),
		},
	}

	// change the tc[1].conf2 value according to runtime.GOOS
	// switch runtime.GOOS {
	// case "darwin":
	// 	tc[1].conf2.commandArgv = []string{"-zsh"}
	// 	tc[1].conf2.commandPath = "/bin/zsh"
	// case "linux":
	// 	tc[1].conf2.commandArgv = []string{"-ash"}
	// 	tc[1].conf2.commandPath = "/bin/ash"
	// }

	for _, v := range tc {
		t.Run(v.label, func(t *testing.T) {
			// intercept log output
			var b strings.Builder
			logW.SetOutput(&b)

			// set SHELL for empty commandArgv
			if len(v.conf0.commandArgv) == 0 {
				shell := os.Getenv("SHELL")
				defer os.Setenv("SHELL", shell)
				os.Unsetenv("SHELL")
			}

			// validate buildConfig
			err := buildConfig(&v.conf0)
			v.conf0.serve = nil // disable the serve func for testing
			if err != nil {
				if err.Error() != v.err.Error() {
					// if !errors.Is(err, v.err) {
					t.Errorf("#test buildConfig expect %q, got %q\n", v.err, err)
				}
			} else if !reflect.DeepEqual(v.conf0, v.conf2) {
				t.Errorf("#test buildConfig got \n%+v, expect \n%+v\n", v.conf0, v.conf2)
			}
			// reset the environment
			clearLocaleVariables()

			// restore logW
			logW = log.New(os.Stdout, "WARN: ", log.Ldate|log.Ltime|log.Lshortfile)
		})
	}
}
