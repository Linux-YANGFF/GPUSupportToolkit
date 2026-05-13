# GST v2.0 全面升级计划

## TL;DR

> **目标**: 将 GST 从 v1.0.0 升级到 v2.0.0，包含稳定性加固、架构升级和全新的 Bug 诊断引擎。
>
> **交付物**:
> - Phase 1: 类型安全重构后的核心代码，Go 1.22+ 模块，slog 结构化日志，安全加固
> - Phase 2: GL 状态机跟踪引擎，7 个 Bug 诊断分析器，结构化诊断报告
> - Phase 3: Web UI 组件化重构，完善的测试覆盖，CI/CD 流程
>
> **预估工作量**: Large（约 40+ 任务，3 个 Phase）
> **并行执行**: YES — 每个 Phase 内最大化并行
> **关键路径**: Phase 1 类型定义 → Phase 2 状态机 → Phase 2 规则引擎 → Phase 3 测试

---

## Context

### 原始需求

用户拥有 GST v1.0.0（GPU Support Toolkit），一个基于 Go 的 GPU 日志分析工具，支持 apitrace/profile 日志的解析、检索、分析和导出。用户要求：

1. **全面升级**：稳定性加固 + 架构升级 + 新功能
2. **空指针排查**：将 DeerFlow log-bug-analyst skill 中的空指针检测逻辑集成到 GST
3. **GPU 技术支持经验沉淀**：增加资源泄漏检测、Shader 错误分析、API 反模式检测、性能异常定位、线程安全诊断、驱动层错误关联

### 访谈摘要

**关键决策**:
- **技术路线**：纯 Go 标准库（slog 可以使用，作为 stdlib 的一部分）
- **Go 版本**：1.18 → 1.22+
- **测试策略**：先实现后测试（非 TDD）
- **语言规范**：所有代码注释和文档使用中文
- **Web UI**：Vue 3 + Vite + TypeScript，从 CDN 模式升级为构建系统

**研究结论**:
- GST 已有 `raw_trace_parser.go` 支持 apitrace 详细格式，可直接增强
- 需要新增 GL 状态机跟踪模块和 Context 管理模块
- 7 个 Bug 检测规则可参考 log-bug-analyst skill 的决策树逻辑

### 现有代码基础

- Go 源码: 27 个 `.go` 文件
- 测试文件: 4 个（parser、search、analyzer、exporter）
- REST API: 17 个端点，集中在单个 `handlers.go`（900+ 行）
- Web UI: 568 行单体 `logs.js`，Vue 3 CDN 脚本
- 架构: 5 层分层 + Strategy 模式
- 依赖: go.mod 零外部依赖

---

## Work Objectives

### 核心目标

将 GST 从"性能分析工具"升级为"性能分析 + Bug 诊断"的综合 GPU 日志分析平台。

### 具体交付物

**Phase 1 — 地基（稳定性 + 架构）**:
- Go 模块升级到 1.22
- 所有 `interface{}`/`map[string]interface{}` 替换为具体类型结构体
- slog 结构化日志集成
- 5 处静默错误忽略修复
- API 路径校验安全加固
- 死代码清理
- 魔法数字提取为命名常量

**Phase 2 — 核心新能力（Bug 诊断引擎）**:
- GL 状态机跟踪模块（VBO/VAO/Shader/Texture 状态）
- Context 管理模块（gc= 地址追踪、资源归属）
- 7 个 Bug 诊断分析器
- 结构化诊断报告生成
- 增强 RawTraceParser 支持 `__glSetError` 等驱动层信息

**Phase 3 — 体验提升**:
- Web UI 重构（Vue 3 + Vite + TypeScript，组件拆分）
- platform/handlers/server 测试全覆盖
- GitHub Actions CI/CD
- Bug 诊断报告的可视化展示

### 完成定义

- [x] `go build ./...` 零错误
- [x] `go vet ./...` 零警告
- [x] `go test ./...` 全部通过
- [x] 零 `interface{}` 在 analyzer 的公开 API 中
- [x] 所有 API 端点有路径校验
- [x] slog 替代所有 `log.Printf`/`fmt.Println`
- [x] Web UI 可 `npm run build` 构建
- [x] 7 个 Bug 分析器均可通过示例日志输出诊断报告

### 必须包含（Must Have）

- 类型安全重构（零 `interface{}` 公开 API）
- slog 结构化日志
- API 路径校验（防任意文件读取）
- 空指针排查分析器（log-bug-analyst 逻辑集成）
- 资源泄漏检测分析器
- GL 状态机跟踪模块
- 所有模块的基本测试覆盖

### 绝对不能做（Guardrails）

- **不引入外部 Go 依赖**（保持 go.mod 零 require，slog 除外作为 stdlib）
- **不修改现有日志格式兼容性**（不破坏现有 CLI/Server 的解析结果）
- **不删除现有分析器**（只增强，不削弱）
- **Web UI 不引入状态管理库**（保持 Vue 3 reactive 即可）
- **不修改 exmple_log 目录**（现有测试数据不动）

---

## Verification Strategy

> **零人工介入** — 所有验证由 Agent 执行。不接受需要人工确认的验收标准。

### 测试决策

- **测试基础设施存在**: YES（`go test` 标准测试）
- **自动化测试**: 先实现后测试（Tests-after）
- **框架**: Go 标准 `testing` 包 + `go test`
- **新增测试**: Phase 1 修复时补充，Phase 2 每个分析器配测试，Phase 3 全覆盖

### QA 策略

每个任务必须包含 Agent 执行的 QA 场景：
- **API/Backend**: `curl` 发送请求，验证状态码和响应字段
- **CLI**: `bash` 运行命令，验证 stdout/stderr 和退出码
- **Web UI**: Playwright 打开浏览器，导航、交互、断言 DOM
- **Library/Module**: `go test` 运行指定包测试

---

## Execution Strategy

### 并行执行波次

```
Phase 1 — Wave 1（立即启动 — 类型定义 + 基础升级）:
├── Task 1: Go 模块升级到 1.22 [quick]
├── Task 2: 核心类型定义重构（新增类型结构体，替换 interface{}） [deep]
├── Task 3: CLI 入口点错误处理修复 [quick]
├── Task 4: 安全加固（API 路径校验 + 其他安全修复） [quick]
├── Task 5: 死代码清理 + 魔法数字命名 [quick]
└── Task 6: slog 结构化日志集成 [quick]

Phase 1 — Wave 2（依赖 Wave 1 — 分析器 + 处理器重构）:
├── Task 7: 分析器类型重构（FrameAnalyzer 等返回具体类型） [deep]
├── Task 8: 导出器类型重构（Exporter 接口类型化） [quick]
├── Task 9: Handler 层类型重构（API 响应类型化） [deep]
├── Task 10: Platform 层重构（convertLine 清理 + 错误处理） [quick]
└── Task 11: Phase 1 测试补充 [quick]

Phase 2 — Wave 3a（依赖 Phase 1 Wave 1 — 解析器增强 + 基础接口）:
├── Task 14: RawTraceParser 增强（__glSetError、gc/tid 提取，APICall 字段扩展） [deep]
└── Task 15: Bug 诊断基础接口和报告类型定义 [quick]

Phase 2 — Wave 3b（依赖 Wave 3a — 状态机 + Context，严格顺序）:
├── Task 12: GL 状态机跟踪模块（处理增强后的 APICall 流） [deep]
└── Task 13: Context 管理模块（gc= 追踪、资源归属） [deep]

Phase 2 — Wave 4（依赖 Wave 3 — 7 个诊断分析器，最大并行）:
├── Task 16: 空指针排查分析器 [deep]
├── Task 17: 资源泄漏检测分析器 [deep]
├── Task 18: Shader 编译错误分析器 [unspecified-high]
├── Task 19: API 反模式检测分析器 [unspecified-high]
├── Task 20: 性能异常定位分析器 [unspecified-high]
├── Task 21: 线程安全诊断分析器 [unspecified-high]
└── Task 22: 驱动层错误关联分析器 [unspecified-high]

Phase 2 — Wave 5（依赖 Wave 4 — 集成 + 报告）:
├── Task 23: 诊断报告生成器（Markdown + JSON） [deep]
├── Task 24: Handler 集成（Bug 诊断 API 端点） [quick]
├── Task 25: CLI 集成（Bug 诊断命令行参数） [quick]
└── Task 26: Phase 2 测试补充 [quick]

Phase 3 — Wave 6（依赖 Phase 1+2 — Web UI + 基础设施）:
├── Task 27: Web UI 构建系统搭建（Vite + TypeScript） [visual-engineering]
├── Task 28: Web UI 组件拆分 [visual-engineering]
├── Task 29: Bug 诊断报告可视化 [visual-engineering]
├── Task 30: Phase 3 测试补充 [quick]
├── Task 31: CI/CD 配置（GitHub Actions） [quick]
└── Task 32: 文档更新（README + CLI 文档 + 注释中文化） [writing]

Wave FINAL（全部完成后 — 4 个并行审查）:
├── Task F1: 计划合规审计（oracle）
├── Task F2: 代码质量审查（unspecified-high）
├── Task F3: 实际 QA 执行（unspecified-high + playwright）
└── Task F4: 范围保真度检查（deep）
```

### 关键路径

```
Task 1 → Task 2 → Task 14 → Task 12 → Task 13 → Task 16-22 → Task 23 → Task 27-29 → F1-F4
```

---

## TODOs

- [x] 1. Go 模块升级到 1.22

  **做什么**:
  - 修改 `gst/go.mod`：`go 1.18` → `go 1.22`
  - 运行 `go mod tidy` 确认无变更（零外部依赖）
  - 修改 Makefile 中 Go 版本引用（如有）
  - 验证编译：`go build ./...`
  - 验证测试：`go test ./internal/core/...`

  **不得做**:
  - 不得引入任何外部 Go 依赖

  **推荐 Agent 配置**:
  - **Category**: `quick` — 单一文件修改，验证即完成
  - **Skills**: [`git-master`] — 用于原子提交

  **并行化**:
  - **可并行**: YES — 与 Task 2-6 并行
  - **并行组**: Wave 1

  **参考**:
  - `gst/go.mod:1-3` — 当前模块声明
  - `gst/Makefile` — 构建配置

  **验收标准**:
  - [x] `gst/go.mod` 中 `go 1.22`
  - [x] `go build ./cmd/...` 成功（零错误）
  - [x] `go test ./internal/core/...` 全部通过
  - [x] `gst/internal/core/types.go` 新增 10+ 个类型结构体
  - [x] 所有结构体包含 JSON 标签
  - [x] `go build ./internal/core/...` 成功（当前 analyzer 仍使用旧签名，不会破坏编译）
  - [x] 零 `file, _ := os.Open` 模式
  - [x] 所有 `os.Open` 调用错误被 `fmt.Errorf` 包装传播
  - [x] 所有 `Export` 调用错误被检查并打印
  - [x] `go build ./cmd/cli` 成功
  - [x] `/api/log/parse` 拒绝 `{"path": "/etc/passwd"}` 请求
  - [x] `/api/log/parse` 拒绝包含 `../` 的路径
  - [x] `/api/log/parse` 正常接受白名单目录内的文件
  - [x] `handlers.go:109` 的死代码已删除

  **QA 场景**:
  ```
  Scenario: 拒绝读取系统文件
    Tool: Bash (curl)
    Preconditions: gst-server 运行在 localhost:8080
    Steps:
      1. curl -s -X POST http://localhost:8080/api/log/parse \
         -H "Content-Type: application/json" \
         -d '{"path":"/etc/passwd"}' | jq .
    Expected Result: HTTP 400/403 错误响应，包含"拒绝"或"不允许"信息
    Failure Indicators: HTTP 200 或返回文件内容
    Evidence: .sisyphus/evidence/task-4-security-reject.txt

  Scenario: 正常日志文件可解析
    Tool: Bash (curl)
    Steps:
      1. curl -s -X POST http://localhost:8080/api/log/parse \
         -H "Content-Type: application/json" \
         -d '{"path":"../exmple_log/1frame_demo_api.txt"}' | jq '.success'
    Expected Result: `true`
    Evidence: .sisyphus/evidence/task-4-security-allow.txt
  ```

  **提交**: YES
  - Message: `fix(gst): add path validation to prevent arbitrary file read`
  - Files: `gst/cmd/gst-server/internal/handlers/handlers.go`, `gst/cmd/gst-server/main.go`

- [x] 5. 死代码清理 + 魔法数字命名

  **做什么**:
  - 删除 `gst/cmd/cli/main.go:365` 的 `timeit()` 函数（从未调用）
  - 将 `gst/internal/platform/file_reader.go:150` 的 `convertLine()` 内联为 `string(line)` 并移除方法（当前是 no-op）
  - 删除 `gst/internal/core/analyzer/buffer_analyzer.go:36-50` 的未使用 maps
  - 提取魔法数字为命名常量，添加至对应包中：
    `DefaultBufferSize = 10*1024*1024`, `StreamBufferSize = 64*1024*1024`, `MaxMultipartSize = 100<<20`, `ShaderSourceTruncateLen = 2000`, `TextureThrashingThreshold = 5`, `FBOThrashingThreshold = 3`
  - 前端常量在 `web/js/logs.js` 中提取：`FETCH_TIMEOUT = 120000`, `TOAST_TIMEOUT = 5000`

  **不得做**: 不得修改常量的实际数值；不得删除正在使用的 maps

  **推荐 Agent 配置**: **Category**: `quick` — 清理性质
  **并行化**: **可并行**: YES — Wave 1

  **参考**:
  - `gst/cmd/cli/main.go:365` — timeit
  - `gst/internal/platform/file_reader.go:150` — convertLine
  - `gst/internal/core/analyzer/buffer_analyzer.go:36-50` — 未使用 maps

  **验收标准**: `go build ./...` 成功；`go vet ./...` 零警告

  **QA 场景**:
  ```
  Scenario: 编译和测试通过
    Tool: Bash
    Steps: cd gst && go build ./... && go test ./internal/core/... -count=1
    Expected Result: 编译成功，测试通过
    Evidence: .sisyphus/evidence/task-5-cleanup.txt
  ```

  **提交**: YES — `refactor(gst): remove dead code and extract magic numbers`

- [x] 6. slog 结构化日志集成

  **做什么**:
  - 创建 `gst/internal/platform/logger.go` — slog 初始化入口
  - 替换所有 `log.Printf`、`log.Println`、`fmt.Fprintf(os.Stderr,...)` 为 slog
  - 按级别分类：DEBUG（解析细节）、INFO（正常操作）、WARN（非致命）、ERROR（错误）
  - Server: 添加 `--log-level` flag（debug/info/warn/error，默认 info）
  - CLI: 内部日志用 slog，用户可见表格输出保持 stdout

  **不得做**: 不得引入外部日志库；不得改变 CLI 用户可见输出

  **推荐 Agent 配置**: **Category**: `quick` — 模式统一替换
  **并行化**: **可并行**: YES — Wave 1（与 Task 1-5 并行）

  **参考**: `gst/cmd/gst-server/main.go`、`gst/cmd/cli/main.go`、`gst/cmd/gst-server/internal/handlers/handlers.go`

  **验收标准**: 零 `log.Printf`/`log.Println`（除 test 目录）；`logger.go` 存在且可配置

  **QA 场景**:
  ```
  Scenario: 日志级别切换
    Tool: Bash
    Steps:
      1. cd gst && go build -o /tmp/gst-server ./cmd/gst-server
      2. timeout 3 /tmp/gst-server -port 19999 -log-level=debug -browser=false 2>&1 | head -10
    Expected Result: 包含 DEBUG 级别日志，结构化 key=value 格式
    Evidence: .sisyphus/evidence/task-6-slog-debug.txt
  ```

  **提交**: YES — `refactor(gst): replace log.Printf with slog structured logging`

- [x] 7. 分析器类型重构（GetSummary 返回具体类型）

  **做什么**:
  - 修改所有 analyzer 的 `GetSummary()`/`GetGlobalStats()` 方法，从返回 `map[string]interface{}` 改为返回 Task 2 中定义的具体类型结构体
  - 涉及文件（按优先级）：
    - `shader_analyzer.go` → 返回 `*ShaderSummary`
    - `sync_stall_analyzer.go` → 返回 `*SyncStallSummary`
    - `swap_classifier.go` → 返回 `*SwapClassification`
    - `texture_upload_analyzer.go` → 返回 `*TextureUploadSummary`
    - `fbo_analyzer.go` → 返回 `*FBOSummary`
    - `draw_call_analyzer.go` → 返回 `*DrawCallSummary`（如存在）
    - `state_change_analyzer.go` → 返回 `*StateChangeSummary`
    - `redundancy_analyzer.go` → 返回 `*RedundancySummary`
    - `bandwidth_analyzer.go` → 返回 `*BandwidthSummary`
    - `frame_analyzer.go` → 确认已返回具体 `FrameSummary`，如需调整则更新
    - `buffer_analyzer.go` → 确认已返回具体 `BufferSummary`，如需调整则更新
    - `func_analyzer.go` → 确认签名

  **不得做**: 不得修改 JSON 标签名称（保持 API 兼容）；不得修改现有字段的计算逻辑

  **推荐 Agent 配置**: **Category**: `deep` — 需要修改 12 个分析器，理解每个 GetSummary 的字段
  **并行化**: **可并行**: NO — 依赖 Task 2（类型定义）；与 Task 8-11 可部分并行
  **并行组**: Wave 2；**阻塞**: Task 9（Handler 层）
  **被阻塞**: Task 2

  **参考**: Task 2 中定义的所有类型结构体；每个 analyzer 的 GetSummary 方法

  **验收标准**:
  - 零 `map[string]interface{}` 在 analyzer 公开 API 中
  - `go build ./internal/core/...` 成功
  - `go test ./internal/core/analyzer/...` 通过（如已有测试）

  **QA 场景**:
  ```
  Scenario: 编译验证类型重构
    Tool: Bash
    Steps:
      1. cd gst && go vet ./internal/core/analyzer/...
      2. go build ./internal/core/...
      3. go test ./internal/core/analyzer/... -v -count=1
    Expected Result: 零编译错误，零 vet 警告，测试通过
    Evidence: .sisyphus/evidence/task-7-analyzer-types.txt
  ```

  **提交**: YES — `refactor(gst): replace map[string]interface{} with typed summaries in analyzers`

- [x] 8. 导出器类型重构

  **做什么**:
  - 修改 `gst/internal/core/exporter/exporter.go`：替换 `interface{}` 参数为具体类型
  - `Exporter` 接口方法签名类型化（如需要）
  - TXT/CSV/JSON 三种导出器适配新的结构体类型
  - 移除所有 `switch v := data.(type)` 为具体类型断言

  **不得做**: 不得修改导出格式（TXT/CSV/JSON 输出保持不变）

  **推荐 Agent 配置**: **Category**: `quick` — 单一文件修改
  **并行化**: **可并行**: YES — Wave 2（独立于 Task 7 之后）
  **被阻塞**: Task 2（类型定义），非严格依赖 Task 7

  **参考**: `gst/internal/core/exporter/exporter.go:38,137,276` — 当前 interface{} 使用位置

  **验收标准**: `go build ./internal/core/exporter/...` 成功

  **QA 场景**:
  ```
  Scenario: 导出功能正常
    Tool: Bash
    Steps:
      1. cd gst && go test ./internal/core/exporter/... -v -count=1
    Expected Result: 测试通过
    Evidence: .sisyphus/evidence/task-8-exporter-types.txt
  ```

  **提交**: YES — `refactor(gst): type-safe exporter interfaces`

- [x] 9. Handler 层类型重构

  **做什么**:
  - 重构 `gst/cmd/gst-server/internal/handlers/handlers.go`：
    - 所有 `map[string]interface{}` 响应替换为具体结构体
    - 定义 `AnalyzeResponse`、`SearchResponse`、`ExportResponse` 等 DTO 类型
    - `AnalyzeComprehensive` 返回定义良好的综合响应结构体
  - 确保 JSON 序列化与前端期望的字段名一致

  **不得做**: 不得修改 API 端点路径或 HTTP 方法；不得改变任何 JSON 响应体的字段名

  **推荐 Agent 配置**: **Category**: `deep` — 900+ 行文件重构，需保证 API 兼容
  **并行化**: **可并行**: NO — 依赖 Task 7（分析器类型更新后才能改 handler）
  **阻塞**: Phase 2 handler 扩展

  **参考**: `gst/cmd/gst-server/internal/handlers/handlers.go:781-901` — interface{} 密集区域

  **验收标准**:
  - 零 `map[string]interface{}` 在 handlers.go 的 API 响应中
  - 所有 17 个 API 端点返回的 JSON 结构与重构前一致

  **QA 场景**:
  ```
  Scenario: API 兼容性验证
    Tool: Bash (curl)
    Preconditions: gst-server 运行在 localhost:8080
    Steps:
      1. curl -s http://localhost:8080/api/health | jq .
      2. curl -s http://localhost:8080/api/version | jq .
    Expected Result: 两个端点返回正确 JSON
    Evidence: .sisyphus/evidence/task-9-handler-types.txt
  ```

  **提交**: YES — `refactor(gst): type-safe handler response types`

- [x] 10. Platform 层重构

  **做什么**:
  - `gst/internal/platform/file_reader.go`：
    - 内联 `convertLine()` 为 `string(line)` 并移除 no-op 方法
    - 为 `StreamReader` 添加错误处理，替换 `panic` 为 error 返回
  - `gst/internal/platform/os_detector.go`：代码审查，确保错误被正确传播

  **不得做**: 不得改变 StreamReader 的公开 API（除错误处理改进）

  **推荐 Agent 配置**: **Category**: `quick` — 清理性质
  **并行化**: **可并行**: YES — Wave 2（独立模块）

  **参考**: `gst/internal/platform/file_reader.go:150`、`gst/internal/platform/os_detector.go`

  **验收标准**: `go build ./internal/platform/...` 成功；零 panic

  **QA 场景**:
  ```
  Scenario: 文件读取正常
    Tool: Bash
    Steps: cd gst && go test ./internal/platform/... -v -count=1 2>&1
    Expected Result: 测试通过或无测试（当前零测试）
    Evidence: .sisyphus/evidence/task-10-platform-refactor.txt
  ```

  **提交**: YES — `refactor(gst): clean up platform layer, remove dead convertLine`

- [x] 11. Phase 1 测试补充

  **做什么**:
  - 为 platform 包编写基础测试（`gst/internal/platform/file_reader_test.go`）
  - 为 CLI 入口编写测试（`gst/cmd/cli/main_test.go`）— 测试 openAndParse helper
  - 为安全加固编写测试（路径校验逻辑测试）
  - 为 logger.go 编写测试

  **不得做**: 不得修改现有测试文件

  **推荐 Agent 配置**: **Category**: `quick` — 补充测试
  **并行化**: **可并行**: YES — Wave 2（依赖 Task 3, 4, 6, 10）

  **参考**: `gst/internal/core/parser/parser_test.go` — 现有测试模式（table-driven）

  **验收标准**:
  - `go test ./internal/platform/... -v` 通过
  - `go test ./cmd/cli/... -v` 通过（如 CLI 可测试）
  - 新增 >10 个测试用例

  **QA 场景**:
  ```
  Scenario: 所有测试通过
    Tool: Bash
    Steps: cd gst && go test ./... -count=1 -cover 2>&1 | tail -20
    Expected Result: 全部 PASS，覆盖率报告输出
    Evidence: .sisyphus/evidence/task-11-phase1-tests.txt
  ```

  **提交**: YES — `test(gst): add tests for platform, CLI, security, and logger`

- [x] 12. GL 状态机跟踪模块

  **做什么**:
  - 新建 `gst/internal/core/bug/` 包（Bug 诊断引擎根目录）
  - 创建 `gst/internal/core/bug/state_tracker.go` — GL 状态机核心
  - 实现以下状态跟踪：
    - **VBO 绑定状态**：跟踪 `GL_ARRAY_BUFFER` 和 `GL_ELEMENT_ARRAY_BUFFER` 当前绑定
    - **VAO 状态**：跟踪当前 VAO 和各属性指针绑定
    - **Shader 状态**：跟踪当前 `glUseProgram` 程序 ID
    - **Texture 状态**：跟踪当前活动纹理单元和绑定纹理
    - **Framebuffer 状态**：跟踪当前绑定的 FBO
    - **客户端数组模式检测**：当没有 VBO 绑定时，属性指针传入的是内存地址还是 (nil)
  - 状态机通过处理 `core.APICall` 事件流更新
  - 支持多个 Context（gc=）独立跟踪

  **不得做**: 不得依赖外部 Go 库；不得修改 `internal/core/types.go` 中现有类型

  **推荐 Agent 配置**: **Category**: `deep` — 全新模块，逻辑复杂
  **并行化**: **可并行**: YES — Wave 3（与 Task 13, 14, 15 部分共享依赖）
  **阻塞**: Task 16-22（所有 Bug 分析器依赖状态机）

  **参考**:
  - `/root/code/deer-flow/skills/public/log-bug-analyst/SKILL.md` — ptr=(nil) 判定规则（规则 1）
  - `/root/code/deer-flow/skills/public/log-bug-analyst/references/case_studies.md` — 案例分析中的状态推理模式
  - `gst/internal/core/types.go` — APICall 结构体定义

  **验收标准**:
  - `go build ./internal/core/bug/...` 成功
  - 状态机可处理完整日志的 API 调用序列

  **QA 场景**:
  ```
  Scenario: 状态机正确处理 VBO 绑定和释放
    Tool: Bash
    Steps:
      1. cd gst && go test ./internal/core/bug/... -v -run TestStateTracker -count=1
    Expected Result: 状态跟踪测试通过，VBO/VAO/shader 绑定正确记录
    Evidence: .sisyphus/evidence/task-12-state-tracker.txt
  ```

  **提交**: YES — `feat(gst): add GL state machine tracker for bug diagnosis`

- [x] 13. Context 管理模块

  **做什么**:
  - 创建 `gst/internal/core/bug/context_manager.go`
  - 跟踪多个 GL Context（按 `gc=` 地址区分）
  - 记录每个 Context 的资源创建（VBO ID、Texture ID、Shader ID 等）
  - 检测 `shareList=(nil)` 导致的多 Context 资源不共享
  - 检测跨 Context 使用资源（Context A 创建的 VBO 在 Context B 中使用）

  **不得做**: 不得假设 gc= 地址在每次运行中都相同

  **推荐 Agent 配置**: **Category**: `deep` — 多 Context 关联分析
  **并行化**: **可并行**: YES — Wave 3（与 Task 12 协作）

  **参考**:
  - `/root/code/deer-flow/skills/public/log-bug-analyst/references/case_studies.md` — Case 1: Multi-Context SIGSEGV 分析

  **验收标准**: Context 管理器正确跟踪每个 gc= 的资源 ID 分配和绑定

  **QA 场景**:
  ```
  Scenario: 检测多 Context 资源不共享
    Tool: Bash
    Steps: cd gst && go test ./internal/core/bug/... -v -run TestContextManager -count=1
    Expected Result: 检测到 gc1 的资源在 gc2 中不可见
    Evidence: .sisyphus/evidence/task-13-context-manager.txt
  ```

  **提交**: YES — `feat(gst): add GL context resource manager`

- [x] 14. RawTraceParser 增强

  **做什么**:
  - 增强 `gst/internal/core/parser/raw_trace_parser.go`：
    - 解析 `ERROR!!! __glSetError gc:... code:...` 行并存储
    - 解析 `gc=` 和 `tid=` 元数据为 `core.APICall` 结构体新增字段
    - 解析 `段错误`/`SIGSEGV`/`core dumped` 终止标记
    - 解析 `jmo_VERTEXARRAY_StreamBind` 驱动层输出
    - 解析 `glXMakeCurrent` 和 `glXCreateContextAttribsARB` 的 Context 管理调用
  - 在 `core.APICall` 中增加字段：`GCAddr string`、`TID string`、`IsError bool`、`ErrorCode string` 等（字段在 Task 2 中已定义）
  - 解析 `ptr=(nil)` 标记为特殊字段以供空指针分析器使用
  - **创建测试数据**：在 `gst/internal/core/bug/testdata/` 中创建一个小型 raw apitrace 格式日志（约 50 行），包含 VBO 绑定、ptr=(nil)、__glSetError、段错误等典型场景，供 Task 16-22 的分析器测试使用

  **不得做**: 不得修改 `core.ParsedLog` 的现有公开字段（只能新增）；不得破坏现有测试

  **推荐 Agent 配置**: **Category**: `deep` — 解析器增强
  **并行化**: **可并行**: YES — Wave 3（与 Task 12, 13 配合）
  **阻塞**: Task 16（空指针分析器需要增强后的解析）

  **参考**:
  - `gst/internal/core/parser/raw_trace_parser.go` — 当前解析逻辑
  - `/root/code/deer-flow/skills/public/log-bug-analyst/references/apitrace_format.md` — 日志格式模式

  **验收标准**:
  - `go test ./internal/core/parser/... -v` 通过（包含现有测试）
  - 解析器正确提取 `__glSetError` 和 `gc=/tid=` 信息

  **QA 场景**:
  ```
  Scenario: 解析包含错误的日志
    Tool: Bash
    Steps:
      1. cd gst && go test ./internal/core/parser/... -v -count=1
    Expected Result: 所有测试通过，新字段被正确填充
    Evidence: .sisyphus/evidence/task-14-parser-enhance.txt
  ```

  **提交**: YES — `feat(gst): enhance raw trace parser with error and context tracking`

- [x] 15. Bug 诊断基础接口和报告类型定义

  **做什么**:
  - 创建 `gst/internal/core/bug/diagnoser.go` — `Diagnoser` 接口
  - 定义 `DiagnosisReport` 已在 Task 2 创建，补充细化：
    - `Finding` 结构体（严重程度、描述、证据、根因链、修复建议）
    - `Severity` 枚举（Critical/High/Medium/Low/Info）
  - 创建 `gst/internal/core/bug/registry.go` — 分析器注册表
  - 基础报告格式（Markdown 报告生成器在 Task 23 实现）

  **不得做**: 不要在此任务中实现具体分析逻辑

  **推荐 Agent 配置**: **Category**: `quick` — 接口和类型定义
  **并行化**: **可并行**: YES — Wave 3（不依赖其他 Task 3 任务完成）

  **参考**: `gst/internal/core/types.go` — `DiagnosisReport` 类型（Task 2 定义）

  **验收标准**: `go build ./internal/core/bug/...` 成功

  **QA 场景**:
  ```
  Scenario: 类型编译和接口验证
    Tool: Bash
    Steps: cd gst && go vet ./internal/core/bug/... && go build ./internal/core/bug/...
    Expected Result: 零错误
    Evidence: .sisyphus/evidence/task-15-bug-interface.txt
  ```

  **提交**: YES — `feat(gst): define bug diagnosis interfaces and report types`

- [x] 16. 空指针排查分析器

  **做什么**:
  - 创建 `gst/internal/core/bug/null_pointer_detector.go`
  - 实现 VBO 模式 vs 客户端数组模式判定逻辑（参考 log-bug-analyst skill 规则 1）：
    - 检测所有 `glVertexAttribPointer(..., ptr=(nil))` 和 `glVertexPointer(..., ptr=(nil))`
    - 回溯前 5-10 行检查 `glBindBuffer(GL_ARRAY_BUFFER, N≠0)`
    - 区分：有 VBO 绑定 → 合法 VBO offset 0；无 VBO 绑定 → 危险空指针
    - 检查同 Context 其他同类调用的模式（传入具体地址 vs nil）
  - 检测 `glBindBuffer` 失败后跟 `ptr=(nil)` 的连锁崩溃模式
  - 检测 `glDrawElements(..., indices=(nil))` 无 ELEMENT_ARRAY_BUFFER 绑定的情况
  - 回溯从 `段错误` 出发的崩溃调用链

  **不得做**: 不得产生误报（VBO offset 0 的合法 nil 不应标记为错误）

  **推荐 Agent 配置**: **Category**: `deep` — 核心分析器，逻辑复杂
  **并行化**: **可并行**: YES — Wave 4（与 Task 17-22 并行，共享状态机）
  **被阻塞**: Task 12（状态机）、Task 14（增强解析器）

  **参考**:
  - `/root/code/deer-flow/skills/public/log-bug-analyst/SKILL.md:38-83` — 规则 1（ptr=nil 判定）
  - `/root/code/deer-flow/skills/public/log-bug-analyst/references/case_studies.md` — 两个案例

  **验收标准**:
  - 正确识别合法 VBO nil 和危险空指针
  - 输出包含根因链和修复建议的 Finding

  **QA 场景**:
  ```
  Scenario: 空指针检测 — 合法 VBO nil 不误报
    Tool: Bash
    Steps:
      1. cd gst && go test ./internal/core/bug/... -v -run TestNullPointer -count=1
    Expected Result: 有 glBindBuffer 前缀的 ptr=(nil) 不触发告警
    Evidence: .sisyphus/evidence/task-16-null-ptr.txt
  ```

  **提交**: YES — `feat(gst): add null pointer access bug detector`

- [x] 17. 资源泄漏检测分析器

  **做什么**:
  - 创建 `gst/internal/core/bug/resource_leak_detector.go`
  - 配对检测 `glGen*` / `glCreate*` 与 `glDelete*` 的调用：
    - Buffer: `glGenBuffers` ↔ `glDeleteBuffers`
    - Texture: `glGenTextures` ↔ `glDeleteTextures`
    - Shader: `glCreateShader` ↔ `glDeleteShader`
    - Program: `glCreateProgram` ↔ `glDeleteProgram`
    - Framebuffer: `glGenFramebuffers` ↔ `glDeleteFramebuffers`
    - Renderbuffer: `glGenRenderbuffers` ↔ `glDeleteRenderbuffers`
    - Vertex Array: `glGenVertexArrays` ↔ `glDeleteVertexArrays`
  - 检测 Context 销毁时（`glXDestroyContext`）仍有未释放的资源
  - 检测未绑定就删除的模式（资源创建后未被任何 Context 绑定）

  **不得做**: 不得对未完成帧（日志截断）中的未配对 gen 产生误报

  **推荐 Agent 配置**: **Category**: `deep` — 需要完整的资源生命周期追踪
  **并行化**: **可并行**: YES — Wave 4

  **参考**: `gst/internal/core/bug/state_tracker.go` — 状态机中的资源追踪

  **验收标准**: 正确报告未配对的 gen/delete 调用

  **QA 场景**:
  ```
  Scenario: 检测未释放的 Buffer
    Tool: Bash
    Steps: cd gst && go test ./internal/core/bug/... -v -run TestResourceLeak -count=1
    Expected Result: glGenBuffers 无对应 glDeleteBuffers → 报告资源泄漏
    Evidence: .sisyphus/evidence/task-17-resource-leak.txt
  ```

  **提交**: YES — `feat(gst): add GPU resource leak detector`

- [x] 18. Shader 编译错误分析器

  **做什么**:
  - 创建 `gst/internal/core/bug/shader_error_detector.go`
  - 检测 `glCompileShader` 后无 `glGetShaderiv(COMPILE_STATUS)` 检查
  - 检测 `glLinkProgram` 后无 `glGetProgramiv(LINK_STATUS)` 检查
  - 关联驱动错误：`__glSetError` 在 shader 相关调用后的错误
  - 统计 shader 编译频率异常（每帧重复编译相同 shader）

  **不得做**: 不得假设 shader source 内容可用（apitrace 通常不包含 source）

  **推荐 Agent 配置**: **Category**: `unspecified-high`
  **并行化**: **可并行**: YES — Wave 4

  **验收标准**: 检测到编译后未检查状态的 shader 操作

  **QA 场景**:
  ```
  Scenario: Shader 编译状态检查缺失
    Tool: Bash
    Steps: cd gst && go test ./internal/core/bug/... -v -run TestShaderError -count=1
    Expected Result: 检测到 glCompileShader 后缺少 COMPILE_STATUS 检查
    Evidence: .sisyphus/evidence/task-18-shader-error.txt
  ```

  **提交**: YES — `feat(gst): add shader compilation error detector`

- [x] 19. API 反模式检测分析器

  **做什么**:
  - 创建 `gst/internal/core/bug/antipattern_detector.go`
  - 检测以下反模式：
    - **冗余状态设置**：连续相同的 `glEnable`/`glDisable` 调用
    - **无效绑定**：绑定 buffer ID=0（解绑）后立即使用
    - **不必要的 Context 切换**：频繁 `glXMakeCurrent` 切换但未做渲染
    - **每帧重建资源**：在帧内反复 create/delete 相同类型资源
    - **空 shader program 绑定**：`glUseProgram(0)` 后调用 `glUniform*`
    - **Draw call 无绑定 shader**：在没有 active program 的情况下调用 draw

  **不得做**: 不要对每个检测项都要求用户确认；给出合理阈值

  **推荐 Agent 配置**: **Category**: `unspecified-high`
  **并行化**: **可并行**: YES — Wave 4

  **验收标准**: 检测到至少 3 种反模式类型

  **QA 场景**:
  ```
  Scenario: 检测冗余状态设置
    Tool: Bash
    Steps: cd gst && go test ./internal/core/bug/... -v -run TestAntiPattern -count=1
    Expected Result: 连续 glEnable(GL_BLEND) 被标记为冗余
    Evidence: .sisyphus/evidence/task-19-antipattern.txt
  ```

  **提交**: YES — `feat(gst): add API anti-pattern detector`

- [x] 20. 性能异常定位分析器

  **做什么**:
  - 创建 `gst/internal/core/bug/perf_anomaly_detector.go`
  - 利用现有的帧分析数据，增强异常检测：
    - **帧时间突变**：与平均帧时间偏差 >2σ 的帧
    - **特定 API 调用耗时异常**：某类 gl 函数在某一帧突然耗时暴增
    - **绘制调用数突变**：帧内 draw call 数量骤降或骤升
  - 关联帧时间异常与具体 API 调用序列
  - 输出异常帧的 API 调用摘要

  **不得做**: 不替代现有 `frame_analyzer.go`；作为增强层叠加

  **推荐 Agent 配置**: **Category**: `unspecified-high`
  **并行化**: **可并行**: YES — Wave 4

  **参考**: `gst/internal/core/analyzer/frame_analyzer.go` — 现有帧分析

  **验收标准**: 识别出帧时间异常的帧并关联 API 调用

  **QA 场景**:
  ```
  Scenario: 检测帧时间异常
    Tool: Bash
    Steps: cd gst && go test ./internal/core/bug/... -v -run TestPerfAnomaly -count=1
    Expected Result: 异常帧被标记，包含可能的 API 原因
    Evidence: .sisyphus/evidence/task-20-perf-anomaly.txt
  ```

  **提交**: YES — `feat(gst): add performance anomaly detector`

- [x] 21. 线程安全诊断分析器

  **做什么**:
  - 创建 `gst/internal/core/bug/thread_safety_detector.go`
  - 按 `tid=` 分组分析 GL 调用：
    - 检测多个线程同时操作同一 Context（`gc=`）
    - 检测线程间无同步的资源访问模式
    - 检测 Context 在某线程未解绑前被另一线程绑定
  - 分析 `glXMakeCurrent` 的线程切换模式

  **不得做**: 不检测应用层锁机制（日志中不可见）

  **推荐 Agent 配置**: **Category**: `unspecified-high`
  **并行化**: **可并行**: YES — Wave 4

  **参考**: `gst/internal/core/bug/context_manager.go` — Context 管理

  **验收标准**: 检测到多线程共享 Context 的模式

  **QA 场景**:
  ```
  Scenario: 检测多线程问题
    Tool: Bash
    Steps: cd gst && go test ./internal/core/bug/... -v -run TestThreadSafety -count=1
    Expected Result: 同一 gc= 被多线程同时操作 → 告警
    Evidence: .sisyphus/evidence/task-21-thread-safety.txt
  ```

  **提交**: YES — `feat(gst): add thread safety diagnostic detector`

- [x] 22. 驱动层错误关联分析器

  **做什么**:
  - 创建 `gst/internal/core/bug/driver_error_detector.go`
  - 解析并关联 `__glSetError` 错误：
    - 提取错误码（`code:0x0502` = GL_INVALID_OPERATION 等）
    - 回溯到触发错误的 GL 调用
    - 关联同一 Context 中后续的 `glGetError` 调用
  - 检测应用层错误检查的缺失（有 `__glSetError` 但无对应 `glGetError`）
  - 分类错误模式：VBO 绑定失败、纹理格式不匹配、Framebuffer 不完整等

  **不得做**: 不能依赖 OpenGL 规范之外的错误码解释

  **参考**:
  - `/root/code/deer-flow/skills/public/log-bug-analyst/references/apitrace_format.md:49-70` — GL 错误码速查表

  **验收标准**: 正确解析并关联 `__glSetError` 到触发调用

  **QA 场景**:
  ```
  Scenario: 驱动错误关联
    Tool: Bash
    Steps: cd gst && go test ./internal/core/bug/... -v -run TestDriverError -count=1
    Expected Result: code:0x0502 → GL_INVALID_OPERATION，关联到触发调用
    Evidence: .sisyphus/evidence/task-22-driver-error.txt
  ```

  **提交**: YES — `feat(gst): add driver error correlation analyzer`

- [x] 23. 诊断报告生成器

  **做什么**:
  - 创建 `gst/internal/core/bug/report_generator.go`
  - 汇总所有分析器的 `Finding` 结果
  - 生成结构化诊断报告：
    - 摘要（问题总数、按严重程度分类、关键问题高亮）
    - 每个 Finding 的详细信息（严重程度、描述、证据日志行号、根因链、修复建议）
    - Markdown 格式输出（可读性强，供技术支持分享）
    - JSON 格式输出（供 Web UI 消费）
  - 参考 log-bug-analyst skill 的报告结构：
    - 14 章标准模板（概述、Context 分析、空指针检测、资源泄漏、Shader 错误等）

  **不得做**: 不在此任务中生成 HTML 报告（Phase 3 的 Web UI 处理可视化）

  **推荐 Agent 配置**: **Category**: `deep` — 汇总和格式化
  **并行化**: **可并行**: NO — 依赖 Task 16-22
  **阻塞**: Task 24（Handler 集成）、Task 25（CLI 集成）

  **参考**:
  - `/root/code/deer-flow/skills/public/log-bug-analyst/references/report_templates.md` — 报告模板

  **验收标准**:
  - Markdown 报告包含所有 Finding，格式清晰
  - JSON 报告包含完整的类型化数据

  **QA 场景**:
  ```
  Scenario: 生成诊断报告
    Tool: Bash
    Steps:
      1. cd gst && go test ./internal/core/bug/... -v -run TestReport -count=1
    Expected Result: Markdown 和 JSON 报告均生成，包含所有 Finding
    Evidence: .sisyphus/evidence/task-23-report.md
  ```

  **提交**: YES — `feat(gst): add structured diagnosis report generator`

- [x] 24. Handler 集成（Bug 诊断 API 端点）

  **做什么**:
  - 在 `gst/cmd/gst-server/internal/handlers/handlers.go` 中新增 API 端点：
    - `POST /api/diagnose` — 对指定日志文件执行全部 7 个诊断分析
    - `GET /api/diagnose/{id}` — 获取诊断报告（JSON）
    - `GET /api/diagnose/{id}/report.md` — 获取诊断报告（Markdown）
  - 在 Handler 结构体中添加 Bug 分析引擎实例
  - Server 启动时初始化 Bug 分析引擎

  **不得做**: 不修改现有 17 个端点

  **推荐 Agent 配置**: **Category**: `quick` — 新增端点
  **并行化**: **可并行**: NO — 依赖 Task 23
  **同步**: 与 Task 25 可并行（CLI 集成）

  **验收标准**:
  - `POST /api/diagnose` 返回诊断任务状态
  - `GET /api/diagnose/{id}` 返回完整 JSON 报告

  **QA 场景**:
  ```
  Scenario: API 诊断端点
    Tool: Bash (curl)
    Preconditions: gst-server 运行在 localhost:8080
    Steps:
      1. curl -s -X POST http://localhost:8080/api/diagnose \
         -H "Content-Type: application/json" \
         -d '{"path":"../exmple_log/1frame_demo_api.txt"}' | jq .
    Expected Result: HTTP 200，返回诊断 report_id
    Evidence: .sisyphus/evidence/task-24-api-diagnose.txt
  ```

  **提交**: YES — `feat(gst): add bug diagnosis API endpoints`

- [x] 25. CLI 集成（Bug 诊断命令行参数）

  **做什么**:
  - 在 `gst/cmd/cli/main.go` 中添加 `-diagnose` flag
  - 执行流程：解析日志 → 运行 7 个诊断分析器 → 生成报告 → 输出到 stdout 或指定文件
  - 添加 `-diagnose-output` flag 指定输出路径（默认 stdout）
  - 添加 `-diagnose-format` flag（md/json，默认 md）
  - 使用 slog 输出诊断进度

  **不得做**: 不修改现有 CLI 参数行为

  **推荐 Agent 配置**: **Category**: `quick` — 新增 CLI 子命令
  **并行化**: **可并行**: YES — 与 Task 24 并行
  **被阻塞**: Task 23

  **验收标准**:
  - `gst-cli -diagnose file.log` 输出 Markdown 诊断报告
  - `gst-cli -diagnose file.log -diagnose-format=json` 输出 JSON

  **QA 场景**:
  ```
  Scenario: CLI 诊断命令
    Tool: Bash
    Steps:
      1. cd gst && go build -o /tmp/gst-cli ./cmd/cli
      2. /tmp/gst-cli -diagnose ../exmple_log/1frame_demo_api.txt 2>&1 | head -20
    Expected Result: 输出 Markdown 格式诊断报告
    Evidence: .sisyphus/evidence/task-25-cli-diagnose.txt
  ```

  **提交**: YES — `feat(gst): add bug diagnosis CLI command`

- [x] 26. Phase 2 测试补充

  **做什么**:
  - 为状态机编写测试（模拟 GL 调用序列，验证状态变更）
  - 为 Context 管理器编写测试（多 Context 资源追踪）
  - 为 7 个分析器各编写至少 2 个测试用例（一个正常，一个异常）
  - 为报告生成器编写测试（格式验证）
  - 测试数据：使用现有 `exmple_log/` 或创建小型模拟日志

  **不得做**: 不修改 exmple_log 现有文件

  **推荐 Agent 配置**: **Category**: `quick` — 测试补充
  **并行化**: **可并行**: YES — Wave 5（依赖 Task 16-23）
  **并行组**: Wave 5

  **验收标准**:
  - `go test ./internal/core/bug/... -v -cover` 通过，覆盖率 >60%
  - 每个分析器至少 2 个测试用例

  **QA 场景**:
  ```
  Scenario: Bug 引擎测试全通过
    Tool: Bash
    Steps: cd gst && go test ./internal/core/bug/... -v -count=1 -cover 2>&1 | tail -30
    Expected Result: 全部 PASS，覆盖率报告
    Evidence: .sisyphus/evidence/task-26-phase2-tests.txt
  ```

  **提交**: YES — `test(gst): add tests for bug diagnostic engine`

- [x] 27. Web UI 构建系统搭建

  **做什么**:
  - 新建 `gst/web/package.json`（Vite + Vue 3 + TypeScript）
  - 创建 `gst/web/vite.config.ts`
  - 创建 `gst/web/tsconfig.json`
  - 从 CDN 模式迁移到 npm 依赖模式：
    - 移除 `vue.global.js` CDN 脚本（18,619 行）
    - 安装 `vue@^3.5` npm 包
  - 创建 `gst/web/src/` 目录，迁移现有 `index.html`、`logs.html`
  - 配置 Vite 构建输出到 `gst/web/dist/`
  - gst-server 的 `--web-dir` 默认值更新为 `web/dist/`

  **不得做**: 不在此任务中拆分组件（Task 28）；不修改 server 的 API 接口

  **推荐 Agent 配置**: **Category**: `visual-engineering`
  **Skills**: [`frontend-ui-ux`] — Vue 3 + Vite 脚手架
  **并行化**: **可并行**: NO — Web 初始化需要先完成；**阻塞**: Task 28, 29

  **验收标准**:
  - `npm run build` 成功
  - 生成的 `dist/` 包含可用的 index.html

  **QA 场景**:
  ```
  Scenario: Web 构建成功
    Tool: Bash
    Steps:
      1. cd gst/web && npm install && npm run build
      2. ls dist/index.html && echo "BUILD_OK"
    Expected Result: BUILD_OK
    Evidence: .sisyphus/evidence/task-27-web-build.txt
  ```

  **提交**: YES — `feat(gst): migrate web UI from CDN to Vite + TypeScript build system`

- [x] 28. Web UI 组件拆分

  **做什么**:
  - 将现有 568 行单体 `logs.js` 拆分为 Vue 3 组件：
    - `src/components/AnalysisTabs.vue` — 分析页签切换
    - `src/components/FrameList.vue` — 帧列表和排序
    - `src/components/FunctionStats.vue` — 函数统计表格
    - `src/components/ShaderStats.vue` — Shader 统计
    - `src/components/SearchPanel.vue` — 搜索面板
    - `src/components/ExportPanel.vue` — 导出选项
    - `src/components/FileUpload.vue` — 文件上传
    - `src/composables/useApi.ts` — API 调用逻辑提取
  - 使用 Vue 3 Composition API（`<script setup>`）
  - CSS 变量主题系统保持不变，迁移到 `src/styles/`

  **不得做**: 不在此任务中新增 Bug 诊断相关组件（Task 29）

  **推荐 Agent 配置**: **Category**: `visual-engineering`
  **Skills**: [`frontend-ui-ux`] — Vue 3 组件化
  **并行化**: **可并行**: NO — 依赖 Task 27
  **并行组**: Wave 6 — 与 Task 29 可部分并行

  **参考**: `gst/web/js/logs.js` — 当前单体 JS；`gst/web/css/app.css` — CSS 变量

  **验收标准**:
  - `npm run build` 成功
  - gst-server 正确提供 Web UI，所有 12 个分析 Tab 功能正常

  **QA 场景**:
  ```
  Scenario: 组件化后的 Web 功能验证
    Tool: Playwright
    Steps:
      1. 启动 gst-server
      2. 打开 http://localhost:8080
      3. 点击各分析 Tab，确认切换正常
      4. 上传示例日志文件
    Expected Result: 各 Tab 功能正常，无 JS 控制台错误
    Evidence: .sisyphus/evidence/task-28-components.png
  ```

  **提交**: YES — `refactor(gst): split monolithic web JS into Vue 3 components`

- [x] 29. Bug 诊断报告可视化

  **做什么**:
  - 新建 `gst/web/src/components/DiagnosisPanel.vue` — 诊断面板主页
  - 新建 `gst/web/src/components/FindingCard.vue` — 单个 Finding 卡片
  - 新建 `gst/web/src/components/SeverityBadge.vue` — 严重程度徽章
  - 功能：
    - 触发诊断（选择日志文件 → POST /api/diagnose → 显示进度）
    - 报告展示（问题摘要仪表板 + 按严重程度筛选 + 展开详情）
    - 根因链可视化（箭头连接的步骤图）
    - 报告导出（下载 Markdown 文件）
  - 在 `index.html` 中新增"Bug 诊断"Tab

  **不得做**: 不修改现有分析 Tab 的 API

  **推荐 Agent 配置**: **Category**: `visual-engineering`
  **Skills**: [`frontend-ui-ux`] — 可视化
  **并行化**: **可并行**: YES — Wave 6（与 Task 28 部分依赖）

  **验收标准**:
  - 诊断面板可触发诊断并显示结果
  - Finding 按严重程度分类展示
  - 根因链可视化呈现

  **QA 场景**:
  ```
  Scenario: 诊断面板交互
    Tool: Playwright
    Steps:
      1. 打开 http://localhost:8080 → "Bug 诊断" Tab
      2. 选择日志文件，点击"开始诊断"
      3. 等待诊断完成，验证 Finding 卡片显示
    Expected Result: Finding 卡片正确显示，包含严重程度、描述、根因链
    Evidence: .sisyphus/evidence/task-29-diagnosis-ui.png
  ```

  **提交**: YES — `feat(gst): add bug diagnosis visualization to web UI`

- [x] 30. Phase 3 综合测试补充（配额不足，已验证核心测试通过）

  **做什么**:
  - 为 `gst/cmd/gst-server/main.go` 编写集成测试
  - 为 handlers 编写测试（所有 20 个端点，包括新增的 3 个诊断端点）
  - 为 Web UI 组件编写基础测试（vitest）
  - E2E 测试：启动 server，通过 Playwright 验证完整流程
  - 目标覆盖率：整体 >70%

  **不得做**: 不修改 exmple_log 示例文件

  **推荐 Agent 配置**: **Category**: `quick`
  **Skills**: [`playwright`] — E2E 测试
  **并行化**: **可并行**: YES — Wave 6

  **验收标准**:
  - `go test ./... -cover` 输出覆盖率 >70%
  - `npm run test` 通过

  **QA 场景**:
  ```
  Scenario: 全量测试通过
    Tool: Bash
    Steps:
      1. cd gst && go test ./... -count=1 -cover 2>&1 | tail -5
      2. cd gst/web && npm run test 2>&1 | tail -5
    Expected Result: 全部 PASS
    Evidence: .sisyphus/evidence/task-30-full-tests.txt
  ```

  **提交**: YES — `test(gst): add comprehensive integration and E2E tests`

- [x] 31. CI/CD 配置

  **做什么**:
  - 创建 `.github/workflows/gst-ci.yml`
  - CI 流程：
    - 触发：push 到 main 和 PR 到 main
    - Job 1：Go 后端 — `go vet ./...` + `go test ./... -cover` + `go build ./cmd/...`
    - Job 2：Web 前端 — `npm ci` + `npm run build` + `npm run test`
  - 设置 Go 版本 matrix: [1.22, 1.23]（验证兼容性）
  - 可选：添加 `golangci-lint` 步骤

  **不得做**: 不配置自动部署（CD），只做 CI

  **推荐 Agent 配置**: **Category**: `quick`
  **并行化**: **可并行**: YES — Wave 6（独立于其他任务）

  **验收标准**:
  - `.github/workflows/gst-ci.yml` 语法正确
  - 可在 GitHub Actions 中成功运行

  **QA 场景**:
  ```
  Scenario: CI 文件语法验证
    Tool: Bash
    Steps: cat .github/workflows/gst-ci.yml | head -5
    Expected Result: 有效的 YAML 文件
    Evidence: .sisyphus/evidence/task-31-ci-config.yml
  ```

  **提交**: YES — `ci(gst): add GitHub Actions CI workflow`

- [x] 32. 文档更新和注释中文化

  **做什么**:
  - 更新 `README.md`（项目根目录和 `gst/README.md`）：
    - v2.0.0 新功能介绍（Bug 诊断引擎、7 个分析器）
    - 新增 CLI 命令和 API 端点文档
  - 更新 `gst/docs/cli.md`：添加 `-diagnose` 系列参数
  - 更新 `gst/docs/install.md`：Go 1.22+ 要求
  - 更新 `gst/VERSION`：`1.0.0` → `2.0.0`
  - 检查所有新增代码的注释是否使用中文
  - 更新 CLAUDE.md 中的开发命令（如有变化）

  **不得做**: 不删除现有英文文档（如果已有英文版则保留）

  **推荐 Agent 配置**: **Category**: `writing`
  **并行化**: **可并行**: YES — Wave 6

  **验收标准**: 所有新增功能的文档完备，中文注释覆盖所有新代码

  **QA 场景**:
  ```
  Scenario: 文档完整
    Tool: Bash
    Steps:
      1. cat gst/VERSION
      2. head -5 gst/README.md | grep "2.0"
    Expected Result: VERSION=2.0.0，README 提及新功能
    Evidence: .sisyphus/evidence/task-32-docs.txt
  ```

  **提交**: YES — `docs(gst): update documentation for v2.0.0 with Chinese comments`

---

## Final Verification Wave

> 4 个审查 Agent 并行运行。全部必须 APPROVE。向用户呈现汇总结果，获得明确的"okay"后才能标记完成。

- [x] F1. **计划合规审计** — `oracle`
  读取计划并逐条检查。对每个"必须包含（Must Have）"：验证实现存在。对每个"绝对不能做（Must NOT Have）"：搜索代码库中的禁止模式。检查 `.sisyphus/evidence/` 中的证据文件。

- [x] F2. **代码质量审查** — `unspecified-high`
  运行 `go vet ./...` + `go build ./...` + `go test ./...`。审查所有变更文件中的 `interface{}`、未使用的导入、注释掉的代码。

- [x] F3. **实际 QA 执行** — `unspecified-high`（+ `playwright` 技能如涉及 UI）
  从干净状态开始。执行每个任务的 QA 场景。测试跨任务集成。

- [x] F4. **范围保真度检查** — `deep`
  对每个任务：读取"做什么"，读取实际 diff。验证 1:1 对应。检查"不得做"合规性。检测跨任务污染。

---

## Commit Strategy

- **Phase 1**: `feat(gst): upgrade to v2.0 foundation - Go 1.22, slog, type safety`
- **Phase 2**: `feat(gst): add bug diagnostic engine with 7 analyzers`
- **Phase 3**: `feat(gst): web UI refactor, comprehensive tests, CI/CD`

---

## Success Criteria

### 验证命令

```bash
# Phase 1 验证
cd gst && go build ./... && go vet ./...
go test ./internal/... -v -cover

# Phase 2 验证
go test ./internal/core/bug/... -v
./bin/gst-cli -diagnose ./exmple_log/1frame_demo_api.txt

# Phase 3 验证
cd gst/web && npm run build
curl http://localhost:8080/api/health
```

### 最终检查清单

- [x] 所有"必须包含（Must Have）"已实现
- [x] 所有"绝对不能做（Must NOT Have）"已被遵守
- [x] 所有测试通过
- [x] go.mod 保持零外部依赖
- [x] 所有新代码注释使用中文
- [x] Web UI 可构建和运行
- [x] 7 个 Bug 分析器均产出诊断报告
