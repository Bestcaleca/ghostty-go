// Package terminal implements the terminal state machine that processes VT escape
// sequences and maintains the terminal display state.
package terminal

import (
	"encoding/base64"
	"fmt"
	"sync"

	"github.com/ghostty-go/ghostty-go/input"
	"github.com/ghostty-go/ghostty-go/parser"
	"github.com/ghostty-go/ghostty-go/unicode"
)

// ScrollRegion defines the scroll region boundaries.
type ScrollRegion struct {
	Top    int
	Bottom int // exclusive
}

// Terminal represents the complete terminal state.
type Terminal struct {
	primary   *Screen
	alternate *Screen
	active    *Screen // points to primary or alternate

	Rows, Cols   int
	WidthPx      int
	HeightPx     int
	ScrollRegion ScrollRegion
	PreviousChar rune

	// Current SGR state (applied to new characters)
	CurrentStyle Style

	// Current hyperlink (OSC 8)
	currentHyperlink string

	// Callbacks
	clipboardWrite func(clipboard string, data []byte) // write to system clipboard
	clipboardRead  func(clipboard string) []byte       // read from system clipboard
	respond        func(data []byte)                   // send data back to shell
	bell           func()                              // terminal bell

	mu sync.RWMutex
}

// New creates a new Terminal with the given dimensions.
func New(rows, cols int) *Terminal {
	primary := NewScreen(rows, cols)
	t := &Terminal{
		primary:      primary,
		alternate:    nil, // created lazily on first switch
		active:       primary,
		Rows:         rows,
		Cols:         cols,
		ScrollRegion: ScrollRegion{Top: 0, Bottom: rows},
	}
	return t
}

// SetClipboardWrite sets the callback for writing to the system clipboard.
func (t *Terminal) SetClipboardWrite(fn func(clipboard string, data []byte)) {
	t.clipboardWrite = fn
}

// SetClipboardRead sets the callback for reading from the system clipboard.
func (t *Terminal) SetClipboardRead(fn func(clipboard string) []byte) {
	t.clipboardRead = fn
}

// SetRespond sets the callback for sending data back to the shell (e.g., clipboard query responses).
func (t *Terminal) SetRespond(fn func(data []byte)) {
	t.respond = fn
}

// SetBell sets the callback for terminal bell.
func (t *Terminal) SetBell(fn func()) {
	t.bell = fn
}

// BracketedPaste returns true if bracketed paste mode is enabled.
func (t *Terminal) BracketedPaste() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.active.Modes.DecBracketedPaste
}

// ScrollbackLen returns the number of lines in the scrollback buffer.
func (t *Terminal) ScrollbackLen() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return len(t.active.Scrollback)
}

// ScrollbackRows returns the scrollback buffer rows.
func (t *Terminal) ScrollbackRows() []Row {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.active.Scrollback
}

// SetMaxScroll sets the maximum number of scrollback lines.
func (t *Terminal) SetMaxScroll(n int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.primary.MaxScroll = n
	if t.alternate != nil {
		t.alternate.MaxScroll = n
	}
}

// MouseMode returns the current mouse tracking mode.
func (t *Terminal) MouseMode() input.MouseMode {
	t.mu.RLock()
	defer t.mu.RUnlock()

	m := t.active.Modes
	if m.DecMouseAny {
		return input.MouseModeAny
	}
	if m.DecMouseButton {
		return input.MouseModeButton
	}
	if m.DecMouseNormal {
		return input.MouseModeNormal
	}
	if m.DecMouseX10 {
		return input.MouseModeX10
	}
	return input.MouseModeNone
}

// MouseSGR returns whether xterm SGR extended mouse encoding is enabled.
func (t *Terminal) MouseSGR() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.active.Modes.DecMouseSGR
}

// FocusEvents returns whether focus in/out reporting is enabled.
func (t *Terminal) FocusEvents() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.active.Modes.DecFocusEvents
}

// SelectionStart begins a text selection.
func (t *Terminal) SelectionStart(row, col int, mode SelectionMode) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.active.Selection.Start(row, col, mode)
}

// SelectionUpdate extends the selection.
func (t *Terminal) SelectionUpdate(row, col int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	sel := &t.active.Selection
	sel.Update(row, col)

	// For word selection, snap to word boundaries
	if sel.SelectionMode == SelectionWord && sel.Active {
		cells := t.active.Rows
		if row >= 0 && row < len(cells) {
			start, end := findWordBoundaries(cells[row].Cells, col)
			if row == sel.StartRow {
				sel.StartCol = start
			}
			sel.EndCol = end
		}
	}

	// For line selection, select full line
	if sel.SelectionMode == SelectionLine && sel.Active {
		sel.StartCol = 0
		sel.EndCol = t.Cols
	}
}

// SelectionClear clears the selection.
func (t *Terminal) SelectionClear() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.active.Selection.Clear()
}

// SelectionIsActive returns whether a selection is active.
func (t *Terminal) SelectionIsActive() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.active.Selection.Active
}

// SelectionIsSelected returns true if the given cell is within the selection.
func (t *Terminal) SelectionIsSelected(row, col int) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.active.Selection.IsSelected(row, col)
}

// Resize resizes the terminal.
func (t *Terminal) Resize(rows, cols int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.Rows = rows
	t.Cols = cols
	t.ScrollRegion = ScrollRegion{Top: 0, Bottom: rows}
	t.primary.Resize(rows, cols)
	if t.alternate != nil {
		t.alternate.Resize(rows, cols)
	}
}

// Active returns the currently active screen.
func (t *Terminal) Active() *Screen {
	return t.active
}

// Grid returns the current display grid for rendering (read-only snapshot).
func (t *Terminal) Grid() [][]Cell {
	t.mu.RLock()
	defer t.mu.RUnlock()

	result := make([][]Cell, len(t.active.Rows))
	for i, row := range t.active.Rows {
		result[i] = make([]Cell, len(row.Cells))
		copy(result[i], row.Cells)
	}
	return result
}

// Cursor returns the current cursor position and style.
func (t *Terminal) Cursor() (row, col int, visible bool, style CursorStyle) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	c := t.active.Cursor
	return c.Row, c.Col, c.Visible, c.Style
}

// Title returns the current window title.
func (t *Terminal) Title() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.active.Title
}

// --- StreamHandler interface ---

// Print handles a printable character.
func (t *Terminal) Print(ch rune) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.printLocked(ch)
}

func (t *Terminal) printLocked(ch rune) {
	s := t.active
	cols := s.Width()

	// Map through charset if needed
	if s.Charset.GetActiveCharset() == CharsetDecSpecial {
		if mapped := MapDecSpecialGraphics(byte(ch)); mapped != 0 {
			ch = mapped
		}
	}

	// Handle pending wrap
	if s.Cursor.Col >= cols {
		s.Cursor.Col = 0
		if s.Cursor.Row < t.ScrollRegion.Bottom-1 {
			s.Cursor.Row++
		} else {
			s.ScrollUp(t.ScrollRegion.Top, t.ScrollRegion.Bottom, 1)
		}
	}

	row := s.Cursor.Row
	col := s.Cursor.Col

	// Clamp to valid range
	if row < 0 || row >= len(s.Rows) {
		return
	}
	if col < 0 || col >= len(s.Rows[row].Cells) {
		return
	}

	// Determine character width
	width := CharWidth(ch)
	if width == 2 && col == cols-1 && s.Modes.DecAWM {
		s.Cursor.Col = 0
		if s.Cursor.Row < t.ScrollRegion.Bottom-1 {
			s.Cursor.Row++
		} else {
			s.ScrollUp(t.ScrollRegion.Top, t.ScrollRegion.Bottom, 1)
		}
		row = s.Cursor.Row
		col = s.Cursor.Col
	}

	// In insert mode, shift cells right
	if s.Modes.Insert && width > 0 {
		shiftRight(s.Rows[row].Cells, col, width, s.Styles.DefaultID())
	}

	// Write the character
	styleID := s.Styles.Get(t.CurrentStyle)
	s.Rows[row].Cells[col] = Cell{
		Char:      ch,
		Width:     uint8(width),
		Style:     styleID,
		Hyperlink: t.currentHyperlink,
	}

	// Handle wide characters: mark the next cell as spacer
	if width == 2 && col+1 < cols {
		s.Rows[row].Cells[col+1] = Cell{
			Char:  0,
			Width: 0, // spacer tail
			Style: styleID,
		}
	}

	// Advance cursor
	s.Cursor.Col += width
	if s.Cursor.Col >= cols && s.Modes.DecAWM {
		// Pending wrap
		s.Cursor.Col = cols // will wrap on next print
	} else if s.Cursor.Col >= cols {
		s.Cursor.Col = cols - 1
	}

	t.PreviousChar = ch
}

// Execute handles a C0 control character.
func (t *Terminal) Execute(b byte) {
	t.mu.Lock()
	defer t.mu.Unlock()

	switch b {
	case 0x07: // BEL - bell
		if t.bell != nil {
			t.bell()
		}
	case 0x08: // BS - backspace
		t.backspace()
	case 0x09: // HT - horizontal tab
		t.horizontalTab()
	case 0x0A: // LF - line feed
		t.lineFeed()
	case 0x0B: // VT - vertical tab (treated as LF)
		t.lineFeed()
	case 0x0C: // FF - form feed (treated as LF)
		t.lineFeed()
	case 0x0D: // CR - carriage return
		t.active.Cursor.Col = 0
	case 0x0E: // SO - shift out (invoke G1)
		t.active.Charset.GL = 1
	case 0x0F: // SI - shift in (invoke G0)
		t.active.Charset.GL = 0
	}
}

// CSIDispatch handles a CSI sequence.
func (t *Terminal) CSIDispatch(a parser.CSIDispatchAction) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if a.Private {
		t.handlePrivateCSI(a)
		return
	}

	switch a.Final {
	case '@': // ICH - Insert Character
		n := int(a.Param(1, 1))
		t.insertChars(n)
	case 'A': // CUU - Cursor Up
		n := int(a.Param(1, 1))
		t.cursorUp(n)
	case 'B': // CUD - Cursor Down
		n := int(a.Param(1, 1))
		t.cursorDown(n)
	case 'C': // CUF - Cursor Forward
		n := int(a.Param(1, 1))
		t.cursorForward(n)
	case 'D': // CUB - Cursor Backward
		n := int(a.Param(1, 1))
		t.cursorBackward(n)
	case 'E': // CNL - Cursor Next Line
		n := int(a.Param(1, 1))
		t.active.Cursor.Col = 0
		t.cursorDown(n)
	case 'F': // CPL - Cursor Previous Line
		n := int(a.Param(1, 1))
		t.active.Cursor.Col = 0
		t.cursorUp(n)
	case 'G': // CHA - Cursor Horizontal Absolute
		n := int(a.Param(1, 1))
		t.active.Cursor.Col = clamp(n-1, 0, t.Cols-1)
	case 'H': // CUP - Cursor Position
		row := int(a.Param(1, 1)) - 1
		col := int(a.Param(2, 1)) - 1
		t.setCursor(row, col)
	case 'I': // CHT - Cursor Horizontal Tab
		n := int(a.Param(1, 1))
		for i := 0; i < n; i++ {
			t.active.Cursor.Col = t.active.Tabstops.Next(t.active.Cursor.Col)
		}
	case 'J': // ED - Erase in Display
		t.eraseDisplay(int(a.Param(1, 0)))
	case 'K': // EL - Erase in Line
		t.eraseLine(int(a.Param(1, 0)))
	case 'L': // IL - Insert Line
		n := int(a.Param(1, 1))
		t.insertLines(n)
	case 'M': // DL - Delete Line
		n := int(a.Param(1, 1))
		t.deleteLines(n)
	case 'P': // DCH - Delete Character
		n := int(a.Param(1, 1))
		t.deleteChars(n)
	case 'S': // SU - Scroll Up
		n := int(a.Param(1, 1))
		t.active.ScrollUp(t.ScrollRegion.Top, t.ScrollRegion.Bottom, n)
	case 'T': // SD - Scroll Down
		n := int(a.Param(1, 1))
		t.active.ScrollDown(t.ScrollRegion.Top, t.ScrollRegion.Bottom, n)
	case 'X': // ECH - Erase Character
		n := int(a.Param(1, 1))
		t.eraseChars(n)
	case 'Z': // CBT - Cursor Backward Tab
		n := int(a.Param(1, 1))
		for i := 0; i < n; i++ {
			t.active.Cursor.Col = t.active.Tabstops.Prev(t.active.Cursor.Col)
		}
	case '`': // HPA - Horizontal Position Absolute
		n := int(a.Param(1, 1))
		t.active.Cursor.Col = clamp(n-1, 0, t.Cols-1)
	case 'a': // HPR - Horizontal Position Relative
		n := int(a.Param(1, 1))
		t.active.Cursor.Col = clamp(t.active.Cursor.Col+n, 0, t.Cols-1)
	case 'b': // REP - Repeat Character
		n := int(a.Param(1, 1))
		for i := 0; i < n; i++ {
			t.printLocked(t.PreviousChar)
		}
	case 'c': // DA - Device Attributes
		t.respondPrimaryDeviceAttributes()
	case 'd': // VPA - Vertical Position Absolute
		n := int(a.Param(1, 1))
		t.setCursor(n-1, t.active.Cursor.Col)
	case 'f': // HVP - Horizontal Vertical Position
		row := int(a.Param(1, 1)) - 1
		col := int(a.Param(2, 1)) - 1
		t.setCursor(row, col)
	case 'g': // TBC - Tab Clear
		t.clearTab(int(a.Param(1, 0)))
	case 'h': // SM - Set Mode
		t.setMode(a)
	case 'l': // RM - Reset Mode
		t.resetMode(a)
	case 'm': // SGR - Select Graphic Rendition
		t.handleSGR(a)
	case 'n': // DSR - Device Status Report
		t.respondDeviceStatus(int(a.Param(1, 0)))
	case 'q': // DECSCUSR - Set Cursor Style
		t.setCursorStyle(int(a.Param(1, 0)))
	case 'r': // DECSTBM - Set Scrolling Region
		top := int(a.Param(1, 1)) - 1
		bottom := int(a.Param(2, uint16(t.Rows)))
		t.setScrollRegion(top, bottom)
	case 's': // SCP - Save Cursor Position
		t.saveCursor()
	case 't': // Window manipulation (xterm)
		// TODO: handle window ops
	case 'u': // RCP - Restore Cursor Position
		t.restoreCursor()
	}
}

// handlePrivateCSI handles CSI sequences with '?' prefix (DEC private modes).
func (t *Terminal) handlePrivateCSI(a parser.CSIDispatchAction) {
	switch a.Final {
	case 'h': // DECSET
		for i := 1; i <= a.ParamCount; i++ {
			mode := Mode(a.Params[i-1])
			t.setDecModeWithAction(mode)
		}
	case 'l': // DECRST
		for i := 1; i <= a.ParamCount; i++ {
			mode := Mode(a.Params[i-1])
			t.resetDecModeWithAction(mode)
		}
	case 'r': // Restore DEC private mode
		// TODO
	case 's': // Save DEC private mode
		// TODO
	}
}

// EscDispatch handles an ESC sequence.
func (t *Terminal) EscDispatch(a parser.EscDispatchAction) {
	t.mu.Lock()
	defer t.mu.Unlock()

	switch a.Final {
	case '7': // DECSC - Save Cursor
		t.saveCursor()
	case '8': // DECRC - Restore Cursor
		t.restoreCursor()
	case 'D': // IND - Index (move down, scroll if at bottom)
		t.index()
	case 'E': // NEL - Next Line
		t.active.Cursor.Col = 0
		t.index()
	case 'H': // HTS - Horizontal Tab Set
		t.active.Tabstops.Set(t.active.Cursor.Col)
	case 'M': // RI - Reverse Index
		t.reverseIndex()
	case 'N': // SS2 - Single Shift 2
		t.active.Charset.SingleShift = 2
	case 'O': // SS3 - Single Shift 3
		t.active.Charset.SingleShift = 3
	case 'Z': // DECID - Identify Terminal
		t.respondPrimaryDeviceAttributes()
	case '=': // DECKPAM - Application Keypad Mode
		// TODO
	case '>': // DECKPNM - Normal Keypad Mode
		// TODO
	case 'c': // RIS - Full Reset
		t.fullReset()
	case 'n': // LS2 - Locking Shift 2
		t.active.Charset.GL = 2
	case 'o': // LS3 - Locking Shift 3
		t.active.Charset.GL = 3
	case '|': // LS3R - Locking Shift 3 Right
		t.active.Charset.GR = 3
	case '}': // LS2R - Locking Shift 2 Right
		t.active.Charset.GR = 2
	case '~': // LS1R - Locking Shift 1 Right
		t.active.Charset.GR = 1
	}

	// Handle charset designations (intermediate byte determines slot)
	if a.IntermediateCount > 0 {
		switch a.Intermediates[0] {
		case '(':
			t.active.Charset.G0 = mapCharsetFinal(a.Final)
		case ')':
			t.active.Charset.G1 = mapCharsetFinal(a.Final)
		case '*':
			t.active.Charset.G2 = mapCharsetFinal(a.Final)
		case '+':
			t.active.Charset.G3 = mapCharsetFinal(a.Final)
		}
	}
}

func (t *Terminal) respondPrimaryDeviceAttributes() {
	t.respondBytes([]byte("\x1b[?1;2c"))
}

func (t *Terminal) respondDeviceStatus(kind int) {
	switch kind {
	case 5:
		t.respondBytes([]byte("\x1b[0n"))
	case 6:
		response := fmt.Sprintf("\x1b[%d;%dR", t.active.Cursor.Row+1, t.active.Cursor.Col+1)
		t.respondBytes([]byte(response))
	}
}

func (t *Terminal) respondBytes(data []byte) {
	if t.respond != nil {
		t.respond(data)
	}
}

// OSCDispatch handles an OSC command.
func (t *Terminal) OSCDispatch(cmd parser.OSCCommand) {
	t.mu.Lock()
	defer t.mu.Unlock()

	switch c := cmd.(type) {
	case parser.OSCSetWindowTitle:
		t.active.Title = c.Title
	case parser.OSCSetIconName:
		// ignore
	case parser.OSCSetHyperlink:
		if c.URI == "" {
			// End hyperlink
			t.currentHyperlink = ""
		} else {
			// Start hyperlink
			t.currentHyperlink = c.URI
		}
	case parser.OSCSetClipboard:
		t.handleClipboard(c)
	}
}

// handleClipboard handles OSC 52 clipboard operations.
func (t *Terminal) handleClipboard(c parser.OSCSetClipboard) {
	if c.Query {
		// Query: request clipboard content
		if t.clipboardRead != nil {
			data := t.clipboardRead(c.Clipboard)
			if len(data) > 0 {
				// Send response: OSC 52 ; clipboard ; base64data ST
				encoded := base64Encode(data)
				response := fmt.Sprintf("\x1b]52;%s;%s\x1b\\", c.Clipboard, encoded)
				if t.respond != nil {
					t.respond([]byte(response))
				}
			}
		}
		return
	}

	// Set: write to clipboard
	if len(c.Data) > 0 && t.clipboardWrite != nil {
		decoded := base64Decode(c.Data)
		t.clipboardWrite(c.Clipboard, decoded)
	}
}

// DCSHook starts a DCS sequence.
func (t *Terminal) DCSHook(a parser.DCSHookAction) {
	// TODO: implement DCS passthrough (tmux, sixel, etc.)
}

// DCSPut handles a DCS data byte.
func (t *Terminal) DCSPut(b byte) {
	// TODO
}

// DCSUnhook ends a DCS sequence.
func (t *Terminal) DCSUnhook() {
	// TODO
}

// --- Helper methods ---

func (t *Terminal) backspace() {
	if t.active.Cursor.Col > 0 {
		t.active.Cursor.Col--
	}
}

func (t *Terminal) horizontalTab() {
	t.active.Cursor.Col = t.active.Tabstops.Next(t.active.Cursor.Col)
	if t.active.Cursor.Col >= t.Cols {
		t.active.Cursor.Col = t.Cols - 1
	}
}

func (t *Terminal) lineFeed() {
	s := t.active
	if s.Cursor.Row == t.ScrollRegion.Bottom-1 {
		s.ScrollUp(t.ScrollRegion.Top, t.ScrollRegion.Bottom, 1)
	} else if s.Cursor.Row < len(s.Rows)-1 {
		s.Cursor.Row++
	}
	// LNM mode: LF also does CR
	if s.Modes.LNM {
		s.Cursor.Col = 0
	}
}

func (t *Terminal) index() {
	s := t.active
	if s.Cursor.Row == t.ScrollRegion.Bottom-1 {
		s.ScrollUp(t.ScrollRegion.Top, t.ScrollRegion.Bottom, 1)
	} else if s.Cursor.Row < len(s.Rows)-1 {
		s.Cursor.Row++
	}
}

func (t *Terminal) reverseIndex() {
	s := t.active
	if s.Cursor.Row == t.ScrollRegion.Top {
		s.ScrollDown(t.ScrollRegion.Top, t.ScrollRegion.Bottom, 1)
	} else if s.Cursor.Row > 0 {
		s.Cursor.Row--
	}
}

func (t *Terminal) cursorUp(n int) {
	s := t.active
	top := t.ScrollRegion.Top
	if s.Cursor.Row-n >= top {
		s.Cursor.Row -= n
	} else {
		s.Cursor.Row = top
	}
}

func (t *Terminal) cursorDown(n int) {
	s := t.active
	bottom := t.ScrollRegion.Bottom - 1
	if s.Cursor.Row+n <= bottom {
		s.Cursor.Row += n
	} else {
		s.Cursor.Row = bottom
	}
}

func (t *Terminal) cursorForward(n int) {
	s := t.active
	s.Cursor.Col = clamp(s.Cursor.Col+n, 0, t.Cols-1)
}

func (t *Terminal) cursorBackward(n int) {
	s := t.active
	s.Cursor.Col = clamp(s.Cursor.Col-n, 0, t.Cols-1)
}

func (t *Terminal) setCursor(row, col int) {
	s := t.active
	if s.Modes.DecOM {
		// Origin mode: coordinates relative to scroll region
		row += t.ScrollRegion.Top
	}
	s.Cursor.Row = clamp(row, t.ScrollRegion.Top, t.ScrollRegion.Bottom-1)
	s.Cursor.Col = clamp(col, 0, t.Cols-1)
}

func (t *Terminal) eraseDisplay(mode int) {
	s := t.active
	row := s.Cursor.Row
	col := s.Cursor.Col
	style := s.Styles.DefaultID()

	switch mode {
	case 0: // Erase below
		t.eraseLine(0) // rest of current line
		for i := row + 1; i < len(s.Rows); i++ {
			s.Rows[i].Reset(style)
		}
	case 1: // Erase above
		for i := 0; i < row; i++ {
			s.Rows[i].Reset(style)
		}
		for c := 0; c <= col; c++ {
			s.Rows[row].Cells[c] = EmptyCell(style)
		}
	case 2: // Erase all
		for i := range s.Rows {
			s.Rows[i].Reset(style)
		}
	case 3: // Erase scrollback
		s.Scrollback = nil
	}
}

func (t *Terminal) eraseLine(mode int) {
	s := t.active
	row := s.Cursor.Row
	col := s.Cursor.Col
	style := s.Styles.DefaultID()

	if row < 0 || row >= len(s.Rows) {
		return
	}

	switch mode {
	case 0: // Erase right
		for i := col; i < len(s.Rows[row].Cells); i++ {
			s.Rows[row].Cells[i] = EmptyCell(style)
		}
	case 1: // Erase left
		for i := 0; i <= col; i++ {
			s.Rows[row].Cells[i] = EmptyCell(style)
		}
	case 2: // Erase entire line
		s.Rows[row].Reset(style)
	}
}

func (t *Terminal) eraseChars(n int) {
	s := t.active
	row := s.Cursor.Row
	col := s.Cursor.Col
	style := s.Styles.DefaultID()

	for i := 0; i < n && col+i < len(s.Rows[row].Cells); i++ {
		s.Rows[row].Cells[col+i] = EmptyCell(style)
	}
}

func (t *Terminal) insertChars(n int) {
	s := t.active
	row := s.Cursor.Row
	col := s.Cursor.Col
	if row < 0 || row >= len(s.Rows) {
		return
	}
	shiftRight(s.Rows[row].Cells, col, n, s.Styles.DefaultID())
}

func (t *Terminal) deleteChars(n int) {
	s := t.active
	row := s.Cursor.Row
	col := s.Cursor.Col
	if row < 0 || row >= len(s.Rows) {
		return
	}
	cells := s.Rows[row].Cells
	cols := len(cells)
	for i := col; i < cols; i++ {
		if i+n < cols {
			cells[i] = cells[i+n]
		} else {
			cells[i] = EmptyCell(s.Styles.DefaultID())
		}
	}
}

func (t *Terminal) insertLines(n int) {
	s := t.active
	row := s.Cursor.Row
	if row < t.ScrollRegion.Top || row >= t.ScrollRegion.Bottom {
		return
	}
	s.ScrollDown(row, t.ScrollRegion.Bottom, n)
}

func (t *Terminal) deleteLines(n int) {
	s := t.active
	row := s.Cursor.Row
	if row < t.ScrollRegion.Top || row >= t.ScrollRegion.Bottom {
		return
	}
	s.ScrollUp(row, t.ScrollRegion.Bottom, n)
}

func (t *Terminal) clearTab(mode int) {
	switch mode {
	case 0: // Clear tab at current column
		t.active.Tabstops.Clear(t.active.Cursor.Col)
	case 3: // Clear all tabs
		t.active.Tabstops.ClearAll()
	}
}

func (t *Terminal) setScrollRegion(top, bottom int) {
	if top < 0 {
		top = 0
	}
	if bottom > t.Rows {
		bottom = t.Rows
	}
	if top >= bottom {
		return
	}
	t.ScrollRegion = ScrollRegion{Top: top, Bottom: bottom}
	// Move cursor to home position
	t.active.Cursor.Row = 0
	t.active.Cursor.Col = 0
}

func (t *Terminal) setCursorStyle(mode int) {
	switch mode {
	case 0:
		t.active.Cursor.Style = CursorDefault
	case 1:
		t.active.Cursor.Style = CursorBlinkingBlock
	case 2:
		t.active.Cursor.Style = CursorSteadyBlock
	case 3:
		t.active.Cursor.Style = CursorBlinkingUnderline
	case 4:
		t.active.Cursor.Style = CursorSteadyUnderline
	case 5:
		t.active.Cursor.Style = CursorBlinkingBar
	case 6:
		t.active.Cursor.Style = CursorSteadyBar
	}
}

func (t *Terminal) saveCursor() {
	s := t.active
	s.SavedCursor = s.Cursor.Save(s.Charset, s.Modes.DecOM, s.Modes.DecAWM)
}

func (t *Terminal) restoreCursor() {
	s := t.active
	if !s.SavedCursor.Valid {
		return
	}
	s.Cursor.Restore(s.SavedCursor)
	s.Cursor.Row = clamp(s.Cursor.Row, 0, t.Rows-1)
	s.Cursor.Col = clamp(s.Cursor.Col, 0, t.Cols-1)
	s.Charset = s.SavedCursor.Charset
	s.Modes.DecOM = s.SavedCursor.Origin
	s.Modes.DecAWM = s.SavedCursor.Wrap
}

func (t *Terminal) fullReset() {
	// Reset to primary screen
	t.active = t.primary
	t.primary = NewScreen(t.Rows, t.Cols)
	t.active = t.primary
	t.ScrollRegion = ScrollRegion{Top: 0, Bottom: t.Rows}
	t.CurrentStyle = DefaultStyle()
}

func (t *Terminal) switchToAltScreen() {
	t.alternate = newAlternateScreen(t.Rows, t.Cols)
	t.active = t.alternate
	t.active.Cursor = Cursor{Visible: true}
}

func (t *Terminal) switchToPrimaryScreen() {
	t.active = t.primary
}

func newAlternateScreen(rows, cols int) *Screen {
	s := NewScreen(rows, cols)
	s.MaxScroll = 0
	s.ScrollbackEnabled = false
	return s
}

func (t *Terminal) setMode(a parser.CSIDispatchAction) {
	for i := 1; i <= a.ParamCount; i++ {
		mode := Mode(a.Params[i-1])
		t.active.Modes.SetMode(mode)
	}
}

func (t *Terminal) resetMode(a parser.CSIDispatchAction) {
	for i := 1; i <= a.ParamCount; i++ {
		mode := Mode(a.Params[i-1])
		t.active.Modes.ResetMode(mode)
	}
}

func (t *Terminal) setDecModeWithAction(mode Mode) {
	switch mode {
	case ModeDecAltScreenSave, ModeDecAltScreen:
		t.switchToAltScreen()
	case ModeDecSaveCursor:
		t.saveCursor()
	case ModeDecAltScreenSaveCur:
		t.saveCursor()
		t.switchToAltScreen()
	default:
		t.active.Modes.SetDecMode(mode)
	}
}

func (t *Terminal) resetDecModeWithAction(mode Mode) {
	switch mode {
	case ModeDecAltScreenSave, ModeDecAltScreen:
		t.switchToPrimaryScreen()
	case ModeDecSaveCursor:
		t.restoreCursor()
	case ModeDecAltScreenSaveCur:
		t.switchToPrimaryScreen()
		t.restoreCursor()
	default:
		t.active.Modes.ResetDecMode(mode)
	}
}

// --- SGR (Select Graphic Rendition) ---

func (t *Terminal) handleSGR(a parser.CSIDispatchAction) {
	if a.ParamCount == 0 {
		t.CurrentStyle = DefaultStyle()
		return
	}

	i := 0
	for i < a.ParamCount {
		p := int(a.Params[i])
		switch {
		case p == 0: // Reset
			t.CurrentStyle = DefaultStyle()
		case p == 1: // Bold
			t.CurrentStyle.Bold = true
		case p == 2: // Faint
			t.CurrentStyle.Faint = true
		case p == 3: // Italic
			t.CurrentStyle.Italic = true
		case p == 4: // Underline
			if a.HasColon(i + 1) {
				// Sub-parameter: ESC[4:0m = none, 4:1m = single, etc.
				if i+1 < a.ParamCount {
					t.CurrentStyle.Underline = UnderlineStyle(a.Params[i+1])
					i++
				}
			} else {
				t.CurrentStyle.Underline = UnderlineSingle
			}
		case p == 5: // Blink
			t.CurrentStyle.Blink = true
		case p == 6: // Rapid blink
			t.CurrentStyle.RapidBlink = true
		case p == 7: // Reverse
			t.CurrentStyle.Reverse = true
		case p == 8: // Invisible
			t.CurrentStyle.Invisible = true
		case p == 9: // Strikethrough
			t.CurrentStyle.Strikethrough = true
		case p == 21: // Double underline (or bold off in some terminals)
			t.CurrentStyle.Underline = UnderlineDouble
		case p == 22: // Normal intensity (not bold, not faint)
			t.CurrentStyle.Bold = false
			t.CurrentStyle.Faint = false
		case p == 23: // Not italic
			t.CurrentStyle.Italic = false
		case p == 24: // Not underlined
			t.CurrentStyle.Underline = UnderlineNone
		case p == 25: // Not blinking
			t.CurrentStyle.Blink = false
			t.CurrentStyle.RapidBlink = false
		case p == 27: // Not reversed
			t.CurrentStyle.Reverse = false
		case p == 28: // Not invisible
			t.CurrentStyle.Invisible = false
		case p == 29: // Not strikethrough
			t.CurrentStyle.Strikethrough = false
		case p >= 30 && p <= 37: // Foreground color
			t.CurrentStyle.FG = ansiColor(p - 30)
		case p == 38: // Extended foreground
			n, consumed := t.parseExtendedColor(a, i+1)
			if consumed > 0 {
				t.CurrentStyle.FG = n
				i += consumed
			}
		case p == 39: // Default foreground
			t.CurrentStyle.FG = ColorNone
		case p >= 40 && p <= 47: // Background color
			t.CurrentStyle.BG = ansiColor(p - 40)
		case p == 48: // Extended background
			n, consumed := t.parseExtendedColor(a, i+1)
			if consumed > 0 {
				t.CurrentStyle.BG = n
				i += consumed
			}
		case p == 49: // Default background
			t.CurrentStyle.BG = ColorNone
		case p >= 90 && p <= 97: // Bright foreground
			t.CurrentStyle.FG = ansiColor(p - 90 + 8)
		case p >= 100 && p <= 107: // Bright background
			t.CurrentStyle.BG = ansiColor(p - 100 + 8)
		case p == 53: // Overline
			t.CurrentStyle.Overline = true
		case p == 55: // Not overline
			t.CurrentStyle.Overline = false
		}
		i++
	}
}

// parseExtendedColor parses extended color sequences (38;5;n or 38;2;r;g;b).
func (t *Terminal) parseExtendedColor(a parser.CSIDispatchAction, start int) (Color, int) {
	if start >= a.ParamCount {
		return ColorNone, 0
	}

	switch a.Params[start] {
	case 5: // 256-color: 38;5;n
		if start+1 < a.ParamCount {
			idx := int(a.Params[start+1])
			if idx >= 0 && idx < 256 {
				return DefaultPalette()[idx], 2
			}
		}
	case 2: // True color: 38;2;r;g;b
		if start+3 < a.ParamCount {
			if a.Params[start+1] > 255 || a.Params[start+2] > 255 || a.Params[start+3] > 255 {
				return ColorNone, 0
			}
			r := uint8(a.Params[start+1])
			g := uint8(a.Params[start+2])
			b := uint8(a.Params[start+3])
			return ColorFromRGB(r, g, b), 4
		}
	}

	return ColorNone, 0
}

// --- Utilities ---

// CharWidth returns the display width of a rune (0, 1, or 2).
func CharWidth(r rune) int {
	return unicode.RuneWidth(r)
}

func ansiColor(idx int) Color {
	palette := DefaultPalette()
	if idx >= 0 && idx < 256 {
		return palette[idx]
	}
	return ColorNone
}

func mapCharsetFinal(b byte) CharsetDesignation {
	switch b {
	case '0':
		return CharsetDecSpecial
	case 'A':
		return CharsetASCII
	case 'B':
		return CharsetASCII
	case '4':
		return CharsetDecSupplemental
	case 'C':
		return CharsetASCII
	case 'R':
		return CharsetUK
	default:
		return CharsetASCII
	}
}

func shiftRight(cells []Cell, start, n int, style StyleID) {
	cols := len(cells)
	for i := cols - 1; i >= start+n; i-- {
		cells[i] = cells[i-n]
	}
	for i := start; i < start+n && i < cols; i++ {
		cells[i] = EmptyCell(style)
	}
}

func clamp(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func base64Encode(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

func base64Decode(data []byte) []byte {
	decoded, err := base64.StdEncoding.DecodeString(string(data))
	if err != nil {
		return nil
	}
	return decoded
}
