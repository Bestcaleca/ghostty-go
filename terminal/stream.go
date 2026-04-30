package terminal

import (
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
		actions := s.parser.Next(b)
		for _, a := range actions {
			if a != nil {
				s.dispatch(a)
			}
		}
	}
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
}
