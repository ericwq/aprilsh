package terminal

var gen_counter uint64 = 0

type Row struct {
	cells []Cell
	// gen is a generation counter.  It can be used to quickly rule
	// out the possibility of two rows being identical; this is useful
	// in scrolling.
	gen uint64
}

func getGen() uint64 {
	gen_counter += 1
	return gen_counter
}

func NewRow(width int, bgColor uint32) *Row {
	r := Row{}
	r.cells = make([]Cell, width)
	for i := range r.cells {
		r.cells[i].SetRenditions(Renditions{bgColor: bgColor})
		// fmt.Printf("NeRow: set cell %v %d\n", c.GetRenditions(), bgColor)
	}
	r.gen = getGen()
	// fmt.Printf("NewRow: %v\n", r.cells)
	return &r
}

func (r *Row) InsertCell(col int, bgColor uint32) bool {
	// validate the column range
	if col < 0 || col > len(r.cells)-1 {
		return false
	}

	// prepare the new cell
	cell := Cell{}
	cell.renditions = Renditions{bgColor: bgColor}

	// insert cell
	r.cells = append(r.cells[:col+1], r.cells[col:]...)
	r.cells[col] = cell

	// pop the last one
	width := len(r.cells) - 1
	r.cells = r.cells[:width]
	return true
}

func (r *Row) DeleteCell(col int, bgColor uint32) bool {
	if col < 0 || col > len(r.cells)-1 {
		return false
	}

	// prepare the new cell
	cell := Cell{}
	cell.renditions = Renditions{bgColor: bgColor}

	// add new cell at the end
	r.cells = append(r.cells, cell)

	// delete cell at col
	copy(r.cells[col:], r.cells[col+1:])

	// remvoe the last one
	width := len(r.cells) - 1
	r.cells = r.cells[:width]
	return true
}

func (r *Row) Reset(bgColor uint32) {
	r.gen = getGen()
	for i := range r.cells {
		r.cells[i].Reset(bgColor)
	}
}

func (r Row) GetWrap() bool {
	return r.cells[len(r.cells)-1].GetWrap()
}

func (r *Row) SetWrap(w bool) {
	r.cells[len(r.cells)-1].SetWrap(w)
}

func (r Row) Equal(other *Row) bool {
	// the easy way to compare
	if r.gen != other.gen {
		return false
	}

	// has different size?
	if len(r.cells) != len(other.cells) {
		return false
	}

	// check the content
	for i := range r.cells {
		if r.cells[i] != other.cells[i] {
			return false
		}
	}
	return true
}

type SavedCursor struct {
	cursorCol, cursoRow int
	renditions          Renditions
	autoWrapMode        bool // default value true
	originMode          bool
}

const (
	MOUSE_REPORTING_NONE          = 0
	MOUSE_REPORTING_X10           = 9
	MOUSE_REPORTING_VT220         = 1000
	MOUSE_REPORTING_VT220_HILIGHT = 1001
	MOUSE_REPORTING_BTN_EVENT     = 1002
	MOUSE_REPORTING_ANY_EVENT     = 1003

	MOUSE_ENCODING_DEFAULT = 0
	MOUSE_ENCODING_UTF8    = 1005
	MOUSE_ENCODING_SGR     = 1006
	MOUSE_ENCODING_URXVT   = 1015
)

type DrawState struct {
	width            int
	height           int
	cursorCol        int
	cursorRow        int
	combiningCharCol int
	combiningCharRow int

	defaultTabs bool
	tabs        []bool

	scrollingRegionTopRow    int
	scrollingRegionBottomRow int
	renditions               Renditions
	save                     SavedCursor

	// public fields
	NextPrintWillWrap         bool
	OriginMode                bool
	AutoWrapMode              bool
	InsertMode                bool
	CursorVisible             bool
	ReverseVideo              bool
	BracketedPaste            bool
	MouseReportingMode        int
	MouseFocusEvent           bool
	MouseAlternateScroll      bool
	MouseEncodingMode         int
	ApplicationModeCursorKeys bool
}

func NewDrawState(width, height int) *DrawState {
	ds := new(DrawState)

	ds.width = width
	ds.height = height
	ds.defaultTabs = true
	ds.tabs = make([]bool, width)
	ds.scrollingRegionBottomRow = height - 1
	ds.renditions = Renditions{bgColor: 0}
	ds.save = SavedCursor{autoWrapMode: true}
	ds.AutoWrapMode = true
	ds.CursorVisible = true
	ds.MouseReportingMode = MOUSE_REPORTING_NONE
	ds.MouseEncodingMode = MOUSE_ENCODING_DEFAULT

	ds.reinitializeTabs(0)
	return ds
}

func (ds *DrawState) reinitializeTabs(start uint) {
	for i := start; i < uint(len(ds.tabs)); i++ {
		ds.tabs[i] = (i % 8) == 0
	}
}

// set the combining col,row position
func (ds *DrawState) newGrapheme() {
	ds.combiningCharCol = ds.cursorCol
	ds.combiningCharRow = ds.cursorRow
}

func (ds *DrawState) snapCursorToBorder() {
	if ds.cursorRow < ds.limitTop() {
		ds.cursorRow = ds.limitTop()
	}
	if ds.cursorRow > ds.LimitBottom() {
		ds.cursorRow = ds.LimitBottom()
	}
	if ds.cursorCol < 0 {
		ds.cursorCol = 0
	}
	if ds.cursorCol >= ds.width {
		ds.cursorCol = ds.width - 1
	}
}

func (ds *DrawState) MoveRow(N int, relative bool) {
	if relative {
		ds.cursorRow += N
	} else {
		ds.cursorRow = N + ds.limitTop()
	}

	ds.snapCursorToBorder()
	ds.newGrapheme()
	ds.NextPrintWillWrap = false
}

func (ds *DrawState) MoveCol(N int, relative bool, implicit bool) {
	if implicit {
		ds.newGrapheme()
	}

	if relative {
		ds.cursorCol += N
	} else {
		ds.cursorCol = N
	}

	if implicit {
		ds.NextPrintWillWrap = ds.cursorCol >= ds.width
	}

	ds.snapCursorToBorder()
	if !implicit {
		ds.newGrapheme()
		ds.NextPrintWillWrap = false
	}
}

func (ds DrawState) GetCursorCol() int        { return ds.cursorCol }
func (ds DrawState) GetCursorRow() int        { return ds.cursorRow }
func (ds DrawState) GetCombiningCharCol() int { return ds.combiningCharCol }
func (ds DrawState) GetCombiningCharRow() int { return ds.combiningCharRow }
func (ds DrawState) GetWidth() int            { return ds.width }
func (ds DrawState) GetHeight() int           { return ds.height }

func (ds *DrawState) SetTab()           { ds.tabs[ds.cursorCol] = true }
func (ds *DrawState) ClearTab(col int)  { ds.tabs[col] = false }
func (ds *DrawState) ClearDefaultTabs() { ds.defaultTabs = false }

/* Default tabs can't be restored without resetting the draw state. */

func (ds DrawState) GetNextTab(count int) int {
	if count >= 0 {
		for i := ds.cursorCol + 1; i < ds.width; i++ {
			count -= 1
			if ds.tabs[i] && count == 0 {
				return i
			}
		}
		return -1
	} else {
		for i := ds.cursorCol - 1; i > 0; i-- {
			count += 1
			if ds.tabs[i] && count == 0 {
				return i
			}
		}
		return 0
	}
}

func (ds *DrawState) SetScrollingRegion(top, bottom int) {
	if ds.height < 1 {
		return
	}
	ds.scrollingRegionTopRow = top
	ds.scrollingRegionBottomRow = bottom

	if ds.scrollingRegionTopRow < 0 {
		ds.scrollingRegionTopRow = 0
	}
	if ds.scrollingRegionBottomRow >= ds.height {
		ds.scrollingRegionBottomRow = ds.height - 1
	}

	if ds.scrollingRegionBottomRow < ds.scrollingRegionTopRow {
		ds.scrollingRegionBottomRow = ds.scrollingRegionTopRow
	}
	/* real rule requires TWO-line scrolling region */

	if ds.OriginMode {
		ds.snapCursorToBorder()
		ds.newGrapheme()
	}
}

func (ds DrawState) GetScrollingRegionTopRow() int    { return ds.scrollingRegionTopRow }
func (ds DrawState) GetScrollingRegionBottomRow() int { return ds.scrollingRegionBottomRow }

func (ds DrawState) limitTop() int {
	if ds.OriginMode {
		return ds.scrollingRegionTopRow
	}
	return 0
}

func (ds DrawState) LimitBottom() int {
	if ds.OriginMode {
		return ds.scrollingRegionBottomRow
	}
	return ds.height - 1
}
