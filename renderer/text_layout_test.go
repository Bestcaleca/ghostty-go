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

func TestMetricsIncludesPadding(t *testing.T) {
	r := &Renderer{cellW: 10, cellH: 20, paddingX: 5, paddingY: 7}

	got := r.Metrics()

	if got.CellWidth != 10 || got.CellHeight != 20 || got.PaddingX != 5 || got.PaddingY != 7 {
		t.Fatalf("metrics = %+v", got)
	}
}

func TestBuildDecorationDataCreatesLines(t *testing.T) {
	r := &Renderer{cellW: 10, cellH: 20, gridRows: 1, gridCols: 3}
	grid := [][]Cell{{
		{FG: Color{R: 1, A: 1}, Underline: true},
		{FG: Color{G: 1, A: 1}, Strikethrough: true},
		{FG: Color{B: 1, A: 1}, Overline: true},
	}}

	count := r.buildDecorationData(grid)

	if count != 3 {
		t.Fatalf("decoration count = %d, want 3", count)
	}
	if len(r.decorationData) != 24 {
		t.Fatalf("decoration data len = %d, want 24", len(r.decorationData))
	}
	underlineY := r.decorationData[2]
	strikeY := r.decorationData[10]
	overlineY := r.decorationData[18]
	if underlineY <= 0 {
		t.Fatalf("underline y offset = %f, want positive cell-local y", underlineY)
	}
	if strikeY <= overlineY || strikeY >= underlineY {
		t.Fatalf("strikethrough y = %f, want between overline %f and underline %f", strikeY, overlineY, underlineY)
	}
	if overlineY != 0 {
		t.Fatalf("overline y offset = %f, want 0", overlineY)
	}
}
