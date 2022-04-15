package terminal

import (
	"fmt"
	"strings"
	"unicode"
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

/* Don't set the fields directly */
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

func (r *Renditions) GetAttributes(attr uint32) bool {
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

type Cell struct {
	contents   strings.Builder
	renditions Renditions
	wide       bool
	fallback   bool
	wrap       bool
}

func (c *Cell) Reset(bgColor uint32) {
	c.contents.Reset()
	c.renditions = Renditions{bgColor: bgColor}
	c.wide = false
	c.fallback = false
	c.wrap = false
}

func (c Cell) Empty() bool {
	return c.contents.Len() == 0
}

// 32 seems like a reasonable limit on combining characters
func (c Cell) Full() bool {
	return c.contents.Len() >= 32
}

func (c *Cell) Clear() {
	c.contents.Reset()
}

func (c Cell) IsBlank() bool {
	return c.Empty() || c.contents.String() == " " || c.contents.String() == "\xC2\xA0"
}

func (c Cell) ContentsMatch(x Cell) bool {
	return (c.IsBlank() && x.IsBlank()) || c.contents.String() == x.contents.String()
}

func (c Cell) Compare(x Cell) bool {
	// TODO
	return false
}

// Is this a printing ISO 8859-1 character?
func IsPrintISO8859_1(r rune) bool {
	// return (r <= 0xff && r >= 0xa0) || (r <= 0x7e && r >= 0x20)
	return unicode.IsGraphic(r)
}

func AppendToStr(dest strings.Builder, r rune) {
	// ASCII?  Cheat.
	if r < 0x7f {
		dest.WriteByte(byte(r))
		return
	}
	dest.WriteRune(r)
}

func (c *Cell) Append(r rune) {
	// ASCII?  Cheat.
	if r < 0x7f {
		c.contents.WriteByte(byte(r))
		return
	}
	c.contents.WriteRune(r)
}

func (c Cell) PrintGrapheme(s strings.Builder) {
	if c.Empty() {
		s.WriteString(" ")
		return
	}
	/*
	 * cells that begin with combining character get combiner
	 * attached to no-break space
	 */
	if c.fallback {
		s.WriteString("\xC2\xA0")
	}
	s.WriteString(c.contents.String())
}

func (c Cell) GetRenditions() Renditions {
	return c.renditions
}

func (c *Cell) SetRenditions(r Renditions) {
	c.renditions = r
}

func (c Cell) GetWide() bool {
	return c.wide
}

func (c Cell) GetWidth() uint {
	if c.wide {
		return 1
	} else {
		return 2
	}
}

func (c *Cell) SetWide(w bool) {
	c.wide = w
}

func (c Cell) GetFallback() bool {
	return c.fallback
}

func (c *Cell) SetFallback(f bool) {
	c.fallback = f
}

func (c Cell) GetWrap() bool {
	return c.wrap
}

func (c *Cell) SetWrap(w bool) {
	c.wrap = w
}
