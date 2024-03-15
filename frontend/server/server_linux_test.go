// Copyright 2022~2023 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

//go:build linux

package main

import (
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/ericwq/aprilsh/frontend"
	"github.com/ericwq/aprilsh/util"
	utmp "github.com/ericwq/goutmp"
)

var idx = 0

var strENOTTY = "inappropriate ioctl for device"

func mockGetUtmpx() *utmp.Utmpx {
	userName := getCurrentUser()
	rs := []struct {
		Type int16
		User string
		Host string
		Line string
	}{
		{utmp.USER_PROCESS, "root", frontend.CommandServerName + " [777]", "pts/1"},
		{utmp.USER_PROCESS, userName, frontend.CommandServerName + " [888]", "pts/7"},
		{utmp.USER_PROCESS, userName, frontend.CommandServerName + " [666]", "pts/0"},
		{utmp.USER_PROCESS, userName, frontend.CommandServerName + " [999]", "pts/1"},
	}
	// the test requires the following files in /dev/pts directory
	// ls /dev/pts
	// 0     1     2     ptmx

	// if idx out of range, rewind it.
	if idx >= len(rs) {
		idx = 0
		return nil
	}

	u := utmp.Utmpx{}
	u.Type = rs[idx].Type

	b := []byte(rs[idx].User)
	for i := range u.User {
		if i >= len(b) {
			break
		}
		u.User[i] = int8(b[i])
	}

	b = []byte(rs[idx].Host)
	for i := range u.Host {
		if i >= len(b) {
			break
		}
		u.Host[i] = int8(b[i])
	}

	b = []byte(rs[idx].Line)
	for i := range u.Line {
		if i >= len(b) {
			break
		}
		u.Line[i] = int8(b[i])
	}

	// fmt.Printf("#mockGetUtmpx() rs[%d]=%v\n", idx, rs[idx])
	// increase to the next one
	idx++

	// return current one
	return &u
}

func TestWarnUnattached(t *testing.T) {
	// fp = mockGetUtmpx
	util.SetFunc4GetUtmpx(mockGetUtmpx)
	idx = 0
	defer func() {
		// fp = utmp.GetUtmpx
		util.SetFunc4GetUtmpx(utmp.GetUtmpx)
		idx = 0
	}()

	tc := []struct {
		label      string
		ignoreHost string
		count      int
	}{
		// 666 pts/1 exist, 888 pts/7 does not exist, only 666 remains
		{"one match", frontend.CommandServerName + " [999]", 1},
		// 666 pts1 exist, 999 pts/0 exist, so 666 and 999 remains
		{"two matches", frontend.CommandServerName + " [888]", 2},
	}

	for _, v := range tc {
		t.Run(v.label, func(t *testing.T) {
			var out strings.Builder
			warnUnattached(&out, v.ignoreHost)
			got := out.String()
			// t.Logf("%q\n", got)
			count := strings.Count(got, "- ")
			switch count {
			case 0: // warnUnattached found one unattached session
				if strings.Index(got, "detached session on this server") != -1 &&
					v.count != 1 {
					t.Errorf("#test warnUnattached() %q expect %d warning, got 1.\n",
						v.label, v.count)
				}
			default: // warnUnattached found more than one unattached session
				if count != v.count {
					t.Errorf("#test warnUnattached() %q expect %d warning, got %d.\n",
						v.label, v.count, count)
				}
			}
		})
	}
}

// always return nil
func mockGetUtmpx0() *utmp.Utmpx {
	return nil
}

func TestWarnUnattached0(t *testing.T) {
	// fp = mockGetUtmpx0
	util.SetFunc4GetUtmpx(mockGetUtmpx0)
	idx = 0
	defer func() {
		util.SetFunc4GetUtmpx(utmp.GetUtmpx)
		// fp = utmp.GetUtmpx
		idx = 0
	}()
	var out strings.Builder
	warnUnattached(&out, "anything")
	got := out.String()
	if len(got) != 0 {
		t.Logf("%s\n", got)
		t.Errorf("#test warnUnattached() zero match expect 0, got %d\n", len(got))
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
				commandPath: "/bin/sh", commandArgv: []string{"-sh"}, withMotd: true,
				flowControl: _FC_DEF_BASH_SHELL,
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

				// getShell() will fail
				defer func() {
					v.conf0.flowControl = 0
				}()

				v.conf0.flowControl = _FC_DEF_BASH_SHELL
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
