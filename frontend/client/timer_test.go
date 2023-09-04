// Copyright 2022~2023 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package main

import (
	"testing"
	"time"
)

func TestTimer(t *testing.T) {
	tc := []struct {
		label   string
		timeout int
		expect  int64
	}{
		{"positive timer", 5, 5},
		{"zero timer", 0, 0},
		{"negative timer", -2000, 0}, // negative value means triger timer immediately
	}

	for _, v := range tc {
		t.Run(v.label, func(t *testing.T) {
			timer := time.NewTimer(time.Duration(v.timeout) * time.Millisecond)
			now := time.Now().UnixMilli()

			<-timer.C
			x := time.Now()

			got := x.UnixMilli() - now
			if !(got-v.expect == 0 || got-v.expect == 1) { // asllow 1ms deviation
				t.Errorf("#test %s expect %d, got %d\n", v.label, v.expect, got)
			}
		})
	}
}
