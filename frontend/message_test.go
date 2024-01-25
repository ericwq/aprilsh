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
	defer util.Log.Restore()
	util.Log.SetOutput(w)
	util.Log.SetLevel(slog.LevelDebug)

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
