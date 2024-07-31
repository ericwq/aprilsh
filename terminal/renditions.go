// Copyright 2022~2024 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package terminal

import (
	"fmt"
	"strings"
)

type charAttribute uint8

const (
	Bold charAttribute = iota + 1
	Faint
	Italic
	Underlined
	Blink
	RapidBlink // this one is added by SGR
	Inverse
	Invisible
)

// underline style
const (
	ULS_NONE charAttribute = iota
	ULS_SINGLE
	ULS_DOUBLE
	ULS_CURLY
	ULS_DOTTED
	ULS_DASHED
)

// Renditions determines the foreground and background color and character attribute.
// it is comparable. default background/foreground is ColorDefault
type Renditions struct {
	fgColor Color
	bgColor Color
	ulColor Color
	ulStyle charAttribute
	link    int
	// character attributes
	bold       bool
	faint      bool
	italic     bool
	underline  bool
	blink      bool
	rapidBlink bool
	inverse    bool
	invisible  bool
}

// func (r *Renditions) Equal(x *Renditions) bool {
// 	if r.fgColor != x.fgColor || r.bgColor != x.bgColor {
// 		return false
// 	}
//
// 	if r.bold != x.bold || r.faint != x.faint || r.italic != x.italic ||
// 		r.underline != x.underline || r.blink != x.blink || r.rapidBlink != x.rapidBlink ||
// 		r.inverse != x.inverse || r.invisible != x.invisible {
// 		return false
// 	}
// 	return true
// }

// set the ANSI foreground indexed color. The index start from 0. represent ANSI standard color.
func (rend *Renditions) SetForegroundColor(index int) {
	rend.fgColor = PaletteColor(index)
}

// set the ANSI background indexed color. The index start from 0. represent ANSI standard color.
func (rend *Renditions) SetBackgroundColor(index int) {
	rend.bgColor = PaletteColor(index)
}

// set the ANSI underline indexed color. The index start from 0. represent ANSI standard color.
func (rend *Renditions) setUnderlineColor(index int) {
	rend.ulColor = PaletteColor(index)
}

// set the ansi foreground palette color based on Color const
func (rend *Renditions) setAnsiForeground(c Color) {
	rend.fgColor = c
}

// set the ansi background palette color based on Color const
func (rend *Renditions) setAnsiBackground(c Color) {
	rend.bgColor = c
}

// set RGB foreground color
func (rend *Renditions) SetFgColor(r, g, b int) {
	rend.fgColor = NewRGBColor(int32(r), int32(g), int32(b))
}

// set RGB background color
func (rend *Renditions) SetBgColor(r, g, b int) {
	rend.bgColor = NewRGBColor(int32(r), int32(g), int32(b))
}

// set RGB underline color
func (rend *Renditions) setUnderlineRGBColor(r, g, b int) {
	rend.ulColor = NewRGBColor(int32(r), int32(g), int32(b))
}

// // set 256 underline color
// func (rend *Renditions) setUnderlineColor3(v Color) {
// 	rend.ulColor = v
// }

// set underline style
func (rend *Renditions) setUnderlineStyle(v charAttribute) {
	rend.ulStyle = v
}

// generate SGR sequence based on Renditions
// CSI Pm m  Character Attributes (SGR).
// https://invisible-island.net/xterm/ctlseqs/ctlseqs.html#h3-Functions-using-CSI-_-ordered-by-the-final-character_s_
// https://sw.kovidgoyal.net/kitty/underlines/
func (rend *Renditions) SGR() string {
	// quick check for default Renditions
	def := Renditions{}
	if *rend == def {
		return "\x1B[0m"
	}
	var sgr strings.Builder

	// starts with reset rendition
	sgr.WriteString("\x1B[0")

	// deal with character attributes first
	if rend.bold {
		sgr.WriteString(";1")
	}
	if rend.faint {
		sgr.WriteString(";2")
	}
	if rend.italic {
		sgr.WriteString(";3")
	}
	if rend.underline {
		if rend.ulStyle > ULS_NONE {
			switch rend.ulStyle {
			case ULS_SINGLE:
				sgr.WriteString(";4")
			default:
				sgr.WriteString(fmt.Sprintf(";4:%d", rend.ulStyle))
			}
		}
		if rend.ulColor != ColorDefault {
			if rend.ulColor.IsRGB() {
				r, g, b := rend.ulColor.RGB()
				fmt.Fprintf(&sgr, ";58:2::%d:%d:%d", r, g, b) // RGB foreground
			} else {
				fmt.Fprintf(&sgr, ";58:5:%d", rend.ulColor.Index())
			}
		}
	}
	if rend.blink {
		sgr.WriteString(";5")
	}
	if rend.rapidBlink {
		sgr.WriteString(";6")
	}
	if rend.inverse {
		sgr.WriteString(";7")
	}
	if rend.invisible {
		sgr.WriteString(";8")
	}

	// deal with foreground and background color. we only support colons to separate the subparameters
	// for default color, we don't do anything.
	if rend.fgColor != ColorDefault {
		if rend.fgColor.IsRGB() {
			r, g, b := rend.fgColor.RGB()
			fmt.Fprintf(&sgr, ";38:2:%d:%d:%d", r, g, b) // RGB foreground
		} else if rend.fgColor.Index() < 8 {
			fmt.Fprintf(&sgr, ";%d", rend.fgColor.Index()+30) // standard foregrounds, 8-color set
		} else if rend.fgColor.Index() < 16 {
			fmt.Fprintf(&sgr, ";%d", rend.fgColor.Index()+82) // bright colored foregrounds, 16-color set
		} else {
			fmt.Fprintf(&sgr, ";38:5:%d", rend.fgColor.Index())
		}
	}
	if rend.bgColor != ColorDefault {
		if rend.bgColor.IsRGB() {
			r, g, b := rend.bgColor.RGB()
			fmt.Fprintf(&sgr, ";48:2:%d:%d:%d", r, g, b) // RGB background
		} else if rend.bgColor.Index() < 8 {
			fmt.Fprintf(&sgr, ";%d", rend.bgColor.Index()+40) // standard backgrounds, 8-color set
		} else if rend.bgColor.Index() < 16 {
			fmt.Fprintf(&sgr, ";%d", rend.bgColor.Index()+92) // bright colored backgrounds, 16-color set
		} else {
			fmt.Fprintf(&sgr, ";48:5:%d", rend.bgColor.Index())
		}
	}

	// the final byte of SGR
	sgr.WriteString("m")
	return sgr.String()
}

func (r *Renditions) SetAttributes(attr charAttribute, value bool) {
	switch attr {
	case Bold:
		r.bold = value
	case Faint:
		r.faint = value
	case Italic:
		r.italic = value
	case Underlined:
		r.underline = value
	case Blink:
		r.blink = value
	case RapidBlink:
		r.rapidBlink = value
	case Inverse:
		r.inverse = value
	case Invisible:
		r.invisible = value
	}
}

func (r *Renditions) GetAttributes(attr charAttribute) (value, ok bool) {
	ok = true

	switch attr {
	case Bold:
		value = r.bold
	case Faint:
		value = r.faint
	case Italic:
		value = r.italic
	case Underlined:
		value = r.underline
	case Blink:
		value = r.blink
	case RapidBlink: // this one is added by SGR
		value = r.rapidBlink
	case Inverse:
		value = r.inverse
	case Invisible:
		value = r.invisible
	default:
		ok = false
	}

	return value, ok
}

func (rend *Renditions) ClearAttributes() {
	rend.bold = false
	rend.faint = false
	rend.italic = false
	rend.underline = false
	rend.blink = false
	rend.rapidBlink = false
	rend.inverse = false
	rend.invisible = false
}

// build renditions based on attribute parameter. This method can process 8-color, 16-color set and
// default color. Can be called multiple times. return true if buildRendition() can process the
// attribute, otherwise false.
func (rend *Renditions) buildRendition(attribute int) (processed bool) {
	processed = true
	// use the default background and foreground color
	switch attribute {
	case 0:
		rend.ClearAttributes()
		rend.setAnsiForeground(ColorDefault) // default foreground color
		rend.setAnsiBackground(ColorDefault) // default background color
		rend.ulColor = ColorDefault
		rend.ulStyle = ULS_NONE
	case 1:
		rend.bold = true
	case 2:
		rend.faint = true
	case 3:
		rend.italic = true
	case 4:
		rend.underline = true
		rend.ulStyle = ULS_SINGLE
	case 5:
		rend.blink = true
	case 6:
		rend.rapidBlink = true
	case 7:
		rend.inverse = true // TODO how to handle inverse?
	case 8:
		rend.invisible = true

	case 22:
		rend.bold = false
	case 23:
		rend.italic = false
	case 24:
		rend.underline = false
	case 25:
		rend.blink = false      // not blinking
		rend.rapidBlink = false // not blinking
	case 27:
		rend.inverse = false // TODO how to handle inverse
	case 28:
		rend.invisible = false

	// standard foregrounds
	case 30, 31, 32, 33, 34, 35, 36, 37:
		rend.SetForegroundColor(attribute - 30) // foreground color in 8-color set
	// default foreground color
	case 39:
		rend.setAnsiForeground(ColorDefault)
	// standard backgrounds
	case 40, 41, 42, 43, 44, 45, 46, 47:
		rend.SetBackgroundColor(attribute - 40) // background color in 8-color set
	// default background color
	case 49:
		rend.setAnsiBackground(ColorDefault)

	// bright colored foregrounds
	case 90, 91, 92, 93, 94, 95, 96, 97:
		rend.SetForegroundColor(attribute - 82) // foreground color in 16-color set
	// bright colored backgrounds
	case 100, 101, 102, 103, 104, 105, 106, 107:
		rend.SetBackgroundColor(attribute - 92) // background color in 16-color set
	default:
		processed = false // false means buildRendition() does not process it
	}

	return processed
}

// create rendition based on colorAttr parameter. This method can only be used to set 16-color set.
func NewRenditions(attribute int) (rend Renditions) {
	if attribute == 0 {
		rend.ClearAttributes()
		rend.fgColor = ColorDefault
		rend.bgColor = ColorDefault
		return
	}

	rend.buildRendition(attribute)
	return rend
}
