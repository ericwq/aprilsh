// Copyright 2022 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

//go:build darwin || freebsd || openbsd || netbsd
// +build darwin freebsd openbsd netbsd

package main

import (
	"golang.org/x/sys/unix"
)

const (
	getTermios = unix.TIOCGETA
	setTermios = unix.TIOCSETA
)
