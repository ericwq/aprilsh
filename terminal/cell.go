package terminal

const (
	bold uint = iota
	faint
	italic
	underlined
	blink
	inverse
	invisible
	SIZE
)

type Renditions struct {
	foregroundColor uint
	backgroundColor uint
	attributes      uint
}

func (r *Renditions) setForeGroundColor(color uint) {
	if 0 < color && color <= 255 {
		r.foregroundColor = 30 + color
	}
}

func (r *Renditions) setBackgroundColor(color uint) {
	if 0 < color && color <= 255 {
		r.backgroundColor = 40 + color
	}
}

func (r *Renditions) setRendition(color int) {
}

func (r *Renditions) sgr() string {
	return ""
}

func (r *Renditions) setAttributes(attr uint, val bool) {
	if val {
		r.attributes |= (1 << attr)
	} else {
		r.attributes &= (^(1 << attr))
	}
}

func (r *Renditions) getAttributes(attr uint) (has bool) {
	return (r.attributes & (1 << attr)) > 0
}

func (r *Renditions) clearAttributes() {
	r.attributes = 0
}
