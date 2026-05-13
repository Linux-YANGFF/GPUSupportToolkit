---
last_mapped_commit: 0e48e4f
analysis_date: 2026-05-09
focus: quality
---

# 编码规范

**分析日期:** 2026-05-09
**最新 Commit:** 0e48e4f

## 语言与版本

- **Go 1.22** (`gst/go.mod`)
- 不使用泛型以外的 Go 1.22 新特性

## 命名规范

### 文件
- Go 源文件使用 `snake_case` 命名，如 `raw_trace_parser.go`
- 测试文件严格使用 `*_test.go` 后缀，与源文件同包或 `xxx_test` 包
- 示例：`cmd/cli/main_test.go` 使用 `package main_test`（黑盒测试），其余测试文件使用同包名（白盒测试）

### 包名
- 包名与目录名一致，全小写，无下划线
- 示例：`internal/core/parser` -> `package parser`
- 示例：`internal/core/bug` -> `package bug`
- 入口包使用 `package main`

### 类型与接口
- 导出类型使用 **PascalCase**
- 示例：`StreamReader`、`APILogEntry`、`FrameInfo`、`ParsedLog`
- 接口名以功能描述为准，无 `I` 前缀或 `er` 后缀强制要求

### 函数
- 导出函数使用 **PascalCase**
- 非导出函数使用 **camelCase**
- 构造函数统一使用 `New` + 类型名前缀：
  - `NewStreamReader` (`internal/platform/file_reader.go`)
  - `NewFrameAnalyzer` (`internal/core/analyzer/frame_analyzer.go`)
  - `NewGLStateTracker` (`internal/core/bug/state_tracker.go`)
  - `NewDefaultRegistry` (`internal/core/bug/report.go`)

### 变量与常量
- 局部变量使用 **camelCase**
- 包级常量使用 **PascalCase**（导出）或全大写蛇形（非导出）
- 示例：`StreamBufferSize = 64 * 1024 * 1024` (`internal/platform/file_reader.go`)
- 示例：`IndexMagicNumber uint32 = 0x47535449`

### 结构体字段
- 导出字段使用 **PascalCase**
- JSON 标签使用 `snake_case`，omitempty 按需使用
- 示例：`GCAddr string `json:"gc_addr,omitempty"`` (`internal/core/types.go`)

## 代码风格

### 格式化
- 使用 `go fmt` 作为唯一格式化工具
- 无 `gofmt` 替代工具（如 `gofumpt`）
- 行宽无硬性限制，但保持自然换行

### 代码检查
- 使用 `go vet ./...` 进行静态分析
- 无 ESLint/Prettier/Biome 等 JS 工具链（前端目录 `gst/web/` 使用 Vite，但无 lint 配置）

## 导入组织

### 导入顺序
1. 标准库
2. 第三方库（本项目无外部依赖）
3. 项目内部包（`gst/internal/...`）

### 路径别名
- 项目模块名为 `gst`，内部导入直接使用完整模块路径：
  ```go
  import "gst/internal/core"
  import "gst/internal/platform"
  ```
- 无自定义路径别名

## 错误处理

### 错误返回模式
- 优先返回 `error`，不 panic
- 使用 `fmt.Errorf` 包装错误，带上下文：
  ```go
  // internal/platform/file_reader.go
  return nil, fmt.Errorf("failed to open file %s: %w", filePath, err)
  ```
- 使用 `%w` 动词保留原始错误链

### 错误类型
- 无自定义 error 类型，均使用标准 `error`
- 无 `errors.Is` / `errors.As` 的显式使用模式

### 空值处理
- 对 `nil` 输入进行防御性检查：
  ```go
  // internal/core/analyzer/frame_analyzer.go
  func NewFrameAnalyzer(log *core.ParsedLog) *FrameAnalyzer {
      if log == nil {
          return nil
      }
  ```

## 日志

### 框架
- 使用 Go 标准库 `log/slog`（Go 1.21+）
- 全局 Logger 为包级变量：`var Logger *slog.Logger` (`internal/platform/logger.go`)

### 使用模式
- 通过 `platform.InitLogger(level)` 初始化，支持 `debug/info/warn/error`
- 结构化日志：
  ```go
  slog.Debug("ParseLog request", "method", r.Method, "remote", r.RemoteAddr)
  ```
- 生产环境默认级别为 `warn`，调试使用 `debug`

## 注释

### 语言
- 导出符号使用中文注释
- 示例：`// LogIndex 日志索引结构` (`internal/platform/file_reader.go`)
- 示例：`// StreamReader 大文件流式读取器`

### 文档注释
- 导出函数/类型必须有注释，以名称开头
- 无 Godoc 特殊格式（如 `Deprecated:` 标记）

## 函数设计

### 大小
- 函数保持适中，单一职责
- 解析器、分析器、检测器均按功能拆分为独立文件

### 参数
- 优先使用值接收器或指针接收器，保持一致
- 配置项使用 flag 包在 `main` 中解析后传入

### 返回值
- 构造函数返回指针类型
- 可能失败的操作返回 `(T, error)` 对

## 模块设计

### 导出模式
- 每个包的核心类型/函数直接导出
- 无 `internal` 子包再封装（`internal/` 已是 Go 的访问控制边界）

### Barrel 文件
- 无 barrel/index 文件，每个文件独立导出
- 跨包引用直接导入目标包

## 并发

### 模式
- 使用 goroutine + channel 实现流式读取：
  ```go
  // internal/platform/file_reader.go
  ch := make(chan string, 1000)
  go func() {
      defer close(ch)
      // ...
  }()
  ```
- HTTP Handler 使用 `sync.RWMutex` 保护共享状态 (`cmd/gst-server/internal/handlers/handlers.go`)

## 测试辅助规范

### 测试辅助函数
- 使用 `t.Helper()` 标记辅助函数（仅 1 处使用，在 `file_reader_test.go`）
- 使用 `t.TempDir()` 创建临时目录（8 处使用）
- 使用 `t.Cleanup()` 注册清理逻辑

---

*规范分析: 2026-05-09*
