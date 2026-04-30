package unicode

// GraphemeCluster represents a user-perceived character, which may consist
// of multiple Unicode code points (e.g., base + combining marks, emoji ZWJ sequences).
type GraphemeCluster struct {
	Runes []rune
	Width int // display width
}

// GraphemeClusters splits a string into grapheme clusters.
// Implements UAX #29 extended grapheme cluster boundaries.
func GraphemeClusters(s string) []GraphemeCluster {
	var clusters []GraphemeCluster
	var current []rune

	for _, r := range s {
		if len(current) == 0 {
			current = append(current, r)
			continue
		}

		if shouldBreakBetween(current[len(current)-1], r) {
			// Break: finalize current cluster
			clusters = append(clusters, makeCluster(current))
			current = []rune{r}
		} else {
			// Don't break: extend current cluster
			current = append(current, r)
		}
	}

	if len(current) > 0 {
		clusters = append(clusters, makeCluster(current))
	}

	return clusters
}

func makeCluster(runes []rune) GraphemeCluster {
	w := 0
	for _, r := range runes {
		w += RuneWidth(r)
	}
	return GraphemeCluster{Runes: runes, Width: w}
}

// shouldBreakBetween returns true if there should be a grapheme cluster
// boundary between 'prev' and 'curr'.
func shouldBreakBetween(prev, curr rune) bool {
	// GB3: Do not break between CR and LF
	if prev == '\r' && curr == '\n' {
		return false
	}

	// GB4: Break before controls (CR, LF, control chars)
	if isControl(curr) {
		return true
	}

	// GB5: Break after controls
	if isControl(prev) {
		return true
	}

	// GB9: Do not break before Extend characters or ZWJ
	if isExtend(curr) || curr == 0x200D { // ZWJ
		return false
	}

	// GB9a: Do not break before SpacingMark
	if isSpacingMark(curr) {
		return false
	}

	// GB9b: Do not break after Prepend
	if isPrepend(prev) {
		return false
	}

	// GB11: Do not break within emoji ZWJ sequences (ZWJ × \p{Extended_Pictographic})
	if prev == 0x200D && isExtendedPictographic(curr) {
		return false
	}

	// GB12/GB13: Do not break within emoji flag sequences (RI × RI)
	if isRegionalIndicator(prev) && isRegionalIndicator(curr) {
		return false
	}

	// GB999: Otherwise, break everywhere
	return true
}

// isControl returns true for control characters (General_Category = Control)
func isControl(r rune) bool {
	return r < 0x20 || r == 0x7F ||
		(r >= 0x80 && r < 0xA0) ||
		r == 0x0D || r == 0x0A || r == 0x09
}

// isExtend returns true for Extend characters (Grapheme_Extend = Yes)
// This includes combining marks, ZWJ, and other extending characters.
func isExtend(r rune) bool {
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
	// Combining marks in various blocks
	if r >= 0x0483 && r <= 0x0489 {
		return true
	}
	// Hangul Jamo combining vowels and consonants
	if r >= 0x1160 && r <= 0x11FF {
		return true
	}
	// Devanagari vowel signs and other Indic
	if r >= 0x0900 && r <= 0x097F {
		return isIndicExtend(r)
	}
	return false
}

func isIndicExtend(r rune) bool {
	// Simplified: vowel signs, virama, etc.
	return (r >= 0x093A && r <= 0x094D) ||
		(r >= 0x0951 && r <= 0x0957) ||
		(r >= 0x0962 && r <= 0x0963)
}

// isSpacingMark returns true for SpacingMark characters
func isSpacingMark(r rune) bool {
	// Thai vowel marks
	if r >= 0x0E31 && r <= 0x0E3A {
		return true
	}
	if r >= 0x0E47 && r <= 0x0E4E {
		return true
	}
	// Lao
	if r >= 0x0EB1 && r <= 0x0EBC {
		return true
	}
	return false
}

// isPrepend returns true for Prepend characters (Grapheme_Prepend = Yes)
func isPrepend(r rune) bool {
	// Thai, Lao, Khmer prepend characters
	return r == 0x0E40 || r == 0x0E41 || r == 0x0E42 ||
		r == 0x0E43 || r == 0x0E44 ||
		r == 0x0EC0 || r == 0x0EC1 || r == 0x0EC2 ||
		r == 0x0EC3 || r == 0x0EC4
}

// isRegionalIndicator returns true for Regional Indicator characters (flag emoji)
func isRegionalIndicator(r rune) bool {
	return r >= 0x1F1E6 && r <= 0x1F1FF
}

// isExtendedPictographic returns true for Extended_Pictographic characters (emoji)
func isExtendedPictographic(r rune) bool {
	// Emoji ranges (simplified but covers common cases)
	if r >= 0x1F000 && r <= 0x1FFFF {
		return true
	}
	// Miscellaneous Symbols and Arrows
	if r >= 0x2600 && r <= 0x27BF {
		return true
	}
	// Supplemental Symbols and Pictographs
	if r >= 0x1F900 && r <= 0x1F9FF {
		return true
	}
	// Symbols and Pictographs Extended-A
	if r >= 0x1FA70 && r <= 0x1FAFF {
		return true
	}
	return false
}
