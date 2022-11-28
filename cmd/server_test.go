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

	if !found {
		t.Errorf("#test expect found %s, found nothing\n", files)
	}
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
