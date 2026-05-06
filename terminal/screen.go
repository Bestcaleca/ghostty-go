package terminal

// Screen represents a terminal screen (primary or alternate).
type Screen struct {
	Rows        []Row
	Cursor      Cursor
	Modes       ModeState
	Styles      *StyleTable
	Tabstops    Tabstops
	Charset     CharsetState
	SavedCursor SavedCursor

	Title string
	PWD   string

	Scrollback []Row // scrollback buffer
	MaxScroll  int   // max scrollback lines
	Selection  Selection

	ScrollbackEnabled bool
}

// NewScreen creates a new screen with the given dimensions.
func NewScreen(rows, cols int) *Screen {
	s := &Screen{
		Rows:              make([]Row, rows),
		Modes:             ModeState{DecAWM: true},
		Styles:            NewStyleTable(),
		Tabstops:          NewTabstops(cols),
		Charset:           NewCharsetState(),
		Cursor:            Cursor{Visible: true},
		MaxScroll:         10000,
		ScrollbackEnabled: true,
	}
	for i := range s.Rows {
		s.Rows[i] = NewRow(cols, s.Styles.DefaultID())
	}
	return s
}

// Resize resizes the screen to new dimensions.
func (s *Screen) Resize(rows, cols int) {
	oldRows := len(s.Rows)
	oldCols := 0
	if oldRows > 0 {
		oldCols = len(s.Rows[0].Cells)
	}

	newRows := make([]Row, rows)
	for i := range newRows {
		if i < oldRows {
			newRows[i] = s.Rows[i]
			// Resize row if needed
			if cols != oldCols {
				newCells := make([]Cell, cols)
				copy(newCells, newRows[i].Cells)
				for j := oldCols; j < cols; j++ {
					newCells[j] = EmptyCell(s.Styles.DefaultID())
				}
				newRows[i].Cells = newCells
			}
		} else {
			newRows[i] = NewRow(cols, s.Styles.DefaultID())
		}
	}
	s.Rows = newRows
	s.Tabstops.Resize(cols)

	// Clamp cursor
	if s.Cursor.Row >= rows {
		s.Cursor.Row = rows - 1
	}
	if s.Cursor.Col >= cols {
		s.Cursor.Col = cols - 1
	}
}

// Width returns the number of columns.
func (s *Screen) Width() int {
	if len(s.Rows) == 0 {
		return 0
	}
	return len(s.Rows[0].Cells)
}

// Height returns the number of rows.
func (s *Screen) Height() int {
	return len(s.Rows)
}

// ScrollUp scrolls the screen up by n lines within the scroll region.
func (s *Screen) ScrollUp(top, bottom, n int) {
	if n <= 0 || top >= bottom {
		return
	}
	width := s.Width()

	// Move lines to scrollback if scrolling the full screen
	if s.ScrollbackEnabled && top == 0 && bottom == len(s.Rows) {
		for i := 0; i < n && i < len(s.Rows); i++ {
			scrollRow := s.Rows[i]
			scrollRow.Wrapped = false
			s.Scrollback = append(s.Scrollback, scrollRow)
			if len(s.Scrollback) > s.MaxScroll {
				s.Scrollback = s.Scrollback[1:]
			}
		}
	}

	// Shift lines up
	for i := top; i < bottom-n; i++ {
		s.Rows[i] = s.Rows[i+n]
	}

	// Clear new lines at bottom
	for i := bottom - n; i < bottom; i++ {
		if i >= 0 && i < len(s.Rows) {
			s.Rows[i] = NewRow(width, s.Styles.DefaultID())
		}
	}
}

// ScrollDown scrolls the screen down by n lines within the scroll region.
func (s *Screen) ScrollDown(top, bottom, n int) {
	if n <= 0 || top >= bottom {
		return
	}
	width := s.Width()

	// Shift lines down
	for i := bottom - 1; i >= top+n; i-- {
		s.Rows[i] = s.Rows[i-n]
	}

	// Clear new lines at top
	for i := top; i < top+n && i < bottom; i++ {
		s.Rows[i] = NewRow(width, s.Styles.DefaultID())
	}
}
