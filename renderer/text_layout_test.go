package renderer

import "testing"

func TestGlyphTopOffsetUsesFontAscent(t *testing.T) {
	got := glyphTopOffset(15, -12)
	if got != 3 {
		t.Fatalf("glyphTopOffset() = %f, want 3", got)
	}
}
