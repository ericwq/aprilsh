package main

import (
	"io/ioutil"
	"os"
	"strings"
	"testing"
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
		{"dynamic found", "xfce", []string{"xfce 8 (dynamic)"}},
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
			b, _ := ioutil.ReadAll(r)
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
			[]string{_COMMAND_NAME},
			"xterm-256color",
			[]string{
				"server parameter (User@Server) is mandatory.", "Usage:", _COMMAND_NAME, "Options:",
				"-c, --colors   print the number of colors of terminal",
			},
		},
		{
			"just version",
			[]string{_COMMAND_NAME, "-v"},
			"xterm-256color",
			[]string{
				_COMMAND_NAME, _PACKAGE_STRING,
				"Copyright (c) 2022~2023 wangqi ericwq057[AT]qq[dot]com", "reborn mosh with aprilsh",
			},
		},
		{
			"just help",
			[]string{_COMMAND_NAME, "-h"},
			"xterm-256color",
			[]string{
				"Usage:", _COMMAND_NAME, "Options:",
				"-c, --colors   print the number of colors of terminal",
			},
		},
		{
			"just colors",
			[]string{_COMMAND_NAME, "-c"},
			"xterm-256color",
			[]string{"xterm-256color", "256"},
		},
		{
			"invalid server parameter",
			[]string{_COMMAND_NAME, "invalid", "server", "parameter"},
			"xterm-256color",
			[]string{
				"only one server parameter (User@Server) is allowed.", "Usage:", _COMMAND_NAME, "Options:",
				"-c, --colors   print the number of colors of terminal",
			},
		},
		{
			"malform server parameter no second part",
			[]string{_COMMAND_NAME, "malform@"},
			"xterm-256color",
			[]string{
				"server parameter should be in the form of User@Server", "Usage:", _COMMAND_NAME, "Options:",
				"-c, --colors   print the number of colors of terminal",
			},
		},
		{
			"malform server parameter no first part",
			[]string{_COMMAND_NAME, "@malform"},
			"xterm-256color",
			[]string{
				"server parameter should be in the form of User@Server", "Usage:", _COMMAND_NAME, "Options:",
				"-c, --colors   print the number of colors of terminal",
			},
		},
		{
			"infvalid port number",
			[]string{_COMMAND_NAME, "-p", "7s"},
			"xterm-256color",
			[]string{
				"invalid value \"7s\" for flag -p: parse error", "Usage:", _COMMAND_NAME, "Options:",
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
			initLog()

			// prepare data
			os.Args = v.args
			os.Setenv("TERM", v.term)
			// test main
			main()

			// restore stdout
			w.Close()
			out, _ := ioutil.ReadAll(r)
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
				t.Errorf("#test %s expect %s, got %s\n", v.label, v.expect, result)
			}
		})
	}
}

func TestBuildConfig(t *testing.T) {
	var conf Config
	conf.server = []string{"usr@localhost"}

	_, ok := conf.buildConfig()
	if !ok {
		t.Errorf("#test buildConfig() should return true, got %t\n", ok)
	}

	if conf.user != "usr" || conf.host != "localhost" {
		t.Errorf("#test buildConfig() usert expect %s, got %s\n", "usr", conf.user)
		t.Errorf("#test buildConfig() host expect %s, got %s\n", "localhost", conf.host)
	}
}
