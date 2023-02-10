// Copyright 2022 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"errors"
	"flag"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"reflect"
	"strings"
	"syscall"
	"testing"

	"github.com/creack/pty"
	"golang.org/x/term"
)

func TestPrintMotd(t *testing.T) {
	// darwin doesn't has the following motd files, so we add /etc/hosts for testing.
	files := []string{"/run/motd.dynamic", "/var/run/motd.dynamic", "/etc/motd", "/etc/hosts"}

	var output bytes.Buffer
	//
	// if !printMotd(&output, files[0]) {
	// 	output.Reset()
	// 	if printMotd(&output, files[1]) {
	// 		fmt.Printf("%s", output.String())
	// 	}
	// }
	//
	// printMotd(files[2])
	//
	// if runtime.GOOS == "darwin" {
	// 	printMotd(files[3])
	// }

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
	var b strings.Builder
	expect := []string{COMMAND_NAME, "build", "wangqi ericwq057[AT]qq[dot]com"}

	printVersion(&b)

	// validate the result
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

func TestPrintUsage(t *testing.T) {
	var b strings.Builder
	expect := []string{
		"Usage:", COMMAND_NAME,
		"[--server] [--verbose] [--ip ADDR] [--port PORT[:PORT2]] [--color COLORS] [--locale NAME=VALUE] [-- command...]",
	}

	printUsage(&b, usage)

	// validate the result
	result := b.String()
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
		os.Args = []string{COMMAND_NAME, "--help"}
		// test help
		main()
	}

	out := captureStdoutRun(testHelpFunc)

	// validate result
	expect := []string{
		"Usage:", COMMAND_NAME,
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
func captureStdoutRun(f func()) []byte {
	// save the stdout and create replaced pipe
	rescueStdout := os.Stdout
	r, w, _ := os.Pipe()
	// replace stdout with pipe writer
	// alll the output to stdout is captured
	os.Stdout = w

	// os.Args is a "global variable", so keep the state from before the test, and restore it after.
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	f()

	// close pipe writer
	w.Close()
	// get the output
	out, _ := ioutil.ReadAll(r)
	os.Stdout = rescueStdout

	return out
}

func TestMainVersion(t *testing.T) {
	testVersionFunc := func() {
		// prepare data
		os.Args = []string{COMMAND_NAME, "--version"}
		// test
		main()
	}

	out := captureStdoutRun(testVersionFunc)

	// validate result
	expect := []string{COMMAND_NAME, "build", "wangqi ericwq057[AT]qq[dot]com"}
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

func TestMainParseError(t *testing.T) {
	testFunc := func() {
		// prepare data
		os.Args = []string{COMMAND_NAME, "--foo"}
		// test
		main()
	}

	out := captureStdoutRun(testFunc)

	// validate result
	expect := []string{"flag provided but not defined", "Usage of aprilsh-server"}
	result := string(out)
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

func TestMainDoConfig(t *testing.T) {
	testFunc := func() {
		// abort doConfig() for test environment
		os.Setenv(DOCONFIG_TEST, "TRUE")
		defer os.Unsetenv(DOCONFIG_TEST)
		// prepare data
		os.Args = []string{COMMAND_NAME, "-locale", "LC_ALL=en_US.UTF-8", "--", "/bin/sh"}
		// test
		main()
	}

	// save the stderr and create replaced pipe
	rescueStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// intercept logW
	var b strings.Builder
	logW.SetOutput(&b)

	testFunc()

	// read and restore the stderr
	w.Close()
	out, _ := ioutil.ReadAll(r)
	os.Stderr = rescueStderr

	// validate result
	expect := []string{COMMAND_NAME, "needs a UTF-8 native locale to run."}
	result := string(out)
	found := 0
	for i := range expect {
		if strings.Contains(result, expect[i]) {
			found++
		}
	}
	if found != 0 {
		t.Errorf("#test doConfig expect %q, got %q\n", expect, result)
	}

	// validate logW
	var expectLog string = DOCONFIG_TEST + " is set."
	got := b.String()
	if !strings.Contains(got, expectLog) {
		t.Errorf("#test doConfig expect %q, got %s\n", expectLog, got)
	}

	// restore logW
	logW = log.New(os.Stderr, "WARN: ", log.Ldate|log.Ltime|log.Lshortfile)
}

func TestParseFlagsCorrect(t *testing.T) {
	tc := []struct {
		args []string
		conf Config
	}{
		{
			[]string{"-locale", "ALL=en_US.UTF-8", "-l", "LANG=UTF-8"},
			Config{
				version: false, server: false, verbose: false, desiredIP: "", desiredPort: "",
				locales: localeFlag{"ALL": "en_US.UTF-8", "LANG": "UTF-8"}, color: 0,
				commandPath: "", commandArgv: []string{}, withMotd: false,
			},
		},
		{
			[]string{"--", "/bin/sh"},
			Config{
				version: false, server: false, verbose: false, desiredIP: "", desiredPort: "",
				locales: localeFlag{}, color: 0,
				commandPath: "", commandArgv: []string{"/bin/sh"}, withMotd: false,
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
				t.Errorf("conf got \n%+v, want \n%+v", *conf, v.conf)
			}
		})
	}
}

func TestDoConfig(t *testing.T) {
	tc := []struct {
		lable string
		conf0 Config
		conf2 Config
		err   error
	}{
		{
			"UTF-8 locale",
			Config{
				version: false, server: false, verbose: false, desiredIP: "", desiredPort: "",
				locales: localeFlag{"LC_ALL": "en_US.UTF-8", "LANG": "en_US.UTF-8"}, color: 0,
				commandPath: "", commandArgv: []string{"/bin/sh"}, withMotd: false,
			},
			Config{
				version: false, server: false, verbose: false, desiredIP: "", desiredPort: "",
				locales: localeFlag{"LC_ALL": "en_US.UTF-8", "LANG": "en_US.UTF-8"}, color: 0,
				commandPath: "/bin/sh", commandArgv: []string{"/bin/sh"}, withMotd: false,
			},
			nil,
		},
		{
			"non UTF-8 locale",
			Config{
				version: false, server: false, verbose: false, desiredIP: "", desiredPort: "",
				locales: localeFlag{"LC_ALL": "zh_CN.GB2312", "LANG": "zh_CN.GB2312"}, color: 0,
				commandPath: "", commandArgv: []string{"/bin/sh"}, withMotd: false,
			}, // TODO GB2312 is not available in apline linux
			Config{
				version: false, server: false, verbose: false, desiredIP: "", desiredPort: "",
				locales: localeFlag{}, color: 0,
				commandPath: "/bin/sh", commandArgv: []string{"/bin/sh"}, withMotd: false,
			},
			errors.New("UTF-8 locale fail."),
		},
		{
			"empty commandArgv",
			Config{
				version: false, server: false, verbose: false, desiredIP: "", desiredPort: "",
				locales: localeFlag{"LC_ALL": "en_US.UTF-8"}, color: 0,
				commandPath: "", commandArgv: []string{}, withMotd: false,
			},
			Config{
				version: false, server: false, verbose: false, desiredIP: "", desiredPort: "",
				locales: localeFlag{"LC_ALL": "en_US.UTF-8"}, color: 0,
				commandPath: "/bin/zsh", commandArgv: []string{"-zsh"}, withMotd: true,
			}, // TODO /bin/zsh is macOS only
			nil,
		},
	}

	for _, v := range tc {
		t.Run(v.lable, func(t *testing.T) {
			// set SHELL for empty commandArgv
			if len(v.conf0.commandArgv) == 0 {
				shell := os.Getenv("SHELL")
				defer os.Setenv("SHELL", shell)
				os.Unsetenv("SHELL")
			}

			// save the stderr and create replaced pipe
			rescueStderr := os.Stderr
			r, w, _ := os.Pipe()
			os.Stderr = w
			oldArgs := os.Args
			defer func() { os.Args = oldArgs }()

			// validate doConfig
			err := doConfig(&v.conf0)
			if err != nil {
				if err.Error() != v.err.Error() {
					t.Errorf("#test doConfig expect %q, got %q\n", v.err, err)
				}
			} else if !reflect.DeepEqual(v.conf0, v.conf2) {
				t.Errorf("#test doConfig got \n%+v, expect \n%+v\n", v.conf0, v.conf2)
			}
			// reset the environment
			clearLocaleVariables()

			// read and restore the stderr
			w.Close()
			ioutil.ReadAll(r)
			os.Stderr = rescueStderr
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

func testMainParameters(t *testing.T) {
	// flag is a global variable, reset it before test
	flag.CommandLine = flag.NewFlagSet("TestMainParameters", flag.ExitOnError)
	testParaFunc := func() {
		// prepare data
		os.Args = []string{COMMAND_NAME, "-validate", "--"} //"-l LC_ALL=en_US.UTF-8", "--"}
		// test
		main()
	}

	out := captureStdoutRun(testParaFunc)

	// validate result
	expect := []string{"main", "commandPath=", "commandArgv=", "withMotd=", "locales=", "color="}
	result := string(out)
	found := 0
	for i := range expect {
		if strings.Contains(result, expect[i]) {
			found++
		}
	}
	if found != len(expect) {
		t.Errorf("#test main() expect %q, got %q\n", expect, result)
	}
}

func TestMainServerPortrangeError(t *testing.T) {
	var b strings.Builder
	logW.SetOutput(&b)

	os.Args = []string{COMMAND_NAME, "-s", "-p=3a"}
	os.Setenv("SSH_CONNECTION", "172.17.0.1 58774 172.17.0.2 22")
	main()

	// validate port range check
	expect := "Bad UDP port range"
	got := b.String()
	if !strings.Contains(got, expect) {
		t.Errorf("#test --port should contains %q, got %s\n", expect, got)
	}

	// restore logW
	logW = log.New(os.Stderr, "WARN: ", log.Ldate|log.Ltime|log.Lshortfile)
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

	// save the stderr and create replaced pipe
	rescueStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	for _, v := range tc {
		os.Setenv("SSH_CONNECTION", v.env)
		got := getSSHip()
		if got != v.expect {
			t.Errorf("%q expect %q, got %q\n", v.label, v.expect, got)
		}
	}

	// read and restore the stderr
	w.Close()
	ioutil.ReadAll(r)
	os.Stderr = rescueStderr
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

	// save the stderr and create replaced pipe
	rescueStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	for _, v := range tc {
		os.Setenv(v.key, v.value)

		got := getTimeFrom(v.key, 0)
		if got != v.expect {
			t.Errorf("%s expct %d, got %d\n", v.lable, v.expect, got)
		}
	}

	// read and restore the stderr
	w.Close()
	ioutil.ReadAll(r)
	os.Stderr = rescueStderr
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
