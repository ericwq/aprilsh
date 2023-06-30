// Copyright 2022 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

//go:build !darwin && !freebsd && !netbsd && !openbsd && !windows

package util

import (
	"golang.org/x/sys/unix"
)

const (
	GetTermios = unix.TCGETS
	SetTermios = unix.TCSETS
)
