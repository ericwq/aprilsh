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
		{
			"on mac, we have SSH_AUTH_SOCK and .ssh/id_rsa.pub .ssh/id_rsa file, so we have ssh agent and public key auths",
			[]string{frontend.CommandClientName, "-vv", "ide@localhost2"},
			"xterm-256color",
			[]string{"No such host"},
		},
	}

	for _, v := range tc {
		t.Run(v.label, func(t *testing.T) {
			// in case we can't log in with ssh
			if _, err := os.Stat(defaultSSHClientID); err != nil {
				t.Skip("no " + defaultSSHClientID + " skip this")
			}
			// intercept stdout
			saveStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

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
					// fmt.Printf("found %s\n", expect[i])
					found++
				}
			}
			if found != len(v.expect) {
				t.Errorf("#test expect %s, got \n%s\n", v.expect, result)
			}
		})
	}
}
