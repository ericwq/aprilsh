// Copyright 2022~2023 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

//go:build darwin

package main

import (
	"io"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/ericwq/aprilsh/frontend"
	"github.com/ericwq/aprilsh/util"
)

var strENOTTY = "operation not supported by device"

func TestBuildConfig_Darwin_DefaultShell(t *testing.T) {
	label := "no SHELL, getShell() return empty string."
	conf0 := &Config{
		version: false, server: false, verbose: 0, desiredIP: "", desiredPort: "",
		locales:     localeFlag{"LANG": "en_US.UTF-8"},
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
				locales:     localeFlag{"LC_ALL": "zh_CN.GB2312", "LANG": "zh_CN.GB2312"},
				commandPath: "", commandArgv: []string{"/bin/sh", "-sh"}, withMotd: false,
			}, // Note GB2312 is not available in apline linux
			Config{
				version: false, server: false, verbose: 0, desiredIP: "", desiredPort: "",
				locales:     localeFlag{},
				commandPath: "/bin/sh", commandArgv: []string{"*sh"}, withMotd: false,
			},
			"UTF-8 locale fail.", false,
		},
	}

	// reset the environment
	util.ClearLocaleVariables()

	for _, v := range tc {
		t.Run(v.label, func(t *testing.T) {

			// save the stdout,stderr and create replaced pipe
			stderr := os.Stderr
			stdout := os.Stdout
			r, w, _ := os.Pipe()
			// replace stdout,stderr with pipe writer
			// alll the output to stdout,stderr is captured
			os.Stderr = w
			os.Stdout = w

			var hint string
			var ok bool

			// validate buildConfig
			hint, ok = v.conf0.buildConfig()
			v.conf0.serve = nil // disable the serve func for testing

			// close pipe writer, get the output
			w.Close()
			out, _ := io.ReadAll(r)
			os.Stderr = stderr
			os.Stdout = stdout
			r.Close()

			if hint != v.hint || ok != v.ok {
				t.Errorf("#test buildConfig got hint=%s, ok=%t, expect hint=%s, ok=%t\n", hint, ok, v.hint, v.ok)
			}

			// validate the output
			expect := []string{frontend.CommandServerName, "needs a UTF-8 native locale to run",
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

func TestBuildConfig(t *testing.T) {
	tc := []struct {
		label string
		conf0 Config
		conf2 Config
		hint  string
		ok    bool
	}{
		{
			"UTF-8 locale",
			Config{
				version: false, server: false, verbose: 0, desiredIP: "", desiredPort: "",
				locales:     localeFlag{"LC_ALL": "en_US.UTF-8", "LANG": "en_US.UTF-8"},
				commandPath: "", commandArgv: []string{"/bin/sh", "-sh"}, withMotd: false,
			},
			Config{
				version: false, server: false, verbose: 0, desiredIP: "", desiredPort: "",
				locales:     localeFlag{"LC_ALL": "en_US.UTF-8", "LANG": "en_US.UTF-8"},
				commandPath: "/bin/sh", commandArgv: []string{"-sh"}, withMotd: false,
			},
			"", true,
		},
		{
			"empty commandArgv",
			Config{
				version: false, server: false, verbose: 0, desiredIP: "", desiredPort: "",
				locales:     localeFlag{"LC_ALL": "en_US.UTF-8"},
				commandPath: "", commandArgv: []string{}, withMotd: false,
			},
			Config{
				version: false, server: false, verbose: 0, desiredIP: "", desiredPort: "",
				locales:     localeFlag{"LC_ALL": "en_US.UTF-8"},
				commandPath: "/bin/zsh", commandArgv: []string{"-zsh"}, withMotd: true,
			},
			// macOS: /bin/zsh
			// alpine: /bin/ash
			"", true,
		},
		// {
		// 	"non UTF-8 locale",
		// 	Config{
		// 		version: false, server: false, verbose: 0, desiredIP: "", desiredPort: "",
		// 		locales: localeFlag{"LC_ALL": "zh_CN.GB2312", "LANG": "zh_CN.GB2312"},
		// 		commandPath: "", commandArgv: []string{"/bin/sh", "-sh"}, withMotd: false,
		// 	}, // TODO GB2312 is not available in apline linux
		// 	Config{
		// 		version: false, server: false, verbose: 0, desiredIP: "", desiredPort: "",
		// 		locales: localeFlag{},
		// 		commandPath: "/bin/sh", commandArgv: []string{"*sh"}, withMotd: false,
		// 	},
		// 	errors.New("UTF-8 locale fail."),
		// },
		{
			"commandArgv is one string",
			Config{
				version: false, server: false, verbose: 0, desiredIP: "", desiredPort: "",
				locales:     localeFlag{"LC_ALL": "en_US.UTF-8", "LANG": "en_US.UTF-8"},
				commandPath: "", commandArgv: []string{"/bin/sh"}, withMotd: false,
			},
			Config{
				version: false, server: false, verbose: 0, desiredIP: "", desiredPort: "",
				locales:     localeFlag{"LC_ALL": "en_US.UTF-8", "LANG": "en_US.UTF-8"},
				commandPath: "/bin/sh", commandArgv: []string{"-sh"}, withMotd: false,
			},
			"", true,
		},
		{
			"missing SSH_CONNECTION",
			Config{
				version: false, server: true, verbose: 0, desiredIP: "", desiredPort: "",
				locales:     localeFlag{"LC_ALL": "en_US.UTF-8", "LANG": "en_US.UTF-8"},
				commandPath: "", commandArgv: []string{"/bin/sh", "-sh"}, withMotd: false,
			},
			Config{
				version: false, server: true, verbose: 0, desiredIP: "", desiredPort: "",
				locales:     localeFlag{"LC_ALL": "en_US.UTF-8", "LANG": "en_US.UTF-8"},
				commandPath: "", commandArgv: []string{"/bin/sh", "-sh"}, withMotd: false,
			},
			"Warning: SSH_CONNECTION not found; binding to any interface.", false,
		},
	}

	for _, v := range tc {
		t.Run(v.label, func(t *testing.T) {

			// set SHELL for empty commandArgv
			if len(v.conf0.commandArgv) == 0 {
				shell := os.Getenv("SHELL")
				defer os.Setenv("SHELL", shell)
				os.Unsetenv("SHELL")
			}

			if v.conf0.server { // unset SSH_CONNECTION, getSSHip will return false
				shell := os.Getenv("SSH_CONNECTION")
				defer os.Setenv("SSH_CONNECTION", shell)
				os.Unsetenv("SSH_CONNECTION")
			}

			// validate buildConfig
			hint, ok := v.conf0.buildConfig()
			v.conf0.serve = nil // disable the serve func for testing

			if hint != v.hint || ok != v.ok {
				t.Errorf("#test buildConfig got hint=%s, ok=%t, expect hint=%s, ok=%t\n", hint, ok, v.hint, v.ok)
			}
			if !reflect.DeepEqual(v.conf0, v.conf2) {
				t.Errorf("#test buildConfig got \n%+v, expect \n%+v\n", v.conf0, v.conf2)
			}
			// reset the environment
			util.ClearLocaleVariables()

			// restore logW
			// logW = log.New(os.Stdout, "WARN: ", log.Ldate|log.Ltime|log.Lshortfile)
		})
	}
}
