package terminal

const (
	Bold uint = iota
	Faint
	Italic
	Underlined
	Blink
	Inverse
	Invisible
	SIZE
)

const TrueColorMask uint = 0x80000000

type Renditions struct {
	foregroundColor uint
	backgroundColor uint
	attributes      uint
}

func (r *Renditions) SetForegroundColor(color uint) {
	if color <= 255 {
		r.foregroundColor = 30 + color
	} else if isTrueColor(color) {
		r.foregroundColor = color
	}
}

func (r *Renditions) SetBackgroundColor(color uint) {
	if color <= 255 {
		r.backgroundColor = 40 + color
	} else if isTrueColor(color) {
		r.backgroundColor = color
	}
}

func (r *Renditions) SetRendition(color uint) {
	if color == 0 {
		r.ClearAttributes()
		r.foregroundColor = 0
		r.backgroundColor = 0
		return
	}

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
	return ""
}

func (r *Renditions) SetAttributes(attr uint, on bool) {
	if on {
		r.attributes |= (1 << attr)
	} else {
		r.attributes &= ^(1 << attr)
	}
}

func (r *Renditions) GetAttributes(attr uint) bool {
	return (r.attributes & (1 << attr)) > 0
}

func (r *Renditions) ClearAttributes() {
	r.attributes = 0
}

func makeTrueColor(r, g, b uint) uint {
	return TrueColorMask | (r << 16) | (g << 8) | b
}

func isTrueColor(color uint) bool {
	return TrueColorMask&color != 0
}
