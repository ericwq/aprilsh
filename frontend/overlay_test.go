/*

MIT License

Copyright (c) 2022 wangqi

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.

*/

package frontend

import (
	"fmt"
	"testing"

	"github.com/ericwq/aprilsh/terminal"
)

func TestOverlay(t *testing.T) {
	co := NewConditionalOverlay(12, 2, 14)

	if co.tentative(15) {
		t.Errorf("expect %t, got %t\n", true, co.tentative(15))
	}

	co.expire(13, 14)
	if co.expirationFrame != 13 || co.predictionTime != 14 {
		t.Errorf("expire() expirationFrame expect %d, got %d\n", 13, co.expirationFrame)
		t.Errorf("expire() predictionTime expect %d, got %d\n", 14, co.predictionTime)
	}

	co.reset()
	if co.expirationFrame != -1 || co.tentativeUntilEpoch != -1 || co.active != false {
		t.Errorf("reset() expirationFrame should be %d, got %d\n", -1, co.expirationFrame)
	}
}

func TestMoveApply(t *testing.T) {
	tc := []struct {
		name           string
		activeParam    bool
		confirmedEpoch int64
		posY, posX     int
	}{
		{"apply() active=T, tentative return F", true, 15, 4, 10},
		{"apply() active=F", false, 15, 0, 0},
		{"apply() active=T, tentative return T", true, 14, 0, 0},
	}
	emu := terminal.NewEmulator3(80, 40, 40)
	ccm := NewConditionalCursorMove(12, 4, 10, 15)

	for _, v := range tc {
		emu.MoveCursor(0, 0) // default cursor position for early return.
		ccm.active = v.activeParam
		ccm.apply(emu, v.confirmedEpoch)
		posY := emu.GetCursorRow()
		posX := emu.GetCursorCol()
		if posX != v.posX || posY != v.posY {
			t.Errorf("%s posY expect %d, got %d\n", v.name, v.posY, posY)
			t.Errorf("%s posX expect %d, got %d\n", v.name, v.posX, posX)
		}
	}
}

func TestMoveGetValidity(t *testing.T) {
	tc := []struct {
		name            string
		lateAck         int64
		expirationFrame int64
		active          bool
		rowEmu, colEmu  int
		rowCcm, colCcm  int
		validity        Validity
	}{
		{"getValidity() active=T, row,col in scope, lateAck >=expirationFrame", 20, 15, true, 10, 10, 10, 10, Correct},
		{"getValidity() active=T, row,col outof scope", 20, 15, true, 10, 10, 50, 50, IncorrectOrExpired},
		{"getValidity() active=T, row,col not equal, lateAck >=expirationFrame", 20, 20, true, 10, 12, 10, 10, IncorrectOrExpired},
		{"getValidity() active=T, row,col in scope, lateAck < expirationFrame", 20, 21, true, 10, 10, 10, 10, Pending},
		{"getValidity() active=F", 20, 21, false, 10, 10, 10, 10, Inactive},
	}

	emu := terminal.NewEmulator3(80, 40, 40)

	for _, v := range tc {
		emu.MoveCursor(v.rowEmu, v.colEmu)
		ccm := NewConditionalCursorMove(v.expirationFrame, v.rowCcm, v.colCcm, 12)
		ccm.active = v.active
		validity := ccm.getValidity(emu, v.lateAck)
		if validity != v.validity {
			t.Errorf("%q getValidity() expect %d, got %d\n", v.name, v.validity, validity)
		}
	}
}

func TestCellApply(t *testing.T) {
	underlineRend := terminal.NewRendition(4) // renditions with underline attribute
	underlineCell := terminal.Cell{}
	underlineCell.SetRenditions(underlineRend)
	plainCell := terminal.Cell{}

	tc := []struct {
		name           string
		active         bool
		confirmedEpoch int64
		flag           bool
		row, col       int
		unknown        bool
		contents       rune
		rend           *terminal.Renditions
		cell           *terminal.Cell
	}{
		{"active=T flag=T unknow=F update cell and rendition", true, 20, true, 10, 10, false, 'E', &underlineRend, &underlineCell},
		{"active=T flag=F unknow=F update cell", true, 20, false, 11, 10, false, 'E', nil, &plainCell},
		{"active=T flag=T unknow=T update rendition", true, 20, true, 12, 10, true, 'E', &underlineRend, nil},
		{"active=T flag=F unknow=T return", true, 20, false, 13, 10, true, 'E', nil, nil},
		{"active=T flag=T unknow=T return", true, 20, true, 14, 10, true, '\x00', nil, nil},
		{"tentative early return", true, 9, true, 14, 10, true, 'E', nil, nil},
		{"active early return", false, 10, true, 14, 10, true, 'E', nil, nil},
	}

	emu := terminal.NewEmulator3(80, 40, 40)
	for _, v := range tc {
		predict := NewConditionalOverlayCell(10, v.col, 10)

		predict.active = v.active
		predict.unknown = v.unknown
		// set content for emulator cell
		if v.contents != '\x00' {
			emu.GetMutableCell(v.row, v.col).Append(v.contents)
		}

		// call apply
		predict.apply(emu, v.confirmedEpoch, v.row, v.flag)

		// validate cell
		cell := emu.GetCell(v.row, v.col)
		if v.cell != nil && cell != *(v.cell) {
			t.Errorf("%q cell (%d,%d) contents expect\n%v\ngot \n%v\n", v.name, v.row, v.col, *v.cell, cell)
		}

		// validate rendition
		rend := emu.GetCell(v.row, v.col).GetRenditions()
		if v.rend != nil && rend != *v.rend {
			t.Errorf("%q cell (%d,%d) renditions expect %v, got %v\n", v.name, v.row, v.col, *v.rend, rend)
		}
	}
}

func TestCellGetValidity(t *testing.T) {
	tc := []struct {
		name     string
		active   bool
		row, col int
		lateAck  int64
		unknown  bool
		contents rune
		validity Validity
	}{
		{"active=T, unknown=F, isBlank=F, content match", true, 10, 10, 20, false, 'E', Correct},
	}

	emu := terminal.NewEmulator3(80, 40, 40)
	for _, v := range tc {
		predict := NewConditionalOverlayCell(10, v.col, 10)
		predict.active = v.active
		predict.unknown = v.unknown
		emu.GetMutableCell(v.row, v.col).Append(v.contents)

		validity := predict.getValidity(emu, v.row, v.lateAck)
		if validity != v.validity {
			t.Errorf("%q expect %d, got %d\n", v.name, v.validity, validity)
		}
	}
}

func TestPredictionHandleGrapheme(t *testing.T) {
	tc := []struct {
		name      string
		rawStr    string // rawString will fill the right side of emulator
		row, col  int    // the specified row and col
		insertStr string
	}{
		{"insert 10 runes", "abcdefghij", 4, 69, "ABCDEFGHIJ"},
	}

	pe := NewPredictionEngine()
	emu := terminal.NewEmulator3(80, 40, 40)

	for _, v := range tc {

		// fill in the rawStr to see the result
		emu.MoveCursor(v.row, v.col)
		emu.HandleStream(v.rawStr)
		// for i := 0; i < len(v.insertStr); i++ {
		// 	fmt.Printf("after HandleStream: cell (%d,%d) contains %q\n", v.row, v.col+i, emu.GetCell(v.row, v.col+i))
		// }
		emu.MoveCursor(v.row, v.col)
		for i := range v.insertStr {
			fmt.Printf("%s: insert %c loop %d\n", v.name, v.insertStr[i], i)
			pe.handleGrapheme(emu, rune(v.insertStr[i]))
		}
	}
}
