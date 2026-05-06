package surface

import (
	"os/exec"
	"runtime"
	"sync"
	"time"

	"github.com/go-gl/glfw/v3.3/glfw"

	"github.com/ghostty-go/ghostty-go/input"
	"github.com/ghostty-go/ghostty-go/renderer"
	"github.com/ghostty-go/ghostty-go/terminal"
	"github.com/ghostty-go/ghostty-go/termio"
)

// Surface connects the terminal, renderer, and input systems.
// It is the integration hub that routes events between subsystems.
type Surface struct {
	terminal *terminal.Terminal
	stream   *terminal.Stream
	termio   *termio.Termio
	renderer *renderer.Renderer
	keyH     *input.KeyHandler
	mouseH   *input.MouseHandler
	msgChan  chan termio.Message
	window   *glfw.Window // for clipboard access

	mu sync.RWMutex // protects terminal state during render

	// Cursor blink state
	cursorVisible bool
	lastBlink     time.Time

	// Viewport scrollback offset (0 = bottom/latest, positive = scrolled up)
	scrollOffset int

	// Selection state
	selecting     bool
	lastClickTime time.Time
	lastClickRow  int
	lastClickCol  int
	clickCount    int

	// Context menu state
	contextMenu contextMenu

	// Bell state
	visualBell bool
	bellTime   time.Time

	// Callbacks
	onTitleChange func(string)
	onChildExit   func(int)
	onBell        func()

	// Config
	rows int
	cols int
}

// Config holds surface configuration.
type Config struct {
	Rows            int
	Cols            int
	Shell           string
	Renderer        *renderer.Renderer
	Window          *glfw.Window
	ScrollbackLines int
}

// New creates a new Surface.
func New(cfg Config) (*Surface, error) {
	term := terminal.New(cfg.Rows, cfg.Cols)
	if cfg.ScrollbackLines > 0 {
		term.SetMaxScroll(cfg.ScrollbackLines)
	}
	msgChan := make(chan termio.Message, 16)

	tio, err := termio.New(termio.Config{
		Shell: cfg.Shell,
		Rows:  cfg.Rows,
		Cols:  cfg.Cols,
	}, term, msgChan)
	if err != nil {
		return nil, err
	}

	s := &Surface{
		terminal:      term,
		stream:        terminal.NewStream(term),
		termio:        tio,
		renderer:      cfg.Renderer,
		keyH:          input.NewKeyHandler(),
		mouseH:        input.NewMouseHandler(),
		msgChan:       msgChan,
		window:        cfg.Window,
		cursorVisible: true,
		lastBlink:     time.Now(),
		rows:          cfg.Rows,
		cols:          cfg.Cols,
	}

	return s, nil
}

// Start begins the IO goroutines.
func (s *Surface) Start() {
	// Set up clipboard callbacks
	s.terminal.SetClipboardWrite(func(clipboard string, data []byte) {
		if s.window != nil {
			s.window.SetClipboardString(string(data))
		}
	})

	s.terminal.SetClipboardRead(func(clipboard string) []byte {
		if s.window == nil {
			return nil
		}
		str := s.window.GetClipboardString()
		if str == "" {
			return nil
		}
		return []byte(str)
	})

	s.terminal.SetRespond(func(data []byte) {
		s.termio.Write(data)
	})

	s.terminal.SetBell(func() {
		if s.onBell != nil {
			s.onBell()
		}
		s.visualBell = true
		s.bellTime = time.Now()
	})

	s.termio.Start()
}

// Stop shuts down the surface.
func (s *Surface) Stop() {
	s.termio.Stop()
}

// SetOnTitleChange sets the callback for title changes.
func (s *Surface) SetOnTitleChange(fn func(string)) {
	s.onTitleChange = fn
}

// SetOnChildExit sets the callback for shell exit.
func (s *Surface) SetOnChildExit(fn func(int)) {
	s.onChildExit = fn
}

// SetOnBell sets the callback for terminal bell.
func (s *Surface) SetOnBell(fn func()) {
	s.onBell = fn
}

// HandlePaste processes a paste event (e.g., Ctrl+Shift+V).
// If bracketed paste mode is enabled, wraps the text in ESC[200~ ... ESC[201~.
func (s *Surface) HandlePaste(text string) {
	if text == "" {
		return
	}

	data := []byte(text)
	if s.terminal.BracketedPaste() {
		// Wrap in bracketed paste sequences
		wrapped := make([]byte, 0, len(data)+20)
		wrapped = append(wrapped, []byte("\x1b[200~")...)
		wrapped = append(wrapped, data...)
		wrapped = append(wrapped, []byte("\x1b[201~")...)
		s.termio.Write(wrapped)
	} else {
		s.termio.Write(data)
	}
}

// ScrollUp scrolls the viewport up by n lines (into scrollback).
func (s *Surface) ScrollUp(n int) {
	maxScroll := s.terminal.ScrollbackLen()
	s.scrollOffset += n
	if s.scrollOffset > maxScroll {
		s.scrollOffset = maxScroll
	}
}

// ScrollDown scrolls the viewport down by n lines (toward current).
func (s *Surface) ScrollDown(n int) {
	s.scrollOffset -= n
	if s.scrollOffset < 0 {
		s.scrollOffset = 0
	}
}

// ScrollToBottom resets the viewport to the current output.
func (s *Surface) ScrollToBottom() {
	s.scrollOffset = 0
}

// HandleKey processes a GLFW key event.
func (s *Surface) HandleKey(key glfw.Key, action glfw.Action, mods glfw.ModifierKey) {
	m := input.Modifiers{
		Shift:   mods&glfw.ModShift != 0,
		Control: mods&glfw.ModControl != 0,
		Alt:     mods&glfw.ModAlt != 0,
		Super:   mods&glfw.ModSuper != 0,
	}

	seq := s.keyH.EncodeKey(key, action, m)
	if seq != nil {
		s.termio.Write(seq)
	}
}

// HandleChar processes a GLFW character input event.
func (s *Surface) HandleChar(char rune) {
	// Encode rune as UTF-8
	buf := make([]byte, 4)
	n := encodeUTF8(char, buf)
	s.termio.Write(buf[:n])
}

// HandleMouseButton processes a GLFW mouse button event.
func (s *Surface) HandleMouseButton(button glfw.MouseButton, action glfw.Action, mods glfw.ModifierKey, x, y float64) {
	if s.renderer == nil {
		return
	}
	metrics := s.renderer.Metrics()
	col := int(x / float64(metrics.CellWidth))
	row := int(y / float64(metrics.CellHeight))

	if s.contextMenu.visible && action == glfw.Press {
		s.handleContextMenuClick(row, col)
		return
	}

	if button == glfw.MouseButtonRight && action == glfw.Press {
		s.contextMenu = newContextMenu(row, col, s.rows, s.cols)
		return
	}

	if button == glfw.MouseButtonLeft {
		if action == glfw.Press {
			// Ctrl+Click on hyperlink: open URL
			if mods&glfw.ModControl != 0 {
				url := s.GetHyperlink(row, col)
				if url != "" {
					openURL(url)
					return
				}
			}

			// Check for double/triple click
			now := time.Now()
			if now.Sub(s.lastClickTime) < 300*time.Millisecond &&
				s.lastClickRow == row && s.lastClickCol == col {
				s.clickCount++
			} else {
				s.clickCount = 1
			}
			s.lastClickTime = now
			s.lastClickRow = row
			s.lastClickCol = col

			// Start selection
			switch s.clickCount {
			case 2:
				s.terminal.SelectionStart(row, col, terminal.SelectionWord)
			case 3:
				s.terminal.SelectionStart(row, col, terminal.SelectionLine)
			default:
				s.terminal.SelectionStart(row, col, terminal.SelectionChar)
			}
			s.selecting = true
			return
		}

		// Release: copy selection to clipboard
		if action == glfw.Release && s.selecting {
			s.selecting = false
			text := s.terminal.GetSelectedText()
			if text != "" && s.window != nil {
				s.window.SetClipboardString(text)
			}
			return
		}
	}

	// Right/middle click: clear selection and send to terminal
	if action == glfw.Press {
		s.terminal.SelectionClear()
	}

	// Update mouse mode from terminal state
	s.mouseH.SetMode(s.terminal.MouseMode())

	m := input.Modifiers{
		Shift:   mods&glfw.ModShift != 0,
		Control: mods&glfw.ModControl != 0,
		Alt:     mods&glfw.ModAlt != 0,
		Super:   mods&glfw.ModSuper != 0,
	}

	seq := s.mouseH.EncodeMouseButton(button, action, m, col, row)
	if seq != nil {
		s.termio.Write(seq)
	}
}

// HandleMouseMotion processes a GLFW mouse motion event.
func (s *Surface) HandleMouseMotion(x, y float64) {
	if s.renderer == nil {
		return
	}

	if s.selecting {
		metrics := s.renderer.Metrics()
		col := int(x / float64(metrics.CellWidth))
		row := int(y / float64(metrics.CellHeight))
		s.terminal.SelectionUpdate(row, col)
	}
}

// HandleScroll processes a GLFW scroll event.
func (s *Surface) HandleScroll(xoff, yoff float64, x, y float64, mods glfw.ModifierKey) {
	if s.renderer == nil {
		return
	}

	// Shift+scroll = viewport scroll through scrollback
	if mods&glfw.ModShift != 0 {
		if yoff > 0 {
			s.ScrollUp(int(yoff * 3))
		} else if yoff < 0 {
			s.ScrollDown(int(-yoff * 3))
		}
		return
	}

	// Auto-scroll to bottom on any scroll event
	if s.scrollOffset > 0 {
		s.ScrollToBottom()
	}

	// Update mouse mode from terminal state
	s.mouseH.SetMode(s.terminal.MouseMode())

	metrics := s.renderer.Metrics()
	col := int(x / float64(metrics.CellWidth))
	row := int(y / float64(metrics.CellHeight))

	seq := s.mouseH.EncodeScroll(xoff, yoff, input.Modifiers{}, col, row)
	if seq != nil {
		s.termio.Write(seq)
	}
}

// HandleResize processes a window resize event.
func (s *Surface) HandleResize(width, height int) {
	if s.renderer == nil {
		return
	}
	s.renderer.Resize(width, height)

	metrics := s.renderer.Metrics()
	cols := int(float64(width) / float64(metrics.CellWidth))
	rows := int(float64(height) / float64(metrics.CellHeight))

	if cols != s.cols || rows != s.rows {
		s.cols = cols
		s.rows = rows
		s.termio.Resize(rows, cols)
	}
}

// ProcessMessages drains the message channel and fires callbacks.
func (s *Surface) ProcessMessages() {
	for {
		select {
		case msg := <-s.msgChan:
			switch m := msg.(type) {
			case termio.TitleChangedMsg:
				if s.onTitleChange != nil {
					s.onTitleChange(m.Title)
				}
			case termio.ChildExitedMsg:
				if s.onChildExit != nil {
					s.onChildExit(m.Code)
				}
			case termio.BellMsg:
				if s.onBell != nil {
					s.onBell()
				}
			}
		default:
			return
		}
	}
}

// UpdateCursor updates cursor blink state and returns whether it should be visible.
func (s *Surface) UpdateCursor() bool {
	if time.Since(s.lastBlink) > 500*time.Millisecond {
		s.cursorVisible = !s.cursorVisible
		s.lastBlink = time.Now()
	}
	return s.cursorVisible
}

// RenderGrid converts the terminal grid to renderer cells and draws a frame.
func (s *Surface) RenderGrid() {
	if s.renderer == nil {
		return
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	grid := s.terminal.Grid()
	cursorRow, cursorCol, _, cursorStyle := s.terminal.Cursor()

	// Update application cursor mode
	s.keyH.SetApplicationCursorKeys(s.terminal.Active().Modes.QueryDecMode(terminal.ModeDecCKM))

	// Build full render grid including scrollback
	var fullGrid [][]terminal.Cell
	if s.scrollOffset > 0 {
		scrollback := s.terminal.ScrollbackRows()
		sbLen := len(scrollback)

		// Calculate how many scrollback lines to show
		offset := s.scrollOffset
		if offset > sbLen {
			offset = sbLen
		}

		// Take the last 'offset' lines from scrollback
		start := sbLen - offset
		for _, row := range scrollback[start:] {
			fullGrid = append(fullGrid, row.Cells)
		}
		fullGrid = append(fullGrid, grid...)

		// Trim to screen size
		if len(fullGrid) > s.rows {
			fullGrid = fullGrid[len(fullGrid)-s.rows:]
		}
	} else {
		fullGrid = grid
	}

	// Convert terminal grid to renderer cells
	renderGrid := make([][]renderer.Cell, len(fullGrid))
	for row := range fullGrid {
		renderGrid[row] = make([]renderer.Cell, len(fullGrid[row]))
		for col := range fullGrid[row] {
			c := fullGrid[row][col]
			style := s.terminal.Active().Styles.Lookup(c.Style)

			fg := styleToColor(style.FG, renderer.Color{R: 0.9, G: 0.9, B: 0.9, A: 1.0})
			bg := styleToColor(style.BG, renderer.Color{R: 0.1, G: 0.1, B: 0.12, A: 1.0})

			cell := renderer.Cell{
				Char:  c.Char,
				FG:    fg,
				BG:    bg,
				Width: int(c.Width),
			}
			cell = applyTextStyle(cell, style)

			// Highlight selected cells
			if s.terminal.SelectionIsSelected(row, col) {
				cell.FG, cell.BG = cell.BG, cell.FG
			}

			renderGrid[row][col] = cell
		}
	}

	applyContextMenuOverlay(renderGrid, s.contextMenu)

	// Hide cursor when scrolled back
	cursorVisible := s.UpdateCursor()
	if s.scrollOffset > 0 {
		cursorVisible = false
	}

	// Convert terminal cursor style to renderer style
	renCursorStyle := convertCursorStyle(cursorStyle)
	s.renderer.SetCursor(cursorRow, cursorCol, cursorVisible, renCursorStyle)

	// Visual bell effect: briefly flash the screen
	if s.visualBell {
		if time.Since(s.bellTime) < 100*time.Millisecond {
			// Invert background for visual bell
			origBG := s.renderer.BGColor()
			s.renderer.SetBGColor(renderer.Color{
				R: 1.0 - origBG.R,
				G: 1.0 - origBG.G,
				B: 1.0 - origBG.B,
				A: origBG.A,
			})
			s.renderer.DrawFrame(renderGrid)
			s.renderer.SetBGColor(origBG)
		} else {
			s.visualBell = false
			s.renderer.DrawFrame(renderGrid)
		}
	} else {
		s.renderer.DrawFrame(renderGrid)
	}
}

// Rows returns the number of visible rows.
func (s *Surface) Rows() int {
	return s.rows
}

// GetHyperlink returns the URL at the given cell position, or empty string.
func (s *Surface) GetHyperlink(row, col int) string {
	grid := s.terminal.Grid()
	if row < 0 || row >= len(grid) {
		return ""
	}
	if col < 0 || col >= len(grid[row]) {
		return ""
	}
	return grid[row][col].Hyperlink
}

// Terminal returns the underlying terminal (for testing).
func (s *Surface) Terminal() *terminal.Terminal {
	return s.terminal
}

func (s *Surface) handleContextMenuClick(row, col int) {
	action, ok := s.contextMenu.actionAt(row, col)
	s.contextMenu.visible = false
	if !ok {
		return
	}

	switch action {
	case contextMenuCopy:
		text := s.terminal.GetSelectedText()
		if text != "" && s.window != nil {
			s.window.SetClipboardString(text)
		}
	case contextMenuPaste:
		if s.window != nil {
			text := s.window.GetClipboardString()
			if text != "" {
				s.HandlePaste(text)
			}
		}
	case contextMenuClearSelection:
		s.terminal.SelectionClear()
	}
}

func applyTextStyle(cell renderer.Cell, style terminal.Style) renderer.Cell {
	if style.Reverse {
		cell.FG, cell.BG = cell.BG, cell.FG
	}
	if style.Bold {
		cell.FG = scaleColor(cell.FG, 1.25)
	}
	if style.Faint {
		cell.FG = scaleColor(cell.FG, 0.6)
	}
	if style.Invisible {
		cell.Char = ' '
	}
	return cell
}

func scaleColor(c renderer.Color, factor float32) renderer.Color {
	c.R = clampUnit(c.R * factor)
	c.G = clampUnit(c.G * factor)
	c.B = clampUnit(c.B * factor)
	return c
}

func clampUnit(v float32) float32 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func styleToColor(c terminal.Color, fallback renderer.Color) renderer.Color {
	if !c.Valid {
		return fallback
	}
	return renderer.Color{
		R: float32(c.R) / 255.0,
		G: float32(c.G) / 255.0,
		B: float32(c.B) / 255.0,
		A: 1.0,
	}
}

// encodeUTF8 encodes a rune into UTF-8 bytes. Returns the number of bytes written.
func encodeUTF8(r rune, buf []byte) int {
	if r < 0x80 {
		buf[0] = byte(r)
		return 1
	} else if r < 0x800 {
		buf[0] = 0xC0 | byte(r>>6)
		buf[1] = 0x80 | byte(r&0x3F)
		return 2
	} else if r < 0x10000 {
		buf[0] = 0xE0 | byte(r>>12)
		buf[1] = 0x80 | byte((r>>6)&0x3F)
		buf[2] = 0x80 | byte(r&0x3F)
		return 3
	} else {
		buf[0] = 0xF0 | byte(r>>18)
		buf[1] = 0x80 | byte((r>>12)&0x3F)
		buf[2] = 0x80 | byte((r>>6)&0x3F)
		buf[3] = 0x80 | byte(r&0x3F)
		return 4
	}
}

// convertCursorStyle converts terminal cursor style to renderer cursor style.
func convertCursorStyle(style terminal.CursorStyle) renderer.CursorStyle {
	switch style {
	case terminal.CursorBlinkingBlock, terminal.CursorSteadyBlock, terminal.CursorDefault:
		return renderer.CursorBlock
	case terminal.CursorBlinkingUnderline, terminal.CursorSteadyUnderline:
		return renderer.CursorUnderline
	case terminal.CursorBlinkingBar, terminal.CursorSteadyBar:
		return renderer.CursorBeam
	default:
		return renderer.CursorBlock
	}
}

// openURL opens a URL in the default browser.
func openURL(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return
	}
	cmd.Start()
}
