---
last_mapped_commit: 0e48e4f
analysis_date: 2026-05-09
focus: quality
---

# 测试模式

**分析日期:** 2026-05-09

## 测试框架

**运行器:**
- Go 标准库 `testing` (无外部框架)
- 配置: 无额外配置文件，使用 `go test` 默认行为

**断言库:**
- 无 testify/assert 等外部库
- 使用标准 `if` + `t.Errorf` / `t.Fatalf` 模式

**运行命令:**
```bash
go test ./... -v                    # 运行所有测试
go test ./internal/core/... -v      # 运行核心模块测试
make test                           # Makefile 封装
go test -coverprofile=coverage.out ./...  # 覆盖率
```

## 测试文件组织

**位置:**
- 测试文件与被测代码同包或 `_test` 包
- 同包测试: `internal/platform/file_reader_test.go` (package `platform`)
- 外部测试包: `cmd/cli/main_test.go` (package `main_test`)

**命名:**
- `*_test.go` 后缀
- 测试函数 `Test` + 被测名称: `TestStreamReaderReadLines`

**结构:**
```
gst/
├── cmd/cli/main_test.go                          # CLI 集成测试
├── cmd/gst-server/internal/handlers/handlers_test.go  # HTTP Handler 测试
├── internal/core/core_test.go                    # 核心类型测试
├── internal/core/parser/parser_test.go           # 解析器测试
├── internal/core/parser/raw_trace_parser_test.go # 原始追踪解析测试
├── internal/core/exporter/exporter_test.go       # 导出器测试
├── internal/core/analyzer/analyzer_test.go       # 分析器测试
├── internal/platform/file_reader_test.go         # 文件读取测试
├── internal/platform/logger_test.go              # 日志测试
└── internal/core/bug/                            # Bug 诊断测试
    └── testdata/sample_trace.log                 # 测试数据
```

## 测试结构

**套件组织:**
```go
func TestStreamReaderReadLines(t *testing.T) {
    tests := []struct {
        name        string
        content     string
        expectLines int
    }{
        {"three lines", "line1\nline2\nline3\n", 3},
        {"single line", "only one line\n", 1},
        {"empty content", "", 0},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            path := createTempFile(t, tt.content)
            sr, err := NewStreamReader(path)
            if err != nil {
                t.Fatalf("NewStreamReader: %v", err)
            }
            defer sr.Close()

            count := 0
            for range sr.ReadLines() {
                count++
            }

            if count != tt.expectLines {
                t.Errorf("expected %d lines, got %d", tt.expectLines, count)
            }
        })
    }
}
```

**模式:**
- 表驱动测试 (Table-Driven Tests): 定义 `tests := []struct{...}` 切片
- 子测试: 使用 `t.Run(tt.name, func(t *testing.T){...})`
- 临时资源: 使用 `t.TempDir()` 创建临时目录，`t.Cleanup()` 注册清理函数
- 辅助函数: `createTempFile(t *testing.T, content string) string`，标记 `t.Helper()`

## Mock 与 Stub

**框架:** 无外部 mock 框架

**模式:**
- 使用 `httptest.NewRecorder()` 模拟 HTTP 响应
- 使用临时文件模拟真实文件系统
- 使用环境变量设置/清理模拟配置

**示例:**
```go
// 来自 cmd/gst-server/internal/handlers/handlers_test.go
func TestHealth(t *testing.T) {
    handler := &Handler{}
    req := httptest.NewRequest(http.MethodGet, "/health", nil)
    w := httptest.NewRecorder()
    handler.Health(w, req)
    if w.Code != http.StatusOK {
        t.Errorf("Health status = %d, want 200", w.Code)
    }
}
```

**需要 Mock 的内容:**
- HTTP 请求/响应
- 文件系统（通过临时文件）
- 环境变量

**不需要 Mock 的内容:**
- 纯函数和结构体方法（直接实例化测试）
- 内部接口实现（使用真实实现）

## 测试数据与工厂

**测试数据:**
- 简单数据内联在测试函数中
- 复杂日志数据放在 `testdata/` 目录
- 使用 `t.TempDir()` 创建隔离的临时文件

**辅助函数示例:**
```go
func createTempFile(t *testing.T, content string) string {
    t.Helper()
    dir := t.TempDir()
    path := filepath.Join(dir, "test.log")
    t.Cleanup(func() { os.Remove(path) })
    if err := os.WriteFile(path, []byte(content), 0644); err != nil {
        t.Fatalf("failed to create temp file: %v", err)
    }
    return path
}
```

**位置:**
- 内联数据: 测试函数内部
- 文件数据: `internal/core/bug/testdata/sample_trace.log`

## 覆盖率

**要求:** 无强制覆盖率目标

**查看覆盖率:**
```bash
make test-cover   # 生成 coverage.out 和 coverage.html
go tool cover -html=coverage.out
```

## 测试类型

**单元测试:**
- 结构体默认值验证: `internal/core/core_test.go` (26 个测试)
- 函数行为验证: `internal/platform/file_reader_test.go` (22 个测试)
- 边界条件: 空输入、nil 值、无效参数

**集成测试:**
- CLI 端到端: `cmd/cli/main_test.go`
  - 编译二进制文件后执行
  - 测试文件不存在的错误处理
  - 测试 `--help` 输出

**HTTP Handler 测试:**
- `cmd/gst-server/internal/handlers/handlers_test.go` (12 个测试)
- 测试路径验证、MIME 类型、日志行匹配
- 使用 `httptest.NewRecorder()`

## 常见模式

**异步测试:**
- 使用 channel 和 `range` 读取: `for range sr.ReadLines() { ... }`
- 进度回调测试: `sr.ReadLinesWithProgress(func(p float64) { ... })`

**错误测试:**
```go
func TestNewStreamReader(t *testing.T) {
    tests := []struct {
        name    string
        path    string
        wantErr bool
    }{
        {"valid file", tempPath, false},
        {"nonexistent", "/nonexistent/path/xyz_deadbeef.log", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            sr, err := NewStreamReader(tt.path)
            if tt.wantErr {
                if err == nil {
                    t.Error("expected error, got nil")
                }
                return
            }
            if err != nil {
                t.Fatalf("unexpected error: %v", err)
            }
        })
    }
}
```

**JSON 测试:**
```go
func TestAPILogEntry_JSONTags(t *testing.T) {
    e := APILogEntry{APIName: "glBindBuffer", ...}
    data, err := json.Marshal(e)
    if err != nil {
        t.Fatalf("json.Marshal: %v", err)
    }
    if len(data) == 0 {
        t.Error("json output should not be empty")
    }
}
```

**边缘情况测试:**
- 空结构体默认值
- nil 指针处理
- 无效/损坏的文件格式（如截断的索引文件）
- 越界路径（`../etc/passwd`）

---

*测试分析: 2026-05-09*
