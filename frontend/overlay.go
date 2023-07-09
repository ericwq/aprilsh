// Copyright 2022 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package frontend

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/ericwq/aprilsh/terminal"
	"github.com/rivo/uniseg"
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
	SEND_INTERVAL_MIN    = 20    /* ms between frames */
	SEND_INTERVAL_MAX    = 250   /* ms between frames */
	ACK_INTERVAL         = 3000  /* ms between empty acks */
	ACK_DELAY            = 100   /* ms before delayed ack */
	SHUTDOWN_RETRIES     = 16    /* number of shutdown packets to send before giving up */
	ACTIVE_RETRY_TIMEOUT = 10000 /* attempt to resend at frame rate */
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

var strValidity = [...]string{
	"Unused",
	"Pending",
	"Correct",
	"CorrectNoCredit",
	"IncorrectOrExpired",
	"Inactive",
}

// base of predition cell or cursor position
type conditionalOverlay struct {
	expirationFrame     int64 // frame number, Emulator number.
	col                 int   // cursor column
	active              bool  // represents a prediction at all, default value false
	tentativeUntilEpoch int64 // when to show
	predictionTime      int64 // used to find long-pending predictions, default value -1
}

func newConditionalOverlay(expirationFrame int64, col int, tentativeUntilEpoch int64) conditionalOverlay {
	// default active is false, default predictionTime is -1
	co := conditionalOverlay{}
	co.expirationFrame = expirationFrame
	co.col = col
	co.active = false
	co.tentativeUntilEpoch = tentativeUntilEpoch
	co.predictionTime = -1

	return co
}

// if the overlay epoch is bigger than confirmedEpoch, return ture. otherwise false.
func (co *conditionalOverlay) tentative(confirmedEpoch int64) bool {
	return co.tentativeUntilEpoch > confirmedEpoch
}

// reset expirationFrame and tentativeUntilEpoch
func (co *conditionalOverlay) reset() {
	co.expirationFrame = -1
	co.tentativeUntilEpoch = -1
	co.active = false
}

// set expirationFrame and predictionTime
func (co *conditionalOverlay) expire(expirationFrame, now int64) {
	co.expirationFrame = expirationFrame
	co.predictionTime = now
}

// represent the prediction cursor move.
type conditionalCursorMove struct {
	conditionalOverlay
	row int // cursor row
}

func newConditionalCursorMove(expirationFrame int64, row int, col int, tentativeUntilEpoch int64) conditionalCursorMove {
	ccm := conditionalCursorMove{}
	ccm.conditionalOverlay = newConditionalOverlay(expirationFrame, col, tentativeUntilEpoch)
	ccm.row = row
	return ccm
}

// set prediction cursor position in emulator if the confirmedEpoch is less than tantative epoch.
func (ccm *conditionalCursorMove) apply(emu *terminal.Emulator, confirmedEpoch int64) {
	if !ccm.active { // only apply to active prediction
		return
	}

	if ccm.tentative(confirmedEpoch) { // check if it's the right time.
		return
	}

	// fmt.Printf("apply #cursorMove to (%d,%d)\n", ccm.row, ccm.col)
	emu.MoveCursor(ccm.row, ccm.col)
}

// check the validity of prediction cursor position. Validate the prediction cursor position:
// return Correct only when lateAck is greater than expirationFrame and cursor position is at
// the same position as the specified emulator. otherwise IncorrectOrExpired. if the prediction
// cursor position is not active, return Inactive.
func (ccm *conditionalCursorMove) getValidity(emu *terminal.Emulator, lateAck int64) Validity {
	if !ccm.active { // only validate active prediction
		return Inactive
	}

	// if cursor is out of active area, report IncorrectOrExpired
	if ccm.row >= emu.GetHeight() || ccm.col >= emu.GetWidth() {
		return IncorrectOrExpired
	}

	// lateAck is greater than expirationFrame
	if lateAck >= ccm.expirationFrame {
		// fmt.Printf("cursor getValidity() cell  (%d,%d)\n", ccm.row, ccm.col)
		// fmt.Printf("cursor getValidity() frame (%d,%d)\n", emu.GetCursorRow(), emu.GetCursorCol())
		if emu.GetCursorCol() == ccm.col && emu.GetCursorRow() == ccm.row {
			return Correct
		} else {
			return IncorrectOrExpired
		}
	}
	return Pending
}

// represent the prediction cell in the specified column. including the original cell contents and
// replacement contents.
type conditionalOverlayCell struct {
	conditionalOverlay
	replacement      terminal.Cell   // the prediction, replace the original content
	unknown          bool            // has replacement?
	originalContents []terminal.Cell // we don't give credit for correct predictions that match the original contents
}

func newConditionalOverlayCell(expirationFrame int64, col int, tentativeUntilEpoch int64) conditionalOverlayCell {
	coc := conditionalOverlayCell{}
	coc.conditionalOverlay = newConditionalOverlay(expirationFrame, col, tentativeUntilEpoch)
	coc.replacement = terminal.Cell{}
	coc.unknown = false
	coc.originalContents = make([]terminal.Cell, 0)
	return coc
}

// reset everything except replacement
func (coc *conditionalOverlayCell) reset2() {
	coc.unknown = false
	coc.originalContents = make([]terminal.Cell, 0)
	coc.reset()
}

// Reset everything if active is F or unknown is T. Otherwise append replacement to the originalContents.
func (coc *conditionalOverlayCell) resetWithOrig() {
	if !coc.active || coc.unknown {
		coc.reset2()
		return
	}

	coc.originalContents = append(coc.originalContents, coc.replacement)
	coc.reset()
}

// Apply cell prediction to the emulator, replace frame cell with prediction if they doesn't match.
//
// For unknown prediction just underline the cell.
// (row,col) specify the cell. confirmedEpoch specified the epoch. flag means underline the cell.
func (coc *conditionalOverlayCell) apply(emu *terminal.Emulator, confirmedEpoch int64, row int, flag bool) {
	// if coc.replacement.GetContents() != "" {
	// 	fmt.Printf("apply #cell (%d,%d) with prediction %q\n", row, coc.col, coc.replacement)
	// 	fmt.Printf("apply #cell coc.active=%t, confirmedEpoch=%d, coc.tentativeUntilEpoch=%d\n",
	// 		coc.active, confirmedEpoch, coc.tentativeUntilEpoch)
	// }

	// if specified position is out of active area or is not active.
	if !coc.active || row >= emu.GetHeight() || coc.col >= emu.GetWidth() {
		return
	}

	if coc.tentative(confirmedEpoch) { // check if it's the right time.
		return
	}

	// both prediction and emulator cell are blank
	if coc.replacement.IsBlank() && emu.GetCell(row, coc.col).IsBlank() {
		flag = false
	}

	// TODO the meaning of unknown?
	if coc.unknown {
		// fmt.Printf("apply #cell (%d,%d) is unknown %q\n", row, coc.col, coc.replacement)
		// underlining the cell except the last column.
		if flag && coc.col != emu.GetWidth()-1 {
			emu.GetCellPtr(row, coc.col).SetUnderline(true)
		}
		return
	}

	// if the cell is different from the prediction, replace it with the prediction.
	// update renditions if flag is true.
	if emu.GetCell(row, coc.col) != coc.replacement {
		// fmt.Printf("apply #cell (%d,%d) with %q\n", row, coc.col, coc.replacement)
		(*emu.GetCellPtr(row, coc.col)) = coc.replacement
		if flag {
			emu.GetCellPtr(row, coc.col).SetUnderline(true)
		}
	}
}

/*
Validate emulator cell against prediction cell:
if the cell is inactive, return Inactive.

if the prediction position is out of range return IncorrectOrExpired.

if the lateAck is smaller than the expiration frame, return Pending.

if the lateAck is greater than or equal to the expiration frame, then:

  - for unknown or blank prediction cell, return CorrectNoCredit.

  - if the frame cell matches the prediction cell and no history match prediction, retrun Correct.

  - if the frame cell matches the prediction cell and some istory match prediction, retrun CorrectNoCredit.

  - if the frame celll doesn't match the prediction cell, return IncorrectOrExpired.
*/
func (coc *conditionalOverlayCell) getValidity(emu *terminal.Emulator, row int, lateAck int64) Validity {
	if !coc.active {
		return Inactive
	}
	if row >= emu.GetHeight() || coc.col >= emu.GetWidth() {
		return IncorrectOrExpired
	}
	current := emu.GetCell(row, coc.col)

	// fmt.Printf("getValidity() (%d,%d) lateAck=%d, expirationFrame=%d unknow=%t\n",
	// 	row, coc.col, lateAck, coc.expirationFrame, coc.unknown)

	// see if it hasn't been updated yet
	if lateAck >= coc.expirationFrame {
		if coc.unknown {
			// fmt.Printf("getValidity() (%d,%d) return CorrectNoCredit\n", row, coc.col)
			return CorrectNoCredit
		}

		// too easy for this to trigger falsely
		if coc.replacement.IsBlank() {
			return CorrectNoCredit
		}

		// fmt.Printf("getValidity() current cell=%s, replacement=%s, result=%t\n",
		// 	current, coc.replacement, current.ContentsMatch(coc.replacement))
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

	// fmt.Printf("getValidity() (%d,%d) return Pending\n", row, coc.col)
	return Pending
}

// represents the prediction row, each row contains a group of prediction cells
// and a row number
type conditionalOverlayRow struct {
	rowNum       int
	overlayCells []conditionalOverlayCell
}

func newConditionalOverlayRow(rowNum int) *conditionalOverlayRow {
	row := conditionalOverlayRow{rowNum: rowNum}
	row.overlayCells = make([]conditionalOverlayCell, 0)
	return &row
}

// check the row number is the same as the specified rowNum
func (c *conditionalOverlayRow) rowNumEqual(ruwNum int) bool {
	return c.rowNum == ruwNum
}

// For each prediction cell in the prediction row applies the prediction to the emulator
//
// confirmedEpoch specified the epoch. flag means underline the cell.
func (c *conditionalOverlayRow) apply(emu *terminal.Emulator, confirmedEpoch int64, flag bool) {
	for i := range c.overlayCells {
		c.overlayCells[i].apply(emu, confirmedEpoch, c.rowNum, flag)
	}
}

// represent the prediction engine, which contains prediction cursor movement and
// prediction row.
type PredictionEngine struct {
	lastByte              rune
	parser                terminal.Parser
	overlays              []conditionalOverlayRow
	cursors               []conditionalCursorMove
	localFrameSent        int64
	localFrameAcked       int64
	localFrameLateAcked   int64
	predictionEpoch       int64 // becomeTentative update predictionEpoch
	confirmedEpoch        int64 // only Correct validity update confirmedEpoch
	flagging              bool  // whether we are underlining predictions
	srttTrigger           bool  // show predictions because of slow round trip time
	glitchTrigger         int   // show predictions temporarily because of long-pending prediction
	lastQuickConfirmation int64
	sendInterval          int
	lastWidth             int
	lastHeight            int
	displayPreference     DisplayPreference
}

func newPredictionEngine() *PredictionEngine {
	pe := PredictionEngine{}
	pe.parser = *terminal.NewParser()
	pe.cursors = make([]conditionalCursorMove, 0)
	pe.overlays = make([]conditionalOverlayRow, 0)
	pe.predictionEpoch = 1
	pe.sendInterval = 250
	pe.displayPreference = Adaptive

	return &pe
}

// get or make a row for the prediction engine.
func (pe *PredictionEngine) getOrMakeRow(rowNum int, nCols int) (it *conditionalOverlayRow) {
	for i := range pe.overlays {
		if pe.overlays[i].rowNumEqual(rowNum) {
			it = &(pe.overlays[i])
		}
	}
	if it == nil {
		it = newConditionalOverlayRow(rowNum)
		it.overlayCells = make([]conditionalOverlayCell, nCols)
		for i := 0; i < nCols; i++ {
			it.overlayCells[i] = newConditionalOverlayCell(0, i, pe.predictionEpoch)
		}
		pe.overlays = append(pe.overlays, *it)
	}
	return
}

// delay the prediction epoch to next time
func (pe *PredictionEngine) becomeTentative() {
	if pe.displayPreference != Experimental {
		pe.predictionEpoch++
		// fmt.Printf("becomeTentative #predictionEpoch=%d\n", pe.predictionEpoch)
	}
}

func (pe *PredictionEngine) newlineCarriageReturn(emu *terminal.Emulator) {
	now := time.Now().UnixMilli()
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

// return the last cursor move stored in the engine
func (pe *PredictionEngine) cursor() *conditionalCursorMove {
	if len(pe.cursors) == 0 {
		return nil
	}
	return &(pe.cursors[len(pe.cursors)-1])
}

// remove previous epoch cursor movement, append a new cursor movement,
// remove previous epoch cell prediction.
// delay the prediction to next epoch.
func (pe *PredictionEngine) killEpoch(epoch int64, emu *terminal.Emulator) {
	// remove cursor movement if epoch expire
	// fmt.Printf("killEpoch #1st cursors length=%d\n", len(pe.cursors))
	cursors := make([]conditionalCursorMove, 0)
	for i := range pe.cursors {
		if pe.cursors[i].tentative(epoch - 1) {
			// fmt.Printf("killEpoch #skip cursors (%2d,%2d), tentativeUntilEpoch=%d, epoch=%d\n",
			// pe.cursors[i].row, pe.cursors[i].col, pe.cursors[i].tentativeUntilEpoch, epoch-1)
			continue
		}
		// fmt.Printf("killEpoch #keep cursors (%2d,%2d)\n", pe.cursors[i].row, pe.cursors[i].col)
		cursors = append(cursors, pe.cursors[i])
	}

	// add current cursor position to prediction cursor
	cursors = append(cursors,
		newConditionalCursorMove(pe.localFrameSent+1, emu.GetCursorRow(), emu.GetCursorCol(), pe.predictionEpoch))
	pe.cursors = cursors
	pe.cursor().active = true

	// remove cell prediction if epoch expire
	for i := range pe.overlays {
		for j := range pe.overlays[i].overlayCells {
			cell := &(pe.overlays[i].overlayCells[j])
			if cell.tentative(epoch - 1) {
				cell.reset2()
				// fmt.Printf("killEpoch #cell (%2d,%2d) reset2\n", pe.overlays[i].rowNum, cell.col)
			}
		}
	}

	pe.becomeTentative()
	// fmt.Printf("killEpoch #last cursors=%d, overlays=%d\n", len(pe.cursors), len(pe.overlays))
}

// predict cursor move based on current cursor or initialize new cursor prediction with last
// cursor position. for the same epoch, just return the current prediction cursor position.
func (pe *PredictionEngine) initCursor(emu *terminal.Emulator) {
	if len(pe.cursors) == 0 {
		// initialize new cursor prediction with current cursor position
		cursor := newConditionalCursorMove(pe.localFrameSent+1, emu.GetCursorRow(), emu.GetCursorCol(), pe.predictionEpoch)
		pe.cursors = append(pe.cursors, cursor)
		pe.cursor().active = true
	} else if pe.cursor().tentativeUntilEpoch != pe.predictionEpoch {
		// initialize new cursor prediction with last cursor position
		cursor := newConditionalCursorMove(pe.localFrameSent+1, pe.cursor().row, pe.cursor().col, pe.predictionEpoch)
		pe.cursors = append(pe.cursors, cursor)
		pe.cursor().active = true
	}

	// fmt.Printf("initCursor #called len=%d, tentativeUntilEpoch=%d, predictionEpoch=%d, last %p\n",
	// 	len(pe.cursors), pe.cursor().tentativeUntilEpoch, pe.predictionEpoch, pe.cursor())
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

func (pe *PredictionEngine) timingTestsNecessary() bool {
	return !(pe.glitchTrigger > 0 && pe.flagging)
}

func (pe *PredictionEngine) SetDisplayPreference(v DisplayPreference) {
	pe.displayPreference = v
}

// apply overlay cells and cursors to Emulator.
func (pe *PredictionEngine) apply(emu *terminal.Emulator) {
	show := pe.displayPreference != Never && (pe.srttTrigger || pe.glitchTrigger > 0 ||
		pe.displayPreference == Always || pe.displayPreference == Experimental)

	// fmt.Printf("apply #engine show=%t\n", show)
	if show {
		for i := range pe.cursors {
			pe.cursors[i].apply(emu, pe.confirmedEpoch)
		}

		for i := range pe.overlays {
			pe.overlays[i].apply(emu, pe.confirmedEpoch, pe.flagging)
		}
	}
}

// process user input to prepare local prediction:cells and cursors.
// before process the input, PredictionEngine calls cull() method to check the prediction validity.
// a.k.a mosh new_user_byte() method
func (pe *PredictionEngine) NewUserInput(emu *terminal.Emulator, str string, delay ...int) {
	var input []rune
	var hd *terminal.Handler

	if len(str) == 0 {
		return
	}

	index := 0
	graphemes := uniseg.NewGraphemes(str)
	for graphemes.Next() {
		if pe.displayPreference == Never {
			continue // option Never means disable the prediction
		} else if pe.displayPreference == Experimental {
			pe.predictionEpoch = pe.confirmedEpoch
			// fmt.Printf("newUserInput #Experimental predictionEpoch = confirmedEpoch = %d\n", pe.confirmedEpoch)
		}

		input = graphemes.Runes()
		// fmt.Printf("newUserInput #epoch predictionEpoch=%d\n", pe.predictionEpoch)

		now := time.Now().UnixMilli()
		if len(delay) > index { // delay parameters is provided to simulate network delay
			pause := time.Duration(delay[index])
			// fmt.Printf("newUserInput #delay %dms.\n", pause)
			time.Sleep(time.Millisecond * pause)
			index++
		}
		pe.cull(emu)

		// translate application-mode cursor control function to ANSI cursor control sequence
		if pe.lastByte == '\x1b' && len(input) == 1 && input[0] == 'O' {
			input[0] = '['
		}
		if len(input) == 1 {
			pe.lastByte = input[0]
		}

		hd = pe.parser.ProcessInput(input...)
		if hd != nil {
			switch hd.GetId() {
			case terminal.Graphemes:
				// if pe.cursor() != nil {
				// 	fmt.Printf("newUserInput #cursors len=%d, tentativeUntilEpoch=%d, predictionEpoch=%d, last %p\n",
				// 		len(pe.cursors), pe.cursor().tentativeUntilEpoch, pe.predictionEpoch, pe.cursor())
				// }
				pe.handleUserGrapheme(emu, now, hd.GetCh())
			case terminal.C0_CR:
				pe.becomeTentative()
				pe.newlineCarriageReturn(emu)
			case terminal.CSI_CUF:
				pe.initCursor(emu)
				if pe.cursor().col < emu.GetWidth()-1 {
					// fmt.Printf("newUserInput #CUF before col=%d\n", pe.cursor().col)
					row := pe.getOrMakeRow(pe.cursor().row, emu.GetWidth())
					predict := row.overlayCells[pe.cursor().col+1].replacement
					cell := emu.GetCell(pe.cursor().row, pe.cursor().col+1)
					// check the next cell width, both predict and emulator need to be checked
					if cell.IsDoubleWidthCont() || predict.IsDoubleWidthCont() {
						if pe.cursor().col+2 >= emu.GetWidth() {
							// fmt.Printf("newUserInput #CUF abort col=%d\n", pe.cursor().col)
							break
						}
						pe.cursor().col += 2
					} else {
						pe.cursor().col++
					}
					pe.cursor().expire(pe.localFrameSent+1, now)
					// fmt.Printf("newUserInput #CUF after  col=%d\n", pe.cursor().col)
				}
			case terminal.CSI_CUB:
				pe.initCursor(emu)
				if pe.cursor().col > 0 { // TODO consider the left right margin.
					// fmt.Printf("newUserInput #CUB before col=%d\n", pe.cursor().col)
					row := pe.getOrMakeRow(pe.cursor().row, emu.GetWidth())
					predict := row.overlayCells[pe.cursor().col-1].replacement
					cell := emu.GetCell(pe.cursor().row, pe.cursor().col-1)
					// check the previous cell width, both predict and emulator need to be checked
					if cell.IsDoubleWidthCont() || predict.IsDoubleWidthCont() {
						if pe.cursor().col-2 <= 0 {
							pe.cursor().col = 0
							// fmt.Printf("newUserInput #CUB abort col=%d\n", pe.cursor().col)
							break
						}
						pe.cursor().col -= 2
					} else {
						pe.cursor().col--
					}
					pe.cursor().expire(pe.localFrameSent+1, now)
					// fmt.Printf("newUserInput #CUB after  col=%d\n", pe.cursor().col)
				}
			default:
				// TODO we can add support for more control sequences to improve the usability of prediction engine.
				pe.becomeTentative()
			}
			// if pe.cursor() != nil {
			// 	fmt.Printf("newUserInput # (%d,%d) input=%q\n", pe.cursor().row, pe.cursor().col, hd.GetSequence())
			// }
		}
	}
	// fmt.Printf("newUserInput #epoch predictionEpoch=%d\n\n", pe.predictionEpoch)
}

// check the validity of cell prediction and perform action based on the validity.
// for IncorrectOrExpired: remove the cell prediction or clear the whole prediction.
// for Correct: update glitch_trigger if possible, update remaining renditions, remove the cell prediction.
// for CorrectNoCredit: remove the cell prediction. update prediction renditions.
// for Pending: update glitch_trigger if possible, keep the prediction
// check the validity of cursor prediction and perform action based on the validity.
// for IncorrectOrExpired: clear the whole prediction.
func (pe *PredictionEngine) cull(emu *terminal.Emulator) {
	if pe.displayPreference == Never {
		return
	}

	// var x strings.Builder
	// fmt.Fprintf(&x, "cull # cell prediction overlays=%d ", len(pe.overlays))
	// for i := range pe.overlays {
	// 	fmt.Fprintf(&x, "row=%d ", pe.overlays[i].rowNum)
	// }
	// fmt.Println(x.String())

	if pe.lastHeight != emu.GetHeight() || pe.lastWidth != emu.GetWidth() {
		pe.lastHeight = emu.GetHeight()
		pe.lastWidth = emu.GetWidth()
		pe.Reset()
	}

	now := time.Now().UnixMilli()

	// fmt.Printf("cull() sendInterval=%d\n", pe.sendInterval)
	// control srtt_trigger with hysteresis
	if pe.sendInterval > SRTT_TRIGGER_HIGH {
		pe.srttTrigger = true
	} else if pe.srttTrigger && pe.sendInterval <= SRTT_TRIGGER_LOW && !pe.active() {
		// second condition: 20 ms is current minimum value
		// third condition: only turn off when no predictions being shown
		pe.srttTrigger = false
		// fmt.Printf("cull #srttTrigger=%t\n", pe.srttTrigger)
	}

	// control underlining with hysteresis
	if pe.sendInterval > FLAG_TRIGGER_HIGH {
		pe.flagging = true
	} else if pe.sendInterval <= FLAG_TRIGGER_LOW {
		pe.flagging = false
		// fmt.Printf("cull #flagging=%t FLAG_TRIGGER_LOW\n", pe.flagging)
	}

	// really big glitches also activate underlining
	if pe.glitchTrigger > GLITCH_REPAIR_COUNT {
		pe.flagging = true
		// fmt.Printf("cull #flagging=%t, glitchTrigger=%d GLITCH_REPAIR_COUNT\n", pe.flagging, pe.glitchTrigger)
	}

	// go through cell predictions
	overlays := make([]conditionalOverlayRow, 0, len(pe.overlays))
	for i := 0; i < len(pe.overlays); i++ {
		if pe.overlays[i].rowNum < 0 || pe.overlays[i].rowNum >= emu.GetHeight() {
			// skip/erase this row if it's out of scope.
			// fmt.Printf("cull #erase row=%d\n", pe.overlays[i].rowNum)
			continue
		} else {
			overlays = append(overlays, pe.overlays[i])
		}

		// fmt.Printf("cull # go through row %d\n", pe.overlays[i].rowNum)
		for j := range pe.overlays[i].overlayCells {
			cell := &(pe.overlays[i].overlayCells[j])
			v := cell.getValidity(emu, pe.overlays[i].rowNum, pe.localFrameLateAcked)
			// if v != Inactive {
			// 	fmt.Printf("cull #cell %p (%2d,%2d) active=%t,unknown=%t, %q, expirationFrame=%d, lateAck=%d, validity=%s\n",
			// 		cell, pe.overlays[i].rowNum, j, cell.active, cell.unknown, cell.replacement, cell.expirationFrame,
			// 		pe.localFrameLateAcked, strValidity[v])
			// }
			switch v {
			case IncorrectOrExpired:
				// fmt.Printf("cull #IncorrectOrExpired cell (%d,%d) tentativeUntilEpoch=%d, confirmedEpoch=%d\n",
				// 	pe.overlays[i].rowNum, j, cell.tentativeUntilEpoch, pe.confirmedEpoch)
				if cell.tentative(pe.confirmedEpoch) {
					// fmt.Printf("Bad tentative prediction in (%d,%d) (think %s, actually %s)\n",
					// 	pe.overlays[i].rowNum, cell.col, cell.replacement, emu.GetCell(pe.overlays[i].rowNum, cell.col))
					if pe.displayPreference == Experimental {
						cell.reset2()
					} else {
						// fmt.Printf("cull #cell killEpoch is called. tentativeUntilEpoch=%d, confirmedEpoch=%d\n",
						// 	cell.tentativeUntilEpoch, pe.confirmedEpoch)
						pe.killEpoch(cell.tentativeUntilEpoch, emu)
					}
				} else {
					// fmt.Printf("[%d=>%d] Killing prediction in row %d, col %d (think %s, actually %s)\n",
					// 	pe.localFrameLateAcked, cell.expirationFrame, pe.overlays[i].rowNum, cell.col,
					// 	cell.replacement, emu.GetCell(pe.overlays[i].rowNum, cell.col))
					if pe.displayPreference == Experimental {
						cell.reset2() // only clear the current cell
					} else {
						pe.Reset() // clear the whole prediction
						return
					}
				}
			case Correct:
				// fmt.Printf("cull #correct validate col=%d replacement=%s, original=%s, active=%t, ack=%d, expire=%d, Correct=%d\n",
				// 	cell.col, cell.replacement, cell.originalContents, cell.active, pe.localFrameLateAcked, cell.expirationFrame, Correct)
				// fmt.Printf("cull #Correct tentativeUntilEpoch=%d, confirmedEpoch=%d\n", cell.tentativeUntilEpoch, pe.confirmedEpoch)
				if cell.tentative(pe.confirmedEpoch) {
					// if cell.tentativeUntilEpoch > pe.confirmedEpoch {
					pe.confirmedEpoch = cell.tentativeUntilEpoch
				}

				// When predictions come in quickly, slowly take away the glitch trigger.
				// fmt.Printf("cull #Correct glitchTrigger=%d, now=%d, predictionTime=%d, now-cell.predictionTime=%d\n",
				// 	pe.glitchTrigger, now, cell.predictionTime, now-cell.predictionTime)
				if now-cell.predictionTime < GLITCH_THRESHOLD {
					if pe.glitchTrigger > 0 && now-GLITCH_REPAIR_MININTERVAL >= pe.lastQuickConfirmation {
						pe.glitchTrigger--
						pe.lastQuickConfirmation = now
					}
					// fmt.Printf("cull #Correct glitchTrigger=%d, now-GLITCH_REPAIR_MININTERVAL=%d, pe.lastQuickConfirmation=%d, cond=%t \n",
					// 	pe.glitchTrigger, now-GLITCH_REPAIR_MININTERVAL, pe.lastQuickConfirmation, now-GLITCH_REPAIR_MININTERVAL >= pe.lastQuickConfirmation)
				}

				// match rest of row to the actual renditions
				actualRenditions := emu.GetCell(pe.overlays[i].rowNum, cell.col).GetRenditions()
				for k := j; k < len(pe.overlays[i].overlayCells); k++ {
					pe.overlays[i].overlayCells[k].replacement.SetRenditions(actualRenditions)
				}

				cell.reset2()
			case CorrectNoCredit:
				// fmt.Printf("cull() (%d,%d) return CorrectNoCredit, replacement=%s, original=%s, active=%t, ack=%d, expire=%d\n",
				// fmt.Printf("cull #CorrectNoCredit tentativeUntilEpoch=%d, confirmedEpoch=%d\n", cell.tentativeUntilEpoch, pe.confirmedEpoch)
				// 	pe.overlays[i].rowNum, cell.col, cell.replacement, cell.originalContents, cell.active, pe.localFrameLateAcked, cell.expirationFrame)
				cell.reset2()
			case Pending:
				// When a prediction takes a long time to be confirmed, we
				// activate the predictions even if SRTT is low
				gap := (now - cell.predictionTime)
				if gap >= GLITCH_FLAG_THRESHOLD {
					// fmt.Printf("cull #Pending (%d,%d) gap=%d > 5000\n", pe.overlays[i].rowNum, cell.col, gap)
					pe.glitchTrigger = GLITCH_REPAIR_COUNT * 2 // display and underline
				} else if gap >= GLITCH_THRESHOLD && pe.glitchTrigger < GLITCH_REPAIR_COUNT {
					// fmt.Printf("cull #Pending (%d,%d) gap=%d > 250, glitchTrigger=%d, tentativeUntilEpoch=%d, confirmedEpoch=%d\n",
					// 	pe.overlays[i].rowNum, cell.col, gap, GLITCH_REPAIR_COUNT, cell.tentativeUntilEpoch, pe.confirmedEpoch)
					pe.glitchTrigger = GLITCH_REPAIR_COUNT // just display
				}
			default:
				// fmt.Printf("cell (%d,%d) return Inactive=%d\n", pe.overlays[i].rowNum, cell.col, Inactive)
				break
			}
		}
	}
	// restore overlay cells
	pe.overlays = overlays

	// go through cursor predictions
	if len(pe.cursors) > 0 {
		// fmt.Printf("cull #cursor (%d,%d) getValidity return %s: lateAck=%d, expirationFrame=%d\n", pe.cursor().row, pe.cursor().col,
		// strValidity[pe.cursor().getValidity(emu, pe.localFrameLateAcked)], pe.localFrameLateAcked, pe.cursor().expirationFrame)
		if pe.cursor().getValidity(emu, pe.localFrameLateAcked) == IncorrectOrExpired {
			// Sadly, we're predicting (%d,%d) vs. (%d,%d) [tau: %ld expiration_time=%ld, now=%ld]\n
			if pe.displayPreference == Experimental {
				pe.cursors = make([]conditionalCursorMove, 0) // only clear the cursor prediction
			} else {
				pe.Reset() // clear the whole prediction
				return
			}
		}
	}

	// fmt.Printf("cull # cursor prediction size=%d.\n", len(pe.cursors))
	cursors := make([]conditionalCursorMove, 0, len(pe.cursors))
	for i := range pe.cursors {
		// remove any cursor prediction except Pending validity.
		it := &(pe.cursors[i])
		if it.getValidity(emu, pe.localFrameLateAcked) != Pending {
			// fmt.Printf("cull #remove cursor at (%d,%d) for state %s\n",
			// 	pe.cursors[i].row, pe.cursors[i].col, strValidity[it.getValidity(emu, pe.localFrameLateAcked)])
			continue
		} else {
			cursors = append(cursors, *it)
		}
	}
	pe.cursors = cursors

	// fmt.Printf("cull # cursor prediction size=%d.\n", len(pe.cursors))
}

func (pe *PredictionEngine) Reset() {
	pe.cursors = make([]conditionalCursorMove, 0)
	pe.overlays = make([]conditionalOverlayRow, 0)
	pe.becomeTentative()
	// fmt.Println("reset #clear cursors and overlays")
}

func (pe *PredictionEngine) SetLocalFrameSent(v int64) {
	pe.localFrameSent = v
}

func (pe *PredictionEngine) SetLocalFrameAcked(v int64) {
	pe.localFrameAcked = v
}

func (pe *PredictionEngine) SetLocalFrameLateAcked(v int64) {
	pe.localFrameLateAcked = v
}

func (pe *PredictionEngine) SetSendInterval(value int) {
	pe.sendInterval = value
}

func (pe *PredictionEngine) waitTime() int {
	if pe.timingTestsNecessary() && pe.active() {
		return 50
	}
	return math.MaxInt
}

func (pe *PredictionEngine) handleUserGrapheme(emu *terminal.Emulator, now int64, chs ...rune) {
	w := uniseg.StringWidth(string(chs))
	pe.initCursor(emu)

	// fmt.Printf("handleUserGrapheme # got %q\n", chs)
	if len(chs) == 1 && chs[0] == '\x7f' {
		// backspace
		theRow := pe.getOrMakeRow(pe.cursor().row, emu.GetWidth())
		if pe.cursor().col > 0 {
			// fmt.Printf("handleUserGrapheme #backspace start at col=%d\n", pe.cursor().col)

			// move cursor to the previous graphemes
			predict := theRow.overlayCells[pe.cursor().col-1].replacement
			cell := emu.GetCell(pe.cursor().row, pe.cursor().col-1)
			// check the previous cell width, both predict and emulator need to check
			if cell.IsDoubleWidthCont() || predict.IsDoubleWidthCont() {
				if pe.cursor().col-2 <= 0 {
					pe.cursor().col = 0
					// fmt.Printf("handleUserGrapheme() backspace edge %d\n", pe.cursor().col)
				} else {
					pe.cursor().col -= 2
				}
			} else {
				pe.cursor().col--
			}
			pe.cursor().expire(pe.localFrameSent+1, now)
			// fmt.Printf("handleUserGrapheme #backspace col to %d\n", pe.cursor().col)

			// iterate to replace the current cell with next cell.
			for i := pe.cursor().col; i < emu.GetWidth(); i++ {
				cell := &(theRow.overlayCells[i])
				wideCell := false

				cell.resetWithOrig()
				cell.active = true
				cell.tentativeUntilEpoch = pe.predictionEpoch
				cell.expire(pe.localFrameSent+1, now)
				if len(cell.originalContents) == 0 {
					// avoid adding original cell content several times
					cell.originalContents = append(cell.originalContents, emu.GetCell(pe.cursor().row, i))
				}

				if i+2 < emu.GetWidth() {
					nextCell := &(theRow.overlayCells[i+1])
					if nextCell.replacement.IsDoubleWidthCont() {
						nextCell = &(theRow.overlayCells[i+2])
						wideCell = true
					}
					nextCellActual := emu.GetCell(pe.cursor().row, i+1)
					if nextCellActual.IsDoubleWidthCont() {
						nextCellActual = emu.GetCell(pe.cursor().row, i+2)
						wideCell = true
					}

					// fmt.Printf("handleUserGrapheme #backspace (%d,%d) iterate cell replacement. nextCell active=%t, unknown=%t\n",
					// 	pe.cursor().row, i, nextCell.active, nextCell.unknown)
					if nextCell.active {
						if nextCell.unknown {
							cell.unknown = true
						} else {
							cell.unknown = false
							cell.replacement = nextCell.replacement
						}
					} else {
						cell.unknown = false
						cell.replacement = nextCellActual
					}
				} else {
					cell.unknown = true
				}

				// fmt.Printf("handleUserGrapheme #backspace %p (%2d,%2d),active=%t,unknown=%t,dwidth=%t,%q,originalContents=%q\n",
				// 	cell, pe.cursor().row, i, cell.active, cell.unknown, cell.replacement.IsDoubleWidth(),
				// 	cell.replacement, cell.originalContents)

				if wideCell {
					i++
				}
			}

			// fmt.Printf("handleUserGrapheme #backspace row %d end.\n\n", pe.cursor().row)
		}
	} else if len(chs) == 1 && chs[0] < 0x20 {
		// unknown print
		pe.becomeTentative()
	} else {
		// normal rune, wide rune, combining grapheme

		// for wide rune, only one cell space is not enough, wrap to next row
		if w == 2 && pe.cursor().col == emu.GetWidth()-1 {
			pe.becomeTentative()
			pe.newlineCarriageReturn(emu)
			// fmt.Printf("handleUserGrapheme() wrap %q to (%d,%d)\n", string(chs), pe.cursor().row, pe.cursor().col)
		}

		theRow := pe.getOrMakeRow(pe.cursor().row, emu.GetWidth())
		if pe.cursor().col+1 >= emu.GetWidth() {
			// prediction in the last column is tricky
			// e.g., emacs will show wrap character, shell will just put the character there
			pe.becomeTentative()
		}

		// do the insert in reverse order
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

			if i-w < pe.cursor().col {
				break
			}

			// fmt.Printf("handleUserGrapheme() iterate col=%d, prev col=%d\n", i, i-w)
			prevCell := &(theRow.overlayCells[i-w])
			prevCellActual := emu.GetCell(pe.cursor().row, i-w)

			if i == emu.GetWidth()-1 {
				cell.unknown = true
				// cell.replacement = prevCellActual // TODO should we remove this?
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

			// fmt.Printf("handleUserGrapheme() cell (%2d,%2d) active=%t\tunknown=%t\treplacement=%q\tdwidth=%t\toriginalContents=%q\n",
			// 	pe.cursor().row, i, cell.active, cell.unknown, cell.replacement, cell.replacement.IsDoubleWidth(), cell.originalContents)
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

		// wide rune occupies 2 cells.
		if w == 2 {
			cell.replacement.SetDoubleWidth(true)
			nextCell := &(theRow.overlayCells[pe.cursor().col+1])
			nextCell.replacement.SetDoubleWidthCont(true)
		}

		cell.replacement.SetContents(chs)
		if len(cell.originalContents) == 0 {
			// avoid adding original cell content several times
			cell.originalContents = append(cell.originalContents, emu.GetCell(pe.cursor().row, pe.cursor().col))
		}

		// fmt.Printf("handleUserGrapheme #cell (%2d,%2d) active=%t, unknown=%t, replacement=%q, dwidth=%t, tentativeUntilEpoch=%d, originalContents=%q\n",
		// 	pe.cursor().row, pe.cursor().col, cell.active, cell.unknown, cell.replacement,
		// 	cell.replacement.IsDoubleWidth(), pe.predictionEpoch, cell.originalContents)

		pe.cursor().expire(pe.localFrameSent+1, now)

		// do we need to wrap?
		if pe.cursor().col < emu.GetWidth()-1 {
			pe.cursor().col += w
		} else {
			pe.becomeTentative()
			pe.newlineCarriageReturn(emu)
		}

		// fmt.Printf("handleUserGrapheme #cursor at (%d,%d) %p size=%d\n\n", pe.cursor().row, pe.cursor().col, pe.cursor(), len(pe.cursors))
	}
}

type TitleEngine struct {
	prefix string
}

func (te *TitleEngine) setPrefix(v string) {
	te.prefix = v
}

// apply the window title with the prefix
func (te *TitleEngine) apply(emu *terminal.Emulator) {
	emu.PrefixWindowTitle(te.prefix)
}

type NotificationEngine struct {
	lastWordFromServer    int64
	lastAckedState        int64
	escapeKeyString       string
	message               string
	messageIsNetworkError bool
	messageExpiration     int64
	showQuitKeystroke     bool
}

func newNotificationEngien() *NotificationEngine {
	ne := &(NotificationEngine{})
	ne.lastWordFromServer = time.Now().UnixMilli()
	ne.lastAckedState = time.Now().UnixMilli()
	ne.messageIsNetworkError = false
	ne.messageExpiration = -1
	ne.showQuitKeystroke = true
	return ne
}

func humanReadableDuration(numSeconds int, secondsAbbr string) string {
	var tmp strings.Builder
	if numSeconds < 60 {
		fmt.Fprintf(&tmp, "%d %s", numSeconds, secondsAbbr)
	} else if numSeconds < 3600 {
		fmt.Fprintf(&tmp, "%d:%02d", numSeconds/60, numSeconds%60)
	} else {
		fmt.Fprintf(&tmp, "%d:%02d:%02d", numSeconds/3600, (numSeconds/60)%60, numSeconds%60)
	}
	return tmp.String()
}

func (ne *NotificationEngine) serverLate(ts int64) bool {
	return ts-ne.lastWordFromServer > 65000
}

func (ne *NotificationEngine) replyLate(ts int64) bool {
	return ts-ne.lastAckedState > 10000
}

func (ne *NotificationEngine) needCountup(ts int64) bool {
	return ne.serverLate(ts) || ne.replyLate(ts)
}

func (ne *NotificationEngine) adjustMessage() {
	if time.Now().UnixMilli() >= ne.messageExpiration {
		ne.message = ""
	}
}

func (ne *NotificationEngine) apply(emu *terminal.Emulator) {
	now := time.Now().UnixMilli()
	timeExpired := ne.needCountup(now)
	// fmt.Printf("notifications\t  #apply timeExpired=%t, replyLate=%t, serverLate=%t, message=%d\n",
	// 	timeExpired, ne.replyLate(now), ne.serverLate(now), len(ne.message))

	if len(ne.message) == 0 && !timeExpired {
		return
	}

	// hide cursor if necessary
	if emu.GetCursorRow() == 0 {
		emu.SetCursorVisible(false)
	}

	// draw bar across top of screen
	notificationBar := &(terminal.Cell{})
	rend := &(terminal.Renditions{})
	rend.SetForegroundColor(7) // 37
	rend.SetBackgroundColor(4) // 44
	notificationBar.SetRenditions(emu.GetRenditions())
	notificationBar.SetContents([]rune{' '})

	for i := 0; i < emu.GetWidth(); i++ {
		emu.GetCellPtr(0, i).Reset2(*notificationBar)
	}

	/* We want to prefer the "last contact" message if we simply haven't
	   heard from the server in a while, but print the "last reply" message
	   if the problem is uplink-only. */

	sinceHeard := float64((now - ne.lastWordFromServer) / 1000.0) // convert millisecond to seconds
	sinceAck := float64((now - ne.lastAckedState) / 1000.0)       // convert millisecond to seconds
	serverMessage := "contact"
	replyMessage := "reply"

	timeElapsed := sinceHeard
	explanation := serverMessage

	if ne.replyLate(now) && !ne.serverLate(now) {
		timeElapsed = sinceAck
		explanation = replyMessage
	}

	keystrokeStr := ""
	if ne.showQuitKeystroke {
		keystrokeStr = ne.escapeKeyString
	}

	var stringToDraw strings.Builder
	// if len(ne.message) == 0 && !timeExpired {
	// 	return
	// } else
	if len(ne.message) == 0 && timeExpired {
		fmt.Fprintf(&stringToDraw, "aprish: Last %s %s ago.%s", explanation,
			humanReadableDuration(int(timeElapsed), "seconds"), keystrokeStr)
	} else if len(ne.message) != 0 && !timeExpired {
		fmt.Fprintf(&stringToDraw, "aprish: %s%s", ne.message, keystrokeStr)
	} else {
		fmt.Fprintf(&stringToDraw, "aprish: %s (%s without %s.)%s", ne.message,
			humanReadableDuration(int(timeElapsed), "s"), explanation, keystrokeStr)
	}

	// write message to screen buffer
	emu.MoveCursor(0, 0)
	emu.HandleStream(stringToDraw.String())
}

func (ne *NotificationEngine) GetNotificationString() string {
	return ne.message
}

func (ne *NotificationEngine) ServerHeard(ts int64) {
	ne.lastWordFromServer = ts
}

func (ne *NotificationEngine) ServerAcked(ts int64) {
	ne.lastAckedState = ts
}

func (ne *NotificationEngine) waitTime() int {
	nextExpiry := math.MaxInt
	now := time.Now().UnixMilli()
	nextExpiry = terminal.Min(nextExpiry, int(ne.messageExpiration-now))

	if ne.needCountup(now) {
		countupInterval := 1000
		if now-ne.lastWordFromServer > 60000 {
			// If we've been disconnected for 60 seconds, save power by updating the display less often.
			countupInterval = ACK_INTERVAL
		}
		nextExpiry = terminal.Min(nextExpiry, countupInterval)
	}

	return nextExpiry
}

// default parameters: permanent = false, showQuitKeystroke = true
func (ne *NotificationEngine) SetNotificationString(message string, permanent bool, showQuitKeystroke bool) {
	ne.message = message
	if permanent {
		ne.messageExpiration = -1
	} else {
		ne.messageExpiration = time.Now().UnixMilli() + 1000
	}

	ne.messageIsNetworkError = false
	ne.showQuitKeystroke = showQuitKeystroke
}

func (ne *NotificationEngine) SetEscapeKeyString(str string) {
	ne.escapeKeyString = fmt.Sprintf(" [To quit: %s .]", str)
}

func (ne *NotificationEngine) setNetworkError(str string) {
	ne.message = str
	ne.messageIsNetworkError = true
	ne.messageExpiration = time.Now().UnixMilli() + ACK_INTERVAL + 100
}

func (ne *NotificationEngine) clearNetworkError() {
	// fmt.Printf("clearNetworkError #debug messageIsNetworkError=%t\n", ne.messageIsNetworkError)
	if ne.messageIsNetworkError {
		ne.messageExpiration = terminal.Min(ne.messageExpiration, time.Now().UnixMilli()+1000)
	}
}

type OverlayManager struct {
	notifications *NotificationEngine
	predictions   *PredictionEngine
	title         *TitleEngine
}

func NewOverlayManager() *OverlayManager {
	om := &OverlayManager{}
	om.predictions = newPredictionEngine()
	om.notifications = newNotificationEngien()
	om.title = &TitleEngine{}
	return om
}

func (om *OverlayManager) GetNotificationEngine() *NotificationEngine {
	return om.notifications
}

func (om *OverlayManager) GetPredictionEngine() *PredictionEngine {
	return om.predictions
}

func (om *OverlayManager) SetTitlePrefix(v string) {
	om.title.setPrefix(v)
}

func (om *OverlayManager) waitTime() int {
	return terminal.Min(om.notifications.waitTime(), om.predictions.waitTime())
}

func (om *OverlayManager) Apply(emu *terminal.Emulator) {
	om.predictions.cull(emu)
	om.predictions.apply(emu)

	om.notifications.adjustMessage()
	om.notifications.apply(emu)

	om.title.apply(emu)
}
