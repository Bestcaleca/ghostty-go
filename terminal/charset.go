package terminal

// CharsetDesignation represents a character set slot (G0-G3).
type CharsetDesignation uint8

const (
	CharsetASCII          CharsetDesignation = iota // ASCII (default)
	CharsetDecSpecial                               // DEC Special Graphics (line drawing)
	CharsetDecSupplemental                          // DEC Supplemental
	CharsetUK                                       // UK
	CharsetUTF8                                     // UTF-8 (handled at a higher level)
)

// CharsetState tracks the current character set state.
type CharsetState struct {
	G0, G1, G2, G3 CharsetDesignation // designated charsets
	GL             int                // 0=G0, 1=G1 (left side active charset)
	GR             int                // 2=G2, 3=G3 (right side active charset)
	SingleShift    int                // 0=none, 2=G2, 3=G3 (one-shot shift)
}

// NewCharsetState returns the default charset state (G0=ASCII, GL=G0).
func NewCharsetState() CharsetState {
	return CharsetState{
		G0: CharsetASCII,
		G1: CharsetASCII,
		G2: CharsetASCII,
		G3: CharsetASCII,
		GL: 0,
		GR: 2,
	}
}

// Designate sets a charset slot.
func (cs *CharsetState) Designate(slot int, charset CharsetDesignation) {
	switch slot {
	case 0:
		cs.G0 = charset
	case 1:
		cs.G1 = charset
	case 2:
		cs.G2 = charset
	case 3:
		cs.G3 = charset
	}
}

// InvokeGL sets the active left-side charset.
func (cs *CharsetState) InvokeGL(slot int) {
	cs.GL = slot
}

// InvokeGR sets the active right-side charset.
func (cs *CharsetState) InvokeGR(slot int) {
	cs.GR = slot
}

// LockingShift sets a locking shift (LS2, LS3, etc.).
func (cs *CharsetState) LockingShift(slot int) {
	cs.GL = slot
}

// GetActiveCharset returns the currently active charset for the left side.
func (cs *CharsetState) GetActiveCharset() CharsetDesignation {
	if cs.SingleShift > 0 {
		slot := cs.SingleShift
		cs.SingleShift = 0
		switch slot {
		case 2:
			return cs.G2
		case 3:
			return cs.G3
		}
	}
	switch cs.GL {
	case 0:
		return cs.G0
	case 1:
		return cs.G1
	case 2:
		return cs.G2
	case 3:
		return cs.G3
	}
	return CharsetASCII
}

// MapChar maps a byte through the DEC Special Graphics charset.
// Returns the mapped rune, or 0 if no mapping exists.
func MapDecSpecialGraphics(b byte) rune {
	// DEC Special Graphics Character Set (VT100 line drawing)
	if b >= 0x5F && b <= 0x7E {
		mapping := map[byte]rune{
			0x5F: ' ',        // blank
			0x60: '◆',        // diamond
			0x61: '▒',        // checkerboard
			0x62: '␉',        // HT
			0x63: '␌',        // FF
			0x64: '␍',        // CR
			0x65: '␊',        // LF
			0x66: '°',        // degree
			0x67: '±',        // plus/minus
			0x68: '┘',        // bottom right corner
			0x69: '┐',        // top right corner
			0x6A: '┌',        // top left corner
			0x6B: '└',        // bottom left corner
			0x6C: '┼',        // crossing lines
			0x6D: '─',        // horizontal line (scan 1)
			0x6E: '─',        // horizontal line (scan 3)
			0x6F: '─',        // horizontal line (scan 5)
			0x70: '─',        // horizontal line (scan 7)
			0x71: '─',        // horizontal line (scan 9)
			0x72: '├',        // left T
			0x73: '┤',        // right T
			0x74: '┴',        // bottom T
			0x75: '┬',        // top T
			0x76: '│',        // vertical line
			0x77: '≤',        // less than or equal
			0x78: '≥',        // greater than or equal
			0x79: 'π',        // pi
			0x7A: '≠',        // not equal
			0x7B: '£',        // pound sign
			0x7C: '·',        // middle dot
			0x7D: '▒',        // not used (checkerboard)
			0x7E: '÷',        // division
		}
		if r, ok := mapping[b]; ok {
			return r
		}
	}
	return 0
}
