// Copyright 2022~2023 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package util

import (
	"github.com/creack/pty"
	"golang.org/x/sys/unix"
)

func CheckIUTF8(fd int) (bool, error) {
	termios, err := unix.IoctlGetTermios(fd, GetTermios)
	if err != nil {
		return false, err
	}

	// Input is UTF-8 (since Linux 2.6.4)
	return (termios.Iflag & unix.IUTF8) != 0, nil
}

func SetIUTF8(fd int) error {
	termios, err := unix.IoctlGetTermios(fd, GetTermios)
	if err != nil {
		return err
	}

	// when the bit is set to 1, enable IUTF8
	termios.Iflag |= unix.IUTF8
	unix.IoctlSetTermios(fd, SetTermios, termios)

	return nil
}

func ConvertWinsize(windowSize *unix.Winsize) *pty.Winsize {
	if windowSize == nil {
		return nil
	}
	var sz pty.Winsize
	sz.Cols = windowSize.Col
	sz.Rows = windowSize.Row
	sz.X = windowSize.Xpixel
	sz.Y = windowSize.Ypixel

	return &sz
}
