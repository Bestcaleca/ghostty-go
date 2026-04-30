package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

// Config holds all terminal configuration.
type Config struct {
	// Font
	FontFamily string  `toml:"font-family"`
	FontSize   float64 `toml:"font-size"`

	// Colors
	Foreground string `toml:"foreground"`
	Background string `toml:"background"`
	CursorColor string `toml:"cursor-color"`

	// Cursor
	CursorStyle string `toml:"cursor-style"` // block, beam, underline
	CursorBlink bool   `toml:"cursor-blink"`

	// Shell
	Shell string `toml:"shell"`

	// Window
	WindowWidth  int `toml:"window-width"`
	WindowHeight int `toml:"window-height"`

	// Terminal
	ScrollbackLines int `toml:"scrollback-lines"`

	// Padding
	PaddingX float64 `toml:"padding-x"`
	PaddingY float64 `toml:"padding-y"`

	// Keybindings
	Keybindings []Keybinding `toml:"keybindings"`
}

// Keybinding represents a key binding.
type Keybinding struct {
	Key      string `toml:"key"`
	Action   string `toml:"action"`
	Mods     string `toml:"mods"` // ctrl, alt, shift, super
}

// DefaultConfig returns the default configuration.
func DefaultConfig() *Config {
	return &Config{
		FontFamily:      "",
		FontSize:        16.0,
		Foreground:      "#e6e6e6",
		Background:      "#1a1a1f",
		CursorColor:     "#e6e6e6cc",
		CursorStyle:     "block",
		CursorBlink:     true,
		Shell:           "",
		WindowWidth:     960,
		WindowHeight:    640,
		ScrollbackLines: 10000,
		PaddingX:        2.0,
		PaddingY:        1.0,
		Keybindings:     []Keybinding{},
	}
}

// LoadFile loads configuration from a TOML file.
// If the file doesn't exist, returns default config.
func LoadFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultConfig(), nil
		}
		return nil, fmt.Errorf("read config: %w", err)
	}

	cfg := DefaultConfig()
	if err := toml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	return cfg, nil
}

// SaveFile saves configuration to a TOML file.
func SaveFile(path string, cfg *Config) error {
	data, err := toml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	return nil
}

// DefaultConfigPath returns the default config file path.
// Follows XDG: ~/.config/ghostty-go/config.toml
func DefaultConfigPath() string {
	configDir := os.Getenv("XDG_CONFIG_HOME")
	if configDir == "" {
		home, _ := os.UserHomeDir()
		configDir = filepath.Join(home, ".config")
	}
	return filepath.Join(configDir, "ghostty-go", "config.toml")
}

// ParseColor parses a hex color string (#RRGGBB or #RRGGBBAA) into RGBA components (0-1).
func ParseColor(s string) (r, g, b, a float32, err error) {
	s = strings.TrimPrefix(s, "#")

	var rgba uint32
	switch len(s) {
	case 6:
		_, err = fmt.Sscanf(s, "%06x", &rgba)
		rgba = (rgba << 8) | 0xFF
	case 8:
		_, err = fmt.Sscanf(s, "%08x", &rgba)
	default:
		return 0, 0, 0, 0, fmt.Errorf("invalid color: %s", s)
	}

	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("parse color %q: %w", s, err)
	}

	r = float32((rgba>>24)&0xFF) / 255.0
	g = float32((rgba>>16)&0xFF) / 255.0
	b = float32((rgba>>8)&0xFF) / 255.0
	a = float32(rgba&0xFF) / 255.0

	return r, g, b, a, nil
}
