// Copyright 2022~2023 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

//go:build linux || freebsd

package util

// https://blog.csdn.net/sg_knight/article/details/134373559
// func AddUtmpx(pts *os.File, host string) bool {
// 	return utmp.UtmpxAddRecord(pts, host)
// }
//
// func ClearUtmpx(pts *os.File) bool {
// 	return utmp.UtmpxRemoveRecord(pts)
// }
//
// func UpdateLastLog(line, userName, host string) bool {
// 	return utmp.PutLastlogEntry(line, userName, host)
// }
