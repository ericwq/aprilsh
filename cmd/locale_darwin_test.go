// Copyright 2022 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

//go:build darwin

package cmd

import (
	"log"
	"os"
	"strings"
	"testing"
)

func TestSetlocaleDarwin(t *testing.T) {
	// https://www.nixcraft.com/t/how-to-change-date-command-output-language-locales-in-alpine-linux/4434?u=nixcraft
	tc := []struct {
		label  string
		locale string
		ret    string
		real   string
	}{
		{"the locale is malformed", "un_KN.ow", "", "UTF-8"},
		{"chinese locale", "zh_CN.GB18030", "zh_CN.GB18030", "GB18030"},
	}

	// initialize locale
	setlocale(LC_ALL, "en_US.UTF-8")

	for _, v := range tc {
		// change the locale
		got := setlocale(LC_ALL, v.locale)
		if got != v.ret {
			t.Errorf("#test %q setlocale() expect %q got %q\n", v.label, v.ret, got)
		}

		// check the real locale
		got = localeCharset()
		if got != v.real {
			t.Errorf("#test %q localeCharset() expect %q got %q\n", v.label, v.real, got)
		}
	}
}

func TestSetNativeLocaleDarwin(t *testing.T) {
	// validate the non utf-8 result
	zhLocale := "zh_CN.GB2312"
	os.Setenv("LC_ALL", zhLocale)
	setNativeLocale()
	if isUtf8Locale() {
		t.Errorf("#test expect non-UTF-8 locale, got %s\n", localeCharset())
	}

	// intercept log output
	var b strings.Builder
	logW.SetOutput(&b)

	badLocale := "un_KN.ow"
	os.Setenv("LC_ALL", badLocale)
	ret := setNativeLocale()

	// restore logW
	logW = log.New(os.Stdout, "WARN: ", log.Ldate|log.Ltime|log.Lshortfile)
	// validate the error handling
	if ret != "" {
		t.Errorf("#test malformed locale expect %q got %q\n", badLocale, ret)
	}
	if isUtf8Locale() {
		t.Errorf("#test expect UTF-8 locale, got %s\n", localeCharset())
	}
}
