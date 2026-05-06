package font

import (
	"image"
	"image/color"
	"os"
	"testing"

	xfont "golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

func TestFaceSetUsesFallbackForMissingGlyph(t *testing.T) {
	primary := newFakeFace('A')
	fallback := newFakeFace('中')
	set := NewFaceSet(primary, fallback)

	if got := set.FaceForRune('A'); got != primary {
		t.Fatal("expected primary face for ASCII glyph")
	}
	if got := set.FaceForRune('中'); got != fallback {
		t.Fatal("expected fallback face for CJK glyph")
	}
}

func TestFaceSetMetricsComeFromPrimary(t *testing.T) {
	primary := newFakeFace('A')
	primary.metrics.CellWidth = 10
	fallback := newFakeFace('中')
	fallback.metrics.CellWidth = 20

	set := NewFaceSet(primary, fallback)

	if got := set.Metrics().CellWidth; got != 10 {
		t.Fatalf("CellWidth = %f, want primary width 10", got)
	}
}

func TestLoadFaceSupportsFontCollections(t *testing.T) {
	path := "/usr/share/fonts/opentype/noto/NotoSansCJK-Regular.ttc"
	if _, err := os.Stat(path); err != nil {
		t.Skipf("font collection not installed: %s", path)
	}

	face, err := LoadFace(path, 16)
	if err != nil {
		t.Fatalf("LoadFace(%q) error = %v", path, err)
	}
	if !face.HasGlyph('中') {
		t.Fatal("expected CJK collection face to contain 中")
	}
}

func newFakeFace(glyphs ...rune) *Face {
	return &Face{
		fontFace: fakeFontFace{glyphs: runeSet(glyphs...)},
		metrics: Metrics{
			CellWidth:  8,
			CellHeight: 16,
			Ascent:     12,
			Descent:    4,
		},
	}
}

func runeSet(glyphs ...rune) map[rune]bool {
	set := make(map[rune]bool, len(glyphs))
	for _, glyph := range glyphs {
		set[glyph] = true
	}
	return set
}

type fakeFontFace struct {
	glyphs map[rune]bool
}

func (f fakeFontFace) Close() error { return nil }

func (f fakeFontFace) Glyph(dot fixed.Point26_6, r rune) (image.Rectangle, image.Image, image.Point, fixed.Int26_6, bool) {
	if !f.glyphs[r] {
		return image.Rectangle{}, nil, image.Point{}, 0, false
	}
	return image.Rect(0, 0, 1, 1), image.NewUniform(color.Alpha{A: 255}), image.Point{}, fixed.I(8), true
}

func (f fakeFontFace) GlyphBounds(r rune) (fixed.Rectangle26_6, fixed.Int26_6, bool) {
	if !f.glyphs[r] {
		return fixed.Rectangle26_6{}, 0, false
	}
	return fixed.R(0, -12, 8, 4), fixed.I(8), true
}

func (f fakeFontFace) GlyphAdvance(r rune) (fixed.Int26_6, bool) {
	if !f.glyphs[r] {
		return 0, false
	}
	return fixed.I(8), true
}

func (f fakeFontFace) Kern(r0, r1 rune) fixed.Int26_6 { return 0 }

func (f fakeFontFace) Metrics() xfont.Metrics {
	return xfont.Metrics{
		Height:  fixed.I(16),
		Ascent:  fixed.I(12),
		Descent: fixed.I(4),
	}
}
