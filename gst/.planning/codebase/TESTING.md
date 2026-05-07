# Testing Patterns

**Analysis Date:** 2026-05-07

## Test Framework

**Runner:**
- Go standard testing package (`testing`)
- Go version: 1.18 (from `go.mod`)
- No external test runner (no testify, ginkgo, or gotestsum detected)

**Assertion Library:**
- Standard Go `t.Fatalf`, `t.Errorf`, `t.Error`, `t.Fatal`, `t.Skipf`
- No testify/assert or other assertion libraries

**Run Commands:**
```bash
go test ./... -v              # Run all tests
go test ./internal/core/... -v # Run core module tests
go vet ./...                  # Static analysis
go fmt ./...                  # Format check
```

## Test File Organization

**Location:**
- Tests are co-located with source files in the same package
- Test file naming: `{source}_test.go` (e.g., `analyzer_test.go` for `analyzer/` package)

**Naming:**
- Test functions: `Test{Type}_{Method}_{Scenario}` (e.g., `TestFrameAnalyzer_FindTopSlowFrames`)
- Benchmark functions: `Benchmark{Type}_{Method}` (e.g., `BenchmarkAPIParser_Parse`)
- Subtests use `t.Run("descriptive name", func(t *testing.T) {...})`

**Structure:**
```
internal/core/
├── parser/
│   ├── parser_test.go          # Parser tests
│   ├── api_parser.go
│   ├── profile_parser.go
│   └── raw_trace_parser.go
├── analyzer/
│   ├── analyzer_test.go        # All analyzer tests (single file)
│   ├── frame_analyzer.go
│   ├── func_analyzer.go
│   └── ... (12 analyzer files)
├── search/
│   ├── search_test.go          # Search tests
│   └── ...
└── exporter/
    ├── exporter_test.go        # Exporter tests
    └── exporter.go
```

## Test Structure

**Suite Organization:**
```go
func TestFrameAnalyzer_FindTopSlowFrames(t *testing.T) {
    log := createTestParsedLog()
    analyzer := NewFrameAnalyzer(log)

    tests := []struct {
        name   string
        n      int
        expect int
    }{
        {"top 1", 1, 1},
        {"top 2", 2, 2},
        {"top all", 10, 3},
        {"n=0", 0, 0},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := analyzer.FindTopSlowFrames(tt.n)
            if len(result) != tt.expect {
                t.Errorf("FindTopSlowFrames(n=%d) returned %d frames, want %d", tt.n, len(result), tt.expect)
            }
        })
    }
}
```

**Patterns:**
- Table-driven tests with `tests := []struct{...}{}` are the dominant pattern
- Subtests via `t.Run()` for parameterized cases
- Helper function `createTestParsedLog()` in `analyzer_test.go` builds reusable test fixtures
- Nil-input tests are standard: `Test{X}_NilLog` or `Test{X}_Nil` verify graceful handling

**Setup pattern:**
```go
func createTestParsedLog() *core.ParsedLog {
    return &core.ParsedLog{
        Frames: []core.FrameInfo{
            {
                FrameNum:    0,
                StartLine:   1,
                EndLine:     10,
                TotalTimeUs: 50000,
                APICalls: []core.APILogEntry{
                    {APIName: "glBindBuffer", Count: 10, TimeUs: 1000, LineNum: 2},
                    {APIName: "glDrawElements", Count: 5, TimeUs: 40000, LineNum: 5},
                },
            },
        },
        TotalTimeUs: 160000,
        FPS:         10.0,
    }
}
```

**Teardown pattern:**
- No explicit teardown; tests rely on garbage collection
- File handles closed with `defer file.Close()` when opening real files

**Assertion pattern:**
```go
if len(result) != tt.expect {
    t.Errorf("FindTopSlowFrames(n=%d) returned %d frames, want %d", tt.n, len(result), tt.expect)
}
```

## Mocking

**Framework:** None — no mocking library used

**Patterns:**
- Tests use handcrafted struct literals for test data
- No interface mocking; tests call real implementations
- I/O is tested against `strings.NewReader` and temporary `bytes.Buffer`

**What to Mock:**
- Not applicable — all tests use real implementations

**What NOT to Mock:**
- File I/O is tested with real files from `../exmple_log/` directory
- Parsers are tested with inline string data or sample log files

## Fixtures and Factories

**Test Data:**
- Inline string literals for parser inputs
- Helper function `createTestParsedLog()` in `analyzer_test.go`
- Real sample files: `/root/code/GPUSupportToolkit/GPUSupportToolkit/exmple_log/1frame_demo_api.txt` and `1frame_profile_demo.txt`

**Location:**
- Inline fixtures in test files
- External sample logs in `../exmple_log/` (repo-relative)

## Coverage

**Requirements:** None enforced

**View Coverage:**
```bash
go test ./... -cover
```

## Test Types

**Unit Tests:**
- All existing tests are unit tests
- Scope: individual analyzer methods, parser functions, exporter formats
- No database or network dependencies

**Integration Tests:**
- Parser file-based tests (`TestAPIParser_ParseFromFile`, `TestRawTraceParser_ParseFromFile`) read real log files
- These serve as lightweight integration tests for the parser pipeline

**E2E Tests:**
- Not used
- `cmd/functionaltest/main.go` exists but is not a formal test suite

## Common Patterns

**Async Testing:**
- Not applicable — no goroutines tested

**Error Testing:**
```go
func TestCSVExporter_Export_UnsupportedType(t *testing.T) {
    var buf bytes.Buffer
    exporter := CSVExporter{Data: "unsupported"}
    err := exporter.Export(&buf)
    if err == nil {
        t.Error("CSVExporter should fail for unsupported type")
    }
}
```

**File-based testing:**
```go
func TestAPIParser_ParseFromFile(t *testing.T) {
    file, err := os.Open("/root/code/GPUSupportToolkit/GPUSupportToolkit/exmple_log/1frame_profile_demo.txt")
    if err != nil {
        t.Skipf("Skipping file test: %v", err)
    }
    defer file.Close()
    // ... test logic
}
```

**Benchmarks:**
```go
func BenchmarkAPIParser_Parse(b *testing.B) {
    content, err := os.ReadFile("/root/code/GPUSupportToolkit/GPUSupportToolkit/exmple_log/1frame_profile_demo.txt")
    if err != nil {
        b.Skipf("Skipping benchmark: %v", err)
    }
    parser := &APIParser{}
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, err := parser.Parse(strings.NewReader(string(content)))
        if err != nil {
            b.Fatalf("Parse failed: %v", err)
        }
    }
}
```

---

*Testing analysis: 2026-05-07*
