package parser

import (
	"bytes"
	"strconv"
)

// OSCCommand represents a parsed OSC command.
type OSCCommand interface {
	oscTag()
}

// OSCSetWindowTitle is OSC 0 / OSC 2 — set window title.
type OSCSetWindowTitle struct {
	Title string
}

// OSCSetIconName is OSC 1 — set icon name.
type OSCSetIconName struct {
	Name string
}

// OSCSetColor is OSC 4 — set/query palette color.
type OSCSetColor struct {
	Index int
	R, G, B uint8
	Query bool
}

// OSCSetForegroundColor is OSC 10 — set/query foreground color.
type OSCSetForegroundColor struct {
	R, G, B uint8
	Query bool
}

// OSCSetBackgroundColor is OSC 11 — set/query background color.
type OSCSetBackgroundColor struct {
	R, G, B uint8
	Query bool
}

// OSCSetCursorColor is OSC 12 — set/query cursor color.
type OSCSetCursorColor struct {
	R, G, B uint8
	Query bool
}

// OSCSetHyperlink is OSC 8 — set hyperlink (start or end).
type OSCSetHyperlink struct {
	ID  string
	URI string
}

// OSCSetClipboard is OSC 52 — clipboard manipulation.
type OSCSetClipboard struct {
	Clipboard string // "c" (clipboard), "p" (primary), etc.
	Data      []byte // base64-encoded data
	Query     bool
}

// OSCResetColor is OSC 104 — reset palette color.
type OSCResetColor struct {
	Index int // -1 = reset all
}

// OSCUnknown is an unrecognized OSC command.
type OSCUnknown struct {
	Number int
	Data   []byte
}

func (OSCSetWindowTitle) oscTag()    {}
func (OSCSetIconName) oscTag()      {}
func (OSCSetColor) oscTag()         {}
func (OSCSetForegroundColor) oscTag() {}
func (OSCSetBackgroundColor) oscTag() {}
func (OSCSetCursorColor) oscTag()    {}
func (OSCSetHyperlink) oscTag()      {}
func (OSCSetClipboard) oscTag()      {}
func (OSCResetColor) oscTag()        {}
func (OSCUnknown) oscTag()           {}

// ParseOSC parses raw OSC command data into a structured command.
// The data is the raw bytes between OSC start and BEL/ST terminator.
func ParseOSC(data []byte) OSCCommand {
	// Find the semicolon separator between command number and payload
	semi := bytes.IndexByte(data, ';')
	if semi < 0 {
		// No payload — try to parse as a number-only command
		num, err := strconv.Atoi(string(data))
		if err != nil {
			return OSCUnknown{Data: data}
		}
		return parseOSCEmptyCommand(num)
	}

	num, err := strconv.Atoi(string(data[:semi]))
	if err != nil {
		return OSCUnknown{Data: data}
	}

	payload := data[semi+1:]

	switch num {
	case 0, 2:
		return OSCSetWindowTitle{Title: string(payload)}
	case 1:
		return OSCSetIconName{Name: string(payload)}
	case 4:
		return parseOSCColor(payload)
	case 8:
		return parseOSCHyperlink(payload)
	case 10:
		return parseOSCRGBColor(payload, func(r, g, b uint8, q bool) OSCCommand {
			return OSCSetForegroundColor{R: r, G: g, B: b, Query: q}
		})
	case 11:
		return parseOSCRGBColor(payload, func(r, g, b uint8, q bool) OSCCommand {
			return OSCSetBackgroundColor{R: r, G: g, B: b, Query: q}
		})
	case 12:
		return parseOSCRGBColor(payload, func(r, g, b uint8, q bool) OSCCommand {
			return OSCSetCursorColor{R: r, G: g, B: b, Query: q}
		})
	case 52:
		return parseOSCClipboard(payload)
	case 104:
		return parseOSCResetColor(payload)
	default:
		return OSCUnknown{Number: num, Data: payload}
	}
}

func parseOSCEmptyCommand(num int) OSCCommand {
	switch num {
	case 10:
		return OSCSetForegroundColor{Query: true}
	case 11:
		return OSCSetBackgroundColor{Query: true}
	case 12:
		return OSCSetCursorColor{Query: true}
	default:
		return OSCUnknown{Number: num}
	}
}

func parseOSCColor(data []byte) OSCSetColor {
	// Format: index;rgb:RR/GG/BB
	semi := bytes.IndexByte(data, ';')
	if semi < 0 {
		return OSCSetColor{Query: true}
	}

	idx, err := strconv.Atoi(string(data[:semi]))
	if err != nil {
		return OSCSetColor{Query: true}
	}

	colorStr := data[semi+1:]
	if bytes.HasPrefix(colorStr, []byte("rgb:")) {
		r, g, b, ok := parseXColor(colorStr[4:])
		if ok {
			return OSCSetColor{Index: idx, R: r, G: g, B: b}
		}
	}

	return OSCSetColor{Index: idx, Query: true}
}

func parseOSCRGBColor(data []byte, mk func(r, g, b uint8, q bool) OSCCommand) OSCCommand {
	if len(data) == 0 || bytes.Equal(data, []byte("?")) {
		return mk(0, 0, 0, true)
	}
	if bytes.HasPrefix(data, []byte("rgb:")) {
		r, g, b, ok := parseXColor(data[4:])
		if ok {
			return mk(r, g, b, false)
		}
	}
	return mk(0, 0, 0, false)
}

// parseXColor parses an X11 color spec like "RR/RR/RR" or "RRRR/RRRR/RRRR"
func parseXColor(data []byte) (r, g, b uint8, ok bool) {
	parts := bytes.Split(data, []byte("/"))
	if len(parts) != 3 {
		return 0, 0, 0, false
	}

	rv, err := parseHexColor(parts[0])
	if err != nil {
		return 0, 0, 0, false
	}
	gv, err := parseHexColor(parts[1])
	if err != nil {
		return 0, 0, 0, false
	}
	bv, err := parseHexColor(parts[2])
	if err != nil {
		return 0, 0, 0, false
	}

	return rv, gv, bv, true
}

func parseHexColor(data []byte) (uint8, error) {
	v, err := strconv.ParseUint(string(data), 16, 32)
	if err != nil {
		return 0, err
	}
	switch len(data) {
	case 2:
		return uint8(v), nil
	case 4:
		return uint8(v >> 8), nil
	default:
		return uint8(v), nil
	}
}

func parseOSCHyperlink(data []byte) OSCSetHyperlink {
	// Format: id=xxx;uri=xxx
	parts := bytes.SplitN(data, []byte(";"), 2)
	if len(parts) < 2 {
		return OSCSetHyperlink{}
	}

	var id, uri string
	for _, part := range parts {
		if bytes.HasPrefix(part, []byte("id=")) {
			id = string(part[3:])
		} else if bytes.HasPrefix(part, []byte("uri=")) {
			uri = string(part[4:])
		}
	}

	return OSCSetHyperlink{ID: id, URI: uri}
}

func parseOSCClipboard(data []byte) OSCSetClipboard {
	// Format: clipboard;b64data or clipboard;?
	if len(data) < 1 {
		return OSCSetClipboard{}
	}

	semi := bytes.IndexByte(data, ';')
	if semi < 0 {
		return OSCSetClipboard{Clipboard: string(data)}
	}

	clipboard := string(data[:semi])
	payload := data[semi+1:]

	if bytes.Equal(payload, []byte("?")) {
		return OSCSetClipboard{Clipboard: clipboard, Query: true}
	}

	return OSCSetClipboard{Clipboard: clipboard, Data: payload}
}

func parseOSCResetColor(data []byte) OSCResetColor {
	if len(data) == 0 {
		return OSCResetColor{Index: -1}
	}
	idx, err := strconv.Atoi(string(data))
	if err != nil {
		return OSCResetColor{Index: -1}
	}
	return OSCResetColor{Index: idx}
}
