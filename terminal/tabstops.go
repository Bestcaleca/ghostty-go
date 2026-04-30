package terminal

// Tabstops manages tab stop positions.
type Tabstops struct {
	stops []bool
	width int
}

// NewTabstops creates tabstops with default positions (every 8 columns).
func NewTabstops(width int) Tabstops {
	stops := make([]bool, width)
	for i := 0; i < width; i += 8 {
		stops[i] = true
	}
	return Tabstops{stops: stops, width: width}
}

// Next returns the column of the next tab stop after col, or width-1 if none.
func (t *Tabstops) Next(col int) int {
	for i := col + 1; i < t.width; i++ {
		if t.stops[i] {
			return i
		}
	}
	return t.width - 1
}

// Prev returns the column of the previous tab stop before col, or 0 if none.
func (t *Tabstops) Prev(col int) int {
	for i := col - 1; i >= 0; i-- {
		if t.stops[i] {
			return i
		}
	}
	return 0
}

// Set sets a tab stop at the given column.
func (t *Tabstops) Set(col int) {
	if col >= 0 && col < t.width {
		t.stops[col] = true
	}
}

// Clear clears a tab stop at the given column.
func (t *Tabstops) Clear(col int) {
	if col >= 0 && col < t.width {
		t.stops[col] = false
	}
}

// ClearAll clears all tab stops.
func (t *Tabstops) ClearAll() {
	for i := range t.stops {
		t.stops[i] = false
	}
}

// IsSet returns true if there's a tab stop at the given column.
func (t *Tabstops) IsSet(col int) bool {
	if col >= 0 && col < t.width {
		return t.stops[col]
	}
	return false
}

// Resize resizes the tabstops for a new width, preserving existing stops.
func (t *Tabstops) Resize(width int) {
	if width > t.width {
		newStops := make([]bool, width)
		copy(newStops, t.stops)
		t.stops = newStops
		t.width = width
	} else {
		t.width = width
	}
}
