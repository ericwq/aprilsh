// Copyright 2022~2023 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

//go:build darwin

package main

type utmpEntry struct{}

func addUtmpEntry(ptmxName string) *utmpEntry {
	logW.Printf("unimplement %s\n", "addUtmpEntry()")
	return nil
}

func updateLasLog(ptmxName string) {
	logW.Printf("unimplement %s\n", "updateLasLog()")
}

func clearUtmpEntry(entry *utmpEntry) {
	logW.Printf("unimplement %s\n", "clearUtmpEntry()")
}
