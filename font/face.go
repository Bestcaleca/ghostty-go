package font

import (
	"fmt"
	"image"
	"os"

	xfont "golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

// Metrics holds font metrics relevant for terminal grid calculations.
type Metrics struct {
	CellWidth  float32 // advance width of 'M' (monospace)
	CellHeight float32 // ascent + descent + line gap
	Ascent     float32
	Descent    float32
	LineGap    float32
}

// Face wraps an OpenType font with precomputed metrics.
type Face struct {
	size     float64
	dpi      float64
	metrics  Metrics
	fontFace xfont.Face
}

// LoadFace loads an OpenType font or font collection from a file path at the given size.
func LoadFace(path string, size float64) (*Face, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read font file: %w", err)
	}
	return LoadFaceFromBytes(data, size)
}

// LoadFaceFromBytes loads an OpenType font or font collection from raw bytes.
func LoadFaceFromBytes(data []byte, size float64) (*Face, error) {
	collection, err := opentype.ParseCollection(data)
	if err != nil {
		return nil, fmt.Errorf("parse opentype collection: %w", err)
	}
	sfntFont, err := collection.Font(0)
	if err != nil {
		return nil, fmt.Errorf("select collection font: %w", err)
	}
	return newFace(sfntFont, size, 72)
}

func newFace(sfntFont *opentype.Font, size, dpi float64) (*Face, error) {
	face, err := opentype.NewFace(sfntFont, &opentype.FaceOptions{
		Size:    size,
		DPI:     dpi,
		Hinting: xfont.HintingFull,
	})
	if err != nil {
		return nil, fmt.Errorf("create opentype face: %w", err)
	}

	metrics := face.Metrics()
	cellW := measureCellWidth(face)

	return &Face{
		size:     size,
		dpi:      dpi,
		fontFace: face,
		metrics: Metrics{
			CellWidth:  float32(cellW) / 64.0,
			CellHeight: float32(metrics.Height) / 64.0,
			Ascent:     float32(metrics.Ascent) / 64.0,
			Descent:    float32(metrics.Descent) / 64.0,
			LineGap:    float32(metrics.Height-metrics.Ascent-metrics.Descent) / 64.0,
		},
	}, nil
}

// DefaultFace returns a built-in basic font face (7x13 fixed).
func DefaultFace() *Face {
	return &Face{
		fontFace: basicfont.Face7x13,
		metrics: Metrics{
			CellWidth:  7,
			CellHeight: 13,
			Ascent:     10,
			Descent:    3,
			LineGap:    0,
		},
	}
}

// Metrics returns the font metrics.
func (f *Face) Metrics() Metrics {
	return f.metrics
}

// FontFace returns the underlying x/image/font.Face.
func (f *Face) FontFace() xfont.Face {
	return f.fontFace
}

// HasGlyph reports whether this face contains a real glyph for r.
func (f *Face) HasGlyph(r rune) bool {
	_, _, ok := f.fontFace.GlyphBounds(r)
	return ok
}

// RasterizeGlyph renders a single glyph and returns its bitmap.
func (f *Face) RasterizeGlyph(r rune) *GlyphBitmap {
	bounds, advance, _ := f.fontFace.GlyphBounds(r)
	w := (bounds.Max.X - bounds.Min.X).Ceil()
	h := (bounds.Max.Y - bounds.Min.Y).Ceil()

	if w <= 0 || h <= 0 {
		return &GlyphBitmap{
			Width:   int(f.metrics.CellWidth),
			Height:  int(f.metrics.CellHeight),
			Advance: f.metrics.CellWidth,
		}
	}

	img := image.NewRGBA(image.Rect(0, 0, w, h))
	d := &xfont.Drawer{
		Dst:  img,
		Src:  image.White,
		Face: f.fontFace,
		Dot:  fixed.P(-bounds.Min.X.Ceil(), -bounds.Min.Y.Ceil()),
	}
	d.DrawString(string(r))

	return &GlyphBitmap{
		Pixels:   img.Pix,
		Width:    w,
		Height:   h,
		Advance:  float32(advance) / 64.0,
		BearingX: float32(bounds.Min.X) / 64.0,
		BearingY: float32(bounds.Min.Y) / 64.0,
	}
}

// measureCellWidth measures the advance width of 'M' for monospace grid sizing.
func measureCellWidth(f xfont.Face) fixed.Int26_6 {
	advance, _ := f.GlyphAdvance('M')
	if advance == 0 {
		advance, _ = f.GlyphAdvance('m')
	}
	if advance == 0 {
		advance = 8 * 64 // fallback
	}
	return advance
}

// GlyphBitmap holds a rasterized glyph's pixel data and metrics.
type GlyphBitmap struct {
	Pixels   []byte // RGBA pixel data
	Width    int
	Height   int
	Advance  float32
	BearingX float32
	BearingY float32
}
