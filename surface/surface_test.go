package surface

import (
	"testing"

	"github.com/ghostty-go/ghostty-go/renderer"
	"github.com/ghostty-go/ghostty-go/terminal"
)

func TestEncodeUTF8(t *testing.T) {
	tests := []struct {
		r    rune
		want []byte
	}{
		{'A', []byte{0x41}},
		{'é', []byte{0xC3, 0xA9}},
		{'中', []byte{0xE4, 0xB8, 0xAD}},
		{'😀', []byte{0xF0, 0x9F, 0x98, 0x80}},
	}

	for _, tt := range tests {
		buf := make([]byte, 4)
		n := encodeUTF8(tt.r, buf)
		if n != len(tt.want) {
			t.Errorf("encodeUTF8(%U) = %d bytes, want %d", tt.r, n, len(tt.want))
			continue
		}
		for i := 0; i < n; i++ {
			if buf[i] != tt.want[i] {
				t.Errorf("encodeUTF8(%U)[%d] = 0x%02X, want 0x%02X", tt.r, i, buf[i], tt.want[i])
			}
		}
	}
}

func TestStyleToColor(t *testing.T) {
	fallback := renderer.Color{R: 0.5, G: 0.5, B: 0.5, A: 1.0}

	// Invalid color returns fallback
	c := styleToColor(terminal.Color{}, fallback)
	if c.R != 0.5 {
		t.Errorf("expected fallback R=0.5, got %f", c.R)
	}

	// Valid color
	c = styleToColor(terminal.Color{R: 255, G: 128, B: 0, Valid: true}, fallback)
	if c.R != 1.0 {
		t.Errorf("expected R=1.0, got %f", c.R)
	}
	if c.G < 0.50 || c.G > 0.51 {
		t.Errorf("expected G≈0.502, got %f", c.G)
	}
}
