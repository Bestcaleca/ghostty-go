package termio

import (
	"time"
)

// ResizeRequest represents a pending resize request.
type ResizeRequest struct {
	Rows int
	Cols int
}

// ResizeCoalescer debounces resize events, only applying the final size
// after a quiet period. This prevents excessive PTY resize calls during
// window dragging.
type ResizeCoalescer struct {
	ch      chan ResizeRequest
	pty     *PTY
	done    chan struct{}
}

// NewResizeCoalescer creates a new resize coalescer for the given PTY.
func NewResizeCoalescer(pty *PTY) *ResizeCoalescer {
	rc := &ResizeCoalescer{
		ch:   make(chan ResizeRequest, 4),
		pty:  pty,
		done: make(chan struct{}),
	}
	go rc.loop()
	return rc
}

// Request submits a resize request. Non-blocking — drops if buffer is full.
func (rc *ResizeCoalescer) Request(rows, cols int) {
	select {
	case rc.ch <- ResizeRequest{Rows: rows, Cols: cols}:
	default:
		// Buffer full, drop (will pick up latest from next request)
	}
}

// Close stops the coalescer goroutine.
func (rc *ResizeCoalescer) Close() {
	close(rc.done)
}

func (rc *ResizeCoalescer) loop() {
	var timer *time.Timer
	var pending *ResizeRequest

	for {
		select {
		case <-rc.done:
			if timer != nil {
				timer.Stop()
			}
			return
		case req := <-rc.ch:
			pending = &req
			if timer != nil {
				timer.Stop()
			}
			timer = time.NewTimer(25 * time.Millisecond)
		case <-func() <-chan time.Time {
			if timer != nil {
				return timer.C
			}
			return nil
		}():
			if pending != nil {
				rc.pty.Resize(pending.Rows, pending.Cols)
				pending = nil
			}
			timer = nil
		}
	}
}
