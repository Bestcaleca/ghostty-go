package terminal

// ModeState holds all terminal mode flags.
type ModeState struct {
	// Standard modes (SM/RM)
	Insert      bool // IRM - Insert mode
	SendReceive bool // SRM - Local echo off (send/receive)
	LNM         bool // Line feed / newline mode

	// DEC private modes (DECSET/DECRST)
	DecCKM     bool // Application cursor keys
	DecANM     bool // ANSI/VT52 mode
	DecCOLM132 bool // 132 column mode
	DecSCNM    bool // Screen reverse video
	DecOM      bool // Origin mode (DECOM)
	DecAWM     bool // Auto-wrap mode (DECAWM)
	DecARM     bool // Auto-repeat mode
	DecINLM    bool // Interlace mode

	// More DEC private modes
	DecTextCursorEnable bool // DECTCEM - cursor visible
	DecTextCursorBlink  bool // cursor blink (xterm)
	DecAltScreen        bool // alternate screen buffer (xterm 1047)
	DecAltScreenSave    bool // alternate screen + save cursor (xterm 1049)
	DecMouseX10         bool // X10 mouse tracking
	DecMouseNormal      bool // Normal mouse tracking
	DecMouseHighlight   bool // Highlight mouse tracking
	DecMouseButton      bool // Button-event mouse tracking
	DecMouseAny         bool // Any-event mouse tracking
	DecMouseSGR         bool // SGR extended mouse coordinates
	DecBracketedPaste   bool // Bracketed paste mode (xterm 2004)
	DecFocusEvents      bool // Focus in/out events (xterm 1004)
	DecAltScroll        bool // Alternate scroll mode (xterm 1007)
	DecSixelScroll      bool // Sixel scrolling mode

	// xterm extensions
	GraphemeCluster bool // Mode 2027 - grapheme cluster mode
	ReverseWrap     bool // Mode 45 - reverse wraparound
	SixelDisplay    bool // Mode 8452 - sixel display mode

	// Kitty keyboard protocol
	KittyKeyboard uint8 // 0=normal, 1=disambiguate, 2=report events, 3=report all
}

// Mode represents a terminal mode that can be set/reset.
type Mode int

// Standard modes (CSI Ps h / CSI Ps l)
const (
	ModeInsert      Mode = 4  // IRM
	ModeSendReceive Mode = 12 // SRM
	ModeLNM         Mode = 20 // LNM
)

// DEC private modes (CSI ? Ps h / CSI ? Ps l)
const (
	ModeDecCKM              Mode = 1    // Application cursor keys
	ModeDecANM              Mode = 2    // ANSI/VT52 mode
	ModeDecCOLM132          Mode = 3    // 132 column mode
	ModeDecSCNM             Mode = 5    // Screen reverse video
	ModeDecOM               Mode = 6    // Origin mode
	ModeDecAWM              Mode = 7    // Auto-wrap mode
	ModeDecARM              Mode = 8    // Auto-repeat mode
	ModeDecINLM             Mode = 9    // Interlace mode
	ModeDecTextCursorBlink  Mode = 12   // Cursor blink
	ModeDecMouseX10         Mode = 9    // X10 mouse tracking
	ModeDecAltScreenSave    Mode = 47   // Legacy alternate screen buffer
	ModeDecTextCursorEnable Mode = 25   // Cursor visible
	ModeDecMouseNormal      Mode = 1000 // Normal mouse tracking
	ModeDecMouseHighlight   Mode = 1001 // Highlight mouse tracking
	ModeDecMouseButton      Mode = 1002 // Button-event mouse tracking
	ModeDecMouseAny         Mode = 1003 // Any-event mouse tracking
	ModeDecFocusEvents      Mode = 1004 // Focus events
	ModeDecMouseSGR         Mode = 1006 // SGR extended mouse coordinates
	ModeDecAltScreen        Mode = 1047 // Alternate screen (xterm)
	ModeDecSaveCursor       Mode = 1048 // Save/restore cursor (xterm)
	ModeDecAltScreenSaveCur Mode = 1049 // Alternate screen + save cursor
	ModeDecBracketedPaste   Mode = 2004 // Bracketed paste
	ModeDecAltScroll        Mode = 1007 // Alternate scroll
	ModeGraphemeCluster     Mode = 2027 // Grapheme cluster mode
	ModeReverseWrap         Mode = 45   // Reverse wraparound
	ModeSixelDisplay        Mode = 8452 // Sixel display mode
)

// SetMode sets a standard mode.
func (m *ModeState) SetMode(mode Mode) {
	switch mode {
	case ModeInsert:
		m.Insert = true
	case ModeSendReceive:
		m.SendReceive = true
	case ModeLNM:
		m.LNM = true
	}
}

// ResetMode resets a standard mode.
func (m *ModeState) ResetMode(mode Mode) {
	switch mode {
	case ModeInsert:
		m.Insert = false
	case ModeSendReceive:
		m.SendReceive = false
	case ModeLNM:
		m.LNM = false
	}
}

// SetDecMode sets a DEC private mode.
func (m *ModeState) SetDecMode(mode Mode) {
	switch mode {
	case ModeDecCKM:
		m.DecCKM = true
	case ModeDecANM:
		m.DecANM = true
	case ModeDecCOLM132:
		m.DecCOLM132 = true
	case ModeDecSCNM:
		m.DecSCNM = true
	case ModeDecOM:
		m.DecOM = true
	case ModeDecAWM:
		m.DecAWM = true
	case ModeDecARM:
		m.DecARM = true
	case ModeDecTextCursorEnable:
		m.DecTextCursorEnable = true
	case ModeDecTextCursorBlink:
		m.DecTextCursorBlink = true
	case ModeDecAltScreen:
		m.DecAltScreen = true
	case ModeDecAltScreenSave:
		m.DecAltScreen = true
	case ModeDecAltScreenSaveCur:
		m.DecAltScreen = true
	case ModeDecMouseX10:
		m.DecMouseX10 = true
	case ModeDecMouseNormal:
		m.DecMouseNormal = true
	case ModeDecMouseHighlight:
		m.DecMouseHighlight = true
	case ModeDecMouseButton:
		m.DecMouseButton = true
	case ModeDecMouseAny:
		m.DecMouseAny = true
	case ModeDecMouseSGR:
		m.DecMouseSGR = true
	case ModeDecBracketedPaste:
		m.DecBracketedPaste = true
	case ModeDecFocusEvents:
		m.DecFocusEvents = true
	case ModeDecAltScroll:
		m.DecAltScroll = true
	case ModeGraphemeCluster:
		m.GraphemeCluster = true
	case ModeReverseWrap:
		m.ReverseWrap = true
	case ModeSixelDisplay:
		m.SixelDisplay = true
	}
}

// ResetDecMode resets a DEC private mode.
func (m *ModeState) ResetDecMode(mode Mode) {
	switch mode {
	case ModeDecCKM:
		m.DecCKM = false
	case ModeDecANM:
		m.DecANM = false
	case ModeDecCOLM132:
		m.DecCOLM132 = false
	case ModeDecSCNM:
		m.DecSCNM = false
	case ModeDecOM:
		m.DecOM = false
	case ModeDecAWM:
		m.DecAWM = false
	case ModeDecARM:
		m.DecARM = false
	case ModeDecTextCursorEnable:
		m.DecTextCursorEnable = false
	case ModeDecTextCursorBlink:
		m.DecTextCursorBlink = false
	case ModeDecAltScreen:
		m.DecAltScreen = false
	case ModeDecAltScreenSave:
		m.DecAltScreen = false
	case ModeDecAltScreenSaveCur:
		m.DecAltScreen = false
	case ModeDecMouseX10:
		m.DecMouseX10 = false
	case ModeDecMouseNormal:
		m.DecMouseNormal = false
	case ModeDecMouseHighlight:
		m.DecMouseHighlight = false
	case ModeDecMouseButton:
		m.DecMouseButton = false
	case ModeDecMouseAny:
		m.DecMouseAny = false
	case ModeDecMouseSGR:
		m.DecMouseSGR = false
	case ModeDecBracketedPaste:
		m.DecBracketedPaste = false
	case ModeDecFocusEvents:
		m.DecFocusEvents = false
	case ModeDecAltScroll:
		m.DecAltScroll = false
	case ModeGraphemeCluster:
		m.GraphemeCluster = false
	case ModeReverseWrap:
		m.ReverseWrap = false
	case ModeSixelDisplay:
		m.SixelDisplay = false
	}
}

// QueryDecMode returns true if the given DEC mode is set.
func (m *ModeState) QueryDecMode(mode Mode) bool {
	switch mode {
	case ModeDecCKM:
		return m.DecCKM
	case ModeDecANM:
		return m.DecANM
	case ModeDecCOLM132:
		return m.DecCOLM132
	case ModeDecSCNM:
		return m.DecSCNM
	case ModeDecOM:
		return m.DecOM
	case ModeDecAWM:
		return m.DecAWM
	case ModeDecARM:
		return m.DecARM
	case ModeDecTextCursorEnable:
		return m.DecTextCursorEnable
	case ModeDecTextCursorBlink:
		return m.DecTextCursorBlink
	case ModeDecAltScreen, ModeDecAltScreenSave, ModeDecAltScreenSaveCur:
		return m.DecAltScreen
	case ModeDecMouseX10:
		return m.DecMouseX10
	case ModeDecMouseNormal:
		return m.DecMouseNormal
	case ModeDecMouseHighlight:
		return m.DecMouseHighlight
	case ModeDecMouseButton:
		return m.DecMouseButton
	case ModeDecMouseAny:
		return m.DecMouseAny
	case ModeDecMouseSGR:
		return m.DecMouseSGR
	case ModeDecBracketedPaste:
		return m.DecBracketedPaste
	case ModeDecFocusEvents:
		return m.DecFocusEvents
	case ModeDecAltScroll:
		return m.DecAltScroll
	case ModeGraphemeCluster:
		return m.GraphemeCluster
	case ModeReverseWrap:
		return m.ReverseWrap
	case ModeSixelDisplay:
		return m.SixelDisplay
	}
	return false
}
