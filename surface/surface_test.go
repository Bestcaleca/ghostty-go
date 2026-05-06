package surface

import (
	"testing"

	"github.com/go-gl/glfw/v3.3/glfw"

	"github.com/ghostty-go/ghostty-go/input"
	"github.com/ghostty-go/ghostty-go/renderer"
	"github.com/ghostty-go/ghostty-go/terminal"
)

func TestEncodeUTF8(t *testing.T) {
	tests := []struct {
		r    rune
		want []byte
	}{
		{'A', []byte{0x41}},
		{'é', []byte{0xC3, 0xA9}},
		{'中', []byte{0xE4, 0xB8, 0xAD}},
		{'😀', []byte{0xF0, 0x9F, 0x98, 0x80}},
	}

	for _, tt := range tests {
		buf := make([]byte, 4)
		n := encodeUTF8(tt.r, buf)
		if n != len(tt.want) {
			t.Errorf("encodeUTF8(%U) = %d bytes, want %d", tt.r, n, len(tt.want))
			continue
		}
		for i := 0; i < n; i++ {
			if buf[i] != tt.want[i] {
				t.Errorf("encodeUTF8(%U)[%d] = 0x%02X, want 0x%02X", tt.r, i, buf[i], tt.want[i])
			}
		}
	}
}

func TestStyleToColor(t *testing.T) {
	fallback := renderer.Color{R: 0.5, G: 0.5, B: 0.5, A: 1.0}

	// Invalid color returns fallback
	c := styleToColor(terminal.Color{}, fallback)
	if c.R != 0.5 {
		t.Errorf("expected fallback R=0.5, got %f", c.R)
	}

	// Valid color
	c = styleToColor(terminal.Color{R: 255, G: 128, B: 0, Valid: true}, fallback)
	if c.R != 1.0 {
		t.Errorf("expected R=1.0, got %f", c.R)
	}
	if c.G < 0.50 || c.G > 0.51 {
		t.Errorf("expected G≈0.502, got %f", c.G)
	}
}

func TestApplyTextStyleReverseAndInvisible(t *testing.T) {
	cell := renderer.Cell{
		Char: 'X',
		FG:   renderer.Color{R: 0.9, G: 0.8, B: 0.7, A: 1},
		BG:   renderer.Color{R: 0.1, G: 0.2, B: 0.3, A: 1},
	}

	style := terminal.Style{Reverse: true}
	got := applyTextStyle(cell, style)
	if got.FG.R != cell.BG.R || got.BG.R != cell.FG.R {
		t.Fatalf("reverse style did not swap colors: got fg=%+v bg=%+v", got.FG, got.BG)
	}

	style = terminal.Style{Invisible: true}
	got = applyTextStyle(cell, style)
	if got.Char != ' ' {
		t.Fatalf("invisible style char = %q, want space", got.Char)
	}
}

func TestApplyTextStyleBoldAndFaint(t *testing.T) {
	cell := renderer.Cell{
		Char: 'X',
		FG:   renderer.Color{R: 0.5, G: 0.5, B: 0.5, A: 1},
		BG:   renderer.Color{R: 0.1, G: 0.1, B: 0.1, A: 1},
	}

	bold := applyTextStyle(cell, terminal.Style{Bold: true})
	if bold.FG.R <= cell.FG.R {
		t.Fatalf("bold did not brighten foreground: got %f, want > %f", bold.FG.R, cell.FG.R)
	}

	faint := applyTextStyle(cell, terminal.Style{Faint: true})
	if faint.FG.R >= cell.FG.R {
		t.Fatalf("faint did not dim foreground: got %f, want < %f", faint.FG.R, cell.FG.R)
	}
}

func TestApplyTextStyleSetsDecorationFlags(t *testing.T) {
	cell := renderer.Cell{Char: 'X'}
	style := terminal.Style{
		Underline:     terminal.UnderlineSingle,
		Strikethrough: true,
		Overline:      true,
	}

	got := applyTextStyle(cell, style)

	if !got.Underline || !got.Strikethrough || !got.Overline {
		t.Fatalf("decorations = underline:%t strike:%t overline:%t", got.Underline, got.Strikethrough, got.Overline)
	}
}

func TestGridSizeFromPixels(t *testing.T) {
	metrics := renderer.CellMetrics{CellWidth: 10, CellHeight: 20}

	rows, cols := gridSizeFromPixels(1000, 600, metrics)

	if rows != 30 || cols != 100 {
		t.Fatalf("grid size = %dx%d, want 30x100", rows, cols)
	}
}

func TestGridSizeFromPixelsHasMinimumOneCell(t *testing.T) {
	metrics := renderer.CellMetrics{CellWidth: 10, CellHeight: 20}

	rows, cols := gridSizeFromPixels(1, 1, metrics)

	if rows != 1 || cols != 1 {
		t.Fatalf("grid size = %dx%d, want minimum 1x1", rows, cols)
	}
}

func TestGridPositionFromPixelsSubtractsPadding(t *testing.T) {
	metrics := renderer.CellMetrics{
		CellWidth:  10,
		CellHeight: 15,
		PaddingX:   5,
		PaddingY:   7,
	}

	row, col := gridPositionFromPixels(24, 36, metrics)

	if row != 1 || col != 1 {
		t.Fatalf("grid position = %d,%d, want 1,1", row, col)
	}
}

func TestGridPositionFromPixelsClampsPaddingAreaToFirstCell(t *testing.T) {
	metrics := renderer.CellMetrics{
		CellWidth:  10,
		CellHeight: 15,
		PaddingX:   5,
		PaddingY:   7,
	}

	row, col := gridPositionFromPixels(2, 3, metrics)

	if row != 0 || col != 0 {
		t.Fatalf("grid position = %d,%d, want 0,0", row, col)
	}
}

func TestTerminalMouseBypassesLocalSelectionUnlessShift(t *testing.T) {
	if !terminalMouseBypassesLocalSelection(input.MouseModeNormal, 0) {
		t.Fatal("terminal mouse mode should bypass local selection")
	}
	if terminalMouseBypassesLocalSelection(input.MouseModeNormal, glfw.ModShift) {
		t.Fatal("shift should force local selection behavior")
	}
	if terminalMouseBypassesLocalSelection(input.MouseModeNone, 0) {
		t.Fatal("mouse mode none should keep local selection behavior")
	}
}

func TestFocusEventSequence(t *testing.T) {
	if got := focusEventSequence(false, true); got != nil {
		t.Fatalf("disabled focus sequence = %q, want nil", got)
	}
	if got := string(focusEventSequence(true, true)); got != "\x1b[I" {
		t.Fatalf("focus in sequence = %q", got)
	}
	if got := string(focusEventSequence(true, false)); got != "\x1b[O" {
		t.Fatalf("focus out sequence = %q", got)
	}
}

func TestCursorVisibleForRenderRequiresTerminalVisibility(t *testing.T) {
	if cursorVisibleForRender(false, true, 0) {
		t.Fatal("hidden terminal cursor should not render")
	}
	if cursorVisibleForRender(true, false, 0) {
		t.Fatal("blink-hidden cursor should not render")
	}
	if cursorVisibleForRender(true, true, 1) {
		t.Fatal("cursor should not render while scrolled back")
	}
	if !cursorVisibleForRender(true, true, 0) {
		t.Fatal("visible terminal cursor should render when blink is visible and not scrolled back")
	}
}
