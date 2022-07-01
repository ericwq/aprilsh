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

import "fmt"

type Point struct {
	x, y int
}

func (p Point) String() string {
	return fmt.Sprintf("(%d,%d)", p.x, p.y)
}

// Point(this) <= Point(rhs)
func (p Point) lessEqual(rhs Point) bool {
	return p.less(rhs) || p.equal(rhs)
}

// Point(this) < Point(rhs)
func (p Point) less(rhs Point) bool {
	return p.y < rhs.y || (p.y == rhs.y && p.x < rhs.x)
}

// Point(this) < Point(rhs)
func (p Point) equal(rhs Point) bool {
	return p.x == rhs.x && p.y == rhs.y
}

type Rect struct {
	tl          Point // top left corner
	br          Point // bottom right corner
	rectangular bool
}

func NewRect() (rect *Rect) {
	rect = &Rect{}
	rect.clear()

	return rect
}

func NewRect4(x1, y1, x2, y2 int) (rect *Rect) {
	rect = &Rect{}
	rect.tl = Point{x1, y1}
	rect.tl = Point{x2, y2}
	return rect
}

func (rect *Rect) String() string {
	return fmt.Sprintf("Rect{tl=%s br=%s rectangular=%t}", rect.tl, rect.br, rect.rectangular)
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

	if dmg.start == dmg.end {
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

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
