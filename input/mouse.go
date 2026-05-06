package input

import (
	"fmt"

	"github.com/go-gl/glfw/v3.3/glfw"
)

// MouseMode tracks which mouse reporting mode is active.
type MouseMode int

const (
	MouseModeNone   MouseMode = iota // No reporting
	MouseModeX10                     // X10 (button press only)
	MouseModeNormal                  // Normal (press + release)
	MouseModeButton                  // Button (press + release + drag)
	MouseModeAny                     // Any (press + release + drag + motion)
	MouseModeSGR                     // SGR extended (like Any but with SGR encoding)
)

// MouseHandler converts GLFW mouse events into terminal escape sequences.
type MouseHandler struct {
	mode MouseMode
	sgr  bool
}

// NewMouseHandler creates a new mouse handler.
func NewMouseHandler() *MouseHandler {
	return &MouseHandler{}
}

// SetMode sets the mouse reporting mode.
func (mh *MouseHandler) SetMode(mode MouseMode) {
	if mode == MouseModeSGR {
		mh.sgr = true
		mode = MouseModeAny
	}
	mh.mode = mode
}

// SetSGR controls whether mouse positions use xterm SGR extended encoding.
func (mh *MouseHandler) SetSGR(enabled bool) {
	mh.sgr = enabled
}

// EncodeMouseButton encodes a mouse button press/release event.
// Returns nil if no reporting is active.
func (mh *MouseHandler) EncodeMouseButton(button glfw.MouseButton, action glfw.Action, mods Modifiers, col, row int) []byte {
	if mh.mode == MouseModeNone {
		return nil
	}

	// X10 mode only reports press
	if mh.mode == MouseModeX10 && action == glfw.Release {
		return nil
	}

	cb := encodeButton(button, action)
	if cb < 0 {
		return nil
	}

	// Add modifier bits
	if mods.Shift {
		cb |= 4
	}
	if mods.Alt {
		cb |= 8
	}
	if mods.Control {
		cb |= 16
	}

	return mh.encodePosition(cb, col, row)
}

// EncodeMouseMotion encodes a mouse motion event.
func (mh *MouseHandler) EncodeMouseMotion(buttons []glfw.MouseButton, mods Modifiers, col, row int) []byte {
	if mh.mode != MouseModeAny && mh.mode != MouseModeButton {
		return nil
	}
	if mh.mode == MouseModeButton && len(buttons) == 0 {
		return nil
	}

	cb := 35 // Passive motion indicator
	if len(buttons) > 0 {
		cb = encodeButton(buttons[0], glfw.Press)
		if cb < 0 {
			cb = 0
		}
		cb += 32 // Motion with button
	}

	if mods.Shift {
		cb |= 4
	}
	if mods.Alt {
		cb |= 8
	}
	if mods.Control {
		cb |= 16
	}

	return mh.encodePosition(cb, col, row)
}

func (mh *MouseHandler) encodePosition(cb, col, row int) []byte {
	if mh.sgr {
		// SGR extended: CSI < Cb ; Cx ; Cy M (press)  or  CSI < Cb ; Cx ; Cy m (release)
		// Bit 5 of cb indicates release in SGR mode (cb & 3 == 3 for release)
		terminator := "M"
		if cb&3 == 3 && cb < 32 {
			terminator = "m"
		}
		return []byte(fmt.Sprintf("\x1b[<%d;%d;%d%s", cb, col+1, row+1, terminator))
	}

	// X10/Normal/Button/Any: CSI M Cb+32 Cx+32 Cy+32
	// Coordinates are limited to 223 (1-based, so max 222)
	cb32 := cb + 32
	cx := col + 33 // 1-based + 32
	cy := row + 33

	// Clamp to valid range
	if cx > 255 {
		cx = 255
	}
	if cy > 255 {
		cy = 255
	}

	return []byte{0x1B, 'M', byte(cb32), byte(cx), byte(cy)}
}

// encodeButton maps a GLFW mouse button and action to an xterm button code.
func encodeButton(button glfw.MouseButton, action glfw.Action) int {
	switch button {
	case glfw.MouseButtonLeft:
		if action == glfw.Release {
			return 3
		}
		return 0
	case glfw.MouseButtonMiddle:
		if action == glfw.Release {
			return 3
		}
		return 1
	case glfw.MouseButtonRight:
		if action == glfw.Release {
			return 3
		}
		return 2
	case glfw.MouseButton4:
		if action == glfw.Release {
			return 3
		}
		return 128
	case glfw.MouseButton5:
		if action == glfw.Release {
			return 3
		}
		return 129
	}
	return -1
}

// EncodeScroll encodes a scroll event.
func (mh *MouseHandler) EncodeScroll(xoff, yoff float64, mods Modifiers, col, row int) []byte {
	if mh.mode == MouseModeNone || mh.mode == MouseModeX10 {
		return nil
	}

	cb := 64 // Scroll up
	if yoff < 0 {
		cb = 65 // Scroll down
	}
	if xoff > 0 {
		cb = 66 // Scroll right
	}
	if xoff < 0 {
		cb = 67 // Scroll left
	}

	if mods.Shift {
		cb |= 4
	}
	if mods.Alt {
		cb |= 8
	}
	if mods.Control {
		cb |= 16
	}

	return mh.encodePosition(cb, col, row)
}
