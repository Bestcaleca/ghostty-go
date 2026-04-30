package terminal

// StyleID is an index into the style table (for deduplication).
type StyleID uint16

// UnderlineStyle represents the underline decoration style.
type UnderlineStyle uint8

const (
	UnderlineNone   UnderlineStyle = iota
	UnderlineSingle                // ESC[4:1m
	UnderlineDouble                // ESC[4:2m
	UnderlineCurly                 // ESC[4:3m
	UnderlineDotted                // ESC[4:4m
	UnderlineDashed                // ESC[4:5m
)

// Style represents the visual attributes of a cell.
type Style struct {
	FG            Color
	BG            Color
	UL            Color           // underline color
	Bold          bool
	Faint         bool
	Italic        bool
	Underline     UnderlineStyle
	Blink         bool
	RapidBlink    bool
	Reverse       bool
	Invisible     bool
	Strikethrough bool
	Overline      bool
}

// DefaultStyle returns a style with default colors and no decorations.
func DefaultStyle() Style {
	return Style{}
}

// StyleTable manages a deduplicated set of styles.
type StyleTable struct {
	styles []Style
	index  map[Style]StyleID
}

// NewStyleTable creates a new style table with the default style at index 0.
func NewStyleTable() *StyleTable {
	st := &StyleTable{
		styles: make([]Style, 1),
		index:  make(map[Style]StyleID),
	}
	st.styles[0] = DefaultStyle()
	return st
}

// Get returns the StyleID for the given style, creating a new entry if needed.
func (st *StyleTable) Get(s Style) StyleID {
	if id, ok := st.index[s]; ok {
		return id
	}
	id := StyleID(len(st.styles))
	st.styles = append(st.styles, s)
	st.index[s] = id
	return id
}

// Lookup returns the Style for a given StyleID.
func (st *StyleTable) Lookup(id StyleID) Style {
	if int(id) < len(st.styles) {
		return st.styles[id]
	}
	return DefaultStyle()
}

// DefaultID returns the StyleID for the default style.
func (st *StyleTable) DefaultID() StyleID {
	return 0
}
