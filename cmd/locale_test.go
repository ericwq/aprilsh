// Copyright 2022 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package main

import (
	"os"
	"testing"
)

func TestLocaleSetNativeLocale(t *testing.T) {
	// prepare differ locale
	zhLocale := "zh_CN.GB2312"
	os.Setenv("LC_ALL", zhLocale)

	// validate the non utf-8 result
	setNativeLocale()
	if isUtf8Locale() {
		t.Errorf("#test expect non-UTF-8 locale, got %s\n", localeCharset())
	}

	utf8Locale := "en_US.UTF-8"
	os.Setenv("LC_ALL", utf8Locale)

	// validate the utf-8 result
	setNativeLocale()
	if !isUtf8Locale() {
		t.Errorf("#test expect UTF-8 locale, got %s\n", localeCharset())
	}
}

func TestLocaleNlLangInfo2(t *testing.T) {
	_, err := nl_langinfo2("locale", []string{"-error -args"})
	if err == nil {
		t.Errorf("#test expect error from nlLangInfo(), got nil\n")
	}
}

func TestLocalseNl_langinfo(t *testing.T) {
	ret0 := nl_langinfo(CODESET)
	ret1, err := nl_langinfo2("locale", []string{"charmap"})
	if err != nil {
		t.Errorf("#test should return nil error, got %s\n", err)
	}

	if ret0 != ret1 {
		t.Errorf("#test nl_langinfo return %s, nl_langinfo2 return %s\n", ret0, ret1)
	}
}

func TestLocaleGetCtype(t *testing.T) {
	tc := []struct {
		label  string
		key    string
		value  string
		expect string
	}{
		{"LC_ALL", "LC_ALL", "zh_CN", "LC_ALL=zh_CN"},
		{"LC_CTYPE", "LC_CTYPE", "en_US.UTF-8", "LC_CTYPE=en_US.UTF-8"},
		{"LANG", "LANG", "it_IT.ISO8859-1", "LANG=it_IT.ISO8859-1"},
		{"empty", "LC_NAME", "ja_JP.eucJP", "[no charset variables]"},
	}

	for _, v := range tc {
		os.Setenv(v.key, v.value)
		lv := getCtype()
		if v.expect != lv.String() {
			t.Errorf("%q expect %q, got %q\n", v.label, v.expect, lv.String())
		}

		clearLocaleVariables()
	}
}
