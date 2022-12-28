// Copyright 2022 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package terminal

type CursorStyle uint

const (
	CursorStyle_Hidden CursorStyle = iota
	CursorStyle_FillBlock
	CursorStyle_HollowBlock
)

type Cursor struct {
	posX  int // current cursor horizontal position (on-screen)
	posY  int // current cursor vertical position (on-screen)
	color Color
	style CursorStyle
}
