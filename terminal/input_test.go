// Copyright 2022~2024 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package terminal

import (
	"strings"
	"testing"

	"github.com/rivo/uniseg"
)

func TestUserByteHandle(t *testing.T) {

	tc := []struct {
		label  string
		input  string
		expect string
	}{
		{"over size   ", "", ""},
		{"english text", "hello", "hello"},
		{"chinese text", "斗罗大陆", "斗罗大陆"},
		{"ESC sequence", "\x88", "�"},
		{"Cursor Up   ", "\x1b[A\x1bOA", "\x1b[A\x1b[A"},
		{"Cursor Down ", "\x1b[B\x1bOB", "\x1b[B\x1b[B"},
		{"Cursor Right", "\x1b[C\x1bOC", "\x1b[C\x1b[C"},
		{"Cursor Left ", "\x1b[D\x1bOD", "\x1b[D\x1b[D"},
		{"NonANSI mode", "\x1bOA\x1bOD", "\x1bOA\x1bOD"},
	}

	emu := NewEmulator3(80, 40, 40) // this is the initialized size.
	for _, v := range tc {
		t.Run(v.label, func(t *testing.T) {
			if strings.HasPrefix(v.label, "NonANSI mode") {
				emu.cursorKeyMode = CursorKeyMode_Application
			} else {
				emu.cursorKeyMode = CursorKeyMode_ANSI
			}

			if strings.HasPrefix(v.label, "over size") {
				empty := UserByte{[]rune(v.label)}
				// fmt.Printf("%v has length %d\n", empty, len(empty.Chs))
				empty.Handle(emu)
			} else {
				// prepare UserByte slice
				graphemes := uniseg.NewGraphemes(v.input)
				ub := make([]UserByte, 0)

				for graphemes.Next() {
					chs := graphemes.Runes()
					ub = append(ub, UserByte{chs})
				}

				// process each UserByte
				for i := range ub {
					ub[i].Handle(emu)
				}
			}
			// validate the result
			got := emu.ReadOctetsToHost()
			if got != v.expect {
				t.Errorf("expect %q got %q\n", v.expect, got)
			}
		})
	}
}

func TestResizeHandle(t *testing.T) {
	type Result struct {
		nCols, nRows      int
		hMargin, nColsEff int
	}
	tc := []struct {
		label        string
		nCols, nRows int
		expect       Result
	}{
		{"extend both", 92, 52, Result{92, 52, 0, 92}},
	}

	emu := NewEmulator3(80, 40, 40) // this is the initialized size.

	for _, v := range tc {
		t.Run(v.label, func(t *testing.T) {
			resize := Resize{v.nCols, v.nRows}
			resize.Handle(emu)

			if v.expect.nCols != emu.nCols || v.expect.nRows != emu.nRows ||
				v.expect.hMargin != emu.hMargin || v.expect.nColsEff != emu.nColsEff {
				t.Errorf("%q expect %v, got (%d,%d,%d,%d)\n",
					v.label, v.expect, emu.nCols, emu.nRows, emu.hMargin, emu.nColsEff)
			}
		})
	}
}
