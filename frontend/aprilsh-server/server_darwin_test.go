// Copyright 2022~2023 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

//go:build darwin

package main

import (
	"os"
	"strings"
	"testing"

	"github.com/ericwq/aprilsh/util"
)

func TestBuildConfig_Darwin_DefaultShell(t *testing.T) {
	label := "no SHELL, getShell() return empty string."
	conf0 := &Config{
		version: false, server: false, verbose: 0, desiredIP: "", desiredPort: "",
		locales: localeFlag{"LC_ALL": "en_US.UTF-8"}, color: 0,
		commandPath: "", commandArgv: []string{}, withMotd: false,
	}

	// no SHELL
	shell := os.Getenv("SHELL")
	defer os.Setenv("SHELL", shell)
	os.Unsetenv("SHELL")

	// getShell() return empty string
	user := os.Getenv("USER")
	defer os.Setenv("USER", user)
	os.Unsetenv("USER")

	// validate commandPath == _PATH_BSHELL
	conf0.buildConfig()
	if conf0.commandPath != _PATH_BSHELL {
		t.Errorf("#test buildConfig %q expect %q, got %q\n", label, _PATH_BSHELL, conf0.commandPath)
	}
}

func TestBuildConfig_Darwin_locale(t *testing.T) {
	tc := []struct {
		label string
		conf0 Config
		conf2 Config
		hint  string
		ok    bool
	}{
		{
			"non UTF-8 locale",
			Config{
				version: false, server: false, verbose: 0, desiredIP: "", desiredPort: "",
				locales: localeFlag{"LC_ALL": "zh_CN.GB2312", "LANG": "zh_CN.GB2312"}, color: 0,
				commandPath: "", commandArgv: []string{"/bin/sh", "-sh"}, withMotd: false,
			}, // Note GB2312 is not available in apline linux
			Config{
				version: false, server: false, verbose: 0, desiredIP: "", desiredPort: "",
				locales: localeFlag{}, color: 0,
				commandPath: "/bin/sh", commandArgv: []string{"*sh"}, withMotd: false,
			},
			"UTF-8 locale fail.", false,
		},
	}

	// reset the environment
	util.ClearLocaleVariables()

	for _, v := range tc {
		t.Run(v.label, func(t *testing.T) {

			var hint string
			var ok bool

			testFunc := func() {
				// validate buildConfig
				hint, ok = v.conf0.buildConfig()
				v.conf0.serve = nil // disable the serve func for testing

			}
			out := captureStdoutRun(testFunc)

			if hint != v.hint || ok != v.ok {
				t.Errorf("#test buildConfig got hint=%s, ok=%t, expect hint=%s, ok=%t\n", hint, ok, v.hint, v.ok)
			}

			// validate the output
			expect := []string{_COMMAND_NAME, "needs a UTF-8 native locale to run",
				"Unfortunately, the local environment", "The client-supplied environment"}
			result := string(out)
			found := 0
			for i := range expect {
				if strings.Contains(result, expect[i]) {
					found++
				}
			}
			if found != len(expect) {
				t.Errorf("#test buildConfig() expect %q, got %s\n", expect, result)
			}
		})
	}
}
