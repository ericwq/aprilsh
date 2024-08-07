// Copyright 2022 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package terminal

import (
	"io"
	"strings"
)

/* contents saved in strings.Builder
 * Don't set the fields directly
 *
 * Cell is comparable
 */
type Cell struct {
	contents   string
	renditions Renditions
	// fallback   bool
	dirty      bool
	wrap       bool // indicate single/double width grapheme which is the last cell in the row.
	earlyWrap  bool // indicate double width grapheme which start from position nColsEff-1
	dwidth     bool // indicate this cell is the first cell of double width grapheme if true
	dwidthCont bool // indicate this cell is the second cell of double width grapheme if true
}

// func (c *Cell) Equal(x *Cell) bool {
// 	if c.contents != x.contents {
// 		return false
// 	}
//
// 	if c.dirty != x.dirty || c.wrap != x.wrap ||
// 		c.earlyWrap != x.earlyWrap || c.dwidth != x.dwidth || c.dwidthCont != x.dwidthCont {
// 		return false
// 	}
// 	return c.renditions.Equal(&x.renditions)
// }

// return true if the contents is "" or " " or non-break space(\uC2A0).
func (c Cell) IsBlank() bool {
	if c.dwidth || c.dwidthCont {
		return false
	}
	return c.Empty() || c.contents == " " || c.contents == "\xC2\xA0"
}

func (c Cell) GetRenditions() Renditions {
	return c.renditions
}

func (c Cell) String() string {
	return c.contents
}

func (c Cell) IsEarlyWrap() bool {
	return c.earlyWrap
}

func (c *Cell) SetEarlyWrap(v bool) {
	c.earlyWrap = v
}

// 32 seems like a reasonable limit on combining characters
// here we only counting the bytes number
func (c *Cell) full() bool {
	return len(c.contents) >= 32
}

// reset cell with specified renditions
// TODO : the default contents is " "?
func (c *Cell) Reset2(attrs Cell) {
	c.contents = attrs.contents
	c.renditions = attrs.renditions
	c.dwidth = false
	c.dwidthCont = false
}

// return true is the contents is "".
func (c *Cell) Empty() bool {
	return len(c.contents) == 0
}

// clear contents
func (c *Cell) Clear() {
	c.contents = ""
}

func (c *Cell) ContentsMatch(x Cell) bool {
	return (c.IsBlank() && x.IsBlank()) || c.contents == x.contents
}

// append to the contents
func (c *Cell) Append(r rune) {
	var builder strings.Builder
	if !c.Empty() {
		builder.WriteString(c.contents)
	}

	// ASCII?  Cheat.
	if r < 0x7f {
		builder.WriteByte(byte(r))
	} else {
		builder.WriteRune(r)
	}
	c.contents = builder.String()
	// set wide automaticlly?
}

// replace the contents
func (c *Cell) SetContents(chs []rune) {
	c.contents = string(chs)
}

func (c *Cell) GetContents() string {
	return c.contents
}

func (c *Cell) SetRenditions(r Renditions) {
	c.renditions = r
}

func (c *Cell) SetDoubleWidth(value bool) {
	c.dwidth = value
}

func (c *Cell) SetDoubleWidthCont(value bool) {
	c.dwidthCont = value
}

func (c *Cell) IsDoubleWidth() bool {
	return c.dwidth
}

func (c *Cell) IsDoubleWidthCont() bool {
	return c.dwidthCont
}

// print grapheme to output
func (c *Cell) printGrapheme(out io.Writer) {
	if c.Empty() {
		out.Write([]byte(" "))
		return
	}
	// * cells that begin with combining character get combiner
	// * attached to no-break space
	// if c.fallback {
	// 	output.WriteString("\xC2\xA0")
	// }
	out.Write([]byte(c.contents))
}

func (c *Cell) SetUnderline(underline bool) {
	c.renditions.underline = underline
	c.renditions.ulStyle = ULS_SINGLE
}

// return cell grapheme width: 0,1,2
func (c *Cell) GetWidth() int {
	if c.dwidthCont { // it's a place holder for wide grapheme
		return 0
	}

	if c.dwidth { // it's a wide grapheme
		return 2
	} else {
		return 1
	}
}

/*
func (c *Cell) GetFallback() bool {
	return c.fallback
}

func (c *Cell) SetFallback(f bool) {
	c.fallback = f
}

func (c *Cell) GetWrap() bool {
	return c.wrap
}

func (c *Cell) SetWrap(w bool) {
	c.wrap = w
}
*/
