// Copyright 2022~2024 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package frontend

import (
	"io"
	"log/slog"
	"os"
	"strings"
	"testing"

	"github.com/ericwq/aprilsh/util"
)

func TestPrintVersion(t *testing.T) {
	stderr := os.Stderr
	stdout := os.Stdout
	r, w, _ := os.Pipe()
	// replace stdout,stderr with pipe writer
	// all the output to stdout,stderr is captured
	os.Stderr = w
	os.Stdout = w

	util.Logger.CreateLogger(w, true, slog.LevelDebug)

	PrintVersion()

	// close pipe writer
	w.Close()
	// get the output
	out, _ := io.ReadAll(r)
	os.Stderr = stderr
	os.Stdout = stdout
	r.Close()

	output := string(out)
	expect := []string{"version\t", "go version", "git commit", "git branch"}
	found := 0
	for _, v := range expect {
		if strings.Contains(output, v) {
			found++
		}
	}

	if found != len(expect) {
		t.Errorf("%s expect \n%s, got \n%s\n", "PrintVersion", expect, output)
	}
}

func TestPrintUsage(t *testing.T) {
	tc := []struct {
		label  string
		hints  string
		usage  []string
		expect []string
	}{
		{"has hint, has usage", "hint", []string{"usage"}, []string{"Hints: hint", "usage"}},
		{"no hint, has usage", "", []string{"usage"}, []string{"usage"}},
		{"has hint, no usage", "hint", []string{}, []string{"hint"}},
	}

	for _, v := range tc {
		stderr := os.Stderr
		stdout := os.Stdout
		r, w, _ := os.Pipe()
		// replace stdout,stderr with pipe writer
		// all the output to stdout,stderr is captured
		os.Stderr = w
		os.Stdout = w

		util.Logger.CreateLogger(w, true, slog.LevelDebug)

		PrintUsage(v.hints, v.usage...)

		// close pipe writer
		w.Close()
		// get the output
		out, _ := io.ReadAll(r)
		os.Stderr = stderr
		os.Stdout = stdout
		r.Close()

		output := string(out)
		found := 0
		for _, v := range v.expect {
			if strings.Contains(output, v) {
				found++
			}
		}

		if found != len(v.expect) {
			t.Errorf("%s expect \n%s, got \n%s\n", v.label, v.expect, output)
		}
	}
}
