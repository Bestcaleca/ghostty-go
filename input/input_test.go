package input

import (
	"testing"

	"github.com/go-gl/glfw/v3.3/glfw"
)

func TestModifierEncode(t *testing.T) {
	tests := []struct {
		name string
		mods Modifiers
		want int
	}{
		{"none", Modifiers{}, 1},
		{"shift", Modifiers{Shift: true}, 2},
		{"alt", Modifiers{Alt: true}, 3},
		{"control", Modifiers{Control: true}, 5},
		{"shift+control", Modifiers{Shift: true, Control: true}, 6},
		{"all", Modifiers{Shift: true, Alt: true, Control: true, Super: true}, 16},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.mods.Encode()
			if got != tt.want {
				t.Errorf("Encode() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestEncodeCSI(t *testing.T) {
	tests := []struct {
		number string
		mod    int
		suffix string
		want   string
	}{
		{"A", 1, "", "\x1b[A"},
		{"A", 2, "", "\x1b[1;2A"},
		{"2", 1, "~", "\x1b[2~"},
		{"2", 3, "~", "\x1b[2;3~"},
	}

	for _, tt := range tests {
		var got []byte
		if tt.suffix != "" {
			got = encodeCSI(tt.number, tt.mod, tt.suffix)
		} else {
			got = encodeCSI(tt.number, tt.mod)
		}
		if string(got) != tt.want {
			t.Errorf("encodeCSI(%q, %d, %q) = %q, want %q", tt.number, tt.mod, tt.suffix, string(got), tt.want)
		}
	}
}

func TestEncodeSS3(t *testing.T) {
	tests := []struct {
		char string
		mod  int
		want string
	}{
		{"A", 1, "\x1bOA"},
		{"C", 2, "\x1bO1;2C"},
	}

	for _, tt := range tests {
		got := encodeSS3(tt.char, tt.mod)
		if string(got) != tt.want {
			t.Errorf("encodeSS3(%q, %d) = %q, want %q", tt.char, tt.mod, string(got), tt.want)
		}
	}
}

func TestKeyHandlerCursorKeys(t *testing.T) {
	kh := NewKeyHandler()

	// Normal mode: cursor keys use SS3
	seq := kh.EncodeKey(glfw.KeyUp, glfw.Press, Modifiers{})
	if string(seq) != "\x1bOA" {
		t.Errorf("normal up = %q, want %q", string(seq), "\x1bOA")
	}

	// Application mode: cursor keys use CSI
	kh.SetApplicationCursorKeys(true)
	seq = kh.EncodeKey(glfw.KeyUp, glfw.Press, Modifiers{})
	if string(seq) != "\x1b[A" {
		t.Errorf("app up = %q, want %q", string(seq), "\x1b[A")
	}
}

func TestKeyHandlerControlKeys(t *testing.T) {
	kh := NewKeyHandler()

	tests := []struct {
		key  glfw.Key
		want byte
	}{
		{glfw.KeyC, 0x03},
		{glfw.KeyD, 0x04},
		{glfw.KeyA, 0x01},
		{glfw.KeyZ, 0x1A},
	}

	for _, tt := range tests {
		seq := kh.EncodeKey(tt.key, glfw.Press, Modifiers{Control: true})
		if len(seq) != 1 || seq[0] != tt.want {
			t.Errorf("Ctrl+%c = %v, want [%d]", rune(tt.key), seq, tt.want)
		}
	}
}

func TestKeyHandlerFunctionKeys(t *testing.T) {
	kh := NewKeyHandler()

	seq := kh.EncodeKey(glfw.KeyF1, glfw.Press, Modifiers{})
	if string(seq) != "\x1b[11~" {
		t.Errorf("F1 = %q, want %q", string(seq), "\x1b[11~")
	}

	seq = kh.EncodeKey(glfw.KeyF12, glfw.Press, Modifiers{})
	if string(seq) != "\x1b[24~" {
		t.Errorf("F12 = %q, want %q", string(seq), "\x1b[24~")
	}
}

func TestKeyHandlerRelease(t *testing.T) {
	kh := NewKeyHandler()

	seq := kh.EncodeKey(glfw.KeyA, glfw.Release, Modifiers{})
	if seq != nil {
		t.Errorf("release should return nil, got %v", seq)
	}
}

func TestMouseHandlerNone(t *testing.T) {
	mh := NewMouseHandler()

	seq := mh.EncodeMouseButton(glfw.MouseButtonLeft, glfw.Press, Modifiers{}, 5, 10)
	if seq != nil {
		t.Errorf("none mode should return nil, got %v", seq)
	}
}

func TestMouseHandlerSGR(t *testing.T) {
	mh := NewMouseHandler()
	mh.SetMode(MouseModeNormal)
	mh.SetSGR(true)

	seq := mh.EncodeMouseButton(glfw.MouseButtonLeft, glfw.Press, Modifiers{}, 5, 10)
	want := "\x1b[<0;6;11M"
	if string(seq) != want {
		t.Errorf("SGR press = %q, want %q", string(seq), want)
	}

	seq = mh.EncodeMouseButton(glfw.MouseButtonLeft, glfw.Release, Modifiers{}, 5, 10)
	want = "\x1b[<3;6;11m"
	if string(seq) != want {
		t.Errorf("SGR release = %q, want %q", string(seq), want)
	}
}

func TestMouseHandlerButtonModeOnlyReportsMotionWithButton(t *testing.T) {
	mh := NewMouseHandler()
	mh.SetMode(MouseModeButton)
	mh.SetSGR(true)

	seq := mh.EncodeMouseMotion(nil, Modifiers{}, 5, 10)
	if seq != nil {
		t.Fatalf("button mouse mode should ignore passive motion, got %q", seq)
	}

	seq = mh.EncodeMouseMotion([]glfw.MouseButton{glfw.MouseButtonLeft}, Modifiers{}, 5, 10)
	want := "\x1b[<32;6;11M"
	if string(seq) != want {
		t.Fatalf("button drag = %q, want %q", string(seq), want)
	}
}

func TestMouseHandlerAnyModeReportsPassiveMotion(t *testing.T) {
	mh := NewMouseHandler()
	mh.SetMode(MouseModeAny)
	mh.SetSGR(true)

	seq := mh.EncodeMouseMotion(nil, Modifiers{}, 5, 10)
	want := "\x1b[<35;6;11M"
	if string(seq) != want {
		t.Fatalf("any passive motion = %q, want %q", string(seq), want)
	}
}

func TestMouseHandlerScroll(t *testing.T) {
	mh := NewMouseHandler()
	mh.SetMode(MouseModeNormal)
	mh.SetSGR(true)

	seq := mh.EncodeScroll(0, 1, Modifiers{}, 5, 10)
	want := "\x1b[<64;6;11M"
	if string(seq) != want {
		t.Errorf("scroll up = %q, want %q", string(seq), want)
	}

	seq = mh.EncodeScroll(0, -1, Modifiers{}, 5, 10)
	want = "\x1b[<65;6;11M"
	if string(seq) != want {
		t.Errorf("scroll down = %q, want %q", string(seq), want)
	}
}
