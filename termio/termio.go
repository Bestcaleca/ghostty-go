package termio

import (
	"io"

	"github.com/ghostty-go/ghostty-go/terminal"
)

// Message represents a message from the IO layer to the surface.
type Message interface {
	ioTag()
}

// TitleChangedMsg is sent when the terminal title changes.
type TitleChangedMsg struct{ Title string }

// ChildExitedMsg is sent when the shell process exits.
type ChildExitedMsg struct{ Code int }

// BellMsg is sent when the terminal bell is triggered.
type BellMsg struct{}

func (TitleChangedMsg) ioTag()  {}
func (ChildExitedMsg) ioTag()  {}
func (BellMsg) ioTag()         {}

// Termio manages the terminal I/O layer: PTY, parser stream, and terminal state.
type Termio struct {
	pty       *PTY
	stream    *terminal.Stream
	terminal  *terminal.Terminal
	resize    *ResizeCoalescer
	msgChan   chan Message
	writeChan chan []byte
	done      chan struct{}
	lastTitle string
}

// Config holds Termio configuration.
type Config struct {
	Shell string
	Rows  int
	Cols  int
}

// New creates a new Termio instance.
func New(cfg Config, term *terminal.Terminal, msgChan chan Message) (*Termio, error) {
	p, err := NewPTY(cfg.Shell, cfg.Rows, cfg.Cols)
	if err != nil {
		return nil, err
	}

	t := &Termio{
		pty:       p,
		stream:    terminal.NewStream(term),
		terminal:  term,
		resize:    NewResizeCoalescer(p),
		msgChan:   msgChan,
		writeChan: make(chan []byte, 256),
		done:      make(chan struct{}),
	}

	return t, nil
}

// Start begins the IO goroutines (read and write loops).
func (t *Termio) Start() {
	go t.readLoop()
	go t.writeLoop()
}

// Stop shuts down the IO layer.
func (t *Termio) Stop() {
	close(t.done)
	t.resize.Close()
	t.pty.Close()
}

// Write sends data to the PTY (keyboard input).
func (t *Termio) Write(data []byte) {
	select {
	case t.writeChan <- data:
	default:
		// Drop if buffer full (shouldn't happen in normal use)
	}
}

// Resize requests a PTY resize (coalesced).
func (t *Termio) Resize(rows, cols int) {
	t.resize.Request(rows, cols)
}

// readLoop reads from the PTY and feeds data through the parser.
func (t *Termio) readLoop() {
	buf := make([]byte, 32*1024)
	for {
		select {
		case <-t.done:
			return
		default:
		}

		n, err := t.pty.Read(buf)
		if n > 0 {
			// Feed through VT parser -> terminal state
			t.stream.Process(buf[:n])

			// Check for title change
			title := t.terminal.Title()
			if title != t.lastTitle {
				t.lastTitle = title
				select {
				case t.msgChan <- TitleChangedMsg{Title: title}:
				default:
				}
			}
		}

		if err != nil {
			if err == io.EOF {
				// Shell exited
				code := 0
				if waitErr := t.pty.Wait(); waitErr != nil {
					if exitErr, ok := waitErr.(interface{ ExitCode() int }); ok {
						code = exitErr.ExitCode()
					}
				}
				select {
				case t.msgChan <- ChildExitedMsg{Code: code}:
				case <-t.done:
				}
			}
			return
		}
	}
}

// writeLoop writes keyboard input to the PTY.
func (t *Termio) writeLoop() {
	for {
		select {
		case <-t.done:
			return
		case data := <-t.writeChan:
			t.pty.Write(data)
		}
	}
}
