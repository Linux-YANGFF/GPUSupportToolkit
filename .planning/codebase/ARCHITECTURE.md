---
last_mapped_commit: 0e48e4f
analysis_date: 2026-05-09
focus: arch
---

# 系统架构

**分析日期：** 2026-05-09

## 系统概览

```text
┌─────────────────────────────────────────────────────────────────────────────┐
│                        表现层 (Presentation Layer)                           │
│  ┌──────────────┐  ┌──────────────────────────────────────────────────────┐ │
│  │   CLI 工具   │  │              Web UI (Vite + Vue 3 SPA)               │ │
│  │ cmd/cli/     │  │  web/src/ (Composition API + provide/inject)         │ │
│  └──────┬───────┘  └──────────────────────┬───────────────────────────────┘ │
└─────────┼─────────────────────────────────┼─────────────────────────────────┘
          │                                 │
          │         HTTP / API              │
          ▼                                 ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                        HTTP Handler 层                                      │
│              cmd/gst-server/internal/handlers/                               │
│         ┌──────────────────────────────────────────┐                        │
│         │  Handler (单例 + sync.RWMutex)            │                        │
│         │  - ParseLog, GetFrames, GetFrameDetail   │                        │
│         │  - Search, AnalyzeTop, AnalyzeShaders    │                        │
│         │  - AnalyzeFuncs, Export, HandleDiagnose  │                        │
│         └──────────────────────────────────────────┘                        │
└──────────────────────────────┬──────────────────────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                        核心引擎层 (Core Engine Layer)                        │
│                    internal/core/                                            │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────────┐  │
│  │  解析器   │  │  搜索器   │  │  分析器   │  │  导出器   │  │  Bug 诊断器   │  │
│  │ parser/  │  │ search/  │  │analyzer/ │  │exporter/ │  │   bug/       │  │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘  └────┬─────┘  └──────┬───────┘  │
│       │             │             │             │               │          │
│       └─────────────┴─────────────┴─────────────┴───────────────┘          │
│                               │                                            │
│                               ▼                                            │
│                    ┌─────────────────────┐                                 │
│                    │   internal/core/    │                                 │
│                    │      types.go       │                                 │
│                    │    (共享领域模型)    │                                 │
│                    └─────────────────────┘                                 │
└──────────────────────────────┬──────────────────────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                        平台层 (Platform Layer)                               │
│                    internal/platform/                                        │
│         ┌─────────────────┬─────────────────┬───────────────┐               │
│         │   file_reader   │   os_detector   │    logger     │               │
│         │  (StreamReader) │  (OS 检测)      │  (slog 初始化) │               │
│         └─────────────────┴─────────────────┴───────────────┘               │
└─────────────────────────────────────────────────────────────────────────────┘
```

## 组件职责

| 组件 | 职责 | 文件 |
|-----------|----------------|------|
| CLI | 命令行入口，支持解析/搜索/分析/导出/诊断 | `cmd/cli/main.go` |
| GST Server | HTTP 服务入口，托管 Web UI 和 REST API | `cmd/gst-server/main.go` |
| 功能测试 | 集成测试入口，验证核心链路 | `cmd/functionaltest/main.go` |
| Handler | HTTP 请求处理、状态管理、路径校验 | `cmd/gst-server/internal/handlers/handlers.go` |
| Parser | 日志解析（apitrace / profile / rawtrace） | `internal/core/parser/*.go` |
| Search | 关键字检索、时间段检索、倒排索引 | `internal/core/search/*.go` |
| Analyzer | 帧分析、函数统计、Shader 统计、Buffer 分析 | `internal/core/analyzer/*.go` |
| Exporter | TXT/CSV/JSON 多格式导出 | `internal/core/exporter/*.go` |
| Bug | Bug 诊断引擎（7 个 diagnoser + registry + GL 状态机） | `internal/core/bug/*.go` |
| Platform | 文件流式读取、OS 检测、slog 日志初始化 | `internal/platform/*.go` |

## 设计模式概览

**整体：** 分层架构 + 管道过滤器（Parser -> Analyzer/Search/Exporter/Bug）

**关键特征：**
- Handler 是单例模式，通过 `sync.RWMutex` 保护共享的解析状态（`current *core.ParsedLog`、`lines []string`、`index *search.KeywordIndex`）
- Bug 诊断引擎采用插件化注册表模式（`bug.Registry`），支持按名称注册和检索 diagnoser
- Parser 采用策略模式，通过 `DetectKindFromReader` 自动选择 `APIParser`、`ProfileParser`、`RawTraceParser`
- Web UI 采用 Vue 3 Composition API + provide/inject 共享全局状态（`useLogAnalysis` composable）

## 分层说明

**表现层：**
- 目的：用户交互入口
- 位置：`cmd/cli/`、`web/`
- 包含：CLI flag 解析、Vue SPA 页面、组件、composables
- 依赖：核心引擎（CLI 直接调用）、HTTP Handler（Web UI 通过 REST API 调用）
- 使用者：终端用户

**HTTP Handler 层：**
- 目的：接收 HTTP 请求，管理会话状态，调用核心引擎
- 位置：`cmd/gst-server/internal/handlers/`
- 包含：`Handler` 结构体、路由处理函数、请求/响应 DTO
- 依赖：核心引擎（parser/analyzer/search/exporter/bug）
- 使用者：Web UI（`logs.html` 中的 Vue 应用）

**核心引擎层：**
- 目的：业务逻辑核心
- 位置：`internal/core/`
- 包含：解析器、检索器、分析器、导出器、诊断器
- 依赖：平台层
- 使用者：Handler 层、CLI

**平台层：**
- 目的：基础设施能力
- 位置：`internal/platform/`
- 包含：大文件流式读取、OS 检测、slog 初始化
- 依赖：标准库 only
- 使用者：核心引擎层

## 数据流

### Web UI 主请求路径

1. **上传/解析** — 用户上传文件或输入路径，前端调用 `POST /api/log/parse`
   - Handler 调用 `parser.CreateParserAuto` 自动检测日志类型并解析
   - 解析结果存入 `Handler.current`，构建 `lines` 和 `KeywordIndex`
   - 返回 `ParseResult`（不含完整帧列表，避免大数据传输）
2. **分页获取帧列表** — 前端调用 `GET /api/log/frames?page=N&page_size=50`
   - Handler 从 `current.Frames` 切片分页，返回轻量 `FrameSummary` 数组
3. **查看帧详情** — 前端调用 `GET /api/log/frames/:id`
   - Handler 按 `FrameNum` 匹配，返回完整 `FrameInfo`（含 `APICalls`、`APISummary`、`Shaders`）
4. **搜索** — 前端调用 `GET /api/log/search?q=keyword`
   - Handler 打开原始日志文件流式扫描，按关键字 AND 匹配，分页返回

### 诊断流程

1. **触发诊断** — 前端点击"开始诊断"，调用 `POST /api/diagnose`
   - 请求体：`{ path: "日志文件路径" }`
2. **Handler 处理** — `HandleDiagnose` 打开文件、自动检测解析器、解析日志
3. **Registry 执行** — 创建 `bug.NewDefaultRegistry()`，调用 `registry.RunAll(parsed)`
4. **7 个 Diagnoser 顺序分析**（每个返回 `[]core.Finding`）：
   - `NullPointerDetector` — 空指针检测（依赖 `GLStateTracker`）
   - `ResourceLeakDetector` — 资源泄漏检测（依赖 `ContextManager`）
   - `ShaderErrorDetector` — Shader 编译/链接错误检测
   - `AntiPatternDetector` — API 反模式检测（依赖 `GLStateTracker` + `ContextManager`）
   - `PerfAnomalyDetector` — 性能异常检测（帧耗时尖峰、DrawCall 突变、慢 API）
   - `ThreadSafetyDetector` — 线程安全诊断（多 TID 访问同一 Context）
   - `DriverErrorDetector` — 驱动层错误关联分析（`__glSetError`）
5. **生成报告** — `bug.GenerateReport` 汇总为 `DiagnosisReport`（JSON），前端渲染

### CLI 批处理流程

1. **解析** — `go run ./cmd/cli -parse <file>`
   - 调用 `parser.CreateParserAuto`，输出帧数、FPS、首末帧耗时
2. **搜索** — `-search <keyword>` 调用 `search.NewKeywordSearch().Search`
3. **分析** — `-top N` 调用 `analyzer.NewFrameAnalyzer`，`-funcs` 调用 `NewFuncAnalyzer`，`-shader` 调用 `NewShaderAnalyzer`
4. **导出** — `-export <format>` 调用 `exporter` 各实现
5. **诊断** — `-diagnose -parse <file>` 调用 `bug.NewDefaultRegistry().RunAll`，输出 Markdown 报告

## 关键抽象

**Parser 接口：**
- 目的：统一日志解析入口
- 示例：`internal/core/parser/parser.go`
- 模式：策略模式 + 工厂方法（`CreateParserAuto` 根据内容自动选择实现）

**Diagnoser 接口：**
- 目的：可扩展的 Bug 诊断插件
- 示例：`internal/core/bug/diagnoser.go`
- 模式：插件注册表（`Registry` 维护 `[]Diagnoser` 和 `map[string]Diagnoser`）

**GLStateTracker：**
- 目的：跨帧追踪 OpenGL 状态（VBO、EBO、VAO、Program、Texture、FBO、VertexAttrib）
- 示例：`internal/core/bug/state_tracker.go`
- 模式：状态机，按 GCAddr（上下文地址）隔离状态，支持 `sync.RWMutex` 并发安全

**ContextManager：**
- 目的：追踪 GL Context 生命周期和资源归属，检测跨 Context 资源使用
- 示例：`internal/core/bug/context_manager.go`
- 模式：资源映射表（`map[string]*ContextInfo`），支持 shareList 分析

## 入口点

**CLI：**
- 位置：`cmd/cli/main.go`
- 触发：命令行参数（`-parse`, `-search`, `-top`, `-funcs`, `-shader`, `-export`, `-diagnose`）
- 职责：批量处理日志文件，输出到 stdout 或文件

**GST Server：**
- 位置：`cmd/gst-server/main.go`
- 触发：`go run ./cmd/gst-server`，默认监听 `:8080`
- 职责：提供 REST API，托管静态 Web 文件，支持 graceful shutdown

**功能测试：**
- 位置：`cmd/functionaltest/main.go`
- 触发：`go run ./cmd/functionaltest`
- 职责：端到端验证 Parser -> Analyzer -> Search -> Exporter 链路

## API 端点

| Method | Path | Handler | 说明 |
|--------|------|---------|------|
| POST | `/api/log/parse` | `ParseLog` | 解析日志文件（支持 multipart 上传或 JSON 路径） |
| GET | `/api/log/frames` | `GetFrames` | 分页获取帧列表（`page`, `page_size`） |
| GET | `/api/log/frames/:id` | `GetFrameDetail` | 获取单帧完整详情 |
| GET | `/api/log/frames/:id/funcs` | `GetFrameFuncs` | 获取单帧函数统计（排序后） |
| GET | `/api/log/search` | `Search` | 关键字搜索（`q` 参数，AND 匹配） |
| GET | `/api/log/analyze/top` | `AnalyzeTop` | Top N 慢帧分析（`n` 参数） |
| GET | `/api/log/analyze/shaders` | `AnalyzeShaders` | 所有帧 Shader 列表（源码截断 2000 字符） |
| GET | `/api/log/analyze/funcs` | `AnalyzeFuncs` | 全量函数统计 |
| POST | `/api/log/export` | `Export` | 导出数据（支持 `frames`/`funcs`/`shader`/`search`/`top`/`longest` + `json`/`csv`/`txt`） |
| POST | `/api/diagnose` | `HandleDiagnose` | Bug 诊断（JSON 路径入参，返回 `DiagnosisReport`） |
| POST | `/api/shutdown` | `handleShutdown` | 优雅关闭服务 |
| GET | `/health` | `Health` | 健康检查 |
| GET | `/` | `serveStatic` | 静态文件服务（SPA fallback 到 `index.html`） |

## 架构约束

- **并发：** Go 标准并发模型（goroutine + channel）。Handler 使用 `sync.RWMutex` 保护共享状态，GLStateTracker 使用 `sync.RWMutex` 保护 contexts map。
- **全局状态：** `platform.Logger` 是包级全局 slog 实例；`cmd/gst-server/main.go` 中的 `srv *http.Server` 是全局变量。
- **循环依赖：** 未发现。`internal/core` 为底层，不依赖上层；`bug/` 依赖 `core/`；`parser/` 依赖 `core/`；`analyzer/` 依赖 `core/`；`exporter/` 依赖 `core/`；`search/` 依赖 `core/`。
- **内存：** 日志文件通过 `bufio.Scanner` 流式读取，buffer 上限 10MB（`DefaultBufferSize`）；大文件索引通过 `StreamReader`（64MB buffer）处理。
- **路径安全：** `validateLogPath` 限制日志文件必须在 `GST_LOG_DIR`（默认 `..`）目录下，防止目录遍历攻击。

## 反模式

### Handler 单例持有大对象

**现象：** `Handler` 单例通过 `sync.RWMutex` 持有完整的 `*core.ParsedLog`，大日志解析后内存占用高，且所有请求串行竞争锁。
**问题：** 高并发场景下锁竞争严重，`RLock` 虽然允许多读，但解析（`ParseLog`）需要 `Lock`，会阻塞所有读请求。
**改进：** 考虑将解析结果缓存到磁盘（如索引文件 `LogIndex`）或使用无锁的 immutable 数据结构；解析时采用 copy-on-write 模式。

### 前端 provide/inject 类型安全缺失

**现象：** `web/src/composables/useLogAnalysis.ts` 通过 `provide('ctx', ctx)` 注入全局状态，组件中使用 `inject<any>('ctx')`。
**问题：** `any` 类型导致编译期无法检查属性访问错误，重构时容易遗漏。
**改进：** 定义 `InjectionKey<ReturnType<typeof useLogAnalysis>>` 并使用 `provide(key, ctx)` / `inject(key)`，启用严格类型推断。

## 错误处理

**策略：** 分层处理，底层返回 `error`，上层根据场景决定 HTTP 状态码或 CLI 退出码。

**模式：**
- Parser: 解析失败返回 `error`，Handler 返回 `500 Internal Server Error`
- Handler: 输入校验失败返回 `400 Bad Request`；路径越界返回 `403/400`
- CLI: 错误输出到 `stderr`，调用 `os.Exit(1)`
- Bug Diagnoser: 单个 diagnoser panic 不应影响其他 diagnoser（当前未实现 recover，建议添加）

## 横切关注点

**日志：** `internal/platform/logger.go` 初始化 slog，支持 `debug/info/warn/error` 级别，通过 `platform.InitLogger` 配置。

**校验：** Handler 层通过 `validateLogPath` 校验文件路径；前端通过表单禁用和校验提示。

**认证：** 未实现。当前为本地工具，无认证层。

---

*Architecture analysis: 2026-05-09*
