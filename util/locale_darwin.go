// Copyright 2022~2023 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package util

import (
	"fmt"
)

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

