package main

import (
	"io"
	"os"
	"strings"
	"testing"

	"github.com/ericwq/aprilsh/frontend"
)

func TestMainRun_Parameters2(t *testing.T) {
	tc := []struct {
		label  string
		args   []string
		term   string
		expect []string
	}{
		{ // by default, we can't login with ssh
			"only password auth, no ssh agent, no public key file",
			[]string{frontend.CommandClientName, "-vv", "ide@localhost"},
			"xterm-256color",
			[]string{"prepareAuthMethod ssh auth password", "password:", "inappropriate ioctl for device"},
		},
	}

	for _, v := range tc {
		t.Run(v.label, func(t *testing.T) {
			// intercept stdout
			saveStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w
			os.Stderr = w

			// prepare data
			os.Args = v.args
			os.Setenv("TERM", v.term)
			// test main
			main()

			// restore stdout
			w.Close()
			out, _ := io.ReadAll(r)
			os.Stdout = saveStdout
			r.Close()

			// validate the result
			result := string(out)
			found := 0
			for i := range v.expect {
				if strings.Contains(result, v.expect[i]) {
					// fmt.Printf("found %s\n", v.expect[i])
					found++
				}
			}
			if found != len(v.expect) {
				t.Errorf("#test expect %s, got \n%s\n", v.expect, result)
			}
		})
	}
}
