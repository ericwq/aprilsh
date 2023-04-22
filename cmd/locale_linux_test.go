// Copyright 2022 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

//go:build linux

package main

import (
	"os"
	"testing"
)

func TestSetNativeLocale(t *testing.T) {
	// validate the non utf-8 result
	zhLocale := "zh_TW.ASCII"
	os.Setenv("LC_ALL", zhLocale)

	ret := setNativeLocale()
	if zhLocale != ret {
		t.Errorf("#test expect %q, got %q\n", zhLocale, ret)
	}
	if !isUtf8Locale() {
		t.Errorf("#test expect non-UTF-8 locale, got %s\n", localeCharset())
	}
}
