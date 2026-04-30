package main

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"

	"github.com/ghostty-go/ghostty-go/font"
	"github.com/ghostty-go/ghostty-go/renderer"
)

const (
	version = "0.1.0"

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
		Width:      windowWidth,
		Height:     windowHeight,
		CellWidth:  metrics.CellWidth,
		CellHeight: metrics.CellHeight,
		GridCols:   gridCols,
		GridRows:   gridRows,
		PaddingX:   paddingX,
		PaddingY:   paddingY,
		BGColor:    renderer.Color{R: 0.1, G: 0.1, B: 0.12, A: 1.0},
		CursorColor: renderer.Color{R: 0.9, G: 0.9, B: 0.9, A: 0.8},
		CursorStyle: renderer.CursorBlock,
	})
	defer ren.Destroy()

	ren.SetBGColor(renderer.Color{R: 0.1, G: 0.1, B: 0.12, A: 1.0})

	// Pre-populate atlas with ASCII printable characters
	for ch := rune(32); ch < 127; ch++ {
		ren.EnsureAtlasGlyph(face, ch)
	}

	// Build a demo grid with "Hello, World!" and colored text
	grid := buildDemoGrid()

	// Handle resize
	window.SetFramebufferSizeCallback(func(w *glfw.Window, width, height int) {
		ren.Resize(width, height)
	})

	// Cursor blink state
	cursorVisible := true
	lastBlink := time.Now()

	// Main loop
	for !window.ShouldClose() {
		glfw.PollEvents()

		// Cursor blink (500ms interval)
		if time.Since(lastBlink) > 500*time.Millisecond {
			cursorVisible = !cursorVisible
			lastBlink = time.Now()
		}

		// Draw frame
		ren.SetCursor(0, 13, cursorVisible, renderer.CursorBlock)
		ren.DrawFrame(grid)

		window.SwapBuffers()
	}

	return nil
}

func loadFont() (*font.Face, error) {
	// Try common monospace font paths on Linux
	candidates := []string{
		// Ubuntu/Debian
		"/usr/share/fonts/truetype/dejavu/DejaVuSansMono.ttf",
		"/usr/share/fonts/truetype/liberation/LiberationMono-Regular.ttf",
		"/usr/share/fonts/truetype/ubuntu/UbuntuMono-R.ttf",
		"/usr/share/fonts/truetype/freefont/FreeMono.ttf",
		// Arch
		"/usr/share/fonts/TTF/DejaVuSansMono.ttf",
		"/usr/share/fonts/noto-mono/NotoSansMono-Regular.ttf",
		// Fedora
		"/usr/share/fonts/dejavu-sans-mono-fonts/DejaVuSansMono.ttf",
		// macOS
		"/System/Library/Fonts/Menlo.ttc",
		"/System/Library/Fonts/Monaco.dfont",
		"/Library/Fonts/Courier New.ttf",
	}

	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return font.LoadFace(path, fontSize)
		}
	}

	// Try finding via fc-match (fontconfig)
	return nil, fmt.Errorf("no monospace font found in: %s", strings.Join(candidates, ", "))
}

func buildDemoGrid() [][]renderer.Cell {
	// Default colors
	fg := renderer.Color{R: 0.9, G: 0.9, B: 0.9, A: 1.0}
	bg := renderer.Color{R: 0.1, G: 0.1, B: 0.12, A: 1.0}

	// Special colors
	red := renderer.Color{R: 0.95, G: 0.3, B: 0.3, A: 1.0}
	green := renderer.Color{R: 0.3, G: 0.95, B: 0.4, A: 1.0}
	blue := renderer.Color{R: 0.4, G: 0.6, B: 0.95, A: 1.0}
	yellow := renderer.Color{R: 0.95, G: 0.85, B: 0.3, A: 1.0}
	cyan := renderer.Color{R: 0.3, G: 0.9, B: 0.9, A: 1.0}
	dim := renderer.Color{R: 0.5, G: 0.5, B: 0.5, A: 1.0}

	// Initialize empty grid
	grid := make([][]renderer.Cell, gridRows)
	for row := range grid {
		grid[row] = make([]renderer.Cell, gridCols)
		for col := range grid[row] {
			grid[row][col] = renderer.Cell{Char: ' ', FG: fg, BG: bg}
		}
	}

	// Title bar
	titleBar := renderer.Color{R: 0.15, G: 0.15, B: 0.2, A: 1.0}
	for col := 0; col < gridCols; col++ {
		grid[0][col].BG = titleBar
	}
	writeString(grid, 0, 2, "ghostty-go v"+version, cyan, titleBar)

	// Main content
	writeString(grid, 2, 2, "Welcome to ", fg, bg)
	writeString(grid, 2, 13, "ghostty-go", green, bg)
	writeString(grid, 2, 23, "!", fg, bg)

	writeString(grid, 4, 2, "A GPU-accelerated terminal emulator written in ", dim, bg)
	writeString(grid, 4, 50, "Go", cyan, bg)

	writeString(grid, 6, 2, "Features:", yellow, bg)
	writeString(grid, 7, 4, "- OpenGL 4.1 rendering with instanced quads", fg, bg)
	writeString(grid, 8, 4, "- Glyph atlas with shelf packing", fg, bg)
	writeString(grid, 9, 4, "- VT500-series terminal parser (coming soon)", dim, bg)
	writeString(grid, 10, 4, "- Cross-platform: Linux + macOS", fg, bg)

	writeString(grid, 12, 2, "Colors: ", fg, bg)
	writeString(grid, 12, 10, "Red ", red, bg)
	writeString(grid, 12, 14, "Green ", green, bg)
	writeString(grid, 12, 20, "Blue ", blue, bg)
	writeString(grid, 12, 25, "Yellow ", yellow, bg)
	writeString(grid, 12, 32, "Cyan", cyan, bg)

	// Prompt simulation
	writeString(grid, 14, 2, "user@ghostty-go", green, bg)
	writeString(grid, 14, 17, ":", fg, bg)
	writeString(grid, 14, 18, "~/project", blue, bg)
	writeString(grid, 14, 27, "$ ", fg, bg)
	writeString(grid, 14, 29, "echo \"Hello, World!\"", fg, bg)

	writeString(grid, 15, 2, "Hello, World!", fg, bg)

	writeString(grid, 17, 2, "user@ghostty-go", green, bg)
	writeString(grid, 17, 17, ":", fg, bg)
	writeString(grid, 17, 18, "~/project", blue, bg)
	writeString(grid, 17, 27, "$ ", fg, bg)

	// Status line at bottom
	statusBar := renderer.Color{R: 0.15, G: 0.15, B: 0.2, A: 1.0}
	for col := 0; col < gridCols; col++ {
		grid[gridRows-1][col].BG = statusBar
	}
	writeString(grid, gridRows-1, 2, "NORMAL", dim, statusBar)
	writeString(grid, gridRows-1, gridCols-20, "utf-8  LF  Ln 1, Col 1", dim, statusBar)

	return grid
}

func writeString(grid [][]renderer.Cell, row, col int, s string, fg, bg renderer.Color) {
	if row < 0 || row >= len(grid) {
		return
	}
	for _, ch := range s {
		if col >= len(grid[row]) {
			break
		}
		grid[row][col] = renderer.Cell{Char: ch, FG: fg, BG: bg}
		col++
	}
}
