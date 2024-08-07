// Copyright 2022~2023 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package frontend

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/ericwq/aprilsh/terminal"
	"github.com/ericwq/aprilsh/util"
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
	Never // disable the prediction
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

// base of prediction (can be cell or cursor)
type conditionalOverlay struct {
	expirationFrame     uint64 // sent frame number
	col                 int    // prediction column
	active              bool   // represents a prediction at all
	tentativeUntilEpoch int64  // prediction epoch, when to show
	predictionTime      int64  // create time, used to find long-pending predictions
}

func newConditionalOverlay(expirationFrame uint64, col int, tentativeUntilEpoch int64) conditionalOverlay {
	co := conditionalOverlay{}
	co.expirationFrame = expirationFrame
	co.col = col
	co.active = false
	co.tentativeUntilEpoch = tentativeUntilEpoch
	co.predictionTime = math.MaxInt64

	return co
}

// if prediction epoch is greater than confirmedEpoch, return true. otherwise false.
func (co *conditionalOverlay) tentative(confirmedEpoch int64) bool {
	return co.tentativeUntilEpoch > confirmedEpoch
}

// reset prediction:
//
// reset expirationFrame, epoch and active
func (co *conditionalOverlay) reset() {
	co.expirationFrame = math.MaxUint64
	co.tentativeUntilEpoch = math.MaxInt64
	co.active = false
}

// set expirationFrame and predictionTime
func (co *conditionalOverlay) expire(expirationFrame uint64, now int64) {
	co.expirationFrame = expirationFrame
	co.predictionTime = now
}

func (co conditionalOverlay) String() string {
	return fmt.Sprintf("{active:%t, frame:%d, epoch:%d, time:%d, col:%d}",
		co.active, co.expirationFrame, co.tentativeUntilEpoch, co.predictionTime, co.col)
}

// represent prediction cursor
type conditionalCursorMove struct {
	conditionalOverlay
	row int
}

func newConditionalCursorMove(expirationFrame uint64, row int, col int, tentativeUntilEpoch int64) conditionalCursorMove {
	ccm := conditionalCursorMove{}
	ccm.conditionalOverlay = newConditionalOverlay(expirationFrame, col, tentativeUntilEpoch)
	ccm.row = row
	return ccm
}

/*
apply prediction cursor to state:

if prediction cursor is active AND the confirmedEpoch is greater than or equal to cursor epoch,
move terminal cursor to prediction position, otherwise do nothing.
*/
func (ccm *conditionalCursorMove) apply(emu *terminal.Emulator, confirmedEpoch int64) {
	if !ccm.active { // only apply to active prediction
		return
	}

	if ccm.tentative(confirmedEpoch) {
		return
	}

	emu.MoveCursor(ccm.row, ccm.col)
	util.Logger.Trace("prediction message", "from", "conditionalCursorMove.apply",
		"row", ccm.row, "col", ccm.col, "cursor", "move")
}

/*
check validity of prediction cursor against state cursor:

if the prediction cursor is not active, return Inactive.

if prediction cursor is out of range, return IncorrectOrExpired

if lateAck is smaller than prediction expirationFrame, return Pending

if lateAck is greater than prediction expirationFrame and
the terminal cursor position is the same as prediction cursor,
return Correct. otherwise return IncorrectOrExpired.
*/
func (ccm *conditionalCursorMove) getValidity(emu *terminal.Emulator, lateAck uint64) Validity {
	if !ccm.active {
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
			util.Logger.Trace("prediction message", "from", "conditionalCursorMove.getValidity",
				"row", ccm.row, "col", ccm.col,
				"state.row", emu.GetCursorRow(), "state.col", emu.GetCursorCol())
			return IncorrectOrExpired
		}
	}
	return Pending
}

func (co conditionalCursorMove) String() string {
	return fmt.Sprintf("{active:%t; frame:%d, epoch:%d, time:%d, row:%d, col=%d}",
		co.active, co.expirationFrame, co.tentativeUntilEpoch, co.predictionTime, co.row, co.col)
}

// represent the prediction cell in some column,
// including the original cell contents and replacement contents.
type conditionalOverlayCell struct {
	// we don't give credit for correct predictions that match the original contents
	originalContents []terminal.Cell // history cell content including original cell
	replacement      terminal.Cell   // the prediction, replace the cell content
	conditionalOverlay
	unknown bool // we do predict, but the prediction is tricky.
}

func newConditionalOverlayCell(expirationFrame uint64, col int, tentativeUntilEpoch int64) conditionalOverlayCell {
	coc := conditionalOverlayCell{}
	coc.conditionalOverlay = newConditionalOverlay(expirationFrame, col, tentativeUntilEpoch)
	coc.replacement = terminal.Cell{}
	coc.unknown = false
	coc.originalContents = make([]terminal.Cell, 0)
	return coc
}

// clear prediction:
//
// reset everything except replacement
func (coc *conditionalOverlayCell) reset() {
	coc.unknown = false
	coc.originalContents = make([]terminal.Cell, 0)
	coc.conditionalOverlay.reset()
}

// reset prediction and remember previous replacement:
//
// For unactive or unknown prediction, clear the prediction.
//
// Otherwise append replacement to the originalContents, reset prediction.
func (coc *conditionalOverlayCell) resetWithOrig() {
	if !coc.active || coc.unknown {
		// fmt.Println("reset2")
		coc.reset()
		return
	}

	coc.originalContents = append(coc.originalContents, coc.replacement)
	coc.conditionalOverlay.reset()
}

func (coc *conditionalOverlayCell) String() string {
	return fmt.Sprintf("{repl:%s; orig:%s, unknown:%t, active:%t}",
		coc.replacement, coc.originalContents, coc.unknown, coc.active)
}

/*
apply prediction cell to state:

if prediction cell is inactive or out of range, do nothing.

if prediction epoch is greater than confirmedEpoch, do nothing.

if prediction cell and state cell are both blank, do nothing.

if prediction cell is unknown, add underline if flag is true AND prediction cell not the last column.

if state cell is different from prediction cell, replace it with the prediction.
add underline if flag is true.
*/
func (coc *conditionalOverlayCell) apply(emu *terminal.Emulator, confirmedEpoch int64, row int, flag bool, f diffFunc) {
	// if specified position is not active OR out of range, do nothing
	if !coc.active || row >= emu.GetHeight() || coc.col >= emu.GetWidth() {
		return
	}

	if coc.tentative(confirmedEpoch) { // need to wait for next epoch
		return
	}

	// both prediction cell and terminal cell are blank
	if coc.replacement.IsBlank() && emu.GetCell(row, coc.col).IsBlank() {
		flag = false
	}

	if coc.unknown {
		// if flag is true and the cell is not the last column, add underline.
		if flag && coc.col != emu.GetWidth()-1 {
			emu.GetCellPtr(row, coc.col).SetUnderline(true)
		}
		return
	}

	// if the state cell is different from the prediction cell, replace it with the prediction.
	// add underline if flag is true.
	if emu.GetCell(row, coc.col) != coc.replacement {

		// util.Logger.Trace("prediction message", "from", "conditionalOverlayCell.apply",
		// 	"row", row, "col", coc.col, "cell", fmt.Sprintf("%#v", emu.GetCell(row, coc.col)))
		//
		// util.Logger.Trace("prediction message", "from", "conditionalOverlayCell.apply",
		// 	"row", row, "col", coc.col, "repl", fmt.Sprintf("%#v", coc.replacement))
		//
		// util.Logger.Trace("prediction message", "from", "conditionalOverlayCell.apply",
		// 	"row", row, "col", coc.col, "emu", fmt.Sprintf("%p", emu),
		// 	"cell", emu.GetCell(row, coc.col), "repl", coc.replacement)

		(*emu.GetCellPtr(row, coc.col)) = coc.replacement

		util.Logger.Trace("prediction message", "from", "conditionalOverlayCell.apply",
			"row", row, "col", coc.col, "emu", fmt.Sprintf("%p", emu),
			"cell", emu.GetCell(row, coc.col), "repl", coc.replacement)

		if flag {
			emu.GetCellPtr(row, coc.col).SetUnderline(true)
		}
		f(row, coc.col, emu.GetCell(row, coc.col))
	}
}

/*
check validity of prediction cell against state cell:

if prediction cell is inactive, return Inactive.

if prediction cell position is out of range return IncorrectOrExpired.

if lateAck is smaller than prediction expirationFrame, return Pending.

if prediction cell is unknown, return CorrectNoCredit.

if prediction cell is blank, return CorrectNoCredit.

if state cell matches prediction cell: if no history match prediction,
return Correct, otherwise return CorrectNoCredit.

if state cell doesn't match prediction cell, return IncorrectOrExpired.
*/
func (coc *conditionalOverlayCell) getValidity(emu *terminal.Emulator, row int, lateAck uint64) Validity {
	if !coc.active {
		return Inactive
	}

	if row >= emu.GetHeight() || coc.col >= emu.GetWidth() {
		return IncorrectOrExpired
	}

	current := emu.GetCell(row, coc.col)

	// see if it hasn't been updated yet
	if lateAck < coc.expirationFrame {
		return Pending
	}

	if coc.unknown {
		return CorrectNoCredit
	}

	// too easy for this to trigger falsely
	if coc.replacement.IsBlank() {
		return CorrectNoCredit
	}

	if current.ContentsMatch(coc.replacement) {
		// it's Correct if no history match prediction
		found := false
		for i := range coc.originalContents {
			if coc.originalContents[i].ContentsMatch(coc.replacement) {
				found = true
				break
			}
		}
		if !found {
			return Correct
		}
		return CorrectNoCredit
	} else {
		util.Logger.Trace("conditionalOverlayCell", "found", false, "current", current, "replacement", coc.replacement)
	}

	return IncorrectOrExpired
}

type conditionalOverlayRow struct {
	overlayCells []conditionalOverlayCell
	rowNum       int
}

func newConditionalOverlayRow(rowNum int) *conditionalOverlayRow {
	row := conditionalOverlayRow{rowNum: rowNum}
	row.overlayCells = make([]conditionalOverlayCell, 0)
	return &row
}

// apply prediction row to state:
//
// For each prediction cell in the row applies prediction cell to state
func (c *conditionalOverlayRow) apply(emu *terminal.Emulator, confirmedEpoch int64, flag bool, f diffFunc) {
	for i := range c.overlayCells {
		c.overlayCells[i].apply(emu, confirmedEpoch, c.rowNum, flag, f)
	}
}

type NotificationEngine struct {
	escapeKeyString       string
	message               string
	lastWordFromServer    int64 // latest received state timestamp
	lastAckedState        int64 // first sent state (acked) timestamp
	messageExpiration     int64
	messageIsNetworkError bool
	showQuitKeystroke     bool
}

func newNotificationEngien() *NotificationEngine {
	ne := &(NotificationEngine{})
	ne.lastWordFromServer = time.Now().UnixMilli()
	ne.lastAckedState = time.Now().UnixMilli()
	ne.messageIsNetworkError = false
	ne.messageExpiration = math.MaxInt64
	ne.showQuitKeystroke = true
	return ne
}

// convert seconds into readable string
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

// not receive over 6.5 seconds.
func (ne *NotificationEngine) serverLate(ts int64) bool {
	return ts-ne.lastWordFromServer > 6500
}

// not send (successfully acked) over 10 seconds.
func (ne *NotificationEngine) replyLate(ts int64) bool {
	return ts-ne.lastAckedState > 10000
}

// return true, if send OR receive over predefined time.
func (ne *NotificationEngine) needCountup(ts int64) bool {
	return ne.serverLate(ts) || ne.replyLate(ts)
}

// if message expired, set empty message.
func (ne *NotificationEngine) adjustMessage() {
	if time.Now().UnixMilli() >= ne.messageExpiration {
		ne.message = ""
	}
}

/*
if there's no message and no expiration, just return.

if there's no message and expiration, print contact/reply time on top line.

if there's message and no expiration, print message on top line.

if there's message and expiration, print message and contact/reply time on top line
*/
func (ne *NotificationEngine) apply(emu *terminal.Emulator) {
	now := time.Now().UnixMilli()
	timeExpired := ne.needCountup(now)

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
	notificationBar.SetRenditions(*rend)
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

	// duplicate code, check the front of method
	// if len(ne.message) == 0 && !timeExpired {
	// 	return
	// }
	if len(ne.message) == 0 && timeExpired {
		fmt.Fprintf(&stringToDraw, "aprish: Last %s %s ago.%s", explanation,
			humanReadableDuration(int(timeElapsed), "seconds"), keystrokeStr)
	} else if len(ne.message) != 0 && !timeExpired {
		fmt.Fprintf(&stringToDraw, "aprish: %s%s", ne.message, keystrokeStr)
	} else {
		fmt.Fprintf(&stringToDraw, "aprish: %s (%s without %s.)%s", ne.message,
			humanReadableDuration(int(timeElapsed), "s"), explanation, keystrokeStr)
	}

	// write message to terminal
	emu.MoveCursor(0, 0)
	util.Logger.Trace("NotificationEngine", "apply", "start")
	emu.HandleStream(stringToDraw.String())
	util.Logger.Trace("NotificationEngine", "arise", "end", "data", stringToDraw.String())
}

func (ne *NotificationEngine) GetNotificationString() string {
	return ne.message
}

// set latest received state timestamp, when network input happens or shutdown
func (ne *NotificationEngine) ServerHeard(ts int64) {
	ne.lastWordFromServer = ts
}

// set first sent state timestamp, when network input happens
func (ne *NotificationEngine) ServerAcked(ts int64) {
	ne.lastAckedState = ts
}

// if send OR receive over predefined time, return 1 second.
// if we've been disconnected for 60 seconds, return 3 seconds.
// otherwise, return message experiation duration.
func (ne *NotificationEngine) waitTime() int {
	var nextExpiry int64 = math.MaxInt64
	now := time.Now().UnixMilli()
	nextExpiry = min(nextExpiry, ne.messageExpiration-now)

	if ne.needCountup(now) {
		var countupInterval int64 = 1000
		if now-ne.lastWordFromServer > 60000 {
			// If we've been disconnected for 60 seconds, save power by updating
			// the display less often.
			countupInterval = ACK_INTERVAL
		}
		nextExpiry = min(nextExpiry, countupInterval)
	}

	return int(nextExpiry)
}

// set message and message expire time, if permanent is true, message expire time is forever.
// if permanent is false, message expires 1 second later. also set showQuitKeystroke.
func (ne *NotificationEngine) SetNotificationString(message string, permanent bool, showQuitKeystroke bool) {
	ne.message = message
	if permanent {
		ne.messageExpiration = math.MaxInt64
	} else {
		ne.messageExpiration = time.Now().UnixMilli() + 1000
	}

	ne.messageIsNetworkError = false
	ne.showQuitKeystroke = showQuitKeystroke
}

func (ne *NotificationEngine) SetEscapeKeyString(str string) {
	ne.escapeKeyString = fmt.Sprintf(" [To quit: %s .]", str)
}

// set (network error) message and message expire time, message expires 3.1 seconds later.
func (ne *NotificationEngine) SetNetworkError(str string) {
	ne.message = str
	ne.messageIsNetworkError = true
	ne.messageExpiration = time.Now().UnixMilli() + ACK_INTERVAL + 100
}

// extend message expire time 1 second later, if it's network message error.
func (ne *NotificationEngine) ClearNetworkError() {
	if ne.messageIsNetworkError {
		ne.messageExpiration = min(ne.messageExpiration, time.Now().UnixMilli()+1000)
	}
}

// predict cursor movement and user input
type PredictionEngine struct {
	lastByte              []rune
	diff                  []appliedDiff
	overlays              []conditionalOverlayRow
	cursors               []conditionalCursorMove
	parser                terminal.Parser
	confirmedEpoch        int64             // only update when prediction cell is correct, in cull()
	glitchTrigger         int               // show predictions temporarily because of long-pending prediction
	localFrameLateAcked   uint64            // when network input happens, set the last received remote state ack
	predictionEpoch       int64             // only in becomeTentative(), update predictionEpoch
	localFrameSent        uint64            // when user input happens, the last sent state num
	displayPreference     DisplayPreference // prediction display mode
	lastHeight            int               // remember terminal last height
	localFrameAcked       uint64            // when network input happens, set the first sent state num
	lastQuickConfirmation int64             // last quick response time
	sendInterval          uint              // when network input happens, set send interval
	lastWidth             int               // remember last terminal width
	srttTrigger           bool              // show predictions because of slow round trip time
	flagging              bool              // whether we are underlining predictions
	predictOverwrite      bool              // if true, overwrite terminal cell
	applied               bool              // if the prediction has been applied
}

/*

The following message is from https://mosh.org/mosh-paper.pdf

Our general strategy is for the Mosh client to make an echo prediction each time
the user hits a key, but not necessarily to display this prediction immediately.

The predictions are made in groups known as “epochs,” with the intention that
either all of the predictions in an epoch will be correct, or none will. An epoch
begins tentatively, making predictions only in the background. If any prediction
from a certain epoch is confirmed by the server, the rest of the predictions in
that epoch are immediately displayed to the use

*/

func newPredictionEngine() *PredictionEngine {
	pe := PredictionEngine{}
	pe.lastByte = make([]rune, 1)
	pe.overlays = make([]conditionalOverlayRow, 0)
	pe.cursors = make([]conditionalCursorMove, 0)
	pe.parser = *terminal.NewParser()
	pe.predictionEpoch = 1
	pe.confirmedEpoch = 0
	pe.sendInterval = 250
	pe.displayPreference = Adaptive

	return &pe
}

// get or make a prediction row
func (pe *PredictionEngine) getOrMakeRow(rowNum int, nCols int) (it *conditionalOverlayRow) {
	// try to find the existing prediction row
	for i := range pe.overlays {
		if pe.overlays[i].rowNum == rowNum {
			it = &(pe.overlays[i])
			break
		}
	}
	if it == nil {
		// make a new prediction row for the rowNum
		it = newConditionalOverlayRow(rowNum)
		it.overlayCells = make([]conditionalOverlayCell, nCols)
		for i := 0; i < nCols; i++ {
			it.overlayCells[i] = newConditionalOverlayCell(0, i, pe.predictionEpoch)
		}
		pe.overlays = append(pe.overlays, *it)
	}
	return
}

// increase prediction epoch by one, except Experimental mode
func (pe *PredictionEngine) becomeTentative() {
	if pe.displayPreference != Experimental {
		pe.predictionEpoch++
	}
}

// prediction action for new line CR:
//
// set prediction cursor to first col.
//
// if prediction cursor is not the last row, increase prediction cursor's row.
//
// if prediction cursor is the last row, add new prediction row.
func (pe *PredictionEngine) newlineCarriageReturn(emu *terminal.Emulator) {
	now := time.Now().UnixMilli()
	pe.initCursor(emu)
	pe.cursor().col = 0
	if pe.cursor().row == emu.GetHeight()-1 {
		// Don't try to predict scroll until we have versioned cell predictions

		// make blank prediction for last row
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

// get the latest prediction cursor
func (pe *PredictionEngine) cursor() *conditionalCursorMove {
	if len(pe.cursors) == 0 {
		return nil
	}
	return &(pe.cursors[len(pe.cursors)-1])
}

/*
remove prediction cursors and cells belong to previous epoch; start new epoch:

remove prediction cursors belong to previous epoch, append current cursor to prediction cursors.

disable prediction cells belong to previous epoch. increase prediction epoch by one,
except Experimental mode.
*/
func (pe *PredictionEngine) killEpoch(epoch int64, emu *terminal.Emulator) {
	// remove prediction cursors belong to previous epoch
	cursors := make([]conditionalCursorMove, 0)
	for i := range pe.cursors {
		if pe.cursors[i].tentative(epoch - 1) {
			continue
		}
		cursors = append(cursors, pe.cursors[i])
	}

	// append current cursor to prediction cursors
	cursors = append(cursors,
		newConditionalCursorMove(pe.localFrameSent+1,
			emu.GetCursorRow(), emu.GetCursorCol(), pe.predictionEpoch))
	pe.cursors = cursors
	pe.cursor().active = true

	// disable prediction cells belong to previous epoch
	for i := range pe.overlays {
		for j := range pe.overlays[i].overlayCells {
			cell := &(pe.overlays[i].overlayCells[j])
			if cell.tentative(epoch - 1) {
				cell.reset()
			}
		} // pe.reset will clean the predictions.
	}

	pe.becomeTentative()
}

/*
if prediction cursor doesn't exist, add new prediction based on terminal's current cursor position.

if prediction cursor exist, if cursor's epoch is different from engine's epoch, add new prediction
based on engine's cursor position.

otherwise don't change the cursor prediction
*/
func (pe *PredictionEngine) initCursor(emu *terminal.Emulator) {
	if len(pe.cursors) == 0 {
		// add new prediction based on terminal's current cursor position
		cursor := newConditionalCursorMove(pe.localFrameSent+1,
			emu.GetCursorRow(), emu.GetCursorCol(), pe.predictionEpoch)
		pe.cursors = append(pe.cursors, cursor)
		pe.cursor().active = true
	} else if pe.cursor().tentativeUntilEpoch != pe.predictionEpoch {
		// initialize new cursor prediction with last cursor position
		cursor := newConditionalCursorMove(pe.localFrameSent+1,
			pe.cursor().row, pe.cursor().col, pe.predictionEpoch)
		pe.cursors = append(pe.cursors, cursor)
		pe.cursor().active = true
	}
}

// return true if there is any prediction cursor or any active prediction cell, otherwise false.
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

// no glitch, no flagging
//
// Are there any timing-based triggers that haven't fired yet?
func (pe *PredictionEngine) timingTestsNecessary() bool {
	return pe.glitchTrigger <= 0 || !pe.flagging
}

// set displayPreference when init
func (pe *PredictionEngine) SetDisplayPreference(v DisplayPreference) {
	pe.displayPreference = v
}

// set predictOverwrite when init.
func (pe *PredictionEngine) SetPredictOverwrite(overwrite bool) {
	pe.predictOverwrite = overwrite
}

// if there is no slow RTT and no glitch, predictions run in the background,
// displayPreference mode also determine whether predictions run in the background.
// if it doesn't run in the background, move the cursor, show the cell prediction in terminal.
func (pe *PredictionEngine) apply(emu *terminal.Emulator) {
	util.Logger.Trace("prediction message", "from", "apply",
		"srttTrigger", pe.srttTrigger, "glitchTrigger", pe.glitchTrigger,
		"displayPreference", pe.displayPreference)

	if pe.applied || pe.displayPreference == Never || (!pe.srttTrigger && pe.glitchTrigger == 0 &&
		pe.displayPreference != Always && pe.displayPreference != Experimental) {
		util.Logger.Trace("prediction message", "from", "apply", "action", "skip")
		return
	}

	util.Logger.Trace("prediction message", "from", "apply", "action", "apply")
	for i := range pe.overlays {
		pe.overlays[i].apply(emu, pe.confirmedEpoch, pe.flagging,
			func(row int, col int, cell terminal.Cell) {
				pe.diff = append(pe.diff, appliedDiff{cell: cell, row: row, col: col})
			})
	}

	for i := range pe.cursors {
		pe.cursors[i].apply(emu, pe.confirmedEpoch)
	}
	util.Logger.Trace("prediction message", "from", "apply", "cursors", pe.cursors)
}

/*
when user input happens, set last sent state number before callling this method,
this method validate previous predictions (cull), then use new input to prepare for new prediction.
prepare for new prediction means updating prediction overlays, which involve cells and cursors.

a.k.a mosh new_user_byte() method
*/
func (pe *PredictionEngine) NewUserInput(emu *terminal.Emulator, input []rune, ptime ...int64) {
	if len(input) == 0 {
		return
	}

	switch pe.displayPreference {
	case Never:
		return
	case Experimental:
		pe.predictionEpoch = pe.confirmedEpoch
	}

	util.Logger.Trace("prediction message", "from", "NewUserInput.start",
		"state", fmt.Sprintf("%p", emu),
		"predictionEpoch", pe.predictionEpoch)
	pe.cull(emu)

	// add ptime for test
	var now int64
	if len(ptime) > 0 {
		now = ptime[0]
	} else {
		now = time.Now().UnixMilli()
	}

	// translate application-mode cursor control function to ANSI cursor control sequence
	// cursorKeyMode, DECCKM; fix bug #25.
	// if len(pe.lastByte) == 1 && pe.lastByte[0] == '\x1b' && len(input) == 1 && input[0] == 'O' {
	// 	input[0] = '['
	// }

	pe.lastByte = make([]rune, len(input))
	copy(pe.lastByte, input)
	// util.Logger.Trace("NewUserInput", "lastByte", pe.lastByte, "input", input)

	// TODO: validate we can handle flag grapheme
	// Fun 🌈with Flag🇧🇷.s
	hd := pe.parser.ProcessInput(input...)
	if hd != nil {
		switch hd.GetId() {
		case terminal.Graphemes:
			// util.Logger.Trace("NewUserInput", "predictionEpoch", pe.predictionEpoch, "Graphemes", hd.GetCh())
			pe.handleUserGrapheme(emu, now, hd.GetCh())
		case terminal.C0_CR:
			pe.becomeTentative()
			pe.newlineCarriageReturn(emu)
		case terminal.CSI_CUF: // right arrow
			pe.initCursor(emu)
			if pe.cursor().col < emu.GetWidth()-1 {
				util.Logger.Trace("NewUserInput", "CSI_CUF", "before", "column", pe.cursor().col)

				row := pe.getOrMakeRow(pe.cursor().row, emu.GetWidth())
				predict := row.overlayCells[pe.cursor().col+1].replacement
				cell := emu.GetCell(pe.cursor().row, pe.cursor().col+1)
				// check the next cell width, both predict and emulator need to be checked
				if cell.IsDoubleWidthCont() || predict.IsDoubleWidthCont() {
					if pe.cursor().col+2 >= emu.GetWidth() {
						util.Logger.Trace("NewUserInput", "CSI_CUF", "right margin", "column", pe.cursor().col)
						break
					}
					pe.cursor().col += 2
				} else {
					pe.cursor().col++
				}
				pe.cursor().expire(pe.localFrameSent+1, now)

				util.Logger.Trace("NewUserInput", "CSI_CUF", "after", "column", pe.cursor().col)
			}
		case terminal.CSI_CUB: // left arrow
			pe.initCursor(emu)
			if pe.cursor().col > 0 {
				util.Logger.Trace("NewUserInput", "CSI_CUB", "before", "column", pe.cursor().col)

				row := pe.getOrMakeRow(pe.cursor().row, emu.GetWidth())
				predict := row.overlayCells[pe.cursor().col-1].replacement
				cell := emu.GetCell(pe.cursor().row, pe.cursor().col-1)
				// check the previous cell width, both predict and emulator need to be checked
				if cell.IsDoubleWidthCont() || predict.IsDoubleWidthCont() {
					if pe.cursor().col-2 <= 0 {
						pe.cursor().col = 0
						util.Logger.Trace("NewUserInput", "CSI_CUB", "left margin", "column", pe.cursor().col)
						break
					}
					pe.cursor().col -= 2
				} else {
					pe.cursor().col--
				}
				pe.cursor().expire(pe.localFrameSent+1, now)

				util.Logger.Trace("NewUserInput", "CSI_CUB", "after", "column", pe.cursor().col)
			}
		default:
			// we can add support for more control sequences to improve the usability of prediction engine.
			pe.becomeTentative()
		}
		pe.applied = false
	}

	util.Logger.Trace("prediction message", "from", "NewUserInput.end",
		"predictionEpoch", pe.predictionEpoch, "overlays", len(pe.overlays))
}

/*
check validity of prediction cell and perform action accordingly:

- for IncorrectOrExpired: remove previous epoch or clear the whole prediction.

- for Correct: update glitch_trigger (decrease), update remaining renditions, reset prediction cell .

- for CorrectNoCredit: reset prediction cell.

- for Pending: update glitch_trigger (increate), keep prediction cell.

check validity of prediction cursor and perform action accordingly:

- if the last prediction cursor is IncorrectOrExpired, clear the whole prediction.

- remove any prediction cursor, except for Pending validity.
*/
func (pe *PredictionEngine) cull(emu *terminal.Emulator) {
	if pe.displayPreference == Never {
		return
	}

	// if the engine's size is different from terminal's, reset the engine.
	if pe.lastHeight != emu.GetHeight() || pe.lastWidth != emu.GetWidth() {
		pe.lastHeight = emu.GetHeight()
		pe.lastWidth = emu.GetWidth()
		pe.Reset()
	}

	now := time.Now().UnixMilli()

	// control srtt_trigger with hysteresis
	if pe.sendInterval > SRTT_TRIGGER_HIGH {
		pe.srttTrigger = true
	} else if pe.srttTrigger && pe.sendInterval <= SRTT_TRIGGER_LOW && !pe.active() {
		// second condition: 20 ms is the minimum value
		// third condition: there is no active predictions
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

	util.Logger.Trace("prediction message", "from", "cull",
		"predictionEpoch", pe.predictionEpoch, "confirmedEpoch", pe.confirmedEpoch, "overlays", len(pe.overlays),
		"sendInterval", pe.sendInterval, "localFrameLateAcked", pe.localFrameLateAcked)

	// go through prediction cells
	overlays := pe.overlays[:0]
	for i := 0; i < len(pe.overlays); i++ {
		if pe.overlays[i].rowNum < 0 || pe.overlays[i].rowNum >= emu.GetHeight() {
			// skip/erase this row if it's out of scope.
			continue
		}

		for j := range pe.overlays[i].overlayCells {
			cell := &(pe.overlays[i].overlayCells[j])
			v := cell.getValidity(emu, pe.overlays[i].rowNum, pe.localFrameLateAcked)

			actualCell := emu.GetCell(pe.overlays[i].rowNum, j)

			switch v {
			case IncorrectOrExpired:
				util.Logger.Trace("prediction message", "from", "cull", "row", pe.overlays[i].rowNum, "col", j,
					"validity", "IncorrectOrExpired",
					"tentativeUntilEpoch", cell.tentativeUntilEpoch, "confirmedEpoch", pe.confirmedEpoch,
					"actualCell", actualCell)

				if cell.tentative(pe.confirmedEpoch) {
					/*
					   fprintf( stderr, "Bad tentative prediction in row %d, col %d (think %lc, actually %lc)\n",
					            i->row_num, j->col,
					            j->replacement.debug_contents(),
					            fb.get_cell( i->row_num, j->col )->debug_contents()
					            );
					*/

					if pe.displayPreference == Experimental {
						cell.reset()
					} else {
						pe.killEpoch(cell.tentativeUntilEpoch, emu)
					}
					/*
					   if ( j->display_time != uint64_t(-1) ) {
					     fprintf( stderr, "TIMING %ld - %ld (TENT)\n", time(NULL), now - j->display_time );
					   }
					*/
				} else {
					/*
					   fprintf( stderr, "[%d=>%d] Killing prediction in row %d, col %d (think %lc, actually %lc)\n",
					            (int)local_frame_acked, (int)j->expiration_frame,
					            i->row_num, j->col,
					            j->replacement.debug_contents(),
					            fb.get_cell( i->row_num, j->col )->debug_contents() );
					*/
					/*
					   if ( j->display_time != uint64_t(-1) ) {
					     fprintf( stderr, "TIMING %ld - %ld\n", time(NULL), now - j->display_time );
					   }
					*/
					if pe.displayPreference == Experimental {
						cell.reset()
					} else {
						pe.Reset()
						return
					}
				}
			case Correct:
				/*
				   if ( j->display_time != uint64_t(-1) ) {
				     fprintf( stderr, "TIMING %ld + %ld\n", now, now - j->display_time );
				   }
				*/

				util.Logger.Trace("prediction message", "from", "cull", "row", pe.overlays[i].rowNum, "col", j,
					"validity", "Correct", "tentativeUntilEpoch", cell.tentativeUntilEpoch, "confirmedEpoch", pe.confirmedEpoch)

				if cell.tentative(pe.confirmedEpoch) {
					pe.confirmedEpoch = cell.tentativeUntilEpoch

					/*
					   fprintf( stderr, "%lc in (%d,%d) confirms epoch %lu (predicting in epoch %lu)\n",
					            j->replacement.debug_contents(), i->row_num, j->col,
					            confirmed_epoch, prediction_epoch );
					*/
				}

				// When predictions come in quickly, slowly take away the glitch trigger.
				if (now - cell.predictionTime) < GLITCH_THRESHOLD {
					if pe.glitchTrigger > 0 && (now-GLITCH_REPAIR_MININTERVAL) >= pe.lastQuickConfirmation {
						pe.glitchTrigger--
						pe.lastQuickConfirmation = now
					}
				}

				// match rest of row to the actual renditions
				actualRenditions := emu.GetCell(pe.overlays[i].rowNum, cell.col).GetRenditions()
				for k := j; k < len(pe.overlays[i].overlayCells); k++ {
					pe.overlays[i].overlayCells[k].replacement.SetRenditions(actualRenditions)
				}

				cell.reset()
			case CorrectNoCredit:

				// util.Logger.Trace("prediction message", "from", "cull", "row", pe.overlays[i].rowNum, "col", j,
				// 	"validity", "CorrectNoCredit",
				//       "tentativeUntilEpoch", cell.tentativeUntilEpoch, "confirmedEpoch", pe.confirmedEpoch)

				cell.reset()
			case Pending:
				// When a prediction takes a long time to be confirmed, we
				// activate the predictions even if SRTT is low
				gap := (now - cell.predictionTime)
				if gap >= GLITCH_FLAG_THRESHOLD {
					pe.glitchTrigger = GLITCH_REPAIR_COUNT * 2 // display and underline
				} else if gap >= GLITCH_THRESHOLD && pe.glitchTrigger < GLITCH_REPAIR_COUNT {
					pe.glitchTrigger = GLITCH_REPAIR_COUNT // just display
				}

				// util.Logger.Trace("prediction message", "from", "cull", "row", pe.overlays[i].rowNum, "col", j,
				// 	"validity", "Pending", "gap", gap,
				// 	"tentativeUntilEpoch", cell.tentativeUntilEpoch, "confirmedEpoch", pe.confirmedEpoch,
				// 	"glitchTrigger", pe.glitchTrigger)

			default:

				// util.Logger.Trace("prediction message", "from", "cull", "row", pe.overlays[i].rowNum, "col", j,
				// 	"validity", "Inactive",
				// 	"tentativeUntilEpoch", cell.tentativeUntilEpoch, "confirmedEpoch", pe.confirmedEpoch)

			}
		}

		overlays = append(overlays, pe.overlays[i])
	}
	pe.overlays = overlays

	// go through cursor predictions
	if len(pe.cursors) > 0 && pe.cursor().getValidity(emu, pe.localFrameLateAcked) == IncorrectOrExpired {
		/*
		   fprintf( stderr, "Sadly, we're predicting (%d,%d) vs. (%d,%d) [tau: %ld, expiration_time=%ld, now=%ld]\n",
		   cursor().row, cursor().col,
		   fb.ds.get_cursor_row(),
		   fb.ds.get_cursor_col(),
		   cursor().tentative_until_epoch,
		   cursor().expiration_time,
		   now );
		*/

		if pe.displayPreference == Experimental {
			pe.cursors = make([]conditionalCursorMove, 0)
		} else {
			pe.Reset()
			util.Logger.Trace("prediction message", "from", "cull", "cursor", "Reset",
				"validity", "IncorrectOrExpired", "predictionEpoch", pe.predictionEpoch)

			return
		}
	}

	cursors := make([]conditionalCursorMove, 0, len(pe.cursors))
	for i := range pe.cursors {
		// remove cursor prediction except Pending validity.
		if pe.cursors[i].getValidity(emu, pe.localFrameLateAcked) != Pending {
			continue
		} else {
			cursors = append(cursors, pe.cursors[i])
		}
	}
	pe.cursors = cursors

	util.Logger.Trace("prediction message", "from", "cull",
		"predictionEpoch", pe.predictionEpoch, "confirmedEpoch", pe.confirmedEpoch,
		"overlays", len(pe.overlays), "cursors", pe.cursors)
}

// clear the whole predictions, start new epoch.
func (pe *PredictionEngine) Reset() {
	pe.cursors = make([]conditionalCursorMove, 0)
	pe.overlays = make([]conditionalOverlayRow, 0)
	pe.becomeTentative()
	// fmt.Println("reset #clear cursors and overlays")
}

// when user input happens, set last sent state num
func (pe *PredictionEngine) SetLocalFrameSent(v uint64) {
	pe.localFrameSent = v
}

// when network input happens, set first sent state num
func (pe *PredictionEngine) SetLocalFrameAcked(v uint64) {
	pe.localFrameAcked = v
}

// when network input happens, set last received remote state acked num
func (pe *PredictionEngine) SetLocalFrameLateAcked(v uint64) {
	pe.localFrameLateAcked = v
}

// when network input happens, set send interval
func (pe *PredictionEngine) SetSendInterval(value uint) {
	pe.sendInterval = value
}

// if no glitch, no flagging and prediction engine is running, return 50.
// otherwise, return max int.
func (pe *PredictionEngine) waitTime() int {
	if pe.timingTestsNecessary() && pe.active() {
		return 50
	}
	return math.MaxInt
}

/*
// backspace
theRow := pe.getOrMakeRow(pe.cursor().row, emu.GetWidth())
if pe.cursor().col > 0 {

		// move predict cursor to the previous position
		prevPredictCell := theRow.overlayCells[pe.cursor().col-1].replacement
		prevActualCell := emu.GetCell(pe.cursor().row, pe.cursor().col-1)

		// util.Logger.Trace("handleUserGrapheme", "row", pe.cursor().row, "col", pe.cursor().col,
		// 	"backspace", true,
		// 	"prePredictCell", prevPredictCell, "preActualCell", prevActualCell)

		wideCell := false
		if prevActualCell.IsDoubleWidthCont() || prevPredictCell.IsDoubleWidthCont() {
			// check the previous cell width, both predict and emulator need to check
			wideCell = true
			if pe.cursor().col-2 <= 0 {
				// util.Logger.Trace("handleUserGrapheme", "row", pe.cursor().row, "col", pe.cursor().col,
				// 	"col", 0)
				pe.cursor().col = 0
			} else {
				pe.cursor().col -= 2
			}
		} else {
			pe.cursor().col--
		}
		pe.cursor().expire(pe.localFrameSent+1, now)

		if pe.predictOverwrite {
			// clear the previous cell
			cell := &(theRow.overlayCells[pe.cursor().col]) // previous predict cell
			cell.resetWithOrig()
			cell.active = true
			cell.tentativeUntilEpoch = pe.predictionEpoch
			cell.expire(pe.localFrameSent+1, now)

			origCell := emu.GetCell(emu.GetCursorRow(), emu.GetCursorCol()) // oritinal cursor cell
			if len(cell.originalContents) == 0 {
				// avoid adding original cell content several times
				cell.originalContents = append(cell.originalContents, origCell)
			}
			cell.replacement = origCell
			cell.replacement.Clear()
			cell.replacement.Append(' ')

			if wideCell { // handle wide cell
				cell2 := &(theRow.overlayCells[pe.cursor().col+1])
				cell2.resetWithOrig()
				cell2.active = true
				cell2.tentativeUntilEpoch = pe.predictionEpoch
				cell2.expire(pe.localFrameSent+1, now)
				cell2.replacement.Clear()
				cell2.replacement.Append(' ')
			}
		} else {
			// iterate from current col to the right end, for each cell,
			// replace the current cell with next cell.
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

				if wideCell {
					i++
				}
			}
		}
	}
*/

func (pe *PredictionEngine) handleUserGrapheme(emu *terminal.Emulator, now int64, chs ...rune) {
	w := uniseg.StringWidth(string(chs))
	pe.initCursor(emu)
	util.Logger.Trace("prediction message", "from", "handleUserGrapheme.initCursor",
		"row", pe.cursor().row, "col", pe.cursor().col)

	if len(chs) == 1 && chs[0] == '\x7f' { // backspace
		pe.becomeTentative()
		// util.Logger.Trace("prediction message", "from", "handleUserGrapheme", "receive", "backspace")
	} else if len(chs) == 1 && chs[0] < 0x20 { // handle non printable control sequence
		// unknown print
		pe.becomeTentative()
	} else { // normal rune, wide rune, combining grapheme

		if pe.cursor().col+w >= emu.GetWidth() {
			// prediction in the last column is tricky
			// e.g., emacs will show wrap character, shell will just put the character there
			pe.becomeTentative()
			// util.Logger.Trace("handleUserGrapheme", "epoch", pe.predictionEpoch, "edge", "cell")
			if w == 2 && pe.cursor().col == emu.GetWidth()-1 {
				pe.newlineCarriageReturn(emu)
			}
		}
		theRow := pe.getOrMakeRow(pe.cursor().row, emu.GetWidth())

		// do the insert in reverse order
		rightMostColumn := emu.GetWidth() - 1
		if pe.predictOverwrite {
			rightMostColumn = pe.cursor().col // skip the for loop
		}
		for i := rightMostColumn; i > pe.cursor().col; i-- {
			cell := &(theRow.overlayCells[i])
			// for cell, unknown=false, active=true, will always add the replacement to originalContents
			cell.resetWithOrig()
			cell.active = true
			cell.tentativeUntilEpoch = pe.predictionEpoch
			cell.expire(pe.localFrameSent+1, now)
			if len(cell.originalContents) == 0 {
				// avoid adding original cell content several times
				cell.originalContents = append(cell.originalContents, emu.GetCell(pe.cursor().row, i))
			}

			if i-w < pe.cursor().col { // reach the left edge
				// util.Logger.Trace("handleUserGrapheme", "row", pe.cursor().row, "col", i,
				// 	"cell", cell, "break", "yes")
				break
			}

			prevCell := &(theRow.overlayCells[i-w])
			prevCellActual := emu.GetCell(pe.cursor().row, i-w)

			if i == emu.GetWidth()-1 { // the last column, unknown replacement
				cell.unknown = true
			} else if prevCell.active { // the previous prediction exist
				if prevCell.unknown {
					// don't change the replacement
					cell.unknown = true
				} else {
					// use the previous prediction cell as replacement
					cell.unknown = false
					cell.replacement = prevCell.replacement
				}
			} else {
				// use the previous actual cell as replacement
				cell.unknown = false
				cell.replacement = prevCellActual
			}

			// util.Logger.Trace("handleUserGrapheme", "row", pe.cursor().row, "col", i,
			// 	"cell", cell, "prevActualCell", prevCellActual)
		}

		cell := &(theRow.overlayCells[pe.cursor().col])
		cell.resetWithOrig()
		cell.active = true
		cell.tentativeUntilEpoch = pe.predictionEpoch
		cell.expire(pe.localFrameSent+1, now)
		cell.replacement.SetRenditions(emu.GetRenditions())

		// heuristic: match renditions of character to the left
		/*
			if pe.cursor().col > 0 {
				prevCell := &(theRow.overlayCells[pe.cursor().col-1])
				prevCellActual := emu.GetCell(pe.cursor().row, pe.cursor().col-1)

				if prevCell.active && !prevCell.unknown {
					cell.replacement.SetRenditions(prevCell.replacement.GetRenditions())
				} else {
					cell.replacement.SetRenditions(prevCellActual.GetRenditions())
				}
			}
		*/

		// wide rune occupies 2 cells.
		if w == 2 {
			cell.replacement.SetDoubleWidth(true)
			nextCell := &(theRow.overlayCells[pe.cursor().col+1])
			nextCell.replacement.SetDoubleWidthCont(true)
		}

		// set current prediction cell's replacement
		cell.replacement.SetContents(chs)
		if len(cell.originalContents) == 0 {
			// avoid adding original cell content several times
			cell.originalContents = append(cell.originalContents, emu.GetCell(pe.cursor().row, pe.cursor().col))
		}

		util.Logger.Trace("prediction message", "from", "handleUserGrapheme", "row", pe.cursor().row, "col",
			pe.cursor().col, "cell", cell)
		util.Logger.Trace("prediction message", "from", "handleUserGrapheme", "row", pe.cursor().row, "col",
			pe.cursor().col, "overlay", cell.conditionalOverlay.String())

		pe.cursor().expire(pe.localFrameSent+1, now)

		// do we need to wrap?
		if pe.cursor().col < emu.GetWidth()-w {
			pe.cursor().col += w
		} else {
			pe.becomeTentative()
			pe.newlineCarriageReturn(emu)
			// util.Logger.Trace("handleUserGrapheme", "epoch", pe.predictionEpoch, "edge", "cursor")
		}

		util.Logger.Trace("prediction message", "from", "handleUserGrapheme", "cursors", pe.cursors)
	}
}

func (pe *PredictionEngine) GetApplied() string {
	if pe.applied || len(pe.diff) == 0 {
		return ""
	}

	var sb strings.Builder
	for i := range pe.diff {
		sb.WriteString(pe.diff[i].cell.String())
	}
	// util.Logger.Trace("prediction message", "from", "GetDiff", "diff", sb.String())
	pe.diff = []appliedDiff{}
	pe.applied = true

	return sb.String()
}

func (pe *PredictionEngine) IsApplied() bool {
	return pe.applied
}

func (pe *PredictionEngine) ClearApplied() {
	pe.applied = false
}

type appliedDiff struct {
	cell     terminal.Cell
	row, col int
}

type diffFunc func(row, col int, cell terminal.Cell)

// represent the prediction title prefix.
type TitleEngine struct {
	prefix string
}

func (te *TitleEngine) setPrefix(v string) {
	te.prefix = v
}

// set prefix title for terminal
func (te *TitleEngine) apply(emu *terminal.Emulator) {
	emu.PrefixWindowTitle(te.prefix)
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

func (om *OverlayManager) WaitTime() int {
	w1 := om.notifications.waitTime()
	w2 := om.predictions.waitTime()
	// util.Log.Debug("waitTime", "predictions", w2, "notifications", w1)
	return min(w1, w2)

	// return terminal.Min(om.notifications.waitTime(), om.predictions.waitTime())
}

func (om *OverlayManager) Apply(emu *terminal.Emulator) {
	util.Logger.Trace("prediction message", "from", "OverlayManager.Apply",
		"state", fmt.Sprintf("%p", emu),
		"predictionEpoch", om.GetPredictionEngine().predictionEpoch)
	om.predictions.cull(emu)
	om.predictions.apply(emu)

	om.notifications.adjustMessage()
	om.notifications.apply(emu)

	om.title.apply(emu)
}
