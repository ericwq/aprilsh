// Copyright 2022 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"os"
	"strings"
)

type localeVar struct {
	name  string
	value string
}

func (lv *localeVar) String() string {
	if lv.name == "" {
		return "[no charset variables]"
	}
	return lv.name + "=" + lv.value
}

func getCtype() localeVar {
	if all := os.Getenv("LC_ALL"); all != "" {
		return localeVar{"LC_ALL", all}
	} else if ctype := os.Getenv("LC_CTYPE"); ctype != "" {
		return localeVar{"LC_CTYPE", ctype}
	} else if lang := os.Getenv("LANG"); lang != "" {
		return localeVar{"LANG", lang}
	}

	return localeVar{"", ""}
}

func localeCharset() (ret string) {
	// ret, err := nl_langinfo2("locale", []string{"charmap"})
	// if err != nil {
	// 	ret = ""
	// }
	// return

	return nl_langinfo(CODESET)
}

func isUtf8Locale() bool {
	cs := localeCharset()
	// fmt.Printf("#isUtf8Locale cs=%s\n", cs)

	if strings.Compare(strings.ToLower(cs), "utf-8") == 0 {
		return true
	}
	return false
}

func setNativeLocale() {
	if setlocale(LC_ALL, "") == "" { // cognizant of the locale environment variable
		ctype := getCtype()
		fmt.Fprintf(os.Stderr, "The locale requested by %s isn't available here.\n", ctype)
		if ctype.name != "" {
			fmt.Fprintf(os.Stderr, "Running 'locale-gen %s' may be necessary.\n\n", ctype.value)
		}
	}
}

func clearLocaleVariables() {
	list := []string{
		"LANG", "LANGUAGE", "LC_CTYPE", "LC_NUMERIC", "LC_TIME", "LC_COLLATE",
		"LC_MONETARY", "LC_MESSAGES", "LC_PAPER", "LC_NAME", "LC_ADDRESS",
		"LC_TELEPHONE", "LC_MEASUREMENT", "LC_IDENTIFICATION", "LC_ALL",
	}
	for _, v := range list {
		os.Unsetenv(v)
	}
}
