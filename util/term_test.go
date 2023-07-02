// Copyright 2022~2023 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package util

import (
	"os"
	"testing"

	"github.com/creack/pty"
	"golang.org/x/sys/unix"
)

func TestCheckIUTF8(t *testing.T) {
	// try pts master and slave first.
	pty, tty, err := pty.Open()
	if err != nil {
		t.Errorf("#checkIUTF8 Open %s\n", err)
	}

	// clean pts fd
	defer func() {
		if err != nil {
			pty.Close()
			tty.Close()
		}
	}()

	flag, err := CheckIUTF8(int(pty.Fd()))
	if err != nil {
		t.Errorf("#checkIUTF8 master %s\n", err)
	}
	if flag {
		t.Errorf("#checkIUTF8 master got %t, expect %t\n", flag, false)
	}

	flag, err = CheckIUTF8(int(tty.Fd()))
	if err != nil {
		t.Errorf("#checkIUTF8 slave %s\n", err)
	}
	if flag {
		t.Errorf("#checkIUTF8 slave got %t, expect %t\n", flag, false)
	}

	// STDIN fd should return error
	// only works for go test command
	flag, err = CheckIUTF8(int(os.Stdin.Fd()))
	if err == nil {
		t.Errorf("#checkIUTF8 stdin should report error, got nil\n")
	}

	nullFD, err := os.OpenFile("/dev/null", os.O_RDWR, 0)
	if err != nil {
		t.Errorf("#checkIUTF8 open %s failed, %s\n", "/dev/null", err)
	}
	defer nullFD.Close()

	// null fd should return error
	flag, err = CheckIUTF8(int(nullFD.Fd()))
	if err == nil {
		t.Errorf("#checkIUTF8 null fd should return error, got nil\n")
	}
}

func TestSetIUTF8(t *testing.T) {
	// try pts master and slave first.
	pty, tty, err := pty.Open()
	if err != nil {
		t.Errorf("#setIUTF8 Open %s\n", err)
	}

	// clean pts fd
	defer func() {
		if err != nil {
			pty.Close()
			tty.Close()
		}
	}()

	// pty master doesn't support IUTF8
	flag, err := CheckIUTF8(int(pty.Fd()))
	if flag {
		t.Errorf("#checkIUTF8 master got %t, expect %t\n", flag, false)
	}

	// set IUTF8 for master
	err = SetIUTF8(int(pty.Fd()))
	if err != nil {
		t.Errorf("#setIUTF8 master got %s, expect nil\n", err)
	}

	// pty master support IUTF8 now
	flag, err = CheckIUTF8(int(pty.Fd()))
	if !flag {
		t.Errorf("#checkIUTF8 master got %t, expect %t\n", flag, true)
	}

	// pty slave support IUTF8
	flag, err = CheckIUTF8(int(tty.Fd()))
	if !flag {
		t.Errorf("#checkIUTF8 slave got %t, expect %t\n", flag, true)
	}

	// set IUTF8 for slave
	err = SetIUTF8(int(tty.Fd()))
	if err != nil {
		t.Errorf("#setIUTF8 slave got %s, expect nil\n", err)
	}

	// STDIN fd doesn't support termios, setIUTF8 return error
	// only works for go test command
	err = SetIUTF8(int(os.Stdin.Fd()))
	if err == nil {
		t.Errorf("#setIUTF8 should report error, got nil\n")
	}

	// open /dev/null
	nullFD, err := os.OpenFile("/dev/null", os.O_RDWR, 0)
	if err != nil {
		t.Errorf("#setIUTF8 open %s failed, %s\n", "/dev/null", err)
	}
	defer nullFD.Close()

	// null fd doesn't support termios, checkIUTF8 return error
	flag, err = CheckIUTF8(int(nullFD.Fd()))
	if err == nil {
		t.Errorf("#setIUTF8 check %s failed, %s\n", "/dev/null", err)
	}

	// null fd should return error
	err = SetIUTF8(int(nullFD.Fd()))
	if err == nil {
		t.Errorf("#setIUTF8 null fd should return nil, error: %s\n", err)
	}
}

func TestConvertWinsize(t *testing.T) {
	tc := []struct {
		label  string
		win    *unix.Winsize
		expect *pty.Winsize
	}{
		{
			"normal case",
			&unix.Winsize{Col: 80, Row: 40, Xpixel: 0, Ypixel: 0},
			&pty.Winsize{Cols: 80, Rows: 40, X: 0, Y: 0},
		},
		{"nil case", nil, nil},
	}

	for _, v := range tc {
		got := ConvertWinsize(v.win)

		if (v.expect != nil) && (*got != *v.expect) {
			t.Errorf("#test %q expect %v, got %v\n", v.label, v.expect, got)
		}

		if v.expect == nil && got != nil {
			t.Errorf("#test %q expect %v, got %v\n", v.label, v.expect, got)
		}
	}
}
