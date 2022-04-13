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

type Renditions struct {
	foregroundColor uint32
	backgroundColor uint32
	attributes      uint32
}

func (r *Renditions) SetForegroundColor(color uint32) {
	if color <= 255 {
		r.foregroundColor = 30 + color
	} else if isTrueColor(color) {
		r.foregroundColor = color
	}
}

func (r *Renditions) SetBackgroundColor(color uint32) {
	if color <= 255 {
		r.backgroundColor = 40 + color
	} else if isTrueColor(color) {
		r.backgroundColor = color
	}
}

func (r *Renditions) SetRendition(color uint32) {
	if color == 0 {
		r.ClearAttributes()
		r.foregroundColor = 0
		r.backgroundColor = 0
		return
	}

	// SGR (Select Graphic Rendition) parameters in
	// https://en.wikipedia.org/wiki/ANSI_escape_code
	switch color {
	case 39: // Sets the foreground color to the user's configured default text color
		r.foregroundColor = 0
		return
	case 49: // Sets the background color to the user's configured default background color
		r.backgroundColor = 0
		return
	}

	if 30 <= color && color <= 37 { // foreground color in 8-color set
		r.foregroundColor = color
		return
	} else if 40 <= color && color <= 47 { // background color in 8-color set
		r.backgroundColor = color
		return
	} else if 90 <= color && color <= 97 { //  foreground color in 16-color set
		r.foregroundColor = color - 90 + 38
		return
	} else if 100 <= color && color <= 107 { // background color in 16-color set
		r.backgroundColor = color - 100 + 48
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

	if r.foregroundColor > 0 {
		if isTrueColor(r.foregroundColor) { // 24 bit color
			fmt.Fprintf(&ret, ";38:2;%d;%d;%d", (r.foregroundColor>>16)&0xff, (r.foregroundColor>>8)&0xff, r.foregroundColor&0xff)
		} else if r.foregroundColor > 37 { // use 256-color set
			fmt.Fprintf(&ret, ";38;5;%d", r.foregroundColor-30)
		} else { // ANSI foreground color
			fmt.Fprintf(&ret, ";%d", r.foregroundColor)
		}
	}
	if r.backgroundColor > 0 {
		if isTrueColor(r.backgroundColor) {
			fmt.Fprintf(&ret, ";48;2;%d;%d;%d", (r.backgroundColor>>16)&0xff, (r.backgroundColor>>8)&0xff, r.backgroundColor&0xff)
		} else if r.backgroundColor > 47 {
			fmt.Fprintf(&ret, ";48;5;%d", r.backgroundColor-40)
		} else {
			fmt.Fprintf(&ret, ";%d", r.backgroundColor)
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
