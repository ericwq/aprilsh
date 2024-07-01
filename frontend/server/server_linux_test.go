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

	"github.com/creack/pty"
	"github.com/ericwq/aprilsh/util"
	utmps "github.com/ericwq/goutmp"
)

var strENOTTY = "inappropriate ioctl for device"

var (
	index         int
	utmpxMockData []*utmps.Utmpx
)

type mockData struct {
	host  string
	line  string
	usr   string
	id    int
	pid   int
	xtype int
}

func initData(data []mockData) {
	utmpxMockData = make([]*utmps.Utmpx, 0)
	for _, v := range data {
		u := &utmps.Utmpx{}
		u.SetType(v.xtype)
		u.SetHost(v.host)
		u.SetLine(v.line)
		u.SetUser(v.usr)
		u.SetId(v.id)
		u.SetPid(v.pid)
		utmpxMockData = append(utmpxMockData, u)
	}
}

// return utmp mock data
func mockGetRecord() *utmps.Utmpx {
	if 0 <= index && index < len(utmpxMockData) {
		p := utmpxMockData[index]
		index++
		return p
	}
	// fmt.Println("mockGetRecord return nil")
	return nil
}

func TestWarnUnattached(t *testing.T) {
	tc := []struct {
		label      string
		ignoreHost string
		count      int
	}{
		// 666 pts/0 exist, 888 pts/non does not exist, only 666 remains
		{"one match", "apshd:999", 1},
		// 666 pts0 exist, 999 pts/ptmx exist, so 666 and 999 remains
		{"two match", "apshd:888", 2},
	}

	data := []mockData{
		{"apshd:777", "pts/1", "root", 3, 1, utmps.USER_PROCESS},
		{"apshd:888", "pts/non", getCurrentUser(), 7, 1221, utmps.USER_PROCESS},
		{"apshd:666", "pts/0", getCurrentUser(), 0, 1222, utmps.USER_PROCESS},
		// 555 doesn't start with apshd
		{"192.168.0.123 via apshd:555", "pts/0", getCurrentUser(), 0, 1223, utmps.USER_PROCESS},
		{"apshd:999", "pts/ptmx", getCurrentUser(), 2, 1224, utmps.USER_PROCESS},
	}
	initData(data)

	// open pts for test,
	// on some container, if pts/0 is not opened, the test will failed. so we open it.
	ptmx, pts, err := pty.Open()
	if err != nil {
		t.Errorf("test warnUnattached open pts failed, %s\n", err)
	}
	defer func() {
		ptmx.Close()
		pts.Close()
	}()

	for _, v := range tc {
		t.Run(v.label, func(t *testing.T) {
			var out strings.Builder
			setGetRecord(mockGetRecord)
			index = 0
			defer func() {
				setGetRecord(utmps.GetRecord)
			}()

			count := warnUnattached(&out, getCurrentUser(), v.ignoreHost)

			got := out.String()
			if count != v.count {
				t.Errorf("#test warnUnattached() %q expect %d warning, got %d. \n%s\n",
					v.label, v.count, count, got)
			}
		})
	}
}

func TestWarnUnattachedZero(t *testing.T) {
	tc := []struct {
		label      string
		ignoreHost string
		count      int
	}{
		{"zero match", "apshd:888", 0},
	}

	data := []mockData{
		// root user doesn't match current user
		{"apshd:777", "pts/1", "root", 3, 1, utmps.USER_PROCESS},
		// others doesn't start with apshd
		{"192.168.0.123 via apshd:888", "pts/8", getCurrentUser(), 7, 1221, utmps.USER_PROCESS},
		{"192.168.0.123 via apshd:666", "pts/0", getCurrentUser(), 0, 1222, utmps.USER_PROCESS},
		{"192.168.0.123 via apshd:555", "pts/9", getCurrentUser(), 0, 1223, utmps.USER_PROCESS},
		{"192.168.0.123 via apshd:999", "pts/ptmx", getCurrentUser(), 2, 1224, utmps.USER_PROCESS},
	}
	initData(data)
	for _, v := range tc {
		t.Run(v.label, func(t *testing.T) {
			var out strings.Builder
			setGetRecord(mockGetRecord)
			index = 0
			defer func() {
				setGetRecord(utmps.GetRecord)
			}()

			warnUnattached(&out, getCurrentUser(), v.ignoreHost)

			got := out.String()
			// t.Logf("%q\n", got)
			if len(got) != 0 {
				t.Errorf("#test %q expect empty string, got %q\n", v.label, got)
			}
		})
	}
}

func TestBuildConfig(t *testing.T) {
	tc := []struct {
		label string
		hint  string
		conf0 Config
		conf2 Config
		ok    bool
	}{
		{
			"UTF-8 locale", "",
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
			true,
		},
		{
			"empty commandArgv", "",
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
			true,
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
			"commandArgv is one string", "",
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
			true,
		},
		{
			"missing SSH_CONNECTION", "Warning: SSH_CONNECTION not found; binding to any interface.",
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
			false,
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
		})
	}
}

func TestDeviceExist(t *testing.T) {
	tc := []struct {
		label   string
		ptsName string
		got     bool
	}{
		{"pts/0 exist", "pts/0", true},
		// {"pts/1 exist", "pts/1", true},
		{"pts/non doesn't exist", "pts/non", false},
	}

	for _, v := range tc {
		got := utmps.DeviceExists(v.ptsName)
		if got != v.got {
			t.Errorf("%s expect %t, got %t\n", v.label, v.got, got)
		}
	}
}
