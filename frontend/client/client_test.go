// Copyright 2022~2024 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package main

import (
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/creack/pty"
	"github.com/ericwq/aprilsh/frontend"
)

func TestPrintColors(t *testing.T) {
	tc := []struct {
		label  string
		term   string
		expect []string
	}{
		{"lookup terminfo failed", "NotExist", []string{"Dynamic load terminfo failed."}},
		{"TERM is empty", "", []string{"The TERM is empty string."}},
		{"TERM doesn't exit", "-remove", []string{"The TERM doesn't exist."}},
		{"normal found", "xterm-256color", []string{"xterm-256color", "256"}},
		// {"dynamic found", "xfce", []string{"xfce 8 (dynamic)"}},
		{"dynamic not found", "xxx", []string{"Dynamic load terminfo failed."}},
	}

	for _, v := range tc {
		t.Run(v.label, func(t *testing.T) {
			// intercept stdout
			saveStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w
			// save original TERM
			term := os.Getenv("TERM")

			// set TERM according to test case
			if v.term == "-remove" {
				os.Unsetenv("TERM")
			} else {
				os.Setenv("TERM", v.term)
			}

			printColors()

			// restore stdout
			w.Close()
			b, _ := io.ReadAll(r)
			os.Stdout = saveStdout
			r.Close()

			// validate the result
			result := string(b)
			found := 0
			for i := range v.expect {
				if strings.Contains(result, v.expect[i]) {
					found++
				}
			}
			if found != len(v.expect) {
				t.Errorf("#test %s expect %q, got %q\n", v.label, v.expect, result)
			}

			// restore original TERM
			os.Setenv("TERM", term)
		})
	}
}

func TestMainRun_Parameters(t *testing.T) {
	// shutdown after 50ms
	// time.AfterFunc(100*time.Millisecond, func() {
	// 	syscall.Kill(os.Getpid(), syscall.SIGHUP)
	// 	syscall.Kill(os.Getpid(), syscall.SIGTERM)
	// })

	tc := []struct {
		label  string
		args   []string
		term   string
		expect []string
	}{
		{
			"no parameters",
			[]string{frontend.CommandClientName},
			"xterm-256color",
			[]string{
				"destination (user@host[:port]) is mandatory.", "Usage:", frontend.CommandClientName, "Options:",
				"-c, --colors   print the number of colors of terminal",
			},
		},
		{
			"just version",
			[]string{frontend.CommandClientName, "-v"},
			"xterm-256color",
			[]string{
				frontend.CommandClientName, frontend.AprilshPackageName,
				"Copyright (c) 2022~2024 wangqi <ericwq057@qq.com>", "remote shell support intermittent or mobile network.",
			},
		},
		{
			"just help",
			[]string{frontend.CommandClientName, "-h"},
			"xterm-256color",
			[]string{
				"Usage:", frontend.CommandClientName, "Options:",
				"-c, --colors   print the number of colors of terminal",
			},
		},
		{
			"just colors",
			[]string{frontend.CommandClientName, "-c"},
			"xterm-256color",
			[]string{"xterm-256color", "256"},
		},
		{
			"invalid target parameter",
			[]string{frontend.CommandClientName, "invalid", "target", "parameter"},
			"xterm-256color",
			[]string{
				"only one destination (user@host[:port]) is allowed.", "Usage:", frontend.CommandClientName, "Options:",
				"-c, --colors   print the number of colors of terminal",
			},
		},
		{
			"destination no second part",
			[]string{frontend.CommandClientName, "malform@"},
			"xterm-256color",
			[]string{
				"destination should be in the form of user@host[:port]", "Usage:", frontend.CommandClientName, "Options:",
				"-c, --colors   print the number of colors of terminal",
			},
		},
		{
			"destination no first part",
			[]string{frontend.CommandClientName, "@malform"},
			"xterm-256color",
			[]string{
				"destination should be in the form of user@host[:port]", "Usage:", frontend.CommandClientName, "Options:",
				"-c, --colors   print the number of colors of terminal",
			},
		},
		{
			"infvalid port number",
			[]string{frontend.CommandClientName, "-p", "7s"},
			"xterm-256color",
			[]string{
				"invalid value \"7s\" for flag -p: parse error", "Usage:", frontend.CommandClientName, "Options:",
				"-c, --colors   print the number of colors of terminal",
			},
		},
	}

	for _, v := range tc {
		t.Run(v.label, func(t *testing.T) {
			// intercept stdout
			saveStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// prepare data
			os.Args = v.args
			os.Setenv("TERM", v.term)
			// test main
			main()

			// restore stdout
			w.Close()
			out, _ := io.ReadAll(r)
			os.Stdout = saveStdout
			r.Close()

			// validate the result
			result := string(out)
			found := 0
			for i := range v.expect {
				if strings.Contains(result, v.expect[i]) {
					// fmt.Printf("found %s\n", expect[i])
					found++
				}
			}
			if found != len(v.expect) {
				t.Errorf("#test expect %s, got \n%q\n", v.expect, result)
			}
		})
	}
}

func TestBuildConfig(t *testing.T) {
	targetMsg := "target parameter should be in the form of User@Server"
	modeMsg := _PREDICTION_DISPLAY + " unknown prediction mode."
	tc := []struct {
		label       string
		target      string
		predictMode string
		expect      string
		ok          bool
	}{
		{"valid target, empty mode", "usr@localhost", "", "", true},
		{"valid target, lack of mode", "gig@factory", "mode", modeMsg, false},
		{"valid target, valid mode", "vfab@factory", "aLwaYs", "", true},
		{"invalid target", "factory", "", targetMsg, false},
		{"invalid @target", "@factory", "", targetMsg, false},
		{"invalid target@", "factory@", "", targetMsg, false},
	}

	for _, v := range tc {
		t.Run(v.label, func(t *testing.T) {
			var conf Config
			conf.destination = []string{v.target}

			// prepare parse result
			var host string
			var user string
			idx := strings.Index(v.target, "@")
			if idx > 0 && idx < len(v.target)-1 {
				host = v.target[idx+1:]
				user = v.target[:idx]
			}

			os.Setenv(_PREDICTION_DISPLAY, v.predictMode)

			got, ok := conf.buildConfig()
			if got != v.expect {
				t.Errorf("#test buildConfig() %s expect %q, got %q\n", v.label, v.expect, got)
			}
			if conf.user != user || conf.host != host {
				t.Errorf("#test buildConfig() %q config.user expect %s, got %s\n", v.label, user, conf.user)
				t.Errorf("#test buildConfig() %q config.host expect %s, got %s\n", v.label, host, conf.host)
			}
			if conf.predictMode != strings.ToLower(v.predictMode) {
				t.Errorf("#test buildConfig() conf.predictMode expect %q, got %q\n", v.predictMode, conf.predictMode)
			}
			if ok != v.ok {
				t.Errorf("#test buildConfig() expect %t, got %t\n", v.ok, ok)
			}
		})
	}
}

func TestFetchKey(t *testing.T) {
	tc := []struct {
		label string
		conf  *Config
		pwd   string
		msg   string
	}{
		{"wrong host", &Config{user: "ide", host: "wrong", port: 60000}, "password", "dial tcp"},
	}
	for _, v := range tc {
		t.Run(v.label, func(t *testing.T) {
			v.conf.pwd = v.pwd
			got := v.conf.fetchKey()
			if !strings.Contains(got.Error(), v.msg) {
				t.Errorf("#test %q expect %q contains %q.\n", v.label, got, v.msg)
			}
		})
	}
}

func TestGetPassword(t *testing.T) {

	tc := []struct {
		label  string
		conf   *Config
		pwd    string //input
		expect string
	}{
		{"normal get password", &Config{}, "password\n", "password"},
		{"just CR", &Config{}, "\n", ""},
	}
	for _, v := range tc {
		t.Run(v.label, func(t *testing.T) {
			// intercept stdout
			saveStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// get password require pts file.
			ptmx, pts, err := pty.Open()
			if err != nil {
				err = errors.New("invalid parameter")
			}

			// prepare input data
			ptmx.WriteString(v.pwd)

			got, err := getPassword("password", pts)

			ptmx.Close()
			pts.Close()

			// restore stdout
			w.Close()
			out, _ := io.ReadAll(r)
			os.Stdout = saveStdout
			r.Close()

			// validate the result.
			if err != nil {
				t.Errorf("#test %q report %s\n", v.label, err)
			}
			if got != v.expect {
				t.Errorf("#test %q expect %q, got %q. out=%s\n", v.label, v.expect, got, out)
			}

		})
	}
}

func TestGetPasswordFail(t *testing.T) {
	// conf := &Config{}

	// intercept stdout
	saveStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	got, err := getPassword("password", r)

	// restore stdout
	w.Close()
	out, _ := io.ReadAll(r)
	os.Stdout = saveStdout
	r.Close()

	// validate, for non-tty input, getPassword return err: inappropriate ioctl for device
	if err == nil {
		t.Errorf("#test getPassword fail expt %q, got=%q, err=%s, out=%s\n", "", got, err, out)
	}
}
