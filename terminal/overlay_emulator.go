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

/*
-----------------------------------------------------------------------------------------------------
The following methods is only used by prediction engine. The coordinate is different from the one used
by control sequence. It use the Emulator internal coordinate, starts from 0.
-----------------------------------------------------------------------------------------------------
*/

// move cursor to specified position, (default screen coordinate = [1,1])
func (emu *Emulator) MoveCursor(posY, posX int) {
	emu.posX = posX
	emu.posY = posY
	emu.normalizeCursorPos()
}

// get current cursor column
func (emu *Emulator) GetCursorCol() int {
	return emu.posX
}

// get current cursor row
func (emu *Emulator) GetCursorRow() int {
	if emu.originMode == OriginMode_Absolute {
		return emu.posY
	}
	return emu.posY - emu.marginTop
}

// get active area height
func (emu *Emulator) GetHeight() int {
	return emu.marginBottom - emu.marginTop
}

// get active area width
func (emu *Emulator) GetWidth() int {
	if emu.horizMarginMode {
		return emu.nColsEff - emu.hMargin
	}

	return emu.nCols
}

func (emu *Emulator) GetCell(posY, posX int) Cell {
	posY, posX = emu.getCellPos(posY, posX)

	return emu.cf.getCell(posY, posX)
}

func (emu *Emulator) GetMutableCell(posY, posX int) *Cell {
	posY, posX = emu.getCellPos(posY, posX)

	return emu.cf.getMutableCell(posY, posX)
}

func (emu *Emulator) getCellPos(posY, posX int) (posY2, posX2 int) {
	// fmt.Printf("@1 (%d,%d)\n", posY, posX)
	// in case we don't provide the row or col
	if posY < 0 || posY > emu.GetHeight() {
		posY = emu.GetCursorRow()
		// fmt.Printf("@2 (%d,%d)\n", posY, posX)
	}

	if posX < 0 || posX > emu.GetWidth() {
		posX = emu.GetCursorCol()
		// fmt.Printf("@3 (%d,%d)\n", posY, posX)
	}

	switch emu.originMode {
	case OriginMode_Absolute:
		posY = Max(0, Min(posY, emu.nRows))
		// fmt.Printf("@4 (%d,%d)\n", posY, posX)
	case OriginMode_ScrollingRegion:
		posY = Max(0, Min(posY, emu.marginBottom))
		posY += emu.marginTop
		// fmt.Printf("@5 (%d,%d)\n", posY, posX)
	}
	posX = Max(0, Min(posX, emu.nCols))
	// fmt.Printf("@6 (%d,%d)\n", posY, posX)

	posX2 = posX
	posY2 = posY
	return
}

func (emu *Emulator) GetRenditions() (rnd Renditions) {
	return emu.attrs.renditions
}

func (emu *Emulator) PrefixWindowTitle(prefix string) {
	emu.cf.PrefixWindowTitle(prefix)
}

func (emu *Emulator) GetWindowTitle() string {
	return emu.cf.GetWindowTitle()
}

func (emu *Emulator) GetIconName() string {
	return emu.cf.GetIconName()
}

func (emu *Emulator) SetCursorVisible(visible bool) {
	if !visible {
		emu.cf.setCursorStyle(CursorStyle_Hidden)
	} else {
		emu.cf.setCursorStyle(CursorStyle_FillBlock)
		// TODO keep the old style?
	}
}
