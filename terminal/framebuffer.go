/*

MIT License

Copyright (c) 2022~2023 wangqi

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

package terminal

import (
	"fmt"
	"strings"
	"unicode"
)

const (
	SaveLineUpperLimit = 50000
)

// support both (scrollable) normal screen buffer and alternate screen buffer
type Framebuffer struct {
	nCols        int          // cols number per window
	nRows        int          // rows number per window
	saveLines    int          // nRows + saveLines is the scrolling area limitation
	scrollHead   int          // row offset of scrolling area's logical top row
	marginTop    int          // current margin top (number of rows above), the top row of scrolling area.
	marginBottom int          // current margin bottom (number of rows above + 1), the bottom row of scrolling area.
	historyRows  int          // number of history (off-screen) rows with data
	viewOffset   int          // how many rows above top row does the view start? screen view start position
	margin       bool         // are there (non-default) top/bottom margins set?
	cells        []Cell       // the cells
	cursor       Cursor       // current cursor style, color and position
	selection    Rect         // selection area
	snapTo       SelectSnapTo // selection state
	damage       Damage       // damage scope

	iconLabel        string // replicated by NewFrame()
	windowTitle      string // replicated by NewFrame()
	bellCount        int    // replicated by NewFrame()
	titleInitialized bool   // replicated by NewFrame()
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
	if fb.nCols == nCols && fb.nRows == nRows {
		return
	}

	// adjust the internal cell storage according to the new size
	// the defalt constructor of Cell set the contents field to ""
	newCells := make([]Cell, nCols*(nRows+fb.saveLines))

	rowLen := Min(fb.nCols, nCols)    // minimal row length
	nCopyRows := Min(fb.nRows, nRows) // minimal row number

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
	fb.historyRows = Min(fb.historyRows+count, fb.saveLines)
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
	fb.historyRows = Max(0, fb.historyRows-count)
	fb.damage.add(fb.marginTop*fb.nCols, fb.marginBottom*fb.nCols)
}

// text down, screen up count rows
func (fb *Framebuffer) pageUp(count int) {
	viewOffset := Min(fb.viewOffset+count, fb.historyRows)
	delta := viewOffset - fb.viewOffset
	fb.cursor.posY += delta
	fb.selection.br.y += delta
	fb.selection.tl.y += delta
	fb.viewOffset = viewOffset
	fb.expose()
}

// text up, screen down count rows
func (fb *Framebuffer) pageDown(count int) {
	viewOffset := Max(0, fb.viewOffset-count)
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

/* --------------------------------------------------- new frame end here */
/*
func (fb *Framebuffer) newRow() *Row {
	w := fb.DS.GetWidth()
	bgColor := fb.DS.GetBackgroundRendition()
	return NewRow(w, bgColor)
}

func (fb *Framebuffer) GetRows() []Row { return fb.rows }

// it is get_mutable_row go version
// there is no inline counterpart in go
func (fb *Framebuffer) GetRow(row int) *Row {
	if row == -1 {
		row = fb.DS.GetCursorRow()
	}

	return &(fb.rows[row])
}

// it is get_mutable_cell go version
// there is no inline counterpart in go
func (fb *Framebuffer) GetCell(row, col int) *Cell {
	if row < 0 || row > len(fb.rows)-1 {
		row = fb.DS.GetCursorRow()
	}

	if col < 0 || col > len(fb.GetRow(0).cells)-1 {
		col = fb.DS.GetCursorCol()
	}

	return fb.GetRow(row).At(col)
}

func (fb *Framebuffer) erase(start int, count int) {
	// delete count rows
	copy(fb.rows[start:], fb.rows[start+count:])
	// fb.rows = fb.rows[:len(fb.rows)-count]
}

func (fb *Framebuffer) insert(start int, count int) {
	// insert count rows
	fb.rows = append(fb.rows[:start+count], fb.rows[start:]...)

	// fill the row : copy or pointer?
	for i := start; i < start+count; i++ {
		fb.rows[i] = *(fb.newRow())
	}

	// remove the extra one
	fb.rows = fb.rows[:len(fb.rows)-count]
}

func (fb *Framebuffer) Scroll(N int) {
	if N >= 0 {
		fb.DeleteLine(fb.DS.GetScrollingRegionTopRow(), N)
	} else {
		fb.InsertLine(fb.DS.GetScrollingRegionTopRow(), -N)
	}
}

func (fb *Framebuffer) MoveRowsAutoscroll(rows int) {
	// don't scroll if outside the scrolling region
	if fb.DS.GetCursorRow() < fb.DS.GetScrollingRegionTopRow() || fb.DS.GetCursorRow() > fb.DS.GetScrollingRegionBottomRow() {
		fb.DS.MoveRow(rows, true)
		return
	}

	if fb.DS.GetCursorRow()+rows > fb.DS.GetScrollingRegionBottomRow() {
		N := fb.DS.GetCursorRow() + rows - fb.DS.GetScrollingRegionBottomRow()
		fb.Scroll(N)
		fb.DS.MoveRow(-N, true)
	} else if fb.DS.GetCursorRow()+rows < fb.DS.GetScrollingRegionTopRow() {
		N := fb.DS.GetCursorRow() + rows - fb.DS.GetScrollingRegionTopRow()
		fb.Scroll(N)
		fb.DS.MoveRow(-N, true)
	}

	fb.DS.MoveRow(rows, true)
}

// TODO the meaning.
func (fb *Framebuffer) GetCombiningCell() *Cell {
	if fb.DS.GetCombiningCharCol() < 0 || fb.DS.GetCombiningCharRow() < 0 || fb.DS.GetCombiningCharCol() >= fb.DS.GetWidth() || fb.DS.GetCombiningCharRow() >= fb.DS.GetHeight() {
		return nil
	}
	return fb.GetCell(fb.DS.GetCombiningCharRow(), fb.DS.GetCombiningCharCol())
}

func (fb *Framebuffer) ApplyRenditionsToCell(cell *Cell) {
	if cell == nil {
		// get cursor cell
		cell = fb.GetCell(-1, -1)
	}
	cell.SetRenditions(*(fb.DS.GetRenditions()))
}

func (fb *Framebuffer) InsertLine(beforeRow int, count int) bool { // #BehaviorChange return bool
	// validate beforeRow
	if beforeRow < fb.DS.GetScrollingRegionTopRow() || beforeRow > fb.DS.GetScrollingRegionBottomRow()+1 {
		return false
	}

	maxRoll := fb.DS.GetScrollingRegionBottomRow() + 1 - beforeRow
	if count > maxRoll {
		count = maxRoll
	}

	if count <= 0 { // #BehaviorChange: original count ==0
		return false
	}

	// delete old rows
	start := 0 + fb.DS.GetScrollingRegionBottomRow() + 1 - count
	fb.erase(start, count)

	// insert new rows
	start = 0 + beforeRow
	fb.insert(start, count)

	return true
}

func (fb *Framebuffer) DeleteLine(row, count int) bool { // #BehaviorChange return bool
	// validate row
	if row < fb.DS.GetScrollingRegionTopRow() || row > fb.DS.GetScrollingRegionBottomRow() {
		return false
	}

	maxRoll := fb.DS.GetScrollingRegionBottomRow() + 1 - row
	if count > maxRoll {
		count = maxRoll
	}

	if count <= 0 { // #BehaviorChange: original count ==0
		return false
	}

	// delete old rows
	start := 0 + row
	fb.erase(start, count)

	// insert new rows
	start = 0 + fb.DS.GetScrollingRegionBottomRow() + 1 - count
	fb.insert(start, count)

	return true
}

func (fb *Framebuffer) InsertCell(row, col int) bool {
	if row < 0 || row > len(fb.rows)-1 || col < 0 || col > fb.DS.GetWidth()-1 {
		return false
	}
	fb.GetRow(row).InsertCell(col, uint32(fb.DS.GetBackgroundRendition()))
	return true
}

func (fb *Framebuffer) DeleteCell(row, col int) bool {
	if row < 0 || row > len(fb.rows)-1 || col < 0 || col > fb.DS.GetWidth()-1 {
		return false
	}
	fb.GetRow(row).DeleteCell(col, uint32(fb.DS.GetBackgroundRendition()))
	return true
}

func (fb *Framebuffer) Reset() {
	width := fb.DS.GetWidth()
	height := fb.DS.GetHeight()

	fb.DS = NewDrawState(width, height)

	fb.rows = make([]Row, height)
	for i := range fb.rows {
		fb.rows[i] = *fb.newRow()
	}
	fb.windowTitle = ""
	// do not reset bell_count
}

func (fb *Framebuffer) SoftReset() {
	fb.DS.InsertMode = false
	fb.DS.OriginMode = false
	fb.DS.CursorVisible = false // per xterm and gnome-terminal
	fb.DS.ApplicationModeCursorKeys = false
	fb.DS.SetScrollingRegion(0, fb.DS.GetHeight()-1)
	fb.DS.AddRenditions()
	fb.DS.ClearSavedCursor()
}
*/
func (fb *Framebuffer) setTitleInitialized()          { fb.titleInitialized = true }
func (fb *Framebuffer) isTitleInitialized() bool      { return fb.titleInitialized }
func (fb *Framebuffer) setIconLabel(iconLabel string) { fb.iconLabel = iconLabel }
func (fb *Framebuffer) setWindowTitle(title string)   { fb.windowTitle = title }
func (fb *Framebuffer) getIconLabel() string          { return fb.iconLabel }
func (fb *Framebuffer) getWindowTitle() string        { return fb.windowTitle }
func (fb *Framebuffer) resetTitle() {
	fb.windowTitle = ""
	fb.iconLabel = ""
	fb.titleInitialized = false
}

func (fb *Framebuffer) prefixWindowTitle(s string) {
	if fb.iconLabel == fb.windowTitle {
		/* preserve equivalence */
		fb.iconLabel = s + fb.iconLabel
	}
	fb.windowTitle = s + fb.windowTitle
}

/*
func (fb *Framebuffer) Resize(width, height int) bool {
	if width <= 0 || height <= 0 {
		return false
	}

	oldWidth := fb.DS.GetWidth()
	oldHeight := fb.DS.GetHeight()
	fb.DS.Resize(width, height)

	if oldHeight != height {
		// adjust the rows
		fb.resizeRows(width, height)
	}

	if oldWidth == width {
		return true
	}

	// adjust the width
	fb.resizeCols(width, oldWidth)
	return true
}

func (fb *Framebuffer) resizeRows(width, height int) {
	count := height - len(fb.rows)
	if count < 0 {
		// quick abs
		count = -count

		// shrink the rows
		fb.rows = fb.rows[:len(fb.rows)-count]
	} else {
		// need to expand the addRows
		addRows := make([]Row, count)
		for i := range addRows {
			addRows[i] = *fb.newRow()
		}
		fb.rows = append(fb.rows, addRows[:]...)
	}
}

func (fb *Framebuffer) resizeCols(width, oldWidth int) {
	count := width - oldWidth
	if count < 0 {
		// shrink
		for i := range fb.rows {
			// shrink the columns
			fb.rows[i].cells = fb.rows[i].cells[:width]
		}
	} else {
		// expand
		for i := range fb.rows {
			// already reach the new width
			if width == len(fb.rows[i].cells) {
				continue
			}

			// expand the addCells
			addCells := make([]Cell, count)
			for i := range addCells {
				addCells[i].SetRenditions(Renditions{bgColor: 0})
			}
			fb.rows[i].cells = append(fb.rows[i].cells, addCells[:]...)
		}
	}
}
*/
// func (fb *Framebuffer) ResetCell(c *Cell) { c.Reset(uint32(fb.DS.GetBackgroundRendition())) }
// func (fb *Framebuffer) ResetRow(r *Row)   { r.Reset(uint32(fb.DS.GetBackgroundRendition())) }
func (fb *Framebuffer) ringBell()         { fb.bellCount += 1 }
func (fb *Framebuffer) getBellCount() int { return fb.bellCount }
func (fb *Framebuffer) resetBell()        { fb.bellCount = 0 }

func cycleSelectSnapTo2(snapTo SelectSnapTo) SelectSnapTo {
	return SelectSnapTo((int(snapTo) + 1) % int(SelectSnapTo_COUNT))
}

// func (fb Framebuffer) Equal(other *Framebuffer) (ret bool) {
// 	// check title and bell count
// 	if fb.windowTitle != other.windowTitle || fb.bellCount != other.bellCount {
// 		return ret
// 	}
//
// 	// check DrawState
// 	if !fb.DS.Equal(other.DS) {
// 		return ret
// 	}
//
// 	// check the rows first
// 	for i := range fb.rows {
// 		if !fb.rows[i].Equal(&other.rows[i]) {
// 			return ret
// 		}
// 	}
//
// 	ret = true
// 	return ret
// }
