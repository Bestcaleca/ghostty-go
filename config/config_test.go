package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.FontSize != 16.0 {
		t.Errorf("expected font-size=16.0, got %f", cfg.FontSize)
	}
	if cfg.WindowWidth != 960 {
		t.Errorf("expected window-width=960, got %d", cfg.WindowWidth)
	}
	if cfg.ScrollbackLines != 10000 {
		t.Errorf("expected scrollback=10000, got %d", cfg.ScrollbackLines)
	}
	if cfg.CursorStyle != "block" {
		t.Errorf("expected cursor-style=block, got %s", cfg.CursorStyle)
	}
}

func TestLoadFileNotExist(t *testing.T) {
	cfg, err := LoadFile("/nonexistent/config.toml")
	if err != nil {
		t.Fatalf("expected no error for missing file, got %v", err)
	}
	if cfg.FontSize != 16.0 {
		t.Errorf("expected default font-size, got %f", cfg.FontSize)
	}
}

func TestLoadAndSaveFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	// Create a custom config
	cfg := DefaultConfig()
	cfg.FontSize = 20.0
	cfg.FontFamily = "JetBrains Mono"
	cfg.Foreground = "#ffffff"
	cfg.Background = "#000000"
	cfg.CursorStyle = "beam"
	cfg.Shell = "/bin/zsh"

	// Save
	if err := SaveFile(path, cfg); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("config file not created: %v", err)
	}

	// Load
	loaded, err := LoadFile(path)
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}

	if loaded.FontSize != 20.0 {
		t.Errorf("expected font-size=20.0, got %f", loaded.FontSize)
	}
	if loaded.FontFamily != "JetBrains Mono" {
		t.Errorf("expected font-family=JetBrains Mono, got %s", loaded.FontFamily)
	}
	if loaded.Foreground != "#ffffff" {
		t.Errorf("expected foreground=#ffffff, got %s", loaded.Foreground)
	}
	if loaded.Background != "#000000" {
		t.Errorf("expected background=#000000, got %s", loaded.Background)
	}
	if loaded.CursorStyle != "beam" {
		t.Errorf("expected cursor-style=beam, got %s", loaded.CursorStyle)
	}
	if loaded.Shell != "/bin/zsh" {
		t.Errorf("expected shell=/bin/zsh, got %s", loaded.Shell)
	}
}

func TestParseColor(t *testing.T) {
	tests := []struct {
		input   string
		r, g, b, a float32
		wantErr bool
	}{
		{"#ffffff", 1.0, 1.0, 1.0, 1.0, false},
		{"#000000", 0.0, 0.0, 0.0, 1.0, false},
		{"#ff0000", 1.0, 0.0, 0.0, 1.0, false},
		{"#00ff00", 0.0, 1.0, 0.0, 1.0, false},
		{"#0000ff", 0.0, 0.0, 1.0, 1.0, false},
		{"#ff000080", 1.0, 0.0, 0.0, 0.502, false},
		{"ffffff", 1.0, 1.0, 1.0, 1.0, false},   // without #
		{"#fff", 0, 0, 0, 0, true},               // 3-digit not supported
		{"invalid", 0, 0, 0, 0, true},
	}

	for _, tt := range tests {
		r, g, b, a, err := ParseColor(tt.input)
		if tt.wantErr {
			if err == nil {
				t.Errorf("ParseColor(%q) expected error", tt.input)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseColor(%q) unexpected error: %v", tt.input, err)
			continue
		}
		if abs(r-tt.r) > 0.01 || abs(g-tt.g) > 0.01 || abs(b-tt.b) > 0.01 || abs(a-tt.a) > 0.01 {
			t.Errorf("ParseColor(%q) = (%.3f, %.3f, %.3f, %.3f), want (%.3f, %.3f, %.3f, %.3f)",
				tt.input, r, g, b, a, tt.r, tt.g, tt.b, tt.a)
		}
	}
}

func TestLoadFileWithKeybindings(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	content := `
font-size = 14.0
foreground = "#c0c0c0"

[[keybindings]]
key = "c"
action = "copy"
mods = "ctrl+shift"

[[keybindings]]
key = "v"
action = "paste"
mods = "ctrl+shift"
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	cfg, err := LoadFile(path)
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}

	if cfg.FontSize != 14.0 {
		t.Errorf("expected font-size=14.0, got %f", cfg.FontSize)
	}
	if len(cfg.Keybindings) != 2 {
		t.Fatalf("expected 2 keybindings, got %d", len(cfg.Keybindings))
	}
	if cfg.Keybindings[0].Key != "c" || cfg.Keybindings[0].Action != "copy" {
		t.Errorf("unexpected keybinding 0: %+v", cfg.Keybindings[0])
	}
	if cfg.Keybindings[1].Key != "v" || cfg.Keybindings[1].Action != "paste" {
		t.Errorf("unexpected keybinding 1: %+v", cfg.Keybindings[1])
	}
}

func abs(x float32) float32 {
	if x < 0 {
		return -x
	}
	return x
}
