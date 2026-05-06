package surface

import (
	"testing"

	"github.com/ghostty-go/ghostty-go/renderer"
)

func TestNewContextMenuClampsToGrid(t *testing.T) {
	menu := newContextMenu(23, 79, 24, 80)

	if !menu.visible {
		t.Fatal("expected context menu to be visible")
	}
	if menu.row+len(menu.items) > 24 {
		t.Fatalf("menu row overflows grid: row=%d items=%d", menu.row, len(menu.items))
	}
	if menu.col+menu.width > 80 {
		t.Fatalf("menu col overflows grid: col=%d width=%d", menu.col, menu.width)
	}
}

func TestApplyContextMenuOverlayWritesMenuText(t *testing.T) {
	grid := make([][]renderer.Cell, 4)
	for row := range grid {
		grid[row] = make([]renderer.Cell, 20)
	}

	menu := newContextMenu(1, 2, 4, 20)
	applyContextMenuOverlay(grid, menu)

	got := string([]rune{
		grid[1][3].Char,
		grid[1][4].Char,
		grid[1][5].Char,
		grid[1][6].Char,
	})
	if got != "Copy" {
		t.Fatalf("overlay text = %q, want %q", got, "Copy")
	}
}
