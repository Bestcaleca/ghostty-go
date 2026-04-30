package unicode

import (
	"testing"
)

func TestRuneWidth(t *testing.T) {
	tests := []struct {
		r    rune
		want int
	}{
		// ASCII
		{'A', 1},
		{'z', 1},
		{'0', 1},
		{' ', 1},
		{'~', 1},

		// Control characters
		{0x00, 0}, // NUL
		{0x07, 0}, // BEL
		{0x08, 0}, // BS
		{0x09, 0}, // HT
		{0x0A, 0}, // LF
		{0x0D, 0}, // CR
		{0x1B, 0}, // ESC
		{0x7F, 0}, // DEL

		// CJK
		{'中', 2},
		{'文', 2},
		{'日', 2},
		{'本', 2},
		{'語', 2},
		{'한', 2},
		{'글', 2},

		// Fullwidth
		{'Ａ', 2}, // Fullwidth A
		{'０', 2}, // Fullwidth 0

		// Emoji (most are wide)
		{'😀', 2},
		{'🎉', 2},

		// Combining marks (zero width)
		{0x0301, 0}, // Combining acute accent
		{0x0300, 0}, // Combining grave accent
		{0xFE00, 0}, // Variation selector
	}

	for _, tt := range tests {
		got := RuneWidth(tt.r)
		if got != tt.want {
			t.Errorf("RuneWidth(%U) = %d, want %d", tt.r, got, tt.want)
		}
	}
}

func TestStringWidth(t *testing.T) {
	tests := []struct {
		s    string
		want int
	}{
		{"Hello", 5},
		{"Hello, 世界!", 12}, // 7 ASCII + 2 CJK (2 each) + 1 ASCII = 7+4+1=12
		{"", 0},
		{"abc", 3},
		{"中文字", 6},
	}

	for _, tt := range tests {
		got := StringWidth(tt.s)
		if got != tt.want {
			t.Errorf("StringWidth(%q) = %d, want %d", tt.s, got, tt.want)
		}
	}
}

func TestTruncateWidth(t *testing.T) {
	tests := []struct {
		s        string
		maxWidth int
		want     string
	}{
		{"Hello", 3, "Hel"},
		{"Hello", 10, "Hello"},
		{"中文字", 3, "中"},     // 2 ≤ 3, 2+2=4 > 3 → only "中"
		{"中文字", 4, "中文"},  // 2+2=4 fits
		{"中文字", 5, "中文"},  // 2+2=4 ≤ 5, 2+2+2=6 > 5
		{"中文字", 6, "中文字"}, // 2+2+2=6 fits
		{"ab中cd", 5, "ab中c"}, // 1+1+2+1=5 fits
	}

	for _, tt := range tests {
		got := TruncateWidth(tt.s, tt.maxWidth)
		if got != tt.want {
			t.Errorf("TruncateWidth(%q, %d) = %q, want %q", tt.s, tt.maxWidth, got, tt.want)
		}
	}
}

func TestGraphemeClusters(t *testing.T) {
	tests := []struct {
		s       string
		wantLen int
	}{
		{"Hello", 5},       // 5 ASCII graphemes
		{"中文字", 3},       // 3 CJK graphemes
		{"é", 1},           // precomposed (single code point)
		{"é", 1},     // e + combining acute = 1 grapheme
		{"👨‍👩‍👧‍👦", 1},     // family emoji ZWJ sequence = 1 grapheme
		{"🏳️‍🌈", 1},         // flag + ZWJ sequence
		{"🇺🇸", 1},          // US flag (2 regional indicators = 1 grapheme with GB12/13)
		{"", 0},            // empty
		{"a\rb", 3},        // CR breaks: a, CR, b
		{"a\nb", 3},        // LF breaks: a, LF, b
		{"a\r\nb", 3},      // CR+LF = 1 break: a, CR+LF, b
	}

	for _, tt := range tests {
		clusters := GraphemeClusters(tt.s)
		if len(clusters) != tt.wantLen {
			t.Errorf("GraphemeClusters(%q) = %d clusters, want %d", tt.s, len(clusters), tt.wantLen)
		}
	}
}

func TestGraphemeClusterWidth(t *testing.T) {
	// Single ASCII
	clusters := GraphemeClusters("A")
	if clusters[0].Width != 1 {
		t.Errorf("cluster width for 'A' = %d, want 1", clusters[0].Width)
	}

	// CJK
	clusters = GraphemeClusters("中")
	if clusters[0].Width != 2 {
		t.Errorf("cluster width for '中' = %d, want 2", clusters[0].Width)
	}

	// e + combining acute (should be width 1)
	clusters = GraphemeClusters("é")
	if clusters[0].Width != 1 {
		t.Errorf("cluster width for 'é' (e+combining) = %d, want 1", clusters[0].Width)
	}

	// Emoji
	clusters = GraphemeClusters("😀")
	if clusters[0].Width != 2 {
		t.Errorf("cluster width for '😀' = %d, want 2", clusters[0].Width)
	}
}

func TestIsRegionalIndicator(t *testing.T) {
	// 🇺 = U+1F1FA (Regional Indicator U)
	if !isRegionalIndicator(0x1F1FA) {
		t.Error("expected U+1F1FA to be regional indicator")
	}
	// 🇸 = U+1F1F8 (Regional Indicator S)
	if !isRegionalIndicator(0x1F1F8) {
		t.Error("expected U+1F1F8 to be regional indicator")
	}
	// 'A' is not
	if isRegionalIndicator('A') {
		t.Error("expected 'A' to not be regional indicator")
	}
}

func TestIsExtendedPictographic(t *testing.T) {
	tests := []struct {
		r    rune
		want bool
	}{
		{'😀', true},     // emoji
		{'🎉', true},     // emoji
		{0x2600, true},   // ☀ sun
		{0x2764, true},   // ❤ heart
		{'A', false},
		{'中', false},
	}

	for _, tt := range tests {
		got := isExtendedPictographic(tt.r)
		if got != tt.want {
			t.Errorf("isExtendedPictographic(%U) = %v, want %v", tt.r, got, tt.want)
		}
	}
}
