// Copyright 2022~2023 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package network

import (
	"testing"
	"time"

	"github.com/ericwq/aprilsh/statesync"
)

func TestTimestampedStaeGetxxx(t *testing.T) {
	blank := &statesync.UserStream{}

	now := time.Now().UnixMilli()
	expectNum := uint64(4)
	s := TimestampedState[*statesync.UserStream]{state: blank, num: expectNum, timestamp: now}

	if s.GetTimestamp() != now {
		t.Errorf("#test GetTimestamp() expect %d, got %d\n", now, s.GetTimestamp())
	}

	if s.GetState() != blank {
		t.Errorf("#test GetState() expect %v, got %v\n", blank, s.GetState())
	}
}
