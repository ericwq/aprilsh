// Copyright 2022~2024 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package main

import (
	"errors"
	"io"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/creack/pty"
	"github.com/ericwq/aprilsh/frontend"
	"github.com/ericwq/aprilsh/terminfo"
	"golang.org/x/term"
)

func TestPrintColors(t *testing.T) {
	tc := []struct {
		label   string
		termEnv string
		expect  []string
	}{
		// {"wrong termname", "NotExist", []string{}},
		{"normal found", "xterm-256color", []string{"xterm-256color", "256"}},
		{"not found", "xxx", []string{}},
		{"TERM is empty", "", []string{}},
		{"no TERM exit", "-remove", []string{}},
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
			if v.termEnv == "-remove" {
				os.Unsetenv("TERM")
			} else {
				os.Setenv("TERM", v.termEnv)
			}

			defer func() {
				if p := recover(); p != nil {
					os.Setenv("TERM", term)
					os.Stdout = saveStdout
					// fmt.Printf("recover from %q\n", v.label)
				}
				terminfo.Reset()
			}()
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
			[]string{"destination (user@host[:port]) is mandatory."},
		},
		{
			"just version",
			[]string{frontend.CommandClientName, "-version"},
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
				"Usage:", frontend.CommandClientName, "Options:", "-c", "--colors",
				"print the number of terminal color",
			},
		},
		{
			"just colors",
			[]string{frontend.CommandClientName, "-c", "-v"},
			"xterm-256color",
			[]string{"xterm-256color", "256"},
		},
		{
			"invalid target parameter",
			[]string{frontend.CommandClientName, "invalid", "target", "parameter"},
			"xterm-256color",
			[]string{"only one destination (user@host[:port]) is allowed."},
		},
		{
			"destination no second part",
			[]string{frontend.CommandClientName, "malform@"},
			"xterm-256color",
			[]string{"destination should be in the form of user@host[:port]"},
		},
		{
			"destination no first part",
			[]string{frontend.CommandClientName, "@malform"},
			"xterm-256color",
			[]string{"destination should be in the form of user@host[:port]"},
		},
		{
			"infvalid port number",
			[]string{frontend.CommandClientName, "-p", "7s"},
			"xterm-256color",
			[]string{"invalid value \"7s\" for flag -p: parse error"},
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
				t.Errorf("#test expect %s, got \n%s\n", v.expect, result)
			}
		})
	}
}

func TestBuildConfig(t *testing.T) {
	targetMsg := "destination should be in the form of user@host[:port]"
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
				t.Errorf("#test buildConfig() %s expect %q, got %s\n", v.label, v.expect, got)
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

func TestBuildConfig2(t *testing.T) {
	tc := []struct {
		label     string
		conf      *Config
		expectStr string
		ok        bool
	}{
		{"destination without port", &Config{destination: []string{"usr@host"}}, "", true},
		{"destination with port", &Config{destination: []string{"usr@host:23"}}, "", true},
		{
			"destination with wrong port",
			&Config{destination: []string{"usr@host:a23"}}, "please check destination, illegal port number.", false,
		},
	}
	for _, v := range tc {
		t.Run(v.label, func(t *testing.T) {
			got, ok := v.conf.buildConfig()
			if ok != v.ok || got != v.expectStr {
				t.Errorf("%q expect (%s,%t) got (%s,%t)\n", v.label, v.expectStr, v.ok, got, ok)
			}
		})
	}
}

// func TestFetchKey(t *testing.T) {
// 	tc := []struct {
// 		label string
// 		conf  *Config
// 		pwd   string
// 		msg   string
// 	}{
// 		{"wrong host", &Config{user: "ide", host: "wrong", port: 60000}, "password", "dial tcp"},
// 	}
// 	for _, v := range tc {
// 		t.Run(v.label, func(t *testing.T) {
// 			v.conf.pwd = v.pwd
// 			got := v.conf.fetchKey()
// 			if !strings.Contains(got.Error(), v.msg) {
// 				t.Errorf("#test %q expect %q contains %q.\n", v.label, got, v.msg)
// 			}
// 		})
// 	}
// }

func TestGetPassword(t *testing.T) {
	tc := []struct {
		label  string
		conf   *Config
		pwd    string // input
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
				t.Errorf("invalid parameter %s\n", err)
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

func TestGetPasswordFail2(t *testing.T) {
	// store stdout/in, open pts pair
	ptmx, pts, err := pty.Open()
	if err != nil {
		t.Errorf("failed to open pts, %s\n", err)
		return
	}
	saveStdout := os.Stdout
	saveStdin := os.Stdin
	os.Stdout = pts
	os.Stdin = pts

	expect := "hello world"

	// provide the input
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		// make sure we provide input after the getPassword()
		timer := time.NewTimer(time.Duration(2) * time.Millisecond)
		<-timer.C
		ptmx.WriteString(expect + "\n") // \n  is important for getPassword()
	}()

	// waiting for the input
	wg.Add(1)
	var got string
	var err2 error
	go func() {
		defer wg.Done()
		got, err2 = getPassword("password", pts)
	}()
	wg.Wait()

	// close pts paire and restore stdou/stdin
	ptmx.Close()
	pts.Close()
	os.Stdout = saveStdout
	os.Stdin = saveStdin

	// validate, for non-tty input, getPassword return err: inappropriate ioctl for device
	if err2 != nil || got != expect {
		t.Errorf("#test getPassword fail expt %q, got=%q, err=%s\n", expect, got, err)
	}
}

/*
func TestSshAgentFail(t *testing.T) {
	tc := []struct {
		label  string
		env    bool
		expect string
	}{
		{"lack of SSH_AUTH_SOCK", false, "Failed to connect ssh agent."},
	}
	for _, v := range tc {
		t.Run(v.label, func(t *testing.T) {
			old := os.Getenv("SSH_AUTH_SOCK")
			defer os.Setenv("SSH_AUTH_SOCK", old)

			// intercept stdout
			saveStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// clear SSH_AUTH_SOCK
			if !v.env {
				os.Unsetenv("SSH_AUTH_SOCK")
			}
			// run the test
			sshAgent()

			// restore stdout
			w.Close()
			out, _ := io.ReadAll(r)
			os.Stdout = saveStdout
			r.Close()

			got := string(out)
			if !strings.HasPrefix(got, v.expect) {
				t.Errorf("%q expect %q got %q\n", v.label, v.expect, got)
			}
		})
	}
}
*/

func TestErrors(t *testing.T) {
	tc := []struct {
		label  string
		error  error
		expect string
	}{
		{
			"hostkeyChangeError", &hostkeyChangeError{hostname: "some.where"},
			"REMOTE HOST IDENTIFICATION HAS CHANGED",
		},
		{"responseErr without error", &responseError{}, "<nil>"},
		{"responseErr error", &responseError{Msg: "hello", Err: errors.New("world")}, "hello, world"},
	}
	for _, v := range tc {
		t.Run(v.label, func(t *testing.T) {
			got := v.error.Error()
			if !strings.Contains(got, v.expect) {
				t.Errorf("%q expect %q got %q\n", v.label, v.expect, got)
			}
		})
	}
}

func TestGetSignersFail(t *testing.T) {
	tc := []struct {
		label  string
		file   string
		expect string
	}{
		{"file doesn't exist", "/do/es/not/exist", "Unable to read private key"},
		{"is not private key", "/etc/hosts", "Unable to parse private key"},
	}
	for _, v := range tc {
		t.Run(v.label, func(t *testing.T) {
			// intercept stdout
			saveStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// run the test
			getSigner(v.file)

			// restore stdout
			w.Close()
			out, _ := io.ReadAll(r)
			os.Stdout = saveStdout
			r.Close()

			// validate the output
			got := string(out)
			if !strings.Contains(got, v.expect) {
				t.Errorf("%q expect %q got %q\n", v.label, v.expect, got)
			}
		})
	}
}

func TestParseHex(t *testing.T) {
	tc := []struct {
		label  string
		resp   string
		expect string
	}{
		{"regular RGB", "\x1bP1+r524742=382F382F38\x1b\\", "RGB=8/8/8"},
		{"regular TN", "\x1bP1+r544E=57657A5465726D\x1b\\", "TN=WezTerm"},
		{"not XTGETTCAP", "\x1b[?2026$p", ""},
		{"empty string", "", ""},
	}
	for _, v := range tc {
		t.Run(v.label, func(t *testing.T) {
			got := parseHex(v.resp)
			if got != v.expect {
				t.Errorf("%q expect %q got %q\n", v.label, v.expect, got)
			}
		})
	}
}

func TestQueryTerminal_Fail(t *testing.T) {
	tc := []struct {
		label  string
		expect string
		args   []string
	}{
		{
			"query pipe", "set raw mode for",
			[]string{frontend.CommandClientName, "-q"},
		},
	}

	for _, v := range tc {
		t.Run(v.label, func(t *testing.T) {
			// intercept stdout
			saveStdout := os.Stdout
			// r, w, _ := pty.Open()
			r, w, _ := os.Pipe()
			os.Stdout = w

			// prepare data
			os.Args = v.args
			// test main
			main()

			// restore stdout
			w.Close()
			out, _ := io.ReadAll(r)
			os.Stdout = saveStdout
			r.Close()

			// validate the result
			result := string(out)
			if !strings.Contains(result, v.expect) {
				t.Errorf("%q expect %q, got \n%s\n", v.label, v.expect, result)
			}
			// fmt.Print(result)
		})
	}
}

func testQueryTerminal_Main(t *testing.T) {
	tc := []struct {
		label  string
		expect string
		args   []string
	}{
		{
			"query pipe", "set raw mode for std-output error",
			[]string{frontend.CommandClientName, "-q"},
		},
	}

	// unfortunately we can't intercept the stdout for this case.
	// run this test will only improve the test coverage
	for _, v := range tc {
		t.Run(v.label, func(t *testing.T) {
			// prepare data
			os.Args = v.args
			// test main
			main()
		})
	}
}

func TestQueryTerminal_Func(t *testing.T) {
	tc := []struct {
		label  string
		expect int
	}{
		{"query func for pty", 0},
		// https://github.com/golang/go/issues/68851
		// we choose to use pty as best effort for this test
		// {"query func for stdout", 3},
	}

	for _, v := range tc {
		t.Run(v.label, func(t *testing.T) {
			_, pts, _ := pty.Open()
			// pts := os.Stdout

			if tm := term.IsTerminal(int(pts.Fd())); !tm {
				t.Fatalf("%s: %s is not terminal\n", v.label, pts.Name())
			}
			caps, err := queryTerminal(pts, 7)
			if err != nil {
				t.Fatalf("%s expect nil err, got %s\n", v.label, err)
			}

			count := 0
			for i := range caps {
				if caps[i].resp.error == nil {
					count++
				}
				// fmt.Printf("query=%q, response=%q, error=%s\n",
				// 	caps[i].query, caps[i].resp.response, caps[i].resp.error)
			}
			if count < v.expect {
				t.Errorf("%q expect at least %d successful response, got %d successful response\n",
					v.label, v.expect, count)
			}
			pts.Close()
		})
	}
}

func TestQT(t *testing.T) {
	f, _ := os.Open("/dev/tty")
	if tm := term.IsTerminal(int(f.Fd())); !tm {
		t.Errorf("%s is not terminal\n", f.Name())
	}
	f.Close()
}
