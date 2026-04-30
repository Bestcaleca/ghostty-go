package terminal

// Selection represents a text selection in the terminal.
type Selection struct {
	Active    bool
	StartRow  int
	StartCol  int
	EndRow    int
	EndCol    int
	SelectionMode SelectionMode
}

// SelectionMode determines how selection behaves.
type SelectionMode int

const (
	SelectionChar SelectionMode = iota // Character-by-character
	SelectionWord                      // Double-click: word boundaries
	SelectionLine                      // Triple-click: full line
)

// Start begins a selection at the given position.
func (s *Selection) Start(row, col int, mode SelectionMode) {
	s.Active = true
	s.StartRow = row
	s.StartCol = col
	s.EndRow = row
	s.EndCol = col
	s.SelectionMode = mode
}

// Update extends the selection to the given position.
func (s *Selection) Update(row, col int) {
	if !s.Active {
		return
	}
	s.EndRow = row
	s.EndCol = col
}

// Clear deactivates the selection.
func (s *Selection) Clear() {
	s.Active = false
}

// Range returns the normalized start/end positions (start <= end).
func (s *Selection) Range() (startRow, startCol, endRow, endCol int) {
	if s.StartRow < s.EndRow || (s.StartRow == s.EndRow && s.StartCol <= s.EndCol) {
		return s.StartRow, s.StartCol, s.EndRow, s.EndCol
	}
	return s.EndRow, s.EndCol, s.StartRow, s.StartCol
}

// IsSelected returns true if the given position is within the selection.
func (s *Selection) IsSelected(row, col int) bool {
	if !s.Active {
		return false
	}
	sr, sc, er, ec := s.Range()
	if row < sr || row > er {
		return false
	}
	if row == sr && col < sc {
		return false
	}
	if row == er && col >= ec {
		return false
	}
	return true
}

// GetText extracts the selected text from the terminal grid.
func (t *Terminal) GetSelectedText() string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	sel := t.active.Selection
	if !sel.Active {
		return ""
	}

	sr, sc, er, ec := sel.Range()
	grid := t.active.Rows

	var result []rune

	for row := sr; row <= er; row++ {
		if row < 0 || row >= len(grid) {
			continue
		}

		startCol := 0
		endCol := len(grid[row].Cells)

		if row == sr {
			startCol = sc
		}
		if row == er {
			endCol = ec
		}

		// Track trailing spaces for trimming
		lineHasContent := false
		var lineRunes []rune

		for col := startCol; col < endCol; col++ {
			ch := grid[row].Cells[col].Char
			if ch == 0 {
				ch = ' '
			}
			lineRunes = append(lineRunes, ch)
			if ch != ' ' {
				lineHasContent = true
			}
		}

		if lineHasContent {
			// Trim trailing spaces
			for len(lineRunes) > 0 && lineRunes[len(lineRunes)-1] == ' ' {
				lineRunes = lineRunes[:len(lineRunes)-1]
			}
		}

		result = append(result, lineRunes...)

		// Add newline between lines (but not after the last line)
		if row < er {
			result = append(result, '\n')
		}
	}

	return string(result)
}

// findWordBoundaries finds word boundaries around the given position.
func findWordBoundaries(cells []Cell, col int) (start, end int) {
	if col < 0 || col >= len(cells) {
		return col, col
	}

	// Find word start
	start = col
	for start > 0 && isWordChar(cells[start-1].Char) {
		start--
	}

	// Find word end
	end = col
	for end < len(cells) && isWordChar(cells[end].Char) {
		end++
	}

	return start, end
}

// isWordChar returns true if the character is part of a word.
func isWordChar(ch rune) bool {
	if ch == 0 || ch == ' ' {
		return false
	}
	// Letters, digits, underscore, dot, dash
	return (ch >= 'a' && ch <= 'z') ||
		(ch >= 'A' && ch <= 'Z') ||
		(ch >= '0' && ch <= '9') ||
		ch == '_' || ch == '.' || ch == '-' || ch == '/'
}
