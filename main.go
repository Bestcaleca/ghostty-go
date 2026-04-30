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
	"github.com/ghostty-go/ghostty-go/parser"
	"github.com/ghostty-go/ghostty-go/renderer"
	"github.com/ghostty-go/ghostty-go/terminal"
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

	// Create terminal and stream
	term := terminal.New(gridRows, gridCols)
	stream := terminal.NewStream(term)

	// Write demo content through the stream (VT sequences)
	stream.Process([]byte(
		"\x1b[2J" +     // Clear screen
			"\x1b[H" +  // Cursor home
			"\x1b[1;36m" + // Bold cyan
			"ghostty-go v" + version + "\r\n\r\n" +
			"\x1b[0m" + // Reset
			"A GPU-accelerated terminal emulator written in " +
			"\x1b[1;32m" + "Go" + "\x1b[0m\r\n\r\n" +
			"\x1b[1;33m" + "Features:" + "\x1b[0m\r\n" +
			"  - OpenGL 4.1 rendering with instanced quads\r\n" +
			"  - Glyph atlas with shelf packing\r\n" +
			"  - VT500-series terminal parser\r\n" +
			"  - Cross-platform: Linux + macOS\r\n\r\n" +
			"\x1b[1;33m" + "Colors: " + "\x1b[0m" +
			"\x1b[31m" + "Red " +
			"\x1b[32m" + "Green " +
			"\x1b[34m" + "Blue " +
			"\x1b[33m" + "Yellow " +
			"\x1b[36m" + "Cyan" +
			"\x1b[0m\r\n\r\n" +
			"\x1b[32m" + "user@ghostty-go" + "\x1b[0m" +
			":" +
			"\x1b[34m" + "~/project" + "\x1b[0m" +
			"$ echo \"Hello, World!\"\r\n" +
			"Hello, World!\r\n" +
			"\x1b[32m" + "user@ghostty-go" + "\x1b[0m" +
			":" +
			"\x1b[34m" + "~/project" + "\x1b[0m" +
			"$ ",
	))

	// Set cursor style
	term.CSIDispatch(parser.CSIDispatchAction{
		Final:      'q',
		Params:     [24]uint16{6},
		ParamCount: 1,
	})

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

		// Get terminal state
		grid := term.Grid()
		cursorRow, cursorCol, _, _ := term.Cursor()

		// Convert terminal grid to renderer cells
		renderGrid := make([][]renderer.Cell, len(grid))
		for row := range grid {
			renderGrid[row] = make([]renderer.Cell, len(grid[row]))
			for col := range grid[row] {
				c := grid[row][col]
				style := term.Active().Styles.Lookup(c.Style)

				fg := styleToColor(style.FG, renderer.Color{R: 0.9, G: 0.9, B: 0.9, A: 1.0})
				bg := styleToColor(style.BG, renderer.Color{R: 0.1, G: 0.1, B: 0.12, A: 1.0})

				renderGrid[row][col] = renderer.Cell{
					Char: c.Char,
					FG:   fg,
					BG:   bg,
					Width: int(c.Width),
				}
			}
		}

		// Draw frame
		ren.SetCursor(cursorRow, cursorCol, cursorVisible, renderer.CursorBlock)
		ren.DrawFrame(renderGrid)

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

func styleToColor(c terminal.Color, fallback renderer.Color) renderer.Color {
	if !c.Valid {
		return fallback
	}
	return renderer.Color{
		R: float32(c.R) / 255.0,
		G: float32(c.G) / 255.0,
		B: float32(c.B) / 255.0,
		A: 1.0,
	}
}
