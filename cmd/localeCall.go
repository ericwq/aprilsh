// Copyright 2022 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package main

// #include <locale.h>
// #include <stdlib.h>
// #include <errno.h>
import "C"

import (
	"os/exec"
	"strings"
	"unsafe"
)

const (
	LC_ALL      = 0
	LC_COLLATE  = 1
	LC_CTYPE    = 2
	LC_MONETARY = 3
	LC_NUMERIC  = 5
	LC_TIME     = 6
)

func setlocale(lc C.int, locale string) string {
	param := C.CString(locale)
	defer C.free(unsafe.Pointer(param))

	// TODO we didn't check the possible errno
	ret := C.setlocale(lc, param)
	return C.GoString(ret)
}

// man nl_langinfo
//
// CODESET (LC_CTYPE)
//
//	Return a string with the name of the character encoding used in
//	the selected locale, such as "UTF-8", "ISO-8859-1", or
//	"ANSI_X3.4-1968" (better known as US-ASCII).  This is the same
//	string that you get with "locale charmap".
func nlLangInfo() (string, error) {
	out, err := exec.Command("locale", "charmap").Output()
	if err != nil {
		return "", err
	}

	charmap := strings.TrimSuffix(string(out), "\n")
	return charmap, nil
}
