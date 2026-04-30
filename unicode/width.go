package unicode

import (
	"golang.org/x/text/width"
)

// RuneWidth returns the display width of a rune (0, 1, or 2).
// Control characters and combining marks return 0. CJK and fullwidth characters return 2.
// Most other characters return 1.
func RuneWidth(r rune) int {
	// Control characters (C0, C1, DEL)
	if r < 0x20 || (r >= 0x7F && r < 0xA0) || r == 0xAD {
		return 0
	}

	// Combining marks (Grapheme_Extend) have zero width
	if isCombiningMark(r) {
		return 0
	}

	// Use x/text/width for East Asian Width property
	p := width.LookupRune(r)
	if p.Kind() == width.EastAsianWide || p.Kind() == width.EastAsianFullwidth {
		return 2
	}

	return 1
}

// isCombiningMark returns true for combining characters that have zero width.
func isCombiningMark(r rune) bool {
	// Combining Diacritical Marks (U+0300–U+036F)
	if r >= 0x0300 && r <= 0x036F {
		return true
	}
	// Combining Diacritical Marks Extended (U+1AB0–U+1AFF)
	if r >= 0x1AB0 && r <= 0x1AFF {
		return true
	}
	// Combining Diacritical Marks Supplement (U+1DC0–U+1DFF)
	if r >= 0x1DC0 && r <= 0x1DFF {
		return true
	}
	// Combining Diacritical Marks for Symbols (U+20D0–U+20FF)
	if r >= 0x20D0 && r <= 0x20FF {
		return true
	}
	// Combining Half Marks (U+FE20–U+FE2F)
	if r >= 0xFE20 && r <= 0xFE2F {
		return true
	}
	// Variation Selectors (U+FE00–U+FE0F)
	if r >= 0xFE00 && r <= 0xFE0F {
		return true
	}
	return false
}

// StringWidth returns the total display width of a string.
func StringWidth(s string) int {
	w := 0
	for _, r := range s {
		w += RuneWidth(r)
	}
	return w
}

// TruncateWidth truncates a string to fit within the given display width.
// Returns the truncated string.
func TruncateWidth(s string, maxWidth int) string {
	w := 0
	for i, r := range s {
		rw := RuneWidth(r)
		if w+rw > maxWidth {
			return s[:i]
		}
		w += rw
	}
	return s
}
