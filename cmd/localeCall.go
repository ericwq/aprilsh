// Copyright 2022 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package main

// #include <locale.h>
// #include <langinfo.h>
// #include <stdlib.h>
import "C"

import (
	"os/exec"
	"strings"
	"unsafe"
)

func setlocale(lc C.int, locale string) string {
	param := C.CString(locale)
	defer C.free(unsafe.Pointer(param))

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
func nl_langinfo2(cmd string, args []string) (string, error) {
	out, err := exec.Command(cmd, args...).Output()
	if err != nil {
		return "", err
	}

	// fmt.Printf("#nl_langinfo2 got %q\n", out)
	charmap := strings.TrimSuffix(string(out), "\n")
	return charmap, nil
}

func nl_langinfo(item C.int) string {
	ret := C.nl_langinfo(item)
	return C.GoString(ret)
}
