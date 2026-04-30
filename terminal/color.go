package terminal

// Color represents a terminal color.
type Color struct {
	R, G, B uint8
	Valid   bool // false = default/inherited
}

// ColorNone is an unset/default color.
var ColorNone = Color{}

// ColorFromRGB creates a color from RGB values.
func ColorFromRGB(r, g, b uint8) Color {
	return Color{R: r, G: g, B: b, Valid: true}
}

// DefaultPalette returns the standard 256-color palette.
func DefaultPalette() [256]Color {
	var p [256]Color

	// 0-7: Standard ANSI colors
	p[0] = Color{0, 0, 0, true}       // Black
	p[1] = Color{205, 49, 49, true}    // Red
	p[2] = Color{13, 188, 84, true}    // Green
	p[3] = Color{229, 229, 16, true}   // Yellow
	p[4] = Color{36, 114, 200, true}   // Blue
	p[5] = Color{188, 63, 188, true}   // Magenta
	p[6] = Color{17, 168, 205, true}   // Cyan
	p[7] = Color{229, 229, 229, true}  // White

	// 8-15: Bright ANSI colors
	p[8] = Color{102, 102, 102, true}  // Bright Black (Gray)
	p[9] = Color{241, 76, 76, true}    // Bright Red
	p[10] = Color{35, 209, 139, true}  // Bright Green
	p[11] = Color{245, 245, 67, true}  // Bright Yellow
	p[12] = Color{59, 142, 234, true}  // Bright Blue
	p[13] = Color{214, 112, 214, true} // Bright Magenta
	p[14] = Color{41, 184, 219, true}  // Bright Cyan
	p[15] = Color{229, 229, 229, true} // Bright White

	// 16-231: 6x6x6 color cube
	for i := 0; i < 216; i++ {
		r := uint8(0)
		g := uint8(0)
		b := uint8(0)
		if i/36 > 0 {
			r = uint8(55 + (i/36)*40)
		}
		if (i/6)%6 > 0 {
			g = uint8(55 + ((i/6)%6)*40)
		}
		if i%6 > 0 {
			b = uint8(55 + (i%6)*40)
		}
		p[16+i] = Color{r, g, b, true}
	}

	// 232-255: Grayscale ramp
	for i := 0; i < 24; i++ {
		v := uint8(8 + i*10)
		p[232+i] = Color{v, v, v, true}
	}

	return p
}
