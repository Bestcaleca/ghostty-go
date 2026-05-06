package renderer

import (
	"testing"

	"github.com/ghostty-go/ghostty-go/font"
)

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

func TestBuildTextDataRasterizesMissingGlyphs(t *testing.T) {
	rasterizer := &fakeGlyphRasterizer{}
	r := &Renderer{
		atlas:           font.NewAtlas(32, 32),
		cellAscent:      12,
		gridRows:        1,
		gridCols:        1,
		glyphRasterizer: rasterizer,
	}
	grid := [][]Cell{{
		{Char: '中', FG: Color{R: 1, G: 1, B: 1, A: 1}},
	}}

	count := r.buildTextData(grid)

	if count != 1 {
		t.Fatalf("text count = %d, want 1", count)
	}
	if rasterizer.calls != 1 {
		t.Fatalf("rasterizer calls = %d, want 1", rasterizer.calls)
	}
	if _, ok := r.atlas.Get('中'); !ok {
		t.Fatal("expected missing glyph to be added to atlas")
	}
}

type fakeGlyphRasterizer struct {
	calls int
}

func (f *fakeGlyphRasterizer) RasterizeGlyph(r rune) *font.GlyphBitmap {
	f.calls++
	return &font.GlyphBitmap{
		Pixels:   []byte{255, 255, 255, 255},
		Width:    1,
		Height:   1,
		Advance:  1,
		BearingX: 0,
		BearingY: -1,
	}
}
