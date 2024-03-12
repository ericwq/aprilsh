// Copyright 2022~2024 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package util

import (
	"io"
	"os"
	"strings"
	"testing"
)

func TestUtmpxAPI(t *testing.T) {
	tc := []struct {
		label  string
		expect []string
	}{
		{"AddUtmpx", []string{"unimplement", "AddUtmpx"}},
		{"ClearUtmpx", []string{"unimplement", "ClearUtmpx"}},
		{"UpdateLastLog", []string{"unimplement", "UpdateLastLog"}},
		{"CheckUnattachedUtmpx", []string{"unimplement", "CheckUnattachedUtmpx"}},
	}

	for _, v := range tc {
		t.Run(v.label, func(t *testing.T) {
			// save the stdout,stderr and create replaced pipe
			stderr := os.Stderr
			stdout := os.Stdout
			r, w, _ := os.Pipe()
			// replace stdout,stderr with pipe writer
			// alll the output to stdout,stderr is captured
			os.Stderr = w
			os.Stdout = w

			switch v.label {
			case "AddUtmpx":
				AddUtmpx(nil, "")
			case "ClearUtmpx":
				ClearUtmpx(nil)
			case "UpdateLastLog":
				UpdateLastLog("", "", "")
			case "CheckUnattachedUtmpx":
				CheckUnattachedUtmpx("", "", "")
			}

			// close pipe writer, get the output
			w.Close()
			out, _ := io.ReadAll(r)
			os.Stderr = stderr
			os.Stdout = stdout
			r.Close()

			// fmt.Println(string(out))
			// validate result
			result := string(out)
			found := 0
			for i := range v.expect {
				if strings.Contains(result, v.expect[i]) {
					found++
				}
			}
			if found != len(v.expect) {
				t.Errorf("#test printVersion expect %q, got %q\n", v.expect, result)
			}
		})
	}
}
