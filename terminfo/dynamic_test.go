// Copyright 2022 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package terminfo

import (
	"io"
	"testing"

	"github.com/ericwq/aprilsh/util"
)

func TestLookupCap(t *testing.T) {
	tc := []struct {
		label  string
		names  []string
		values []string
	}{}

	util.Logger.CreateLogger(io.Discard, false, util.LevelTrace)

	for _, v := range tc {
		t.Run(v.label, func(t *testing.T) {
			for i := range v.names {
			}
			// if strings.Contains(result, "2026l") {
			// 	t.Errorf("%s got warn log \n%s", v.label, result)
			// }
		})
	}
}
