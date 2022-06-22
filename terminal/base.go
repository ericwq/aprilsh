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

type Point struct {
	x, y int
}

type Rect struct {
	tl          Point // top left corner
	br          Point // bottom right corner
	rectangular bool
}

func NewRect() (rect *Rect) {
	rect.tl = Point{-1, -1}
	rect.br = Point{-1, -1}
	return rect
}

// empty rectangular
func (rect *Rect) empty() bool {
	return rect.tl == rect.br
}

// null rectangular
func (rect *Rect) null() bool {
	raw := Point{-1, -1}
	return rect.tl == raw && rect.br == raw
}

// return the middle point of rectangular
func (rect *Rect) mid() Point {
	return Point{(rect.tl.x + rect.br.x) / 2, (rect.tl.y + rect.br.y) / 2}
}

func (rect *Rect) clear() {
	rect.tl = Point{-1, -1}
	rect.br = Point{-1, -1}
}

func (rect *Rect) toggleRectangular() {
	rect.rectangular = !rect.rectangular
}

type Damage struct {
	start      int
	end        int
	totalCells int
}

func (dmg *Damage) reset() {
	dmg.start = 0
	dmg.end = 0
}

func (dmg *Damage) expose() {
	dmg.start = 0
	dmg.end = dmg.totalCells
}

func (dmg *Damage) add(start, end int) {
	if end < start {
		start = 0
		end = dmg.totalCells
	}

	if start == end {
		dmg.start = start
		dmg.end = end
	} else {
		dmg.start = min(dmg.start, start)
		dmg.end = max(dmg.end, end)
	}
}

func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

func max(x, y int) int {
	if x > y {
		return x
	}
	return y
}
