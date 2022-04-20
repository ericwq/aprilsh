package terminal

import (
	"testing"
)

func TestFramebufferNewFramebuffer(t *testing.T) {
	width := 80
	height := 40
	fb := NewFramebuffer(width, height)
	if fb.DS.GetWidth() != width || fb.DS.GetHeight() != height {
		t.Errorf("DS size expect %dx%d, got %dx%d\n", width, height, fb.DS.GetWidth(), fb.DS.GetHeight())
	}
	if len(fb.rows) != height {
		t.Errorf("rows expect %d, got %d\n", height, len(fb.rows))
	}

	fb = NewFramebuffer(-1, -2)
	if fb != nil {
		t.Errorf("new expect nil, got %v\n", fb)
	}
}
