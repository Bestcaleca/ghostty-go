# Ghostty-Go 功能文档

## 目录

1. [VT 终端模拟](#vt-终端模拟)
2. [Unicode 支持](#unicode-支持)
3. [字体和渲染](#字体和渲染)
4. [输入处理](#输入处理)
5. [鼠标功能](#鼠标功能)
6. [剪贴板](#剪贴板)
7. [滚动缓冲](#滚动缓冲)
8. [超链接](#超链接)
9. [文本选择](#文本选择)
10. [配置系统](#配置系统)
11. [终端铃](#终端铃)

---

## VT 终端模拟

### VT500 系列解析器

完整的 VT500 系列终端解析器，支持 14 种状态：

| 状态 | 说明 |
|------|------|
| Ground | 默认状态 |
| Escape | ESC 序列 |
| EscapeIntermediate | ESC 中间字节 |
| CSIEntry | CSI 入口 |
| CSIParam | CSI 参数 |
| CSIIntermediate | CSI 中间字节 |
| CSIIgnore | CSI 忽略 |
| DCSEntry/Param/Intermediate/Passthrough/Ignore | DCS 序列 |
| OSCString | OSC 字符串 |
| SOSPMAPCString | SOS/PM/APC 字符串 |

**支持的动作：**
- `Print` — 打印字符
- `Execute` — 执行控制字符
- `CSIDispatch` — CSI 序列分发
- `EscDispatch` — ESC 序列分发
- `OSCDispatch` — OSC 命令分发
- `DCS` — 设备控制字符串

### CSI 处理

**光标移动：**
```
CSI A        — 光标上移
CSI B        — 光标下移
CSI C        — 光标右移
CSI D        — 光标左移
CSI H        — 光标定位 (CUP)
CSI J        — 擦除显示 (ED)
CSI K        — 擦除行 (EL)
CSI L        — 插入行 (IL)
CSI M        — 删除行 (DL)
CSI P        — 删除字符 (DCH)
CSI @        — 插入字符 (ICH)
```

**SGR (图形渲染)：**
```
CSI 0 m      — 重置
CSI 1 m      — 粗体
CSI 3 m      — 斜体
CSI 4 m      — 下划线
CSI 7 m      — 反转
CSI 9 m      — 删除线
CSI 30-37 m  — 前景色 (ANSI)
CSI 40-47 m  — 背景色 (ANSI)
CSI 38;5;n m — 前景色 (256色)
CSI 48;5;n m — 背景色 (256色)
CSI 38;2;r;g;b m — 前景色 (真彩色)
CSI 48;2;r;g;b m — 背景色 (真彩色)
```

**模式设置：**
```
CSI ? 7 h    — DECAWM (自动换行)
CSI ? 6 h    — DECOM (原点模式)
CSI ? 25 h   — DECTCEM (光标可见)
CSI ? 1049 h — 备用屏幕
CSI ? 2004 h — Bracketed paste
CSI ? 1000 h — 鼠标跟踪 (普通)
CSI ? 1002 h — 鼠标跟踪 (按钮)
CSI ? 1003 h — 鼠标跟踪 (任意)
```

### ESC 处理

```
ESC 7        — 保存光标 (DECSC)
ESC 8        — 恢复光标 (DECRC)
ESC c        — 完全重置 (RIS)
ESC D        — 索引 (IND)
ESC M        — 反向索引 (RI)
ESC ( 0      — G0 字符集 (DEC 特殊图形)
ESC ( B      — G0 字符集 (ASCII)
```

### OSC 处理

```
OSC 0 ; title ST  — 设置窗口标题
OSC 2 ; title ST  — 设置图标名称
OSC 4 ; idx;spec ST — 设置调色板颜色
OSC 8 ; id;uri ST — 设置超链接
OSC 10 ; spec ST  — 设置前景色
OSC 11 ; spec ST  — 设置背景色
OSC 12 ; spec ST  — 设置光标颜色
OSC 52 ; c;b64 ST — 剪贴板操作
OSC 104 ; idx ST  — 重置调色板颜色
```

---

## Unicode 支持

### 东亚字符宽度

正确处理东亚字符的显示宽度：

| 字符 | 宽度 | 示例 |
|------|------|------|
| ASCII | 1 | A, B, 1, 2 |
| CJK 汉字 | 2 | 中, 文, 日 |
| 全角 | 2 | Ａ, ０ |
| Emoji | 2 | 😀, 🎉 |
| 组合标记 | 0 | ́ (重音) |
| 控制字符 | 0 | ESC, BEL |

### Grapheme Cluster 分割

实现 UAX #29 扩展 Grapheme cluster 边界规则：

- **GB3** — CR+LF 不分割
- **GB4/5** — 控制字符前后分割
- **GB6-8** — Hangul 音节序列不分割
- **GB9** — 扩展字符和 ZWJ 不分割
- **GB9a** — 间距标记不分割
- **GB9b** — 前置字符后不分割
- **GB11** — Emoji ZWJ 序列不分割
- **GB12/13** — 区域指示符对不分割

**示例：**
```
"é" (e + 组合重音) → 1 个 cluster
"👨‍👩‍👧‍👦" (家庭 emoji) → 1 个 cluster
"🇺🇸" (美国国旗) → 1 个 cluster
```

---

## 字体和渲染

### 字体加载

- 支持 TrueType (.ttf) 和 OpenType (.otf) 字体
- 自动检测系统等宽字体
- 支持配置指定字体族

**默认搜索路径：**
```
Linux:
  /usr/share/fonts/truetype/dejavu/DejaVuSansMono.ttf
  /usr/share/fonts/truetype/liberation/LiberationMono-Regular.ttf
  /usr/share/fonts/truetype/ubuntu/UbuntuMono-R.ttf

macOS:
  /System/Library/Fonts/Menlo.ttc
  /System/Library/Fonts/Monaco.dfont
```

### 字形图集

**Shelf Packing 算法：**
```
┌─────────────────────────────────────┐
│  Atlas (2048x2048)                  │
│  ┌───┬───┬───┬───┬───┬───┬───┐     │
│  │ A │ B │ C │ D │ E │ F │ G │     │ Shelf 1 (高度: ascent)
│  └───┴───┴───┴───┴───┴───┴───┘     │
│  ┌───┬───┬───┬───┬───┬───┐         │
│  │ H │ I │ J │ K │ L │ M │   ...   │ Shelf 2
│  └───┴───┴───┴───┴───┴───┘         │
│  ...                                │
└─────────────────────────────────────┘
```

- O(1) 分配：只需维护当前 shelf 的 x 偏移
- 按需生成：字符首次使用时才光栅化
- 灰度纹理：单通道用于文本渲染

### OpenGL 渲染

**两遍渲染：**
1. **背景 Pass** — 绘制每个单元格的背景色（实例化四边形）
2. **文本 Pass** — 从图集采样字形，绘制文本（实例化四边形）
3. **光标 Pass** — 绘制光标

**Instanced Rendering：**
```go
// 每个单元格是一个实例，6 个顶点（2 个三角形）
gl.DrawArraysInstanced(gl.TRIANGLES, 0, 6, instanceCount)
```

**着色器：**
- 顶点着色器：变换位置，传递 UV 坐标和颜色
- 片段着色器：采样图集纹理，混合前景/背景色

---

## 输入处理

### 键盘编码

**功能键：**
| 键 | 普通模式 | 应用模式 |
|----|----------|----------|
| Up | \x1b[A | \x1bOA |
| Down | \x1b[B | \x1bOB |
| Right | \x1b[C | \x1bOC |
| Left | \x1b[D | \x1bOD |
| Home | \x1b[H | \x1bOH |
| End | \x1b[F | \x1bOF |
| F1 | \x1b[11~ | — |
| F12 | \x1b[24~ | — |

**修饰键编码：**
| 修饰键 | 值 |
|--------|-----|
| Shift | +1 |
| Alt | +2 |
| Control | +4 |
| Super | +8 |

**Control 组合键：**
| 组合键 | 输出 |
|--------|------|
| Ctrl+C | 0x03 (ETX/SIGINT) |
| Ctrl+D | 0x04 (EOT) |
| Ctrl+Z | 0x1A (SIGTSTP) |
| Ctrl+[ | 0x1B (ESC) |

### 鼠标编码

**SGR 扩展模式：**
```
按下: CSI < Cb ; Cx ; Cy M
释放: CSI < Cb ; Cx ; Cy m

Cb = 按钮代码 + 修饰键位
Cx = 列 (1-based)
Cy = 行 (1-based)
```

**按钮代码：**
| 按钮 | 代码 |
|------|------|
| 左键 | 0 |
| 中键 | 1 |
| 右键 | 2 |
| 释放 | 3 |
| 滚轮上 | 64 |
| 滚轮下 | 65 |

---

## 鼠标功能

### 鼠标跟踪模式

| 模式 | DECSET | 说明 |
|------|--------|------|
| 无 | — | 不报告 |
| X10 | 9 | 仅报告按下 |
| Normal | 1000 | 按下 + 释放 |
| Button | 1002 | 按下 + 释放 + 拖拽 |
| Any | 1003 | 所有事件 |
| SGR | 1006 | SGR 扩展编码 |

### 鼠标事件处理

```go
// 点击
surface.HandleMouseButton(button, action, mods, x, y)

// 移动
surface.HandleMouseMotion(x, y)

// 滚动
surface.HandleScroll(xoff, yoff, x, y, mods)
```

---

## 剪贴板

### OSC 52 支持

**写入剪贴板：**
```
OSC 52 ; c ; base64data ST
```

**查询剪贴板：**
```
OSC 52 ; c ; ? ST
→ OSC 52 ; c ; base64data ST (响应)
```

**剪贴板类型：**
- `c` — 系统剪贴板
- `p` — 主选择
- `s` — 次选择

**系统集成：**
- Linux: GLFW 剪贴板 API
- macOS: GLFW 剪贴板 API
- Windows: GLFW 剪贴板 API

---

## 滚动缓冲

### 配置

```toml
# config.toml
scrollback-lines = 10000
```

### 滚动操作

| 操作 | 快捷键 |
|------|--------|
| 向上滚动半页 | Shift+PageUp |
| 向下滚动半页 | Shift+PageDown |
| 滚动到底部 | Shift+Home |
| 视口滚动 | Shift+滚轮 |

### 实现

- 滚动缓冲存储在 `Screen.Scrollback`
- 达到最大行数时丢弃最旧的行
- 视口偏移追踪滚动位置
- 光标在滚动回溯时隐藏

---

## 超链接

### OSC 8 协议

**开始超链接：**
```
OSC 8 ; id=xxx ; uri=https://example.com ST
```

**结束超链接：**
```
OSC 8 ; ; ST
```

### 使用方式

1. 应用程序输出带超链接的文本
2. 终端存储 URL 到 Cell.Hyperlink
3. 用户 Ctrl+Click 打开链接
4. 调用系统默认浏览器

**支持的应用：**
- `ls --hyperlink=auto`
- `gcc` 错误信息
- `git diff` 输出

**浏览器调用：**
```go
// Linux
exec.Command("xdg-open", url)

// macOS
exec.Command("open", url)

// Windows
exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
```

---

## 文本选择

### 选择模式

| 模式 | 触发 | 行为 |
|------|------|------|
| 字符 | 单击拖拽 | 逐字符选择 |
| 单词 | 双击 | 选择整个单词 |
| 行 | 三击 | 选择整行 |

### 单词边界

单词字符包括：
- 字母 (a-z, A-Z)
- 数字 (0-9)
- 下划线 (_)
- 点 (.)
- 破折号 (-)
- 斜杠 (/)

### 选择操作

- **拖拽选择** — 按住左键拖动
- **自动复制** — 释放鼠标时自动复制到剪贴板
- **高亮显示** — 选中文本交换前景/背景色
- **清除选择** — 右键点击清除选择

---

## 配置系统

### 配置文件

**位置：** `~/.config/ghostty-go/config.toml`

**首次运行：** 自动创建默认配置文件

### 完整配置项

```toml
# === 字体 ===
font-family = "DejaVu Sans Mono"  # 字体名称
font-size = 16.0                   # 字体大小

# === 颜色 ===
foreground = "#e6e6e6"            # 前景色
background = "#1a1a1f"            # 背景色
cursor-color = "#e6e6e6cc"        # 光标颜色

# === 光标 ===
cursor-style = "block"            # block, beam, underline
cursor-blink = true               # 光标闪烁

# === Shell ===
shell = ""                        # 留空自动检测

# === 窗口 ===
window-width = 960                # 窗口宽度
window-height = 640               # 窗口高度

# === 终端 ===
scrollback-lines = 10000          # 滚动缓冲行数

# === 布局 ===
padding-x = 2.0                   # 水平内边距
padding-y = 1.0                   # 垂直内边距

# === 快捷键 ===
[[keybindings]]
key = "c"
action = "copy"
mods = "ctrl+shift"

[[keybindings]]
key = "v"
action = "paste"
mods = "ctrl+shift"
```

### 颜色格式

支持两种格式：
- `#RRGGBB` — 6 位十六进制
- `#RRGGBBAA` — 8 位十六进制（带 Alpha）

**示例：**
```
#ffffff     — 白色
#000000     — 黑色
#ff0000     — 红色
#00ff0080   — 半透明绿色
```

---

## 终端铃

### BEL 字符 (0x07)

当终端收到 BEL 字符时触发铃。

### 视觉铃

- 屏幕背景短暂反转（100ms）
- 自动恢复原始颜色

### 自定义回调

```go
surface.SetOnBell(func() {
    // 自定义铃处理
    fmt.Println("Bell!")
})
```

---

## 快捷键参考表

| 快捷键 | 功能 | 说明 |
|--------|------|------|
| `Ctrl+Shift+V` | 粘贴 | Bracketed paste |
| `Shift+PageUp` | 向上滚动 | 半页 |
| `Shift+PageDown` | 向下滚动 | 半页 |
| `Shift+Home` | 滚动到底部 | — |
| `Shift+滚轮` | 视口滚动 | 连续滚动 |
| `Ctrl+Click` | 打开超链接 | 在浏览器中打开 |
| `鼠标拖拽` | 选择文本 | 字符模式 |
| `双击` | 选择单词 | 单词模式 |
| `三击` | 选择行 | 行模式 |
| `Ctrl+C` | SIGINT | 中断信号 |
| `Ctrl+D` | EOF | 文件结束 |
| `Ctrl+Z` | SIGTSTP | 挂起信号 |
