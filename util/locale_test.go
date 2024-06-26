// Copyright 2022 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package util

import (
	"os"
	"testing"
)

func TestSetlocale(t *testing.T) {
	// https://www.nixcraft.com/t/how-to-change-date-command-output-language-locales-in-alpine-linux/4434?u=nixcraft
	tc := []struct {
		label  string
		locale string
		ret    string
		real   string
	}{
		// {"the locale is malformed", "un_KN.ow", "", "UTF-8"},
		{"the locale is supported by OS", "en_US.UTF-8", "en_US.UTF-8", "UTF-8"},
		// {"chinese locale", "zh_CN.GB18030", "zh_CN.GB18030", "GB18030"},
		{"alpine doesn't support this locale", "en_GB.UTF-8", "en_GB.UTF-8", "UTF-8"},
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
		got = LocaleCharset()
		if got != v.real {
			t.Errorf("#test %q localeCharset() expect %q got %q\n", v.label, v.real, got)
		}
	}
}

func TestLocaleSetNativeLocale(t *testing.T) {
	// validate the utf-8 result
	utf8Locale := "en_US.UTF-8"
	os.Setenv("LC_ALL", utf8Locale)
	SetNativeLocale()
	if !IsUtf8Locale() {
		t.Errorf("#test expect UTF-8 locale, got %s\n", LocaleCharset())
	}
}

func testLocale_nl_langinfo2(t *testing.T) {
	_, err := nl_langinfo2("locale", []string{"-error -args"})
	if err == nil {
		t.Errorf("#test expect error from nlLangInfo(), got nil\n")
	}
}

func testLocale_nl_langinfo(t *testing.T) {
	os.Setenv("LC_ALL", "en_US.UTF-8")
	SetNativeLocale()
	ret1, err := nl_langinfo2("locale", []string{"charmap"})
	ret0 := nl_langinfo(CODESET)
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
		lv := GetCtype()
		if v.expect != lv.String() {
			t.Errorf("%q expect %q, got %q\n", v.label, v.expect, lv.String())
		}

		ClearLocaleVariables()
	}
}
