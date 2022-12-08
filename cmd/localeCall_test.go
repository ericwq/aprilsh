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
	setlocale(LC_ALL, zhLocale)

	// prepare differen locale environment variable
	os.Setenv("LC_ALL", zhLocale)

	setNativeLocale()
	if isUtf8Locale() {
		t.Errorf("#test expect UTF-8 locale, got %s\n", localeCharset())
	}

	utf8Locale := "en_US.UTF-8"
	os.Setenv("LC_ALL", utf8Locale)

	setNativeLocale()
	if !isUtf8Locale() {
		t.Errorf("#test expect UTF-8 locale, got %s\n", localeCharset())
	}
}

func TestNlLangInfo(t *testing.T) {
	_, err := nlLangInfo("locale", []string{"-error -args"})
	if err == nil {
		t.Errorf("#test expect error from nlLangInfo(), got nil\n")
	}
}
