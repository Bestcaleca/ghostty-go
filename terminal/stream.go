package terminal

import (
	"unicode/utf8"

	"github.com/ghostty-go/ghostty-go/parser"
)

// StreamHandler is the interface for handling parsed VT actions.
type StreamHandler interface {
	Print(ch rune)
	Execute(b byte)
	CSIDispatch(a parser.CSIDispatchAction)
	EscDispatch(a parser.EscDispatchAction)
	OSCDispatch(cmd parser.OSCCommand)
	DCSHook(a parser.DCSHookAction)
	DCSPut(b byte)
	DCSUnhook()
}

// Stream processes raw bytes through the VT parser and dispatches
// actions to a StreamHandler (typically a Terminal).
type Stream struct {
	parser  *parser.Parser
	handler StreamHandler
	utf8Buf []byte
}

// NewStream creates a new Stream with the given handler.
func NewStream(handler StreamHandler) *Stream {
	return &Stream{
		parser:  parser.New(),
		handler: handler,
	}
}

// Process feeds raw bytes through the parser and dispatches actions.
func (s *Stream) Process(data []byte) {
	for _, b := range data {
		if s.parser.State() == parser.StateGround && s.processUTF8Byte(b) {
			continue
		}
		s.processParserByte(b)
	}
}

func (s *Stream) processParserByte(b byte) {
	actions := s.parser.Next(b)
	for _, a := range actions {
		if a != nil {
			s.dispatch(a)
		}
	}
}

func (s *Stream) processUTF8Byte(b byte) bool {
	if len(s.utf8Buf) == 0 {
		if b < utf8.RuneSelf {
			return false
		}
		if b >= 0x80 && b <= 0x9F {
			return false
		}
	}

	s.utf8Buf = append(s.utf8Buf, b)
	if !utf8.FullRune(s.utf8Buf) {
		return true
	}

	ch, size := utf8.DecodeRune(s.utf8Buf)
	remaining := append([]byte(nil), s.utf8Buf[size:]...)
	s.utf8Buf = s.utf8Buf[:0]

	if ch == utf8.RuneError && size == 1 {
		s.handler.Print(utf8.RuneError)
	} else {
		s.handler.Print(ch)
	}

	for _, rb := range remaining {
		if !s.processUTF8Byte(rb) {
			s.processParserByte(rb)
		}
	}
	return true
}

// dispatch routes a parser action to the appropriate handler method.
func (s *Stream) dispatch(a parser.Action) {
	switch act := a.(type) {
	case parser.PrintAction:
		s.handler.Print(act.Char)
	case parser.ExecuteAction:
		s.handler.Execute(act.Byte)
	case parser.CSIDispatchAction:
		s.handler.CSIDispatch(act)
	case parser.EscDispatchAction:
		s.handler.EscDispatch(act)
	case parser.OSCDispatchAction:
		cmd := parser.ParseOSC(act.Command)
		s.handler.OSCDispatch(cmd)
	case parser.DCSHookAction:
		s.handler.DCSHook(act)
	case parser.DCSPutAction:
		s.handler.DCSPut(act.Byte)
	case parser.DCSUnhookAction:
		s.handler.DCSUnhook()
	// APC actions are silently ignored
	case parser.APCStartAction:
	case parser.APCEndAction:
	}
}

// Reset resets the parser to the Ground state.
func (s *Stream) Reset() {
	s.parser.Reset()
	s.utf8Buf = s.utf8Buf[:0]
}
