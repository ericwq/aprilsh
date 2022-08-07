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
	"time"

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

// set prediction cursor position in emulator if the confirmedEpoch is greater than tantative epoch.
func (ccm *ConditionalCursorMove) apply(emu *terminal.Emulator, confirmedEpoch int64) {
	if !ccm.active { // only apply to active prediction
		return
	}

	if ccm.tentative(confirmedEpoch) { // check if it's the right time.
		return
	}

	emu.MoveCursor(ccm.row, ccm.col)
}

// check the validity of prediction cursor move.
// return Correct only when lateAck is greater than expirationFrame and cursor position is at the same position.
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

// Reset everything if active is F or unknown is T. Otherwise append replacement to the originalContents.
func (coc *ConditionalOverlayCell) resetWithOrig() {
	if !coc.active || coc.unknown {
		coc.reset2()
		return
	}

	coc.originalContents = append(coc.originalContents, coc.replacement)
	coc.reset()
}

// apply cell prediction to the emulator, replace frame cell with prediction. (row,col) specify the cell.
// confirmedEpoch specified the epoch. flag means underlining the cell.
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
		// underlining the cell except the last column.
		if flag && coc.col != emu.GetWidth()-1 {
			emu.GetMutableCell(row, coc.col).SetUnderline(true)
		}
		return
	}

	// if the cell is different from the prediction, replace it with the prediction.
	// update renditions if flag is true.
	if emu.GetCell(row, coc.col) != coc.replacement {
		(*emu.GetMutableCell(row, coc.col)) = coc.replacement
		if flag {
			emu.GetMutableCell(row, coc.col).SetUnderline(true)
		}
	}
}

// for frame cell is the same as the prediction and any history content doesn't match prediction, return Correct .
// for unknown or blank cell, or history content match prediction, return CorrectNoCredit.
// for prediction cursor is out of range or prediction doesn't match frame cell, return IncorrectOrExpired.
// for the lasteAck is not greater than expirationFrame, return Pending.
// for inactive prediction, return Inactive.
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

		// fmt.Printf("getValidity() current cell=%s, replacement=%s, result=%t\n", current, coc.replacement, current.ContentsMatch(coc.replacement))
		// if the frame cell is the same as the prediction
		if current.ContentsMatch(coc.replacement) {
			// it's Correct if any history content doesn't match prediction
			found := false
			for i := range coc.originalContents {
				if coc.originalContents[i].ContentsMatch(coc.replacement) {
					found = true
					break
				}
			}
			if !found {
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

//
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
	glitchTrigger         int  // show predictions temporarily because of long-pending prediction
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

// apply overlay cells and cursors to Emulator.
func (pe *PredictionEngine) apply(emu *terminal.Emulator) {
	show := pe.displayPreference != Never && (pe.srttTrigger || pe.glitchTrigger > 0 ||
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

// delay the prediction epoch to next time
func (pe *PredictionEngine) becomeTentative() {
	if pe.displayPreference != Experimental {
		pe.predictionEpoch++
	}
}

func (pe *PredictionEngine) newlineCarriageReturn(emu *terminal.Emulator) {
	now := time.Now().Unix()
	pe.initCursor(emu)
	pe.cursor().col = 0
	if pe.cursor().row == emu.GetHeight()-1 {
		// Don't try to predict scroll until we have versioned cell predictions
		// TODO need to consider the scrolling part
		newRow := pe.getOrMakeRow(pe.cursor().row, emu.GetWidth())
		for i := range newRow.overlayCells {
			newRow.overlayCells[i].active = true
			newRow.overlayCells[i].tentativeUntilEpoch = pe.predictionEpoch
			newRow.overlayCells[i].expire(pe.localFrameSent+1, now)
			newRow.overlayCells[i].replacement.Clear()
		}
	} else {
		pe.cursor().row++
	}
}

// process user input to prepare local prediction:cells and cursors.
// before process the input, PredictionEngine calls cull() method to check the prediction validity.
// a.k.a mosh new_user_byte() method
func (pe *PredictionEngine) newUserInput(emu *terminal.Emulator, chs ...rune) {
	if pe.displayPreference == Never {
		return // option Never means disable the prediction
	} else if pe.displayPreference == Experimental {
		pe.predictionEpoch = pe.confirmedEpoch
	}

	pe.cull(emu)

	now := time.Now().Unix()
	ch := chs[0]
	pe.lastByte = chs[0] // lastByte seems useless.
	if len(chs) > 1 {
		// for multi runes, it should be grapheme.
		pe.handleUserGrapheme(emu, chs...)
		return
	}

	hd := pe.parser.ProcessInput(ch)
	if hd != nil {
		switch hd.GetId() {
		case terminal.Graphemes:
			pe.handleUserGrapheme(emu, ch)
		case terminal.C0_CR:
			pe.becomeTentative()
			pe.newlineCarriageReturn(emu)
		case terminal.CSI_CUF:
			pe.initCursor(emu)
			if pe.cursor().col < emu.GetWidth()-1 {
				pe.cursor().col++
				pe.cursor().expire(pe.localFrameSent+1, now)
			}
		case terminal.CSI_CUB:
			pe.initCursor(emu)
			if pe.cursor().col > 0 { // TODO consider the left right margin.
				pe.cursor().col--
				pe.cursor().expire(pe.localFrameSent+1, now)
			}
		default:
			pe.becomeTentative()
		}
	}
}

func (pe *PredictionEngine) handleUserGrapheme(emu *terminal.Emulator, chs ...rune) {
	w := terminal.RunesWidth(chs)
	pe.initCursor(emu)
	now := time.Now().Unix()

	if len(chs) == 1 && chs[0] == '\x7f' {
		// TODO handle backspace
	} else if chs[0] < 0x20 || w != 1 {
		// TODO handle wide rune, combining grapheme
	} else {
		// normal rune
		theRow := pe.getOrMakeRow(pe.cursor().row, emu.GetWidth())
		if pe.cursor().col+1 >= emu.GetWidth() {
			// prediction in the last column is tricky
			// e.g., emacs will show wrap character, shell will just put the character there
			pe.becomeTentative()
		}

		// do the insert
		for i := emu.GetWidth() - 1; i > pe.cursor().col; i-- {
			cell := &(theRow.overlayCells[i])
			cell.resetWithOrig()
			cell.active = true
			cell.tentativeUntilEpoch = pe.predictionEpoch
			cell.expire(pe.localFrameSent+1, now)
			if len(cell.originalContents) == 0 {
				// avoid adding original cell content several times
				cell.originalContents = append(cell.originalContents, emu.GetCell(pe.cursor().row, i))
			}

			prevCell := &(theRow.overlayCells[i-1])
			prevCellActual := emu.GetCell(pe.cursor().row, i-1)

			if i == emu.GetWidth()-1 {
				cell.unknown = true
			} else if prevCell.active {
				if prevCell.unknown {
					// prevCell active=T unknown=T
					cell.unknown = true
				} else {
					// prevCell active=T unknown=F
					cell.unknown = false
					cell.replacement = prevCell.replacement
				}
			} else {
				// prevCell active=F
				cell.unknown = false
				cell.replacement = prevCellActual
			}

			// fmt.Printf("handleGrapheme() cell (%d,%d) active=%t\tunknown=%t\treplacement=%s\toriginalContents=%s\n",
			// 	pe.cursor().row, i, cell.active, cell.unknown, cell.replacement, cell.originalContents)
		}

		cell := &(theRow.overlayCells[pe.cursor().col])
		cell.resetWithOrig()
		cell.active = true
		cell.tentativeUntilEpoch = pe.predictionEpoch
		cell.expire(pe.localFrameSent+1, now)
		cell.replacement.SetRenditions(emu.GetRenditions())

		// heuristic: match renditions of character to the left
		if pe.cursor().col > 0 {
			prevCell := &(theRow.overlayCells[pe.cursor().col-1])
			prevCellActual := emu.GetCell(pe.cursor().row, pe.cursor().col-1)

			if prevCell.active && !prevCell.unknown {
				cell.replacement.SetRenditions(prevCell.replacement.GetRenditions())
			} else {
				cell.replacement.SetRenditions(prevCellActual.GetRenditions())
			}
		}

		cell.replacement.Clear()
		cell.replacement.Append(chs[0])
		if len(cell.originalContents) == 0 {
			// avoid adding original cell content several times
			cell.originalContents = append(cell.originalContents, emu.GetCell(pe.cursor().row, pe.cursor().col))
		}
		// fmt.Printf("handleGrapheme() cell (%d,%d) active=%t\tunknown=%t\treplacement=%s\toriginalContents=%s\n",
		// 	pe.cursor().row, pe.cursor().col, cell.active, cell.unknown, cell.replacement, cell.originalContents)

		// move cursor
		pe.cursor().expire(pe.localFrameSent+1, now)

		// do we need to wrap?
		if pe.cursor().col < emu.GetWidth()-1 {
			pe.cursor().col++
		} else {
			pe.becomeTentative()
			pe.newlineCarriageReturn(emu)
		}
	}
}

// return true if there is any cursor move prediction or any input prediction, otherwise false.
func (pe *PredictionEngine) active() bool {
	if len(pe.cursors) != 0 {
		return true
	}

	for i := range pe.overlays {
		for j := range pe.overlays[i].overlayCells {
			if pe.overlays[i].overlayCells[j].active {
				return true
			}
		}
	}

	return false
}

// remove expire epoch cursor movement, append a new cursor movement,
// remove expire epoch cell prediction.
// delay the prediction to next time
func (pe *PredictionEngine) killEpoch(epoch int64, emu *terminal.Emulator) {
	// remove cursor movement if epoch expire
	// fmt.Printf("killEpoch() 1st cursors length=%d\n", len(pe.cursors))
	cursors := make([]ConditionalCursorMove, 0)
	for i := range pe.cursors {
		if pe.cursors[i].tentative(epoch - 1) {
			// fmt.Printf("killEpoch() skip cursors col=%d\n", pe.cursors[i].col)
			continue
		}
		cursors = append(cursors, pe.cursors[i])
	}
	cursors = append(cursors,
		NewConditionalCursorMove(pe.localFrameSent+1, emu.GetCursorRow(), emu.GetCursorCol(), pe.predictionEpoch))
	pe.cursors = cursors
	pe.cursor().active = true
	// fmt.Printf("killEpoch() 2nd cursors length=%d\n", len(pe.cursors))
	// remove cell prediction if epoch expire
	for i := range pe.overlays {
		for j := range pe.overlays[i].overlayCells {
			cell := &(pe.overlays[i].overlayCells[j])
			if cell.tentative(epoch - 1) {
				cell.reset2()
				// fmt.Printf("killEpoch() cell (%d,%d) reset2\n", pe.overlays[i].rowNum, cell.col)
			}
		}
	}

	pe.becomeTentative()
}

// check the validity of cell prediction and perform action based on the validity.
// for IncorrectOrExpired: remove the cell prediction or clear the whole prediction.
// for Correct: update glitch_trigger if possible, update remaining renditions, remove the cell prediction.
// for CorrectNoCredit: remove the cell prediction. keeps prediction.
// for Pending: update glitch_trigger if possible, the pre
// check the validity of cursor prediction and perform action based on the validity.
// for IncorrectOrExpired: clear the whole prediction.
func (pe *PredictionEngine) cull(emu *terminal.Emulator) {
	if pe.displayPreference == Never {
		return
	}

	if pe.lastHeight != emu.GetHeight() || pe.lastWidth != emu.GetWidth() {
		pe.lastHeight = emu.GetHeight()
		pe.lastWidth = emu.GetWidth()
		pe.reset()
	}

	now := time.Now().Unix()

	// control srtt_trigger with hysteresis
	if pe.sendInterval > SRTT_TRIGGER_HIGH {
		pe.srttTrigger = true
	} else if pe.srttTrigger && pe.sendInterval <= SRTT_TRIGGER_LOW && !pe.active() {
		// second condition: 20 ms is current minimum value
		// third condition: only turn off when no predictions being shown
		pe.srttTrigger = false
	}

	// control underlining with hysteresis
	if pe.sendInterval > FLAG_TRIGGER_HIGH {
		pe.flagging = true
	} else if pe.sendInterval <= FLAG_TRIGGER_LOW {
		pe.flagging = false
	}

	// really big glitches also activate underlining
	if pe.glitchTrigger > GLITCH_REPAIR_COUNT {
		pe.flagging = true
	}

	// go through cell predictions
	overlays := make([]ConditionalOverlayRow, 0, len(pe.overlays))
	for i := 0; i < len(pe.overlays); i++ {
		if pe.overlays[i].rowNum < 0 || pe.overlays[i].rowNum >= emu.GetHeight() {
			// skip/erase this row if it's out of scope.
			continue
		} else {
			overlays = append(overlays, pe.overlays[i])
		}

		for j := range pe.overlays[i].overlayCells {
			cell := &(pe.overlays[i].overlayCells[j])
			switch cell.getValidity(emu, pe.overlays[i].rowNum, pe.localFrameLateAcked) {
			case IncorrectOrExpired:
				if cell.tentative(pe.confirmedEpoch) {
					// fmt.Printf("Bad tentative prediction in row %d, col %d (think %s, actually %s)\n",
					// 	pe.overlays[i].rowNum, cell.col, cell.replacement, emu.GetCell(pe.overlays[i].rowNum, cell.col))
					if pe.displayPreference == Experimental {
						cell.reset2()
					} else {
						pe.killEpoch(cell.tentativeUntilEpoch, emu)
					}
				} else {
					// fmt.Printf("[%d=>%d] Killing prediction in row %d, col %d (think %s, actually %s)\n",
					// 	pe.localFrameLateAcked, cell.expirationFrame, pe.overlays[i].rowNum, cell.col, cell.replacement, emu.GetCell(pe.overlays[i].rowNum, cell.col))
					if pe.displayPreference == Experimental {
						cell.reset2() // only clear the current cell
					} else {
						pe.reset() // clear the whole prediction
						return
					}
				}
			case Correct:
				// fmt.Printf("cull() validate col=%d replacement=%s, original=%s, active=%t, ack=%d, expire=%d, Correct=%d\n",
				// 	cell.col, cell.replacement, cell.originalContents, cell.active, pe.localFrameLateAcked, cell.expirationFrame, Correct)
				if cell.tentative(pe.confirmedEpoch) {
					// if cell.tentativeUntilEpoch > pe.confirmedEpoch {
					pe.confirmedEpoch = cell.tentativeUntilEpoch
				}

				// When predictions come in quickly, slowly take away the glitch trigger.
				if now-cell.predictionTime < GLITCH_THRESHOLD {
					if pe.glitchTrigger > 0 && now-GLITCH_REPAIR_MININTERVAL >= pe.lastQuickConfirmation {
						pe.glitchTrigger--
						pe.lastQuickConfirmation = now
					}
				}

				// match rest of row to the actual renditions
				actualRenditions := emu.GetCell(pe.overlays[i].rowNum, cell.col).GetRenditions()
				for k := j; k < len(pe.overlays[i].overlayCells); k++ {
					pe.overlays[i].overlayCells[k].replacement.SetRenditions(actualRenditions)
				}

				cell.reset2()
			case CorrectNoCredit:
				// fmt.Printf("cull() validate col=%d replacement=%s, original=%s, active=%t, ack=%d, expire=%d, CorrectNoCredit=%d\n",
				// 	cell.col, cell.replacement, cell.originalContents, cell.active, pe.localFrameLateAcked, cell.expirationFrame, CorrectNoCredit)
				cell.reset2()
			case Pending:
				fmt.Printf("cull() return Pending=%d\n", Pending)
				// When a prediction takes a long time to be confirmed, we
				// activate the predictions even if SRTT is low
				if now-cell.predictionTime >= GLITCH_FLAG_THRESHOLD {
					pe.glitchTrigger = GLITCH_REPAIR_COUNT * 2 // display and underline
				} else if now-cell.predictionTime >= GLITCH_THRESHOLD && pe.glitchTrigger < GLITCH_REPAIR_COUNT {
					pe.glitchTrigger = GLITCH_REPAIR_COUNT // just display
				}
			default:
				// fmt.Printf("cull() return Inactive=%d\n", Inactive)
				break
			}
		}
	}
	// restore overlay cells
	pe.overlays = overlays

	// go through cursor predictions
	if len(pe.cursors) > 0 {
		if pe.cursor().getValidity(emu, pe.localFrameLateAcked) == IncorrectOrExpired {
			// Sadly, we're predicting (%d,%d) vs. (%d,%d) [tau: %ld expiration_time=%ld, now=%ld]\n
			if pe.displayPreference == Experimental {
				pe.cursors = make([]ConditionalCursorMove, 0) // only clear the cursor prediction
			} else {
				pe.reset() // clear the whole prediction
				return
			}
		}
	}

	cursors := make([]ConditionalCursorMove, 0, len(pe.cursors))
	for i := range pe.cursors {
		// remove any cursor prediction except Pending validity.
		it := &(pe.cursors[i])
		if it.getValidity(emu, pe.localFrameLateAcked) != Pending {
			continue
		} else {
			cursors = append(cursors, *it)
		}
	}
	pe.cursors = cursors
}
