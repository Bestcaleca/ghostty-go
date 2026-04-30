package parser

// Entry represents a parser state transition: the next state and the action to take.
type Entry struct {
	Next   State
	Action TransitionAction
}

// stateTable is a [256][14] lookup table indexed by [byte][current_state].
// Each entry gives the next state and transition action.
var stateTable [256][14]Entry

func init() {
	// Initialize all entries to {current state, ActionNone} (no-op, stay in state)
	for b := 0; b < 256; b++ {
		for s := 0; s < 14; s++ {
			stateTable[b][s] = Entry{State(s), ActionNone}
		}
	}

	// === ANYWHERE transitions (apply to all states) ===
	for s := 0; s < 14; s++ {
		// 0x18, 0x1A -> Ground (execute C0 CAN/SUB)
		stateTable[0x18][s] = Entry{StateGround, ActionExecute}
		stateTable[0x1A][s] = Entry{StateGround, ActionExecute}

		// 0x1B -> Escape (cancel and restart)
		stateTable[0x1B][s] = Entry{StateEscape, ActionNone}

		// C1 control characters (80-9F)
		// These map to: CSI=0x9B, OSC=0x9D, DCS=0x90, etc.
		// But in UTF-8 mode, bytes 80-9F don't appear as single bytes
		// We handle the standard single-byte C1 controls for legacy mode
		for _, b := range []byte{0x80, 0x81, 0x82, 0x83, 0x84, 0x85, 0x86, 0x87,
			0x88, 0x89, 0x8A, 0x8B, 0x8C, 0x8D, 0x8E, 0x8F} {
			stateTable[b][s] = Entry{StateGround, ActionExecute}
		}
		// 0x90 -> DCS Entry
		stateTable[0x90][s] = Entry{StateDCSEntry, ActionNone}
		// 0x9B -> CSI Entry
		stateTable[0x9B][s] = Entry{StateCSIEntry, ActionNone}
		// 0x9C -> Ground (ST - string terminator)
		stateTable[0x9C][s] = Entry{StateGround, ActionNone}
		// 0x9D -> OSC String
		stateTable[0x9D][s] = Entry{StateOSCString, ActionOSCStart}
		// 0x9E -> SOS/PM/APC
		stateTable[0x9E][s] = Entry{StateSOSPMAPCString, ActionNone}
		// 0x9F -> APC
		stateTable[0x9F][s] = Entry{StateSOSPMAPCString, ActionAPCStart}
	}

	// === GROUND state ===
	// C0 control characters: execute and stay in ground
	for _, b := range []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07,
		0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F,
		0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17,
		0x19, 0x1C, 0x1D, 0x1E, 0x1F} {
		stateTable[b][StateGround] = Entry{StateGround, ActionExecute}
	}

	// Printable characters (0x20-0x7E): print and stay in ground
	for b := 0x20; b <= 0x7E; b++ {
		stateTable[b][StateGround] = Entry{StateGround, ActionPrint}
	}

	// 0x7F (DEL): ignore in ground
	stateTable[0x7F][StateGround] = Entry{StateGround, ActionIgnore}

	// === ESCAPE state ===
	// C0 controls: execute, stay in escape
	for _, b := range []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07,
		0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F,
		0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17,
		0x19, 0x1C, 0x1D, 0x1E, 0x1F} {
		stateTable[b][StateEscape] = Entry{StateEscape, ActionExecute}
	}

	// Intermediate bytes (0x20-0x2F): collect
	for b := 0x20; b <= 0x2F; b++ {
		stateTable[b][StateEscape] = Entry{StateEscapeIntermediate, ActionCollect}
	}

	// Final bytes (0x30-0x4F, 0x51-0x57, 0x59, 0x5A, 0x5C, 0x60-0x7E): ESC dispatch
	for b := 0x30; b <= 0x4F; b++ {
		stateTable[b][StateEscape] = Entry{StateGround, ActionEscDispatch}
	}
	stateTable[0x51][StateEscape] = Entry{StateGround, ActionEscDispatch} // SCI
	stateTable[0x52][StateEscape] = Entry{StateGround, ActionEscDispatch}
	stateTable[0x53][StateEscape] = Entry{StateGround, ActionEscDispatch}
	stateTable[0x54][StateEscape] = Entry{StateGround, ActionEscDispatch}
	stateTable[0x55][StateEscape] = Entry{StateGround, ActionEscDispatch}
	stateTable[0x56][StateEscape] = Entry{StateGround, ActionEscDispatch}
	stateTable[0x57][StateEscape] = Entry{StateGround, ActionEscDispatch}
	stateTable[0x59][StateEscape] = Entry{StateGround, ActionEscDispatch}
	stateTable[0x5A][StateEscape] = Entry{StateGround, ActionEscDispatch}
	stateTable[0x5C][StateEscape] = Entry{StateGround, ActionEscDispatch} // ST
	for b := 0x60; b <= 0x7E; b++ {
		stateTable[b][StateEscape] = Entry{StateGround, ActionEscDispatch}
	}

	// Special escape transitions
	stateTable[0x50][StateEscape] = Entry{StateDCSEntry, ActionNone}        // P -> DCS
	stateTable[0x58][StateEscape] = Entry{StateSOSPMAPCString, ActionNone}  // X -> SOS
	stateTable[0x5B][StateEscape] = Entry{StateCSIEntry, ActionNone}        // [ -> CSI
	stateTable[0x5D][StateEscape] = Entry{StateOSCString, ActionOSCStart}   // ] -> OSC
	stateTable[0x5E][StateEscape] = Entry{StateSOSPMAPCString, ActionNone}  // ^ -> PM
	stateTable[0x5F][StateEscape] = Entry{StateSOSPMAPCString, ActionAPCStart} // _ -> APC

	// 0x7F in escape: ignore
	stateTable[0x7F][StateEscape] = Entry{StateEscape, ActionIgnore}

	// === ESCAPE INTERMEDIATE state ===
	for _, b := range []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07,
		0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F,
		0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17,
		0x19, 0x1C, 0x1D, 0x1E, 0x1F} {
		stateTable[b][StateEscapeIntermediate] = Entry{StateEscapeIntermediate, ActionExecute}
	}
	for b := 0x20; b <= 0x2F; b++ {
		stateTable[b][StateEscapeIntermediate] = Entry{StateEscapeIntermediate, ActionCollect}
	}
	for b := 0x30; b <= 0x7E; b++ {
		stateTable[b][StateEscapeIntermediate] = Entry{StateGround, ActionEscDispatch}
	}
	stateTable[0x7F][StateEscapeIntermediate] = Entry{StateEscapeIntermediate, ActionIgnore}

	// === CSI ENTRY state ===
	for _, b := range []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07,
		0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F,
		0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17,
		0x19, 0x1C, 0x1D, 0x1E, 0x1F} {
		stateTable[b][StateCSIEntry] = Entry{StateCSIEntry, ActionExecute}
	}
	// Parameter bytes: digits (0x30-0x39), semicolon (0x3B), colon (0x3A)
	for b := 0x30; b <= 0x39; b++ {
		stateTable[b][StateCSIEntry] = Entry{StateCSIParam, ActionParam}
	}
	stateTable[0x3A][StateCSIEntry] = Entry{StateCSIIgnore, ActionNone} // colon before params = ignore
	stateTable[0x3B][StateCSIEntry] = Entry{StateCSIParam, ActionParamSeparator}
	// Intermediate bytes (0x20-0x2F)
	for b := 0x20; b <= 0x2F; b++ {
		stateTable[b][StateCSIEntry] = Entry{StateCSIIntermediate, ActionCollect}
	}
	// Final bytes (0x40-0x7E): CSI dispatch
	for b := 0x40; b <= 0x7E; b++ {
		stateTable[b][StateCSIEntry] = Entry{StateGround, ActionCSIDispatch}
	}
	stateTable[0x7F][StateCSIEntry] = Entry{StateCSIEntry, ActionIgnore}
	// Invalid: 0x3C-0x3E -> CSI Ignore
	for b := 0x3C; b <= 0x3E; b++ {
		stateTable[b][StateCSIEntry] = Entry{StateCSIIgnore, ActionNone}
	}
	// 0x3F (?) = private mode prefix -> collect and enter param state
	stateTable[0x3F][StateCSIEntry] = Entry{StateCSIParam, ActionCollectPrivate}

	// === CSI PARAM state ===
	for _, b := range []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07,
		0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F,
		0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17,
		0x19, 0x1C, 0x1D, 0x1E, 0x1F} {
		stateTable[b][StateCSIParam] = Entry{StateCSIParam, ActionExecute}
	}
	for b := 0x30; b <= 0x39; b++ {
		stateTable[b][StateCSIParam] = Entry{StateCSIParam, ActionParam}
	}
	stateTable[0x3A][StateCSIParam] = Entry{StateCSIParam, ActionParamSeparator} // colon separator
	stateTable[0x3B][StateCSIParam] = Entry{StateCSIParam, ActionParamSeparator} // semicolon separator
	for b := 0x20; b <= 0x2F; b++ {
		stateTable[b][StateCSIParam] = Entry{StateCSIIntermediate, ActionCollect}
	}
	for b := 0x40; b <= 0x7E; b++ {
		stateTable[b][StateCSIParam] = Entry{StateGround, ActionCSIDispatch}
	}
	stateTable[0x7F][StateCSIParam] = Entry{StateCSIParam, ActionIgnore}
	for b := 0x3C; b <= 0x3E; b++ {
		stateTable[b][StateCSIParam] = Entry{StateCSIIgnore, ActionNone}
	}

	// === CSI INTERMEDIATE state ===
	for _, b := range []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07,
		0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F,
		0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17,
		0x19, 0x1C, 0x1D, 0x1E, 0x1F} {
		stateTable[b][StateCSIIntermediate] = Entry{StateCSIIntermediate, ActionExecute}
	}
	for b := 0x20; b <= 0x2F; b++ {
		stateTable[b][StateCSIIntermediate] = Entry{StateCSIIntermediate, ActionCollect}
	}
	for b := 0x40; b <= 0x7E; b++ {
		stateTable[b][StateCSIIntermediate] = Entry{StateGround, ActionCSIDispatch}
	}
	stateTable[0x7F][StateCSIIntermediate] = Entry{StateCSIIntermediate, ActionIgnore}
	for b := 0x30; b <= 0x3F; b++ {
		stateTable[b][StateCSIIntermediate] = Entry{StateCSIIgnore, ActionNone}
	}

	// === CSI IGNORE state ===
	// Ignore everything until a final byte (0x40-0x7E)
	for _, b := range []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07,
		0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F,
		0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17,
		0x19, 0x1C, 0x1D, 0x1E, 0x1F} {
		stateTable[b][StateCSIIgnore] = Entry{StateCSIIgnore, ActionExecute}
	}
	for b := 0x20; b <= 0x3F; b++ {
		stateTable[b][StateCSIIgnore] = Entry{StateCSIIgnore, ActionNone}
	}
	for b := 0x40; b <= 0x7E; b++ {
		stateTable[b][StateCSIIgnore] = Entry{StateGround, ActionNone}
	}
	stateTable[0x7F][StateCSIIgnore] = Entry{StateCSIIgnore, ActionIgnore}

	// === DCS ENTRY state ===
	for _, b := range []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07,
		0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F,
		0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17,
		0x19, 0x1C, 0x1D, 0x1E, 0x1F} {
		stateTable[b][StateDCSEntry] = Entry{StateDCSEntry, ActionExecute}
	}
	for b := 0x30; b <= 0x39; b++ {
		stateTable[b][StateDCSEntry] = Entry{StateDCSParam, ActionParam}
	}
	stateTable[0x3B][StateDCSEntry] = Entry{StateDCSParam, ActionParamSeparator}
	for b := 0x20; b <= 0x2F; b++ {
		stateTable[b][StateDCSEntry] = Entry{StateDCSIntermediate, ActionCollect}
	}
	for b := 0x40; b <= 0x7E; b++ {
		stateTable[b][StateDCSEntry] = Entry{StateDCSPassthrough, ActionDCSHook}
	}
	stateTable[0x7F][StateDCSEntry] = Entry{StateDCSEntry, ActionIgnore}
	for b := 0x3C; b <= 0x3E; b++ {
		stateTable[b][StateDCSEntry] = Entry{StateDCSIgnore, ActionNone}
	}

	// === DCS PARAM state ===
	for _, b := range []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07,
		0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F,
		0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17,
		0x19, 0x1C, 0x1D, 0x1E, 0x1F} {
		stateTable[b][StateDCSParam] = Entry{StateDCSParam, ActionExecute}
	}
	for b := 0x30; b <= 0x39; b++ {
		stateTable[b][StateDCSParam] = Entry{StateDCSParam, ActionParam}
	}
	stateTable[0x3B][StateDCSParam] = Entry{StateDCSParam, ActionParamSeparator}
	for b := 0x20; b <= 0x2F; b++ {
		stateTable[b][StateDCSParam] = Entry{StateDCSIntermediate, ActionCollect}
	}
	for b := 0x40; b <= 0x7E; b++ {
		stateTable[b][StateDCSParam] = Entry{StateDCSPassthrough, ActionDCSHook}
	}
	stateTable[0x7F][StateDCSParam] = Entry{StateDCSParam, ActionIgnore}
	for b := 0x3C; b <= 0x3E; b++ {
		stateTable[b][StateDCSParam] = Entry{StateDCSIgnore, ActionNone}
	}

	// === DCS INTERMEDIATE state ===
	for _, b := range []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07,
		0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F,
		0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17,
		0x19, 0x1C, 0x1D, 0x1E, 0x1F} {
		stateTable[b][StateDCSIntermediate] = Entry{StateDCSIntermediate, ActionExecute}
	}
	for b := 0x20; b <= 0x2F; b++ {
		stateTable[b][StateDCSIntermediate] = Entry{StateDCSIntermediate, ActionCollect}
	}
	for b := 0x40; b <= 0x7E; b++ {
		stateTable[b][StateDCSIntermediate] = Entry{StateDCSPassthrough, ActionDCSHook}
	}
	stateTable[0x7F][StateDCSIntermediate] = Entry{StateDCSIntermediate, ActionIgnore}
	for b := 0x30; b <= 0x3F; b++ {
		stateTable[b][StateDCSIntermediate] = Entry{StateDCSIgnore, ActionNone}
	}

	// === DCS PASSTHROUGH state ===
	// All bytes are passed through as data, except ESC (which starts exit)
	for _, b := range []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07,
		0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F,
		0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17,
		0x19, 0x1C, 0x1D, 0x1E, 0x1F} {
		stateTable[b][StateDCSPassthrough] = Entry{StateDCSPassthrough, ActionDCSPut}
	}
	for b := 0x20; b <= 0x7E; b++ {
		stateTable[b][StateDCSPassthrough] = Entry{StateDCSPassthrough, ActionDCSPut}
	}
	stateTable[0x7F][StateDCSPassthrough] = Entry{StateDCSPassthrough, ActionIgnore}
	// 0x9C (ST) ends DCS - handled by anywhere transition to Ground, then exit hook
	// ESC (0x1B) is handled by anywhere transitions -> Escape -> exit triggers DCSUnhook

	// === DCS IGNORE state ===
	// Ignore everything until ST (0x9C) or ESC+backslash
	for _, b := range []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07,
		0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F,
		0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17,
		0x19, 0x1C, 0x1D, 0x1E, 0x1F} {
		stateTable[b][StateDCSIgnore] = Entry{StateDCSIgnore, ActionNone}
	}
	for b := 0x20; b <= 0x7E; b++ {
		stateTable[b][StateDCSIgnore] = Entry{StateDCSIgnore, ActionNone}
	}
	stateTable[0x7F][StateDCSIgnore] = Entry{StateDCSIgnore, ActionIgnore}

	// === OSC STRING state ===
	// Accumulate bytes until BEL (0x07) or ST (0x9C, or ESC+\)
	// All printable + control bytes are collected
	for _, b := range []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06} {
		stateTable[b][StateOSCString] = Entry{StateOSCString, ActionIgnore}
	}
	stateTable[0x07][StateOSCString] = Entry{StateGround, ActionOSCEnd} // BEL terminates OSC
	for _, b := range []byte{0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F,
		0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17,
		0x19, 0x1C, 0x1D, 0x1E, 0x1F} {
		stateTable[b][StateOSCString] = Entry{StateOSCString, ActionIgnore}
	}
	// 0x20-0x7E: OSC data
	for b := 0x20; b <= 0x7E; b++ {
		stateTable[b][StateOSCString] = Entry{StateOSCString, ActionOSCPut}
	}
	stateTable[0x7F][StateOSCString] = Entry{StateOSCString, ActionIgnore}

	// === SOS/PM/APC STRING state ===
	// Ignore everything until BEL or ST
	for _, b := range []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06} {
		stateTable[b][StateSOSPMAPCString] = Entry{StateSOSPMAPCString, ActionNone}
	}
	stateTable[0x07][StateSOSPMAPCString] = Entry{StateGround, ActionNone} // BEL terminates
	for _, b := range []byte{0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F,
		0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17,
		0x19, 0x1C, 0x1D, 0x1E, 0x1F} {
		stateTable[b][StateSOSPMAPCString] = Entry{StateSOSPMAPCString, ActionNone}
	}
	for b := 0x20; b <= 0x7E; b++ {
		stateTable[b][StateSOSPMAPCString] = Entry{StateSOSPMAPCString, ActionAPCPut}
	}
	stateTable[0x7F][StateSOSPMAPCString] = Entry{StateSOSPMAPCString, ActionIgnore}
}
