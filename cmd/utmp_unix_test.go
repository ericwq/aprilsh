// Copyright 2022 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

//go:build linux

package main

import (
	"testing"

	utmp "github.com/ericwq/goutmp"
)

func TestHasUtmpSupport(t *testing.T) {
	fp = mockGetUtmpx0
	defer func() {
		fp = utmp.GetUtmpx
		idx = 0
	}()

	got := hasUtmpSupport()
	if got {
		t.Errorf("#test hasUtmpSupport() expect %t, got %t\n", false, got)
	}
}
