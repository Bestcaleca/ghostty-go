package terminal

// Cell represents a single character cell in the terminal grid.
type Cell struct {
	Char  rune   // Unicode character (0 = empty)
	Width uint8  // 0=empty, 1=narrow, 2=wide (spacer head)
	Style StyleID
}

// EmptyCell returns a cell with default style.
func EmptyCell(style StyleID) Cell {
	return Cell{Char: ' ', Width: 1, Style: style}
}

// IsEmpty returns true if the cell contains no character.
func (c Cell) IsEmpty() bool {
	return c.Char == 0 || c.Char == ' '
}

// Row represents a line of cells in the terminal.
type Row struct {
	Cells   []Cell
	Wrapped bool // soft-wrap flag (line continued from previous)
}

// NewRow creates a new row with the given width and default style.
func NewRow(width int, style StyleID) Row {
	cells := make([]Cell, width)
	for i := range cells {
		cells[i] = EmptyCell(style)
	}
	return Row{Cells: cells}
}

// Reset fills the row with empty cells using the given style.
func (r *Row) Reset(style StyleID) {
	for i := range r.Cells {
		r.Cells[i] = EmptyCell(style)
	}
	r.Wrapped = false
}
