package font

import (
	"image"
	"image/color"
)

const (
	DefaultAtlasWidth  = 2048
	DefaultAtlasHeight = 2048
)

// GlyphEntry represents a glyph's position in the atlas texture.
type GlyphEntry struct {
	X, Y          int     // top-left position in atlas (pixels)
	Width, Height int     // glyph dimensions (pixels)
	Advance       float32 // horizontal advance
	BearingX      float32 // left bearing
	BearingY      float32 // top bearing (from baseline)
	Valid         bool
}

// Shelf represents a horizontal strip in the atlas for shelf-packing.
type Shelf struct {
	X        int // current x position
	Y        int // top y position
	Height   int // shelf height
	MaxWidth int // total width available
}

// Atlas manages a texture atlas for glyph rendering using shelf packing.
type Atlas struct {
	width  int
	height int
	pixels []byte // RGBA pixel data

	shelves []Shelf
	entries map[rune]GlyphEntry

	dirty bool // needs GPU upload
}

// NewAtlas creates a new glyph atlas with the given dimensions.
func NewAtlas(width, height int) *Atlas {
	return &Atlas{
		width:   width,
		height:  height,
		pixels:  make([]byte, width*height*4),
		entries: make(map[rune]GlyphEntry),
		dirty:   true,
	}
}

// Get retrieves a glyph entry from the atlas. Returns false if not found.
func (a *Atlas) Get(r rune) (GlyphEntry, bool) {
	e, ok := a.entries[r]
	return e, ok
}

// Add rasterizes and adds a glyph to the atlas. Returns the entry.
func (a *Atlas) Add(r rune, bmp *GlyphBitmap) GlyphEntry {
	if e, ok := a.entries[r]; ok {
		return e
	}

	// Add 1px padding to prevent texture bleeding
	padW := bmp.Width + 2
	padH := bmp.Height + 2

	x, y, shelfIdx := a.findSpace(padW, padH)
	if shelfIdx < 0 {
		// Atlas full — return empty entry
		return GlyphEntry{}
	}

	// Copy glyph pixels into atlas (with 1px padding offset)
	dstX := x + 1
	dstY := y + 1
	a.blitGlyph(dstX, dstY, bmp)

	entry := GlyphEntry{
		X:       dstX,
		Y:       dstY,
		Width:   bmp.Width,
		Height:  bmp.Height,
		Advance: bmp.Advance,
		BearingX: bmp.BearingX,
		BearingY: bmp.BearingY,
		Valid:   true,
	}
	a.entries[r] = entry
	a.dirty = true

	// Update shelf position
	a.shelves[shelfIdx].X += padW

	return entry
}

// Pixels returns the RGBA pixel data of the atlas.
func (a *Atlas) Pixels() []byte {
	return a.pixels
}

// Width returns the atlas width in pixels.
func (a *Atlas) Width() int {
	return a.width
}

// Height returns the atlas height in pixels.
func (a *Atlas) Height() int {
	return a.height
}

// IsDirty returns whether the atlas has been modified since last upload.
func (a *Atlas) IsDirty() bool {
	return a.dirty
}

// ClearDirty clears the dirty flag (call after GPU upload).
func (a *Atlas) ClearDirty() {
	a.dirty = false
}

// findSpace finds a position for a glyph of the given size using shelf packing.
// Returns x, y, shelf index. Returns -1 for shelf index if no space.
func (a *Atlas) findSpace(w, h int) (int, int, int) {
	// Try to fit in an existing shelf
	for i := range a.shelves {
		s := &a.shelves[i]
		if h <= s.Height && s.X+w <= s.MaxWidth {
			return s.X, s.Y, i
		}
	}

	// Create a new shelf
	newY := 0
	if len(a.shelves) > 0 {
		last := a.shelves[len(a.shelves)-1]
		newY = last.Y + last.Height
	}

	if newY+h > a.height {
		return 0, 0, -1 // no space
	}

	shelf := Shelf{
		X:        0,
		Y:        newY,
		Height:   h,
		MaxWidth: a.width,
	}
	a.shelves = append(a.shelves, shelf)
	return 0, newY, len(a.shelves) - 1
}

// blitGlyph copies glyph pixels into the atlas at the given position.
func (a *Atlas) blitGlyph(dstX, dstY int, bmp *GlyphBitmap) {
	for row := 0; row < bmp.Height; row++ {
		for col := 0; col < bmp.Width; col++ {
			srcOff := (row*bmp.Width + col) * 4
			dstOff := ((dstY+row)*a.width + (dstX + col)) * 4
			if srcOff+3 < len(bmp.Pixels) && dstOff+3 < len(a.pixels) {
				a.pixels[dstOff+0] = bmp.Pixels[srcOff+0] // R
				a.pixels[dstOff+1] = bmp.Pixels[srcOff+1] // G
				a.pixels[dstOff+2] = bmp.Pixels[srcOff+2] // B
				a.pixels[dstOff+3] = bmp.Pixels[srcOff+3] // A
			}
		}
	}
}

// Clear fills the atlas with transparent black pixels.
func (a *Atlas) Clear() {
	for i := range a.pixels {
		a.pixels[i] = 0
	}
	a.entries = make(map[rune]GlyphEntry)
	a.shelves = nil
	a.dirty = true
}

// Image returns the atlas as an image.RGBA (useful for debugging).
func (a *Atlas) Image() *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, a.width, a.height))
	copy(img.Pix, a.pixels)
	return img
}

// CompositeGlyph draws a colored glyph into a destination RGBA image at the given position.
func CompositeGlyph(dst *image.RGBA, x, y int, entry GlyphEntry, atlasPixels []byte, atlasWidth int, fg color.RGBA) {
	for row := 0; row < entry.Height; row++ {
		for col := 0; col < entry.Width; col++ {
			srcX := entry.X + col
			srcY := entry.Y + row
			srcOff := (srcY*atlasWidth + srcX) * 4

			if srcOff+3 >= len(atlasPixels) {
				continue
			}

			alpha := atlasPixels[srcOff+3]
			if alpha == 0 {
				continue
			}

			r := uint16(fg.R) * uint16(alpha) / 255
			g := uint16(fg.G) * uint16(alpha) / 255
			b := uint16(fg.B) * uint16(alpha) / 255

			dst.SetRGBA(x+col, y+row, color.RGBA{
				R: uint8(r),
				G: uint8(g),
				B: uint8(b),
				A: alpha,
			})
		}
	}
}
