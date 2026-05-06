package terminal

// CursorStyle represents the visual style of the cursor.
type CursorStyle uint8

const (
	CursorDefault           CursorStyle = iota // block (default)
	CursorBlinkingBlock                        // blinking block
	CursorSteadyBlock                          // steady block
	CursorBlinkingUnderline                    // blinking underline
	CursorSteadyUnderline                      // steady underline
	CursorBlinkingBar                          // blinking bar/beam
	CursorSteadyBar                            // steady bar/beam
)

// Cursor represents the terminal cursor state.
type Cursor struct {
	Row     int
	Col     int
	Style   CursorStyle
	Visible bool
}

// SavedCursor holds the state saved by ESC 7 (DECSC) and restored by ESC 8 (DECRC).
type SavedCursor struct {
	Row     int
	Col     int
	Style   CursorStyle
	Charset CharsetState
	Origin  bool
	Wrap    bool
	Visible bool
	Valid   bool
}

// Save saves the current cursor and terminal state.
func (c *Cursor) Save(charset CharsetState, origin, wrap bool) SavedCursor {
	return SavedCursor{
		Row:     c.Row,
		Col:     c.Col,
		Style:   c.Style,
		Charset: charset,
		Origin:  origin,
		Wrap:    wrap,
		Visible: c.Visible,
		Valid:   true,
	}
}

// Restore restores cursor state from a saved state.
func (c *Cursor) Restore(s SavedCursor) {
	c.Row = s.Row
	c.Col = s.Col
	c.Style = s.Style
	c.Visible = s.Visible
}
