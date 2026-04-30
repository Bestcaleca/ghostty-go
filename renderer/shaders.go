package renderer

import (
	"fmt"
	"strings"

	"github.com/go-gl/gl/v4.1-core/gl"
)

// Cell background vertex shader — draws a colored quad per cell.
const bgVertSrc = `
#version 410 core

layout(location = 0) in vec2 cell_pos;   // col, row (instance)
layout(location = 1) in vec4 cell_color; // RGBA background (instance)

uniform mat4 projection;
uniform vec2 cell_size;

out vec4 bg_color;

void main() {
    vec2 corner = vec2(
        float(gl_VertexID == 1 || gl_VertexID == 3),
        float(gl_VertexID == 2 || gl_VertexID == 3)
    );

    vec2 pos = cell_pos * cell_size + cell_size * corner;
    gl_Position = projection * vec4(pos, 0.0, 1.0);
    bg_color = cell_color;
}
` + "\x00"

const bgFragSrc = `
#version 410 core

in vec4 bg_color;
out vec4 out_color;

void main() {
    out_color = bg_color;
}
` + "\x00"

// Text vertex shader — draws textured quads from glyph atlas.
const textVertSrc = `
#version 410 core

layout(location = 0) in uvec2 atlas_pos;   // atlas position (instance)
layout(location = 1) in uvec2 glyph_size;  // glyph dimensions (instance)
layout(location = 2) in ivec2 bearings;    // left, top bearing (instance)
layout(location = 3) in uvec2 grid_pos;    // col, row (instance)
layout(location = 4) in uvec4 color;       // RGBA foreground (instance)

uniform mat4 projection;
uniform vec2 cell_size;
uniform vec2 atlas_size;

out vec2 tex_coord;
out vec4 fg_color;

void main() {
    vec2 corner = vec2(
        float(gl_VertexID == 1 || gl_VertexID == 3),
        float(gl_VertexID == 2 || gl_VertexID == 3)
    );

    vec2 cell_pixel = vec2(grid_pos) * cell_size;
    vec2 pos = cell_pixel + vec2(bearings) + vec2(glyph_size) * corner;

    gl_Position = projection * vec4(pos, 0.0, 1.0);

    // Texture coordinates in atlas (normalized)
    vec2 atlas_norm = vec2(atlas_pos) / atlas_size;
    vec2 size_norm = vec2(glyph_size) / atlas_size;
    tex_coord = atlas_norm + size_norm * corner;

    fg_color = vec4(color) / 255.0;
}
` + "\x00"

const textFragSrc = `
#version 410 core

in vec2 tex_coord;
in vec4 fg_color;

uniform sampler2D atlas;

out vec4 out_color;

void main() {
    float alpha = texture(atlas, tex_coord).r;
    out_color = vec4(fg_color.rgb, fg_color.a * alpha);
}
` + "\x00"

// Cursor vertex shader.
const cursorVertSrc = `
#version 410 core

layout(location = 0) in vec2 cursor_pos; // col, row (instance)
layout(location = 1) in vec4 cursor_col; // RGBA color (instance)

uniform mat4 projection;
uniform vec2 cell_size;

out vec4 c_color;

void main() {
    vec2 corner = vec2(
        float(gl_VertexID == 1 || gl_VertexID == 3),
        float(gl_VertexID == 2 || gl_VertexID == 3)
    );

    vec2 pos = cursor_pos * cell_size + cell_size * corner;
    gl_Position = projection * vec4(pos, 0.0, 1.0);
    c_color = cursor_col;
}
` + "\x00"

const cursorFragSrc = `
#version 410 core

in vec4 c_color;
out vec4 out_color;

void main() {
    out_color = c_color;
}
` + "\x00"

// CompileShader compiles a GLSL shader from source.
func CompileShader(src string, shaderType uint32) (uint32, error) {
	shader := gl.CreateShader(shaderType)
	cSrc, free := gl.Strs(src)
	gl.ShaderSource(shader, 1, cSrc, nil)
	free()
	gl.CompileShader(shader)

	var status int32
	gl.GetShaderiv(shader, gl.COMPILE_STATUS, &status)
	if status == gl.FALSE {
		var logLen int32
		gl.GetShaderiv(shader, gl.INFO_LOG_LENGTH, &logLen)
		log := strings.Repeat("\x00", int(logLen+1))
		gl.GetShaderInfoLog(shader, logLen, nil, gl.Str(log))
		gl.DeleteShader(shader)
		return 0, fmt.Errorf("shader compile: %s", log)
	}
	return shader, nil
}

// LinkProgram links vertex and fragment shaders into a program.
func LinkProgram(vert, frag uint32) (uint32, error) {
	prog := gl.CreateProgram()
	gl.AttachShader(prog, vert)
	gl.AttachShader(prog, frag)
	gl.LinkProgram(prog)

	var status int32
	gl.GetProgramiv(prog, gl.LINK_STATUS, &status)
	if status == gl.FALSE {
		var logLen int32
		gl.GetProgramiv(prog, gl.INFO_LOG_LENGTH, &logLen)
		log := strings.Repeat("\x00", int(logLen+1))
		gl.GetProgramInfoLog(prog, logLen, nil, gl.Str(log))
		gl.DeleteProgram(prog)
		return 0, fmt.Errorf("program link: %s", log)
	}

	gl.DeleteShader(vert)
	gl.DeleteShader(frag)
	return prog, nil
}

// CompileProgram compiles vertex and fragment shader sources and links them.
func CompileProgram(vertSrc, fragSrc string) (uint32, error) {
	vert, err := CompileShader(vertSrc, gl.VERTEX_SHADER)
	if err != nil {
		return 0, fmt.Errorf("vertex: %w", err)
	}
	frag, err := CompileShader(fragSrc, gl.FRAGMENT_SHADER)
	if err != nil {
		gl.DeleteShader(vert)
		return 0, fmt.Errorf("fragment: %w", err)
	}
	return LinkProgram(vert, frag)
}
