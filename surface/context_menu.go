package surface

import "github.com/ghostty-go/ghostty-go/renderer"

type contextMenuAction int

const (
	contextMenuCopy contextMenuAction = iota
	contextMenuPaste
	contextMenuClearSelection
)

type contextMenuItem struct {
	label  string
	action contextMenuAction
}

type contextMenu struct {
	visible bool
	row     int
	col     int
	width   int
	items   []contextMenuItem
}

func newContextMenu(row, col, rows, cols int) contextMenu {
	items := []contextMenuItem{
		{label: "Copy", action: contextMenuCopy},
		{label: "Paste", action: contextMenuPaste},
		{label: "Clear Selection", action: contextMenuClearSelection},
	}

	width := 0
	for _, item := range items {
		if len(item.label)+2 > width {
			width = len(item.label) + 2
		}
	}

	if rows > 0 {
		row = clampInt(row, 0, maxInt(0, rows-len(items)))
	}
	if cols > 0 {
		col = clampInt(col, 0, maxInt(0, cols-width))
	}

	return contextMenu{
		visible: true,
		row:     row,
		col:     col,
		width:   width,
		items:   items,
	}
}

func applyContextMenuOverlay(grid [][]renderer.Cell, menu contextMenu) {
	if !menu.visible {
		return
	}

	fg := renderer.Color{R: 0.95, G: 0.95, B: 0.95, A: 1}
	bg := renderer.Color{R: 0.16, G: 0.17, B: 0.20, A: 1}

	for i, item := range menu.items {
		row := menu.row + i
		if row < 0 || row >= len(grid) {
			continue
		}

		label := " " + item.label
		for x := 0; x < menu.width; x++ {
			col := menu.col + x
			if col < 0 || col >= len(grid[row]) {
				continue
			}

			ch := ' '
			if x < len(label) {
				ch = rune(label[x])
			}

			grid[row][col] = renderer.Cell{
				Char:  ch,
				FG:    fg,
				BG:    bg,
				Width: 1,
			}
		}
	}
}

func (m contextMenu) actionAt(row, col int) (contextMenuAction, bool) {
	if !m.visible {
		return 0, false
	}
	if col < m.col || col >= m.col+m.width {
		return 0, false
	}
	item := row - m.row
	if item < 0 || item >= len(m.items) {
		return 0, false
	}
	return m.items[item].action, true
}

func clampInt(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
