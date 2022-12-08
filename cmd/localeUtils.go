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

func (lv *localeVar) str() string {
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
	ASCII_name := "US-ASCII"
	ret, err := nlLangInfo()
	if err != nil {
		ret = ""
	}
	if ret == "ANSI_X3.4-1968" {
		ret = ASCII_name
	}

	return
}

func isUtf8Locale() bool {
	cs := localeCharset()
	if strings.Compare(strings.ToLower(cs), "utf8") != 0 {
		return false
	}
	return true
}

func setNativeLocale() {
	if setlocale(LC_ALL, "") == "" {
		ctype := getCtype()
		fmt.Fprintf(os.Stderr, "The locale requested by %s isn't available here.\n", ctype.str())
		if ctype.name != "" {
			fmt.Fprintf(os.Stderr, "Running `locale-gen %s' may be necessary.\n\n", ctype.value)
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
