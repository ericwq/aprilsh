// Copyright 2022 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/exec"
	"os/signal"
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
	"golang.org/x/sys/unix"
	"golang.org/x/term"
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
	initLog()

	expect := []string{_COMMAND_NAME, "build", "wangqi ericwq057[AT]qq[dot]com"}

	printVersion()

	// restore stdout
	w.Close()
	b, _ := ioutil.ReadAll(r)
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

func TestPrintUsage(t *testing.T) {
	// intercept stdout
	saveStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	initLog()

	// var b strings.Builder
	expect := []string{
		"Usage:", _COMMAND_NAME,
		"[--server] [--verbose] [--ip ADDR] [--port PORT[:PORT2]] [--color COLORS] [--locale NAME=VALUE] [-- command...]",
	}

	printUsage(usage)

	// restore stdout
	w.Close()
	b, _ := ioutil.ReadAll(r)
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
		t.Errorf("#test printUsage expect %q, got %q\n", expect, result)
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
		os.Args = []string{_COMMAND_NAME, "--help"}
		// test help
		main()
	}

	out := captureStdoutRun(testHelpFunc)

	// validate result
	expect := []string{
		"Usage:", _COMMAND_NAME,
		"[--server] [--verbose] [--ip ADDR] [--port PORT[:PORT2]] [--color COLORS] [--locale NAME=VALUE] [-- command...]",
	}

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
func captureLogRun(f func()) string {
	// intercept log output
	var b strings.Builder
	logW.SetOutput(&b)
	logI.SetOutput(&b)

	// call the test func
	f()

	// restore logW
	logW = log.New(os.Stdout, "WARN: ", log.Ldate|log.Ltime|log.Lshortfile)
	logI = log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)

	return b.String()
}

// capture the stdout and run the
func captureStdoutRun(f func()) []byte {
	// save the stdout and create replaced pipe
	rescueStdout := os.Stdout
	r, w, _ := os.Pipe()
	// replace stdout with pipe writer
	// alll the output to stdout is captured
	os.Stdout = w
	initLog()

	// os.Args is a "global variable", so keep the state from before the test, and restore it after.
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	f()

	// close pipe writer
	w.Close()
	// get the output
	out, _ := ioutil.ReadAll(r)
	os.Stdout = rescueStdout
	r.Close()

	return out
}

func TestMainVersion(t *testing.T) {
	// intercept stdout
	saveStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	initLog()

	// prepare data
	os.Args = []string{_COMMAND_NAME, "--version"}
	// test
	main()

	// restore stdout
	w.Close()
	out, _ := ioutil.ReadAll(r)
	os.Stdout = saveStdout
	r.Close()

	// validate result
	expect := []string{_COMMAND_NAME, "build", "wangqi ericwq057[AT]qq[dot]com"}
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
		os.Args = []string{_COMMAND_NAME, "--foo"}
		// test
		main()
	}

	result := captureLogRun(testFunc)

	// validate result
	expect := []string{"flag provided but not defined: -foo", "Usage of aprilsh-server"}
	found := 0
	for i := range expect {
		if strings.Contains(result, expect[i]) {
			found++
		}
	}
	if found != len(expect) {
		t.Errorf("#test parserError expect %q, got \n%s\n", expect, result)
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
	// shutdown after 50ms
	time.AfterFunc(100*time.Millisecond, func() {
		// fmt.Printf("#test main buildConfig PID:%d\n", os.Getpid())
		syscall.Kill(os.Getpid(), syscall.SIGHUP)
		syscall.Kill(os.Getpid(), syscall.SIGTERM)
	})

	// intercept stdout
	saveStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	initLog()

	// prepare data
	os.Args = []string{_COMMAND_NAME, "-locale", "LC_ALL=en_US.UTF-8", "-p", "6100", "--", "/bin/sh", "-sh"}
	// test
	main()

	// restore stdout
	w.Close()
	out, _ := ioutil.ReadAll(r)
	os.Stdout = saveStdout
	r.Close()

	// validate the result from printWelcome
	expect := []string{
		"aprilsh-server", "a MIT-stylelicense that can be found in the LICENSE file.",
		"got message SIGHUP", // "got message SIGTERM",
	}
	result := string(out)
	found := 0
	for i := range expect {
		if strings.Contains(result, expect[i]) {
			// fmt.Printf("found %s\n", expect[i])
			found++
		}
	}
	if found != len(expect) {
		t.Errorf("#test expect %q, got %q\n", expect, result)
	}
}

func TestMainBuildConfigFail(t *testing.T) {
	testFunc := func() {
		// prepare parameter
		os.Args = []string{_COMMAND_NAME, "-locale", "LC_ALL=en_US.UTF-8", "-p", "6100", "--", "/bin/sh", "-sh"}
		// test
		main()
	}

	// intercept log output
	var b strings.Builder
	logW.SetOutput(&b)

	// prepare for buildConfig fail
	buildConfigTest = true
	testFunc()

	// restore the condition
	buildConfigTest = false

	// validate the result
	expect := []string{"UTF-8 locale fail."}
	result := b.String()
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

func TestParseFlagsCorrect(t *testing.T) {
	tc := []struct {
		args []string
		conf Config
	}{
		{
			[]string{"-locale", "ALL=en_US.UTF-8", "-l", "LANG=UTF-8"},
			Config{
				version: false, server: false, verbose: 0, desiredIP: "", desiredPort: "6000",
				locales: localeFlag{"ALL": "en_US.UTF-8", "LANG": "UTF-8"}, color: 0,
				commandPath: "", commandArgv: []string{}, withMotd: false,
			},
		},
		{
			[]string{"--", "/bin/sh", "-sh"},
			Config{
				version: false, server: false, verbose: 0, desiredIP: "", desiredPort: "6000",
				locales: localeFlag{}, color: 0,
				commandPath: "", commandArgv: []string{"/bin/sh", "-sh"}, withMotd: false,
			},
		},
		{
			[]string{"--", ""},
			Config{
				version: false, server: false, verbose: 0, desiredIP: "", desiredPort: "6000",
				locales: localeFlag{}, color: 0,
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

func TestBuildConfig(t *testing.T) {
	tc := []struct {
		label string
		conf0 Config
		conf2 Config
		err   error
	}{
		{
			"UTF-8 locale",
			Config{
				version: false, server: false, verbose: 0, desiredIP: "", desiredPort: "",
				locales: localeFlag{"LC_ALL": "en_US.UTF-8", "LANG": "en_US.UTF-8"}, color: 0,
				commandPath: "", commandArgv: []string{"/bin/sh", "-sh"}, withMotd: false,
			},
			Config{
				version: false, server: false, verbose: 0, desiredIP: "", desiredPort: "",
				locales: localeFlag{"LC_ALL": "en_US.UTF-8", "LANG": "en_US.UTF-8"}, color: 0,
				commandPath: "/bin/sh", commandArgv: []string{"-sh"}, withMotd: false,
			},
			nil,
		},
		{
			"empty commandArgv",
			Config{
				version: false, server: false, verbose: 0, desiredIP: "", desiredPort: "",
				locales: localeFlag{"LC_ALL": "en_US.UTF-8"}, color: 0,
				commandPath: "", commandArgv: []string{}, withMotd: false,
			},
			Config{
				version: false, server: false, verbose: 0, desiredIP: "", desiredPort: "",
				locales: localeFlag{"LC_ALL": "en_US.UTF-8"}, color: 0,
				commandPath: "/bin/ash", commandArgv: []string{"-ash"}, withMotd: true,
			},
			// macOS: /bin/zsh
			// alpine: /bin/ash
			nil,
		},
		// {
		// 	"non UTF-8 locale",
		// 	Config{
		// 		version: false, server: false, verbose: 0, desiredIP: "", desiredPort: "",
		// 		locales: localeFlag{"LC_ALL": "zh_CN.GB2312", "LANG": "zh_CN.GB2312"}, color: 0,
		// 		commandPath: "", commandArgv: []string{"/bin/sh", "-sh"}, withMotd: false,
		// 	}, // TODO GB2312 is not available in apline linux
		// 	Config{
		// 		version: false, server: false, verbose: 0, desiredIP: "", desiredPort: "",
		// 		locales: localeFlag{}, color: 0,
		// 		commandPath: "/bin/sh", commandArgv: []string{"*sh"}, withMotd: false,
		// 	},
		// 	errors.New("UTF-8 locale fail."),
		// },
		{
			"commandArgv is one string",
			Config{
				version: false, server: false, verbose: 0, desiredIP: "", desiredPort: "",
				locales: localeFlag{"LC_ALL": "en_US.UTF-8", "LANG": "en_US.UTF-8"}, color: 0,
				commandPath: "", commandArgv: []string{"/bin/sh"}, withMotd: false,
			},
			Config{
				version: false, server: false, verbose: 0, desiredIP: "", desiredPort: "",
				locales: localeFlag{"LC_ALL": "en_US.UTF-8", "LANG": "en_US.UTF-8"}, color: 0,
				commandPath: "/bin/sh", commandArgv: []string{"-sh"}, withMotd: false,
			},
			nil,
		},
	}

	// change the tc[1].conf2 value according to runtime.GOOS
	switch runtime.GOOS {
	case "darwin":
		tc[1].conf2.commandArgv = []string{"-zsh"}
		tc[1].conf2.commandPath = "/bin/zsh"
	case "linux":
		tc[1].conf2.commandArgv = []string{"-ash"}
		tc[1].conf2.commandPath = "/bin/ash"
	}

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

func TestParseFlagsError(t *testing.T) {
	tests := []struct {
		args   []string
		errstr string
	}{
		{[]string{"-foo"}, "flag provided but not defined"},
		{[]string{"-color", "joe"}, "invalid value"},
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
	var b strings.Builder
	logW.SetOutput(&b)

	os.Args = []string{_COMMAND_NAME, "-s", "-p=3a"}
	os.Setenv("SSH_CONNECTION", "172.17.0.1 58774 172.17.0.2 22")
	main()

	// validate port range check
	expect := "Bad UDP port range"
	got := b.String()
	if !strings.Contains(got, expect) {
		t.Errorf("#test --port should contains %q, got %s\n", expect, got)
	}

	// restore logW
	logW = log.New(os.Stdout, "WARN: ", log.Ldate|log.Ltime|log.Lshortfile)
}

func TestGetSSHip(t *testing.T) {
	tc := []struct {
		label  string
		env    string
		expect string
	}{
		{"no env variable", "", ""},
		{"ipv4 address", "172.17.0.1 58774 172.17.0.2 22", "172.17.0.2"},
		{"malform variable", " 1 2 3 4", ""},
		{"ipv6 address", "fe80::14d5:1215:f8c9:11fa%en0 42000 fe80::aede:48ff:fe00:1122%en5 22", "fe80::aede:48ff:fe00:1122%en5"},
		{"ipv4 mapped address", "::FFFF:172.17.0.1 42200 ::FFFF:129.144.52.38 22", "129.144.52.38"},
	}

	// intercept log output
	var b strings.Builder
	logW.SetOutput(&b)

	for _, v := range tc {
		b.Reset()

		os.Setenv("SSH_CONNECTION", v.env)
		got := getSSHip()
		if got != v.expect {
			t.Errorf("%q expect %q, got %q\n", v.label, v.expect, got)
		}
	}

	// restore logW
	logW = log.New(os.Stdout, "WARN: ", log.Ldate|log.Ltime|log.Lshortfile)
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
	initLog()

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
	ioutil.ReadAll(r)
	os.Stdout = rescueStdout
}

func TestCheckIUTF8(t *testing.T) {
	// try pts master and slave first.
	pty, tty, err := pty.Open()
	if err != nil {
		t.Errorf("#checkIUTF8 Open %s\n", err)
	}

	// clean pts fd
	defer func() {
		if err != nil {
			pty.Close()
			tty.Close()
		}
	}()

	flag, err := checkIUTF8(int(pty.Fd()))
	if err != nil {
		t.Errorf("#checkIUTF8 master %s\n", err)
	}
	if flag {
		t.Errorf("#checkIUTF8 master got %t, expect %t\n", flag, false)
	}

	flag, err = checkIUTF8(int(tty.Fd()))
	if err != nil {
		t.Errorf("#checkIUTF8 slave %s\n", err)
	}
	if flag {
		t.Errorf("#checkIUTF8 slave got %t, expect %t\n", flag, false)
	}

	// STDIN fd should return error
	// only works for go test command
	flag, err = checkIUTF8(int(os.Stdin.Fd()))
	if err == nil {
		t.Errorf("#checkIUTF8 stdin should report error, got nil\n")
	}

	nullFD, err := os.OpenFile("/dev/null", os.O_RDWR, 0)
	if err != nil {
		t.Errorf("#checkIUTF8 open %s failed, %s\n", "/dev/null", err)
	}
	defer nullFD.Close()

	// null fd should return error
	flag, err = checkIUTF8(int(nullFD.Fd()))
	if err == nil {
		t.Errorf("#checkIUTF8 null fd should return error, got nil\n")
	}
}

func TestSetIUTF8(t *testing.T) {
	// try pts master and slave first.
	pty, tty, err := pty.Open()
	if err != nil {
		t.Errorf("#setIUTF8 Open %s\n", err)
	}

	// clean pts fd
	defer func() {
		if err != nil {
			pty.Close()
			tty.Close()
		}
	}()

	// pty master doesn't support IUTF8
	flag, err := checkIUTF8(int(pty.Fd()))
	if flag {
		t.Errorf("#checkIUTF8 master got %t, expect %t\n", flag, false)
	}

	// set IUTF8 for master
	err = setIUTF8(int(pty.Fd()))
	if err != nil {
		t.Errorf("#setIUTF8 master got %s, expect nil\n", err)
	}

	// pty master support IUTF8 now
	flag, err = checkIUTF8(int(pty.Fd()))
	if !flag {
		t.Errorf("#checkIUTF8 master got %t, expect %t\n", flag, true)
	}

	// pty slave support IUTF8
	flag, err = checkIUTF8(int(tty.Fd()))
	if !flag {
		t.Errorf("#checkIUTF8 slave got %t, expect %t\n", flag, true)
	}

	// set IUTF8 for slave
	err = setIUTF8(int(tty.Fd()))
	if err != nil {
		t.Errorf("#setIUTF8 slave got %s, expect nil\n", err)
	}

	// STDIN fd doesn't support termios, setIUTF8 return error
	// only works for go test command
	err = setIUTF8(int(os.Stdin.Fd()))
	if err == nil {
		t.Errorf("#setIUTF8 should report error, got nil\n")
	}

	// open /dev/null
	nullFD, err := os.OpenFile("/dev/null", os.O_RDWR, 0)
	if err != nil {
		t.Errorf("#setIUTF8 open %s failed, %s\n", "/dev/null", err)
	}
	defer nullFD.Close()

	// null fd doesn't support termios, checkIUTF8 return error
	flag, err = checkIUTF8(int(nullFD.Fd()))
	if err == nil {
		t.Errorf("#setIUTF8 check %s failed, %s\n", "/dev/null", err)
	}

	// null fd should return error
	err = setIUTF8(int(nullFD.Fd()))
	if err == nil {
		t.Errorf("#setIUTF8 null fd should return nil, error: %s\n", err)
	}
}

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

func TestStart(t *testing.T) {
	tc := []struct {
		label  string
		pause  int    // pause between client send and read
		resp   string // response client read
		finish int    // pause before shutdown message
		conf   Config
	}{
		{
			"start normally", 20, _ASH_OPEN + "7101,This is the mock key\n", 50,
			Config{
				version: false, server: true, verbose: 0, desiredIP: "", desiredPort: "7100",
				locales: localeFlag{"LC_ALL": "en_US.UTF-8", "LANG": "en_US.UTF-8"}, color: 0,
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
			initLog()

			srv := newMainSrv(&v.conf, mockRunWorker)

			// send shutdown message after some time
			timer1 := time.NewTimer(time.Duration(v.finish) * time.Millisecond)
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
			resp := mockClient(v.conf.desiredPort, v.pause, _ASH_OPEN)
			if resp != v.resp {
				t.Errorf("#test run expect %q got %q\n", v.resp, resp)
			}

			srv.wait()

			// restore stdout
			w.Close()
			ioutil.ReadAll(r)
			os.Stdout = saveStdout
			r.Close()
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
				locales: localeFlag{"LC_ALL": "en_US.UTF-8", "LANG": "en_US.UTF-8"}, color: 0,
				commandPath: "/bin/sh", commandArgv: []string{"/bin/sh"}, withMotd: false,
			},
		},
	}

	for _, v := range tc {
		t.Run(v.label, func(t *testing.T) {
			m := newMainSrv(&v.conf, mockRunWorker)

			// intercept logW
			var b strings.Builder
			logW.SetOutput(&b)
			defer func() {
				logW = log.New(os.Stdout, "WARN: ", log.Ldate|log.Ltime|log.Lshortfile)
			}()

			// start mainserver
			m.start(&v.conf)
			// fmt.Println("#test start fail!")

			// validate result: result contains WARN and COMMAND_NAME
			expect := []string{_COMMAND_NAME, "WARN"}
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
func mockRunWorker(conf *Config, exChan chan string, whChan chan *workhorse) error {
	// send the mock key
	// fmt.Println("#mockRunWorker send mock key to run().")
	exChan <- "This is the mock key"

	// pause some time
	time.Sleep(time.Duration(2) * time.Millisecond)

	// notify the server
	// fmt.Println("#mockRunWorker finish run().")
	exChan <- conf.desiredPort

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
	case _ASH_OPEN:
		txbuf = []byte(_ASH_OPEN)
	case _ASH_CLOSE:
		p, _ := strconv.Atoi(port)
		if len(ex) == 0 {
			txbuf = []byte(fmt.Sprintf("%s%d", _ASH_CLOSE, p+1))
		} else if len(ex) == 2 {
			txbuf = []byte(fmt.Sprintf("%s%d", "unknow header:", p+1)) // 2 parameters: unknow header
		} else if len(ex) == 1 {
			p2, err := strconv.Atoi(ex[0])
			if err == nil {
				txbuf = []byte(fmt.Sprintf("%s%d", _ASH_CLOSE, p2)) // 1 digital parameter: wrong port
			} else {
				txbuf = []byte(fmt.Sprintf("%s%s", _ASH_CLOSE, ex[0])) // 1 str parameter: malform port
			}
		}
	}

	_, err := conn.Write(txbuf)
	// fmt.Printf("#mockClient send %q to server: %v from %v\n", txbuf, server_addr, conn.LocalAddr())
	if err != nil {
		fmt.Printf("#mockClient send %s, error %s\n", string(txbuf), err)
	}

	// pause some time
	time.Sleep(time.Millisecond * time.Duration(pause))

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
	flag, err := checkIUTF8(int(pty.Fd()))
	if flag {
		t.Errorf("#test printWelcome master got %t, expect %t\n", flag, false)
	}

	expect := []string{
		_COMMAND_NAME, "build", "Use of this source code is governed by a MIT-style",
		"Warning: termios IUTF8 flag not defined.",
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
		initLog()

		printWelcome(os.Getpid(), 6000, v.tty)

		// restore stdout
		w.Close()
		b, _ := ioutil.ReadAll(r)
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
			t.Errorf("#test printWelcome expect %q, got %q\n", expect, result)
		}
	}
}

func TestConvertWinsize(t *testing.T) {
	tc := []struct {
		label  string
		win    *unix.Winsize
		expect *pty.Winsize
	}{
		{
			"normal case",
			&unix.Winsize{Col: 80, Row: 40, Xpixel: 0, Ypixel: 0},
			&pty.Winsize{Cols: 80, Rows: 40, X: 0, Y: 0},
		},
		{"nil case", nil, nil},
	}

	for _, v := range tc {
		got := convertWinsize(v.win)

		if (v.expect != nil) && (*got != *v.expect) {
			t.Errorf("#test %q expect %v, got %v\n", v.label, v.expect, got)
		}

		if v.expect == nil && got != nil {
			t.Errorf("#test %q expect %v, got %v\n", v.label, v.expect, got)
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
			"worker failed with wrong port number", 20, _ASH_OPEN + "7101,mock key from mockRunWorker2\n", 30,
			Config{
				version: false, server: true, verbose: 0, desiredIP: "", desiredPort: "7100",
				locales: localeFlag{"LC_ALL": "en_US.UTF-8", "LANG": "en_US.UTF-8"}, color: 0,
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
			initLog()

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
			resp := mockClient(v.conf.desiredPort, v.pause, _ASH_OPEN)

			// validate the result.
			if resp != v.resp {
				t.Errorf("#test run expect %q got %q\n", v.resp, resp)
			}

			srv.wait()

			// restore stdout
			w.Close()
			ioutil.ReadAll(r)
			os.Stdout = saveStdout
			r.Close()
		})
	}

	// test case for run() without connection

	srv2 := &mainSrv{}
	srv2.run(&Config{})
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
				locales: localeFlag{"LC_ALL": "en_US.UTF-8", "LANG": "en_US.UTF-8"}, color: 0,
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
			initLog()

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
			ioutil.ReadAll(r)
			os.Stdout = saveStdout
			r.Close()
		})
	}
}

func TestWaitError(t *testing.T) {
	tc := []struct {
		label  string
		pause  int    // pause between client send and read
		resp   string // response client read
		finish int    // pause before shutdown message
		conf   Config
	}{
		{
			"wait error", 20, "", 50,
			Config{
				version: false, server: true, verbose: 0, desiredIP: "", desiredPort: "7000",
				locales: localeFlag{"LC_ALL": "en_US.UTF-8", "LANG": "en_US.UTF-8"}, color: 0,
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
			initLog()

			m := newMainSrv(&v.conf, failRunWorker)

			// send shutdown message after some time
			timer1 := time.NewTimer(time.Duration(v.finish) * time.Millisecond)
			go func() {
				<-timer1.C
				m.downChan <- true
			}()

			// start mainserver
			m.start(&v.conf)

			// mock client operation
			mockClient(v.conf.desiredPort, v.pause, _ASH_OPEN)

			m.wait()

			// restore stdout
			w.Close()
			out, _ := ioutil.ReadAll(r)
			os.Stdout = saveStdout
			r.Close()

			// validate result
			expect := []string{"#mainSrv wait() reports"}
			result := string(out)
			found := 0
			for i := range expect {
				if strings.Contains(result, expect[i]) {
					found++
				}
			}
			if found != 1 {
				t.Errorf("#test start() expect %q, got %q\n", expect, result)
			}
		})
	}
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

func TestRunWorkerKill(t *testing.T) {
	tc := []struct {
		label  string
		pause  int    // pause between client send and read
		resp   string // response client read
		finish int    // pause before shutdown message
		conf   Config
	}{
		{
			"runWorker stopped by signal kill", 20, _ASH_OPEN + "7101,", 50,
			Config{
				version: false, server: true, verbose: 1, desiredIP: "", desiredPort: "7100",
				locales: localeFlag{"LC_ALL": "en_US.UTF-8", "LANG": "en_US.UTF-8"}, color: 0,
				commandPath: "/bin/sh", commandArgv: []string{"-sh"}, withMotd: true,
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

			// set serve func and runWorker func
			v.conf.serve = serve
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

			// mock client operation
			resp := mockClient(v.conf.desiredPort, v.pause, _ASH_OPEN)
			if !strings.HasPrefix(resp, v.resp) {
				t.Errorf("#test run expect %q got %q\n", v.resp, resp)
			}

			// stop the comd process
			wh := srv.workers[7101]
			time.AfterFunc(time.Duration(100)*time.Millisecond, func() {
				wh.shell.Kill()
				// fmt.Printf("-- #test stop workhorse reports error=%v\n", e)
			})

			srv.wait()

			// restore stdout
			w.Close()
			ioutil.ReadAll(r)
			os.Stdout = saveStdout
			r.Close()
		})
	}
}

func TestRunWorkerStop(t *testing.T) {
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
			"runWorker stopped by " + _ASH_CLOSE, 20, _ASH_OPEN + "7101,", _ASH_CLOSE + "done",
			[]string{},
			50,
			Config{
				version: false, server: true, verbose: 1, desiredIP: "", desiredPort: "7100",
				locales: localeFlag{"LC_ALL": "en_US.UTF-8", "LANG": "en_US.UTF-8"}, color: 0,
				commandPath: "/bin/sh", commandArgv: []string{"-sh"}, withMotd: true,
			},
		},
		{
			"runWorker stop port not exist", 20, _ASH_OPEN + "7121,", _ASH_CLOSE + "port does not exist",
			[]string{"7100"},
			50,
			Config{
				version: false, server: true, verbose: 1, desiredIP: "", desiredPort: "7120",
				locales: localeFlag{"LC_ALL": "en_US.UTF-8", "LANG": "en_US.UTF-8"}, color: 0,
				commandPath: "/bin/sh", commandArgv: []string{"-sh"}, withMotd: true,
			},
		},
		{
			"runWorker stop wrong port number", 20, _ASH_OPEN + "7131,", _ASH_CLOSE + "wrong port number",
			[]string{"7121x"},
			50,
			Config{
				version: false, server: true, verbose: 1, desiredIP: "", desiredPort: "7130",
				locales: localeFlag{"LC_ALL": "en_US.UTF-8", "LANG": "en_US.UTF-8"}, color: 0,
				commandPath: "/bin/sh", commandArgv: []string{"-sh"}, withMotd: true,
			},
		},
		{
			"runWorker stop unknow request", 20, _ASH_OPEN + "7141,", _ASH_CLOSE + "unknow request",
			[]string{"two", "params"},
			50,
			Config{
				version: false, server: true, verbose: 1, desiredIP: "", desiredPort: "7140",
				locales: localeFlag{"LC_ALL": "en_US.UTF-8", "LANG": "en_US.UTF-8"}, color: 0,
				commandPath: "/bin/sh", commandArgv: []string{"-sh"}, withMotd: true,
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

			// set serve func and runWorker func
			v.conf.serve = serve
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
			resp1 := mockClient(v.conf.desiredPort, v.pause, _ASH_OPEN)
			if !strings.HasPrefix(resp1, v.resp1) {
				t.Errorf("#test run expect %q got %q\n", v.resp1, resp1)
			}
			// fmt.Printf("#test got start response %s\n", resp1)

			time.Sleep(30 * time.Millisecond)

			// stop the new connection
			resp2 := mockClient(v.conf.desiredPort, v.pause, _ASH_CLOSE, v.exp...)
			if !strings.HasPrefix(resp2, v.resp2) {
				t.Errorf("#test run expect %q got %q\n", v.resp1, resp2)
			}

			// stop the connection
			if len(v.exp) > 0 {
				expect := _ASH_CLOSE + "done"
				resp2 := mockClient(v.conf.desiredPort, v.pause, _ASH_CLOSE)
				if !strings.HasPrefix(resp2, expect) {
					t.Errorf("#test run stop the connection expect %q got %q\n", v.resp1, resp2)
				}
			}

			// fmt.Printf("#test got stop response %s\n", resp2)
			srv.wait()

			// restore stdout
			w.Close()
			ioutil.ReadAll(r)
			os.Stdout = saveStdout
			r.Close()
		})
	}
}

func TestStartShellFail(t *testing.T) {
	conf := &Config{
		version: false, server: true, verbose: 1, desiredIP: "", desiredPort: "7100",
		locales: localeFlag{"LC_ALL": "en_US.UTF-8", "LANG": "en_US.UTF-8"}, color: 0, term: "kitty",
		commandPath: "/bin/xxxsh", commandArgv: []string{"-sh"}, withMotd: false,
	}

	// os.Stdin doesn't support IUTF8 flag, startShell should failed
	if _, err := startShell(os.Stdin, "aprilsh [123]", conf); err == nil {
		t.Errorf("#test startShell should report error.\n")
		// t.Error(err)
	}

	ptmx, pts, _ := pty.Open() // open pty master and slave
	defer func() {
		ptmx.Close()
		pts.Close()
	}()

	// commandPath is wrong, startShell should failed.
	if _, err := startShell(pts, "aprilsh [123]", conf); err == nil {
		t.Errorf("#test startShell should report error.\n")
		// t.Error(err)
	}
}

func TestOpenPTSFail(t *testing.T) {
	var ws *unix.Winsize

	// nil ws is the test condition
	ptmx, pts, err := openPTS(ws)
	defer func() {
		if e1 := ptmx.Close(); e1 != nil {
			t.Errorf("#test close pty master failed: %s\n", e1)
		}
		if e2 := pts.Close(); e2 != nil {
			t.Errorf("#test close pty slave failed,: %s\n", e2)
		}
	}()

	if err == nil {
		t.Errorf("#test openPTS should report error.\n")
	}
}

func TestRunWorkerFail(t *testing.T) {
	tc := []struct {
		label string
		conf  Config
	}{
		{
			"openPTS fail", Config{
				version: false, server: true, verbose: _VERBOSE_OPEN_PTS, desiredIP: "", desiredPort: "7100",
				locales: localeFlag{"LC_ALL": "en_US.UTF-8", "LANG": "en_US.UTF-8"}, color: 0, term: "kitty",
				commandPath: "/bin/xxxsh", commandArgv: []string{"-sh"}, withMotd: false,
			},
		},
		{
			"startShell fail", Config{
				version: false, server: true, verbose: _VERBOSE_START_SHELL, desiredIP: "", desiredPort: "7200",
				locales: localeFlag{"LC_ALL": "en_US.UTF-8", "LANG": "en_US.UTF-8"}, color: 0, term: "kitty",
				commandPath: "/bin/xxxsh", commandArgv: []string{"-sh"}, withMotd: false,
			},
		},
	}

	exChan := make(chan string, 1)
	whChan := make(chan *workhorse, 1)

	for _, v := range tc {
		t.Run(v.label, func(t *testing.T) {
			var wg sync.WaitGroup

			// intercept log output
			var b strings.Builder
			logW.SetOutput(&b)

			wg.Add(1)
			go func() {
				defer wg.Done()
				<-exChan       // get the key
				wh := <-whChan // get the workhorse
				if wh.ptmx != nil || wh.shell != nil {
					t.Errorf("#test runWorker fail should return empty workhorse\n")
				}
				msg := <-exChan // get the done message
				if msg != v.conf.desiredPort {
					t.Errorf("#test runWorker fail should return %s, got %s\n", v.conf.desiredPort, msg)
				}
			}()

			if err := runWorker(&v.conf, exChan, whChan); err == nil {
				t.Errorf("#test runWorker should report error.\n")
				// t.Error(err)
			}

			wg.Wait()

			// restore logW
			logW = log.New(os.Stdout, "WARN: ", log.Ldate|log.Ltime|log.Lshortfile)
		})
	}
}

func TestShellWaitFail(t *testing.T) {
	tc := []struct {
		label string
		conf  Config
	}{
		{
			"shell.Wait fail", Config{
				version: false, server: true, verbose: 0, desiredIP: "", desiredPort: "7300",
				locales: localeFlag{"LC_ALL": "en_US.UTF-8", "LANG": "en_US.UTF-8"}, color: 0, term: "kitty",
				commandPath: "echo", commandArgv: []string{"2"}, withMotd: false,
			},
		},
	}

	exChan := make(chan string, 1)
	whChan := make(chan *workhorse, 1)

	for _, v := range tc {
		t.Run(v.label, func(t *testing.T) {
			if strings.HasPrefix(v.label, "shell.Wait") {
				// find executable path
				v.conf.commandPath, _ = exec.LookPath(v.conf.commandPath)
				// set serve func
				v.conf.serve = serve
			}

			var wg sync.WaitGroup

			// intercept log output
			var b strings.Builder
			logW.SetOutput(&b)

			wg.Add(1)
			go func() {
				defer wg.Done()
				<-exChan       // get the key
				wh := <-whChan // get the workhorse
				if wh.ptmx == nil || wh.shell == nil {
					t.Errorf("#test runWorker fail should return a workhorse.\n")
				}
				msg := <-exChan // get the done message
				if msg != v.conf.desiredPort {
					t.Errorf("#test runWorker fail should return %s, got %s\n", v.conf.desiredPort, msg)
				}
			}()

			if err := runWorker(&v.conf, exChan, whChan); err != nil {
				t.Errorf("#test runWorker should report error.\n")
				// t.Error(err)
			}

			wg.Wait()

			// restore logW
			logW = log.New(os.Stdout, "WARN: ", log.Ldate|log.Ltime|log.Lshortfile)
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
	logW.SetOutput(&b)

	userCurrentTest = true
	got = getCurrentUser()
	if got != "" {
		t.Errorf("#test getCurrentUser expect empty string, got %s\n", got)
	}
	// restore logW
	logW = log.New(os.Stdout, "WARN: ", log.Ldate|log.Ltime|log.Lshortfile)
}

func TestGetAvailablePort(t *testing.T) {
	tc := []struct {
		label   string
		expect  int
		max     int
		workers map[int]*workhorse
	}{
		{
			"empty worker list", 6001, 6001,
			map[int]*workhorse{},
		},
		{
			"right most", 6004, 6004,
			map[int]*workhorse{6001: nil, 6002: nil, 6003: nil},
		},
		{
			"left most", 6001, 6006,
			map[int]*workhorse{6003: nil, 6004: nil, 6005: nil},
		},
		{
			"middle hole", 6004, 6009,
			map[int]*workhorse{6001: nil, 6002: nil, 6003: nil, 6008: nil},
		},
	}

	conf := &Config{desiredPort: "6000"}

	for _, v := range tc {
		t.Run(v.label, func(t *testing.T) {
			srv := newMainSrv(conf, mockRunWorker)
			srv.workers = v.workers
			srv.maxPort = v.max

			got := srv.getAvailabePort()

			if got != v.expect {
				t.Errorf("#test %s expect %d, got %d\n", v.label, v.expect, got)
			}
		})
	}
}

func TestDeviceExists(t *testing.T) {
	tc := []struct {
		label  string
		line   string
		expect bool
	}{
		{"pts/1", "pts/1", true},
		{"tty", "tty", true},
		// {"tty0", "tty0", true}, // this one doesn't work for some linux container,
		{"tty/1", "tty/1", false},
	}

	for _, v := range tc {
		t.Run(v.label, func(t *testing.T) {
			got := deviceExists(v.line)
			if got != v.expect {
				t.Errorf("#test %s expect %t, got %t\n", v.label, v.expect, got)
			}
		})
	}
}
