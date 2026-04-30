# Ghostty-Go 架构文档

## 概述

Ghostty-Go 是一个用 Go 编写的 GPU 加速终端模拟器，灵感来自 [Ghostty](https://github.com/ghostty-org/ghostty)（Zig 实现）。项目采用 OpenGL 4.1 渲染，支持 Linux 和 macOS。

## 系统架构

```
┌─────────────────────────────────────────────────────────────┐
│                        GLFW Window                          │
│  ┌─────────────────────────────────────────────────────┐   │
│  │                    Main Goroutine                    │   │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────────────┐  │   │
│  │  │  GLFW    │  │  OpenGL  │  │   Surface Hub    │  │   │
│  │  │  Events  │→ │ Renderer │← │  (Integration)   │  │   │
│  │  └──────────┘  └──────────┘  └──────────────────┘  │   │
│  └─────────────────────────────────────────────────────┘   │
│                           ↕                                 │
│  ┌─────────────────────────────────────────────────────┐   │
│  │                   IO Goroutines                     │   │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────────────┐  │   │
│  │  │   PTY    │  │  Stream  │  │   Terminal       │  │   │
│  │  │  Read    │→ │  Parser  │→ │   State          │  │   │
│  │  └──────────┘  └──────────┘  └──────────────────┘  │   │
│  └─────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

## 包依赖关系

```
main
├── config          # TOML 配置
├── surface         # 集成中心
│   ├── terminal    # 终端状态机
│   │   ├── parser  # VT500 解析器
│   │   └── unicode # Unicode 工具
│   ├── renderer    # OpenGL 渲染
│   │   └── font    # 字体系统
│   ├── input       # 输入处理
│   └── termio      # PTY 管理
└── (依赖 go-gl, creack/pty, freetype, x/text)
```

## 并发模型

Ghostty-Go 使用 Go 的 goroutine + channel 模型，替代 Zig 的 mailbox 机制：

```
┌─────────────────┐         ┌─────────────────┐
│  Main Goroutine  │         │  IO Goroutine   │
│  (GLFW + GL)    │         │  (PTY Read)     │
│                 │         │                 │
│  glfw.PollEvents│         │  PTY.Read       │
│  RenderGrid     │←─chan───│  stream.Process │
│  SwapBuffers    │         │  Check Title    │
└─────────────────┘         └─────────────────┘
         ↑                           │
         │                           │
    ┌────┴────┐              ┌───────┴───────┐
    │  Input  │              │  Write        │
    │  Events │              │  Goroutine    │
    └─────────┘              └───────────────┘
```

### 线程安全

- **终端状态**：受 `sync.RWMutex` 保护
  - IO goroutine 获取写锁
  - 渲染 goroutine 获取读锁
- **GLFW/GL**：必须在主线程调用
- **Channel 通信**：
  - `writeChan chan []byte` — 键盘输入 → PTY
  - `msgChan chan Message` — IO → Surface 通知
  - 非阻塞发送，防止 goroutine 泄漏

## 包详细设计

### 1. parser/ — VT500 解析器

**职责：** 将原始字节流解析为 VT 逃逸序列动作。

```go
type Parser struct {
    state           State           // 14 种状态之一
    intermediates   [4]byte         // 中间字节
    params          [24]uint16      // 参数
    paramCount      int
    private         bool            // DEC 私有模式 (?)
    oscBuf          []byte          // OSC 缓冲
}

func (p *Parser) Next(b byte) [3]Action
```

**状态机：**
```
Ground ←── Execute/Print
  │
  ├── ESC → Escape → EscapeIntermediate
  │                    └── Dispatch → Ground
  │
  ├── CSI → CSIEntry → CSIParam → CSIDispatch → Ground
  │           └── CSIIntermediate
  │
  └── OSC → OSCString → Ground
```

**关键设计：**
- 表驱动：`[256][14]Entry` 查找表，O(1) 状态转换
- 零分配热路径：`Print` 和 `Execute` 不分配内存
- 固定大小数组：24 参数、4 中间字节

### 2. terminal/ — 终端状态机

**职责：** 维护终端显示状态，处理 VT 序列。

```go
type Terminal struct {
    primary     *Screen     // 主屏幕
    alternate   *Screen     // 备用屏幕
    active      *Screen     // 当前活动屏幕
    Rows, Cols  int
    CurrentStyle Style      // 当前 SGR 样式
    // ... 回调函数
}
```

**Screen 结构：**
```go
type Screen struct {
    Rows      []Row         // 显示行
    Cursor    Cursor        // 光标状态
    Modes     ModeState     // ~30 种模式标志
    Styles    *StyleTable   // 样式去重表
    Scrollback []Row        // 滚动缓冲
    Selection Selection     // 文本选择
}
```

**Cell 结构：**
```go
type Cell struct {
    Char      rune      // Unicode 字符
    Width     uint8     // 显示宽度 (0/1/2)
    Style     StyleID   // 样式 ID
    Hyperlink string    // OSC 8 超链接
}
```

**关键模式：**
| 模式 | 编号 | 说明 |
|------|------|------|
| DECAWM | -7 | 自动换行 |
| DECOM | -6 | 原点模式 |
| IRM | 4 | 插入模式 |
| DECSCUSR | - | 光标样式 |
| 1049 | -1049 | 备用屏幕 |
| 2004 | -2004 | Bracketed paste |

### 3. renderer/ — OpenGL 渲染器

**职责：** 将终端网格渲染为 GPU 加速的文本。

**渲染管线：**
```
1. 背景 Pass：绘制彩色背景矩形（实例化）
2. 文本 Pass：从字形图集采样，绘制文本四边形（实例化）
3. 光标 Pass：绘制光标（block/beam/underline）
```

**字形图集：**
```
┌──────────────────────────────────┐
│  Atlas (2048x2048)               │
│  ┌───┬───┬───┬───┬───┬───┐      │
│  │ A │ B │ C │ D │ E │ F │ ...  │  Shelf 1
│  └───┴───┴───┴───┴───┴───┘      │
│  ┌───┬───┬───┬───┬───┐          │
│  │ G │ H │ I │ J │ K │    ...   │  Shelf 2
│  └───┴───┴───┴───┴───┘          │
│  ...                             │
└──────────────────────────────────┘
```

- Shelf packing 算法：O(1) 分配
- 按需光栅化：首次使用时生成字形
- 灰度纹理：单通道用于文本渲染

**Instanced Rendering：**
```go
// 每个单元格是一个实例
// 背景：6 floats/instance (col, row, r, g, b, a)
// 文本：12 floats/instance (atlasX, atlasY, glyphW, glyphH, ...)
gl.DrawArraysInstanced(gl.TRIANGLES, 0, 6, instanceCount)
```

### 4. input/ — 输入处理

**职责：** 将 GLFW 事件转换为终端转义序列。

**键盘编码：**
```
GLFW Key + Modifiers → Escape Sequence

示例：
  Up Arrow      → \x1bOA (应用模式) 或 \x1b[A (普通模式)
  Ctrl+C        → \x03
  F1            → \x1b[11~
  Shift+Insert  → \x1b[2;2~
```

**鼠标编码：**
```
GLFW Mouse + Position → SGR Sequence

示例：
  左键按下 (5,10) → \x1b[<0;6;11M
  左键释放        → \x1b[<3;6;11m
  滚轮上滚        → \x1b[<64;6;11M
```

### 5. termio/ — PTY 管理

**职责：** 管理伪终端连接和 IO 循环。

```go
type Termio struct {
    pty       *PTY              // PTY 连接
    stream    *terminal.Stream  // 解析流
    terminal  *terminal.Terminal
    resize    *ResizeCoalescer  // 调整防抖
    writeChan chan []byte        // 写入通道
    done      chan struct{}      // 关闭信号
}
```

**Resize Coalescer：**
```
快速调整事件 → [25ms 防抖] → 最终大小 → PTY.Resize

防止窗口拖拽时频繁调整 PTY 大小
```

### 6. surface/ — 集成中心

**职责：** 连接所有子系统，路由事件。

```go
type Surface struct {
    terminal  *terminal.Terminal
    stream    *terminal.Stream
    termio    *termio.Termio
    renderer  *renderer.Renderer
    keyH      *input.KeyHandler
    mouseH    *input.MouseHandler
    window    *glfw.Window
    // ... 状态
}
```

**事件流：**
```
键盘输入 → KeyHandler.EncodeKey → termio.Write → PTY
鼠标点击 → MouseHandler.EncodeMouseButton → termio.Write → PTY
PTY 输出 → stream.Process → 终端状态更新 → RenderGrid
窗口调整 → HandleResize → renderer.Resize + termio.Resize
```

### 7. config/ — 配置系统

**职责：** 加载和保存 TOML 配置。

```go
type Config struct {
    FontFamily      string  `toml:"font-family"`
    FontSize        float64 `toml:"font-size"`
    Foreground      string  `toml:"foreground"`
    Background      string  `toml:"background"`
    // ...
}
```

**配置路径：** `~/.config/ghostty-go/config.toml`（XDG 标准）

### 8. unicode/ — Unicode 工具

**职责：** 字符宽度计算和 Grapheme cluster 分割。

- **RuneWidth**：东亚宽度 (0/1/2)
- **GraphemeClusters**：UAX #29 分割规则
- 支持 emoji ZWJ 序列、区域指示符

### 9. font/ — 字体系统

**职责：** 字体加载和字形光栅化。

- TrueType 解析（golang/freetype）
- 字形位图生成
- 字体度量（cell 宽高、ascent/descent）

## 数据流

### 完整输入→输出流程

```
1. 用户按键
2. GLFW 捕获键盘事件
3. input.KeyHandler 编码为转义序列
4. surface.HandleKey 调用 termio.Write
5. termio.writeLoop 写入 PTY
6. Shell 处理输入，产生输出
7. termio.readLoop 从 PTY 读取
8. terminal.Stream.Process 解析字节
9. parser.Parser.Next 解析每个字节
10. terminal 执行动作（Print/CSI/ESC/OSC）
11. 终端状态更新（Grid/Cursor/Styles）
12. surface.RenderGrid 读取状态
13. 转换为 renderer.Cell
14. OpenGL 渲染到屏幕
```

## 设计决策

| 决策 | 原因 |
|------|------|
| 表驱动解析器 | O(1) 状态转换，零分配 |
| 固定大小数组 | 避免热路径内存分配 |
| sync.RWMutex | 渲染器读锁，IO 写锁，无竞争 |
| Channel 通信 | Go 原生并发，无外部依赖 |
| Shelf packing atlas | O(1) 字形分配，足够终端使用 |
| OpenGL 4.1 | macOS 最高支持版本 |
| 接口解耦 | StreamHandler 接口分离解析器和终端 |

## 性能指标

| 指标 | 值 |
|------|-----|
| 解析器 Print | 7.8 ns/op, 0 allocs |
| 解析器混合序列 | 302 ns/op, 2 allocs |
| 测试数量 | 63 |
| 代码行数 | ~8000 |
