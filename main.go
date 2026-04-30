package main

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"

	"github.com/ghostty-go/ghostty-go/font"
	"github.com/ghostty-go/ghostty-go/renderer"
	"github.com/ghostty-go/ghostty-go/surface"
)

const (
	version = "0.2.0"

	windowWidth  = 960
	windowHeight = 640
	gridCols     = 80
	gridRows     = 24
	fontSize     = 16.0
	paddingX     = 2.0
	paddingY     = 1.0
)

func main() {
	fmt.Printf("ghostty-go v%s\n", version)

	// GLFW must be called from the main thread
	runtime.LockOSThread()

	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
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

	window, err := glfw.CreateWindow(windowWidth, windowHeight, "ghostty-go", nil, nil)
	if err != nil {
		return fmt.Errorf("create window: %w", err)
	}
	window.MakeContextCurrent()
	glfw.SwapInterval(1) // vsync

	// Initialize OpenGL
	if err := gl.Init(); err != nil {
		return fmt.Errorf("gl init: %w", err)
	}

	// Load font
	face, err := loadFont()
	if err != nil {
		log.Printf("font load failed, using default: %v", err)
		face = font.DefaultFace()
	}

	metrics := face.Metrics()
	log.Printf("Font metrics: cell=%.1fx%.1f ascent=%.1f descent=%.1f",
		metrics.CellWidth, metrics.CellHeight, metrics.Ascent, metrics.Descent)

	// Create renderer
	ren := renderer.New(renderer.Config{
		Width:       windowWidth,
		Height:      windowHeight,
		CellWidth:   metrics.CellWidth,
		CellHeight:  metrics.CellHeight,
		GridCols:    gridCols,
		GridRows:    gridRows,
		PaddingX:    paddingX,
		PaddingY:    paddingY,
		BGColor:     renderer.Color{R: 0.1, G: 0.1, B: 0.12, A: 1.0},
		CursorColor: renderer.Color{R: 0.9, G: 0.9, B: 0.9, A: 0.8},
		CursorStyle: renderer.CursorBlock,
	})
	defer ren.Destroy()

	ren.SetBGColor(renderer.Color{R: 0.1, G: 0.1, B: 0.12, A: 1.0})

	// Pre-populate atlas with ASCII printable characters
	for ch := rune(32); ch < 127; ch++ {
		ren.EnsureAtlasGlyph(face, ch)
	}

	// Create surface (connects terminal + renderer + input + termio)
	s, err := surface.New(surface.Config{
		Rows:     gridRows,
		Cols:     gridCols,
		Shell:    "", // auto-detect
		Renderer: ren,
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
		s.HandleKey(key, action, mods)
	})

	window.SetCharModsCallback(func(w *glfw.Window, char rune, mods glfw.ModifierKey) {
		s.HandleChar(char)
	})

	window.SetMouseButtonCallback(func(w *glfw.Window, button glfw.MouseButton, action glfw.Action, mods glfw.ModifierKey) {
		x, y := w.GetCursorPos()
		s.HandleMouseButton(button, action, mods, x, y)
	})

	window.SetScrollCallback(func(w *glfw.Window, xoff, yoff float64) {
		x, y := w.GetCursorPos()
		s.HandleScroll(xoff, yoff, x, y)
	})

	window.SetFramebufferSizeCallback(func(w *glfw.Window, width, height int) {
		s.HandleResize(width, height)
	})

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

func loadFont() (*font.Face, error) {
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
			return font.LoadFace(path, fontSize)
		}
	}

	return nil, fmt.Errorf("no monospace font found in: %s", strings.Join(candidates, ", "))
}
