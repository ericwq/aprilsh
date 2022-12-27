/*

MIT License

Copyright (c) 2022~2023 wangqi

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.

*/

package main

import (
	"bytes"
	"errors"
	"flag"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"testing"
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

func TestParseFlagsCorrect(t *testing.T) {
	tc := []struct {
		args []string
		conf Config
	}{
		{
			[]string{"-locale", "ALL=en_US.UTF-8", "-l", "LANG=UTF-8"},
			Config{
				version: false, server: false, verbose: false, validate: false, desiredIP: "", desiredPort: "",
				locales: localeFlag{"ALL": "en_US.UTF-8", "LANG": "UTF-8"}, color: 0,
				commandPath: "", commandArgv: []string{}, withMotd: false, args: []string{},
			},
		},
		{
			[]string{"--", "/bin/sh"},
			Config{
				version: false, server: false, verbose: false, validate: false, desiredIP: "", desiredPort: "",
				locales: localeFlag{}, color: 0,
				commandPath: "", commandArgv: []string{"/bin/sh"}, withMotd: false, args: []string{"/bin/sh"},
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
				version: false, server: false, verbose: false, validate: false, desiredIP: "", desiredPort: "",
				locales: localeFlag{"LC_ALL": "en_US.UTF-8", "LANG": "en_US.UTF-8"}, color: 0,
				commandPath: "", commandArgv: []string{"/bin/sh"}, withMotd: false, args: []string{},
			},
			Config{
				version: false, server: false, verbose: false, validate: false, desiredIP: "", desiredPort: "",
				locales: localeFlag{"LC_ALL": "en_US.UTF-8", "LANG": "en_US.UTF-8"}, color: 0,
				commandPath: "/bin/sh", commandArgv: []string{"/bin/sh"}, withMotd: false, args: []string{},
			},
			nil,
		},
		{
			"non UTF-8 locale",
			Config{
				version: false, server: false, verbose: false, validate: false, desiredIP: "", desiredPort: "",
				locales: localeFlag{"LC_ALL": "zh_CN.GB2312", "LANG": "zh_CN.GB2312"}, color: 0,
				commandPath: "", commandArgv: []string{"/bin/sh"}, withMotd: false, args: []string{},
			},
			Config{
				version: false, server: false, verbose: false, validate: false, desiredIP: "", desiredPort: "",
				locales: localeFlag{}, color: 0,
				commandPath: "/bin/sh", commandArgv: []string{"/bin/sh"}, withMotd: false, args: []string{},
			},
			errors.New("UTF-8 locale fail."),
		},
		{
			"empty commandArgv",
			Config{
				version: false, server: false, verbose: false, validate: false, desiredIP: "", desiredPort: "",
				locales: localeFlag{"LC_ALL": "en_US.UTF-8"}, color: 0,
				commandPath: "", commandArgv: []string{}, withMotd: false, args: []string{},
			},
			Config{
				version: false, server: false, verbose: false, validate: false, desiredIP: "", desiredPort: "",
				locales: localeFlag{"LC_ALL": "en_US.UTF-8"}, color: 0,
				commandPath: "/bin/zsh", commandArgv: []string{"-zsh"}, withMotd: true, args: []string{},
			},
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

	// save the stdout and create replaced pipe
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

	// read and restore the stdout
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
