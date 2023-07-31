package main

import (
	"errors"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/creack/pty"
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
				"target parameter (User@Server) is mandatory.", "Usage:", _COMMAND_NAME, "Options:",
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
			"invalid target parameter",
			[]string{_COMMAND_NAME, "invalid", "target", "parameter"},
			"xterm-256color",
			[]string{
				"only one target parameter (User@Server) is allowed.", "Usage:", _COMMAND_NAME, "Options:",
				"-c, --colors   print the number of colors of terminal",
			},
		},
		{
			"malform target parameter no second part",
			[]string{_COMMAND_NAME, "malform@"},
			"xterm-256color",
			[]string{
				"target parameter should be in the form of User@Server", "Usage:", _COMMAND_NAME, "Options:",
				"-c, --colors   print the number of colors of terminal",
			},
		},
		{
			"malform target parameter no first part",
			[]string{_COMMAND_NAME, "@malform"},
			"xterm-256color",
			[]string{
				"target parameter should be in the form of User@Server", "Usage:", _COMMAND_NAME, "Options:",
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
	tc := []struct {
		label       string
		target      string
		aprilshKey  string
		predictMode string
		expect      string
	}{
		{"lack of key", "usr@localhost", "", "", _APRILSH_KEY + " environment variable not found."},
		{"has key, lack of mode", "gig@factory", "secret key", "mode", _PREDICTION_DISPLAY + " unknown prediction mode."},
		{"has key, has mode", "vfab@factory", "secret key", "aLwaYs", ""},
	}

	for _, v := range tc {
		t.Run(v.label, func(t *testing.T) {
			var conf Config
			conf.target = []string{v.target}

			// prepare parse result
			idx := strings.Index(v.target, "@")
			host := v.target[idx+1:]
			user := v.target[:idx]

			os.Setenv(_APRILSH_KEY, v.aprilshKey)
			os.Setenv(_PREDICTION_DISPLAY, v.predictMode)

			got, _ := conf.buildConfig()
			if got != v.expect {
				t.Errorf("#test buildConfig() %s expect %q, got %q\n", v.label, v.expect, got)
			}
			if conf.user != user || conf.host != host {
				t.Errorf("#test buildConfig() config.user expect %s, got %s\n", user, conf.user)
				t.Errorf("#test buildConfig() config.host expect %s, got %s\n", host, conf.host)
			}
			if conf.predictMode != strings.ToLower(v.predictMode) {
				t.Errorf("#test buildConfig() conf.predictMode expect %q, got %q\n", v.predictMode, conf.predictMode)
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
		{"wrong host", &Config{user: "ide", host: "wrong", port: 60000}, "password", "can't dial host"},
	}
	for _, v := range tc {
		t.Run(v.label, func(t *testing.T) {
			got := v.conf.fetchKey(v.pwd)
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

			got, err := v.conf.getPassword(pts)

			ptmx.Close()
			pts.Close()

			// restore stdout
			w.Close()
			out, _ := ioutil.ReadAll(r)
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
	conf := &Config{}

	// intercept stdout
	saveStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	got, err := conf.getPassword(r)

	// restore stdout
	w.Close()
	out, _ := ioutil.ReadAll(r)
	os.Stdout = saveStdout
	r.Close()

	// validate, for non-tty input, getPassword return err: inappropriate ioctl for device
	if err == nil {
		t.Errorf("#test getPassword fail expt %q, got=%q, err=%s, out=%s\n", "", got, err, out)
	}
}
