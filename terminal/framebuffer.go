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

package terminal

type Framebuffer struct {
	rows             []Row
	iconName         string
	windowTitle      string
	bellCount        int
	titleInitialized bool
	DS               *DrawState
	selectionData    string

	// support both (scrollable) normal screen buffer and alternate screen buffer
	cells        []Cell // the cells
	nCols        int    // cols number per window
	nRows        int    // rows number per window
	saveLines    int    // nRows + saveLines is the scrolling area limitation
	scrollHead   int    // row offset of scrolling area's logical top row
	marginTop    int    // current margin top (number of rows above)
	marginBottom int    // current margin bottom (number of rows above + 1)
	historyRows  int    // number of history (off-screen) rows with data
	viewOffset   int    // how many rows above top row does the view start?
	margin       bool   // are there (non-default) top/bottom margins set?
	selection    Rect   // selection area
	damage       Damage // damage scope
	cursor       Cursor // current cursor style, color and position
}

func NewFramebuffer(width, height int) *Framebuffer {
	if width <= 0 || height <= 0 {
		return nil
	}

	fb := Framebuffer{}
	fb.DS = NewDrawState(width, height)
	fb.rows = make([]Row, height)
	for i := range fb.rows {
		fb.rows[i] = *NewRow(width, 0)
	}

	return &fb
}

// saveLines: for alternate screen buffer default is 0, for normal screen buffer the default is 500, max 50000
func NewFramebuffer3(nCols, nRows, saveLines int) *Framebuffer {
	fb := Framebuffer{}

	fb.DS = NewDrawState(nCols, nRows)

	fb.cells = make([]Cell, nCols*(nRows+saveLines))
	fb.nCols = nCols
	fb.nRows = nRows

	fb.saveLines = saveLines
	fb.scrollHead = 0
	fb.marginTop = 0
	fb.marginBottom = nRows + saveLines
	fb.historyRows = 0
	fb.viewOffset = 0
	fb.margin = false

	fb.damage.totalCells = nCols * (nRows + saveLines)
	fb.selection = *NewRect()

	return &fb
}

func (fb *Framebuffer) expose() {
	fb.damage.expose()
}

func (fb *Framebuffer) resetDamage() {
	fb.damage.reset()
}

func (fb *Framebuffer) getHistroryRows() int {
	return fb.historyRows
}

func (fb *Framebuffer) invalidateSelection(damage *Rect) {
	if fb.selection.empty() {
		return
	}

	if fb.selection.br.lessEqual(damage.tl) || damage.br.lessEqual(fb.selection.tl) {
		return
	}

	fb.selection.clear()
}

// text up, screen down count rows
func (fb *Framebuffer) pageUp(count int) {
	viewOffset := min(fb.viewOffset+count, fb.historyRows)
	delta := viewOffset - fb.viewOffset
	fb.cursor.posY += delta
	fb.selection.br.y += delta
	fb.selection.tl.y += delta
	fb.viewOffset = viewOffset
	fb.expose()
}

// text down, screen up count rows
func (fb *Framebuffer) pageDown(count int) {
	viewOffset := max(0, fb.viewOffset-count)
	delta := viewOffset - fb.viewOffset
	fb.cursor.posY += delta
	fb.selection.br.y += delta
	fb.selection.tl.y += delta
	fb.viewOffset = viewOffset
	fb.expose()
}

// text down, screen up to the top row
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

// insert blank cols at and to the right of startX, within the scrolling area
func (fb *Framebuffer) insertCols(startX, count int) {
	for r := fb.marginTop; r < fb.marginBottom; r++ {
		fb.moveInRow(r, startX+count, startX, fb.DS.nColsEff-startX-count)
		fb.eraseInRow(r, startX, count, fb.DS.renditions) // use the default renditions
	}
}

// delete cols at and to the right of startX, within the scrolling area
func (fb *Framebuffer) deleteCols(startX, count int) {
	for r := fb.marginTop; r < fb.marginBottom; r++ {
		fb.moveInRow(r, startX, startX+count, fb.DS.nColsEff-startX-count)
		fb.eraseInRow(r, fb.DS.nColsEff-count, count, fb.DS.renditions) // use the default renditions
	}
}

// erase (reset) fromt start to end with specified renditions
func (fb *Framebuffer) eraseRange(start, end int, rend Renditions) {
	for i := range fb.cells[start:end] {
		fb.cells[start+i].Reset2(rend)
	}
	fb.damage.add(start, end)
}

// erase (count) cells from startX column
func (fb *Framebuffer) eraseInRow(pY, startX, count int, rend Renditions) {
	if count == 0 {
		return
	}

	idx := fb.getIdx(pY, startX)
	fb.eraseRange(idx, idx+count, rend)
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

// move (count) cells from srcIx to dstIx. Both of the parameters are the index for cells.
func (fb *Framebuffer) moveCells(dstIx, srcIx, count int) {
	copy(fb.cells[dstIx:], fb.cells[srcIx:srcIx+count])
	fb.damage.add(dstIx, dstIx+count)
}

func (fb *Framebuffer) getIdx(pY, pX int) int {
	return fb.nCols*fb.getPhysicalRow(pY-fb.viewOffset) + pX
}

// TODO the relationship between viewOffset and scrollHead
func (fb *Framebuffer) getPhysicalRow(pY int) int {
	if pY < 0 {
		if !fb.margin {
			pY += fb.scrollHead
		}
		if pY < 0 {
			pY += fb.nRows + fb.saveLines
		}

		return pY
	}

	if fb.margin && (pY < fb.marginTop || pY >= fb.marginBottom) {
		return pY
	}

	pY += fb.scrollHead - fb.marginTop
	if pY >= fb.marginBottom {
		pY -= fb.marginBottom - fb.marginTop
	}

	return pY
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

// copy (count) cells from srcIx to dstIx. Both of the parameters are the index for cells.
func (fb *Framebuffer) copyCells(dstIx, srcIx, count int) {
	copy(fb.cells[dstIx:], fb.cells[srcIx:srcIx+count])
	fb.damage.add(dstIx, dstIx+count)
}

// insert blank rows at and below startY, within the scrolling area
func (fb *Framebuffer) insertRows(startY, count int) {
	for pY := fb.marginBottom - count - 1; pY >= startY; pY-- {
		fb.copyRow(pY+count, pY, fb.DS.hMargin, fb.DS.nColsEff-fb.DS.hMargin)
		if pY == 0 {
			break
		}
	}

	for pY := startY; pY < startY+count; pY++ {
		// use the default renditions
		fb.eraseRow(pY, fb.DS.hMargin, fb.DS.nColsEff-fb.DS.hMargin, fb.DS.renditions)
	}
}

// delete rows at and below startY, within the scrolling area
func (fb *Framebuffer) deleteRows(startY, count int) {
	// move the rest (startY+count) rows up
	for pY := startY; pY < fb.marginBottom-count; pY++ {
		fb.copyRow(pY, pY+count, fb.DS.hMargin, fb.DS.nColsEff-fb.DS.hMargin)
	}

	// erase the remain rows
	for pY := fb.marginBottom - count; pY < fb.marginBottom; pY++ {
		// use the default renditions
		fb.eraseRow(pY, fb.DS.hMargin, fb.DS.nColsEff-fb.DS.hMargin, fb.DS.renditions)
	}
}

// erase rows at and below startY, within the scrolling area
func (fb *Framebuffer) eraseRows(startY, count int) {
	for pY := startY; pY < startY+count; pY++ {
		// use the default renditions
		fb.eraseRow(pY, fb.DS.hMargin, fb.DS.nColsEff-fb.DS.hMargin, fb.DS.renditions)
	}
}

// erase a row at pY, within the left-right scrolling area. (startX,count) defines the area.
func (fb *Framebuffer) eraseRow(pY, startX, count int, rend Renditions) {
	if count == 0 {
		return
	}

	idx := fb.getIdx(pY, startX)
	fb.eraseRange(idx, idx+count, rend)
	fb.invalidateSelection(NewRect4(startX, pY, startX+count, pY))
}

func (fb *Framebuffer) freeCells() {
	fb.cells = nil
}

// fill current screen with specified rune and renditions.
func (fb *Framebuffer) fillCells(ch rune, rend Renditions) {
	for r := 0; r < fb.nRows; r++ {

		start := fb.getIdx(r, 0)
		end := start + fb.nCols
		for k := start; k < end; k++ {
			fb.cells[k].renditions = rend
			fb.cells[k].contents = string(ch)
		}
		fb.damage.add(start, end)
	}
}

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
	/* don't scroll if outside the scrolling region */
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
	/* do not reset bell_count */
}

func (fb *Framebuffer) SoftReset() {
	fb.DS.InsertMode = false
	fb.DS.OriginMode = false
	fb.DS.CursorVisible = false /* per xterm and gnome-terminal */
	fb.DS.ApplicationModeCursorKeys = false
	fb.DS.SetScrollingRegion(0, fb.DS.GetHeight()-1)
	fb.DS.AddRenditions()
	fb.DS.ClearSavedCursor()
}

func (fb *Framebuffer) SetTitleInitialized()        { fb.titleInitialized = true }
func (fb Framebuffer) IsTitleInitialized() bool     { return fb.titleInitialized }
func (fb *Framebuffer) SetIconName(iconName string) { fb.iconName = iconName }
func (fb *Framebuffer) SetWindowTitle(title string) { fb.windowTitle = title }
func (fb Framebuffer) GetIconName() string          { return fb.iconName }
func (fb Framebuffer) GetWindowTitle() string       { return fb.windowTitle }

func (fb *Framebuffer) PrefixWindowTitle(s string) {
	if fb.iconName == fb.windowTitle {
		/* preserve equivalence */
		fb.iconName = s + fb.iconName
	}
	fb.windowTitle = s + fb.windowTitle
}

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

func (fb *Framebuffer) ResetCell(c *Cell) { c.Reset(uint32(fb.DS.GetBackgroundRendition())) }
func (fb *Framebuffer) ResetRow(r *Row)   { r.Reset(uint32(fb.DS.GetBackgroundRendition())) }
func (fb *Framebuffer) RingBell()         { fb.bellCount += 1 }
func (fb Framebuffer) GetBellCount() int  { return fb.bellCount }

func (fb Framebuffer) Equal(other *Framebuffer) (ret bool) {
	// check title and bell count
	if fb.windowTitle != other.windowTitle || fb.bellCount != other.bellCount {
		return ret
	}

	// check DrawState
	if !fb.DS.Equal(other.DS) {
		return ret
	}

	// check the rows first
	for i := range fb.rows {
		if !fb.rows[i].Equal(&other.rows[i]) {
			return ret
		}
	}

	ret = true
	return ret
}
