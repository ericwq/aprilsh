package terminal

type Framebuffer struct {
	rows             []Row
	iconName         string
	windowTitle      string
	bellCount        int
	titleInitialized bool
	DS               DrawState
}

func NewFramebuffer(width, height int) *Framebuffer {
	if width <= 0 || height <= 0 {
		return nil
	}

	fb := Framebuffer{}
	fb.DS = *NewDrawState(width, height)
	fb.rows = make([]Row, height)
	for i := range fb.rows {
		fb.rows[i] = *NewRow(width, 0)
	}

	return &fb
}

func (fb *Framebuffer) newRow(width, height int) *Row {
	w := fb.DS.GetWidth()
	bgColor := fb.DS.GetBackgroundRendition()
	return NewRow(w, uint32(bgColor))
}
