package main

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"

	"github.com/ghostty-go/ghostty-go/config"
	"github.com/ghostty-go/ghostty-go/font"
	"github.com/ghostty-go/ghostty-go/renderer"
	"github.com/ghostty-go/ghostty-go/surface"
	"github.com/ghostty-go/ghostty-go/terminal"
)

const version = "0.3.0"

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "--help" || os.Args[1] == "-h") {
		fmt.Print(helpText())
		return
	}

	fmt.Printf("ghostty-go v%s\n", version)

	// GLFW must be called from the main thread
	runtime.LockOSThread()

	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func helpText() string {
	return fmt.Sprintf(`ghostty-go v%s

Usage:
  ghostty-go [--help]

Shortcuts:
  Ctrl+Shift+V      Paste clipboard text
  Shift+PageUp      Scroll up through scrollback
  Shift+PageDown    Scroll down through scrollback
  Shift+Home        Jump to the bottom
  Right click       Open context menu

Mouse:
  Left drag         Select text and copy on release
  Double click      Select word
  Triple click      Select line
`, version)
}

func run() error {
	// Load configuration
	cfgPath := config.DefaultConfigPath()
	cfg, err := config.LoadFile(cfgPath)
	if err != nil {
		log.Printf("config load failed, using defaults: %v", err)
		cfg = config.DefaultConfig()
	} else if _, statErr := os.Stat(cfgPath); statErr == nil {
		log.Printf("loaded config from %s", cfgPath)
	}

	// Parse colors
	bgR, bgG, bgB, bgA, _ := config.ParseColor(cfg.Background)
	_, _, _, _, _ = config.ParseColor(cfg.Foreground) // foreground used by terminal internally
	curR, curG, curB, curA, _ := config.ParseColor(cfg.CursorColor)

	bgColor := renderer.Color{R: bgR, G: bgG, B: bgB, A: bgA}
	cursorColor := renderer.Color{R: curR, G: curG, B: curB, A: curA}

	// Determine grid size from window dimensions
	fontSize := cfg.FontSize
	if fontSize <= 0 {
		fontSize = 16.0
	}

	// Initialize GLFW
	if err := glfw.Init(); err != nil {
		return fmt.Errorf("glfw init: %w", err)
	}
	defer glfw.Terminate()

	glfw.WindowHint(glfw.ContextVersionMajor, 4)
	glfw.WindowHint(glfw.ContextVersionMinor, 1)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)
	glfw.WindowHint(glfw.Resizable, glfw.True)

	window, err := glfw.CreateWindow(cfg.WindowWidth, cfg.WindowHeight, "ghostty-go", nil, nil)
	if err != nil {
		return fmt.Errorf("create window: %w", err)
	}
	window.MakeContextCurrent()
	glfw.SwapInterval(1) // vsync

	// Initialize OpenGL
	if err := gl.Init(); err != nil {
		return fmt.Errorf("gl init: %w", err)
	}

	// Load font stack
	fontSet, err := loadFontSet(cfg.FontFamily, fontSize)
	if err != nil {
		log.Printf("font load failed, using default: %v", err)
		fontSet = font.NewFaceSet(font.DefaultFace())
	}

	metrics := fontSet.Metrics()
	log.Printf("Font metrics: cell=%.1fx%.1f ascent=%.1f descent=%.1f faces=%d",
		metrics.CellWidth, metrics.CellHeight, metrics.Ascent, metrics.Descent, fontSet.Len())

	// Calculate grid dimensions
	paddingX := float32(cfg.PaddingX)
	paddingY := float32(cfg.PaddingY)
	cols := int((float64(cfg.WindowWidth) - 2*cfg.PaddingX) / float64(metrics.CellWidth))
	rows := int((float64(cfg.WindowHeight) - 2*cfg.PaddingY) / float64(metrics.CellHeight))
	if cols < 1 {
		cols = 80
	}
	if rows < 1 {
		rows = 24
	}

	// Map cursor style
	cursorStyle := renderer.CursorBlock
	switch cfg.CursorStyle {
	case "beam":
		cursorStyle = renderer.CursorBeam
	case "underline":
		cursorStyle = renderer.CursorUnderline
	}

	// Create renderer
	ren := renderer.New(renderer.Config{
		Width:           cfg.WindowWidth,
		Height:          cfg.WindowHeight,
		CellWidth:       metrics.CellWidth,
		CellHeight:      metrics.CellHeight,
		CellAscent:      metrics.Ascent,
		GridCols:        cols,
		GridRows:        rows,
		PaddingX:        paddingX,
		PaddingY:        paddingY,
		BGColor:         bgColor,
		CursorColor:     cursorColor,
		CursorStyle:     cursorStyle,
		GlyphRasterizer: fontSet,
	})
	defer ren.Destroy()

	ren.SetBGColor(bgColor)

	// Pre-populate atlas with ASCII printable characters
	for ch := rune(32); ch < 127; ch++ {
		ren.EnsureAtlasGlyph(ch)
	}

	// Create surface (connects terminal + renderer + input + termio)
	s, err := surface.New(surface.Config{
		Rows:            rows,
		Cols:            cols,
		Shell:           cfg.Shell,
		Renderer:        ren,
		Window:          window,
		ScrollbackLines: cfg.ScrollbackLines,
		CursorBlink:     cfg.CursorBlink,
		CursorStyle:     terminalCursorStyleFromConfig(cfg.CursorStyle, cfg.CursorBlink),
	})
	if err != nil {
		return fmt.Errorf("create surface: %w", err)
	}

	s.SetOnTitleChange(func(title string) {
		window.SetTitle(title + " - ghostty-go")
	})

	s.SetOnChildExit(func(code int) {
		log.Printf("shell exited with code %d", code)
		window.SetShouldClose(true)
	})

	s.Start()
	defer s.Stop()

	// Wire GLFW callbacks
	window.SetKeyCallback(func(w *glfw.Window, key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {
		if action == glfw.Press {
			// Ctrl+Shift+V = paste
			if key == glfw.KeyV &&
				mods&glfw.ModControl != 0 && mods&glfw.ModShift != 0 {
				text := w.GetClipboardString()
				if text != "" {
					s.HandlePaste(text)
				}
				return
			}

			// Shift+PageUp = scroll up
			if key == glfw.KeyPageUp && mods&glfw.ModShift != 0 {
				s.ScrollUp(s.Rows() / 2)
				return
			}

			// Shift+PageDown = scroll down
			if key == glfw.KeyPageDown && mods&glfw.ModShift != 0 {
				s.ScrollDown(s.Rows() / 2)
				return
			}

			// Home = scroll to bottom
			if key == glfw.KeyHome && mods&glfw.ModShift != 0 {
				s.ScrollToBottom()
				return
			}
		}

		s.HandleKey(key, action, mods)
	})

	window.SetCharCallback(func(w *glfw.Window, char rune) {
		s.HandleChar(char)
	})

	window.SetMouseButtonCallback(func(w *glfw.Window, button glfw.MouseButton, action glfw.Action, mods glfw.ModifierKey) {
		x, y := w.GetCursorPos()
		s.HandleMouseButton(button, action, mods, x, y)
	})

	window.SetCursorPosCallback(func(w *glfw.Window, x, y float64) {
		var mods glfw.ModifierKey
		if w.GetKey(glfw.KeyLeftShift) == glfw.Press || w.GetKey(glfw.KeyRightShift) == glfw.Press {
			mods |= glfw.ModShift
		}
		if w.GetKey(glfw.KeyLeftControl) == glfw.Press || w.GetKey(glfw.KeyRightControl) == glfw.Press {
			mods |= glfw.ModControl
		}
		if w.GetKey(glfw.KeyLeftAlt) == glfw.Press || w.GetKey(glfw.KeyRightAlt) == glfw.Press {
			mods |= glfw.ModAlt
		}
		if w.GetKey(glfw.KeyLeftSuper) == glfw.Press || w.GetKey(glfw.KeyRightSuper) == glfw.Press {
			mods |= glfw.ModSuper
		}
		s.HandleMouseMotion(x, y, mods)
	})

	window.SetFocusCallback(func(w *glfw.Window, focused bool) {
		s.HandleFocus(focused)
	})

	window.SetScrollCallback(func(w *glfw.Window, xoff, yoff float64) {
		x, y := w.GetCursorPos()
		// Get current modifier state
		mods := w.GetKey(glfw.KeyLeftShift) == glfw.Press || w.GetKey(glfw.KeyRightShift) == glfw.Press
		var mod glfw.ModifierKey
		if mods {
			mod = glfw.ModShift
		}
		s.HandleScroll(xoff, yoff, x, y, mod)
	})

	window.SetFramebufferSizeCallback(func(w *glfw.Window, width, height int) {
		s.HandleResize(width, height)
	})

	// Generate default config if none exists
	if _, statErr := os.Stat(cfgPath); os.IsNotExist(statErr) {
		if saveErr := config.SaveFile(cfgPath, cfg); saveErr != nil {
			log.Printf("failed to save default config: %v", saveErr)
		} else {
			log.Printf("created default config at %s", cfgPath)
		}
	}

	// Main loop
	for !window.ShouldClose() {
		glfw.PollEvents()

		// Process messages from termio (title changes, bell, etc.)
		s.ProcessMessages()

		// Render the terminal grid
		s.RenderGrid()

		window.SwapBuffers()
	}

	return nil
}

func loadFont(family string, size float64) (*font.Face, error) {
	// If a specific font family is requested, try to find it
	if family != "" {
		paths := findFontByName(family)
		for _, path := range paths {
			if _, err := os.Stat(path); err == nil {
				return loadFontFile(path, size)
			}
		}
	}

	// Fallback to common monospace fonts
	candidates := []string{
		"/usr/share/fonts/truetype/dejavu/DejaVuSansMono.ttf",
		"/usr/share/fonts/truetype/liberation/LiberationMono-Regular.ttf",
		"/usr/share/fonts/truetype/ubuntu/UbuntuMono-R.ttf",
		"/usr/share/fonts/truetype/freefont/FreeMono.ttf",
		"/usr/share/fonts/TTF/DejaVuSansMono.ttf",
		"/usr/share/fonts/noto-mono/NotoSansMono-Regular.ttf",
		"/usr/share/fonts/dejavu-sans-mono-fonts/DejaVuSansMono.ttf",
		"/System/Library/Fonts/Menlo.ttc",
		"/System/Library/Fonts/Monaco.dfont",
		"/Library/Fonts/Courier New.ttf",
	}

	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			face, err := loadFontFile(path, size)
			if err == nil {
				return face, nil
			}
		}
	}

	return nil, fmt.Errorf("no monospace font found in: %s", strings.Join(candidates, ", "))
}

func terminalCursorStyleFromConfig(style string, blink bool) terminal.CursorStyle {
	switch style {
	case "beam":
		if blink {
			return terminal.CursorBlinkingBar
		}
		return terminal.CursorSteadyBar
	case "underline":
		if blink {
			return terminal.CursorBlinkingUnderline
		}
		return terminal.CursorSteadyUnderline
	case "block":
		if !blink {
			return terminal.CursorSteadyBlock
		}
	}
	return terminal.CursorDefault
}

func loadFontSet(family string, size float64) (*font.FaceSet, error) {
	primary, err := loadFont(family, size)
	if err != nil {
		return nil, err
	}

	fallbacks := make([]*font.Face, 0)
	for _, path := range fallbackFontCandidates() {
		if _, err := os.Stat(path); err != nil {
			continue
		}
		face, err := loadFontFile(path, size)
		if err != nil {
			continue
		}
		fallbacks = append(fallbacks, face)
	}

	return font.NewFaceSet(primary, fallbacks...), nil
}

func fallbackFontCandidates() []string {
	return []string{
		"/usr/share/fonts/truetype/droid/DroidSansFallbackFull.ttf",
		"/usr/share/fonts/opentype/noto/NotoSansCJK-Regular.ttc",
		"/usr/share/fonts/truetype/noto/NotoSansSymbols2-Regular.ttf",
		"/usr/share/fonts/truetype/noto/NotoColorEmoji.ttf",
		"/System/Library/Fonts/Apple Symbols.ttf",
		"/System/Library/Fonts/Apple Color Emoji.ttc",
	}
}

func loadFontFile(path string, size float64) (face *font.Face, err error) {
	defer func() {
		if r := recover(); r != nil {
			face = nil
			err = fmt.Errorf("load font %q panicked: %v", path, r)
		}
	}()
	return font.LoadFace(path, size)
}

// findFontByName returns possible paths for a font by name.
func findFontByName(name string) []string {
	lower := strings.ToLower(name)
	base := "/usr/share/fonts"

	var paths []string
	// Try common locations
	dirs := []string{
		base + "/truetype",
		base + "/TTF",
		base + "/opentype",
		filepathJoin(os.Getenv("HOME"), ".local/share/fonts"),
	}

	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			entryLower := strings.ToLower(entry.Name())
			if strings.Contains(entryLower, lower) {
				paths = append(paths, dir+"/"+entry.Name())
			}
		}
	}

	return paths
}

func filepathJoin(a, b string) string {
	return a + "/" + b
}
