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
	"github.com/ericwq/aprilsh/terminal"
)

type (
	Validity          uint
	DisplayPreference uint
)

const (
	ValidityUnused Validity = iota
	Pending
	Correct
	CorrectNoCredit
	IncorrectOrExpired
	Inactive
)

const (
	DisplayPreferenceUnused DisplayPreference = iota
	Always
	Never
	Adaptive
	Experimental
)

const (
	SRTT_TRIGGER_LOW          = 20   // <= ms cures SRTT trigger to show predictions
	SRTT_TRIGGER_HIGH         = 30   // > ms starts SRTT trigger
	FLAG_TRIGGER_LOW          = 50   // <= ms cures flagging
	FLAG_TRIGGER_HIGH         = 80   // > ms starts flagging
	GLITCH_THRESHOLD          = 250  // prediction outstanding this long is glitch
	GLITCH_REPAIR_COUNT       = 10   // non-glitches required to cure glitch trigger
	GLITCH_REPAIR_MININTERVAL = 150  // required time in between non-glitches
	GLITCH_FLAG_THRESHOLD     = 5000 // prediction outstanding this long => underline
)

type ConditionalOverlay struct {
	expirationFrame     int64
	col                 int
	active              bool  // represents a prediction at all, default value false
	tentativeUntilEpoch int64 // when to show
	predictionTime      int64 // used to find long-pending predictions, default value -1
}

func NewConditionalOverlay(expirationFrame int64, col int, tentativeUntilEpoch int64) ConditionalOverlay {
	// default active is false, default predictionTiem is -1
	co := ConditionalOverlay{}
	co.expirationFrame = expirationFrame
	co.col = col
	co.active = false
	co.tentativeUntilEpoch = tentativeUntilEpoch
	co.predictionTime = -1

	return co
}

// if the overlay is ready?
func (co *ConditionalOverlay) tentative(confirmedEpoch int64) bool {
	return co.tentativeUntilEpoch > confirmedEpoch
}

// reset expirationFrame and tentativeUntilEpoch
func (co *ConditionalOverlay) reset() {
	co.expirationFrame = -1
	co.tentativeUntilEpoch = -1
	co.active = false
}

// set expirationFrame and predictionTime
func (co *ConditionalOverlay) expire(expirationFrame, now int64) {
	co.expirationFrame = expirationFrame
	co.predictionTime = now
}

type ConditionalCursorMove struct {
	ConditionalOverlay
	row int
}

func NewConditionalCursorMove(expirationFrame int64, row int, col int, tentativeUntilEpoch int64) ConditionalCursorMove {
	ccm := ConditionalCursorMove{}
	ccm.ConditionalOverlay = NewConditionalOverlay(expirationFrame, col, tentativeUntilEpoch)
	ccm.row = row
	return ccm
}

// set cursor position in emulator if the confirmedEpoch is greater than tantative epoch.
func (ccm *ConditionalCursorMove) apply(emu *terminal.Emulator, confirmedEpoch int64) {
	if !ccm.active { // only apply to active prediction
		return
	}

	if ccm.tentative(confirmedEpoch) { // check if it's the right time.
		return
	}

	emu.MoveCursor(ccm.row, ccm.col)
}

// return Correct only when lateAck is greater than expirationFrame and cursor position is at the
// same position.
func (ccm *ConditionalCursorMove) getValidity(emu *terminal.Emulator, lateAck int64) Validity {
	if !ccm.active { // only validate active prediction
		return Inactive
	}

	// if cursor is out of active area, report IncorrectOrExpired
	if ccm.row >= emu.GetHeight() || ccm.col >= emu.GetWidth() {
		return IncorrectOrExpired
	}

	// lateAck is greater than expirationFrame
	if lateAck >= ccm.expirationFrame {
		if emu.GetCursorCol() == ccm.col && emu.GetCursorRow() == ccm.row {
			return Correct
		} else {
			return IncorrectOrExpired
		}
	}
	return Pending
}

type ConditionalOverlayCell struct {
	ConditionalOverlay
	replacement      terminal.Cell   // the prediction, replace the original content
	unknown          bool            // has replacement?
	originalContents []terminal.Cell // we don't give credit for correct predictions that match the original contents
}

func NewConditionalOverlayCell(expirationFrame int64, col int, tentativeUntilEpoch int64) ConditionalOverlayCell {
	coc := ConditionalOverlayCell{}
	coc.ConditionalOverlay = NewConditionalOverlay(expirationFrame, col, tentativeUntilEpoch)
	coc.replacement = terminal.Cell{}
	coc.unknown = false
	coc.originalContents = make([]terminal.Cell, 0)
	return coc
}

// reset everything except replacement
func (coc *ConditionalOverlayCell) reset2() {
	coc.unknown = false
	coc.originalContents = make([]terminal.Cell, 0)
	coc.reset()
}

// reset everything but fill the originalContents with replacement
func (coc *ConditionalOverlayCell) resetWithOrig() {
	if !coc.active || coc.unknown {
		coc.reset2()
		return
	}

	coc.originalContents = append(coc.originalContents, coc.replacement)
	coc.reset()
}

func (coc *ConditionalOverlayCell) apply(emu *terminal.Emulator, confirmedEpoch int64, row int, flag bool) {
	// if specified position is out of active area or is not active.
	if !coc.active || row >= emu.GetHeight() || coc.col >= emu.GetWidth() {
		return
	}

	if coc.tentative(confirmedEpoch) { // check if it's the right time.
		return
	}

	// both prediction and framebuffer cell are blank
	if coc.replacement.IsBlank() && emu.GetCell(row, coc.col).IsBlank() {
		flag = false
	}

	// TOODO the meaning of unknown?
	if coc.unknown {
		// except the last column add underline for the cell.
		if flag && coc.col != emu.GetWidth()-1 {
			emu.GetMutableCell(row, coc.col).SetUnderline(true)
		}
		return
	}

	// if the cell is not the same as the prediction, replace it with the prediction.
	if emu.GetCell(row, coc.col) != coc.replacement {
		(*emu.GetMutableCell(row, coc.col)) = coc.replacement
		if flag {
			emu.GetMutableCell(row, coc.col).SetUnderline(true)
		}
	}
}

func (coc *ConditionalOverlayCell) getValidity(emu *terminal.Emulator, row int, lateAck int64) Validity {
	if !coc.active {
		return Inactive
	}
	if row >= emu.GetHeight() || coc.col >= emu.GetWidth() {
		return IncorrectOrExpired
	}
	current := emu.GetCell(row, coc.col)

	// see if it hasn't been updated yet
	if lateAck >= coc.expirationFrame {
		if coc.unknown {
			return CorrectNoCredit
		}

		// too easy for this to trigger falsely
		if coc.replacement.IsBlank() {
			return CorrectNoCredit
		}

		if current.ContentsMatch(coc.replacement) {
			pos := 0
			for i := range coc.originalContents {
				if coc.originalContents[i].ContentsMatch(coc.replacement) {
					break
				}
				pos = i
			}
			if pos == len(coc.originalContents)-1 {
				return Correct
			} else {
				return CorrectNoCredit
			}
		} else {
			return IncorrectOrExpired
		}
	}
	return Pending
}

type ConditionalOverlayRow struct {
	rowNum       int
	overlayCells []ConditionalOverlayCell
}

func NewConditionalOverlayRow(rowNum int) *ConditionalOverlayRow {
	row := ConditionalOverlayRow{rowNum: rowNum}
	row.overlayCells = make([]ConditionalOverlayCell, 0)
	return &row
}

// TODO do we need this in golang?
func (cor *ConditionalOverlayRow) rowNumEqual(rowNum int) bool {
	return cor.rowNum == rowNum
}

func (cor *ConditionalOverlayRow) apply(emu *terminal.Emulator, confirmedEpoch int64, flag bool) {
	for i := range cor.overlayCells {
		cor.overlayCells[i].apply(emu, confirmedEpoch, cor.rowNum, flag)
	}
}

type PredictionEngine struct {
	lastByte              rune
	parser                terminal.Parser
	overlays              []ConditionalOverlayRow
	cursors               []ConditionalCursorMove
	localFrameSent        int64
	localFrameAcked       int64
	localFrameLateAcked   int64
	predictionEpoch       int64
	confirmedEpoch        int64
	flagging              bool // whether we are underlining predictions
	srttTrigger           bool // show predictions because of slow round trip time
	glitchTrigger         bool // show predictions temporarily because of long-pending prediction
	lastQuickConfirmation int64
	sendInterval          int
	lastWidth             int
	lastHeight            int
	displayPreference     DisplayPreference
}

func NewPredictionEngine() *PredictionEngine {
	pe := PredictionEngine{}
	pe.parser = *terminal.NewParser()
	pe.cursors = make([]ConditionalCursorMove, 0)
	pe.overlays = make([]ConditionalOverlayRow, 0)
	pe.predictionEpoch = 1
	pe.sendInterval = 250
	pe.displayPreference = Adaptive

	return &pe
}

// return the last cursor move stored in the engine
func (pe *PredictionEngine) cursor() *ConditionalCursorMove {
	if len(pe.cursors) == 0 {
		return nil
	}
	return &(pe.cursors[len(pe.cursors)-1])
}

// add cursor move to prediction engine.
func (pe *PredictionEngine) initCursor(emu *terminal.Emulator) {
	if len(pe.cursors) == 0 {
		// initialize new cursor prediction with current cursor position
		cursor := NewConditionalCursorMove(pe.localFrameSent+1, emu.GetCursorRow(), emu.GetCursorCol(), pe.predictionEpoch)
		pe.cursors = append(pe.cursors, cursor)
		pe.cursor().active = true
	} else if pe.cursor().tentativeUntilEpoch != pe.predictionEpoch {
		// initialize new cursor prediction with last cursor position
		cursor := NewConditionalCursorMove(pe.localFrameSent+1, pe.cursor().row, pe.cursor().col, pe.predictionEpoch)
		pe.cursors = append(pe.cursors, cursor)
		pe.cursor().active = true
	}
}

// get or make a row for the prediction engine.
func (pe *PredictionEngine) getOrMakeRow(rowNum int, nCols int) (it *ConditionalOverlayRow) {
	for i := range pe.overlays {
		if pe.overlays[i].rowNumEqual(rowNum) {
			it = &(pe.overlays[i])
		}
	}
	if it == nil {
		it = NewConditionalOverlayRow(rowNum)
		it.overlayCells = make([]ConditionalOverlayCell, nCols)
		for i := 0; i < nCols; i++ {
			it.overlayCells[i] = NewConditionalOverlayCell(0, i, pe.predictionEpoch)
		}
		pe.overlays = append(pe.overlays, *it)
	}
	return
}

func (pe *PredictionEngine) apply(emu *terminal.Emulator) {
	show := pe.displayPreference != Never && (pe.srttTrigger || pe.glitchTrigger ||
		pe.displayPreference == Always || pe.displayPreference == Experimental)

	if show {
		for i := range pe.cursors {
			pe.cursors[i].apply(emu, pe.confirmedEpoch)
		}

		for i := range pe.overlays {
			pe.overlays[i].apply(emu, pe.confirmedEpoch, pe.flagging)
		}
	}
}

func (pe *PredictionEngine) reset() {
	pe.cursors = make([]ConditionalCursorMove, 0)
	pe.overlays = make([]ConditionalOverlayRow, 0)
	pe.becomeTentative()
}

func (pe *PredictionEngine) becomeTentative() {
	if pe.displayPreference == Experimental {
		pe.predictionEpoch++
	}
}

// new_user_byte
func (pe *PredictionEngine) newUserInput(emu *terminal.Emulator, chs ...rune) {
	if pe.displayPreference == Never {
		return // Never disable the prediction
	} else if pe.displayPreference == Experimental {
		pe.predictionEpoch = pe.confirmedEpoch
	}

	pe.cull(emu)

	// now := time.Now().Unix()
	if len(chs) > 1 {
		// for multi runes, it should be grapheme.
		pe.handleGrapheme(emu, chs...)
		return
	}
	ch := chs[0]
	pe.lastByte = chs[0]

	hd := pe.parser.ProcessInput(ch)
	if hd != nil {
		switch hd.GetId() {
		case terminal.Graphemes:
		}
	}
}

func (pe *PredictionEngine) handleGrapheme(emu *terminal.Emulator, ch ...rune) {
}

func (pe *PredictionEngine) cull(emu *terminal.Emulator) {
	// TODO
}
