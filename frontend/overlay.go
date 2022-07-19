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

import "github.com/ericwq/aprilsh/terminal"

type Validity uint

const (
	Pending Validity = iota
	Correct
	ValidityCorrectNoCredit
	IncorrectOrExpired
	Inactive
)

type ConditionalOverlay struct {
	expirationFrame     int64
	col                 int
	active              bool  // represents a prediction at all, default value false
	tentativeUntilEpoch int64 // when to show
	predictionTiem      int64 // used to find long-pending predictions, default value -1
}

// if the overlay is ready?
func (co *ConditionalOverlay) tentative(confirmedEpoch int64) bool {
	return co.tentativeUntilEpoch > confirmedEpoch
}

func (co *ConditionalOverlay) reset() {
	co.expirationFrame = -1
	co.active = false
}

func (co *ConditionalOverlay) expire(expirationFrame, now int64) {
	co.expirationFrame = expirationFrame
	co.predictionTiem = now
}

type ConditionalCursorMove struct {
	ConditionalOverlay
	row int
}

// set cursor position in emulator if the confirmedEpoch is greater than tantative epoch.
func (ccm *ConditionalCursorMove) apply(emu *terminal.Emulator, confirmedEpoch int64) {
	if !ccm.active { // only apply to active prediction
		return
	}

	if ccm.tentative(confirmedEpoch) { // only apply to specified epoch
		return
	}

	emu.MoveCursor(ccm.row, ccm.col)
}

// return Correct only when lateAck is greater than expirationFrame and cursor position is at the
// same position.
func (ccm *ConditionalCursorMove) getValidity(emu *terminal.Emulator, lateAck int64) (v Validity) {
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
	replacement      terminal.Cell
	unknown          bool
	originalContents []terminal.Cell // we don't give credit for correct predictions that match the original contents
}

func (coc *ConditionalOverlayCell) apply(emu *terminal.Emulator, confirmedEpoch int64, row int, flag bool) {
	// if specified position is out of active area or is not active.
	if !coc.active || row >= emu.GetHeight() || coc.col >= emu.GetWidth() {
		return
	}

	if coc.tentative(confirmedEpoch) { // only apply to specified epoch
		return
	}
}
