// Copyright 2022~2024 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package main

// https://stackoverflow.com/questions/14926020/setting-process-name-as-seen-by-ps-in-go
// https://tycho.pizza/blog/2015/02/setproctitle.html
// func SetProcessName_Linux(name string) error {
// 	bytes := append([]byte(name), 0)
// 	ptr := unsafe.Pointer(&bytes[0])
// 	if _, _, errno := syscall.RawSyscall6(syscall.SYS_PRCTL, syscall.PR_SET_NAME, uintptr(ptr), 0, 0, 0, 0); errno != 0 {
// 		return syscall.Errno(errno)
// 	}
// 	return nil
// }
//
// func SetProcessName(name string) error {
// 	argv0str := (*reflect.StringHeader)(unsafe.Pointer(&os.Args[0]))
// 	argv0 := (*[1 << 30]byte)(unsafe.Pointer(argv0str.Data))[:argv0str.Len]
//
// 	n := copy(argv0, name)
// 	if n < len(argv0) {
// 		argv0[n] = 0
// 	}
//
// 	return nil
// }
