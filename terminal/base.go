// Copyright 2022 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package terminal

import (
	"fmt"

	"golang.org/x/exp/constraints"
)

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
	return p == rhs
	// return p.x == rhs.x && p.y == rhs.y
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
	start      int // inclusive
	end        int // exclusive
	totalCells int
}

func (dmg *Damage) count() int {
	return dmg.end - dmg.start
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
		dmg.start = Min(dmg.start, start)
		dmg.end = Max(dmg.end, end)
	}
	// fmt.Printf("Damage.add start=%d, end=%d\n", dmg.start, dmg.end)
}

func Min[T constraints.Ordered](x, y T) T {
	if x < y {
		return x
	}
	return y
}

func Max[T constraints.Ordered](x, y T) T {
	if x > y {
		return x
	}
	return y
}

func Abs[T constraints.Signed | constraints.Float](x T) T {
	if x < 0 {
		return -x
	}
	return x
}

func LowerBound(array []int, target int) int {
	low, high, mid := 0, len(array)-1, 0
	for low <= high {
		mid = (low + high) / 2
		if array[mid] >= target {
			high = mid - 1
		} else {
			low = mid + 1
		}
	}
	return low
}

func RemoveIndex(s []int, index int) []int {
	return append(s[:index], s[index+1:]...)
}
