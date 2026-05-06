# AI 时代终端功能清单

这份清单把 Ghostty-Go 的方向定义为“Agent-native terminal”，不是在传统终端旁边放一个聊天框。核心目标是：保留真正终端的速度、兼容性和可控性，同时让 AI 能读懂会话、解释输出、规划命令、执行任务，并在关键动作前给用户清晰审批。

## 产品定位

- **目标用户**：开发者、运维/SRE、数据工程师、AI 工程师、需要频繁使用 CLI 工具的高级用户。
- **核心场景**：调试报错、理解陌生项目、自动化重复 shell 工作流、代码修改与测试、部署排障、日志分析、远程机器运维。
- **关键原则**：终端永远可直接使用；AI 是增强层，不抢夺控制权；所有破坏性操作必须可预览、可审批、可回滚。
- **差异化方向**：把终端输出结构化成可引用的 Command Block，让 Agent 基于真实上下文行动，而不是只基于用户复制粘贴的片段猜测。

## 当前开发进度

### 2026-05-06 Phase 0 第一批

- 已支持 UTF-8 输出流式解码，终端程序输出中文和其他多字节字符不会再被 parser 忽略。
- 已修复 CSI REP (`CSI b`) 重复字符时的锁重入死锁。
- 已修复 SGR 256 色 (`38;5;n` / `48;5;n`) palette 解析。
- 已补齐基础 DA、DSR、DECID 响应，减少 TUI 程序等待终端查询结果的概率。
- 已让窗口 resize 同步更新 terminal grid、PTY size 和 renderer grid bounds。
- 已增加上述行为的回归测试。

### 2026-05-06 Phase 0 第二批

- 已让鼠标点击、拖拽选择、滚轮上报统一扣除 renderer padding，避免命中位置与实际文字显示偏移。
- 已扩展 renderer cell 样式，支持 underline、strikethrough、overline 装饰标记。
- 已增加独立 decoration shader pass，用细矩形绘制下划线、删除线和上划线。
- 已增加 padding 坐标映射、renderer metrics 和装饰线构建的回归测试。

## P0：终端基础能力

这些能力是 AI 功能上线前的地基。地基不稳，Agent 只会放大问题。

### 1. VT/xterm 兼容

- 完整 UTF-8 流式解码，支持中文、emoji、组合字符和宽字符。
- 修复 CSI REP、DA、DSR、DECID 等常见查询响应。
- 修复窗口 resize 后的终端网格、PTY size、渲染网格同步。
- 完善备用屏幕、鼠标模式、Bracketed paste、焦点事件、光标样式。
- 增加 terminfo，默认 `TERM` 建议使用兼容值，例如 `xterm-ghostty-go` 或 `xterm-256color` 过渡。
- 增加兼容性测试：`vttest`、`ncurses`、`vim`、`nvim`、`tmux`、`less`、`fzf`、`htop`、`ssh`。

### 2. 字体和渲染

- 字体 fallback：主字体缺字时自动查找 CJK、emoji、符号字体。
- 支持粗体、斜体、下划线、删除线、反转、暗淡、隐藏字符。
- 支持 ligature、字重映射、字体大小缩放、行高配置。
- 文本布局使用字体 ascent/descent/advance，不用经验值。
- GPU 渲染继续保留背景、文本、光标分层，后续可增加装饰线 pass。
- 增加 FPS、帧耗时、glyph cache 命中率调试面板。

### 3. 传统终端交互

- 右键菜单：复制、粘贴、全选、清除选择、搜索、用 AI 解释选中内容。
- 文本选择：拖拽、双击词、三击行、矩形选择、按块选择。
- 搜索：当前屏幕、scrollback、正则、大小写、跳转匹配。
- 滚动：scrollback 限制、清屏策略、保存/导出会话。
- 快捷键系统：配置文件可覆盖默认快捷键，`--help` 输出常用快捷键。
- 标签页、分屏、面板切换、会话恢复。

## P1：现代输入层

### 4. Universal Input

- 一个输入框同时支持 Shell Mode、Agent Mode、Auto Mode。
- Shell Mode：行为与传统终端一致。
- Agent Mode：自然语言任务，例如“帮我找出这个测试为什么失败”。
- Auto Mode：本地轻量判断输入是命令还是自然语言，可手动切换。
- 支持多行编辑、历史搜索、补全、语法高亮、粘贴预览。
- 支持输入前缀：`!` 强制命令，`*` 强制 Agent，`/` 触发内置命令。

### 5. 命令智能

- 自然语言转命令：生成命令前解释用途和风险。
- 命令解释：解释光标所在命令、选中命令、历史命令。
- 命令修复：命令失败后基于退出码和输出给出修复建议。
- 命令预检：识别 `rm -rf`、生产库、远程执行、权限提升等高风险动作。
- 命令模板：把常用命令保存为可参数化 workflow。
- Shell 补全增强：结合当前目录、Git 状态、包管理器、项目类型补全。

## P1：Command Block 系统

### 6. 结构化会话

- 每次命令执行形成一个 block：命令、工作目录、环境摘要、开始时间、结束时间、退出码、输出。
- block 可折叠、复制、搜索、置顶、重命名。
- 支持对单个 block 做 AI 操作：解释、总结、修复、生成 issue、生成 commit message。
- 支持把多个 block 附加为 Agent 上下文。
- 支持 block 导出为 Markdown、JSON、纯文本。

### 7. 输出理解

- 自动识别错误：编译错误、测试失败、权限错误、网络错误、Git 冲突。
- 错误行可点击跳转到文件。
- 日志输出支持时间戳、级别、trace id、URL、文件路径识别。
- 长输出自动折叠，保留错误和摘要。
- 支持“只把失败片段交给 AI”，减少 token 和隐私风险。

## P2：Agent Mode

### 8. Agent 执行模型

- Agent 运行在终端内部，但通过明确工具边界行动。
- 工具层至少包括：读文件、搜索文件、编辑文件、运行命令、读取 Git、读取 block、写任务列表。
- 每个工具调用都记录到会话历史。
- 支持多步任务：计划、执行、观察输出、修正计划、继续执行。
- 支持后台任务：长时间构建、测试、日志跟踪、issue triage。
- 支持中断、暂停、恢复、重试。

### 9. 权限与审批

- **Suggest**：只读上下文，只给建议。
- **Auto Edit**：可自动改文件，但运行命令前询问。
- **Full Auto**：可在沙箱或工作区内自动改文件和运行命令。
- 每个命令执行前显示：命令、工作目录、可能影响、是否联网、是否写文件。
- 文件修改必须有 diff 预览，可逐块接受。
- 支持 allowlist：例如允许 `go test ./...`、`npm test`，但拒绝 `sudo`。
- 支持 blocklist：危险命令、敏感路径、生产环境变量。
- 支持一键回滚：Git checkpoint、文件快照、命令历史。

### 10. 上下文系统

- 自动读取项目类型：Go、Node、Python、Rust、Docker、Kubernetes。
- 自动附加 Git 状态、当前分支、最近提交、未提交 diff。
- 支持项目说明文件：`AGENTS.md`、`GEMINI.md`、`.codex`、`.ghostty-go/context.md`。
- 支持用户记忆：偏好的命令、测试方式、代码风格、常用服务。
- 支持隐私过滤：`.env`、密钥、token、SSH key、云凭证默认不发送。
- 支持上下文预算：显示当前任务使用了哪些文件、blocks、命令输出。

## P2：AI 工作流

### 11. 开发工作流

- “解释这个项目”：读取目录、README、构建脚本，生成结构图。
- “修复这个测试”：读取失败 block，定位文件，修改代码，跑测试。
- “生成提交”：总结 diff，建议 commit message，必要时拆分提交。
- “代码审查”：检查未提交 diff，列出风险和缺失测试。
- “依赖升级”：查看 package/go modules，生成升级计划并跑验证。
- “生成文档”：从代码和命令输出生成 README、操作手册、排障文档。

### 12. 运维工作流

- 日志解释：从 `kubectl logs`、`journalctl`、`docker logs` 中提取异常。
- 部署排障：读取 pod、service、events、ingress、helm 状态。
- SSH 会话保护：明确本地/远程边界，远程高危命令二次确认。
- 资源分析：CPU、内存、磁盘、网络、进程树摘要。
- Runbook 执行：把团队操作手册转成可审批步骤。
- 事故复盘：导出关键 blocks、时间线、命令、结论。

### 13. 数据和云工作流

- SQL 辅助：解释查询、生成索引建议、标记危险写操作。
- 云 CLI 辅助：AWS/GCP/Azure 命令解释、权限错误诊断。
- Terraform 辅助：解释 plan、标记会删除的资源。
- CI/CD 辅助：读取失败日志，定位失败阶段，生成修复建议。
- API 调试：curl/httpie 请求生成、响应解释、重试策略。

## P3：生态与协议

### 14. MCP 集成

- 内置 MCP client，连接 GitHub、Slack、数据库、浏览器、云服务等工具。
- 可在设置中管理 MCP server：启用、禁用、权限、工具范围。
- 工具调用必须显示来源、参数和结果。
- 支持把当前终端作为 MCP server 暴露给外部 Agent，提供只读 block 查询和受控命令执行。

### 15. Agent 互操作

- 支持接入多种 Agent 后端：OpenAI Codex、Claude Code、Gemini CLI、本地模型、自定义 HTTP provider。
- 保留 provider 抽象：模型、上下文格式、工具协议、审批策略分离。
- 评估 Agent Client Protocol 这类编辑器/Agent 互操作协议，避免只绑定单一生态。
- 支持同一任务切换模型：便宜模型做分类，强模型做复杂修改。

### 16. 扩展系统

- Slash command：`/fix-test`、`/explain-last`、`/deploy-check`。
- Workflow 插件：用 YAML/TOML 定义输入、命令、审批和输出格式。
- 主题插件：颜色、字体、边框、动效、block 样式。
- Tool 插件：新增命令检查器、日志解析器、云服务连接器。
- 插件必须有权限声明和签名校验。

## P3：产品体验

### 17. UI 组件

- Agent 侧栏：任务计划、工具调用、diff、审批、日志。
- Command Palette：搜索所有动作、快捷键、workflow、profile。
- Status Bar：当前目录、Git 分支、Agent 模式、权限模式、远程状态。
- Diff Viewer：文件改动逐块接受、拒绝、打开编辑器。
- Task List：展示 Agent 当前计划和进度。
- Context Drawer：展示本次回答引用了哪些 block、文件、命令。

### 18. 可配置性

- 配置文件支持字体、主题、快捷键、shell、启动目录、Agent provider。
- Profile 支持不同场景：本地开发、SSH 运维、生产只读、演示模式。
- 每个 profile 可设置不同权限策略。
- 设置界面和纯文本配置并存。
- `ghostty-go --help` 展示 CLI 参数，`ghostty-go config --schema` 输出配置 schema。

### 19. 隐私、安全、审计

- 本地优先：不启用 AI 时不上传任何终端内容。
- AI 请求前可预览上下文。
- 密钥自动打码，支持自定义敏感规则。
- 审计日志记录 Agent 看过什么、运行了什么、改了什么。
- 企业模式支持模型白名单、MCP 白名单、网络策略。
- 高风险命令必须二次确认，不能被 allowlist 静默绕过。

## 建议实施路线

### Phase 0：把终端修到能长期使用

- 修复 UTF-8、REP、DA/DSR、resize、256 色。
- 完善字体 fallback 和文本装饰线。
- 增加 `vttest`/真实 TUI 应用回归清单。
- 把右键菜单、选择、复制粘贴、help 做到稳定。

### Phase 1：现代终端体验

- 实现 Command Block。
- 实现搜索、命令历史、Command Palette。
- 实现 tabs/splits 和 profile。
- 增加配置 schema 和快捷键配置。

### Phase 2：AI 辅助但不自动改

- 增加 Agent 侧栏和 Universal Input。
- 支持解释命令、解释错误、总结 block。
- 支持自然语言生成命令，但默认只建议不执行。
- 增加上下文预览和隐私过滤。

### Phase 3：可审批 Agent

- 实现工具层、权限模式、命令审批、文件 diff。
- 支持修测试、改代码、跑验证、生成提交。
- 支持 checkpoint 和回滚。
- 支持后台长任务。

### Phase 4：生态化

- 增加 MCP client/server。
- 接入多 Agent provider。
- 支持 workflow 插件和团队共享 runbook。
- 增加审计、策略、企业配置。

## 近期最小可行版本

第一版不要追求“大而全”。建议 MVP 只做这些：

- 稳定终端：UTF-8、resize、选择、复制粘贴、字体、右键菜单。
- Command Block：命令、输出、退出码、耗时、可复制、可折叠。
- `Ask AI about this block`：对失败输出解释原因。
- `Suggest command`：自然语言生成命令，但用户手动确认执行。
- 权限模式：只读 AI，不允许自动写文件。
- 上下文预览：用户能看到要发给模型的文本。

做到这个阶段，产品就已经不是“玩具终端”，而是一个可靠的 AI 增强终端原型。

## 参考产品与资料

- Ghostty Features: https://ghostty.org/docs/features
- Warp Universal Input: https://docs.warp.dev/terminal
- Warp Agents: https://docs.warp.dev/agents/warp-ai/agent-mode
- OpenAI Codex CLI: https://help.openai.com/en/articles/11096431-openai-codex-cli-getting-started
- Claude Code Overview: https://code.claude.com/docs/en/overview
- Gemini CLI: https://developers.google.com/gemini-code-assist/docs/gemini-cli
- Gemini CLI README: https://github.com/google-gemini/gemini-cli
- Zed Agent Client Protocol: https://zed.dev/acp
- Windows Terminal Command Palette: https://learn.microsoft.com/windows/terminal/command-palette
