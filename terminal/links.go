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
	linkSet []link
	nextNum int
}

func newLinks() *links {
	v := &links{}
	v.linkSet = make([]link, 0, 16)
	v.nextNum = 0

	return v
}

func (x *links) AddLink(url string) (ret int) {
	ret = x.nextNum
	x.linkSet = append(x.linkSet, link{url: url, num: x.nextNum})
	x.nextNum++

	sort.Slice(x.linkSet, func(i, j int) bool {
		return x.linkSet[i].num < x.linkSet[j].num
	})
	return
}

func (x *links) ChangeLink(v int, nUrl string) bool {
	idx := slices.IndexFunc(x.linkSet, func(c link) bool { return c.url == nUrl })
	if idx >= 0 {
		x.linkSet[idx].url = nUrl
	}

	return idx >= 0
}

func (x *links) RemoveLink(url string) bool {
	idx := slices.IndexFunc(x.linkSet, func(c link) bool { return c.url == url })
	if idx >= 0 {
		x.linkSet = append(x.linkSet[:idx], x.linkSet[idx+1:]...)
	}

	return idx >= 0
}
