// Copyright 2022~2023 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package frontend

import (
	"fmt"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/ericwq/aprilsh/terminal"
	"github.com/rivo/uniseg"
)

func TestOverlay(t *testing.T) {
	co := newConditionalOverlay(12, 2, 14)

	if co.tentative(15) {
		t.Errorf("expect %t, got %t\n", true, co.tentative(15))
	}

	co.expire(13, 14)
	if co.expirationFrame != 13 || co.predictionTime != 14 {
		t.Errorf("expire() expirationFrame expect %d, got %d\n", 13, co.expirationFrame)
		t.Errorf("expire() predictionTime expect %d, got %d\n", 14, co.predictionTime)
	}

	co.reset()
	if co.expirationFrame != math.MaxUint64 || co.tentativeUntilEpoch != math.MaxInt64 || co.active != false {
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
	ccm := newConditionalCursorMove(12, 4, 10, 15)

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
		lateAck         uint64
		expirationFrame uint64
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
		ccm := newConditionalCursorMove(v.expirationFrame, v.rowCcm, v.colCcm, 12)
		ccm.active = v.active
		validity := ccm.getValidity(emu, v.lateAck)
		if validity != v.validity {
			t.Errorf("%q getValidity() expect %d, got %d\n", v.name, v.validity, validity)
		}
	}
}

func TestCellApply(t *testing.T) {
	underlineRend := terminal.NewRenditions(4) // renditions with underline attribute
	underlineCell := terminal.Cell{}
	underlineCell.SetRenditions(underlineRend)
	plainCell := terminal.Cell{}

	tc := []struct {
		rend           *terminal.Renditions
		cell           *terminal.Cell
		name           string
		confirmedEpoch int64
		row            int
		col            int
		contents       rune
		active         bool
		flag           bool
		unknown        bool
	}{
		{&underlineRend, &underlineCell, "active=T flag=T unknown=F update cell and rendition", 20, 10, 10, 'E', true, true, false},
		{nil, &plainCell, "active=T flag=F unknown=F update cell", 20, 11, 10, 'E', true, false, false},
		{&underlineRend, nil, "active=T flag=T unknown=T update rendition", 20, 12, 10, 'E', true, true, true},
		{nil, nil, "active=T flag=F unknown=T return", 20, 13, 10, 'E', true, false, true},
		{nil, nil, "active=T flag=T unknown=T return", 14, 10, 20, '\x00', true, true, true},
		{nil, nil, "tentative early return", 9, 14, 10, 'E', true, true, true},
		{nil, nil, "active early return", 10, 14, 10, 'E', false, true, true},
	}

	emu := terminal.NewEmulator3(80, 40, 40)
	for _, v := range tc {
		predict := newConditionalOverlayCell(10, v.col, 10)

		predict.active = v.active
		predict.unknown = v.unknown
		// set content for emulator cell
		if v.contents != '\x00' {
			emu.GetCellPtr(v.row, v.col).Append(v.contents)
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
		base     string // base content
		predict  string // prediction
		frame    string // frame content
		row      int
		col      int
		lateAck  uint64
		validity Validity
		active   bool
		unknown  bool
	}{
		// the test case only check the first cell in babse, prediction and frame
		{
			"active=F, unknown=F", "", "active", "false",
			13, 70, 20, Inactive, false, false,
		}, // active is false
		{
			"active=T, cursor out of range", "", "smaller", "lateAck",
			41, 70, 0, IncorrectOrExpired, true, false,
		}, // row out of range
		{
			"active=T, smaller lateAck", "", "smaller", "lateAck",
			13, 70, 0, Pending, true, false,
		}, // smaller lateAck
		{
			"active=T, unknown=T", "", "unknow", "true",
			13, 70, 20, CorrectNoCredit, true, true,
		}, // unknown=T
		{
			"active=T, unknown=F, blank predict", "----", "    ", "some",
			13, 70, 20, CorrectNoCredit, true, false,
		}, // blank prediction
		{
			"active=T, unknown=F, found original", "Else", "Else", "Else",
			12, 70, 20, CorrectNoCredit, true, false,
		}, // found original
		{
			"active=T, unknown=T, isBlank=F correct", "     ", "right", "right",
			14, 70, 5, Correct, true, false,
		}, // not found original
		{
			"active=T, unknown=F, content not match", "-----", "Alpha", "Beta",
			11, 70, 20, IncorrectOrExpired, true, false,
		},
	}

	emu := terminal.NewEmulator3(80, 40, 40)
	pe := newPredictionEngine()

	for _, v := range tc {
		t.Run(v.name, func(t *testing.T) {
			pe.Reset()

			// set the base content
			emu.MoveCursor(v.row, v.col)
			emu.HandleStream(v.base)

			// mimic user input for prediction engine
			emu.MoveCursor(v.row, v.col)
			now := time.Now().UnixMilli()
			for i := range v.predict {
				pe.handleUserGrapheme(emu, now, rune(v.predict[i]))
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
		})
	}
}

func TestPredictionNewUserInput_Normal(t *testing.T) {
	tc := []struct {
		label             string
		base              string // base content
		predict           string // prediction
		result            string // frame content
		row, col          int    // the specified row and col
		displayPreference DisplayPreference
		posY, posX        int  // new cursor position, 0 means doesn't matter
		predictOverwrite  bool // predictOverwrite
	}{
		/* 0*/ {
			"insert english", "12345", "abcde", "abcde", 3, 75,
			Adaptive, -1, -1, false,
		},
		/* 1*/ {
			"insert chinese", "", "四姑娘山", "四姑娘山", 4, 70,
			Adaptive, -1, -1, false,
		},
		/* 2*/ {
			"Experimental", "", "Experimental", "Experimental", 4, 60,
			Experimental, -1, -1, false,
		},
		/* 3*/ {
			"insert CUF", "", "\x1B[C", "", 4, 75,
			Adaptive, 4, 76, false,
		},
		/* 4*/ {
			"insert CUB", "", "\x1B[D", "", 4, 75,
			Adaptive, 4, 74, false,
		},
		/* 5*/ {
			"insert CR", "", "\r", "", 4, 75,
			Adaptive, 5, 0, false,
		},
		/* 6*/ {
			"insert CUF", "", "\x1BOC", "", 4, 75,
			Adaptive, 4, 76, false,
		},
		/* 7*/ {
			"BEL becomeTentative", "", "\x07", "", 5, 70,
			Adaptive, -1, -1, false,
		},
		/* 8*/ {
			"Never", "", "Never", "", 4, 75,
			Never, 0, 0, false,
		},
		/* 9*/ {
			"insert chinese with base contents", "上海56789", "四姑娘", "四姑娘上", 6, 71,
			Adaptive, -1, -1, false,
		},
		/*10*/ {
			"insert chinese with wrap", "", "四", "四", 7, 79,
			Adaptive, 8, 0, false,
		},
		/*11*/ {
			"insert control becomeTentative", "", "\x11", "", 9, 0,
			Adaptive, -1, -1, false,
		},
		/*12*/ {
			"insert overwrite", "12345", "abcde", "abcde", 10, 75,
			Adaptive, -1, -1, true,
		},
	}

	pe := newPredictionEngine()
	emu := terminal.NewEmulator3(80, 40, 40)

	for k, v := range tc {
		t.Run(v.label, func(t *testing.T) {
			pe.Reset()

			// set the base content
			emu.MoveCursor(v.row, v.col)
			emu.HandleStream(v.base)

			// set the displayPreference field
			pe.displayPreference = v.displayPreference
			pe.predictOverwrite = v.predictOverwrite

			// mimic user input for prediction engine
			emu.MoveCursor(v.row, v.col)
			epoch := pe.predictionEpoch
			pe.inputString(emu, v.predict)

			switch k {
			case 0, 1, 2, 9, 12:
				// validate the result against predict cell
				predictRow := pe.getOrMakeRow(v.row, emu.GetWidth())
				i := 0
				for _, ch := range v.result {
					if v.col+i > emu.GetWidth()-1 {
						break
					}

					cell := predictRow.overlayCells[v.col+i].replacement
					if cell.String() != string(ch) {
						t.Errorf("%s expect %q at (%d,%d), got %q\n", v.label, string(ch), v.row, v.col+i, cell)
						t.Errorf("predict cell (%d,%d) is %q dw=%t, dwcont=%t\n", v.row, v.col+i, cell, cell.IsDoubleWidth(), cell.IsDoubleWidthCont())
					}
					i += uniseg.StringWidth(string([]rune{ch}))
				}
			case 3, 4, 5, 6:
				// validate the cursor position
				gotX := pe.cursor().col
				gotY := pe.cursor().row
				if gotX != v.posX || gotY != v.posY {
					t.Errorf("%s expect cursor at (%d,%d), got (%d,%d)\n", v.label, v.posY, v.posX, gotY, gotX)
				}
			case 10:
				// validate the result against predict cell in target row
				predictRow := pe.getOrMakeRow(v.posY, emu.GetWidth())
				i := 0
				for _, ch := range v.result {
					cell := predictRow.overlayCells[v.posX+i].replacement
					if cell.String() != string(ch) {
						t.Errorf("%s expect %q at (%d,%d), got %q\n", v.label, string(ch), v.posY, v.posX+i, cell)
						t.Errorf("predict cell (%d,%d) is %q dw=%t, dwcont=%t\n", v.posY, v.posX+i, cell, cell.IsDoubleWidth(), cell.IsDoubleWidthCont())
					}
					i += uniseg.StringWidth(string([]rune{ch}))
				}
			case 11, 7:
				// validate predictionEpoch
				if pe.predictionEpoch-epoch != 1 {
					t.Errorf("%q expect %d, got %d, %d->%d\n", v.label, 1, pe.predictionEpoch-epoch, epoch, pe.predictionEpoch)
				}
			case 8:
				// Never do nothing, just ignore it.
			default:
				t.Errorf("#test %q test failure. check the test case number.\n", v.label)
			}
		})
	}
}

func TestPredictionApply(t *testing.T) {
	tc := []struct {
		name        string
		base        string // base content
		predict     string // prediction
		result      string // frame content
		row, col    int    // the specified row and col
		earlyReturn bool   // apply early return
	}{
		/*01*/ {"apply wrapped english input", "", "abcdef", "abcdef", 9, 75, false},
		/*02*/ {"apply wrapped chinese input", "", "柠檬水", "柠檬水", 10, 75, false},
		/*03*/ {"apply early return", "", "early return", "early return", 11, 70, true},
	}

	pe := newPredictionEngine()
	emu := terminal.NewEmulator3(80, 40, 40)

	for k, v := range tc {
		pe.Reset()

		// set the base content
		emu.MoveCursor(v.row, v.col)
		emu.HandleStream(v.base)

		if v.earlyReturn {
			pe.SetDisplayPreference(Never)
		}

		// mimic user input for prediction engine
		emu.MoveCursor(v.row, v.col)
		pe.inputString(emu, v.predict)
		// predictRow := pe.getOrMakeRow(v.row+1, emu.GetWidth())
		// predict := predictRow.overlayCells[0].replacement
		// t.Logf("%q overlay at (%d,%d) is %q\n", v.name, v.row+1, 0, predict.GetContents())

		// mimic the result from server
		emu.MoveCursor(v.row, v.col)
		emu.HandleStream(v.result)
		// cell := emu.GetMutableCell(v.row+1, 0) // cr to next row
		// t.Logf("%q emulator at (%d,%d) is %q @%p\n", v.name, v.row+1, 0, cell.GetContents(), cell)

		// apply to emulator
		pe.cull(emu)
		pe.apply(emu)
		// t.Logf("%q apply at (%d,%d) is %q @%p\n", v.name, v.row+1, 0, cell.GetContents(), cell)

		switch k {
		case 0:
			for i := 0; i < 5; i++ {
				cell := emu.GetCell(v.row, v.col+i)
				if string(v.predict[i]) != cell.GetContents() {
					t.Errorf("%q expect %q at (%d,%d), got %q\n", v.name, v.predict[i], v.row, v.col+i, cell.GetContents())
				}
			}

			cell := emu.GetCell(v.row+1, 0) // cr to next row
			if string(v.predict[5]) != cell.GetContents() {
				t.Errorf("%q expect %q at (%d,%d), got %q\n", v.name, v.predict[5], v.row+1, 0, cell.GetContents())
			}
		case 1:
			i := 0
			for _, ch := range "柠檬" {
				cell := emu.GetCell(v.row, v.col+i*2)
				if string(ch) != cell.GetContents() {
					t.Errorf("%q expect %q at (%d,%d), got %q\n", v.name, ch, v.row, v.col+i*2, cell.GetContents())
				}
				i++
			}
			cell := emu.GetCell(v.row+1, 0) // cr to next row
			if cell.GetContents() != `水` {
				t.Errorf("%q expect %q at (%d,%d), got %q\n", v.name, "水", v.row+1, 0, cell.GetContents())
			}
		case 2: // early return does nothing.
		}
	}
}

func printEmulatorCell(emu *terminal.Emulator, row, col int, sample string, prefix string) {
	graphemes := uniseg.NewGraphemes(sample)
	i := 0
	for graphemes.Next() {
		chs := graphemes.Runes()

		cell := emu.GetCellPtr(row, col+i)
		fmt.Printf("%s # cell %p (%d,%d) is %q\n", prefix, cell, row, col+i, cell)
		i += uniseg.StringWidth(string(chs))
	}
}

func printPredictionCell(emu *terminal.Emulator, pe *PredictionEngine, row, col int, sample string, prefix string) {
	predictRow := pe.getOrMakeRow(row, emu.GetWidth())
	graphemes := uniseg.NewGraphemes(sample)
	i := 0
	for graphemes.Next() {
		chs := graphemes.Runes()
		predict := &(predictRow.overlayCells[col+i])
		fmt.Printf("%s # predict cell %p (%d,%d) is %q active=%t, unknown=%t\n",
			prefix, predict, row, col+i, predict.replacement, predict.active, predict.unknown)
		i += uniseg.StringWidth(string(chs))
	}
}

func TestPrediction_NewUserInput_Backspace(t *testing.T) {
	tc := []struct {
		label          string
		base           string
		predict        string
		expect         string
		row            int
		col            int
		lateAck        uint64
		confirmedEpoch int64
	}{
		{"input backspace for simple cell", "", "abcde\x1B[D\x1B[D\x1B[D\x7f", "acde", 0, 70, 0, 4},
		{"input backspace for wide cell", "", "abc太学生\x1B[D\x1B[D\x1B[D\x1B[C\x7f", "abc学生", 1, 60, 0, 4},
		{"input backspace for wide cell with base", "东部战区", "\x1B[C\x1B[C\x7f", "东战区", 2, 60, 0, 5},
		{"move cursor right, wide cell right edge", "平潭", "\x1B[C\x1B[C", "平潭", 3, 76, 0, 5},
		{"move cursor left, wide cell left edge", "三号木", "\x1B[C\x1B[D\x1B[D", "三号木", 4, 0, 0, 5},
		{"input backspace left edge", "小鸡腿", "\x1B[C\x7f\x7f", "鸡腿", 5, 0, 0, 8},
		{"input backspace unknown case", "", "gocto\x1B[D\x1B[D\x7f\x7f", "gto", 6, 74, 0, 4},
		{"backspace, predict unknown case", "", "捉鹰打goto\x7f\x7f\x7f\x7f鸟", "捉鹰打鸟", 7, 60, 0, 4},
	}

	emu := terminal.NewEmulator3(80, 40, 40) // TODO why we can't init emulator outside of for loop
	pe := newPredictionEngine()

	for _, v := range tc {
		t.Run(v.label, func(t *testing.T) {
			pe.Reset()
			// t.Logf("%q predictionEpoch=%d\n", v.name, pe.predictionEpoch)
			pe.predictionEpoch = 1 // TODO: when it's time to update predictionEpoch?
			// fmt.Printf("%s base=%q expect=%q, pos=(%d,%d)\n", v.label, v.base, v.expect, emu.GetCursorRow(), emu.GetCursorCol())

			// set the base content
			emu.MoveCursor(v.row, v.col)
			emu.HandleStream(v.base)
			// printEmulatorCell(emu, v.row, v.col, v.expect, "After Base")

			// mimic user input for prediction engine
			emu.MoveCursor(v.row, v.col)
			pe.localFrameLateAcked = v.lateAck
			pe.inputString(emu, v.predict)
			// printPredictionCell(emu, pe, v.row, v.col, v.expect, "Predict")

			// merge the last predict
			pe.cull(emu)
			// printPredictionCell(emu, pe, v.row, v.col, v.expect, "After Cull")
			pe.confirmedEpoch = v.confirmedEpoch
			pe.apply(emu)
			// printEmulatorCell(emu, v.row, v.col, v.expect, "Merge")

			// predictRow := pe.getOrMakeRow(v.row, emu.GetWidth())
			i := 0
			graphemes := uniseg.NewGraphemes(v.expect)
			for graphemes.Next() {
				chs := graphemes.Runes()

				cell := emu.GetCell(v.row, v.col+i)
				// fmt.Printf("#test %s (%d,%d) is %s\n", v.label, v.row, v.col+i, cell)
				// predict := predictRow.overlayCells[v.col+i].replacement
				if cell.String() != string(chs) {
					t.Errorf("%s expect %q at (%d,%d), got cell %q dw=%t, dwcont=%t\n",
						v.label, string(chs), v.row, v.col+i, cell, cell.IsDoubleWidth(), cell.IsDoubleWidthCont())
				}

				i += uniseg.StringWidth(string(chs))
			}
		})
	}
}

func TestPrediction_NewUserInput_Backspace_Overwrite(t *testing.T) {
	tc := []struct {
		label          string
		base           string // base content
		predict        string // prediction
		expect         string // the expect content
		row, col       int    // the specified row and col
		lateAck        uint64 // lateAck control the pending result
		confirmedEpoch int64  // this control the appply result
	}{
		{
			"input backspace for simple cell", "", "abcde\x1B[D\x1B[D\x1B[D\x7f", "a cde",
			0, 70, 0, 4,
		},
		{
			"input backspace for wide cell", "", "abc太学生\x1B[D\x1B[D\x1B[D\x1B[C\x7f", "abc  学生",
			1, 60, 0, 4,
		},
		{
			"input backspace for wide cell with base", "东部战区", "\x1B[C\x1B[C\x7f", "东  战区",
			2, 60, 0, 5,
		},
		{
			"move cursor right, wide cell right edge", "平潭", "\x1B[C\x1B[C", "平潭",
			3, 76, 0, 5,
		},
		{
			"move cursor left, wide cell left edge", "三号木", "\x1B[C\x1B[D\x1B[D", "三号木",
			0, 5, 4, 0,
		},
		{
			"input backspace left edge", "小鸡腿", "\x1B[C\x7f", "  鸡腿",
			5, 0, 0, 8,
		},
		{
			"input backspace unknown case", "", "gocto\x1B[D\x1B[D\x7f\x7f", "g  to",
			6, 74, 0, 4,
		},
		{
			"backspace, predict unknown case", "", "捉鹰打goto\x7f\x7f\x7f\x7f鸟", "捉鹰打鸟",
			7, 60, 0, 4,
		},
	}

	emu := terminal.NewEmulator3(80, 40, 40)
	pe := newPredictionEngine()
	pe.SetPredictOverwrite(true) // set predict overwrite

	for _, v := range tc {
		t.Run(v.label, func(t *testing.T) {
			pe.Reset()
			pe.predictionEpoch = 1
			// fmt.Printf("%s base=%q expect=%q, pos=(%d,%d)\n", v.label, v.base, v.expect, v.row, v.col)

			// set the base content
			emu.MoveCursor(v.row, v.col)
			emu.HandleStream(v.base)
			// printEmulatorCell(emu, v.row, v.col, v.expect, "Base row")

			// mimic user input for prediction engine
			emu.MoveCursor(v.row, v.col)
			pe.localFrameLateAcked = v.lateAck
			pe.inputString(emu, v.predict)
			// printPredictionCell(emu, pe, v.row, v.col, v.expect, "Predict row")

			// merge the last predict
			pe.cull(emu)
			// printPredictionCell(emu, pe, v.row, v.col, v.expect, "After Cull")
			pe.confirmedEpoch = v.confirmedEpoch
			pe.apply(emu)
			// printEmulatorCell(emu, v.row, v.col, v.expect, "Apply merge")

			// predictRow := pe.getOrMakeRow(v.row, emu.GetWidth())
			i := 0
			graphemes := uniseg.NewGraphemes(v.expect)
			for graphemes.Next() {
				chs := graphemes.Runes()

				cell := emu.GetCell(v.row, v.col+i)
				// fmt.Printf("#test %q cell    (%d,%d),cell=%s\n", v.label, v.row, v.col+i, cell)
				if cell.String() != string(chs) {
					t.Errorf("%s expect %q at (%d,%d), got cell %q dw=%t, dwcont=%t\n",
						v.label, string(chs), v.row, v.col+i, cell, cell.IsDoubleWidth(), cell.IsDoubleWidthCont())
				}

				i += uniseg.StringWidth(string(chs))
			}
		})
	}
}

func TestPredictionActive(t *testing.T) {
	tc := []struct {
		name     string
		row, col int
		content  rune
		result   bool
	}{
		{"no cursor,  no cell prediction", -1, -1, ' ', false}, // test active()
		{"no cursor, has cell prediction", 1, 0, ' ', true},    // test active()
		{"has cursor, no cell", 3, 1, ' ', true},               // test active()
		{"no cursor, has cell", 2, 0, 'n', true},               // test cursor()
	}

	pe := newPredictionEngine()
	emu := terminal.NewEmulator3(80, 40, 40)

	for k, v := range tc {
		pe.Reset()

		switch v.col {
		case 0:
			// add cell for col==0
			predictRow := pe.getOrMakeRow(v.row, emu.GetWidth())
			predict := &(predictRow.overlayCells[v.col])
			predict.active = true
			predict.replacement = terminal.Cell{}
			predict.replacement.SetContents([]rune{v.content})
		case 1:
			// add cursor for col==1
			pe.initCursor(emu)
		}

		switch v.content {
		case 'n':
			got := pe.cursor()
			if got != nil {
				t.Errorf("%q expect nil,got %p\n", v.name, got)
			}
		default:
			got := pe.active()
			if got != v.result {
				t.Errorf("%q expect %t, got %t\n", v.name, v.result, got)
			}

			// jump the queue for waitTime() test case
			if k == 1 {
				// this is the perfect time to add waitTime test case
				if pe.waitTime() != 50 {
					t.Errorf("%q expect waitTime = %d, got %d\n", v.name, 50, pe.waitTime())
				}
			}
		}
	}
}

func TestPredictionNewlineCarriageReturn(t *testing.T) {
	tc := []struct {
		name       string
		predict    string
		posY, posX int
		gotY, gotX int
	}{
		{"normal CR", "CR\x0D", 2, 3, 3, 0},
		{"bottom CR", "CR\x0D", 39, 0, 39, 0}, // TODO gap is too big, why?
	}
	pe := newPredictionEngine()
	emu := terminal.NewEmulator3(80, 40, 40)

	for _, v := range tc {
		pe.Reset()
		pe.predictionEpoch = 1 // reset it

		// mimic user input for prediction engine
		emu.MoveCursor(v.posY, v.posX)
		pe.inputString(emu, v.predict)
		pe.cull(emu)

		// validate the cursor position
		gotX := pe.cursor().col
		gotY := pe.cursor().row
		if gotX != v.gotX || gotY != v.gotY {
			t.Errorf("%s expect cursor at (%d,%d), got (%d,%d)\n", v.name, v.gotY, v.gotX, gotY, gotX)
		}
	}
}

func printCursors(pe *PredictionEngine, prefix string) {
	for i, cursor := range pe.cursors {
		fmt.Printf("%q #cursor at (%d,%d) %p active=%t, tentativeUntilEpoch=%d\n",
			prefix, cursor.row, cursor.col, &(pe.cursors[i]), cursor.active, cursor.tentativeUntilEpoch)
	}
	fmt.Printf("%q done\n\n", prefix)
}

func TestPredictionKillEpoch(t *testing.T) {
	tc := struct {
		name  string
		epoch int64
		size  int
	}{name: "4 rows", epoch: 3}

	rows := []struct {
		predict string
		posY    int
		posX    int
	}{
		// rows: 0,5,9,10
		{"history\r\r\r\r\rchannel\r\r\r\rstarts\rworking", 0, 0},
	}

	pe := newPredictionEngine()
	emu := terminal.NewEmulator3(80, 40, 40)

	// printCursors(pe, "BEFORE newUserInput.")
	// fill the rows
	for _, v := range rows {
		emu.MoveCursor(v.posY, v.posX)
		pe.inputString(emu, v.predict)
		// printPredictionCell(emu, pe, v.posY, v.posX, v.predict, "INPUT ")
	}
	pe.cull(emu)

	// printCursors(pe, "AFTER newUserInput.")

	// posYs := []int{0, 5, 9, 10}
	// for _, posY := range posYs {
	// 	printPredictionCell(emu, pe, posY, 0, "channel", "PREDICT -")
	// }

	// it should be 11
	gotA := len(pe.cursors)
	// fmt.Println("killEpoch #testing called it explicitily.")
	pe.killEpoch(tc.epoch, emu)

	// it should be 2
	gotB := len(pe.cursors)

	// printCursors(pe, "AFTER killEpoch.")
	if gotB != 2 {
		t.Errorf("%q A=%d, B=%d\n", tc.name, gotA, gotB)
	}
}

func TestPredictionCull(t *testing.T) {
	tc := []struct {
		label               string
		base                string            // base content
		predict             string            // prediction
		frame               string            // the expect content
		row, col            int               // cursor start position
		displayPreference   DisplayPreference // display preference
		localFrameLateAcked uint64            // getValidity use localFrameLateAcked to validity cell or cursor prediction
		localFrameSent      uint64            // the cell prediction expirationFrame is set by localFrameSent+1
		sendInterval        uint
	}{
		/* 0*/ {
			"displayPreference is never", "", "", "",
			0, 0, Never, 0, 0, 0,
		},
		/* 1*/ {
			"IncorrectOrExpired >confirmedEpoch, killEpoch()", "", "right", "wrong",
			1, 70, Adaptive, 2, 1, 0,
		},
		/* 2*/ {
			"IncorrectOrExpired <confirmedEpoch, Experimental, reset2()", "", "rig", "won",
			2, 72, Experimental, 3, 2, 0,
		},
		/* 3*/ {
			"IncorrectOrExpired <confirmedEpoch, Reset()", "", "right", "wrong",
			3, 0, Adaptive, 4, 3, 0,
		},
		/* 4*/ {
			"Correct", "", "correct正确", "correct正确",
			4, 0, Adaptive, 5, 4, 0,
		},
		/* 5*/ {
			"Correct validity, delay >250", "", "正确delay>250", "正确delay>250",
			5, 0, Adaptive, 6, 5, 0,
		},
		/* 6*/ {
			"Correct validity, delay >5000", "", "delay>5000", "delay>5000",
			6, 0, Adaptive, 7, 6, 0,
		},
		/* 7*/ {
			"Correct validity, sendInterval=40", "", "sendInterval=40", "sendInterval=40",
			7, 0, Adaptive, 8, 7, 40,
		},
		/* 8*/ {
			"Correct validity, sendInterval=20", "", "sendInterval=20", "sendInterval=20",
			8, 0, Adaptive, 9, 8, 20,
		},
		/* 9*/ {
			"Correct validity + wrong cursor", "", "wrong cursor", "wrong cursor",
			9, 0, Adaptive, 10, 9, 0,
		},
		/*10*/ {
			"Correct validity + wrong cursor + Experimental", "", "wrong cursor + Experimental", "wrong cursor + Experimental",
			10, 0, Experimental, 11, 10, 0,
		},
		/*11*/ {
			"wrong row", "", "wrong row", "wrong row",
			40, 0, Adaptive, 12, 11, 0,
		},
		/*12*/ {
			"IncorrectOrExpired + >confirmedEpoch + Experimental", "", "Epoch", "confi",
			12, 0, Experimental, 13, 12, 0,
		},
	}
	emu := terminal.NewEmulator3(80, 40, 40)
	pe := newPredictionEngine()

	for k, v := range tc {
		t.Run(v.label, func(t *testing.T) {
			// fmt.Printf("\n%q #testing call cull A.\n", v.name)
			pe.SetDisplayPreference(v.displayPreference)

			// set the base content
			emu.MoveCursor(v.row, v.col)
			// fmt.Printf("#test cull %q HandleStream()\n", v.label)
			emu.HandleStream(v.base)

			// mimic user input for prediction engine
			emu.MoveCursor(v.row, v.col)
			pe.SetLocalFrameSent(v.localFrameSent)

			// fmt.Printf("#test %q cull B1. localFrameSend=%d, localFrameLateAcked=%d, predictionEpoch=%d, confirmedEpoch=%d\n",
			// 	v.label, pe.localFrameSent, pe.localFrameLateAcked, pe.predictionEpoch, pe.confirmedEpoch)

			// cull will be called for each rune, except last rune
			switch k {
			case 5:
				delay := []int{0, 0, 251, 0, 0, 0, 0, 0, 0}
				pe.inputString(emu, v.predict, delay...)
			case 6:
				delay := []int{0, 0, 5001, 0, 0, 0, 0, 0, 0}
				pe.inputString(emu, v.predict, delay...)
			case 7:
				pe.SetSendInterval(v.sendInterval)
				pe.inputString(emu, v.predict)
			case 8:
				pe.SetSendInterval(v.sendInterval)
				pe.inputString(emu, v.predict)
			case 11:
				pe.Reset()                             // clear the previous rows
				pe.getOrMakeRow(v.row, emu.GetWidth()) // add the illegal row
			case 12:
				// fmt.Printf("#test before inputString() %q confirmedEpoch=%d\n", v.label, pe.confirmedEpoch)
				now := time.Now().UnixMilli()
				for _, ch := range v.predict {
					pe.handleUserGrapheme(emu, now, ch)
				}
				// fmt.Printf("#test after inputString() %q confirmedEpoch=%d\n", v.label, pe.confirmedEpoch)
			default:
				pe.inputString(emu, v.predict)
			}
			// fmt.Printf("#test %q cull B2. localFrameSend=%d, localFrameLateAcked=%d, predictionEpoch=%d, confirmedEpoch=%d\n",
			// 	v.label, pe.localFrameSent, pe.localFrameLateAcked, pe.predictionEpoch, pe.confirmedEpoch)

			// mimic the result from server
			emu.MoveCursor(v.row, v.col)
			emu.HandleStream(v.frame)

			switch k {
			case 9, 10:
				emu.MoveCursor(v.row, v.col+1)
			}

			pe.SetLocalFrameLateAcked(v.localFrameLateAcked)
			pe.cull(emu)
			// fmt.Printf("#test %q cull B3. localFrameSend=%d, localFrameLateAcked=%d, predictionEpoch=%d, confirmedEpoch=%d\n",
			// 	v.label, pe.localFrameSent, pe.localFrameLateAcked, pe.predictionEpoch, pe.confirmedEpoch)

			switch k {
			case 1:
				// validate the result of killEpoch
				if len(pe.overlays) == 1 && len(pe.cursors) == 0 {
					// after killEpoch, cull() remove the last cursor because it's correct
					break
				} else {
					t.Errorf("%q should call killEpoch. got overlays=%d, cursors=%d\n", v.label, len(pe.overlays), len(pe.cursors))
				}
			case 6:
				if !pe.flagging {
					t.Errorf("%q expect true for flagging, got %t\n", v.label, pe.flagging)
				}
				fallthrough
			case 5:
				if pe.glitchTrigger == 0 {
					t.Errorf("%q glitchTrigger should >0, got %d\n", v.label, pe.glitchTrigger)
				}
				fallthrough
			case 2, 4, 12:
				// validate the result of cell reset2
				predictRow := pe.getOrMakeRow(v.row, emu.GetWidth())
				for i := range v.frame {
					predict := &(predictRow.overlayCells[v.col+i])
					if predict.active {
						t.Errorf("%q should not be active, got active=%t\n", v.label, predict.active)
					}
				}
				if k == 12 {
					if pe.confirmedEpoch != 2 {
						t.Errorf("%q expect confirmedEpoch < tentativeUntilEpoch. got %d\n", v.label, pe.confirmedEpoch)
					}
				}

			case 7:
				if !pe.flagging {
					t.Errorf("%q expect true for flagging, got %t\n", v.label, pe.flagging)
				}
			case 8:
				if pe.srttTrigger {
					t.Errorf("%q expect false for srttTrigger, got %t\n", v.label, pe.srttTrigger)
				}
			case 10:
				if len(pe.cursors) != 0 {
					t.Errorf("%q expect clean cursor prediction, got %d\n", v.label, len(pe.cursors))
				}
			case 11:
				if len(pe.overlays) != 0 {
					t.Errorf("%q expect zero rows, got %d\n", v.label, len(pe.overlays))
				}
			default:
				// validate pe.Reset()
				if len(pe.overlays) != 0 || len(pe.cursors) != 0 {
					t.Errorf("%s the engine should be reset. got overlays=%d, cursors=%d\n", v.label, len(pe.overlays), len(pe.cursors))
				}
			}
		})
	}
}

func TestPredictionNewInput(t *testing.T) {
	emu := terminal.NewEmulator3(80, 40, 40)
	pe := newPredictionEngine()

	pe.NewUserInput(emu, []rune{})
	// the pe and emu doesn't change so we don't validate the result.
}

func TestSetLocalFrameAcked(t *testing.T) {
	pe := newPredictionEngine()

	var expect uint64 = 7
	pe.SetLocalFrameAcked(expect)

	if pe.localFrameAcked != expect {
		t.Errorf("#test SetLocalFrameAcked expect %d, got %d\n", expect, pe.localFrameAcked)
	}
}

func TestTitleEngine(t *testing.T) {
	tc := []struct {
		name   string
		prefix string
		result string
	}{
		{"english title", " - aprish", " - aprish"},
		{"chinese title", "终端模拟器", "终端模拟器 - aprish"},
	}
	te := TitleEngine{}
	emu := terminal.NewEmulator3(80, 40, 40)
	for _, v := range tc {
		te.setPrefix(v.prefix)
		te.apply(emu)

		got := emu.GetWindowTitle()
		if v.result != got {
			t.Errorf("%q window title expect %q, got %q\n", v.name, v.result, got)
		}
		got = emu.GetIconLabel()
		if v.result != got {
			t.Errorf("%q icon name expect %q, got %q\n", v.name, v.result, got)
		}
	}

	omTitle := " [aprish]"
	om := NewOverlayManager()
	om.SetTitlePrefix(omTitle)

	if om.title.prefix != omTitle {
		t.Errorf("jump the queue, expect %q, got %q\n", omTitle, om.title.prefix)
	}
}

func TestNotificationEngine(t *testing.T) {
	tc := []struct {
		label                 string
		message               string
		escapeKeyString       string
		result                string
		lastWordFromServer    int64 // delta value based on now
		lastAckedState        int64 // delta value base on now
		permanent             bool
		messageIsNetworkError bool
		showQuitKeystroke     bool
	}{
		{
			"no message, no expire", "", "Ctrl-z",
			"",
			60, 80,
			false, false, true,
		},
		{
			"english message, no expire", "hello world",
			"Ctrl-z", "aprish: hello world [To quit: Ctrl-z .]",
			60, 80,
			false, false, true,
		},
		{
			"chinese message, no expire", "你好世界", "Ctrl-z",
			"aprish: 你好世界",
			60, 80,
			true, false, false,
		},
		{
			"server late", "你好世界", "Ctrl-z",
			"aprish: 你好世界 (1:05 without contact.)",
			65001, 80,
			true, false, false,
		},
		{
			"reply late", "aia group", "Ctrl-z",
			"aprish: aia group (10 s without reply.) [To quit: Ctrl-z .]",
			65, 10001,
			false, false, true,
		},
		{
			"no message, server late", "top gun 2", "Ctrl-z",
			"aprish: top gun 2 (1:05 without contact.) [To quit: Ctrl-z .]",
			65001, 10001,
			false, false, true,
		},
		{
			"no message, server too late", "top gun 2", "Ctrl-z",
			"aprish: top gun 2 (1:03:22 without contact.) [To quit: Ctrl-z .]",
			3802001, 100,
			false, false, true,
		},
		{
			"network error", "***", "Ctrl-z",
			"aprish: network error (10 s without reply.) [To quit: Ctrl-z .]",
			200, 10001,
			false, true, true,
		},
		{
			"restore from network failure", "restore from", "Ctrl-z",
			"aprish: restore from (20 s without reply.) [To quit: Ctrl-z .]",
			200, 20001,
			false, false, true,
		},
		{
			"no message, server late", "", "Ctrl-z",
			"aprish: Last contact 1:05 ago. [To quit: Ctrl-z .]",
			65001, 20001,
			false, false, true,
		},
	}

	for _, v := range tc {
		t.Run(v.label, func(t *testing.T) {
			ne := newNotificationEngien()
			emu := terminal.NewEmulator3(80, 40, 40)
			// fmt.Printf("%s start\n", v.name)
			if !ne.messageIsNetworkError {
				ne.SetNotificationString(v.message, v.permanent, v.showQuitKeystroke)
			}
			ne.SetEscapeKeyString(v.escapeKeyString)
			ne.ServerHeard(time.Now().UnixMilli() - v.lastWordFromServer)
			ne.ServerAcked(time.Now().UnixMilli() - v.lastAckedState)

			if v.messageIsNetworkError {
				ne.SetNetworkError(v.label)
			} else {
				ne.ClearNetworkError()
				ne.SetNotificationString(v.message, v.permanent, v.showQuitKeystroke)
			}

			ne.apply(emu)

			// build the string from emulator
			var got strings.Builder
			for i := 0; i < emu.GetWidth(); i++ {
				cell := emu.GetCell(0, i)
				if cell.IsDoubleWidthCont() {
					continue
				}

				got.WriteString(cell.GetContents())
			}

			// validate the result
			if len(v.result) != 0 {
				gotStr := strings.TrimSpace(got.String())
				if gotStr != v.result {
					t.Errorf("%q expect \n%q, got \n%q\n", v.label, v.result, gotStr)
				}
			}
			// fmt.Printf("%s end\n\n", v.name)
		})
	}
}

func TestNotificationEngine_adjustMessage(t *testing.T) {
	tc := []struct {
		name              string
		message           string
		expect            string
		messageExpiration int64
	}{
		{"message expire", "message expire", "", 0},
		{"message ready", "message 准备好了", "message 准备好了", 20},
	}

	ne := newNotificationEngien()
	for _, v := range tc {
		ne.SetNotificationString(v.message, false, false)

		// validate the message string
		if ne.GetNotificationString() != v.message {
			t.Errorf("%q expect %q, got %q\n", v.name, v.message, ne.GetNotificationString())
		}

		ne.messageExpiration = time.Now().UnixMilli() + v.messageExpiration
		ne.adjustMessage()

		// validate the empty string
		if ne.GetNotificationString() != v.expect {
			t.Errorf("%q expect %q, got %q\n", v.name, v.expect, ne.GetNotificationString())
		}
	}

	if min(7, 8) == 8 {
		t.Errorf("min should return %d, for min(7,8), got %d\n", 7, 8)
	}
}

func TestOverlayManager_waitTime(t *testing.T) {
	tc := []struct {
		name               string
		lastWordFromServer int64 // delta value based on now
		lastAckedState     int64 // delta value base on now
		messageExpiration  int64 // delta value base on now
		expect             int
	}{
		{"reply late", 600, 10001, 4000, 1000},
		{"server late", 65001, 100, 4000, 3000},
		{"no server late, no reply late", 65, 100, 400, 400},
	}

	om := NewOverlayManager()
	for _, v := range tc {
		ne := om.GetNotificationEngine()
		ne.ServerHeard(time.Now().UnixMilli() - v.lastWordFromServer)
		ne.ServerAcked(time.Now().UnixMilli() - v.lastAckedState)

		ne.messageExpiration = time.Now().UnixMilli() + v.messageExpiration

		got := om.WaitTime()
		if got != v.expect {
			t.Errorf("%q expect waitTime=%d, got %d\n", v.name, v.expect, got)
		}
	}
}

func TestOverlayManager_apply(t *testing.T) {
	om := NewOverlayManager()
	emu := terminal.NewEmulator3(80, 40, 40)
	om.GetPredictionEngine()

	// all the components of OverlayManager has been tested by previouse test case
	// add this for coverage 100%
	om.Apply(emu)
}

// add this method for test purpose
func (pe *PredictionEngine) inputString(emu *terminal.Emulator, str string, delay ...int) {
	var input []rune

	index := 0
	graphemes := uniseg.NewGraphemes(str)
	for graphemes.Next() {
		input = graphemes.Runes()
		if len(delay) > index { // delay parameters is provided to simulate network delay
			pause := time.Duration(delay[index])
			// fmt.Printf("#test inputString delay %dms.\n", pause)
			time.Sleep(time.Millisecond * pause)
			index++
		}
		// fmt.Printf("#test inputString() user input %s\n", string(input))
		pe.NewUserInput(emu, input)
	}
}

func TestOverlayCellResetWithOrig(t *testing.T) {
	emu := terminal.NewEmulator3(80, 40, 40)
	pe := newPredictionEngine()

	emu.MoveCursor(1, 0)
	pe.initCursor(emu)

	theRow := pe.getOrMakeRow(pe.cursor().row, emu.GetWidth())
	cell := &(theRow.overlayCells[0])

	/*
		here is the sample output:

		#test before resetWithOrig replacement=, active=false, originalContents=[], size=0, unknown=false
		#test before resetWithOrig replacement=, active=false, originalContents=[], size=0, unknown=false
		#test before resetWithOrig replacement=, active=false, originalContents=[], size=1, unknown=false
	*/
	got1 := fmt.Sprintf("#test before resetWithOrig replacement=%s, active=%t, originalContents=%s, size=%d, unknown=%t\n",
		cell.replacement, cell.active, cell.originalContents, len(cell.originalContents), cell.unknown)

	cell.active = false
	cell.unknown = false
	cell.resetWithOrig()
	got2 := fmt.Sprintf("#test before resetWithOrig replacement=%s, active=%t, originalContents=%s, size=%d, unknown=%t\n",
		cell.replacement, cell.active, cell.originalContents, len(cell.originalContents), cell.unknown)

	// validate the reset2 is called
	if got1 != got2 {
		t.Errorf("#test resetWithOrig() expect %s, got %s\n", got1, got2)
	}

	cell.active = true
	cell.unknown = false
	cell.resetWithOrig()
	got3 := fmt.Sprintf("#test before resetWithOrig replacement=%s, active=%t, originalContents=%s, size=%d, unknown=%t\n",
		cell.replacement, cell.active, cell.originalContents, len(cell.originalContents), cell.unknown)

	key := "size=1"
	if !strings.Contains(got3, key) {
		t.Errorf("#test resetWithOrig() expect %s, got %s\n", key, got3)
	}
}

func TestOverlayCellString(t *testing.T) {
	cell := newConditionalOverlayCell(12, 5, 1)

	got := cell.String()
	pieces := []string{"{repl:", "orig:", "unknown:", "active:", "}"}

	found := 0
	for i := range pieces {
		if strings.Contains(got, pieces[i]) {
			found++
		}
	}

	if found != len(pieces) {
		t.Errorf("#test conditionalOverlayCell String() method expect %s, got %s\n", pieces, &cell)
	}
}
