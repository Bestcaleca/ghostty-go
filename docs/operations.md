# Ghostty-Go 操作指南

## 环境要求

### 系统依赖

**Linux (Ubuntu/Debian):**
```bash
sudo apt-get install -y \
    libgl1-mesa-dev \
    libx11-dev \
    libxcursor-dev \
    libxrandr-dev \
    libxinerama-dev \
    libxi-dev \
    libxxf86vm-dev
```

**Linux (Fedora/RHEL):**
```bash
sudo dnf install -y \
    mesa-libGL-devel \
    libX11-devel \
    libXcursor-devel \
    libXrandr-devel \
    libXinerama-devel \
    libXi-devel \
    libXxf86vm-devel
```

**macOS:**
```bash
# Xcode Command Line Tools 已包含所需依赖
xcode-select --install
```

### Go 环境

- Go 1.25.0 或更高版本
- CGo 必须启用（OpenGL 绑定需要）

## 构建

### 标准构建

```bash
# 下载依赖
go mod tidy

# 构建
go build -o ghostty-go .
```

### 特定系统构建问题

**Linux - 缺少 libXxf86vm:**

如果遇到链接错误 `-lXxf86vm`，使用以下 workaround：

```bash
mkdir -p /tmp/ghostty-libs
ln -sf /lib/x86_64-linux-gnu/libXxf86vm.so.1 /tmp/ghostty-libs/libXxf86vm.so
CGO_LDFLAGS="-L/tmp/ghostty-libs" go build -o ghostty-go .
```

### 交叉编译

```bash
# macOS ARM64
GOOS=darwin GOARCH=arm64 CGO_ENABLED=1 go build -o ghostty-go-darwin-arm64 .

# Linux AMD64
GOOS=linux GOARCH=amd64 CGO_ENABLED=1 go build -o ghostty-go-linux-amd64 .
```

## 运行

### 直接运行

```bash
# 使用 go run
CGO_LDFLAGS="-L/tmp/ghostty-libs" go run .

# 或使用编译后的二进制
./ghostty-go
```

### 环境变量

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `SHELL` | 默认 shell 路径 | `/bin/sh` |
| `XDG_CONFIG_HOME` | 配置目录 | `~/.config` |
| `TERM` | 终端类型 | `xterm-256color` |

## 配置

### 配置文件位置

```
~/.config/ghostty-go/config.toml
```

首次运行时自动创建默认配置文件。

### 配置示例

```toml
# 字体
font-family = "DejaVu Sans Mono"
font-size = 14.0

# 颜色 (十六进制)
foreground = "#e6e6e6"
background = "#1a1a1f"
cursor-color = "#e6e6e6cc"

# 光标样式: block, beam, underline
cursor-style = "block"
cursor-blink = true

# Shell (留空自动检测)
shell = ""

# 窗口大小
window-width = 1200
window-height = 800

# 滚动缓冲
scrollback-lines = 10000

# 内边距
padding-x = 2.0
padding-y = 1.0

# 快捷键绑定
[[keybindings]]
key = "c"
action = "copy"
mods = "ctrl+shift"

[[keybindings]]
key = "v"
action = "paste"
mods = "ctrl+shift"
```

### 配置项说明

| 配置项 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `font-family` | string | `""` | 字体名称，留空使用系统默认等宽字体 |
| `font-size` | float | `16.0` | 字体大小（磅） |
| `foreground` | string | `#e6e6e6` | 前景色 |
| `background` | string | `#1a1a1f` | 背景色 |
| `cursor-color` | string | `#e6e6e6cc` | 光标颜色（支持 Alpha） |
| `cursor-style` | string | `block` | 光标样式 |
| `cursor-blink` | bool | `true` | 光标闪烁 |
| `shell` | string | `""` | Shell 路径 |
| `window-width` | int | `960` | 窗口宽度 |
| `window-height` | int | `640` | 窗口高度 |
| `scrollback-lines` | int | `10000` | 滚动缓冲行数 |
| `padding-x` | float | `2.0` | 水平内边距 |
| `padding-y` | float | `1.0` | 垂直内边距 |

## 测试

### 运行全部测试

```bash
CGO_LDFLAGS="-L/tmp/ghostty-libs" go test ./... -v -count=1 -timeout 30s
```

### 运行特定包测试

```bash
# 解析器测试
go test ./parser/ -v -count=1

# 终端测试
go test ./terminal/ -v -count=1

# 输入测试
go test ./input/ -v -count=1
```

### 运行基准测试

```bash
go test ./parser/ -bench=. -benchmem
```

### 代码检查

```bash
# 静态分析
go vet ./...

# 格式检查
gofmt -d .
```

## 故障排除

### 问题：窗口无法创建

**症状：** `glfw init: VersionUnavailable: X11: The DISPLAY environment variable is missing`

**解决：** 需要在图形桌面环境中运行，或设置 `DISPLAY` 变量。

### 问题：字体加载失败

**症状：** `font load failed, using default`

**解决：**
1. 安装等宽字体：`sudo apt-get install fonts-dejavu-mono`
2. 或在配置中指定 `font-family`

### 问题：链接错误 `-lXxf86vm`

**解决：**
```bash
mkdir -p /tmp/ghostty-libs
ln -sf /lib/x86_64-linux-gnu/libXxf86vm.so.1 /tmp/ghostty-libs/libXxf86vm.so
CGO_LDFLAGS="-L/tmp/ghostty-libs" go build
```

### 问题：Shell 无法启动

**检查：**
```bash
# 确认 shell 存在
which $SHELL

# 手动指定 shell
# 在 config.toml 中设置 shell = "/bin/bash"
```

## 快捷键参考

| 快捷键 | 功能 |
|--------|------|
| `Ctrl+Shift+V` | 粘贴（带 bracketed paste） |
| `Shift+PageUp` | 向上滚动半页 |
| `Shift+PageDown` | 向下滚动半页 |
| `Shift+Home` | 滚动到底部 |
| `Shift+滚轮` | 视口滚动 |
| `Ctrl+Click` | 打开超链接 |
| `鼠标拖拽` | 选择文本 |
| `双击` | 选择单词 |
| `三击` | 选择行 |
