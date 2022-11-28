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
	"os"
	"os/exec"
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

	// output.Reset()
	//
	// creat a .hide file and write random data into it
	// fName := ".hide"
	// hide, _ := os.Create(fName)
	// data := encrypt.PrngFill(4)
	// hide.Write(data)
	// hide.Close()
	//
	// if printMotd(&output, fName) {
	// 	t.Errorf("#test printMotd should return false, instead it return true. %q", output.String())
	// }
	//
	// os.Remove(fName)
}

func TestPrintVersion(t *testing.T) {
	var b strings.Builder
	expect := []string{"aprish-server", "build", "wangqi ericwq057[AT]qq[dot]com"}

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
	usage := []string{
		"Usage:", "aprilsh-server",
		"[-s] [-v] [-i LOCALADDR] [-p PORT[:PORT2]] [-c COLORS] [-l NAME=VALUE]", "[-- COMMAND...]",
	}

	printUsage(&b, usage[1])

	// validate the result
	result := b.String()
	found := 0
	for i := range usage {
		if strings.Contains(result, usage[i]) {
			found++
		}
	}
	if found != len(usage) {
		t.Errorf("#test printUsage expect %q, got %q\n", usage, result)
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
