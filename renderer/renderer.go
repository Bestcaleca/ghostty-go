package renderer

import (
	"fmt"
	"image"
	"image/draw"
	"unsafe"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/go-gl/mathgl/mgl32"

	"github.com/ghostty-go/ghostty-go/font"
)

// Color represents an RGBA color.
type Color struct {
	R, G, B, A float32
}

// Cell represents a single terminal cell for rendering.
type Cell struct {
	Char          rune
	FG            Color
	BG            Color
	Width         int // 1 for normal, 2 for wide
	Underline     bool
	Strikethrough bool
	Overline      bool
}

// GlyphRasterizer provides rasterized glyph bitmaps for atlas population.
type GlyphRasterizer interface {
	RasterizeGlyph(r rune) *font.GlyphBitmap
}

// CursorStyle represents cursor display mode.
type CursorStyle int

const (
	CursorBlock CursorStyle = iota
	CursorUnderline
	CursorBeam
)

// Renderer manages OpenGL rendering of a terminal grid.
type Renderer struct {
	window *glfw.Window

	// Shader programs
	bgProg         uint32
	textProg       uint32
	decorationProg uint32
	cursorProg     uint32

	// VAOs
	bgVAO         uint32
	textVAO       uint32
	decorationVAO uint32
	cursorVAO     uint32

	// VBOs
	bgVBO         uint32
	textVBO       uint32
	decorationVBO uint32
	cursorVBO     uint32

	// Atlas texture
	atlasTex        uint32
	atlas           *font.Atlas
	atlasW          int32
	atlasH          int32
	lastAtlasW      int32
	lastAtlasH      int32
	glyphRasterizer GlyphRasterizer

	// Dimensions
	cellW, cellH float32
	cellAscent   float32
	gridCols     int
	gridRows     int
	screenW      int
	screenH      int
	paddingX     float32
	paddingY     float32

	// Projection
	proj mgl32.Mat4

	// Background color
	bgColor Color

	// Instance data buffers (CPU-side)
	bgData         []float32 // per-cell: col, row, r, g, b, a
	textData       []float32 // per-cell: atlasX, atlasY, glyphW, glyphH, bearingX, bearingY, col, row, r, g, b, a
	decorationData []float32 // per-line: col, row, y offset, thickness, r, g, b, a

	// Cursor
	cursorRow int
	cursorCol int
	cursorVis bool
	cursorSty CursorStyle
	cursorClr Color
}

// Config holds renderer configuration.
type Config struct {
	Width           int
	Height          int
	CellWidth       float32
	CellHeight      float32
	CellAscent      float32
	GridCols        int
	GridRows        int
	PaddingX        float32
	PaddingY        float32
	BGColor         Color
	CursorColor     Color
	CursorStyle     CursorStyle
	GlyphRasterizer GlyphRasterizer
}

// New creates a new Renderer. Must be called from the main thread with a current GL context.
func New(cfg Config) *Renderer {
	r := &Renderer{
		cellW:           cfg.CellWidth,
		cellH:           cfg.CellHeight,
		cellAscent:      cfg.CellAscent,
		gridCols:        cfg.GridCols,
		gridRows:        cfg.GridRows,
		screenW:         cfg.Width,
		screenH:         cfg.Height,
		paddingX:        cfg.PaddingX,
		paddingY:        cfg.PaddingY,
		bgColor:         cfg.BGColor,
		cursorClr:       cfg.CursorColor,
		cursorSty:       cfg.CursorStyle,
		cursorVis:       true,
		atlas:           font.NewAtlas(font.DefaultAtlasWidth, font.DefaultAtlasHeight),
		glyphRasterizer: cfg.GlyphRasterizer,
		bgData:          make([]float32, 0, cfg.GridCols*cfg.GridRows*6),
		textData:        make([]float32, 0, cfg.GridCols*cfg.GridRows*12),
		decorationData:  make([]float32, 0, cfg.GridCols*cfg.GridRows*8),
	}

	r.initShaders()
	r.initBuffers()
	r.initAtlasTexture()
	r.updateProjection()

	return r
}

// Atlas returns the glyph atlas for populating with glyphs.
func (r *Renderer) Atlas() *font.Atlas {
	return r.atlas
}

func (r *Renderer) initShaders() {
	var err error
	r.bgProg, err = CompileProgram(bgVertSrc, bgFragSrc)
	if err != nil {
		panic(fmt.Sprintf("bg shader: %v", err))
	}
	r.textProg, err = CompileProgram(textVertSrc, textFragSrc)
	if err != nil {
		panic(fmt.Sprintf("text shader: %v", err))
	}
	r.decorationProg, err = CompileProgram(decorationVertSrc, decorationFragSrc)
	if err != nil {
		panic(fmt.Sprintf("decoration shader: %v", err))
	}
	r.cursorProg, err = CompileProgram(cursorVertSrc, cursorFragSrc)
	if err != nil {
		panic(fmt.Sprintf("cursor shader: %v", err))
	}
}

func (r *Renderer) initBuffers() {
	// Background: 1 VAO + 1 VBO, instanced
	gl.GenVertexArrays(1, &r.bgVAO)
	gl.GenBuffers(1, &r.bgVBO)
	gl.BindVertexArray(r.bgVAO)
	gl.BindBuffer(gl.ARRAY_BUFFER, r.bgVBO)
	// Per-instance: cell_pos (vec2) + cell_color (vec4) = 6 floats = 24 bytes
	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointerWithOffset(0, 2, gl.FLOAT, false, 24, 0)
	gl.VertexAttribDivisor(0, 1)
	gl.EnableVertexAttribArray(1)
	gl.VertexAttribPointerWithOffset(1, 4, gl.FLOAT, false, 24, 8)
	gl.VertexAttribDivisor(1, 1)
	gl.BindVertexArray(0)

	// Text: 1 VAO + 1 VBO, instanced
	gl.GenVertexArrays(1, &r.textVAO)
	gl.GenBuffers(1, &r.textVBO)
	gl.BindVertexArray(r.textVAO)
	gl.BindBuffer(gl.ARRAY_BUFFER, r.textVBO)
	// Per-instance: atlas_pos + glyph_size + bearings + grid_pos + color
	// = 12 float values, 12 * 4 = 48 bytes
	var stride int32 = 48
	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointerWithOffset(0, 2, gl.FLOAT, false, stride, 0) // atlas_pos
	gl.VertexAttribDivisor(0, 1)
	gl.EnableVertexAttribArray(1)
	gl.VertexAttribPointerWithOffset(1, 2, gl.FLOAT, false, stride, 8) // glyph_size
	gl.VertexAttribDivisor(1, 1)
	gl.EnableVertexAttribArray(2)
	gl.VertexAttribPointerWithOffset(2, 2, gl.FLOAT, false, stride, 16) // bearings
	gl.VertexAttribDivisor(2, 1)
	gl.EnableVertexAttribArray(3)
	gl.VertexAttribPointerWithOffset(3, 2, gl.FLOAT, false, stride, 24) // grid_pos
	gl.VertexAttribDivisor(3, 1)
	gl.EnableVertexAttribArray(4)
	gl.VertexAttribPointerWithOffset(4, 4, gl.FLOAT, false, stride, 32) // color
	gl.VertexAttribDivisor(4, 1)
	gl.BindVertexArray(0)

	// Decorations: underline/strike/overline rectangles, instanced
	gl.GenVertexArrays(1, &r.decorationVAO)
	gl.GenBuffers(1, &r.decorationVBO)
	gl.BindVertexArray(r.decorationVAO)
	gl.BindBuffer(gl.ARRAY_BUFFER, r.decorationVBO)
	// Per-instance: line_rect (vec4) + color (vec4) = 8 floats = 32 bytes
	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointerWithOffset(0, 4, gl.FLOAT, false, 32, 0)
	gl.VertexAttribDivisor(0, 1)
	gl.EnableVertexAttribArray(1)
	gl.VertexAttribPointerWithOffset(1, 4, gl.FLOAT, false, 32, 16)
	gl.VertexAttribDivisor(1, 1)
	gl.BindVertexArray(0)

	// Cursor: 1 VAO + 1 VBO, instanced
	gl.GenVertexArrays(1, &r.cursorVAO)
	gl.GenBuffers(1, &r.cursorVBO)
	gl.BindVertexArray(r.cursorVAO)
	gl.BindBuffer(gl.ARRAY_BUFFER, r.cursorVBO)
	// Per-instance: cursor_pos (vec2) + cursor_col (vec4) = 6 floats
	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointerWithOffset(0, 2, gl.FLOAT, false, 24, 0)
	gl.VertexAttribDivisor(0, 1)
	gl.EnableVertexAttribArray(1)
	gl.VertexAttribPointerWithOffset(1, 4, gl.FLOAT, false, 24, 8)
	gl.VertexAttribDivisor(1, 1)
	gl.BindVertexArray(0)
}

func (r *Renderer) initAtlasTexture() {
	gl.GenTextures(1, &r.atlasTex)
	gl.BindTexture(gl.TEXTURE_2D, r.atlasTex)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
	r.uploadAtlas()
}

func (r *Renderer) uploadAtlas() {
	a := r.atlas
	w := int32(a.Width())
	h := int32(a.Height())

	// Only reallocate texture if size changed
	if w != r.lastAtlasW || h != r.lastAtlasH {
		gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA, w, h, 0, gl.RGBA, gl.UNSIGNED_BYTE, unsafe.Pointer(&a.Pixels()[0]))
		r.lastAtlasW = w
		r.lastAtlasH = h
	} else {
		gl.TexSubImage2D(gl.TEXTURE_2D, 0, 0, 0, w, h, gl.RGBA, gl.UNSIGNED_BYTE, unsafe.Pointer(&a.Pixels()[0]))
	}
	a.ClearDirty()
	r.atlasW = w
	r.atlasH = h
}

func (r *Renderer) updateProjection() {
	// Orthographic projection: (0,0) top-left, (screenW, screenH) bottom-right
	r.proj = mgl32.Ortho(0, float32(r.screenW), float32(r.screenH), 0, -1, 1)
}

// SetCursor sets the cursor position and visibility.
func (r *Renderer) SetCursor(row, col int, visible bool, style CursorStyle) {
	r.cursorRow = row
	r.cursorCol = col
	r.cursorVis = visible
	r.cursorSty = style
}

// Resize handles window resize events.
func (r *Renderer) Resize(width, height int) {
	r.screenW = width
	r.screenH = height
	gl.Viewport(0, 0, int32(width), int32(height))
	r.updateProjection()
}

// SetGridSize updates the terminal grid bounds used for instanced rendering.
func (r *Renderer) SetGridSize(rows, cols int) {
	r.gridRows = rows
	r.gridCols = cols
}

// SetGlyphRasterizer sets the source used to populate missing atlas glyphs.
func (r *Renderer) SetGlyphRasterizer(glyphs GlyphRasterizer) {
	r.glyphRasterizer = glyphs
}

// DrawFrame renders a complete terminal frame.
func (r *Renderer) DrawFrame(grid [][]Cell) {
	gl.Clear(gl.COLOR_BUFFER_BIT)

	// Upload atlas if dirty
	if r.atlas.IsDirty() {
		r.uploadAtlas()
	}

	// Build and draw background instances
	bgCount := r.buildBackgroundData(grid)
	if bgCount > 0 {
		gl.UseProgram(r.bgProg)
		projLoc := gl.GetUniformLocation(r.bgProg, gl.Str("projection\x00"))
		gl.UniformMatrix4fv(projLoc, 1, false, &r.proj[0])
		cellSizeLoc := gl.GetUniformLocation(r.bgProg, gl.Str("cell_size\x00"))
		gl.Uniform2f(cellSizeLoc, r.cellW, r.cellH)
		paddingLoc := gl.GetUniformLocation(r.bgProg, gl.Str("padding\x00"))
		gl.Uniform2f(paddingLoc, r.paddingX, r.paddingY)

		gl.BindVertexArray(r.bgVAO)
		gl.BindBuffer(gl.ARRAY_BUFFER, r.bgVBO)
		gl.BufferData(gl.ARRAY_BUFFER, len(r.bgData)*4, unsafe.Pointer(&r.bgData[0]), gl.DYNAMIC_DRAW)
		gl.DrawArraysInstanced(gl.TRIANGLE_STRIP, 0, 4, int32(bgCount))
	}

	// Build and draw text instances
	textCount := r.buildTextData(grid)
	if textCount > 0 {
		gl.UseProgram(r.textProg)
		projLoc := gl.GetUniformLocation(r.textProg, gl.Str("projection\x00"))
		gl.UniformMatrix4fv(projLoc, 1, false, &r.proj[0])
		cellSizeLoc := gl.GetUniformLocation(r.textProg, gl.Str("cell_size\x00"))
		gl.Uniform2f(cellSizeLoc, r.cellW, r.cellH)
		paddingLoc := gl.GetUniformLocation(r.textProg, gl.Str("padding\x00"))
		gl.Uniform2f(paddingLoc, r.paddingX, r.paddingY)
		atlasSizeLoc := gl.GetUniformLocation(r.textProg, gl.Str("atlas_size\x00"))
		gl.Uniform2f(atlasSizeLoc, float32(r.atlasW), float32(r.atlasH))

		gl.ActiveTexture(gl.TEXTURE0)
		gl.BindTexture(gl.TEXTURE_2D, r.atlasTex)
		atlasLoc := gl.GetUniformLocation(r.textProg, gl.Str("atlas\x00"))
		gl.Uniform1i(atlasLoc, 0)

		gl.Enable(gl.BLEND)
		gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)

		gl.BindVertexArray(r.textVAO)
		gl.BindBuffer(gl.ARRAY_BUFFER, r.textVBO)
		gl.BufferData(gl.ARRAY_BUFFER, len(r.textData)*4, unsafe.Pointer(&r.textData[0]), gl.DYNAMIC_DRAW)
		gl.DrawArraysInstanced(gl.TRIANGLE_STRIP, 0, 4, int32(textCount))

		gl.Disable(gl.BLEND)
	}

	// Draw text decorations after glyphs so strikethrough stays visible.
	decorationCount := r.buildDecorationData(grid)
	if decorationCount > 0 {
		gl.UseProgram(r.decorationProg)
		projLoc := gl.GetUniformLocation(r.decorationProg, gl.Str("projection\x00"))
		gl.UniformMatrix4fv(projLoc, 1, false, &r.proj[0])
		cellSizeLoc := gl.GetUniformLocation(r.decorationProg, gl.Str("cell_size\x00"))
		gl.Uniform2f(cellSizeLoc, r.cellW, r.cellH)
		paddingLoc := gl.GetUniformLocation(r.decorationProg, gl.Str("padding\x00"))
		gl.Uniform2f(paddingLoc, r.paddingX, r.paddingY)

		gl.BindVertexArray(r.decorationVAO)
		gl.BindBuffer(gl.ARRAY_BUFFER, r.decorationVBO)
		gl.BufferData(gl.ARRAY_BUFFER, len(r.decorationData)*4, unsafe.Pointer(&r.decorationData[0]), gl.DYNAMIC_DRAW)
		gl.DrawArraysInstanced(gl.TRIANGLE_STRIP, 0, 4, int32(decorationCount))
	}

	// Draw cursor
	if r.cursorVis {
		r.drawCursor()
	}

	gl.BindVertexArray(0)
}

func (r *Renderer) buildBackgroundData(grid [][]Cell) int {
	r.bgData = r.bgData[:0]
	cols := r.gridCols
	rows := r.gridRows

	for row := 0; row < rows && row < len(grid); row++ {
		for col := 0; col < cols && col < len(grid[row]); col++ {
			c := grid[row][col]
			// Skip cells with default background (optimization: could batch)
			r.bgData = append(r.bgData,
				float32(col), float32(row),
				c.BG.R, c.BG.G, c.BG.B, c.BG.A,
			)
		}
	}
	return len(r.bgData) / 6
}

func (r *Renderer) buildTextData(grid [][]Cell) int {
	r.textData = r.textData[:0]
	a := r.atlas
	cols := r.gridCols
	rows := r.gridRows

	for row := 0; row < rows && row < len(grid); row++ {
		for col := 0; col < cols && col < len(grid[row]); col++ {
			c := grid[row][col]
			if c.Char == 0 || c.Char == ' ' {
				continue
			}

			entry, ok := a.Get(c.Char)
			if !ok {
				entry, ok = r.ensureAtlasGlyph(c.Char)
			}
			if !ok || !entry.Valid {
				continue
			}

			bearingY := glyphTopOffset(r.cellAscent, entry.BearingY)

			r.textData = append(r.textData,
				float32(entry.X), float32(entry.Y), // atlas_pos
				float32(entry.Width), float32(entry.Height), // glyph_size
				entry.BearingX, bearingY, // bearings
				float32(col), float32(row), // grid_pos
				c.FG.R, c.FG.G, c.FG.B, c.FG.A, // color
			)
		}
	}
	return len(r.textData) / 12
}

func (r *Renderer) ensureAtlasGlyph(ch rune) (font.GlyphEntry, bool) {
	if r.glyphRasterizer == nil {
		return font.GlyphEntry{}, false
	}
	bmp := r.glyphRasterizer.RasterizeGlyph(ch)
	if bmp == nil {
		return font.GlyphEntry{}, false
	}
	entry := r.atlas.Add(ch, bmp)
	if !entry.Valid {
		return font.GlyphEntry{}, false
	}
	return entry, true
}

func (r *Renderer) buildDecorationData(grid [][]Cell) int {
	r.decorationData = r.decorationData[:0]
	cols := r.gridCols
	rows := r.gridRows
	thickness := decorationThickness(r.cellH)

	for row := 0; row < rows && row < len(grid); row++ {
		for col := 0; col < cols && col < len(grid[row]); col++ {
			c := grid[row][col]
			if c.Overline {
				r.appendDecoration(col, row, 0, thickness, c.FG)
			}
			if c.Strikethrough {
				r.appendDecoration(col, row, r.cellH*0.55, thickness, c.FG)
			}
			if c.Underline {
				r.appendDecoration(col, row, r.cellH-thickness-1, thickness, c.FG)
			}
		}
	}
	return len(r.decorationData) / 8
}

func (r *Renderer) appendDecoration(col, row int, yOffset, thickness float32, color Color) {
	r.decorationData = append(r.decorationData,
		float32(col), float32(row), yOffset, thickness,
		color.R, color.G, color.B, color.A,
	)
}

func decorationThickness(cellHeight float32) float32 {
	thickness := cellHeight / 12
	if thickness < 1 {
		return 1
	}
	return thickness
}

func (r *Renderer) drawCursor() {
	gl.UseProgram(r.cursorProg)
	projLoc := gl.GetUniformLocation(r.cursorProg, gl.Str("projection\x00"))
	gl.UniformMatrix4fv(projLoc, 1, false, &r.proj[0])
	cellSizeLoc := gl.GetUniformLocation(r.cursorProg, gl.Str("cell_size\x00"))
	gl.Uniform2f(cellSizeLoc, r.cellW, r.cellH)
	paddingLoc := gl.GetUniformLocation(r.cursorProg, gl.Str("padding\x00"))
	gl.Uniform2f(paddingLoc, r.paddingX, r.paddingY)

	cursorData := []float32{
		float32(r.cursorCol), float32(r.cursorRow),
		r.cursorClr.R, r.cursorClr.G, r.cursorClr.B, r.cursorClr.A,
	}

	gl.BindVertexArray(r.cursorVAO)
	gl.BindBuffer(gl.ARRAY_BUFFER, r.cursorVBO)
	gl.BufferData(gl.ARRAY_BUFFER, len(cursorData)*4, unsafe.Pointer(&cursorData[0]), gl.DYNAMIC_DRAW)
	gl.DrawArraysInstanced(gl.TRIANGLE_STRIP, 0, 4, 1)
	gl.BindVertexArray(0)
}

func glyphTopOffset(cellAscent, glyphMinY float32) float32 {
	return cellAscent + glyphMinY
}

// EnsureAtlasGlyph ensures a rune is in the atlas, rasterizing it if needed.
func (r *Renderer) EnsureAtlasGlyph(ch rune) {
	if _, ok := r.atlas.Get(ch); ok {
		return
	}
	r.ensureAtlasGlyph(ch)
}

// SetBGColor sets the background clear color.
func (r *Renderer) SetBGColor(c Color) {
	r.bgColor = c
	gl.ClearColor(c.R, c.G, c.B, c.A)
}

// BGColor returns the current background color.
func (r *Renderer) BGColor() Color {
	return r.bgColor
}

// CellMetrics holds the dimensions of a single cell.
type CellMetrics struct {
	CellWidth  float32
	CellHeight float32
	PaddingX   float32
	PaddingY   float32
}

// Metrics returns the cell dimensions.
func (r *Renderer) Metrics() CellMetrics {
	return CellMetrics{
		CellWidth:  r.cellW,
		CellHeight: r.cellH,
		PaddingX:   r.paddingX,
		PaddingY:   r.paddingY,
	}
}

// Destroy cleans up GL resources.
func (r *Renderer) Destroy() {
	gl.DeleteBuffers(1, &r.bgVBO)
	gl.DeleteBuffers(1, &r.textVBO)
	gl.DeleteBuffers(1, &r.decorationVBO)
	gl.DeleteBuffers(1, &r.cursorVBO)
	gl.DeleteVertexArrays(1, &r.bgVAO)
	gl.DeleteVertexArrays(1, &r.textVAO)
	gl.DeleteVertexArrays(1, &r.decorationVAO)
	gl.DeleteVertexArrays(1, &r.cursorVAO)
	gl.DeleteProgram(r.bgProg)
	gl.DeleteProgram(r.textProg)
	gl.DeleteProgram(r.decorationProg)
	gl.DeleteProgram(r.cursorProg)
	gl.DeleteTextures(1, &r.atlasTex)
}

// ImageToRGBA converts an image.Image to an NRGBA-compat byte slice.
func ImageToRGBA(img image.Image) []byte {
	bounds := img.Bounds()
	rgba := image.NewRGBA(bounds)
	draw.Draw(rgba, bounds, img, bounds.Min, draw.Src)
	return rgba.Pix
}
