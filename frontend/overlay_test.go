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
		base     string // base content
		predict  string // prediction
		frame    string // frame content
		validity Validity
	}{
		{"active=F, unknown=F", false, 13, 70, 20, false, "", "active", "false", Inactive},                      // active=F
		{"active=T, cursor out of range", true, 41, 70, 0, false, "", "smaller", "lateAck", IncorrectOrExpired}, // cursor out of range
		{"active=T, smaller lateAck", true, 13, 70, 0, false, "", "smaller", "lateAck", Pending},                // smaller lateAck
		{"active=T, unknown=T", true, 13, 70, 20, true, "", "unknow", "true", CorrectNoCredit},                  // unknown=T
		{"active=T, unknown=F, blank predict", true, 13, 70, 20, false, "----", "    ", "some", CorrectNoCredit},
		{"active=T, unknown=F, content match", true, 12, 70, 20, false, "Else", "Easy", "Easy", CorrectNoCredit},
		{"active=T, unknown=T, isBlank=F correct", true, 14, 70, 5, false, "     ", "right", "right", Correct},
		{"active=T, unknown=F, content not match", true, 11, 70, 20, false, "-----", "Alpha", "Beta", IncorrectOrExpired},
	}

	emu := terminal.NewEmulator3(80, 40, 40)
	pe := NewPredictionEngine()

	for _, v := range tc {
		pe.reset()

		// set the base content
		emu.MoveCursor(v.row, v.col)
		emu.HandleStream(v.base)

		// mimic user input for prediction engine
		emu.MoveCursor(v.row, v.col)
		for i := range v.predict {
			pe.handleUserGrapheme(emu, rune(v.predict[i]))
		}

		// mimic the result from server
		emu.MoveCursor(v.row, v.col)
		emu.HandleStream(v.frame)

		// get the predict row
		predictRow := pe.getOrMakeRow(v.row, emu.GetWidth())
		predict := &(predictRow.overlayCells[v.col])

		predict.active = v.active
		predict.unknown = v.unknown

		validity := predict.getValidity(emu, v.row, v.lateAck)
		if validity != v.validity {
			t.Errorf("%q expect %d, got %d\n", v.name, v.validity, validity)
			t.Errorf("cell (%d,%d) replacement=%s, originalContents=%s\n", v.row, v.col, predict.replacement, predict.originalContents)
		}
	}
}

func TestPredictionCull(t *testing.T) {
	tc := []struct {
		name            string
		row, col        int
		base            string // base content
		predict         string // prediction
		frame           string // frame content
		lateAck         int
		confirmedEpoch  int
		incorrectCursor bool
		sendInterval    int
	}{
		{"reset by height and width", 12, 70, "    ", "----", "****", 0, 0, false, 0},
		{"Correct validity", 9, 70, "     ", "right", "right", 2, 0, false, 0},
		{"IncorrectOrExpired validity", 10, 70, "-----", "Alpha", "Beta", 2, 0, false, 0},
		{"Pending validity", 11, 70, "    ", "----", "****", 0, 0, false, 0},
		{"IncorrectOrExpired < confirmedEpoch", 13, 70, "-----", "Alpha", "Beta", 2, 10, false, 0},
		{"Correct validity + incorrect cursor", 14, 70, "     ", "right", "right", 2, 0, true, 0},
		{"Correct validity + sendInterval", 14, 70, "     ", "right", "right", 2, 0, true, 20},
	}
	emu := terminal.NewEmulator3(80, 40, 40)
	pe := NewPredictionEngine()

	for k, v := range tc {
		pe.reset()

		// set the base content
		emu.MoveCursor(v.row, v.col)
		emu.HandleStream(v.base)

		// cell := emu.GetCell(v.row, v.col)
		// fmt.Printf("set base content =%q vs  %q\n", cell, v.base)

		// mimic user input for prediction engine
		emu.MoveCursor(v.row, v.col)
		for i := range v.predict {
			// fmt.Printf("cull() test predictionEpoch=%d\n", pe.predictionEpoch)
			pe.handleUserGrapheme(emu, rune(v.predict[i]))
		}

		// mimic the result from server
		emu.MoveCursor(v.row, v.col)
		emu.HandleStream(v.frame)

		// set the lateAck
		pe.localFrameLateAcked = int64(v.lateAck)

		// get the predict cell
		predictRow := pe.getOrMakeRow(v.row, emu.GetWidth())
		predict := &(predictRow.overlayCells[v.col])

		if v.confirmedEpoch != 0 {
			pe.confirmedEpoch = int64(v.confirmedEpoch)
		}
		if v.incorrectCursor {
			emu.MoveCursor(v.row, v.col-1)
		}
		if v.sendInterval != 0 {
			pe.sendInterval = v.sendInterval
		}

		pe.cull(emu)

		switch k {
		case 0, 4, 5:
			if len(pe.overlays) != 0 || len(pe.cursors) != 0 {
				t.Errorf("%s should be reset by width and height. got overlays=%d, cursors=%d\n", v.name, len(pe.overlays), len(pe.cursors))
			}
		case 1:
			// Correct cell's active is false, it's originalContents field is cleared.
			if predict.active != false && len(predict.originalContents) != 0 {
				t.Errorf("cell (%d,%d) replacement=%s, originalContents=%s\n", v.row, v.col, predict.replacement, predict.originalContents)
				t.Errorf("%s expect empty predict cell, got cell active=%t\n", v.name, predict.active)
			}
		case 2:
			// IncorrectOrExpired cell's active is false
			if predict.active {
				t.Errorf("%s expect empty predict cell, got cell active=%t\n", v.name, predict.active)
			}
		case 3:
			// Pending cell's active is true.
			if !predict.active {
				t.Errorf("%s expect exmpty predict cell, got cell active=%t\n", v.name, predict.active)
			}
		}
	}
}

func TestPredictionNewUserInput(t *testing.T) {
	tc := []struct {
		name              string
		row, col          int    // the specified row and col
		base              string // base content
		predict           string // prediction
		result            string // frame content
		displayPreference DisplayPreference
		posY, posX        int // new cursor position
	}{
		{"insert english", 3, 75, "******", "abcdef", "abcdef", Adaptive, 0, 0},
		{"insert chinese", 4, 70, "", "四姑娘山", "四姑娘山", Adaptive, 4, 0},
		{"Experimental", 4, 60, "", "Experimental", "Experimental", Experimental, 4, 76},
		{"insert CUF", 4, 75, "", "\x1B[C", "", Adaptive, 4, 76},
		{"insert CUB", 4, 75, "", "\x1B[D", "", Adaptive, 4, 74},
		{"insert CR", 4, 75, "", "\r", "", Adaptive, 5, 0},
		{"insert CUF", 4, 75, "", "\x1BOC", "", Adaptive, 4, 76},
		{"BEL becomeTentative", 5, 70, "", "\x07", "", Adaptive, 0, 0},
		{"Never", 4, 75, "", "Never", "", Never, 0, 0},
		{"insert chinese with base content", 6, 71, "上海56789", "四姑娘", "四姑娘上", Adaptive, 0, 0},
		{"insert chinese with wrap", 7, 79, "", "四", "四", Adaptive, 8, 0},
	}

	pe := NewPredictionEngine()
	emu := terminal.NewEmulator3(80, 40, 40)

	for k, v := range tc {
		pe.reset()

		// set the base content
		emu.MoveCursor(v.row, v.col)
		emu.HandleStream(v.base)

		// if k == 9 {
		// 	fmt.Println("k==9")
		// 	for i := 0; i < len(v.base); i++ {
		// 		cell := emu.GetCell(v.row, v.col+i)
		// 		fmt.Printf("cell (%d,%d) is %q dw=%t, dwcont=%t\n", v.row, v.col+i, cell, cell.IsDoubleWidth(), cell.IsDoubleWidthCont())
		// 	}
		// }

		// set the displayPreference field
		pe.displayPreference = v.displayPreference

		// mimic user input for prediction engine
		emu.MoveCursor(v.row, v.col)
		pe.newUserInput(emu, v.predict)

		switch k {
		case 0, 1, 2, 9:
			// get the predict cell
			predictRow := pe.getOrMakeRow(v.row, emu.GetWidth())
			i := 0
			for _, ch := range v.result {
				if v.col+i > emu.GetWidth()-1 {
					break
				}

				cell := predictRow.overlayCells[v.col+i].replacement
				if cell.String() != string(ch) {
					t.Errorf("%s expect %q at (%d,%d), got %q\n", v.name, string(ch), v.row, v.col+i, cell)
					t.Errorf("predict cell (%d,%d) is %q dw=%t, dwcont=%t\n", v.row, v.col+i, cell, cell.IsDoubleWidth(), cell.IsDoubleWidthCont())
				}
				i += terminal.RunesWidth([]rune{ch})
			}
		case 3, 4, 5, 6:
			gotX := pe.cursor().col
			gotY := pe.cursor().row
			if gotX != v.posX || gotY != v.posY {
				t.Errorf("%s expect cursor at (%d,%d), got (%d,%d)\n", v.name, v.posY, v.posX, gotY, gotX)
			}
		case 10:
			predictRow := pe.getOrMakeRow(v.posY, emu.GetWidth())
			i := 0
			for _, ch := range v.result {
				cell := predictRow.overlayCells[v.posX+i].replacement
				if cell.String() != string(ch) {
					t.Errorf("%s expect %q at (%d,%d), got %q\n", v.name, string(ch), v.posY, v.posX+i, cell)
					t.Errorf("predict cell (%d,%d) is %q dw=%t, dwcont=%t\n", v.posY, v.posX+i, cell, cell.IsDoubleWidth(), cell.IsDoubleWidthCont())
				}
				i += terminal.RunesWidth([]rune{ch})
			}

		default:
		}
	}
}
