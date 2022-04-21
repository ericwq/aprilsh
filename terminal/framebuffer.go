package terminal

type Framebuffer struct {
	rows             []Row
	iconName         string
	windowTitle      string
	bellCount        int
	titleInitialized bool
	DS               DrawState
}

func NewFramebuffer(width, height int) *Framebuffer {
	if width <= 0 || height <= 0 {
		return nil
	}

	fb := Framebuffer{}
	fb.DS = *NewDrawState(width, height)
	fb.rows = make([]Row, height)
	for i := range fb.rows {
		fb.rows[i] = *NewRow(width, 0)
	}

	return &fb
}

func (fb *Framebuffer) newRow() *Row {
	w := fb.DS.GetWidth()
	bgColor := fb.DS.GetBackgroundRendition()
	return NewRow(w, uint32(bgColor))
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
	} else if fb.DS.GetCursorRow()+rows < fb.DS.GetScrollingRegionTopRow() {
		N := fb.DS.GetCursorRow() + rows - fb.DS.GetScrollingRegionTopRow()
		fb.Scroll(N)
	}

	fb.DS.MoveRow(rows, true)
}

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
	cell.SetRenditions(fb.DS.GetRenditions())
}

func (fb *Framebuffer) InsertLine(beforeRow int, count int) bool { // #BehaviorChange return bool
	// invalide beforeRow
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
	// invalide row
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

func (fb *Framebuffer) InsertCell(row, col int) {
	fb.GetRow(row).InsertCell(col, uint32(fb.DS.GetBackgroundRendition()))
}

func (fb *Framebuffer) DeleteCell(row, col int) {
	fb.GetRow(row).DeleteCell(col, uint32(fb.DS.GetBackgroundRendition()))
}

func (fb *Framebuffer) Reset() {
	width := fb.DS.GetWidth()
	height := fb.DS.GetHeight()

	fb.DS = *NewDrawState(width, height)

	fb.rows = make([]Row, height)
	for i := range fb.rows {
		fb.rows[i] = *NewRow(width, 0)
	}
	fb.windowTitle = ""
	/* do not reset bell_count */
}

func (fb *Framebuffer) SoftReset() {
	fb.DS.InsertMode = false
	fb.DS.OriginMode = false
	fb.DS.CursorVisible = false
	fb.DS.ApplicationModeCursorKeys = false
	fb.DS.SetScrollingRegion(0, fb.DS.GetHeight()-1)
	fb.DS.AddRenditions(0)
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

func (fb *Framebuffer) Resize(width, height int) {
	if width <= 0 || height <= 0 {
		panic("Framebuffer.Resize(), width or height is negative.")
	}

	oldWidth := fb.DS.GetWidth()
	oldHeight := fb.DS.GetHeight()
	fb.DS.Resize(width, height)

	if oldHeight != height {
		// adjust the rows
		fb.resizeRows(width, height)
	}

	if oldWidth == width {
		return
	}

	// adjust the width
	fb.resizeCols(width, oldWidth)
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
			addRows[i] = *NewRow(width, 0)
		}
		fb.rows = append(fb.rows, addRows[:]...)
	}
}

func (fb *Framebuffer) resizeCols(width, oldWidth int) {
	count := width - oldWidth
	if count < 0 {
		// shrink
		for i := range fb.rows {
			// already reach the new width
			if width == len(fb.rows[i].cells) {
				continue
			}
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
	// check the rows first
	for i := range fb.rows {
		if !fb.rows[i].Equal(&other.rows[i]) {
			return ret
		}
	}

	if fb.windowTitle != other.windowTitle || fb.bellCount != other.bellCount {
		return ret
	}
	if !fb.DS.Equal(&other.DS) {
		return ret
	}

	ret = true
	return ret
}
