// Copyright 2022~2023 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"os/user"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/creack/pty"
	"github.com/ericwq/aprilsh/frontend"
	"github.com/ericwq/aprilsh/network"
	"github.com/ericwq/aprilsh/statesync"
	"github.com/ericwq/aprilsh/util"
	"golang.org/x/exp/slog"
	"golang.org/x/sys/unix"
)

func TestPrintMotd(t *testing.T) {
	// darwin doesn't has the following motd files, so we add /etc/hosts for testing.
	files := []string{"/run/motd.dynamic", "/var/run/motd.dynamic", "/etc/motd", "/etc/hosts"}

	var output bytes.Buffer

	found := false
	for i := range files {
		output.Reset()
		if printMotd(&output, files[i]) {
			if output.Len() > 0 { // we got and print the file content
				found = true
				break
			}
		}
	}

	// validate the result
	if !found {
		t.Errorf("#test expect found %s, found nothing\n", files)
	}

	output.Reset()

	// creat a .hide file and write long token into it
	fName := ".hide"
	hide, _ := os.Create(fName)
	for i := 0; i < 1025; i++ {
		data := bytes.Repeat([]byte{'s'}, 64)
		hide.Write(data)
	}
	hide.Close()

	if printMotd(&output, fName) {
		t.Errorf("#test printMotd should return false, instead it return true.")
	}

	os.Remove(fName)
}

func TestPrintVersion(t *testing.T) {
	// intercept stdout
	saveStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	// initLog()

	expect := []string{frontend.CommandServerName, "build", "wangqi ericwq057@qq.com"}

	printVersion()

	// restore stdout
	w.Close()
	b, _ := io.ReadAll(r)
	os.Stdout = saveStdout
	r.Close()

	// validate the result
	result := string(b)
	found := 0
	for i := range expect {
		if strings.Contains(result, expect[i]) {
			found++
		}
	}
	if found != len(expect) {
		t.Errorf("#test printVersion expect %q, got %q\n", expect, result)
	}
}

var cmdOptions = "[-s] [--verbose V] [-i LOCALADDR] [-p PORT[:PORT2]] [-l NAME=VALUE] [-- command...]"

func TestPrintUsage(t *testing.T) {
	tc := []struct {
		label  string
		hints  string
		expect []string
	}{
		{"no hint", "", []string{"Usage:", frontend.CommandServerName, cmdOptions}},
		{"some hints", "some hints", []string{"Usage:", frontend.CommandServerName, "some hints", cmdOptions}},
	}

	for _, v := range tc {
		t.Run(v.label, func(t *testing.T) {

			out := captureOutputRun(func() {
				printUsage(v.hints, usage)
			})

			// validate the result
			result := string(out)
			found := 0
			for i := range v.expect {
				if strings.Contains(result, v.expect[i]) {
					found++
				}
			}
			if found != len(v.expect) {
				t.Errorf("#test printUsage expect %s, got %s\n", v.expect, result)
			}
		})
	}
}

func TestChdirHomedir(t *testing.T) {
	// save the current dir
	oldPwd := os.Getenv("PWD")

	// use the HOME
	got := ""
	if !chdirHomedir("") {
		got = os.Getenv("PWD")
		t.Errorf("#test chdirHomedir expect change to home directory, got %s\n", got)
	}

	// validate the PWD
	got = os.Getenv("PWD")
	// fmt.Printf("#test chdirHomedir home=%q\n", got)
	if got == oldPwd {
		t.Errorf("#test chdirHomedir home dir %q, is different from old dir %q\n", got, oldPwd)
	}

	// unset HOME
	os.Unsetenv("HOME")
	// validate the false
	if chdirHomedir("") {
		t.Errorf("#test chdirHomedir return false.\n")
	}

	// use the parameter as HOME
	if chdirHomedir("/does/not/exist") {
		t.Errorf("#test chdirHomedir should return false\n")
	}

	// restore the current dir and PWD
	os.Chdir(oldPwd)
	os.Setenv("PWD", oldPwd)
}

func TestGetHomeDir(t *testing.T) {
	tc := []struct {
		label  string
		env    string
		expect string
	}{
		{"normal case", "/home/aprish", "/home/aprish"},
		{"no HOME case", "", ""}, // for unix anc macOS, no HOME means getHomeDir() return ""
	}

	for _, v := range tc {
		oldHome := os.Getenv("HOME")
		if v.env == "" { // unset HOME env
			os.Unsetenv("HOME")
		} else {
			os.Setenv("HOME", v.env)
		}
		got := getHomeDir()

		if got != v.expect {
			t.Errorf("%s getHomeDir() expect %q got %q\n", v.label, v.expect, got)
		}
		os.Setenv("HOME", oldHome)
	}
}

func TestMotdHushed(t *testing.T) {
	label := "#test motdHushed "
	if motdHushed() != false {
		t.Errorf("%s should report false, got %t\n", label, motdHushed())
	}

	cmd := exec.Command("touch", ".hushlogin")
	if err := cmd.Run(); err != nil {
		t.Errorf("%s create .hushlogin failed, %s\n", label, err)
	}
	if motdHushed() != true {
		t.Errorf("%s should report true, got %t\n", label, motdHushed())
	}

	cmd = exec.Command("rm", ".hushlogin")
	if err := cmd.Run(); err != nil {
		t.Errorf("%s delete .hushlogin failed, %s\n", label, err)
	}
}

func TestMainHelp(t *testing.T) {
	testHelpFunc := func() {
		// prepare data
		os.Args = []string{frontend.CommandServerName, "--help"}
		// test help
		main()
	}

	out := captureOutputRun(testHelpFunc)

	// validate result
	expect := []string{"Usage:", frontend.CommandServerName, cmdOptions}

	// validate the result
	result := string(out)
	found := 0
	for i := range expect {
		if strings.Contains(result, expect[i]) {
			found++
		}
	}
	if found != len(expect) {
		t.Errorf("#test printUsage expect %q, got %q\n", expect, result)
	}
}

// capture the stdout and run the
func captureOutputRun(f func()) []byte {
	// save the stdout,stderr and create replaced pipe
	stderr := os.Stderr
	stdout := os.Stdout
	r, w, _ := os.Pipe()
	// replace stdout,stderr with pipe writer
	// alll the output to stdout,stderr is captured
	os.Stderr = w
	os.Stdout = w
	defer util.Log.Restore()
	util.Log.SetOutput(w)
	util.Log.SetLevel(slog.LevelDebug)

	// os.Args is a "global variable", so keep the state from before the test, and restore it after.
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	f()

	// close pipe writer
	w.Close()
	// get the output
	out, _ := io.ReadAll(r)
	os.Stderr = stderr
	os.Stdout = stdout
	r.Close()

	return out
}

func TestMainVersion(t *testing.T) {

	testHelpFunc := func() {
		// prepare data
		os.Args = []string{frontend.CommandServerName, "--version"}
		// test
		main()

	}

	out := captureOutputRun(testHelpFunc)

	// validate result
	expect := []string{frontend.CommandServerName, "build", "wangqi ericwq057@qq.com", "reborn mosh with aprilsh"}
	result := string(out)
	found := 0
	for i := range expect {
		if strings.Contains(result, expect[i]) {
			found++
		}
	}
	if found != len(expect) {
		t.Errorf("#test printVersion expect %q, got %q\n", expect, result)
	}
}

func TestMainParseFlagsError(t *testing.T) {
	testFunc := func() {
		// prepare data
		os.Args = []string{frontend.CommandServerName, "--foo"}
		// test
		main()
	}

	out := captureOutputRun(testFunc)

	// validate result
	expect := []string{"Hints:", "flag provided but not defined: -foo", "Usage:", "Options:"}
	found := 0
	for i := range expect {
		if strings.Contains(string(out), expect[i]) {
			found++
		}
	}
	if found != len(expect) {
		t.Errorf("#test parserError expect %q, got \n%s\n", expect, out)
	}
}

func TestParseFlagsUsage(t *testing.T) {
	usageArgs := []string{"-help", "-h", "--help"}

	for _, arg := range usageArgs {
		t.Run(arg, func(t *testing.T) {
			conf, output, err := parseFlags("prog", []string{arg})
			if err != flag.ErrHelp {
				t.Errorf("err got %v, want ErrHelp", err)
			}
			if conf != nil {
				t.Errorf("conf got %v, want nil", conf)
			}
			if strings.Index(output, "Usage of") < 0 {
				t.Errorf("output can't find \"Usage of\": %q", output)
			}
		})
	}
}

func TestMainRun(t *testing.T) {
	tc := []struct {
		label  string
		args   []string
		expect []string
	}{
		{"run main and killed by signal",
			[]string{frontend.CommandServerName, "-verbose", "0", "-locale",
				"LC_ALL=en_US.UTF-8", "-p", "6100", "--", "/bin/sh", "-sh"},
			[]string{frontend.CommandServerName, "start listening on", "buildVersion",
				/* "got signal: SIGHUP",  */ "got signal: SIGTERM or SIGINT",
				"stop listening", "6100"}},
		{"main killed by -a", // auto stop after 1 second
			[]string{frontend.CommandServerName, "-verbose", "1", "-auto", "1", "-locale",
				"LC_ALL=en_US.UTF-8", "-p", "6200", "--", "/bin/sh", "-sh"},
			[]string{frontend.CommandServerName, "start listening on", "buildVersion",
				"stop listening", "6200"}},
		{"main killed by -a, write to syslog", // auto stop after 1 second
			[]string{frontend.CommandServerName, "-verbose", "514", "-auto", "1", "-locale",
				"LC_ALL=en_US.UTF-8", "-p", "6300", "--", "/bin/sh", "-sh"},
			[]string{}}, // log write to syslog, we can't get anything
	}

	for _, v := range tc {

		if strings.Contains(v.label, "by signal") {
			// shutdown after 15ms
			time.AfterFunc(time.Duration(15)*time.Millisecond, func() {
				util.Log.Debug("#test kill process by signal")
				syscall.Kill(os.Getpid(), syscall.SIGTERM)
				// syscall.Kill(os.Getpid(), syscall.SIGHUP)
			})
		}

		testFunc := func() {
			os.Args = v.args
			main()
		}

		out := captureOutputRun(testFunc)

		// validate the result from printWelcome
		result := string(out)
		found := 0
		for i := range v.expect {
			if strings.Contains(result, v.expect[i]) {
				// fmt.Printf("found %s\n", expect[i])
				found++
			}
		}
		if found != len(v.expect) {
			t.Errorf("#test expect %q, got %s\n", v.expect, result)
		}
		// fmt.Printf("###\n%s\n###\n", string(out))
	}
}

func TestMainBuildConfigFail(t *testing.T) {
	testFunc := func() {
		// prepare parameter
		os.Args = []string{frontend.CommandServerName, "-locale", "LC_ALL=en_US.UTF-8",
			"-p", "6100", "--", "/bin/sh", "-sh"}
		// test
		main()
	}

	// prepare for buildConfig fail
	buildConfigTest = true
	out := captureOutputRun(testFunc)

	// restore the condition
	buildConfigTest = false

	// validate the result
	expect := []string{"Usage:", "Hints:", "UTF-8 locale fail.", "Options:"}
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
}

func TestParseFlagsCorrect(t *testing.T) {
	tc := []struct {
		args []string
		conf Config
	}{
		{
			[]string{"-locale", "ALL=en_US.UTF-8", "-l", "LANG=UTF-8"},
			Config{
				version: false, server: false, verbose: 0, desiredIP: "", desiredPort: "60000",
				locales:     localeFlag{"ALL": "en_US.UTF-8", "LANG": "UTF-8"},
				commandPath: "", commandArgv: []string{}, withMotd: false,
			},
		},
		{
			[]string{"--", "/bin/sh", "-sh"},
			Config{
				version: false, server: false, verbose: 0, desiredIP: "", desiredPort: "60000",
				locales:     localeFlag{},
				commandPath: "", commandArgv: []string{"/bin/sh", "-sh"}, withMotd: false,
			},
		},
		{
			[]string{"--", ""},
			Config{
				version: false, server: false, verbose: 0, desiredIP: "", desiredPort: "60000",
				locales:     localeFlag{},
				commandPath: "", commandArgv: []string{""}, withMotd: false,
			},
		},
	}

	for _, v := range tc {
		t.Run(strings.Join(v.args, " "), func(t *testing.T) {
			conf, output, err := parseFlags("prog", v.args)
			if err != nil {
				t.Errorf("err got %v, want nil", err)
			}
			if output != "" {
				t.Errorf("output got %q, want empty", output)
			}
			if !reflect.DeepEqual(*conf, v.conf) {
				t.Logf("#test parseFlags got commandArgv=%+v\n", conf.commandArgv)
				t.Errorf("conf got \n%+v, want \n%+v", *conf, v.conf)
			}
		})
	}
}

func TestGetShell(t *testing.T) {
	tc := []struct {
		label  string
		expect string
	}{
		{"get unix shell from cmd", "fill later"},
	}
	switch runtime.GOOS {
	case "darwin":
		tc[0].expect = "/bin/zsh"
	case "linux":
		tc[0].expect = "/bin/ash"
	}

	for _, v := range tc {
		if got, _ := util.GetShell(); got != v.expect {
			if got != v.expect {
				t.Errorf("#test getShell() %s expect %q, got %q\n", v.label, v.expect, got)
			}
		}
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

	// change the tc[1].conf2 value according to runtime.GOOS
	switch runtime.GOOS {
	case "darwin":
		tc[1].conf2.commandArgv = []string{"-zsh"}
		tc[1].conf2.commandPath = "/bin/zsh"
		// case "linux":
		// 	tc[1].conf2.commandArgv = []string{"-ash"}
		// 	tc[1].conf2.commandPath = "/bin/ash"
	}

	for _, v := range tc {
		t.Run(v.label, func(t *testing.T) {

			// set SHELL for empty commandArgv
			if len(v.conf0.commandArgv) == 0 {
				shell := os.Getenv("SHELL")
				defer os.Setenv("SHELL", shell)
				os.Unsetenv("SHELL")

				if runtime.GOOS == "linux" {
					// getShell() will fail
					defer func() {
						userCurrentTest = false
						getShellTest = false
					}()

					getShellTest = true
					userCurrentTest = false
				}
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

func TestParseFlagsError(t *testing.T) {
	tests := []struct {
		args   []string
		errstr string
	}{
		{[]string{"-foo"}, "flag provided but not defined"},
		// {[]string{"-color", "joe"}, "invalid value"},
		{[]string{"-locale", "a=b=c"}, "malform locale parameter"},
	}

	for _, tt := range tests {
		t.Run(strings.Join(tt.args, " "), func(t *testing.T) {
			conf, output, err := parseFlags("prog", tt.args)
			if conf != nil {
				t.Errorf("conf got %v, want nil", conf)
			}
			if strings.Index(err.Error(), tt.errstr) < 0 {
				t.Errorf("err got %q, want to find %q", err.Error(), tt.errstr)
			}
			if strings.Index(output, "Usage of prog") < 0 {
				t.Errorf("output got %q", output)
			}
		})
	}
}

// func TestMainParameters(t *testing.T) {
// 	// flag is a global variable, reset it before test
// 	flag.CommandLine = flag.NewFlagSet("TestMainParameters", flag.ExitOnError)
// 	testParaFunc := func() {
// 		// prepare data
// 		os.Args = []string{COMMAND_NAME, "--", "/bin/sh","-sh"} //"-l LC_ALL=en_US.UTF-8", "--"}
// 		// test
// 		main()
// 	}
//
// 	out := captureStdoutRun(testParaFunc)
//
// 	// validate result
// 	expect := []string{"main", "commandPath=", "commandArgv=", "withMotd=", "locales=", "color="}
// 	result := string(out)
// 	found := 0
// 	for i := range expect {
// 		if strings.Contains(result, expect[i]) {
// 			found++
// 		}
// 	}
// 	if found != len(expect) {
// 		t.Errorf("#test main() expect %s, got %s\n", expect, result)
// 	}
// }

func TestMainServerPortrangeError(t *testing.T) {
	testFunc := func() {
		os.Args = []string{frontend.CommandServerName, "-s", "-p=3a"}
		os.Setenv("SSH_CONNECTION", "172.17.0.1 58774 172.17.0.2 22")
		main()
	}

	out := captureOutputRun(testFunc)
	// validate port range check
	expect := "Bad UDP port"
	got := string(out)
	if !strings.Contains(got, expect) {
		t.Errorf("#test --port should contains %q, got %s\n", expect, got)
	}
}

func TestGetSSHip(t *testing.T) {
	tc := []struct {
		label  string
		env    string
		expect string
		ok     bool
	}{
		{"no env variable", "", "Warning: SSH_CONNECTION not found; binding to any interface.", false},
		{"ipv4 address", "172.17.0.1 58774 172.17.0.2 22", "172.17.0.2", true},
		{"malform variable", " 1 2 3 4",
			"Warning: Could not parse SSH_CONNECTION; binding to any interface.", false},
		{"ipv6 address", "fe80::14d5:1215:f8c9:11fa%en0 42000 fe80::aede:48ff:fe00:1122%en5 22",
			"fe80::aede:48ff:fe00:1122%en5", true},
		{"ipv4 mapped address", "::FFFF:172.17.0.1 42200 ::FFFF:129.144.52.38 22", "129.144.52.38", true},
	}

	for _, v := range tc {

		os.Setenv("SSH_CONNECTION", v.env)
		got, ok := getSSHip()
		if got != v.expect || ok != v.ok {
			t.Errorf("%q expect %q, got %q, ok=%t\n", v.label, v.expect, got, ok)
		}
	}
}

func TestGetShellNameFrom(t *testing.T) {
	tc := []struct {
		label     string
		shellPath string
		shellName string
	}{
		{"normal", "/bin/sh", "-sh"},
		{"no slash sign", "noslash", "-noslash"},
	}

	for _, v := range tc {
		got := getShellNameFrom(v.shellPath)
		if got != v.shellName {
			t.Errorf("%q expect %q, got %q\n", v.label, v.shellName, got)
		}
	}
}

func TestGetTimeFrom(t *testing.T) {
	tc := []struct {
		lable      string
		key, value string
		expect     int64
	}{
		{"positive int64", "ENV1", "123", 123},
		{"malform int64", "ENV2", "123a", 0},
		{"negative int64", "ENV3", "-123", 0},
	}

	// save the stdout and create replaced pipe
	rescueStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	// initLog()

	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	for _, v := range tc {
		os.Setenv(v.key, v.value)

		got := getTimeFrom(v.key, 0)
		if got != v.expect {
			t.Errorf("%s expct %d, got %d\n", v.lable, v.expect, got)
		}
	}

	// read and restore the stdout
	w.Close()
	io.ReadAll(r)
	os.Stdout = rescueStdout
}

/*
	func testPTY() error {
		// Create arbitrary command.
		c := exec.Command("bash")

		// Start the command with a pty.
		ptmx, err := pty.Start(c)
		if err != nil {
			return err
		}
		// Make sure to close the pty at the end.
		defer func() { _ = ptmx.Close() }() // Best effort.

		// Handle pty size.
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, syscall.SIGWINCH)
		go func() {
			for range ch {
				if err := pty.InheritSize(os.Stdin, ptmx); err != nil {
					log.Printf("error resizing pty: %s", err)
				}
			}
		}()
		ch <- syscall.SIGWINCH                        // Initial resize.
		defer func() { signal.Stop(ch); close(ch) }() // Cleanup signals when done.

		// Set stdin in raw mode.
		oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
		if err != nil {
			panic(err)
		}
		defer func() { _ = term.Restore(int(os.Stdin.Fd()), oldState) }() // Best effort.

		// Copy stdin to the pty and the pty to stdout.
		// NOTE: The goroutine will keep reading until the next keystroke before returning.
		go func() { _, _ = io.Copy(ptmx, os.Stdin) }()
		_, _ = io.Copy(os.Stdout, ptmx)

		return nil
	}
*/

func TestMainSrvStart(t *testing.T) {
	tc := []struct {
		label    string
		pause    int    // pause between client send and read
		resp     string // response client read
		shutdown int    // pause before shutdown message
		conf     Config
	}{
		{
			"start normally", 20, frontend.AprilshMsgOpen + "7101,This is the mock key\n", 50,
			Config{
				version: false, server: true, verbose: 0, desiredIP: "", desiredPort: "7100",
				locales:     localeFlag{"LC_ALL": "en_US.UTF-8", "LANG": "en_US.UTF-8"},
				commandPath: "/bin/sh", commandArgv: []string{"/bin/sh"}, withMotd: false,
			},
		},
	}

	for _, v := range tc {
		t.Run(v.label, func(t *testing.T) {
			// intercept stdout
			// saveStdout := os.Stdout
			// r, w, _ := os.Pipe()
			// os.Stdout = w

			// init log
			defer util.Log.Restore()
			util.Log.SetLevel(slog.LevelDebug)
			util.Log.SetOutput(io.Discard)
			// util.Log.SetOutput(os.Stderr)

			// v.conf.serve = mockServe
			srv := newMainSrv(&v.conf, mockRunWorker)

			// send shutdown message after some time
			timer1 := time.NewTimer(time.Duration(v.shutdown) * time.Millisecond)
			go func() {
				<-timer1.C
				// fmt.Printf("#test start PID:%d\n", os.Getpid())
				// all the go test run in the same process
				// syscall.Kill(os.Getpid(), syscall.SIGHUP)
				// syscall.Kill(os.Getpid(), syscall.SIGTERM)
				srv.downChan <- true
				// stop the worker correctly, because mockRunWorker2 failed to
				// do it on purpose.
				srv.exChan <- fmt.Sprintf("%d", srv.maxPort)
			}()

			srv.start(&v.conf)

			// mock client operation
			resp := mockClient(v.conf.desiredPort, v.pause, frontend.AprilshMsgOpen)
			if resp != v.resp {
				t.Errorf("#test run expect %q got %q\n", v.resp, resp)
			}

			srv.wait()

			// restore stdout
			// w.Close()
			// io.ReadAll(r)
			// os.Stdout = saveStdout
			// r.Close()
		})
	}
}

func TestStartFail(t *testing.T) {
	tc := []struct {
		label  string
		pause  int    // pause between client send and read
		resp   string // response client read
		finish int    // pause before shutdown message
		conf   Config
	}{
		{
			"illegal port", 20, "", 50,
			Config{
				version: false, server: true, verbose: 0, desiredIP: "", desiredPort: "7000a",
				locales:     localeFlag{"LC_ALL": "en_US.UTF-8", "LANG": "en_US.UTF-8"},
				commandPath: "/bin/sh", commandArgv: []string{"/bin/sh"}, withMotd: false,
			},
		},
	}

	for _, v := range tc {
		t.Run(v.label, func(t *testing.T) {
			// intercept logW
			var b strings.Builder
			// logW.SetOutput(&b)
			defer util.Log.Restore()
			util.Log.SetOutput(&b)
			util.Log.SetLevel(slog.LevelDebug)

			m := newMainSrv(&v.conf, mockRunWorker)

			// defer func() {
			// 	logW = log.New(os.Stdout, "WARN: ", log.Ldate|log.Ltime|log.Lshortfile)
			// }()

			// start mainserver
			m.start(&v.conf)
			// fmt.Println("#test start fail!")

			// validate result: result contains WARN and COMMAND_NAME
			expect := []string{"WARN", "listen failed"}
			result := b.String()
			found := 0
			for i := range expect {
				if strings.Contains(result, expect[i]) {
					found++
				}
			}
			if found != 2 {
				t.Errorf("#test start() expect %q, got %q\n", expect, result)
			}
		})
	}
}

// the mock runWorker send the key, pause some time and close the
// worker by send finish message
func mockRunWorker(conf *Config, exChan chan string, whChan chan *workhorse) error {
	// send the mock key
	// fmt.Println("#mockRunWorker send mock key to run().")
	exChan <- "This is the mock key"

	// pause some time
	time.Sleep(time.Duration(2) * time.Millisecond)

	whChan <- &workhorse{}

	// notify the server
	// fmt.Println("#mockRunWorker finish run().")
	exChan <- conf.desiredPort
	return nil
}

// the mock runWorker send the key, pause some time and try to close the
// worker by send wrong finish message: port+"x"
func mockRunWorker2(conf *Config, exChan chan string, whChan chan *workhorse) error {
	// send the mock key
	exChan <- "mock key from mockRunWorker2"

	// pause some time
	time.Sleep(time.Duration(2) * time.Millisecond)

	// fail to stop the worker on purpose
	exChan <- conf.desiredPort + "x"

	whChan <- &workhorse{}

	return nil
}

// mock client connect to the port, send handshake message, pause some time
// return the response message.
func mockClient(port string, pause int, action string, ex ...string) string {
	server_addr, _ := net.ResolveUDPAddr("udp", "localhost:"+port)
	local_addr, _ := net.ResolveUDPAddr("udp", "localhost:0")
	conn, _ := net.DialUDP("udp", local_addr, server_addr)

	defer conn.Close()

	// send handshake message based on action & port
	var txbuf []byte
	switch action {
	case frontend.AprilshMsgOpen:
		switch len(ex) {
		case 0:
			txbuf = []byte(frontend.AprilshMsgOpen + "xterm,mock@client")
		case 1:
			// the request missing the ','
			txbuf = []byte(fmt.Sprintf("%s%s", frontend.AprilshMsgOpen, ex[0]))
		}
	case frontend.AprishMsgClose:
		p, _ := strconv.Atoi(port)
		if len(ex) == 0 {
			txbuf = []byte(fmt.Sprintf("%s%d", frontend.AprishMsgClose, p+1))
		} else if len(ex) == 2 {
			txbuf = []byte(fmt.Sprintf("%s%d", "unknow header:", p+1)) // 2 parameters: unknow header
		} else if len(ex) == 1 {
			p2, err := strconv.Atoi(ex[0])
			if err == nil {
				txbuf = []byte(fmt.Sprintf("%s%d", frontend.AprishMsgClose, p2)) // 1 digital parameter: wrong port
			} else {
				txbuf = []byte(fmt.Sprintf("%s%s", frontend.AprishMsgClose, ex[0])) // 1 str parameter: malform port
			}
		}
	}

	_, err := conn.Write(txbuf)
	// fmt.Printf("#mockClient send %q to server: %v from %v\n", txbuf, server_addr, conn.LocalAddr())
	if err != nil {
		fmt.Printf("#mockClient send %s, error %s\n", string(txbuf), err)
	}

	// pause some time
	time.Sleep(time.Duration(pause) * time.Millisecond)

	// read the response
	rxbuf := make([]byte, 512)
	n, _, err := conn.ReadFromUDP(rxbuf)

	// fmt.Printf("#mockClient read %q from server: %v\n", rxbuf[0:n], server_addr)
	return string(rxbuf[0:n])
}

func TestPrintWelcome(t *testing.T) {
	// open pts master and slave first.
	pty, tty, err := pty.Open()
	if err != nil {
		t.Errorf("#test printWelcome Open %s\n", err)
	}

	// clean pts fd
	defer func() {
		if err != nil {
			pty.Close()
			tty.Close()
		}
	}()

	// pty master doesn't support IUTF8
	flag, err := util.CheckIUTF8(int(pty.Fd()))
	if flag {
		t.Errorf("#test printWelcome master got %t, expect %t\n", flag, false)
	}

	expect := []string{frontend.CommandServerName, "start listening on",
		"buildVersion", "Warning: termios IUTF8 flag not defined.",
	}

	tc := []struct {
		label string
		tty   *os.File
	}{
		{"tty doesn't support IUTF8 flag", pty},
		{"tty failed with checkIUTF8", os.Stdin},
	}

	for _, v := range tc {
		// intercept stdout
		saveStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w
		// initLog()
		defer util.Log.Restore()
		util.Log.SetLevel(slog.LevelDebug)
		util.Log.SetOutput(w)

		printWelcome(os.Getpid(), 6000, v.tty)

		// restore stdout
		w.Close()
		b, _ := io.ReadAll(r)
		os.Stdout = saveStdout
		r.Close()

		// validate the result
		result := string(b)
		found := 0
		for i := range expect {
			if strings.Contains(result, expect[i]) {
				found++
			}
		}
		if found != len(expect) {
			t.Errorf("#test printWelcome expect %q, got %s\n", expect, result)
		}
	}
}

func TestListenFail(t *testing.T) {
	tc := []struct {
		label  string
		port   string
		repeat bool // if true, will listen twice.
	}{
		{"illegal port number", "22a", false},
		{"port already in use", "60001", true}, // 60001 is the docker port on macOS
	}
	for _, v := range tc {
		conf := &Config{desiredPort: v.port}
		s := newMainSrv(conf, mockRunWorker)

		var e error
		e = s.listen(conf)
		// fmt.Printf("#test %q got 1st error: %q\n", v.label, e)
		if v.repeat {
			e = s.listen(conf)
			// fmt.Printf("#test %q got 2nd error: %q\n", v.label, e)
		}

		// check the error does happens
		if e == nil {
			t.Errorf("#test %q expect error return, got nil\n", v.label)
		}

		// close the listen port
		if v.repeat {
			s.exChan <- conf.desiredPort
		}
	}
}

func TestRunFail(t *testing.T) {
	tc := []struct {
		label  string
		pause  int    // pause between client send and read
		resp   string // response client read
		finish int    // pause before shutdown message
		conf   Config
	}{
		{
			"worker failed with wrong port number", 20, frontend.AprilshMsgOpen + "7101,mock key from mockRunWorker2\n", 30,
			Config{
				version: false, server: true, verbose: 0, desiredIP: "", desiredPort: "7100",
				locales:     localeFlag{"LC_ALL": "en_US.UTF-8", "LANG": "en_US.UTF-8"},
				commandPath: "/bin/sh", commandArgv: []string{"/bin/sh"}, withMotd: false,
			},
		},
	}

	for _, v := range tc {
		t.Run(v.label, func(t *testing.T) {
			// intercept stdout
			saveStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w
			// initLog()
			defer util.Log.Restore()
			util.Log.SetLevel(slog.LevelDebug)
			util.Log.SetOutput(w)

			srv := newMainSrv(&v.conf, mockRunWorker2)

			// send shutdown message after some time
			timer1 := time.NewTimer(time.Duration(v.finish) * time.Millisecond)
			go func() {
				<-timer1.C
				// prepare to shudown the mainSrv
				// syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
				srv.downChan <- true
				// stop the worker correctly, because mockRunWorker2 failed to
				// do it on purpose.
				port, _ := strconv.Atoi(v.conf.desiredPort)
				srv.exChan <- fmt.Sprintf("%d", port+1)
			}()
			// fmt.Println("#test start timer for shutdown")

			srv.start(&v.conf)

			// mock client operation
			resp := mockClient(v.conf.desiredPort, v.pause, frontend.AprilshMsgOpen)

			// validate the result.
			if resp != v.resp {
				t.Errorf("#test run expect %q got %q\n", v.resp, resp)
			}

			srv.wait()

			// restore stdout
			w.Close()
			io.ReadAll(r)
			os.Stdout = saveStdout
			r.Close()
		})
	}

	// test case for run() without connection

	srv2 := &mainSrv{}
	srv2.run(&Config{})
}

func TestRunFail2(t *testing.T) {
	tc := []struct {
		label  string
		pause  int    // pause between client send and read
		resp   string // response client read
		finish int    // pause before shutdown message
		conf   Config
	}{
		{
			"read udp error", 20, "7101,This is the mock key", 50,
			Config{
				version: false, server: true, verbose: 0, desiredIP: "", desiredPort: "7100",
				locales:     localeFlag{"LC_ALL": "en_US.UTF-8", "LANG": "en_US.UTF-8"},
				commandPath: "/bin/sh", commandArgv: []string{"/bin/sh"}, withMotd: false,
			},
		},
	}

	for _, v := range tc {
		t.Run(v.label, func(t *testing.T) {
			// intercept stdout
			saveStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w
			// initLog()
			defer util.Log.Restore()
			util.Log.SetLevel(slog.LevelDebug)
			util.Log.SetOutput(w)

			srv := newMainSrv(&v.conf, mockRunWorker)

			// send shutdown message after some time
			timer1 := time.NewTimer(time.Duration(v.finish) * time.Millisecond)
			go func() {
				<-timer1.C
				srv.downChan <- true
				// syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
			}()
			// fmt.Println("#test start timer for shutdown")

			srv.start(&v.conf)

			// close the connection, this will cause read error: use of closed network connection.
			srv.conn.Close()

			srv.wait()

			// restore stdout
			w.Close()
			io.ReadAll(r)
			os.Stdout = saveStdout
			r.Close()
		})
	}
}

func TestMaxPortLimit(t *testing.T) {
	tc := []struct {
		label        string
		maxPortLimit int
		pause        int    // pause between client send and read
		resp         string // response client read
		shutdownTime int    // pause before shutdown message
		conf         Config
	}{
		{
			"run() over max port", 0, 20, "over max port limit", 50,
			Config{
				version: false, server: true, verbose: 0, desiredIP: "", desiredPort: "7700",
				locales:     localeFlag{"LC_ALL": "en_US.UTF-8", "LANG": "en_US.UTF-8"},
				commandPath: "/bin/sh", commandArgv: []string{"/bin/sh"}, withMotd: false,
			},
		},
	}

	for _, v := range tc {
		t.Run(v.label, func(t *testing.T) {
			// intercept stdout
			defer util.Log.Restore()
			util.Log.SetLevel(slog.LevelDebug)
			// util.Log.SetOutput(os.Stderr)
			util.Log.SetOutput(io.Discard)

			// init mainSrv and workers
			m := newMainSrv(&v.conf, runWorker)

			// save maxPortLimit
			old := maxPortLimit
			maxPortLimit = v.maxPortLimit

			// send shutdown message after some time
			timer1 := time.NewTimer(time.Duration(v.shutdownTime) * time.Millisecond)
			go func() {
				<-timer1.C
				m.downChan <- true
			}()

			// start mainserver
			m.start(&v.conf)

			// mock client operation
			resp := mockClient(v.conf.desiredPort, v.pause, frontend.AprilshMsgOpen)

			m.wait()

			if !strings.Contains(resp, v.resp) {
				t.Errorf("%q expect response %q, got %q\n ", v.label, v.resp, resp)
			}

			// restore maxPortLimit
			maxPortLimit = old
		})
	}
}

func TestMalformRequest(t *testing.T) {
	tc := []struct {
		label        string
		pause        int    // pause between client send and read
		resp         string // response client read
		shutdownTime int    // pause before shutdown message
		conf         Config
	}{
		{
			"run() malform request", 20, "malform request", 50,
			Config{
				version: false, server: true, verbose: 0, desiredIP: "", desiredPort: "7700",
				locales:     localeFlag{"LC_ALL": "en_US.UTF-8", "LANG": "en_US.UTF-8"},
				commandPath: "/bin/sh", commandArgv: []string{"/bin/sh"}, withMotd: false,
			},
		},
	}

	for _, v := range tc {
		// intercept stdout
		defer util.Log.Restore()
		util.Log.SetLevel(slog.LevelDebug)
		// util.Log.SetOutput(os.Stderr)
		util.Log.SetOutput(io.Discard)

		// init mainSrv and workers
		m := newMainSrv(&v.conf, runWorker)

		// send shutdown message after some time
		timer1 := time.NewTimer(time.Duration(v.shutdownTime) * time.Millisecond)
		go func() {
			<-timer1.C
			syscall.Kill(os.Getpid(), syscall.SIGHUP) // add SIGHUP test condition
			time.Sleep(5 * time.Millisecond)
			m.downChan <- true
		}()

		// start mainserver
		m.start(&v.conf)

		// mock client operation
		resp := mockClient(v.conf.desiredPort, v.pause, frontend.AprilshMsgOpen, "extraParam")

		m.wait()

		if !strings.Contains(resp, v.resp) {
			t.Errorf("%q expect response %q, got %q\n ", v.label, v.resp, resp)
		}
	}
}

func mockServe(ptmx *os.File, pw *io.PipeWriter, terminal *statesync.Complete, x chan bool,
	network *network.Transport[*statesync.Complete, *statesync.UserStream],
	networkTimeout int64, networkSignaledTimeout int64) error {
	time.Sleep(20 * time.Millisecond)
	x <- true
	return nil
}

// the mock runWorker send empty key, pause some time and close the worker
func failRunWorker(conf *Config, exChan chan string, whChan chan *workhorse) error {
	// send the empty key
	// fmt.Println("#mockRunWorker send mock key to run().")
	exChan <- ""

	// pause some time
	time.Sleep(time.Duration(2) * time.Millisecond)

	// notify this worker is done
	defer func() {
		exChan <- conf.desiredPort
	}()

	whChan <- &workhorse{}
	return errors.New("failed worker.")
}

func TestRunWorkerKillSignal(t *testing.T) {
	tc := []struct {
		label  string
		pause  int    // pause between client send and read
		resp   string // response client read
		finish int    // pause before shutdown message
		conf   Config
	}{
		{
			"runWorker stopped by signal kill", 10, frontend.AprilshMsgOpen + "7101,", 50,
			Config{
				version: false, server: true, verbose: _VERBOSE_SKIP_READ_PIPE, desiredIP: "", desiredPort: "7100",
				locales:     localeFlag{"LC_ALL": "en_US.UTF-8", "LANG": "en_US.UTF-8"},
				commandPath: "/bin/sh", commandArgv: []string{"-sh"}, withMotd: true,
			},
		},
	}

	for _, v := range tc {
		t.Run(v.label, func(t *testing.T) {
			// defer util.Log.Restore()
			// util.Log.SetLevel(slog.LevelDebug)
			// util.Log.SetOutput(os.Stdout)

			// intercept stdout
			saveStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			defer util.Log.Restore()
			util.Log.SetLevel(slog.LevelDebug)
			util.Log.SetOutput(w)

			// set serve func and runWorker func
			v.conf.serve = mockServe
			srv := newMainSrv(&v.conf, runWorker)

			/// set commandPath and commandArgv based on environment
			v.conf.commandPath = os.Getenv("SHELL")
			v.conf.commandArgv = []string{getShellNameFrom(v.conf.commandPath)}

			// send kill signal after some time (finish ms)
			timer1 := time.NewTimer(time.Duration(v.finish) * time.Millisecond)
			go func() {
				<-timer1.C
				srv.downChan <- true
			}()

			srv.start(&v.conf)

			// mock client operation
			resp := mockClient(v.conf.desiredPort, v.pause, frontend.AprilshMsgOpen)
			if !strings.HasPrefix(resp, v.resp) {
				t.Errorf("#test run expect %q got %q\n", v.resp, resp)
			}

			srv.wait()

			// restore stdout
			w.Close()
			io.ReadAll(r)
			os.Stdout = saveStdout
			r.Close()
		})
	}
}

func TestRunWorkerFail(t *testing.T) {
	tc := []struct {
		label string
		conf  Config
	}{
		{
			"openPTS fail", Config{
				version: false, server: true, verbose: _VERBOSE_OPEN_PTS_FAIL, desiredIP: "", desiredPort: "7100",
				locales: localeFlag{"LC_ALL": "en_US.UTF-8", "LANG": "en_US.UTF-8"}, term: "kitty",
				commandPath: "/bin/xxxsh", commandArgv: []string{"-sh"}, withMotd: false,
			},
		},
		{
			"startShell fail", Config{
				version: false, server: true, verbose: _VERBOSE_SKIP_START_SHELL, desiredIP: "", desiredPort: "7200",
				locales: localeFlag{"LC_ALL": "en_US.UTF-8", "LANG": "en_US.UTF-8"}, term: "kitty",
				commandPath: "/bin/xxxsh", commandArgv: []string{"-sh"}, withMotd: false,
			},
		},
		{
			"shell.Wait fail", Config{
				version: false, server: true, verbose: _VERBOSE_SKIP_READ_PIPE, desiredIP: "", desiredPort: "7300",
				locales: localeFlag{"LC_ALL": "en_US.UTF-8", "LANG": "en_US.UTF-8"}, term: "kitty",
				commandPath: "echo", commandArgv: []string{"2"}, withMotd: false,
			},
		},
	}

	exChan := make(chan string, 1)
	whChan := make(chan *workhorse, 1)

	for _, v := range tc {
		t.Run(v.label, func(t *testing.T) {

			// intercept log output
			var b strings.Builder
			defer util.Log.Restore()
			util.Log.SetOutput(&b)
			util.Log.SetLevel(slog.LevelDebug)

			var wg sync.WaitGroup
			var hasWorkhorse bool
			v.conf.serve = mockServe
			if strings.HasPrefix(v.label, "shell.Wait") {
				v.conf.commandPath, _ = exec.LookPath(v.conf.commandPath)
				hasWorkhorse = true // last one has effective work horse.
			}

			wg.Add(1)
			go func() {
				defer wg.Done()
				<-exChan       // get the key
				wh := <-whChan // get the workhorse
				if hasWorkhorse {
					if wh.ptmx == nil || wh.shell == nil {
						t.Errorf("#test runWorker fail should return empty workhorse\n")
					}

				} else {
					if wh.ptmx != nil || wh.shell != nil {
						t.Errorf("#test runWorker fail should return empty workhorse\n")
					}
					msg := <-exChan // get the done message
					if msg != v.conf.desiredPort {
						t.Errorf("#test runWorker fail should return %s, got %s\n", v.conf.desiredPort, msg)
					}
				}
			}()

			if hasWorkhorse {
				if err := runWorker(&v.conf, exChan, whChan); err != nil {
					t.Errorf("#test runWorker should not report error.\n")
				}
			} else {
				if err := runWorker(&v.conf, exChan, whChan); err == nil {
					t.Errorf("#test runWorker should report error.\n")
				}
			}

			wg.Wait()
		})
	}
}

func TestRunCloseFail(t *testing.T) {
	tc := []struct {
		label  string
		pause  int      // pause between client send and read
		resp1  string   // response of start action
		resp2  string   // response of stop action
		exp    []string // ex parameter
		finish int      // pause before shutdown message
		conf   Config
	}{
		{
			"runWorker stopped by " + frontend.AprishMsgClose, 20, frontend.AprilshMsgOpen + "7111,", frontend.AprishMsgClose + "done",
			[]string{},
			50,
			Config{
				version: false, server: true, verbose: _VERBOSE_SKIP_READ_PIPE, desiredIP: "", desiredPort: "7110",
				locales:     localeFlag{"LC_ALL": "en_US.UTF-8", "LANG": "en_US.UTF-8"},
				commandPath: "/bin/sh", commandArgv: []string{"-sh"}, withMotd: true,
			},
		},
		{
			"runWorker stop port not exist", 5, frontend.AprilshMsgOpen + "7121,", frontend.AprishMsgClose + "port does not exist",
			[]string{"7100"},
			50,
			Config{
				version: false, server: true, verbose: _VERBOSE_SKIP_READ_PIPE, desiredIP: "", desiredPort: "7120",
				locales:     localeFlag{"LC_ALL": "en_US.UTF-8", "LANG": "en_US.UTF-8"},
				commandPath: "/bin/sh", commandArgv: []string{"-sh"}, withMotd: true,
			},
		},
		{
			"runWorker stop wrong port number", 5, frontend.AprilshMsgOpen + "7131,", frontend.AprishMsgClose + "wrong port number",
			[]string{"7121x"},
			50,
			Config{
				version: false, server: true, verbose: _VERBOSE_SKIP_READ_PIPE, desiredIP: "", desiredPort: "7130",
				locales:     localeFlag{"LC_ALL": "en_US.UTF-8", "LANG": "en_US.UTF-8"},
				commandPath: "/bin/sh", commandArgv: []string{"-sh"}, withMotd: true,
			},
		},
		{
			"runWorker stop unknow request", 5, frontend.AprilshMsgOpen + "7141,", frontend.AprishMsgClose + "unknow request",
			[]string{"two", "params"},
			50,
			Config{
				version: false, server: true, verbose: _VERBOSE_SKIP_READ_PIPE, desiredIP: "", desiredPort: "7140",
				locales:     localeFlag{"LC_ALL": "en_US.UTF-8", "LANG": "en_US.UTF-8"},
				commandPath: "/bin/sh", commandArgv: []string{"-sh"}, withMotd: true,
			},
		},
	}

	for _, v := range tc {
		t.Run(v.label, func(t *testing.T) {
			// defer util.Log.Restore()
			// util.Log.SetLevel(slog.LevelDebug)
			// util.Log.SetOutput(os.Stdout)

			// intercept stdout
			saveStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w
			defer util.Log.Restore()
			util.Log.SetOutput(w)
			util.Log.SetLevel(slog.LevelDebug)

			// set serve func and runWorker func
			v.conf.serve = mockServe
			srv := newMainSrv(&v.conf, runWorker)

			/// set commandPath and commandArgv based on environment
			v.conf.commandPath = os.Getenv("SHELL")
			v.conf.commandArgv = []string{getShellNameFrom(v.conf.commandPath)}

			// send shutdown message after some time (finish ms)
			timer1 := time.NewTimer(time.Duration(v.finish) * time.Millisecond)
			go func() {
				<-timer1.C
				srv.downChan <- true
			}()

			srv.start(&v.conf)

			// start a new connection
			resp1 := mockClient(v.conf.desiredPort, v.pause, frontend.AprilshMsgOpen)
			if !strings.HasPrefix(resp1, v.resp1) {
				t.Errorf("#test run expect %q got %q\n", v.resp1, resp1)
			}
			// fmt.Printf("#test got start response %s\n", resp1)

			time.Sleep(30 * time.Millisecond)

			// stop the new connection
			resp2 := mockClient(v.conf.desiredPort, v.pause, frontend.AprishMsgClose, v.exp...)
			if !strings.HasPrefix(resp2, v.resp2) {
				t.Errorf("#test run expect %q got %q\n", v.resp1, resp2)
			}

			// fmt.Printf("#test got start response %s\n", resp2)
			// stop the connection
			if len(v.exp) > 0 {
				expect := frontend.AprishMsgClose + "done"
				resp2 := mockClient(v.conf.desiredPort, v.pause, frontend.AprishMsgClose)
				if !strings.HasPrefix(resp2, expect) {
					t.Errorf("#test run stop the connection expect %q got %q\n", v.resp1, resp2)
				}
			}

			// fmt.Printf("#test got stop response %s\n", resp2)
			srv.wait()

			// restore stdout
			w.Close()
			io.ReadAll(r)
			os.Stdout = saveStdout
			r.Close()
		})
	}
}

func TestRunWith2Clients(t *testing.T) {
	tc := []struct {
		label  string
		pause  int      // pause between client send and read
		resp1  string   // response of start action
		resp2  string   // response of stop action
		resp3  string   // response of additinoal open request
		exp    []string // ex parameter
		finish int      // pause before shutdown message
		conf   Config
	}{
		{
			"open aprilsh with duplicate request", 20, frontend.AprilshMsgOpen + "7101,", frontend.AprishMsgClose + "done",
			frontend.AprilshMsgOpen + "7102", []string{}, 50,
			Config{
				version: false, server: true, verbose: _VERBOSE_SKIP_READ_PIPE, desiredIP: "", desiredPort: "7100",
				locales:     localeFlag{"LC_ALL": "en_US.UTF-8", "LANG": "en_US.UTF-8"},
				commandPath: "/bin/sh", commandArgv: []string{"-sh"}, withMotd: true,
			},
		},
	}

	for _, v := range tc {
		t.Run(v.label, func(t *testing.T) {
			defer util.Log.Restore()
			util.Log.SetLevel(slog.LevelDebug)
			util.Log.SetOutput(os.Stdout)

			// intercept stdout
			defer util.Log.Restore()
			util.Log.SetLevel(slog.LevelDebug)
			util.Log.SetOutput(io.Discard)

			// set serve func and runWorker func
			v.conf.serve = mockServe
			srv := newMainSrv(&v.conf, runWorker)

			/// set commandPath and commandArgv based on environment
			v.conf.commandPath = os.Getenv("SHELL")
			v.conf.commandArgv = []string{getShellNameFrom(v.conf.commandPath)}

			srv.start(&v.conf)

			// start a new connection
			resp1 := mockClient(v.conf.desiredPort, v.pause, frontend.AprilshMsgOpen)
			if !strings.HasPrefix(resp1, v.resp1) {
				t.Errorf("#test first client start expect %q got %q\n", v.resp1, resp1)
			}
			// fmt.Printf("#test got 1 response %q\n", resp1)

			// start a new connection
			resp3 := mockClient(v.conf.desiredPort, v.pause, frontend.AprilshMsgOpen)
			if !strings.HasPrefix(resp3, v.resp3) {
				t.Errorf("#test second client start expect %q got %q\n", v.resp3, resp3)
			}
			// fmt.Printf("#test got 3 response %q\n", resp3)

			// stop the new connection
			resp2 := mockClient(v.conf.desiredPort, v.pause, frontend.AprishMsgClose, v.exp...)
			if !strings.HasPrefix(resp2, v.resp2) {
				t.Errorf("#test firt client stop expect %q got %q\n", v.resp1, resp2)
			}
			// fmt.Printf("#test got 2 response %q\n", resp2)

			// send shutdown message after some time (finish ms)
			timer1 := time.NewTimer(time.Duration(v.finish) * time.Millisecond)
			go func() {
				<-timer1.C
				srv.downChan <- true
			}()

			srv.wait()
		})
	}
}

func TestStartShellFail(t *testing.T) {
	tc := []struct {
		label    string
		errStr   string
		pts      *os.File
		pr       *io.PipeReader
		utmpHost string
		conf     Config
	}{
		{"first error return", "fail to start shell", nil, nil, "",
			Config{verbose: _VERBOSE_SKIP_START_SHELL},
		},
		{"IUTF8 error return", strENOTTY, os.Stdin, nil, "",
			Config{verbose: 0},
		}, // os.Stdin doesn't support IUTF8 flag, startShell should failed
		{"start shell error return", "/bin/xxxsh", os.Stdin, nil, "aprilsh [123]",
			Config{version: false, server: true, verbose: 1, desiredIP: "", desiredPort: "7100",
				locales: localeFlag{"LC_ALL": "en_US.UTF-8", "LANG": "en_US.UTF-8"}, term: "kitty",
				commandPath: "/bin/xxxsh", commandArgv: []string{"-sh"}, withMotd: false},
		}, // commandPath is wrong, startShell should failed.
		{"start shell error with term", "/bin/xxxsh", os.Stdin, nil, "aprilsh [123]",
			Config{version: false, server: true, verbose: 1, desiredIP: "", desiredPort: "7100",
				locales: localeFlag{"LC_ALL": "en_US.UTF-8", "LANG": "en_US.UTF-8"}, term: "",
				commandPath: "/bin/xxxsh", commandArgv: []string{"-sh"}, withMotd: true},
		}, // term is empty, withMotd is true, startShell should failed.
	}

	defer util.Log.Restore()
	util.Log.SetOutput(io.Discard)
	// util.Log.SetOutput(os.Stdout)
	// util.Log.SetLevel(slog.LevelDebug)

	for i, v := range tc {
		t.Run(v.label, func(t *testing.T) {
			if i >= 2 {
				// open pty master and slave
				ptmx, pts, _ := pty.Open()
				defer func() {
					ptmx.Close()
					pts.Close()
				}()
				v.pts = pts

				// open pipe for parameter
				pr, pw := io.Pipe()
				v.pr = pr

				go func() {
					time.Sleep(2 * time.Millisecond)
					pw.Close()
					util.Log.With("action", "send").Debug("start shell message")
				}()
			}

			_, err := startShell(v.pts, v.pr, v.utmpHost, &v.conf)

			// validate error
			if !strings.Contains(err.Error(), v.errStr) {
				t.Errorf("%q should report %q, got %q\n", v.label, v.errStr, err)
				fmt.Printf("%#v\n", err)
			}
		})
	}
}

func TestOpenPTS(t *testing.T) {

	tc := []struct {
		label  string
		ws     unix.Winsize
		errStr string
	}{
		{"invalid parameter error", unix.Winsize{}, "invalid parameter"},
		{"invalid parameter error", unix.Winsize{Row: 4, Col: 4}, ""},
	}

	for i, v := range tc {
		t.Run(v.label, func(t *testing.T) {
			var ptmx, pts *os.File
			var err error
			if i == 0 {
				ptmx, pts, err = openPTS(nil)
			} else {
				ptmx, pts, err = openPTS(&v.ws)
			}
			defer ptmx.Close()
			defer pts.Close()
			if i == 0 {
				if !strings.Contains(err.Error(), v.errStr) {
					t.Errorf("%q should report %q, got %q\n", v.label, v.errStr, err)
					fmt.Printf("%#v\n", err)
				}
			} else {
				if err != nil {
					t.Errorf("%q expect no error, got %s\n", v.label, err)
				}
			}
		})
	}
}

func TestGetCurrentUser(t *testing.T) {
	// normal invocation
	userCurrentTest = false
	uid := fmt.Sprintf("%d", os.Getuid())
	expect, _ := user.LookupId(uid)

	got := getCurrentUser()
	if len(got) == 0 || expect.Username != got {
		t.Errorf("#test getCurrentUser expect %s, got %s\n", expect.Username, got)
	}

	// getCurrentUser fail
	old := userCurrentTest
	defer func() {
		userCurrentTest = old
	}()

	// intercept log output
	var b strings.Builder
	// logW.SetOutput(&b)
	defer util.Log.Restore()
	util.Log.SetOutput(&b)
	util.Log.SetLevel(slog.LevelDebug)

	userCurrentTest = true
	got = getCurrentUser()
	if got != "" {
		t.Errorf("#test getCurrentUser expect empty string, got %s\n", got)
	}
	// restore logW
	// logW = log.New(os.Stdout, "WARN: ", log.Ldate|log.Ltime|log.Lshortfile)
}

func TestGetAvailablePort(t *testing.T) {
	tc := []struct {
		label      string
		max        int // pre-condition before getAvailabePort
		expectPort int
		expectMax  int
		workers    map[int]*workhorse
	}{
		{
			"empty worker list", 6001, 6001, 6002,
			map[int]*workhorse{},
		},
		{
			"lart gap empty worker", 6008, 6001, 6002,
			map[int]*workhorse{},
		},
		{
			"add one port", 6002, 6002, 6003,
			map[int]*workhorse{6001: nil},
		},
		{
			"shrink max", 6013, 6002, 6003,
			map[int]*workhorse{6001: nil},
		},
		{
			"right most", 6004, 6004, 6005,
			map[int]*workhorse{6001: nil, 6002: nil, 6003: nil},
		},
		{
			"left most", 6006, 6001, 6006,
			map[int]*workhorse{6003: nil, 6004: nil, 6005: nil},
		},
		{
			"middle hole", 6009, 6004, 6009,
			map[int]*workhorse{6001: nil, 6002: nil, 6003: nil, 6008: nil},
		},
		{
			"border shape hole", 6019, 6002, 6019,
			map[int]*workhorse{6001: nil, 6018: nil},
		},
	}

	conf := &Config{desiredPort: "6000"}

	for _, v := range tc {
		t.Run(v.label, func(t *testing.T) {
			// intercept log output
			defer util.Log.Restore()
			util.Log.SetOutput(io.Discard)
			// util.Log.SetOutput(os.Stdout)

			srv := newMainSrv(conf, mockRunWorker)
			srv.workers = v.workers
			srv.maxPort = v.max

			got := srv.getAvailabePort()

			if got != v.expectPort {
				t.Errorf("%q expect port=%d, got %d\n", v.label, v.expectPort, got)
			}

			if srv.maxPort != v.expectMax {
				t.Errorf("%q expect maxPort=%d, got %d\n", v.label, v.expectMax, srv.maxPort)
			}
		})
	}
}

// func TestIsPortExist(t *testing.T) {
// 	tc := []struct {
// 		label string
// 		port  int
// 		ret   bool
// 	}{
// 		{"port exist", 101, true},
// 		{"port does not exist", 10, false},
// 	}
//
// 	// prepare workers data
// 	conf := &Config{desiredPort: "6000"}
//
// 	srv := newMainSrv(conf, mockRunWorker)
// 	srv.workers[100] = &workhorse{nil, os.Stderr}
// 	srv.workers[101] = &workhorse{nil, os.Stdout}
// 	srv.workers[111] = &workhorse{nil, os.Stdin}
//
// 	for _, v := range tc {
// 		t.Run(v.label, func(t *testing.T) {
// 			got := srv.isPortExist(v.port)
// 			if got != v.ret {
// 				t.Errorf("%q port %d: expect %t, got %t\n", v.label, v.port, v.ret, got)
// 			}
//
// 		})
// 	}
// }

func BenchmarkGetAvailablePort(b *testing.B) {

	conf := &Config{desiredPort: "100"}
	srv := newMainSrv(conf, mockRunWorker)
	srv.workers[100] = &workhorse{nil, os.Stderr}
	srv.workers[101] = &workhorse{nil, os.Stdout}
	srv.workers[102] = &workhorse{nil, os.Stdin}

	srv.maxPort = 102

	for i := 0; i < b.N; i++ {
		srv.getAvailabePort()
		srv.maxPort-- // hedge maxPort++ in getAvailabePort
	}
}
