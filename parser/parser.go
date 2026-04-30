// Package parser implements a VT500-series terminal escape sequence parser.
//
// The parser is a table-driven state machine that consumes bytes and emits
// typed actions. It follows the DEC ANSI parser specification with 14 states.
package parser

// State represents a parser state in the VT500 state machine.
type State uint8

const (
	StateGround             State = iota // Normal character processing
	StateEscape                         // ESC received
	StateEscapeIntermediate             // ESC intermediate byte(s)
	StateCSIEntry                       // CSI sequence start
	StateCSIParam                       // CSI parameter bytes
	StateCSIIntermediate                // CSI intermediate byte(s)
	StateCSIIgnore                      // CSI ignore (invalid sequence)
	StateDCSEntry                       // DCS sequence start
	StateDCSParam                       // DCS parameter bytes
	StateDCSIntermediate                // DCS intermediate byte(s)
	StateDCSPassthrough                 // DCS data passthrough
	StateDCSIgnore                      // DCS ignore (invalid sequence)
	StateOSCString                      // OSC string accumulation
	StateSOSPMAPCString                 // SOS/PM/APC string
)

// TransitionAction represents the action to take during a state transition.
type TransitionAction uint8

const (
	ActionNone             TransitionAction = iota
	ActionIgnore                            // Ignore the byte
	ActionPrint                             // Print character (Ground entry)
	ActionExecute                           // Execute C0/C1 control
	ActionCollect                           // Collect intermediate byte
	ActionParam                             // Accumulate parameter digit
	ActionParamSeparator                    // Parameter separator (; or :)
	ActionEscDispatch                       // Dispatch ESC sequence
	ActionCSIDispatch                       // Dispatch CSI sequence
	ActionDCSHook                           // DCS entry (start DCS)
	ActionDCSPut                            // DCS data byte
	ActionDCSUnhook                         // DCS exit (end DCS)
	ActionOSCPut                            // OSC data byte
	ActionAPCPut                            // APC data byte
	ActionOSCStart                          // OSC sequence start
	ActionOSCEnd                            // OSC sequence end (BEL or ST)
	ActionAPCStart                          // APC sequence start
	ActionAPCEnd                            // APC sequence end
	ActionCollectPrivate                    // Collect private mode marker (?)
)

// Action represents a parsed output from the parser.
// Use type switch to handle different action types.
type Action interface {
	actionTag()
}

// PrintAction is emitted for printable characters.
type PrintAction struct {
	Char rune
}

// ExecuteAction is emitted for C0/C1 control characters.
type ExecuteAction struct {
	Byte byte
}

// CSIDispatchAction is emitted when a CSI sequence is complete.
type CSIDispatchAction struct {
	Final        byte
	Params       [MaxParams]uint16
	ParamCount   int
	Intermediates [MaxIntermediates]byte
	IntermediateCount int
	ParamSeps    [MaxParams]bool // true = colon, false = semicolon
	Private      bool            // true if '?' prefix (xterm private mode)
}

// EscDispatchAction is emitted when an ESC sequence is complete.
type EscDispatchAction struct {
	Final        byte
	Intermediates [MaxIntermediates]byte
	IntermediateCount int
}

// OSCDispatchAction is emitted when an OSC command is received.
type OSCDispatchAction struct {
	Command []byte
}

// DCSHookAction is emitted when a DCS sequence starts.
type DCSHookAction struct {
	Final        byte
	Params       [MaxParams]uint16
	ParamCount   int
	Intermediates [MaxIntermediates]byte
	IntermediateCount int
}

// DCSPutAction is emitted for each DCS data byte.
type DCSPutAction struct {
	Byte byte
}

// DCSUnhookAction is emitted when a DCS sequence ends.
type DCSUnhookAction struct{}

// APCStartAction is emitted when an APC sequence starts.
type APCStartAction struct{}

// APCEndAction is emitted when an APC sequence ends.
type APCEndAction struct{}

// actionTag implementations
func (PrintAction) actionTag()        {}
func (ExecuteAction) actionTag()      {}
func (CSIDispatchAction) actionTag()  {}
func (EscDispatchAction) actionTag()  {}
func (OSCDispatchAction) actionTag()  {}
func (DCSHookAction) actionTag()      {}
func (DCSPutAction) actionTag()       {}
func (DCSUnhookAction) actionTag()    {}
func (APCStartAction) actionTag()     {}
func (APCEndAction) actionTag()       {}

const (
	MaxParams       = 24
	MaxIntermediates = 4
)

// Parser is a VT500-series terminal escape sequence parser.
type Parser struct {
	state State

	// Intermediate byte collection
	intermediates     [MaxIntermediates]byte
	intermediatesCount int

	// Parameter accumulation
	params     [MaxParams]uint16
	paramCount int
	paramAcc   uint16
	paramSeps  [MaxParams]bool // colon tracking for SGR sub-parameters

	// Private mode flag (? prefix)
	private bool

	// OSC accumulation
	oscBuf []byte

	// DCS active flag
	dcsActive bool
}

// New creates a new Parser in the Ground state.
func New() *Parser {
	return &Parser{
		state: StateGround,
	}
}

// Reset resets the parser to the Ground state.
func (p *Parser) Reset() {
	p.state = StateGround
	p.intermediatesCount = 0
	p.paramCount = 0
	p.paramAcc = 0
	p.paramSeps = [MaxParams]bool{}
	p.oscBuf = p.oscBuf[:0]
	p.dcsActive = false
}

// State returns the current parser state.
func (p *Parser) State() State {
	return p.state
}

// SetState forces the parser to a specific state.
func (p *Parser) SetState(s State) {
	p.state = s
}

// Next processes a single byte and returns one or more actions.
// Returns up to 3 actions: [exit action, transition action, entry action].
// Unused slots are nil.
func (p *Parser) Next(b byte) [3]Action {
	entry := &stateTable[b][p.state]
	oldState := p.state
	newState := entry.Next
	action := entry.Action

	var result [3]Action
	idx := 0

	// Exit action from old state
	if exitAction := p.exitAction(oldState, newState); exitAction != nil {
		result[idx] = exitAction
		idx++
	}

	// Transition action
	if transitionAction := p.transitionAction(action, b); transitionAction != nil {
		result[idx] = transitionAction
		idx++
	}

	// Entry action for new state
	if entryAction := p.entryAction(newState); entryAction != nil {
		result[idx] = entryAction
		idx++
	}

	p.state = newState
	return result
}

// exitAction returns the action to take when leaving a state.
func (p *Parser) exitAction(old, new State) Action {
	switch old {
	case StateOSCString:
		if new != StateOSCString {
			return p.buildOSCAction()
		}
	case StateDCSPassthrough:
		if new != StateDCSPassthrough {
			p.dcsActive = false
			return DCSUnhookAction{}
		}
	case StateSOSPMAPCString:
		if new != StateSOSPMAPCString {
			return APCEndAction{}
		}
	}
	return nil
}

// transitionAction returns the action for a transition action type.
func (p *Parser) transitionAction(act TransitionAction, b byte) Action {
	switch act {
	case ActionNone, ActionIgnore:
		return nil
	case ActionPrint:
		return PrintAction{Char: rune(b)}
	case ActionExecute:
		return ExecuteAction{Byte: b}
	case ActionCollect:
		if p.intermediatesCount < MaxIntermediates {
			p.intermediates[p.intermediatesCount] = b
			p.intermediatesCount++
		}
		return nil
	case ActionCollectPrivate:
		p.private = true
		return nil
	case ActionParam:
		if p.paramCount == 0 {
			p.paramCount = 1
		}
		d := uint16(b - '0')
		// Saturating arithmetic to prevent overflow
		if p.paramAcc <= 0xFFFF/10 {
			p.paramAcc = p.paramAcc*10 + d
		} else {
			p.paramAcc = 0xFFFF
		}
		return nil
	case ActionParamSeparator:
		p.commitParam()
		// Track separator type for SGR colon sub-parameters
		if p.paramCount-1 < MaxParams {
			p.paramSeps[p.paramCount-1] = (b == ':')
		}
		return nil
	case ActionEscDispatch:
		return p.buildEscDispatch(b)
	case ActionCSIDispatch:
		return p.buildCSIDispatch(b)
	case ActionDCSHook:
		p.dcsActive = true
		return p.buildDCSHook(b)
	case ActionDCSPut:
		return DCSPutAction{Byte: b}
	case ActionDCSUnhook:
		p.dcsActive = false
		return DCSUnhookAction{}
	case ActionOSCPut:
		if len(p.oscBuf) < 4096 { // limit OSC buffer size
			p.oscBuf = append(p.oscBuf, b)
		}
		return nil
	case ActionAPCPut:
		return nil // APC data is ignored
	case ActionOSCStart:
		p.oscBuf = p.oscBuf[:0]
		return nil
	case ActionOSCEnd:
		return p.buildOSCAction()
	case ActionAPCStart:
		return APCStartAction{}
	case ActionAPCEnd:
		return APCEndAction{}
	}
	return nil
}

// entryAction returns the action to take when entering a state.
func (p *Parser) entryAction(s State) Action {
	switch s {
	case StateCSIEntry, StateDCSEntry, StateEscape:
		p.intermediatesCount = 0
		p.paramCount = 0
		p.paramAcc = 0
		p.paramSeps = [MaxParams]bool{}
		p.private = false
	}
	return nil
}

// commitParam saves the current parameter accumulator.
func (p *Parser) commitParam() {
	if p.paramCount == 0 {
		p.paramCount = 1
	}
	if p.paramCount <= MaxParams {
		p.params[p.paramCount-1] = p.paramAcc
	}
	p.paramCount++
	p.paramAcc = 0
}

// buildCSIDispatch creates a CSIDispatchAction from accumulated state.
func (p *Parser) buildCSIDispatch(final byte) CSIDispatchAction {
	p.commitParam()

	var action CSIDispatchAction
	action.Final = final
	action.Private = p.private

	// Copy params (default to 0 for unset)
	count := p.paramCount - 1 // last commitParam incremented past last real param
	if count > MaxParams {
		count = MaxParams
	}
	if count < 0 {
		count = 0
	}
	action.ParamCount = count
	for i := 0; i < count; i++ {
		action.Params[i] = p.params[i]
		action.ParamSeps[i] = p.paramSeps[i]
	}

	// Copy intermediates
	action.IntermediateCount = p.intermediatesCount
	for i := 0; i < p.intermediatesCount; i++ {
		action.Intermediates[i] = p.intermediates[i]
	}

	return action
}

// buildEscDispatch creates an EscDispatchAction from accumulated state.
func (p *Parser) buildEscDispatch(final byte) EscDispatchAction {
	var action EscDispatchAction
	action.Final = final
	action.IntermediateCount = p.intermediatesCount
	for i := 0; i < p.intermediatesCount; i++ {
		action.Intermediates[i] = p.intermediates[i]
	}
	return action
}

// buildDCSHook creates a DCSHookAction from accumulated state.
func (p *Parser) buildDCSHook(final byte) DCSHookAction {
	p.commitParam()

	var action DCSHookAction
	action.Final = final

	count := p.paramCount - 1
	if count > MaxParams {
		count = MaxParams
	}
	if count < 0 {
		count = 0
	}
	action.ParamCount = count
	for i := 0; i < count; i++ {
		action.Params[i] = p.params[i]
	}

	action.IntermediateCount = p.intermediatesCount
	for i := 0; i < p.intermediatesCount; i++ {
		action.Intermediates[i] = p.intermediates[i]
	}

	return action
}

// buildOSCAction creates an OSCDispatchAction from accumulated data.
func (p *Parser) buildOSCAction() OSCDispatchAction {
	action := OSCDispatchAction{
		Command: make([]byte, len(p.oscBuf)),
	}
	copy(action.Command, p.oscBuf)
	return action
}

// Param returns the nth parameter value (1-indexed, 0 = default).
func (a *CSIDispatchAction) Param(n int, def uint16) uint16 {
	if n < 1 || n > a.ParamCount {
		return def
	}
	v := a.Params[n-1]
	if v == 0 {
		return def
	}
	return v
}

// HasColon returns true if the nth parameter was separated by a colon (SGR sub-parameters).
func (a *CSIDispatchAction) HasColon(n int) bool {
	if n < 1 || n > a.ParamCount {
		return false
	}
	return a.ParamSeps[n-1]
}

// ParamOrDefault returns the nth parameter or its default.
func (a *CSIDispatchAction) ParamOrDefault(n int, def uint16) uint16 {
	if n < 1 || n > a.ParamCount {
		return def
	}
	v := a.Params[n-1]
	if v == 0 {
		return def
	}
	return v
}
