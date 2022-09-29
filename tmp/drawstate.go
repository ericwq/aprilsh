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

type SavedCursor struct {
	cursorCol, cursorRow int
	renditions           Renditions
	autoWrapMode         bool // default value true
	originMode           bool
}

type DrawState struct {
	width            int
	height           int
	cursorCol        int
	cursorRow        int
	cursorColor      Color
	combiningCharCol int
	combiningCharRow int

	defaultTabs bool
	tabs        []bool

	scrollingRegionTopRow    int
	scrollingRegionBottomRow int
	renditions               Renditions
	save                     SavedCursor

	// public fields
	NextPrintWillWrap bool

	// DEC private mode
	OriginMode                bool // two possiible value: ScrollingRegion(true), Absolute(false)
	AutoWrapMode              bool // true/false
	CursorVisible             bool // true/false
	ReverseVideo              bool // two possible value: Reverse(true), Normal(false)
	BracketedPaste            bool // true/false
	MouseReportingMode        int  // replace it with MouseTrackingMode
	MouseFocusEvent           bool // replace it with MouseTrackingState.focusEventMode
	MouseAlternateScroll      bool // rename to altScrollMode
	MouseEncodingMode         int  // replace it with MouseTrackingEnc
	ApplicationModeCursorKeys bool // =cursorKeyMode two possible value : Application(true), ANSI(false)
	mouseTrk                  MouseTrackingState
	altSendsEscape            bool

	// ANSI mode
	keyboardLocked  bool
	InsertMode      bool // true/false
	localEcho       bool
	autoNewlineMode bool

	// added for vt400 compatibility
	compatLevel         CompatibilityLevel // VT52, VT100, VT400
	altScreenBufferMode bool               // Alternate Screen Buffer support: default false
	columnMode          ColMode            // column mode 80 or 132, just for compatibility
	horizMarginMode     bool               // left and right margins support
	hMargin             int                // left margins
	nColsEff            int                // right margins
	bkspSendsDel        bool               // backspace send delete

	savedCursorSCO SavedCursorSCO // SCO console cursor state
}

type (
	MouseTrackingMode  uint
	MouseTrackingEnc   uint
	CompatibilityLevel uint
	CursorKeyMode      uint
	KeypadMode         uint
	ColMode            uint
	OriginMode         uint
	SelectSnapTo       uint
)

const (
	CursorKeyMode_ANSI CursorKeyMode = iota
	CursorKeyMode_Application
)

const (
	MouseTrackingMode_Disable MouseTrackingMode = iota
	MouseTrackingMode_X10_Compat
	MouseTrackingMode_VT200
	MouseTrackingMode_VT200_ButtonEvent
	MouseTrackingMode_VT200_AnyEvent
)

const (
	MouseTrackingEnc_Default MouseTrackingEnc = iota
	MouseTrackingEnc_UTF8
	MouseTrackingEnc_SGR
	MouseTrackingEnc_URXVT
)

const (
	CompatLevel_Unused CompatibilityLevel = iota
	CompatLevel_VT52
	CompatLevel_VT100
	CompatLevel_VT400
)

const (
	KeypadMode_Normal KeypadMode = iota
	KeypadMode_Application
)

const (
	ColMode_C80 ColMode = iota
	ColMode_C132
)

const (
	OriginMode_Absolute OriginMode = iota
	OriginMode_ScrollingRegion
)

const (
	SelectSnapTo_Char SelectSnapTo = iota
	SelectSnapTo_Word
	SelectSnapTo_Line
	SelectSnapTo_COUNT
)

// TODO replace the following const with the above one
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

// TODO default constructor checking
type MouseTrackingState struct {
	mode           MouseTrackingMode
	enc            MouseTrackingEnc
	focusEventMode bool
}

type SavedCursorSCO struct {
	col     int
	row     int
	isSet   bool
	lastCol bool
}

type SavedCursor_SCO struct {
	isSet   bool
	posX    int
	posY    int
	lastCol bool
}

// TODO refine the constructor
type SavedCursor_DEC struct {
	SavedCursor_SCO
	attrs        Cell
	originMode   OriginMode
	charsetState CharsetState
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
	ds.cursorColor = ColorWhite

	ds.MouseReportingMode = MOUSE_REPORTING_NONE
	ds.MouseEncodingMode = MOUSE_ENCODING_DEFAULT

	ds.columnMode = ColMode_C80
	ds.bkspSendsDel = true

	ds.reinitializeTabs(0)
	return ds
}

func (ds *DrawState) reinitializeTabs(start uint) {
	for i := start; i < uint(len(ds.tabs)); i++ {
		ds.tabs[i] = (i % 8) == 0 // TODO : tab size adjustable?
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

// Default tabs can't be restored without resetting the draw state.

func (ds DrawState) GetNextTab(count int) int {
	if count >= 0 {
		for i := ds.cursorCol + 1; i < ds.width; i++ {
			if ds.tabs[i] { // find one next tab stop
				count -= 1 // finish one tab stop
				if count == 0 {
					return i
				}
			}
		}
		return -1
	} else {
		for i := ds.cursorCol - 1; i > 0; i-- {
			if ds.tabs[i] { // find one previous tab stop
				count += 1 // finish one tab stop
				if count == 0 {
					return i
				}
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
	// real rule requires TWO-line scrolling region

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

// set index color for foreground
func (ds *DrawState) SetForegroundColor(index int) { ds.renditions.SetForegroundColor(index) }

// set index color for background
func (ds *DrawState) SetBackgroundColor(index int) { ds.renditions.SetBackgroundColor(index) }

// TODO change the parameter of AddRenditions() from uint32 to none
func (ds *DrawState) AddRenditions()               { ds.renditions = Renditions{} }
func (ds *DrawState) GetRenditions() *Renditions   { return &ds.renditions }
func (ds DrawState) GetBackgroundRendition() Color { return ds.renditions.bgColor }

func (ds *DrawState) SaveCursor() {
	ds.save.cursorCol = ds.cursorCol
	ds.save.cursorRow = ds.cursorRow
	ds.save.renditions = ds.renditions
	ds.save.autoWrapMode = ds.AutoWrapMode
	ds.save.originMode = ds.OriginMode
}

func (ds *DrawState) RestoreCursor() {
	ds.cursorCol = ds.save.cursorCol
	ds.cursorRow = ds.save.cursorRow
	ds.renditions = ds.save.renditions
	ds.AutoWrapMode = ds.save.autoWrapMode
	ds.OriginMode = ds.save.originMode

	ds.snapCursorToBorder()
	ds.newGrapheme()
}

func (ds *DrawState) ClearSavedCursor() { ds.save = SavedCursor{autoWrapMode: true} }

func (ds *DrawState) Resize(width, height int) {
	if ds.width != width || ds.height != height {
		// reset entire scrolling region on any resize
		// xterm and rxvt-unicode do this. gnome-terminal only
		// resets scrolling region if it has to become smaller in resize
		ds.scrollingRegionTopRow = 0
		ds.scrollingRegionBottomRow = height - 1
	}

	// TODO : we initialize the tabs slice from the very beginning?
	// if something wired happened, please consider to modify it.
	ds.tabs = make([]bool, width)
	if ds.defaultTabs {
		ds.reinitializeTabs(0)
	}

	ds.width = width
	ds.height = height

	ds.snapCursorToBorder()

	// saved cursor will be snapped to border on restore

	// invalidate combining char cell if necessary
	if ds.combiningCharCol >= width || ds.combiningCharRow >= height {
		ds.combiningCharCol = -1
		ds.combiningCharRow = -1
	}
}

// use pointer parameter to avoid struct copy
func (ds DrawState) Equal(x *DrawState) bool {
	// only compare fields that affect display
	return ds.width == x.width && ds.height == x.height &&
		ds.cursorCol == x.cursorCol && ds.cursorRow == x.cursorRow &&
		ds.CursorVisible == x.CursorVisible && ds.ReverseVideo == x.ReverseVideo &&
		ds.renditions == x.renditions &&
		ds.BracketedPaste == x.BracketedPaste &&
		ds.MouseReportingMode == x.MouseReportingMode &&
		ds.MouseFocusEvent == x.MouseFocusEvent &&
		ds.MouseAlternateScroll == x.MouseAlternateScroll &&
		ds.MouseEncodingMode == x.MouseEncodingMode
}
