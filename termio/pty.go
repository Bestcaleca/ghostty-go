package termio

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/creack/pty"
)

// PTY manages a pseudo-terminal connection to a shell process.
type PTY struct {
	file *os.File
	cmd  *exec.Cmd
	rows int
	cols int
}

// NewPTY spawns a new shell process with a pseudo-terminal.
func NewPTY(shell string, rows, cols int) (*PTY, error) {
	if shell == "" {
		shell = detectShell()
	}

	cmd := exec.Command(shell)
	cmd.Env = append(os.Environ(),
		"TERM=xterm-256color",
		fmt.Sprintf("COLUMNS=%d", cols),
		fmt.Sprintf("LINES=%d", rows),
	)

	f, err := pty.Start(cmd)
	if err != nil {
		return nil, fmt.Errorf("pty start: %w", err)
	}

	// Set initial size
	if err := pty.Setsize(f, &pty.Winsize{
		Rows: uint16(rows),
		Cols: uint16(cols),
	}); err != nil {
		f.Close()
		cmd.Process.Kill()
		return nil, fmt.Errorf("pty setsize: %w", err)
	}

	return &PTY{
		file: f,
		cmd:  cmd,
		rows: rows,
		cols: cols,
	}, nil
}

// Read reads data from the PTY.
func (p *PTY) Read(buf []byte) (int, error) {
	return p.file.Read(buf)
}

// Write writes data to the PTY (keyboard input).
func (p *PTY) Write(data []byte) (int, error) {
	return p.file.Write(data)
}

// Resize resizes the PTY.
func (p *PTY) Resize(rows, cols int) error {
	p.rows = rows
	p.cols = cols
	return pty.Setsize(p.file, &pty.Winsize{
		Rows: uint16(rows),
		Cols: uint16(cols),
	})
}

// Close closes the PTY and kills the shell process.
func (p *PTY) Close() error {
	p.cmd.Process.Kill()
	return p.file.Close()
}

// Wait waits for the shell process to exit.
func (p *PTY) Wait() error {
	return p.cmd.Wait()
}

// Rows returns the current row count.
func (p *PTY) Rows() int { return p.rows }

// Cols returns the current column count.
func (p *PTY) Cols() int { return p.cols }

// File returns the underlying PTY file (for polling).
func (p *PTY) File() *os.File { return p.file }

// detectShell returns the user's default shell.
func detectShell() string {
	shell := os.Getenv("SHELL")
	if shell != "" {
		return shell
	}
	return "/bin/sh"
}
