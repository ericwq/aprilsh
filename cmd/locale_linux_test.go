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
	// validate the non utf-8 result
	zhLocale := "zh_TW.ASCII"
	os.Setenv("LC_ALL", zhLocale)

	ret := SetNativeLocale()
	// setlocale(LC_CTYPE, ".ASCII")
	if zhLocale != ret { // the return value should be "zh_TW.ASCII"
		t.Errorf("#test expect %q, got %q\n", zhLocale, ret)
	}
	if IsUtf8Locale() { // the return value should be false
		t.Errorf("#test expect non-UTF-8 locale, got %s\n", LocaleCharset())
	}

	badLocale := "un_KN.ow"
	os.Setenv("LC_ALL", badLocale)
	ret = SetNativeLocale()

	// validate the error handling
	if ret != "" {
		t.Errorf("#test malformed locale expect %q got %q\n", badLocale, ret)
	}
	if IsUtf8Locale() {
		t.Errorf("#test expect UTF-8 locale, got %s\n", LocaleCharset())
	}
}
