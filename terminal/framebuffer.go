// Copyright 2022 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package terminal

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/ericwq/aprilsh/util"
)

const (
	SaveLineUpperLimit  = 50000
	windowTitleStackMax = 9
	SaveLinesRowsOption = 60
)

// support both (scrollable) normal screen buffer and alternate screen buffer
type Framebuffer struct {
	cells        []Cell       // the cells
	selection    Rect         // selection area
	cursor       Cursor       // current cursor style, color and position
	damage       Damage       // damage scope
	scrollHead   int          // row offset of scrolling area's logical top row
	marginBottom int          // current margin bottom (number of rows above + 1), the bottom row of scrolling area.
	historyRows  int          // number of history (off-screen) rows with data
	viewOffset   int          // how many rows above top row does the view start? screen view start position
	marginTop    int          // current margin top (number of rows above), the top row of scrolling area.
	nCols        int          // cols number per window
	saveLines    int          // nRows + saveLines is the scrolling area limitation
	nRows        int          // rows number per window
	margin       bool         // are there (non-default) top/bottom margins set?
	snapTo       SelectSnapTo // selection state
}

// create a framebuffer, with zero saveLines.
func NewFramebuffer2(nCols, nRows int) Framebuffer {
	pfb, _, _ := NewFramebuffer3(nCols, nRows, 0)
	return pfb
}

// create a framebuffer according to the specified nCols, nRows and saveLines.
// saveLines: for alternate screen buffer default is 0, for normal screen buffer the default is 500, max 50000
// return the framebuffer and external marginTop,marginBottom.
func NewFramebuffer3(nCols, nRows, saveLines int) (fb Framebuffer, marginTop int, marginBottom int) {
	fb = Framebuffer{}

	fb.cells = make([]Cell, nCols*(nRows+saveLines))
	fb.nCols = nCols
	fb.nRows = nRows

	// saveLines limitation is 50000
	if saveLines > SaveLineUpperLimit {
		saveLines = SaveLineUpperLimit
	}
	fb.saveLines = saveLines
	fb.scrollHead = 0
	fb.marginTop = 0
	fb.marginBottom = nRows + saveLines
	fb.historyRows = 0
	fb.viewOffset = 0
	fb.margin = false

	fb.damage.totalCells = nCols * (nRows + saveLines)
	fb.selection = *NewRect()
	fb.snapTo = SelectSnapTo_Char

	marginTop = fb.marginTop
	marginBottom = fb.nRows

	return
}

func (fb *Framebuffer) resize(nCols, nRows int) (marginTop, marginBottom int) {
	// marginTop = fb.marginTop
	// marginBottom = nRows

	// TestDiffFrom/vi_and_quit need to disable it.
	// if fb.nCols == nCols && fb.nRows == nRows {
	// 	// fmt.Printf("Framebuffer.resize same cols*rows\n")
	// 	return
	// }

	// adjust the internal cell storage according to the new size
	// the defalt constructor of Cell set the contents field to ""
	newCells := make([]Cell, nCols*(nRows+fb.saveLines))

	rowLen := min(fb.nCols, nCols)    // minimal row length
	nCopyRows := min(fb.nRows, nRows) // minimal row number

	// copy the active area
	for pY := 0; pY < nCopyRows; pY++ {
		srcStartIdx := fb.getPhysRowIdx(pY)
		srcEndIdx := srcStartIdx + rowLen
		dstStartIdx := nCols * pY
		copy(newCells[dstStartIdx:], fb.cells[srcStartIdx:srcEndIdx])
	}
	// copy the history rows
	base := (nRows + fb.saveLines - fb.historyRows) * nCols
	j := 0
	for pY := -fb.historyRows; pY < 0; pY++ {
		srcStartIdx := fb.getPhysRowIdx(pY)
		srcEndIdx := srcStartIdx + rowLen
		dstStartIdx := base + nCols*j
		copy(newCells[dstStartIdx:], fb.cells[srcStartIdx:srcEndIdx])
		j++
	}

	fb.cells = newCells
	fb.nCols = nCols
	fb.nRows = nRows
	fb.marginTop = 0
	fb.scrollHead = fb.marginTop
	fb.marginBottom = fb.nRows + fb.saveLines // internal marginBottom
	fb.margin = false
	fb.viewOffset = 0
	fb.damage.totalCells = fb.nCols * (fb.nRows + fb.saveLines)

	marginTop = 0
	marginBottom = nRows // external report marginBottom

	// fmt.Printf("Framebuffer.resize nCols=%d, nRows=%d, saveLines=%d\n", nCols, nRows, fb.saveLines)
	return
}

// drop the scrollback history and view offset
func (fb *Framebuffer) dropScrollbackHistory() {
	fb.viewOffset = 0
	fb.historyRows = 0
	fb.expose()
}

func (fb *Framebuffer) setMargins(marginTop, marginBottom int) {
	fb.unwrapCellStorage()
	fb.marginTop = marginTop
	fb.scrollHead = fb.marginTop
	fb.marginBottom = marginBottom
	fb.margin = true
	fb.expose()
}

// return marginTop = 0, marginBottom = nRows
func (fb *Framebuffer) resetMargins() (marginTop, marginBottom int) {
	fb.unwrapCellStorage()
	marginTop = 0
	fb.marginTop = marginTop
	fb.scrollHead = fb.marginTop
	fb.marginBottom = fb.nRows + fb.saveLines // internal marginBottom = nRows+saveLines
	marginBottom = fb.nRows                   // external reported value of marginBottom
	fb.margin = false
	fb.expose()

	return
}

// fill current screen with specified rune and renditions.
// ch = 0x00 meabs clear the contents.
func (fb *Framebuffer) fillCells(ch rune, attrs Cell) {
	for r := 0; r < fb.nRows; r++ {

		start := fb.getIdx(r, 0)
		end := start + fb.nCols
		for k := start; k < end; k++ {
			fb.cells[k] = attrs
			fb.cells[k].contents = string(ch)
		}
		fb.damage.add(start, end)
	}
}

// copy screen view to dst, dst must be allocated in advance.
// dst := make([]Cell, fb.nCols*fb.nRows)
func (fb *Framebuffer) fullCopyCells(dst []Cell) {
	for pY := 0; pY < fb.nRows; pY++ {
		srcStartIdx := fb.getViewRowIdx(pY)
		srcEndIdx := srcStartIdx + fb.nCols
		dstStartIdx := fb.nCols * pY
		// fmt.Printf("#fullCopyCells copy from src[%d:%d] to dst[%d:]\n",
		// 	srcStartIdx, srcEndIdx, dstStartIdx)
		copy(dst[dstStartIdx:], fb.cells[srcStartIdx:srcEndIdx])
	}
}

func (fb *Framebuffer) deltaCopyCells(dst []Cell) {
	dstIdx := 0
	for pY := -fb.viewOffset; pY < fb.nRows-fb.viewOffset; pY++ {
		fb.damageDeltaCopy(dst[dstIdx:], fb.nCols*fb.getPhysicalRow(pY), fb.nCols)
		dstIdx += fb.nCols
	}
}

func (fb *Framebuffer) freeCells() {
	fb.cells = nil
}

// return a copy of the specified cell
func (fb *Framebuffer) getCell(pY, pX int) (cell Cell) {
	idx := fb.getIdx(pY, pX)
	cell = fb.cells[idx]
	return
}

// retrun a reference of the specified cell
func (fb *Framebuffer) getCellPtr(pY, pX int) (cell *Cell) {
	idx := fb.getIdx(pY, pX)
	fb.damage.add(idx, idx+1)
	fb.invalidateSelection(NewRect4(pX, pY, pX+1, pY))

	cell = &(fb.cells[idx])
	return
}

// erase (count) cells from startX column
func (fb *Framebuffer) eraseInRow(pY, startX, count int, attrs Cell) {
	if count == 0 {
		return
	}

	idx := fb.getIdx(pY, startX)
	fb.eraseRange(idx, idx+count, attrs)
	fb.invalidateSelection(NewRect4(startX, pY, startX+count, pY))
}

// move (count) cells from srcX column to dstX column in row pY
func (fb *Framebuffer) moveInRow(pY, dstX, srcX, count int) {
	if count == 0 {
		return
	}

	dstIdx := fb.getIdx(pY, dstX)
	srcIdx := fb.getIdx(pY, srcX)
	fb.moveCells(dstIdx, srcIdx, count)
	fb.invalidateSelection(NewRect4(dstX, pY, dstX+count, pY))
}

// copy a row from srcY to dstY, within the left-right scrolling area. (startX,count) defines the area.
func (fb *Framebuffer) copyRow(dstY, srcY, startX, count int) {
	if count == 0 {
		return
	}

	dstIdx := fb.getIdx(dstY, startX)
	srcIdx := fb.getIdx(srcY, startX)
	fb.copyCells(dstIdx, srcIdx, count)
	fb.invalidateSelection(NewRect4(startX, dstY, startX+count, dstY))
}

// text up, move scrolling area down count rows
func (fb *Framebuffer) scrollUp(count int) {
	fb.vscrollSelection(-count)
	for k := 0; k < count; k++ {
		fb.scrollHead += 1
		if fb.scrollHead == fb.marginBottom {
			// wrap around the end of the scrolling area
			fb.scrollHead = fb.marginTop
		}
	}
	fb.historyRows = min(fb.historyRows+count, fb.saveLines)
	fb.damage.add(fb.marginTop*fb.nCols, fb.marginBottom*fb.nCols)
}

// text down, move scrolling area up count rows
func (fb *Framebuffer) scrollDown(count int) {
	fb.vscrollSelection(count)
	for k := 0; k < count; k++ {
		if fb.scrollHead >= fb.marginTop+1 {
			fb.scrollHead -= 1
		} else {
			// wrap around the head of the scrolling area
			fb.scrollHead = fb.marginBottom - 1
		}
	}
	fb.historyRows = max(0, fb.historyRows-count)
	fb.damage.add(fb.marginTop*fb.nCols, fb.marginBottom*fb.nCols)
}

// text down, screen up count rows
func (fb *Framebuffer) pageUp(count int) {
	viewOffset := min(fb.viewOffset+count, fb.historyRows)
	delta := viewOffset - fb.viewOffset
	fb.cursor.posY += delta
	fb.selection.br.y += delta
	fb.selection.tl.y += delta
	fb.viewOffset = viewOffset
	fb.expose()
}

// text up, screen down count rows
func (fb *Framebuffer) pageDown(count int) {
	viewOffset := max(0, fb.viewOffset-count)
	delta := viewOffset - fb.viewOffset
	fb.cursor.posY += delta
	fb.selection.br.y += delta
	fb.selection.tl.y += delta
	fb.viewOffset = viewOffset
	fb.expose()
}

// text up, screen down to the scrollHead row
func (fb *Framebuffer) pageToBottom() {
	if fb.viewOffset == 0 {
		return
	}

	fb.cursor.posY -= fb.viewOffset
	fb.selection.br.y -= fb.viewOffset
	fb.selection.tl.y -= fb.viewOffset
	fb.viewOffset = 0
	fb.expose()
}

func (fb *Framebuffer) getHistroryRows() int {
	return fb.historyRows
}

func (fb *Framebuffer) expose() {
	fb.damage.expose()
}

func (fb *Framebuffer) resetDamage() {
	fb.damage.reset()
}

func (fb *Framebuffer) getCursor() (cursor Cursor) {
	cursor = fb.cursor
	return
}

func (fb *Framebuffer) setCursorPos(pY, pX int) {
	fb.cursor.posY = pY + fb.viewOffset
	fb.cursor.posX = pX
}

func (fb *Framebuffer) setCursorStyle(cs CursorStyle) {
	fb.cursor.style = cs
}

func (fb *Framebuffer) setSelectSnapTo(snapTo SelectSnapTo) {
	fb.snapTo = snapTo
}

func (fb *Framebuffer) cycleSelectSnapTo() {
	fb.snapTo = cycleSelectSnapTo2(fb.snapTo)
}

func (fb *Framebuffer) getSelection() Rect {
	return fb.selection
}

func (fb *Framebuffer) getSelectionPtr() *Rect {
	return &fb.selection
}

func (fb *Framebuffer) getSnappedSelection() (ret Rect) {
	ret = fb.getSelection()

	if ret.null() || ret.empty() {
		return ret
	}

	if ret.rectangular {
		return ret
	}

	switch fb.snapTo {
	case SelectSnapTo_Char:
		break
	case SelectSnapTo_Word:
		cp := fb.getViewRowIdx(ret.tl.y) // it's the base
		for ret.tl.x < fb.nCols && fb.cells[cp+ret.tl.x].IsBlank() {
			ret.tl.x++ // find the next non-blank cell
		}
		for ret.tl.x > 0 && !fb.cells[cp+ret.tl.x-1].IsBlank() {
			ret.tl.x-- // find the previous blank cell
		}

		cp = fb.getViewRowIdx(ret.br.y)
		for ret.br.x > 0 && fb.cells[cp+ret.br.x].IsBlank() {
			ret.br.x--
		}
		for ret.br.x < fb.nCols && !fb.cells[cp+ret.br.x].IsBlank() {
			ret.br.x++
		}
	case SelectSnapTo_Line:
		ret.tl.x = 0
		ret.br.x = fb.nCols
	default:
	}
	return
}

func (fb *Framebuffer) getSelectedUtf8() (ok bool, utf8Selection string) {
	sel := fb.getSnappedSelection()

	if sel.empty() {
		ok = false
		return
	}

	var lines []string
	var wrap bool

	// fmt.Printf("#getSelectedUtf8 selection=%s\n", &sel)

	addLine := func(y, x1, x2 int) {
		var line strings.Builder
		wrapBack := wrap
		wrap = false

		cp := fb.getViewRowIdx(y)
		for x := x1; x < x2; x++ {
			cell := &fb.cells[cp+x]
			if !cell.dwidthCont {
				line.WriteString(cell.contents)
			}
			if cell.wrap {
				wrap = true
				break
			}
		}

		ln := line.String()
		if !wrap && line.Len() > 0 { // discard trailing whitespace
			ln = strings.TrimRightFunc(line.String(), unicode.IsSpace)
		}
		// fmt.Printf("#getSelectedUtf8 trim line:\n%q\n", ln)

		if wrapBack && len(lines) > 0 { // deal with extreme long line
			lines[len(lines)-1] = fmt.Sprintf("%s%s", lines[len(lines)-1], ln)
		} else {
			lines = append(lines, ln)
		}
	}

	if sel.tl.y == sel.br.y { // selection area is in the same line
		addLine(sel.tl.y, sel.tl.x, sel.br.x)
	} else if sel.rectangular { // selection area is rectangular
		for y := sel.tl.y; y <= sel.br.y; y++ {
			addLine(y, sel.tl.x, sel.br.x)
		}
	} else { // selection area contains multi lines
		addLine(sel.tl.y, sel.tl.x, fb.nCols)
		for y := sel.tl.y + 1; y < sel.br.y; y++ {
			addLine(y, 0, fb.nCols)
		}
		addLine(sel.br.y, 0, sel.br.x)
	}

	var b strings.Builder
	for i := range lines {
		b.WriteString(fmt.Sprintf("%s\n", lines[i]))
	}

	// discard trailing whitespace
	utf8Selection = strings.TrimRightFunc(b.String(), func(x rune) bool {
		if x == '\n' {
			return true
		} else {
			return false
		}
	})
	if len(utf8Selection) > 0 {
		ok = true
	}

	return
}

// viewOffset is used to display something other than the active area.
// scrollHead marks the current logical top of the scrolling area.
func (fb *Framebuffer) getPhysicalRow(pY int) int {
	// defer func() {
	// 	fmt.Printf("#getPhysicalRow scrollHead=%d, nRows=%d, saveLines=%d -> pY=%d\n",
	// 		fb.scrollHead, fb.nRows, fb.saveLines, pY)
	// }()

	if pY < 0 {
		if !fb.margin {
			pY += fb.scrollHead
		}
		if pY < 0 {
			pY += fb.nRows + fb.saveLines
		}
		return pY
	}

	// margin rows keeps unchanged
	if fb.margin && (pY < fb.marginTop || pY >= fb.marginBottom) {
		return pY
	}

	// map the row according to scrollHead and marginTop
	pY += fb.scrollHead - fb.marginTop
	if pY >= fb.marginBottom {
		// wrap the buffer
		pY -= fb.marginBottom - fb.marginTop
	}

	return pY
}

func (fb *Framebuffer) getPhysRowIdx(pY int) int {
	return fb.nCols * fb.getPhysicalRow(pY)
}

// when preparing frame data for display, viewOffset has to be subtracted
// from the start of the frame (scrollHead in case of no margins, or 0 if
// there are margins), wrapping around the buffer limits as necessary.
// Then, the data has to be copied from that starting point in order,
// until nRows rows have been copied.
func (fb *Framebuffer) getViewRowIdx(pY int) int {
	return fb.getPhysRowIdx(pY - fb.viewOffset)
}

func (fb *Framebuffer) getIdx(pY, pX int) int {
	return fb.nCols*fb.getPhysicalRow(pY-fb.viewOffset) + pX
}

// erase (reset) fromt start to end with specified renditions
func (fb *Framebuffer) eraseRange(start, end int, attrs Cell) {
	for i := range fb.cells[start:end] {
		fb.cells[start+i].Reset2(attrs)
	}
	fb.damage.add(start, end)
}

// copy (count) cells from srcIx to dstIx. Both of the parameters are the index for cells.
func (fb *Framebuffer) copyCells(dstIx, srcIx, count int) {
	copy(fb.cells[dstIx:], fb.cells[srcIx:srcIx+count])
	fb.damage.add(dstIx, dstIx+count)
}

// move (count) cells from srcIx to dstIx. Both of the parameters are the index for cells.
func (fb *Framebuffer) moveCells(dstIx, srcIx, count int) {
	copy(fb.cells[dstIx:], fb.cells[srcIx:srcIx+count])
	fb.damage.add(dstIx, dstIx+count)
}

// copy range is specified by start and count. if the copy range is not intersect with
// damage area, copy nothing, Otherwise, if damage area is smaller than copy range,
// the real copy range is determined by damage area. if damage area is bigger than copy
// range, the real copy range is determined by copy range.
//
// dst must be allocated in advance.
func (fb *Framebuffer) damageDeltaCopy(dst []Cell, start, count int) {
	end := start + count

	// fmt.Printf("#damageDeltaCopy start=%d, count=%d\n", start, count)
	if fb.damage.end <= start || end <= fb.damage.start {
		return // no intersection
	}

	base := 0
	if start < fb.damage.start {
		base += fb.damage.start - start // change the base, skip the un-damage part.
		start = fb.damage.start
	}

	if fb.damage.end < end {
		end = fb.damage.end
	}

	// fmt.Printf("#damageDeltaCopy start=%d, count=%d, base=%d\n", start, count, base)
	i := 0
	for j := start; j < end; j++ {
		if dst[base+i] != fb.cells[j] {
			dst[base+i] = fb.cells[j]
			dst[base+i].dirty = true
		}
		i++
	}
}

func (fb *Framebuffer) copyAllCells(dst []Cell) {
	// copy the active area
	for pY := 0; pY < fb.nRows; pY++ {
		srcStartIdx := fb.nCols * fb.getPhysicalRow(pY)
		srcEndIdx := srcStartIdx + fb.nCols
		dstStartIdx := fb.nCols * pY
		copy(dst[dstStartIdx:], fb.cells[srcStartIdx:srcEndIdx])
	}
	// copy the history rows
	base := (fb.nRows + fb.saveLines - fb.historyRows) * fb.nCols
	j := 0
	for pY := -fb.historyRows; pY < 0; pY++ {
		srcStartIdx := fb.nCols * fb.getPhysicalRow(pY)
		srcEndIdx := srcStartIdx + fb.nCols
		dstStartIdx := base + fb.nCols*j
		copy(dst[dstStartIdx:], fb.cells[srcStartIdx:srcEndIdx])
		j++
	}
}

// rearrange the storage to solve the wrap around case
// resize and set top/bottom margin will call this function
func (fb *Framebuffer) unwrapCellStorage() {
	if fb.scrollHead == fb.marginTop {
		return
	}
	newCells := make([]Cell, fb.nCols*(fb.nRows+fb.saveLines))
	fb.copyAllCells(newCells)
	fb.cells = newCells
	fb.scrollHead = fb.marginTop
}

// move the selection area vertically: up and down.
func (fb *Framebuffer) vscrollSelection(vertOffset int) {
	if fb.selection.null() {
		return
	}

	y1 := fb.selection.tl.y + vertOffset
	y2 := fb.selection.br.y + vertOffset

	if (fb.margin && y1 < fb.marginTop) || y1 < -fb.saveLines ||
		y2 > fb.marginBottom || (y2 == fb.marginBottom && fb.selection.br.x > 0) {
		fb.selection.clear()
		return
	}

	fb.selection.tl.y = y1
	fb.selection.br.y = y2
}

// clear the selection area if it overlaped with damage area
func (fb *Framebuffer) invalidateSelection(damage *Rect) {
	if fb.selection.empty() {
		return
	}

	// damage area is not overlaped with selection area
	if fb.selection.br.lessEqual(damage.tl) || damage.br.lessEqual(fb.selection.tl) {
		return
	}

	fb.selection.clear()
}

// return row cells, rowY is tht physical row index, see getPhysicalRow() for detail
func (fb *Framebuffer) getRow(rowY int) []Cell {
	start := fb.nCols * rowY
	end := start + fb.nCols
	return fb.cells[start:end]
}

// return the active area reference
// func (fb *Framebuffer) getScreenRef() []Cell {
// 	startIdx := fb.nCols * fb.getPhysicalRow(0)
// 	endIdx := startIdx + (fb.nRows)*fb.nCols
// 	maxIdx := len(fb.cells)
// 	if endIdx > maxIdx || startIdx > maxIdx {
// 		util.Log.Warn("getScreenRef",
// 			"startIdx", startIdx,
// 			"endIdx", endIdx,
// 			"length", maxIdx)
// 	}
// 	return fb.cells[startIdx:endIdx]
// }

func (fb *Framebuffer) Equal(x *Framebuffer) bool {
	return fb.equal(x, false)
}

func (fb *Framebuffer) equal(x *Framebuffer, trace bool) (ret bool) {
	ret = true
	if fb.nCols != x.nCols || fb.nRows != x.nRows || fb.saveLines != x.saveLines ||
		fb.marginTop != x.marginTop || fb.marginBottom != x.marginBottom || fb.margin != x.margin {
		if trace {
			msg := fmt.Sprintf("nCols=(%d,%d), nRows=(%d,%d), saveLines=(%d,%d)",
				fb.nCols, x.nCols, fb.nRows, x.nRows, fb.saveLines, x.saveLines)
			util.Logger.Warn(msg)
			msg = fmt.Sprintf("marginTop=(%d,%d), marginBottom=(%d,%d), cells length=(%d,%d)",
				fb.marginTop, x.marginTop, fb.marginBottom, x.marginBottom, len(fb.cells), len(x.cells))
			util.Logger.Warn(msg)
			ret = false
		} else {
			return false
		}
	}

	if fb.scrollHead != x.scrollHead || fb.historyRows != x.historyRows ||
		fb.viewOffset != x.viewOffset {
		if trace {
			msg := fmt.Sprintf("scrollHead=(%d,%d), historyRows=(%d,%d), viewOffset=(%d,%d)",
				fb.scrollHead, x.scrollHead, fb.historyRows, x.historyRows, fb.viewOffset, x.viewOffset)
			util.Logger.Warn(msg)
			ret = false
		} else {
			return false
		}
	}

	if fb.cursor.color != x.cursor.color || fb.cursor.showStyle != x.cursor.showStyle {
		if trace {
			msg := fmt.Sprintf("cursor.color=(%d,%d), cursor.showStyle=(%d,%d)",
				fb.cursor.color, x.cursor.color, fb.cursor.showStyle, x.cursor.showStyle)
			util.Logger.Warn(msg)
			ret = false
		} else {
			return false
		}
	}

	if fb.selection != x.selection || fb.snapTo != x.snapTo || fb.damage != x.damage {
		if trace {
			msg := fmt.Sprintf("selection=(%v,%v), snapTo=(%v,%v), damage=(%v,%v)",
				fb.selection, x.selection, fb.snapTo, x.snapTo, fb.damage, x.damage)
			util.Logger.Warn(msg)
			ret = false
		} else {
			return false
		}
	}

	// same terminal size, check different content
	if fb.saveLines == 0 { // no saveLines
		for pY := 0; pY < fb.nRows; pY++ {
			srcStartIdx := fb.getViewRowIdx(pY)
			srcEndIdx := srcStartIdx + fb.nCols
			dstStartIdx := x.getViewRowIdx(pY)
			dstEndIdx := dstStartIdx + x.nCols

			newR := fb.cells[srcStartIdx:srcEndIdx]
			oldR := x.cells[dstStartIdx:dstEndIdx]
			if !equalRow(newR, oldR) {
				if trace {
					util.Logger.Warn("equal", "newRow", outputRow(newR, pY, fb.nCols))
					util.Logger.Warn("equal", "oldRow", outputRow(oldR, pY, x.nCols))
				}
				ret = false
				break
			}
		}
	} else { // has saveLines
		for i := 0; i < len(fb.cells); i++ {
			if fb.cells[i] != x.cells[i] {
				if trace {
					row := i / fb.nCols
					util.Logger.Warn("equal", "newRow", printRow(fb.cells, row, fb.nCols))
					util.Logger.Warn("equal", "oldRow", printRow(x.cells, row, x.nCols))
					ret = false
					// break
					i += fb.nCols - 1
				} else {
					return false
				}
			}
		}
	}

	return ret
}

func (fb *Framebuffer) reachMaxRows(lastRows int) bool {
	return lastRows >= fb.marginBottom-1
}

func (fb *Framebuffer) isFullFrame(lastRows int, oldR int, newR int) bool {
	return fb.getRowsGap(oldR, newR)+lastRows == fb.marginBottom-1
}

func (fb *Framebuffer) getRowsGap(oldR int, newR int) (gap int) {
	if oldR == newR {
		gap = 0
	} else if oldR > newR {
		gap = fb.marginBottom - oldR + newR
	} else {
		// new row > old row
		gap = newR - oldR
	}
	return gap
}

func printRow(cells []Cell, row int, nCols int) string {
	base := row * nCols
	return outputRow(cells[base:base+nCols], row, nCols)
}

func outputRow(row []Cell, rowIdx int, nCols int) string {
	var b strings.Builder
	base := 0

	b.WriteString(fmt.Sprintf("[%3d]", rowIdx))
	for i := 0; i < nCols; i++ {
		if row[base+i].dwidthCont {
			continue
		}
		b.WriteString(row[base+i].contents)
	}
	return b.String()
}
