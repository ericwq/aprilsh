// Copyright 2022~2024 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package terminal

import (
	"slices"
	"sort"
)

type link struct {
	url string
	num int
}

type links struct {
	linkSet   []link
	linkIndex int
}

func newLinks() *links {
	v := &links{}
	v.linkSet = make([]link, 0, 16)
	v.linkIndex = 1

	return v
}

func (x *links) addLink(url string) (idx int) {
	idx = slices.IndexFunc(x.linkSet, func(c link) bool { return c.url == url })
	if idx > 0 {
		return idx
	}

	idx = x.linkIndex
	x.linkSet = append(x.linkSet, link{url: url, num: x.linkIndex})
	x.linkIndex++

	sort.Slice(x.linkSet, func(i, j int) bool {
		return x.linkSet[i].num < x.linkSet[j].num
	})
	return
}

func (x *links) clone() *links {
	clone := links{}

	clone.linkIndex = x.linkIndex
	clone.linkSet = make([]link, len(x.linkSet))
	copy(clone.linkSet, x.linkSet)

	return &clone
}

// func (x *links) changeLink(nUrl string) bool {
// 	idx := slices.IndexFunc(x.linkSet, func(c link) bool { return c.url == nUrl })
// 	if idx >= 0 {
// 		x.linkSet[idx].url = nUrl
// 	}
//
// 	return idx >= 0
// }
//
// func (x *links) removeLink(url string) bool {
// 	idx := slices.IndexFunc(x.linkSet, func(c link) bool { return c.url == url })
// 	if idx >= 0 {
// 		x.linkSet = append(x.linkSet[:idx], x.linkSet[idx+1:]...)
// 	}
//
// 	return idx >= 0
// }
