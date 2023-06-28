package main

import (
	"io/ioutil"
	"os"
	"strings"
	"testing"
)

func TestPrintColors(t *testing.T) {
	tc := []struct {
		label  string
		term   string
		expect []string
	}{
		{"lookup terminfo failed", "NotExist", []string{"Dynamic load terminfo failed."}},
		{"TERM is empty", "", []string{"The TERM is empty string."}},
		{"TERM doesn't exit", "-remove", []string{"The TERM doesn't exist."}},
		{"normal found", "xterm-256color", []string{"xterm-256color","256"}},
		{"dynamic found", "xfce", []string{"xfce 8 (dynamic)"}},
		{"dynamic not found", "xxx", []string{"Dynamic load terminfo failed."}},
	}

	for _, v := range tc {
		// intercept stdout
		saveStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w
		// save original TERM
		term := os.Getenv("TERM")

		// set TERM according to test case
		if v.term == "-remove" {
			os.Unsetenv("TERM")
		} else {
			os.Setenv("TERM", v.term)
		}

		printColors()

		// restore stdout
		w.Close()
		b, _ := ioutil.ReadAll(r)
		os.Stdout = saveStdout
		r.Close()

		// validate the result
		result := string(b)
		found := 0
		for i := range v.expect {
			if strings.Contains(result, v.expect[i]) {
				found++
			}
		}
		if found != len(v.expect) {
			t.Errorf("#test %s expect %q, got %q\n", v.label, v.expect, result)
		}

		// restore original TERM
		os.Setenv("TERM", term)
	}
}
