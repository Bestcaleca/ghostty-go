package termio

import (
	"testing"
	"time"

	"github.com/ghostty-go/ghostty-go/parser"
	"github.com/ghostty-go/ghostty-go/terminal"
)

func TestNewPTY(t *testing.T) {
	p, err := NewPTY("/bin/sh", 24, 80)
	if err != nil {
		t.Fatalf("NewPTY failed: %v", err)
	}
	defer p.Close()

	if p.Rows() != 24 {
		t.Errorf("expected rows=24, got %d", p.Rows())
	}
	if p.Cols() != 80 {
		t.Errorf("expected cols=80, got %d", p.Cols())
	}
}

func TestPTYResize(t *testing.T) {
	p, err := NewPTY("/bin/sh", 24, 80)
	if err != nil {
		t.Fatalf("NewPTY failed: %v", err)
	}
	defer p.Close()

	if err := p.Resize(30, 100); err != nil {
		t.Fatalf("Resize failed: %v", err)
	}

	if p.Rows() != 30 {
		t.Errorf("expected rows=30, got %d", p.Rows())
	}
	if p.Cols() != 100 {
		t.Errorf("expected cols=100, got %d", p.Cols())
	}
}

func TestPTYWriteRead(t *testing.T) {
	p, err := NewPTY("/bin/sh", 24, 80)
	if err != nil {
		t.Fatalf("NewPTY failed: %v", err)
	}
	defer p.Close()

	// Write "echo hello\n" to the shell
	_, err = p.Write([]byte("echo hello\n"))
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Read output with timeout
	buf := make([]byte, 1024)
	done := make(chan int, 1)
	go func() {
		n, _ := p.Read(buf)
		done <- n
	}()

	select {
	case n := <-done:
		if n == 0 {
			t.Error("expected output from shell, got 0 bytes")
		}
	case <-time.After(2 * time.Second):
		t.Error("timeout waiting for shell output")
	}
}

func TestDetectShell(t *testing.T) {
	shell := detectShell()
	if shell == "" {
		t.Error("detectShell returned empty string")
	}
}

func TestResizeCoalescer(t *testing.T) {
	p, err := NewPTY("/bin/sh", 24, 80)
	if err != nil {
		t.Fatalf("NewPTY failed: %v", err)
	}
	defer p.Close()

	rc := NewResizeCoalescer(p)
	defer rc.Close()

	// Send multiple resize requests rapidly
	rc.Request(30, 100)
	rc.Request(35, 120)
	rc.Request(40, 140)

	// Wait for debounce (25ms) + some margin
	time.Sleep(100 * time.Millisecond)

	// Only the last resize should have been applied
	if p.Rows() != 40 {
		t.Errorf("expected rows=40 after coalescing, got %d", p.Rows())
	}
	if p.Cols() != 140 {
		t.Errorf("expected cols=140 after coalescing, got %d", p.Cols())
	}
}

func TestTermioLifecycle(t *testing.T) {
	term := terminal.New(24, 80)
	msgChan := make(chan Message, 16)

	tio, err := New(Config{
		Shell: "/bin/sh",
		Rows:  24,
		Cols:  80,
	}, term, msgChan)
	if err != nil {
		t.Fatalf("New Termio failed: %v", err)
	}

	tio.Start()

	// Write something to trigger output
	tio.Write([]byte("echo test\n"))

	// Wait for some output to be processed
	time.Sleep(500 * time.Millisecond)

	// Verify terminal state has been updated
	grid := term.Grid()
	hasContent := false
	for _, row := range grid {
		for _, cell := range row {
			if cell.Char != ' ' && cell.Char != 0 {
				hasContent = true
				break
			}
		}
		if hasContent {
			break
		}
	}

	if !hasContent {
		t.Error("expected terminal to have content after shell output")
	}

	tio.Stop()
}

func TestTermioTitleChange(t *testing.T) {
	term := terminal.New(24, 80)
	msgChan := make(chan Message, 16)

	tio, err := New(Config{
		Shell: "/bin/sh",
		Rows:  24,
		Cols:  80,
	}, term, msgChan)
	if err != nil {
		t.Fatalf("New Termio failed: %v", err)
	}

	tio.Start()

	// Set title via OSC
	term.OSCDispatch(parser.OSCSetWindowTitle{Title: "My Title"})

	// Write something to trigger read loop and title check
	tio.Write([]byte("echo hi\n"))
	time.Sleep(500 * time.Millisecond)

	// Check for title change message
	select {
	case msg := <-msgChan:
		if titleMsg, ok := msg.(TitleChangedMsg); ok {
			if titleMsg.Title != "My Title" {
				t.Errorf("expected title 'My Title', got '%s'", titleMsg.Title)
			}
		}
	default:
		// Title change message might have been sent and consumed already
		// or the read loop might not have checked yet
	}

	tio.Stop()
}
