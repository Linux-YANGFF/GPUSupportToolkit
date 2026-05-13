---
last_mapped_commit: 0e48e4f
analysis_date: 2026-05-09
focus: arch
---

# 目录结构

**分析日期：** 2026-05-09

## 目录布局

```
GPUSupportToolkit/
├── CLAUDE.md                         # AI agent 工作指南
├── README.md                         # 项目概览（中文）
├── docs/                             # 顶层设计/产品文档
│   ├── DESIGN.md                     # 技术设计（架构、模块、数据流）
│   └── PRODUCT.md                    # 产品需求（用户故事、功能、规格）
├── exmple_log/                       # 测试用示例日志文件
│   ├── 1frame_demo_api.txt           # 单帧聚合 API trace
│   └── 1frame_profile_demo.txt       # 单帧 profile 日志
└── gst/                              # 主 Go 项目根目录
    ├── go.mod                        # Go 模块：gst (Go 1.22)
    ├── go.sum                        # 依赖校验和
    ├── Makefile                      # 构建、测试、打包目标
    ├── VERSION                       # 1.0.0
    ├── README.md                     # 英文项目 readme
    ├── .gitignore                    # bin/、IDE 文件、OS 产物
    ├── bin/                          # 构建输出（不提交）
    ├── assets/                       # 静态资源（图标、图片）
    ├── cmd/                          # 可执行入口点
    │   ├── cli/                      # gst-cli：命令行工具
    │   │   ├── main.go               # 基于 flag 的 CLI，451 行
    │   │   └── main_test.go          # CLI 集成测试
    │   ├── gst-server/               # gst-server：Web 服务
    │   │   ├── main.go               # HTTP 服务启动，191 行
    │   │   └── internal/
    │   │       └── handlers/
    │   │           ├── handlers.go   # REST API handlers，810 行
    │   │           └── handlers_test.go # Handler 单元测试
    │   └── functionaltest/           # 进程内集成测试
    │       └── main.go               # 测试 parser->analyzer->exporter 链路
    ├── internal/                     # 私有应用代码
    │   ├── core/                     # 核心业务逻辑
    │   │   ├── types.go              # 领域模型：20+ 类型，140 行
    │   │   ├── core_test.go          # 领域模型单元测试
    │   │   ├── parser/               # 日志解析层
    │   │   │   ├── parser.go         # Parser 接口 + 格式检测，242 行
    │   │   │   ├── api_parser.go     # 聚合 API trace 解析器，296 行
    │   │   │   ├── profile_parser.go # Profile 解析器（委托给 APIParser），26 行
    │   │   │   ├── raw_trace_parser.go # 逐调用原始格式解析器，300 行
    │   │   │   └── parser_test.go    # 单元测试：检测、解析，434 行
    │   │   ├── search/               # 搜索层
    │   │   │   ├── keyword_search.go # 多关键字 AND 搜索（正则 + 简单），138 行
    │   │   │   ├── time_range_search.go # 时间范围 + 帧范围搜索，50 行
    │   │   │   ├── index.go          # 倒排关键字索引，55 行
    │   │   │   └── search_test.go    # 单元测试：关键字、分页、时间范围，189 行
    │   │   ├── analyzer/             # 分析层
    │   │   │   ├── frame_analyzer.go # Top-N 帧排名、摘要统计，67 行
    │   │   │   ├── func_analyzer.go  # 函数调用次数 + 耗时聚合，101 行
    │   │   │   ├── shader_analyzer.go # Shader 编译/创建/源码统计，94 行
    │   │   │   ├── buffer_analyzer.go # Buffer 生命周期分析，389 行
    │   │   │   └── analyzer_test.go   # 单元测试：所有分析器类型，724 行
    │   │   ├── exporter/             # 导出层
    │   │   │   ├── exporter.go       # TXT/CSV/JSON 导出器 + 便捷函数，360 行
    │   │   │   └── exporter_test.go  # 单元测试：所有格式和数据类型，235 行
    │   │   └── bug/                  # Bug 诊断引擎
    │   │       ├── diagnoser.go      # Diagnoser 接口，8 行
    │   │       ├── registry.go       # 插件注册表，36 行
    │   │       ├── registry_test.go  # 注册表单元测试，256 行
    │   │       ├── state_tracker.go  # GL 状态机（VBO/EBO/VAO/Program/Texture/FBO），400 行
    │   │       ├── state_tracker_test.go # GL 状态追踪测试
    │   │       ├── context_manager.go # GL 上下文生命周期追踪，205 行
    │   │       ├── context_manager_test.go # 上下文管理器测试
    │   │       ├── report.go         # 报告生成 + 默认注册表，167 行
    │   │       ├── report_test.go    # 报告生成测试
    │   │       ├── null_pointer_detector.go # 空指针检测，151 行
    │   │       ├── null_pointer_detector_test.go
    │   │       ├── resource_leak_detector.go # 资源泄漏检测，300 行
    │   │       ├── resource_leak_detector_test.go
    │   │       ├── shader_error_detector.go # Shader 编译/链接错误检测，147 行
    │   │       ├── shader_error_detector_test.go
    │   │       ├── antipattern_detector.go # API 反模式检测，296 行
    │   │       ├── antipattern_detector_test.go
    │   │       ├── perf_anomaly_detector.go # 性能异常检测，290 行
    │   │       ├── perf_anomaly_detector_test.go
    │   │       ├── driver_error_detector.go # 驱动错误关联，144 行
    │   │       ├── driver_error_detector_test.go
    │   │       ├── thread_safety_detector.go # 线程安全诊断，170 行
    │   │       ├── thread_safety_detector_test.go
    │   │       └── testdata/         # Bug diagnoser 测试固件
    │   └── platform/                 # 平台工具
    │       ├── file_reader.go        # StreamReader、LogIndex（二进制格式），276 行
    │       ├── file_reader_test.go   # 文件读取测试
    │       ├── os_detector.go        # OS 检测、桌面环境检查，123 行
    │       ├── logger.go             # slog 初始化，36 行
    │       └── logger_test.go        # Logger 测试
    ├── web/                          # Vue 3 Web UI（Vite + TypeScript）
    │   ├── index.html                # 着陆页，62 行
    │   ├── logs.html                 # 日志分析 SPA 入口，521 行
    │   ├── package.json              # npm 依赖：vue@3.5.13、vite@6.3.5
    │   ├── tsconfig.json             # TypeScript 配置（严格模式、ES2021、bundler 解析）
    │   ├── vite.config.ts            # Vite 配置（多页面：main + logs）
    │   ├── src/
    │   │   ├── main.ts               # Vue 应用启动
    │   │   ├── App.vue               # 根组件（script setup）
    │   │   ├── types.ts              # 共享 TypeScript 接口
    │   │   ├── css/
    │   │   │   └── app.css           # 应用样式表
    │   │   ├── composables/
    │   │   │   └── useLogAnalysis.ts # 全局状态 composable
    │   │   └── components/
    │   │       ├── AnalysisTabs.vue  # 标签导航组件
    │   │       ├── FileUpload.vue    # 文件输入 + 解析控制
    │   │       ├── FrameList.vue     # 分页帧表格 + 模态框
    │   │       ├── FunctionStats.vue # Top-N + Shader 统计
    │   │       ├── SearchPanel.vue   # 关键字搜索结果
    │   │       └── DiagnosisPanel.vue # Bug 诊断 UI
    │   └── dist/                     # 构建输出（不提交）
    ├── packaging/                    # OS 打包配置
    │   ├── deb/
    │   │   ├── DEBIAN/
    │   │   │   ├── control           # 包元数据
    │   │   │   ├── postinst          # 安装后脚本
    │   │   │   └── prerm             # 卸载前脚本
    │   │   └── usr/
    │   │       └── share/
    │   │           └── applications/
    │   │               └── gst.desktop # 桌面入口
    │   └── rpm/
    │       └── gst.spec              # RPM spec 文件
    ├── docs/                         # 用户文档
    │   ├── cli.md                    # CLI 使用指南
    │   └── install.md               # 安装指南（deb/rpm）
    ├── .claude/                      # AI 助手配置
    └── .planning/                    # GSD 规划目录
        └── codebase/
            ├── ARCHITECTURE.md       # 本文档
            └── STRUCTURE.md          # 目录结构文档
```

## 目录用途

**`gst/cmd/`：**
- 目的：应用入口点。每个子目录编译为一个独立的二进制文件。
- 包含：`main.go` 文件，含 CLI flag 解析 / HTTP 服务启动
- 关键文件：`cli/main.go`（gst-cli）、`gst-server/main.go`（gst-server）、`functionaltest/main.go`

**`gst/cmd/gst-server/internal/handlers/`：**
- 目的：Web 服务的 REST API handlers。唯一对 server 命令私有的包。
- 包含：`handlers.go` — `Handler` 结构体及 REST 端点方法、请求/响应 DTO
- 关键文件：`handlers.go`、`handlers_test.go`

**`gst/internal/core/`：**
- 目的：核心业务逻辑 — 应用的心脏。所有入口点共享。
- 包含：`types.go`（领域模型），以及 `parser/`、`search/`、`analyzer/`、`exporter/`、`bug/` 子包
- 关键文件：`types.go`（共享领域类型，140 行）

**`gst/internal/core/parser/`：**
- 目的：流式日志到领域模型的解析，支持格式自动检测。
- 包含：Parser 接口、3 个实现、格式检测逻辑、工厂函数
- 关键文件：`parser.go`（接口 + 检测）、`api_parser.go`（主聚合解析器，296 行）

**`gst/internal/core/search/`：**
- 目的：对解析和原始日志数据的文本搜索和时间范围过滤。
- 包含：关键字搜索（正则 + 简单）、时间范围搜索、倒排索引
- 关键文件：`keyword_search.go`（138 行）

**`gst/internal/core/analyzer/`：**
- 目的：对解析后的帧数据进行统计分析。
- 包含：帧/函数/shader/buffer 分析
- 关键文件：`analyzer_test.go`（724 行，全面的测试覆盖）、`buffer_analyzer.go`（389 行）

**`gst/internal/core/exporter/`：**
- 目的：将分析数据序列化为 TXT/CSV/JSON 格式。
- 包含：Exporter 接口、3 个格式实现、领域类型专用便捷函数
- 关键文件：`exporter.go`（360 行）

**`gst/internal/core/bug/`：**
- 目的：Bug 诊断引擎 — 7 个 diagnoser + GL 状态机 + 上下文管理器 + 报告生成。
- 包含：11 个 Go 源文件 + 9 个测试文件 + `testdata/` 固件
- 关键文件：`registry.go`（插件注册表）、`state_tracker.go`（GL 状态机，400 行）、`report.go`（默认注册表 + 报告生成）

**`gst/internal/platform/`：**
- 目的：OS 级工具 — 文件 I/O、OS 检测、slog 初始化。
- 包含：`file_reader.go`（流式读取、索引）、`os_detector.go`（Linux OS 检测）、`logger.go`（slog 设置）
- 关键文件：`file_reader.go`（276 行）

**`gst/web/`：**
- 目的：浏览器 UI — 由 gst-server 提供的 Vite + Vue 3 SPA。
- 包含：2 个 HTML 入口点、Vue SFC 组件、composables、TypeScript 类型、CSS
- 关键文件：`logs.html`（SPA 外壳）、`src/App.vue`（根组件）、`src/composables/useLogAnalysis.ts`（全局状态）

**`gst/packaging/`：**
- 目的：Debian/RPM 打包配置，用于 Linux 分发。
- 包含：deb control 文件、postinst/prerm 脚本、桌面入口、RPM spec
- 关键文件：`deb/DEBIAN/control`

**`gst/docs/`：**
- 目的：终端用户文档（中文）。
- 包含：CLI 使用指南、安装指南
- 关键文件：`cli.md`

**`exmple_log/`：**
- 目的：用于开发和测试的示例日志文件（顶层，与 `gst/` 同级）。
- 包含：`1frame_demo_api.txt`、`1frame_profile_demo.txt`
- 关键文件：均为单帧示例日志

**`docs/`：**
- 目的：项目级技术设计和产品需求（顶层）。
- 包含：`DESIGN.md`、`PRODUCT.md`
- 关键文件：`DESIGN.md`（316 行 — 原始架构规格）

## 关键文件位置

**入口点：**
- `gst/cmd/cli/main.go`：gst-cli — 命令行日志分析工具（451 行）
- `gst/cmd/gst-server/main.go`：gst-server — HTTP Web 服务（191 行）
- `gst/cmd/functionaltest/main.go`：进程内集成测试框架（113 行）

**配置：**
- `gst/go.mod`：Go 模块定义（`module gst`，Go 1.22）
- `gst/Makefile`：构建系统 — 目标：`dev`、`build-*`、`test`、`package`（deb/rpm）
- `gst/.gitignore`：忽略 `bin/`、IDE 文件、`.env`
- `gst/VERSION`：版本字符串（1.0.0）
- `gst/web/package.json`：npm 依赖（vue@3.5.13、vite@6.3.5、typescript@5.8.3）
- `gst/web/vite.config.ts`：Vite 构建配置（多页面输入：main + logs）
- `gst/web/tsconfig.json`：TypeScript 编译器选项（严格模式、ES2021、bundler 解析）

**核心逻辑（领域模型）：**
- `gst/internal/core/types.go`：所有共享数据类型 — `APILogEntry`、`FrameInfo`、`ParsedLog`、`SearchResult`、`FuncStats`、`ShaderInfo`、`APISummary`、`ShaderCompileInfo`、`BufferInfo`、`BufferSummary`、`FrameSummary`、`FuncSummary`、`ShaderSummary`、`Finding`、`DiagnosisReport`、`DiagnosisSummary`、`Severity`（140 行）

**核心逻辑（解析器）：**
- `gst/internal/core/parser/parser.go`：`Parser` 接口、`LogKind` 枚举、`DetectKind`/`DetectKindFromReader`、`CreateParser` 工厂（242 行）
- `gst/internal/core/parser/api_parser.go`：`APIParser` — 聚合格式解析器，含帧边界检测（296 行）
- `gst/internal/core/parser/raw_trace_parser.go`：`RawTraceParser` — 逐调用原始格式解析器，含 gc/tid 提取（300 行）
- `gst/internal/core/parser/profile_parser.go`：`ProfileParser` — 对 `APIParser` 的薄委托（26 行）

**核心逻辑（搜索）：**
- `gst/internal/core/search/keyword_search.go`：`KeywordSearch` + `KeywordSearchSimple`（138 行）
- `gst/internal/core/search/time_range_search.go`：`TimeRangeSearch`（50 行）
- `gst/internal/core/search/index.go`：`KeywordIndex` — 倒排索引（55 行）

**核心逻辑（分析）：**
- `gst/internal/core/analyzer/frame_analyzer.go`：Top-N 帧、帧摘要（67 行）
- `gst/internal/core/analyzer/func_analyzer.go`：函数调用统计，含前缀过滤（101 行）
- `gst/internal/core/analyzer/shader_analyzer.go`：每帧 + 全局 Shader 编译统计（94 行）
- `gst/internal/core/analyzer/buffer_analyzer.go`：Buffer 生命周期 + OpenGL 枚举映射（389 行）

**核心逻辑（Bug 诊断）：**
- `gst/internal/core/bug/registry.go`：`Registry` — 插件注册表，含 `Register`/`RegisterNamed`/`RunAll`/`GetDiagnoser`（36 行）
- `gst/internal/core/bug/diagnoser.go`：`Diagnoser` 接口 — `Diagnose(log *core.ParsedLog) []core.Finding`（8 行）
- `gst/internal/core/bug/state_tracker.go`：`GLStateTracker` — 按 GCAddr 追踪 VBO、EBO、VAO、Program、Texture、FBO、VertexAttrib（400 行）
- `gst/internal/core/bug/context_manager.go`：`ContextManager` — 追踪 GL 上下文资源、shareList、跨上下文使用（205 行）
- `gst/internal/core/bug/report.go`：`GenerateReport`、`GenerateMarkdownReport`、`NewDefaultRegistry`（167 行）
- `gst/internal/core/bug/null_pointer_detector.go`：检测 GL 调用中的 nil 指针参数（151 行）
- `gst/internal/core/bug/resource_leak_detector.go`：检测泄漏的 buffer/texture/shader/program/framebuffer（300 行）
- `gst/internal/core/bug/shader_error_detector.go`：检测缺少编译/链接状态检查、重复编译、shader 调用附近的 GL 错误（147 行）
- `gst/internal/core/bug/antipattern_detector.go`：检测冗余状态变更、无效绑定、不必要的上下文切换、null program 上的 uniform、无 program 的 draw（296 行）
- `gst/internal/core/bug/perf_anomaly_detector.go`：使用鲁棒统计检测帧耗时尖峰、DrawCall 突变、慢 API 调用（290 行）
- `gst/internal/core/bug/driver_error_detector.go`：将 `__glSetError` 与触发调用和 glGetError 使用关联（144 行）
- `gst/internal/core/bug/thread_safety_detector.go`：检测多线程未经正确解绑访问同一 GL 上下文（170 行）

**核心逻辑（导出）：**
- `gst/internal/core/exporter/exporter.go`：`TXTExporter`、`CSVExporter`、`JSONExporter`、便捷函数（360 行）

**Server Handlers：**
- `gst/cmd/gst-server/internal/handlers/handlers.go`：REST API handlers、请求/响应 DTO、报告聚合（810 行）

**平台层：**
- `gst/internal/platform/file_reader.go`：`StreamReader`（缓冲流式读取）、`LogIndex`（二进制序列化）、模式搜索（276 行）
- `gst/internal/platform/os_detector.go`：`DetectOS()`、`GetEnvInfo()`、`CheckDesktopEnvironment()`、`GetOSVersion()`、`IsKylinV10()`（123 行）
- `gst/internal/platform/logger.go`：`InitLogger()` — 可配置级别的 slog text handler（36 行）

**Web UI：**
- `gst/web/index.html`：带功能卡片的着陆页（62 行）
- `gst/web/logs.html`：内联 Vue 模板的日志分析 SPA 外壳（521 行）
- `gst/web/src/main.ts`：使用 `createApp` 的 Vue 应用启动（5 行）
- `gst/web/src/App.vue`：根组件 — 文件上传、统计、标签、导出（163 行）
- `gst/web/src/types.ts`：TypeScript 接口 — `FrameData`、`ParseResult`、`SearchResultItem`、`FuncStat`、`ShaderStat`、`TabItem`（51 行）
- `gst/web/src/composables/useLogAnalysis.ts`：全局状态 composable — 解析、帧、搜索、分析、导出、诊断集成（490 行）
- `gst/web/src/components/AnalysisTabs.vue`：5 个标签导航（帧/搜索/分析/诊断/导出）（58 行）
- `gst/web/src/components/FileUpload.vue`：文件路径输入、浏览、解析、重置、停止按钮（59 行）
- `gst/web/src/components/FrameList.vue`：分页帧表格 + 帧详情模态框（148 行）
- `gst/web/src/components/FunctionStats.vue`：Top-N 帧 + Shader 统计表格（125 行）
- `gst/web/src/components/SearchPanel.vue`：关键字搜索输入 + 结果分页（76 行）
- `gst/web/src/components/DiagnosisPanel.vue`：Bug 诊断 UI — 进度、严重度过滤、发现列表、Markdown 导出（707 行）

**测试：**
- `gst/internal/core/parser/parser_test.go`：解析器检测 + 解析测试（434 行）
- `gst/internal/core/search/search_test.go`：关键字搜索 + 分页 + 时间范围测试（189 行）
- `gst/internal/core/analyzer/analyzer_test.go`：全面分析器测试（724 行）
- `gst/internal/core/exporter/exporter_test.go`：导出格式测试（235 行）
- `gst/internal/core/core_test.go`：领域模型单元测试（342 行）
- `gst/internal/core/bug/registry_test.go`：注册表插件测试（256 行）
- `gst/internal/core/bug/*_test.go`：各 diagnoser 测试（9 个文件）
- `gst/cmd/gst-server/internal/handlers/handlers_test.go`：Handler 单元测试（237 行）
- `gst/cmd/cli/main_test.go`：CLI 集成测试（74 行）
- `gst/internal/platform/file_reader_test.go`：文件读取测试
- `gst/internal/platform/logger_test.go`：Logger 测试
- `gst/cmd/functionaltest/main.go`：集成测试（113 行）

**文档：**
- `docs/DESIGN.md`：原始架构设计文档（316 行）
- `docs/PRODUCT.md`：含用户故事的产品需求（290 行）
- `gst/docs/cli.md`：CLI 使用指南
- `gst/docs/install.md`：通过 deb/rpm 安装

## 命名规范

**文件：**
- Go 源码：`snake_case.go`（如 `file_reader.go`、`raw_trace_parser.go`、`state_tracker.go`）
- 测试文件：`_test.go` 后缀，与源码同位置（如 `parser/parser_test.go`、`bug/registry_test.go`）
- Vue SFC：`PascalCase.vue`（如 `App.vue`、`DiagnosisPanel.vue`、`FrameList.vue`）
- TypeScript：`camelCase.ts`（如 `useLogAnalysis.ts`、`main.ts`、`types.ts`）
- Web 文件：`kebab-case.html/css`（如 `logs.html`、`app.css`）
- 配置/元数据：UPPERCASE 或 PascalCase（`Makefile`、`VERSION`、`README.md`、`CLAUDE.md`）

**目录：**
- Go 包：小写单字或复合词用 `snake_case`（`parser`、`file_reader`）
- 命令包：描述性复合名称（`gst-server`、`functionaltest`）
- 内部包：按领域关注点组织（`core/analyzer/`、`core/search/`、`core/bug/`）
- Web 目录：小写（`web/`、`css/`、`src/`、`components/`、`composables/`）

**Go 类型（来自 `types.go`）：**
- 领域模型：PascalCase（`APILogEntry`、`FrameInfo`、`ParsedLog`、`FuncStats`）
- 嵌套/DTO 类型：PascalCase（`SearchResult`、`BufferInfo`、`DiagnosisReport`）
- 常量：PascalCase（`SeverityCritical`、`KindAPITrace`、`TargetArrayBuffer`）
- 函数：导出的用 PascalCase，私有的用 camelCase（`NewFrameAnalyzer`、`FindTopSlowFrames` vs `isFrameBoundary`、`matchAll`）

**Go 包命名：**
- 包名匹配目录（`parser/` -> `package parser`，`analyzer/` -> `package analyzer`）
- 核心类型：`internal/core/types.go` 中的 `package core`

## 新增代码指南

**新解析器格式（如 Vulkan 日志）：**
- 主代码：`gst/internal/core/parser/vulkan_parser.go`
- 在工厂注册：`gst/internal/core/parser/parser.go` -> 添加 `KindVulkan` 常量 + `CreateParser` case
- 测试：`gst/internal/core/parser/parser_test.go` — 添加 `TestDetectKind` case
- 示例日志：`exmple_log/` 目录

**新分析器（如内存分析）：**
- 主代码：`gst/internal/core/analyzer/memory_analyzer.go`
- 遵循现有模式：结构体含 `*core.ParsedLog` 字段、`New*Analyzer()` 构造器、`Analyze()` 方法、可选 `GetSummary()`
- 在 handler 注册：`gst/cmd/gst-server/internal/handlers/handlers.go` -> 添加 `AnalyzeMemory` 端点方法 + 路由
- 测试：`gst/internal/core/analyzer/analyzer_test.go` — 添加测试函数
- 类型：如需新输出类型，在 `gst/internal/core/types.go` 添加统计类型

**新 Bug Diagnoser：**
- 主代码：`gst/internal/core/bug/my_diagnoser.go`
- 实现 `Diagnoser` 接口：`Diagnose(log *core.ParsedLog) []core.Finding`
- 在默认注册表注册：`gst/internal/core/bug/report.go` -> `NewDefaultRegistry()`
- 测试：`gst/internal/core/bug/my_diagnoser_test.go`
- 遵循模式：如需状态追踪，使用 `GLStateTracker` / `ContextManager`

**新导出格式（如 HTML 报告）：**
- 主代码：`gst/internal/core/exporter/exporter.go` — 添加实现 `Export(w io.Writer) error` 的 `HTMLExporter` 结构体
- 在 CLI 注册：`gst/cmd/cli/main.go` -> `exportResults()` 中添加 `"html"` case
- 在 handler 注册：`gst/cmd/gst-server/internal/handlers/handlers.go` -> `Export()` 中添加 `"html"` case

**新 HTTP 端点：**
- 主代码：`gst/cmd/gst-server/internal/handlers/handlers.go` — 在 `Handler` 上添加方法
- 注册路由：`gst/cmd/gst-server/main.go` -> `mux.HandleFunc("/api/...", h.NewEndpoint)`

**新 CLI 命令：**
- 主代码：`gst/cmd/cli/main.go` — 添加 `flag.String/Int/Bool` 变量 + handler 函数
- 更新帮助：`gst/cmd/cli/main.go` -> `printHelp()` 函数

**新平台工具：**
- 主代码：`gst/internal/platform/new_utility.go`
- 应为框架无关（仅依赖标准库）

**Web UI 变更：**
- 新组件：`gst/web/src/components/MyComponent.vue`
- 在 App 注册：`gst/web/src/App.vue` -> template 中 import 并使用
- 新 composable：`gst/web/src/composables/useMyFeature.ts`
- 样式：`gst/web/src/css/app.css`

## 特殊目录

**`gst/bin/`：**
- 目的：构建输出目录
- 生成：是（由 `make build-*`）
- 提交：否（在 `.gitignore` 中）

**`gst/web/dist/`：**
- 目的：Vite 构建输出
- 生成：是（由 `vite build`）
- 提交：否（在 `.gitignore` 中）

**`gst/web/node_modules/`：**
- 目的：npm 依赖
- 生成：是（由 `npm install`）
- 提交：否（在 `.gitignore` 中）

**`exmple_log/`：**
- 目的：用于开发和测试的示例日志文件
- 生成：否
- 提交：是

**`gst/.claude/`：**
- 目的：Claude AI 助手配置（项目级）
- 生成：部分
- 提交：是

**`gst/.planning/`：**
- 目的：GSD 规划产物（阶段、计划、规格）
- 生成：由 GSD 工作流
- 提交：是

---

*Structure analysis: 2026-05-09*
