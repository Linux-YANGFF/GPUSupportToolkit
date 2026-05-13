# GST 项目速读

## 项目定位

GST (GPU Support Toolkit) 是 GPU 日志分析工具，把 GB 级 apitrace / rawtrace / profile 日志解析成结构化数据，服务于 AI agent、FAE 和研发定位渲染、性能、崩溃、资源使用问题。

核心价值：做 AI 和 GPU 原始日志之间的翻译层，把超大文本日志压缩为可搜索、可诊断、可导出的结构化信息。

## 当前技术栈

- 主目录：`gst/`
- 后端：Go 1.22，标准库 HTTP 服务，核心逻辑零外部依赖。
- 前端：Vue 3 + Vite + TypeScript，目录在 `gst/web/`。
- 入口：
  - `gst/cmd/gst-server` — Web/API 服务。
  - `gst/cmd/cli` — 命令行分析工具。
  - `gst/cmd/functionaltest` — 集成验证入口。

## 核心模块

- `internal/core/parser`：日志格式检测与流式解析，支持 API trace、profile、rawtrace。
- `internal/core/analyzer`：帧、函数、shader、buffer、draw call、texture、overview、bottleneck、Trace Inspector 等分析。
- `internal/core/bug`：诊断引擎，基于注册表运行多个检测器。
- `internal/core/search`：关键字与范围搜索，包含 KeywordIndex。
- `internal/core/exporter`：TXT/CSV/JSON 导出。
- `cmd/gst-server/internal/handlers`：HTTP handlers，持有当前解析日志和索引。

## 主要用户工作流

1. 用户上传或指定日志文件。
2. GST 自动检测格式并解析为 `ParsedLog`。
3. 用户通过 Web UI、CLI 或 API 查看帧、搜索、统计、诊断、Trace Inspector 和导出。
4. AI agent 可读取结构化摘要，用更少上下文理解日志问题。
