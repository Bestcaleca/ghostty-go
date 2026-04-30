package input

import (
	"fmt"

	"github.com/go-gl/glfw/v3.3/glfw"
)

// Modifiers tracks modifier key state.
type Modifiers struct {
	Shift   bool
	Control bool
	Alt     bool
	Super   bool
}

// EncodeModifier returns the xterm modifier parameter (1-based).
func (m Modifiers) Encode() int {
	mod := 1
	if m.Shift {
		mod += 1
	}
	if m.Alt {
		mod += 2
	}
	if m.Control {
		mod += 4
	}
	if m.Super {
		mod += 8
	}
	return mod
}

// KeyHandler converts GLFW key events into terminal escape sequences.
type KeyHandler struct {
	applicationCursorKeys bool
	applicationKeypad     bool
}

// NewKeyHandler creates a new key handler.
func NewKeyHandler() *KeyHandler {
	return &KeyHandler{}
}

// SetApplicationCursorKeys enables/disables application cursor key mode.
func (kh *KeyHandler) SetApplicationCursorKeys(enabled bool) {
	kh.applicationCursorKeys = enabled
}

// SetApplicationKeypad enables/disables application keypad mode.
func (kh *KeyHandler) SetApplicationKeypad(enabled bool) {
	kh.applicationKeypad = enabled
}

// EncodeKey converts a GLFW key event into a terminal escape sequence.
// Returns nil if the key has no mapping.
func (kh *KeyHandler) EncodeKey(key glfw.Key, action glfw.Action, mods Modifiers) []byte {
	if action == glfw.Release {
		return nil
	}

	mod := mods.Encode()

	// Handle Control+key combinations first
	if mods.Control {
		if seq := encodeControlKey(key); seq != nil {
			return seq
		}
	}

	// Handle Alt+key as ESC prefix
	if mods.Alt && !mods.Control {
		if seq := kh.encodeNormalKey(key, mod); seq != nil {
			return append([]byte{0x1B}, seq...)
		}
	}

	return kh.encodeNormalKey(key, mod)
}

func (kh *KeyHandler) encodeNormalKey(key glfw.Key, mod int) []byte {
	// Cursor keys
	switch key {
	case glfw.KeyUp:
		if kh.applicationCursorKeys {
			return encodeCSI("A", mod)
		}
		return encodeSS3("A", mod)
	case glfw.KeyDown:
		if kh.applicationCursorKeys {
			return encodeCSI("B", mod)
		}
		return encodeSS3("B", mod)
	case glfw.KeyRight:
		if kh.applicationCursorKeys {
			return encodeCSI("C", mod)
		}
		return encodeSS3("C", mod)
	case glfw.KeyLeft:
		if kh.applicationCursorKeys {
			return encodeCSI("D", mod)
		}
		return encodeSS3("D", mod)
	case glfw.KeyHome:
		if kh.applicationCursorKeys {
			return encodeCSI("H", mod)
		}
		return encodeSS3("H", mod)
	case glfw.KeyEnd:
		if kh.applicationCursorKeys {
			return encodeCSI("F", mod)
		}
		return encodeSS3("F", mod)
	}

	// Function keys (xterm-style CSI sequences)
	switch key {
	case glfw.KeyF1:
		return encodeCSI("11", mod, "~")
	case glfw.KeyF2:
		return encodeCSI("12", mod, "~")
	case glfw.KeyF3:
		return encodeCSI("13", mod, "~")
	case glfw.KeyF4:
		return encodeCSI("14", mod, "~")
	case glfw.KeyF5:
		return encodeCSI("15", mod, "~")
	case glfw.KeyF6:
		return encodeCSI("17", mod, "~")
	case glfw.KeyF7:
		return encodeCSI("18", mod, "~")
	case glfw.KeyF8:
		return encodeCSI("19", mod, "~")
	case glfw.KeyF9:
		return encodeCSI("20", mod, "~")
	case glfw.KeyF10:
		return encodeCSI("21", mod, "~")
	case glfw.KeyF11:
		return encodeCSI("23", mod, "~")
	case glfw.KeyF12:
		return encodeCSI("24", mod, "~")
	}

	// Editing keys
	switch key {
	case glfw.KeyInsert:
		return encodeCSI("2", mod, "~")
	case glfw.KeyDelete:
		return encodeCSI("3", mod, "~")
	case glfw.KeyPageUp:
		return encodeCSI("5", mod, "~")
	case glfw.KeyPageDown:
		return encodeCSI("6", mod, "~")
	case glfw.KeyEnter:
		return []byte("\r")
	case glfw.KeyKPEnter:
		if kh.applicationKeypad {
			return []byte{0x1B, 'O', 'M'}
		}
		return []byte("\r")
	case glfw.KeyTab:
		return []byte("\t")
	case glfw.KeyBackspace:
		return []byte{0x7F}
	case glfw.KeyEscape:
		return []byte{0x1B}
	}

	// Keypad keys (application mode)
	if kh.applicationKeypad {
		switch key {
		case glfw.KeyKP0:
			return []byte{0x1B, 'O', 'p'}
		case glfw.KeyKP1:
			return []byte{0x1B, 'O', 'q'}
		case glfw.KeyKP2:
			return []byte{0x1B, 'O', 'r'}
		case glfw.KeyKP3:
			return []byte{0x1B, 'O', 's'}
		case glfw.KeyKP4:
			return []byte{0x1B, 'O', 't'}
		case glfw.KeyKP5:
			return []byte{0x1B, 'O', 'u'}
		case glfw.KeyKP6:
			return []byte{0x1B, 'O', 'v'}
		case glfw.KeyKP7:
			return []byte{0x1B, 'O', 'w'}
		case glfw.KeyKP8:
			return []byte{0x1B, 'O', 'x'}
		case glfw.KeyKP9:
			return []byte{0x1B, 'O', 'y'}
		case glfw.KeyKPDecimal:
			return []byte{0x1B, 'O', 'n'}
		case glfw.KeyKPDivide:
			return []byte{0x1B, 'O', 'o'}
		case glfw.KeyKPMultiply:
			return []byte{0x1B, 'O', 'j'}
		case glfw.KeyKPSubtract:
			return []byte{0x1B, 'O', 'm'}
		case glfw.KeyKPAdd:
			return []byte{0x1B, 'O', 'k'}
		}
	}

	return nil
}

// encodeControlKey handles Control+key combinations.
func encodeControlKey(key glfw.Key) []byte {
	switch key {
	case glfw.KeySpace:
		return []byte{0x00} // NUL
	case glfw.KeyA:
		return []byte{0x01}
	case glfw.KeyB:
		return []byte{0x02}
	case glfw.KeyC:
		return []byte{0x03} // ETX / SIGINT
	case glfw.KeyD:
		return []byte{0x04} // EOT
	case glfw.KeyE:
		return []byte{0x05}
	case glfw.KeyF:
		return []byte{0x06}
	case glfw.KeyG:
		return []byte{0x07}
	case glfw.KeyH:
		return []byte{0x08} // BS
	case glfw.KeyI:
		return []byte{0x09} // HT
	case glfw.KeyJ:
		return []byte{0x0A} // LF
	case glfw.KeyK:
		return []byte{0x0B}
	case glfw.KeyL:
		return []byte{0x0C}
	case glfw.KeyM:
		return []byte{0x0D} // CR
	case glfw.KeyN:
		return []byte{0x0E}
	case glfw.KeyO:
		return []byte{0x0F}
	case glfw.KeyP:
		return []byte{0x10}
	case glfw.KeyQ:
		return []byte{0x11}
	case glfw.KeyR:
		return []byte{0x12}
	case glfw.KeyS:
		return []byte{0x13}
	case glfw.KeyT:
		return []byte{0x14}
	case glfw.KeyU:
		return []byte{0x15}
	case glfw.KeyV:
		return []byte{0x16}
	case glfw.KeyW:
		return []byte{0x17}
	case glfw.KeyX:
		return []byte{0x18}
	case glfw.KeyY:
		return []byte{0x19}
	case glfw.KeyZ:
		return []byte{0x1A}
	case glfw.KeyLeftBracket:
		return []byte{0x1B} // ESC
	case glfw.KeyBackslash:
		return []byte{0x1C}
	case glfw.KeyRightBracket:
		return []byte{0x1D}
	}
	return nil
}

// encodeCSI generates a CSI sequence with optional modifier and suffix.
// When suffix is empty, number is the final character (e.g. "A" for cursor up).
// When suffix is provided, number is a parameter and suffix is the final (e.g. "11","~" for F1).
func encodeCSI(number string, mod int, suffix ...string) []byte {
	if len(suffix) > 0 {
		// Format: CSI number [;mod] suffix
		if mod > 1 {
			return []byte(fmt.Sprintf("\x1b[%s;%d%s", number, mod, suffix[0]))
		}
		return []byte(fmt.Sprintf("\x1b[%s%s", number, suffix[0]))
	}
	// Format: CSI [1;mod] finalChar  (number IS the final char)
	if mod > 1 {
		return []byte(fmt.Sprintf("\x1b[1;%d%s", mod, number))
	}
	return []byte(fmt.Sprintf("\x1b[%s", number))
}

// encodeSS3 generates an SS3 sequence with optional modifier.
// Format: SS3 char  or  SS3 1 ; modifier char
func encodeSS3(char string, mod int) []byte {
	if mod > 1 {
		return []byte(fmt.Sprintf("\x1bO1;%d%s", mod, char))
	}
	return []byte(fmt.Sprintf("\x1bO%s", char))
}
