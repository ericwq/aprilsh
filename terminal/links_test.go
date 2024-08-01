// Copyright 2022~2024 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package terminal

import (
	"fmt"
	"strings"
	"testing"
)

func TestLinkSet(t *testing.T) {
	tc := []struct {
		label string
		id    string
		url   string
		exist []string
		index int
	}{
		{
			"add new link", "", "http://go.dev",
			[]string{"http://x.y.z", "http://a.b.c"},
			3,
		},
		{
			"got exist link", "2", "http://a.b.c",
			[]string{"http://x.y.z", "http://a.b.c"},
			2,
		},
		{
			"max uri length", "", strings.Repeat("x", maxURILength),
			[]string{"http://x.y.z", "http://a.b.c"},
			3,
		},
		{
			"max id length", strings.Repeat("x", maxIDLength), "http://c.h.i",
			[]string{"http://x.y.z", "http://a.b.c"},
			3,
		},
	}

	for _, v := range tc {
		t.Run(v.label, func(t *testing.T) {
			links := newLinks()

			// add pre-filled link to links
			for i, u := range v.exist {
				id := ""
				if v.id != "" {
					id = fmt.Sprintf("%d", i+1)
				}
				links.addLink(id, u)
			}

			index := links.addLink(v.id, v.url)
			if v.index != index {
				t.Errorf("%s expect index %d, got %d\n", v.label, v.index, index)
			}
		})
	}
}
