// Copyright 2022 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

//go:build darwin || freebsd || openbsd || netbsd

package cmd

import (
	"golang.org/x/sys/unix"
)

const (
	GetTermios = unix.TIOCGETA
	SetTermios = unix.TIOCSETA
)
