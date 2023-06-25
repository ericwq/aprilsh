// Copyright 2022~2023 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package cmd

/*
for alpine: musl setlocale doesn't return NULL

If its value is not a valid locale specification, the locale is unchanged,
and setlocale() returns NULL.

If locale is an empty string, "", each part of the locale that should
be modified is set according to the environment variables.
*/
func SetNativeLocale() (ret string) {
	ret = setlocale(LC_ALL, "")
	// fmt.Printf("#setNativeLocale setlocale return %q\n", ret)
	return
}
