// Copyright 2022~2024 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package frontend

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"log/slog"
	"maps"
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

func TestTerminalCaps(t *testing.T) {
	tc := []struct {
		expect map[int]string
		label  string
	}{
		{map[int]string{1: "first", 2: "second"}, "normal"},
		{map[int]string{}, "empty map"},
		{nil, "nil map"},
	}

	for _, v := range tc {
		t.Run(v.label, func(t *testing.T) {
			middle := EncodeTerminalCaps(v.expect)
			got, _ := DecodeTerminalCaps(middle)
			if !maps.Equal(got, v.expect) {
				t.Errorf("%s expect map %v, got %v\n", v.label, v.expect, got)
			}
		})
	}

	_, err := DecodeTerminalCaps([]byte("bad base64"))
	if err == nil {
		t.Errorf("expect error, got nil\n")
	}
	// fmt.Printf("%s err=%s\n", "report", err)

	jsonData, _ := json.Marshal("some string")
	dst := make([]byte, base64.StdEncoding.EncodedLen(len(jsonData)))
	base64.StdEncoding.Encode(dst, []byte(jsonData))

	_, err = DecodeTerminalCaps(dst)
	if err == nil {
		t.Errorf("expect error, got nil\n")
	}
}
