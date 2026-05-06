package terminal

import (
	"testing"
	"time"

	"github.com/ghostty-go/ghostty-go/input"
	"github.com/ghostty-go/ghostty-go/parser"
)

// newTestTerminal creates a terminal for testing.
func newTestTerminal(rows, cols int) *Terminal {
	return New(rows, cols)
}

func TestTerminalPrint(t *testing.T) {
	term := newTestTerminal(24, 80)
	term.Print('H')
	term.Print('i')

	grid := term.Grid()
	if grid[0][0].Char != 'H' {
		t.Errorf("expected 'H' at (0,0), got %c", grid[0][0].Char)
	}
	if grid[0][1].Char != 'i' {
		t.Errorf("expected 'i' at (0,1), got %c", grid[0][1].Char)
	}
}

func TestTerminalNewline(t *testing.T) {
	term := newTestTerminal(24, 80)
	term.Print('A')
	term.Execute(0x0A) // LF (no CR unless LNM mode)
	term.Print('B')

	grid := term.Grid()
	if grid[0][0].Char != 'A' {
		t.Errorf("expected 'A' at (0,0), got %c", grid[0][0].Char)
	}
	// LF moves down but cursor col stays at 1 (no LNM mode)
	if grid[1][1].Char != 'B' {
		t.Errorf("expected 'B' at (1,1), got %c", grid[1][1].Char)
	}
}

func TestTerminalAutoWrapEnabledByDefault(t *testing.T) {
	term := newTestTerminal(2, 3)
	for _, ch := range []rune{'A', 'B', 'C', 'D'} {
		term.Print(ch)
	}

	grid := term.Grid()
	if grid[0][2].Char != 'C' {
		t.Fatalf("grid[0][2] = %q, want 'C'", grid[0][2].Char)
	}
	if grid[1][0].Char != 'D' {
		t.Fatalf("grid[1][0] = %q, want 'D'", grid[1][0].Char)
	}
}

func TestTerminalWideCharWrapsBeforeRightMargin(t *testing.T) {
	term := newTestTerminal(2, 3)
	term.Print('A')
	term.Print('B')
	term.Print('中')

	grid := term.Grid()
	if grid[0][2].Char != ' ' {
		t.Fatalf("grid[0][2] = %q, want blank before wide char wrap", grid[0][2].Char)
	}
	if grid[1][0].Char != '中' {
		t.Fatalf("grid[1][0] = %q, want '中'", grid[1][0].Char)
	}
	if grid[1][1].Char != 0 || grid[1][1].Width != 0 {
		t.Fatalf("wide char spacer = %q/%d, want zero-width spacer", grid[1][1].Char, grid[1][1].Width)
	}
}

func TestTerminalCarriageReturn(t *testing.T) {
	term := newTestTerminal(24, 80)
	term.Print('A')
	term.Print('B')
	term.Execute(0x0D) // CR
	term.Print('C')

	grid := term.Grid()
	if grid[0][0].Char != 'C' {
		t.Errorf("expected 'C' at (0,0), got %c", grid[0][0].Char)
	}
	if grid[0][1].Char != 'B' {
		t.Errorf("expected 'B' at (0,1), got %c", grid[0][1].Char)
	}
}

func TestTerminalCursorMovement(t *testing.T) {
	term := newTestTerminal(24, 80)

	// CSI 5 ; 10 H = Move to row 5, col 10
	term.CSIDispatch(parser.CSIDispatchAction{
		Final:      'H',
		Params:     [24]uint16{5, 10},
		ParamCount: 2,
	})

	term.Print('X')

	grid := term.Grid()
	if grid[4][9].Char != 'X' {
		t.Errorf("expected 'X' at (4,9), got %c", grid[4][9].Char)
	}
}

func TestTerminalEraseLine(t *testing.T) {
	term := newTestTerminal(24, 80)
	term.Print('A')
	term.Print('B')
	term.Print('C')
	term.Execute(0x0D) // CR

	// EL 0 = Erase right
	term.CSIDispatch(parser.CSIDispatchAction{
		Final:      'K',
		Params:     [24]uint16{0},
		ParamCount: 1,
	})

	grid := term.Grid()
	if grid[0][0].Char != ' ' {
		t.Errorf("expected ' ' at (0,0) after erase, got %c", grid[0][0].Char)
	}
	if grid[0][1].Char != ' ' {
		t.Errorf("expected ' ' at (0,1) after erase, got %c", grid[0][1].Char)
	}
}

func TestTerminalScrollUp(t *testing.T) {
	term := newTestTerminal(3, 10)

	// Fill all 3 rows
	for row := 0; row < 3; row++ {
		for col := 0; col < 10; col++ {
			term.Print(rune('A' + row))
		}
		if row < 2 {
			term.Execute(0x0A) // LF
			term.Execute(0x0D) // CR
		}
	}

	// This should scroll (cursor is at last row)
	term.Execute(0x0A)
	term.Execute(0x0D)
	term.Print('Z')

	grid := term.Grid()
	// Row 0 should now be what was row 1 ('B')
	if grid[0][0].Char != 'B' {
		t.Errorf("expected 'B' at (0,0) after scroll, got %c", grid[0][0].Char)
	}
	// Last row should be 'Z'
	if grid[2][0].Char != 'Z' {
		t.Errorf("expected 'Z' at (2,0), got %c", grid[2][0].Char)
	}
}

func TestTerminalSGR(t *testing.T) {
	term := newTestTerminal(24, 80)

	// SGR 1 = Bold
	term.CSIDispatch(parser.CSIDispatchAction{
		Final:      'm',
		Params:     [24]uint16{1},
		ParamCount: 1,
	})

	if !term.CurrentStyle.Bold {
		t.Error("expected Bold after SGR 1")
	}

	// SGR 31 = Red foreground
	term.CSIDispatch(parser.CSIDispatchAction{
		Final:      'm',
		Params:     [24]uint16{31},
		ParamCount: 1,
	})

	if !term.CurrentStyle.FG.Valid {
		t.Error("expected valid FG after SGR 31")
	}
	if term.CurrentStyle.FG.R != 205 {
		t.Errorf("expected FG R=205, got %d", term.CurrentStyle.FG.R)
	}

	// SGR 0 = Reset
	term.CSIDispatch(parser.CSIDispatchAction{
		Final:      'm',
		Params:     [24]uint16{0},
		ParamCount: 1,
	})

	if term.CurrentStyle.Bold {
		t.Error("expected not Bold after SGR 0")
	}
	if term.CurrentStyle.FG.Valid {
		t.Error("expected default FG after SGR 0")
	}
}

func TestTerminalSGR256Color(t *testing.T) {
	term := newTestTerminal(24, 80)

	term.CSIDispatch(parser.CSIDispatchAction{
		Final:      'm',
		Params:     [24]uint16{38, 5, 196},
		ParamCount: 3,
	})

	if got := term.CurrentStyle.FG; got.R != 255 || got.G != 0 || got.B != 0 || !got.Valid {
		t.Fatalf("expected palette color 196, got %+v", got)
	}
}

func TestTerminalREPRepeatsPreviousCharacter(t *testing.T) {
	term := newTestTerminal(24, 80)
	term.Print('A')

	done := make(chan struct{})
	go func() {
		term.CSIDispatch(parser.CSIDispatchAction{
			Final:      'b',
			Params:     [24]uint16{3},
			ParamCount: 1,
		})
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("REP dispatch deadlocked")
	}

	grid := term.Grid()
	for col := 0; col < 4; col++ {
		if grid[0][col].Char != 'A' {
			t.Fatalf("grid[0][%d] = %q, want 'A'", col, grid[0][col].Char)
		}
	}
}

func TestTerminalDeviceReportsRespond(t *testing.T) {
	term := newTestTerminal(24, 80)
	var responses [][]byte
	term.SetRespond(func(data []byte) {
		responses = append(responses, append([]byte(nil), data...))
	})

	term.CSIDispatch(parser.CSIDispatchAction{Final: 'c'})
	term.CSIDispatch(parser.CSIDispatchAction{
		Final:      'n',
		Params:     [24]uint16{5},
		ParamCount: 1,
	})
	term.CSIDispatch(parser.CSIDispatchAction{
		Final:      'H',
		Params:     [24]uint16{3, 4},
		ParamCount: 2,
	})
	term.CSIDispatch(parser.CSIDispatchAction{
		Final:      'n',
		Params:     [24]uint16{6},
		ParamCount: 1,
	})
	term.EscDispatch(parser.EscDispatchAction{Final: 'Z'})

	want := []string{"\x1b[?1;2c", "\x1b[0n", "\x1b[3;4R", "\x1b[?1;2c"}
	if len(responses) != len(want) {
		t.Fatalf("got %d responses, want %d: %q", len(responses), len(want), responses)
	}
	for i := range want {
		if string(responses[i]) != want[i] {
			t.Fatalf("response %d = %q, want %q", i, responses[i], want[i])
		}
	}
}

func TestTerminalSGRMouseMode(t *testing.T) {
	term := newTestTerminal(24, 80)

	term.CSIDispatch(parser.CSIDispatchAction{
		Final:      'h',
		Private:    true,
		Params:     [24]uint16{1000, 1006},
		ParamCount: 2,
	})

	if got := term.MouseMode(); got != input.MouseModeNormal {
		t.Fatalf("MouseMode() = %v, want normal", got)
	}
	if !term.MouseSGR() {
		t.Fatal("MouseSGR() = false, want true")
	}

	term.CSIDispatch(parser.CSIDispatchAction{
		Final:      'l',
		Private:    true,
		Params:     [24]uint16{1006},
		ParamCount: 1,
	})
	if term.MouseSGR() {
		t.Fatal("MouseSGR() stayed true after reset")
	}
}

func TestTerminalFocusEventsMode(t *testing.T) {
	term := newTestTerminal(24, 80)

	term.CSIDispatch(parser.CSIDispatchAction{
		Final:      'h',
		Private:    true,
		Params:     [24]uint16{1004},
		ParamCount: 1,
	})
	if !term.FocusEvents() {
		t.Fatal("FocusEvents() = false, want true")
	}

	term.CSIDispatch(parser.CSIDispatchAction{
		Final:      'l',
		Private:    true,
		Params:     [24]uint16{1004},
		ParamCount: 1,
	})
	if term.FocusEvents() {
		t.Fatal("FocusEvents() stayed true after reset")
	}
}

func TestTerminalScrollRegion(t *testing.T) {
	term := newTestTerminal(5, 10)

	// Set scroll region to rows 2-4
	term.CSIDispatch(parser.CSIDispatchAction{
		Final:      'r',
		Params:     [24]uint16{2, 4},
		ParamCount: 2,
	})

	if term.ScrollRegion.Top != 1 {
		t.Errorf("expected scroll top=1, got %d", term.ScrollRegion.Top)
	}
	if term.ScrollRegion.Bottom != 4 {
		t.Errorf("expected scroll bottom=4, got %d", term.ScrollRegion.Bottom)
	}
}

func TestTerminalOriginMode(t *testing.T) {
	term := newTestTerminal(24, 80)

	// Set scroll region
	term.CSIDispatch(parser.CSIDispatchAction{
		Final:      'r',
		Params:     [24]uint16{5, 20},
		ParamCount: 2,
	})

	// Enable origin mode
	term.active.Modes.SetDecMode(ModeDecOM)

	// CUP 1;1 in origin mode should be relative to scroll region
	term.CSIDispatch(parser.CSIDispatchAction{
		Final:      'H',
		Params:     [24]uint16{1, 1},
		ParamCount: 2,
	})

	if term.active.Cursor.Row != 4 { // scroll top (5-1) + 0
		t.Errorf("expected row=4 in origin mode, got %d", term.active.Cursor.Row)
	}
}

func TestTerminalBackspace(t *testing.T) {
	term := newTestTerminal(24, 80)
	term.Print('A')
	term.Print('B')
	term.Execute(0x08) // BS
	term.Print('C')

	grid := term.Grid()
	if grid[0][0].Char != 'A' {
		t.Errorf("expected 'A' at (0,0), got %c", grid[0][0].Char)
	}
	if grid[0][1].Char != 'C' {
		t.Errorf("expected 'C' at (0,1), got %c", grid[0][1].Char)
	}
}

func TestTerminalTab(t *testing.T) {
	term := newTestTerminal(24, 80)
	term.Execute(0x09) // HT

	if term.active.Cursor.Col != 8 {
		t.Errorf("expected col=8 after tab, got %d", term.active.Cursor.Col)
	}
}

func TestTerminalResize(t *testing.T) {
	term := newTestTerminal(24, 80)
	term.Print('A')
	term.Resize(30, 100)

	if term.Rows != 30 || term.Cols != 100 {
		t.Errorf("expected 30x100, got %dx%d", term.Rows, term.Cols)
	}
	if term.active.Cursor.Col != 1 {
		t.Errorf("expected cursor col=1 after resize, got %d", term.active.Cursor.Col)
	}
}

func TestTerminalOSC(t *testing.T) {
	term := newTestTerminal(24, 80)
	term.OSCDispatch(parser.OSCSetWindowTitle{Title: "Test Title"})

	if term.Title() != "Test Title" {
		t.Errorf("expected title 'Test Title', got '%s'", term.Title())
	}
}

func TestTerminalCSIIntermediate(t *testing.T) {
	term := newTestTerminal(24, 80)

	// DECSCUSR 2 = Steady block cursor
	term.CSIDispatch(parser.CSIDispatchAction{
		Final:             'q',
		Params:            [24]uint16{2},
		ParamCount:        1,
		Intermediates:     [4]byte{' '},
		IntermediateCount: 1,
	})

	if term.active.Cursor.Style != CursorSteadyBlock {
		t.Errorf("expected SteadyBlock cursor, got %d", term.active.Cursor.Style)
	}
}

func TestTerminalDecTextCursorVisibility(t *testing.T) {
	term := newTestTerminal(24, 80)

	_, _, visible, _ := term.Cursor()
	if !visible {
		t.Fatal("cursor should be visible by default")
	}

	term.CSIDispatch(parser.CSIDispatchAction{
		Final:      'l',
		Private:    true,
		Params:     [24]uint16{25},
		ParamCount: 1,
	})
	_, _, visible, _ = term.Cursor()
	if visible {
		t.Fatal("cursor stayed visible after DECRST ?25")
	}

	term.CSIDispatch(parser.CSIDispatchAction{
		Final:      'h',
		Private:    true,
		Params:     [24]uint16{25},
		ParamCount: 1,
	})
	_, _, visible, _ = term.Cursor()
	if !visible {
		t.Fatal("cursor stayed hidden after DECSET ?25")
	}
}

func TestTerminalInsertMode(t *testing.T) {
	term := newTestTerminal(24, 80)
	term.Print('A')
	term.Print('B')
	term.Print('C')

	// Enable insert mode
	term.active.Modes.SetMode(ModeInsert)

	// Move cursor to position 1
	term.Execute(0x0D) // CR
	term.cursorForward(1)
	term.Print('X')

	grid := term.Grid()
	if grid[0][0].Char != 'A' {
		t.Errorf("expected 'A' at (0,0), got %c", grid[0][0].Char)
	}
	if grid[0][1].Char != 'X' {
		t.Errorf("expected 'X' at (0,1), got %c", grid[0][1].Char)
	}
	if grid[0][2].Char != 'B' {
		t.Errorf("expected 'B' at (0,2) (shifted), got %c", grid[0][2].Char)
	}
}

func TestTerminalDECSCSavesAndRestoresCursor(t *testing.T) {
	term := newTestTerminal(24, 80)
	term.CSIDispatch(parser.CSIDispatchAction{
		Final:      'H',
		Params:     [24]uint16{5, 10},
		ParamCount: 2,
	})
	term.EscDispatch(parser.EscDispatchAction{Final: '7'})

	term.CSIDispatch(parser.CSIDispatchAction{
		Final:      'H',
		Params:     [24]uint16{9, 20},
		ParamCount: 2,
	})
	term.EscDispatch(parser.EscDispatchAction{Final: '8'})

	row, col, _, _ := term.Cursor()
	if row != 4 || col != 9 {
		t.Fatalf("cursor = %d,%d, want 4,9", row, col)
	}
}

func TestTerminalSCPSavesAndRestoresCursor(t *testing.T) {
	term := newTestTerminal(24, 80)
	term.CSIDispatch(parser.CSIDispatchAction{
		Final:      'H',
		Params:     [24]uint16{3, 4},
		ParamCount: 2,
	})
	term.CSIDispatch(parser.CSIDispatchAction{Final: 's'})

	term.CSIDispatch(parser.CSIDispatchAction{
		Final:      'H',
		Params:     [24]uint16{8, 9},
		ParamCount: 2,
	})
	term.CSIDispatch(parser.CSIDispatchAction{Final: 'u'})

	row, col, _, _ := term.Cursor()
	if row != 2 || col != 3 {
		t.Fatalf("cursor = %d,%d, want 2,3", row, col)
	}
}

func TestTerminalAltScreen1049RestoresPrimaryCursor(t *testing.T) {
	term := newTestTerminal(24, 80)
	term.CSIDispatch(parser.CSIDispatchAction{
		Final:      'H',
		Params:     [24]uint16{6, 7},
		ParamCount: 2,
	})

	term.CSIDispatch(parser.CSIDispatchAction{
		Final:      'h',
		Private:    true,
		Params:     [24]uint16{1049},
		ParamCount: 1,
	})
	if term.active != term.alternate {
		t.Fatal("expected alternate screen")
	}
	if !term.primary.SavedCursor.Valid || term.primary.SavedCursor.Row != 5 || term.primary.SavedCursor.Col != 6 {
		t.Fatalf("primary saved cursor = %+v, want row=5 col=6 valid", term.primary.SavedCursor)
	}
	term.CSIDispatch(parser.CSIDispatchAction{
		Final:      'H',
		Params:     [24]uint16{2, 2},
		ParamCount: 2,
	})

	term.CSIDispatch(parser.CSIDispatchAction{
		Final:      'l',
		Private:    true,
		Params:     [24]uint16{1049},
		ParamCount: 1,
	})

	if term.active != term.primary {
		t.Fatal("expected primary screen")
	}
	row, col, _, _ := term.Cursor()
	if row != 5 || col != 6 {
		t.Fatalf("cursor = %d,%d, want 5,6", row, col)
	}
}

func TestTerminalAltScreen47SwitchesBuffers(t *testing.T) {
	term := newTestTerminal(24, 80)
	term.Print('P')

	term.CSIDispatch(parser.CSIDispatchAction{
		Final:      'h',
		Private:    true,
		Params:     [24]uint16{47},
		ParamCount: 1,
	})
	if term.active != term.alternate {
		t.Fatal("expected alternate screen after DECSET 47")
	}
	term.Print('A')
	if grid := term.Grid(); grid[0][0].Char != 'A' {
		t.Fatalf("alternate grid[0][0] = %q, want 'A'", grid[0][0].Char)
	}

	term.CSIDispatch(parser.CSIDispatchAction{
		Final:      'l',
		Private:    true,
		Params:     [24]uint16{47},
		ParamCount: 1,
	})
	if term.active != term.primary {
		t.Fatal("expected primary screen after DECRST 47")
	}
	if grid := term.Grid(); grid[0][0].Char != 'P' {
		t.Fatalf("primary grid[0][0] = %q, want 'P'", grid[0][0].Char)
	}
}

func TestTerminalDecSaveCursor1048RestoresWithoutAltScreen(t *testing.T) {
	term := newTestTerminal(24, 80)
	term.CSIDispatch(parser.CSIDispatchAction{
		Final:      'H',
		Params:     [24]uint16{4, 5},
		ParamCount: 2,
	})

	term.CSIDispatch(parser.CSIDispatchAction{
		Final:      'h',
		Private:    true,
		Params:     [24]uint16{1048},
		ParamCount: 1,
	})
	term.CSIDispatch(parser.CSIDispatchAction{
		Final:      'H',
		Params:     [24]uint16{9, 10},
		ParamCount: 2,
	})
	term.CSIDispatch(parser.CSIDispatchAction{
		Final:      'l',
		Private:    true,
		Params:     [24]uint16{1048},
		ParamCount: 1,
	})

	if term.active != term.primary {
		t.Fatal("expected 1048 to leave active screen unchanged")
	}
	row, col, _, _ := term.Cursor()
	if row != 3 || col != 4 {
		t.Fatalf("cursor = %d,%d, want 3,4", row, col)
	}
}

func TestTerminalAltScreen1049StartsFreshEachEntry(t *testing.T) {
	term := newTestTerminal(24, 80)

	term.CSIDispatch(parser.CSIDispatchAction{
		Final:      'h',
		Private:    true,
		Params:     [24]uint16{1049},
		ParamCount: 1,
	})
	term.Print('X')
	term.CSIDispatch(parser.CSIDispatchAction{
		Final:      'l',
		Private:    true,
		Params:     [24]uint16{1049},
		ParamCount: 1,
	})

	term.CSIDispatch(parser.CSIDispatchAction{
		Final:      'h',
		Private:    true,
		Params:     [24]uint16{1049},
		ParamCount: 1,
	})

	grid := term.Grid()
	if grid[0][0].Char != ' ' {
		t.Fatalf("alternate screen retained stale char %q, want blank", grid[0][0].Char)
	}
}

func TestTerminalAltScreenDoesNotCollectScrollback(t *testing.T) {
	term := newTestTerminal(2, 5)

	term.CSIDispatch(parser.CSIDispatchAction{
		Final:      'h',
		Private:    true,
		Params:     [24]uint16{1049},
		ParamCount: 1,
	})
	for _, ch := range []rune{'A', '\n', '\r', 'B', '\n', '\r', 'C'} {
		if ch == '\n' {
			term.Execute(0x0A)
			continue
		}
		if ch == '\r' {
			term.Execute(0x0D)
			continue
		}
		term.Print(ch)
	}

	if got := term.ScrollbackLen(); got != 0 {
		t.Fatalf("alternate scrollback len = %d, want 0", got)
	}
}

// TestStreamEndToEnd tests the full pipeline: raw bytes -> parser -> terminal
func TestStreamEndToEnd(t *testing.T) {
	term := newTestTerminal(24, 80)
	stream := NewStream(term)

	// Send "Hello" + newline + colored "World"
	stream.Process([]byte("Hello\r\n\x1b[31mWorld\x1b[0m"))

	grid := term.Grid()
	if grid[0][0].Char != 'H' {
		t.Errorf("expected 'H' at (0,0), got %c", grid[0][0].Char)
	}
	if grid[1][0].Char != 'W' {
		t.Errorf("expected 'W' at (1,0), got %c", grid[1][0].Char)
	}

	// The 'W' should have red foreground from SGR 31
	wStyle := term.active.Styles.Lookup(grid[1][0].Style)
	if wStyle.FG.R != 205 {
		t.Errorf("expected red FG for 'W', got R=%d", wStyle.FG.R)
	}

	// 'o' should also be red (SGR 0 comes after all of "World")
	oStyle := term.active.Styles.Lookup(grid[1][1].Style)
	if oStyle.FG.R != 205 {
		t.Errorf("expected red FG for 'o', got R=%d", oStyle.FG.R)
	}
}

func TestStreamPrintsUTF8Runes(t *testing.T) {
	term := newTestTerminal(24, 80)
	stream := NewStream(term)

	stream.Process([]byte("é中"))

	grid := term.Grid()
	if grid[0][0].Char != 'é' {
		t.Fatalf("grid[0][0] = %q, want 'é'", grid[0][0].Char)
	}
	if grid[0][1].Char != '中' {
		t.Fatalf("grid[0][1] = %q, want '中'", grid[0][1].Char)
	}
	if grid[0][2].Char != 0 {
		t.Fatalf("wide character spacer = %q, want zero-width spacer", grid[0][2].Char)
	}
}

func TestStreamKeepsSplitUTF8RuneAcrossProcessCalls(t *testing.T) {
	term := newTestTerminal(24, 80)
	stream := NewStream(term)

	stream.Process([]byte{0xE4})
	stream.Process([]byte{0xB8})
	stream.Process([]byte{0xAD})

	grid := term.Grid()
	if grid[0][0].Char != '中' {
		t.Fatalf("grid[0][0] = %q, want '中'", grid[0][0].Char)
	}
}

func TestStreamCSICursorUp(t *testing.T) {
	term := newTestTerminal(24, 80)
	stream := NewStream(term)

	// Move down 5 lines, then up 2
	stream.Process([]byte("\x1b[5B\x1b[2A"))

	if term.active.Cursor.Row != 3 {
		t.Errorf("expected row=3 after 5 down 2 up, got %d", term.active.Cursor.Row)
	}
}

func TestStreamAltScreen(t *testing.T) {
	term := newTestTerminal(24, 80)
	stream := NewStream(term)

	stream.Process([]byte("Hello"))
	if term.active != term.primary {
		t.Error("expected primary screen")
	}

	// Switch to alt screen (xterm 1049)
	stream.Process([]byte("\x1b[?1049h"))
	if term.active != term.alternate {
		t.Error("expected alternate screen after DECSET 1049")
	}

	// Switch back
	stream.Process([]byte("\x1b[?1049l"))
	if term.active != term.primary {
		t.Error("expected primary screen after DECRST 1049")
	}
}
