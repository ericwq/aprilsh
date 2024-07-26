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

/*
Set cursor style (DECSCUSR), VT520.

	Ps = 0  ⇒  blinking block.
	Ps = 1  ⇒  blinking block (default).
	Ps = 2  ⇒  steady block.
	Ps = 3  ⇒  blinking underline.
	Ps = 4  ⇒  steady underline.
	Ps = 5  ⇒  blinking bar, xterm.
	Ps = 6  ⇒  steady bar, xterm.
*/
const (
	CursorStyle_BlinkBlock CursorStyle = iota
	_hidden_cursor_style
	CursorStyle_SteadyBlock
	CursorStyle_BlinkUnderline
	CursorStyle_SteadyUnderline
	CursorStyle_BlinkBar
	CursorStyle_SteadyBar
	CursorStyle_Invalid
)

type Cursor struct {
	posX      int // current cursor horizontal position (on-screen)
	posY      int // current cursor vertical position (on-screen)
	color     Color
	style     CursorStyle // hidden, fill, hollow
	showStyle CursorStyle // blinking block, steady block, underline etc.
}
