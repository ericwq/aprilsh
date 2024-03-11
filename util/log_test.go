// Copyright 2022~2024 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package util

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"
	"testing"
)

func TestCreateLogger(t *testing.T) {
	// save the stdout,stderr and create replaced pipe
	stderr := os.Stderr
	stdout := os.Stdout
	r, w, _ := os.Pipe()
	// replace stdout,stderr with pipe writer
	// alll the output to stdout,stderr is captured
	os.Stderr = w
	os.Stdout = w

	Logger.CreateLogger(w, false, LevelTrace)

	// log trace
	msg1 := "trace message"
	Logger.Trace(msg1) // level with name

	// level without name
	LevelDebug_2 := slog.Level(-6)
	msg2 := "no name debug message"
	Logger.Log(context.Background(), LevelDebug_2, msg2)

	// close pipe writer, get the output
	w.Close()
	out, _ := io.ReadAll(r)
	os.Stderr = stderr
	os.Stdout = stdout
	r.Close()

	// fmt.Println(string(out))
	// validate result
	expect := []string{"level=TRACE", "level=DEBUG-2", msg1, msg2}
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
