package renderer

import "testing"

func TestGlyphTopOffsetUsesFontAscent(t *testing.T) {
	got := glyphTopOffset(15, -12)
	if got != 3 {
		t.Fatalf("glyphTopOffset() = %f, want 3", got)
	}
}

func TestSetGridSizeUpdatesRenderBounds(t *testing.T) {
	r := &Renderer{gridRows: 24, gridCols: 80}

	r.SetGridSize(30, 100)

	if r.gridRows != 30 || r.gridCols != 100 {
		t.Fatalf("grid size = %dx%d, want 30x100", r.gridRows, r.gridCols)
	}
}
