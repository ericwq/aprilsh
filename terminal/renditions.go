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

import (
	"fmt"
	"strings"
)

const (
	Bold uint32 = iota
	Faint
	Italic
	Underlined
	Blink
	RapidBlink // this one is added by SGR
	Inverse
	Invisible
	SIZE
)

const TrueColorMask uint32 = 0x1000000

/* Don't set the fields directly
 *
 * Renditions is comparable
 */
type Renditions struct {
	fgColor    uint32
	bgColor    uint32
	attributes uint32
}

/*
The index start from 0. represent ANSI standard color
The index can also be a true color in the format: TrueColorMask , r , g , b
*/
func (r *Renditions) SetForegroundColor(index uint32) {
	if index <= 255 {
		r.fgColor = 30 + index
	} else if isTrueColor(index) {
		r.fgColor = index
	}
}

/*
The index start from 0. represent ANSI standard color
The index can also be a true color in the format: TrueColorMask , r , g , b
*/
func (r *Renditions) SetBackgroundColor(index uint32) {
	if index <= 255 {
		r.bgColor = 40 + index
	} else if isTrueColor(index) {
		r.bgColor = index
	}
}

// set the 24bit foreground color
func (d *Renditions) SetFgColor(r, g, b uint32) {
	d.fgColor = makeTrueColor(r, g, b)
}

// set the 24bit foreground color
func (d *Renditions) SetBgColor(r, g, b uint32) {
	d.bgColor = makeTrueColor(r, g, b)
}

// This method cannot be used to set a color beyond the 16-color set.
func (r *Renditions) SetRendition(color uint32) {
	if color == 0 {
		r.ClearAttributes()
		r.fgColor = 0
		r.bgColor = 0
		return
	}

	// SGR (Select Graphic Rendition) parameters in
	// https://en.wikipedia.org/wiki/ANSI_escape_code
	switch color {
	case 39: // Sets the foreground color to the user's configured default text color
		r.fgColor = 0
		return
	case 49: // Sets the background color to the user's configured default background color
		r.bgColor = 0
		return
	}

	if 30 <= color && color <= 37 { // foreground color in 8-color set
		r.fgColor = color
		return
	} else if 40 <= color && color <= 47 { // background color in 8-color set
		r.bgColor = color
		return
	} else if 90 <= color && color <= 97 { //  foreground color in 16-color set
		r.fgColor = color - 90 + 38
		return
	} else if 100 <= color && color <= 107 { // background color in 16-color set
		r.bgColor = color - 100 + 48
	}

	turnOn := color < 9
	switch color {
	case 1, 22:
		r.SetAttributes(Bold, turnOn)
	case 3, 23:
		r.SetAttributes(Italic, turnOn)
	case 4, 24:
		r.SetAttributes(Underlined, turnOn)
	case 5, 25:
		r.SetAttributes(Blink, turnOn)
	case 7, 27:
		r.SetAttributes(Inverse, turnOn)
	case 8, 28:
		r.SetAttributes(Invisible, turnOn)
	}
}

/*
 * https://invisible-island.net/xterm/ctlseqs/ctlseqs.html#h3-Functions-using-CSI-_-ordered-by-the-final-character_s_
 *
 * CSI Pm m  Character Attributes (SGR).
 *
 */
func (r *Renditions) SGR() string {
	var ret strings.Builder

	ret.WriteString("\033[0") // starrts with reset rendition

	if r.GetAttributes(Bold) {
		ret.WriteString(";1")
	}
	if r.GetAttributes(Italic) {
		ret.WriteString(";3")
	}
	if r.GetAttributes(Underlined) {
		ret.WriteString(";4")
	}
	if r.GetAttributes(Blink) {
		ret.WriteString(";5")
	}
	if r.GetAttributes(Inverse) {
		ret.WriteString(";7")
	}
	if r.GetAttributes(Invisible) {
		ret.WriteString(";8")
	}

	if r.fgColor > 0 {
		if isTrueColor(r.fgColor) { // 24 bit color
			fmt.Fprintf(&ret, ";38:2:%d:%d:%d", (r.fgColor>>16)&0xff, (r.fgColor>>8)&0xff, r.fgColor&0xff)
		} else if r.fgColor > 37 { // use 256-color set
			fmt.Fprintf(&ret, ";38:5:%d", r.fgColor-30)
		} else { // ANSI foreground color
			fmt.Fprintf(&ret, ";%d", r.fgColor)
		}
	}
	if r.bgColor > 0 {
		if isTrueColor(r.bgColor) {
			fmt.Fprintf(&ret, ";48:2:%d:%d:%d", (r.bgColor>>16)&0xff, (r.bgColor>>8)&0xff, r.bgColor&0xff)
		} else if r.bgColor > 47 {
			fmt.Fprintf(&ret, ";48:5:%d", r.bgColor-40)
		} else {
			fmt.Fprintf(&ret, ";%d", r.bgColor)
		}
	}
	ret.WriteString("m")
	return ret.String()
}

func (r *Renditions) SetAttributes(attr uint32, turnOn bool) {
	if turnOn {
		r.attributes |= (1 << attr)
	} else {
		r.attributes &= ^(1 << attr)
	}
}

func (r Renditions) GetAttributes(attr uint32) bool {
	return (r.attributes & (1 << attr)) > 0
}

func (r *Renditions) ClearAttributes() {
	r.attributes = 0
}

func makeTrueColor(r, g, b uint32) uint32 {
	return TrueColorMask | (r << 16) | (g << 8) | b
}

func isTrueColor(color uint32) bool {
	return color&TrueColorMask != 0
}
