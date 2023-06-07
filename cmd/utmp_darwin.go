// Copyright 2022~2023 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

//go:build darwin

package main

type utmpEntry struct{}

func addUtmpEntry(ptmxName string, host string) *utmpEntry {
	logW.Printf("unimplement %s\n", "addUtmpEntry()")
	return nil
}

func updateLastLog(ptmxName string) {
	logW.Printf("unimplement %s\n", "updateLastLog()")
}

func clearUtmpEntry(entry *utmpEntry) {
	logW.Printf("unimplement %s\n", "clearUtmpEntry()")
}

func checkUnattachedRecord(userName string, ignore string) []string {
	logW.Printf("unimplement %s\n", "checkUnattachedRecord()")
	return nil
}
