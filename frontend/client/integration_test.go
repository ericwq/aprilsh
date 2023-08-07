//go:build integration

package main

// https://go.dev/blog/integration-test-coverage
func TestFetchKey(t *testing.T) {
	tc := []struct {
		label  string
		conf   *Config
		pwd    string
		expect string
	}{
		{"normal response", &Config{user: "ide", host: "localhost", port: 60000}, "password", ""},
	}
	for _, v := range tc {
		t.Run(v.label, func(t *testing.T) {
			got := v.conf.fetchKey(v.pwd)
			if got != v.expect {
				t.Errorf("#test %q expect %q, got %q\n", v.label, v.expect, got)
			}
		})
	}
}
