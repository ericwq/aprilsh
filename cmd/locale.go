// Copyright 2022 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package cmd

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

func GetCtype() localeVar {
	if all := os.Getenv("LC_ALL"); all != "" {
		return localeVar{"LC_ALL", all}
	} else if ctype := os.Getenv("LC_CTYPE"); ctype != "" {
		return localeVar{"LC_CTYPE", ctype}
	} else if lang := os.Getenv("LANG"); lang != "" {
		return localeVar{"LANG", lang}
	}

	return localeVar{"", ""}
}

func LocaleCharset() (ret string) {
	// ret, err := nl_langinfo2("locale", []string{"charmap"})
	// if err != nil {
	// 	ret = ""
	// }
	// return

	return nl_langinfo(CODESET)
}

// return true if current locale charset is utf-8, otherwise false.
func IsUtf8Locale() bool {
	cs := LocaleCharset()
	// fmt.Printf("#isUtf8Locale cs=%s\n", cs)

	if strings.Compare(strings.ToLower(cs), "utf-8") == 0 {
		return true
	}
	return false
}

func SetNativeLocale() (ret string) {
	ret = setlocale(LC_ALL, "")
	// fmt.Printf("#setNativeLocale setlocale return %q\n", ret)
	if ret == "" { // cognizant of the locale environment variable
		ctype := GetCtype()
		fmt.Printf("The locale requested by %s isn't available here.\n", ctype)
		if ctype.name != "" {
			fmt.Printf("Running 'locale-gen %s' may be necessary.\n", ctype.value)
		}
		// } else {
		// 	fmt.Fprintf(os.Stderr, "#setNativeLocale setlocale return %q\n", setlocale(LC_ALL, ""))
	}
	return
}

func ClearLocaleVariables() {
	list := []string{
		"LANG", "LANGUAGE", "LC_CTYPE", "LC_NUMERIC", "LC_TIME", "LC_COLLATE",
		"LC_MONETARY", "LC_MESSAGES", "LC_PAPER", "LC_NAME", "LC_ADDRESS",
		"LC_TELEPHONE", "LC_MEASUREMENT", "LC_IDENTIFICATION", "LC_ALL",
	}
	for _, v := range list {
		os.Unsetenv(v)
	}
}
