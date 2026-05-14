# GST 项目速读

## 项目定位

GST (GPU Support Toolkit) 是 GPU 日志分析工具，把 GB 级 apitrace 日志解析成结构化帧级 OpenGL 事实数据，服务于 AI agent、FAE 和研发定位渲染、性能、资源使用问题。

核心价值：做 AI 和 GPU 原始日志之间的翻译层，把超大文本日志压缩为可搜索、可导出、可追溯的结构化信息。

当前重点不是做完整 `.trace` 回放器，也不在工具内做智能 Bug 诊断；GST 只把每一帧统计清楚：耗时、API 分类、重点接口、program/shader、drawcall 和原始行证据。Bug 解释交给 AI。

## 当前技术栈

- 主目录：`gst/`
- 后端：Go 1.22，标准库 HTTP 服务，核心逻辑零外部依赖。
- 前端：Vue 3 + Vite + TypeScript + Element Plus，目录在 `gst/web/`。
- E2E：Playwright 已接入，默认驱动系统 Google Chrome。
- 构建/打包：`gst/Makefile` 统一构建 Web、`gst-server`、`gst-cli`，支持 Linux amd64/arm64 的 deb/rpm 目标。
- 入口：
  - `gst/cmd/gst-server` — Web/API 服务。
  - `gst/cmd/cli` — 命令行分析工具。
  - `gst/cmd/functionaltest` — 集成验证入口。

## 核心模块

- `internal/core/parser`：日志格式检测与解析；大 rawtrace/hybrid apitrace 走 indexed streaming parser，只保存帧索引和摘要，按需读取帧 API/drawcall 明细。
- `internal/core/glstats`：OpenGL 函数分类注册表和帧级统计；这是分类唯一事实源。
- `internal/core/analyzer`：帧、函数、shader、buffer、draw call、texture、overview、bottleneck、Trace Inspector 等分析。
- `internal/core/bug`：legacy 诊断引擎，新产品主线不再扩展。
- `internal/core/search`：关键字与范围搜索，包含 KeywordIndex。
- `internal/core/exporter`：TXT/CSV/JSON 导出。
- `cmd/gst-server/internal/handlers`：HTTP handlers，持有当前解析日志和索引。

## 主要用户工作流

1. 用户上传或指定日志文件。
2. GST 自动检测格式并解析为 `ParsedLog`。
3. 用户通过 Web UI、CLI 或 API 查看帧、分页原始日志、OpenGL 分类统计、Trace Inspector、搜索和导出。
4. AI agent 可读取结构化摘要，用更少上下文理解日志问题。

## 对外接口

- CLI：`gst/docs/cli.md`，包含传统解析/搜索/导出、legacy 诊断，以及 `-ai-summary`、`-frame-stats <N>`。
- HTTP：`gst/docs/http-api.md`，以 `gst/cmd/gst-server/main.go` 的路由为准，包含 v1 Web API、Trace Inspector、export、v2 AI API 和 shutdown/health。
- 安装打包：`gst/docs/install.md`，记录 Go 1.22、Web build、deb/rpm 构建和安装路径。

## v2 方向

- v1/v2 输入边界统一为 `-trace=1` 生成的完整 apitrace 日志。
- profile 聚合块视为完整日志中某帧的统计摘要，归属前一个 swap 结束的帧。
- 新 API 先使用 current case：`/api/v2/cases/current/overview`、`/frames/:id/stats`、`/ai-summary`。
- AI 使用小而稳定的 JSON 摘要；原始 API 明细必须分页获取，整帧证据通过下载接口流式读取。

## 大日志策略

- 对 `tab22_api.log` 这类 700MB+ 日志，不能把每条 API 常驻内存。
- `FrameInfo` 保存 `StartOffset/EndOffset`、真实 API/draw count、耗时摘要和 frame cost。
- `/api/log/frames/:id/apis` 和 `/drawcalls` 根据 offset 范围按页二次扫描该帧。
- `/api/log/frames/:id/raw-lines` 根据完整帧 offset 分页返回文本预览，默认去除 apitrace 行号前缀。
- `/api/log/frames/:id/download` 流式下载指定帧完整日志，不在内存构造大字符串。
- frame cost 行是 hybrid apitrace 的帧来源，启动段/尾段不发布为帧，避免帧数多两帧。
