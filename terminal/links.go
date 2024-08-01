// Copyright 2022~2024 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package terminal

import (
	"slices"
)

const (
	maxURILength = 2083
	maxIDLength  = 250
)

/*
Terminal emulators traditionally use maybe a dozen or so bytes per cell. Adding hyperlinks
potentially increases it by magnitudes. As such, it's tricky to implement this feature in
terminal emulators (without consuming way too much memory), and they probably want to expose
some safety limits.

Both VTE and iTerm2 limit the URI to 2083 bytes. There's no de jure limit, the de facto is
2000-ish. Internet Explorer supports 2083.

VTE currently limits the id to 250 bytes. It's subject to change without notice, and you
should most definitely not rely on this particular number. Utilities are kindly requested
to stay way below this limit, so that a few layers of intermediate software that need to
mangle the id (e.g. add a prefix denoting their window/pane ID) still stay safe. Of course
such intermediate layers are also kindly requested to keep their added prefix at a reasonable
size. There's no limit for the id's length in iTerm2.
*/

type link struct {
	url   string
	id    string
	index int
}

type linkSet struct {
	links     []link
	nextIndex int
}

func newLinks() *linkSet {
	v := &linkSet{}
	v.links = make([]link, 0, 8)
	v.nextIndex = 1

	return v
}

// return exist link index or return new link index
func (x *linkSet) addLink(id string, url string) (index int) {
	if len(url) > maxURILength-1 {
		url = url[:maxURILength-1]
	}

	if len(id) > maxIDLength-1 {
		id = id[:maxIDLength-1]
	}

	if len(x.links) > 0 {
		idx := slices.IndexFunc(x.links, func(l link) bool {
			return l.url == url && l.id == id
		})

		if idx != -1 {
			return x.links[idx].index
		}
	}
	index = x.nextIndex
	x.links = append(x.links, link{id: id, url: url, index: index})
	x.nextIndex++

	return
}

func (x *linkSet) clone() *linkSet {
	clone := linkSet{}

	clone.nextIndex = x.nextIndex
	clone.links = make([]link, len(x.links))
	copy(clone.links, x.links)

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
