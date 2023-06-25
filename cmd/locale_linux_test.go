// Copyright 2022 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

//go:build linux

package cmd

import (
	"os"
	"testing"
)

func TestSetNativeLocale(t *testing.T) {
	tc := []struct {
		label  string
		locale string
		expect string
		utf8   bool
	}{
		// {"set locale zh_TW.ASCII", "zh_TW.ASCII", "zh_TW.ASCII", false},
		{"set locale POSIX", "POSIX", "C", false},
		// {"set locale unKNow", "unKNow", "", false},
	}

	for _, v := range tc {
		os.Setenv("LC_ALL", v.locale)
		got := SetNativeLocale()
		if got != v.expect {
			t.Errorf("#test SetNativeLocale() %s expect %q, got %q\n", v.label, v.expect, got)
		}

		if IsUtf8Locale() != v.utf8 {
			t.Errorf("#test IsUtf8Locale() %s expect %t, got %t\n", v.label, v.utf8, IsUtf8Locale())
		}
	}
}
