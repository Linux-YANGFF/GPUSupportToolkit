# Coding Conventions

**Analysis Date:** 2026-05-07

## Naming Patterns

**Files:**
- Go source files use `snake_case.go` (e.g., `frame_analyzer.go`, `buffer_analyzer.go`)
- Test files use `snake_case_test.go` (e.g., `analyzer_test.go`, `parser_test.go`)
- One primary type per file, named after the file (e.g., `frame_analyzer.go` contains `FrameAnalyzer`)

**Functions:**
- Constructors use `New{Type}` pattern (e.g., `NewFrameAnalyzer`, `NewBufferAnalyzer`, `NewKeywordSearch`)
- Methods on structs use receiver name matching first letter of type in lowercase (e.g., `(fa *FrameAnalyzer)`, `(ba *BufferAnalyzer)`)
- Private helper functions use `camelCase` (e.g., `shouldSkipLine`, `isFrameBoundary`, `extractSizeFromParams`)
- Package-level exported functions use `PascalCase` (e.g., `DetectKind`, `CreateParserAuto`, `ExportSearchResults`)

**Variables:**
- Local variables use `camelCase` (e.g., `totalUs`, `frameSummaries`, `allMatches`)
- Constants use `PascalCase` for exported, `camelCase` for unexported (e.g., `TargetArrayBuffer`, `UsageStaticDraw`)
- Enum-like string types use `PascalCase` for type and values (e.g., `SwapClass` with `SwapVSyncBound`, `SwapGPULimited`)
- Map variables use descriptive names ending in map/hint/pattern (e.g., `BufferUsageHint`, `BufferUsagePattern`)

**Types:**
- Structs use `PascalCase` (e.g., `FrameInfo`, `APILogEntry`, `FuncStats`)
- Interfaces use `PascalCase` noun names (e.g., `Parser`, `Exporter`)
- JSON tags use `snake_case` with `omitempty` where appropriate (e.g., ``json:"frame_num"``, ``json:"CommandLine,omitempty"``)

## Code Style

**Formatting:**
- Standard `gofmt` formatting (no custom `.gofmt` or `golangci-lint` config detected)
- `go fmt ./...` is the documented formatting command
- Chinese comments are used extensively for documentation and inline explanations

**Linting:**
- `go vet ./...` is the documented linting command
- No `.golangci.yml` or custom lint configuration detected
- Standard Go vet rules apply

## Import Organization

**Order:**
1. Standard library imports
2. Blank line
3. Internal module imports (`gst/internal/...`)

**Example from `internal/core/exporter/exporter.go`:**
```go
import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"gst/internal/core"
)
```

**Path Aliases:**
- No custom path aliases used
- Module name is `gst` (declared in `go.mod`)

## Error Handling

**Patterns:**
- Errors are returned explicitly and checked immediately
- `t.Fatalf` is used in tests for setup failures that prevent test continuation
- `t.Errorf` is used for assertion failures that should not stop the test
- HTTP handlers use `http.Error()` with appropriate status codes
- CLI prints errors to stdout with Chinese prefix "错误:" and returns early

**Example from tests:**
```go
parsed, err := parser.Parse(strings.NewReader(input))
if err != nil {
    t.Fatalf("Parse failed: %v", err)
}
```

**Example from handlers:**
```go
if current == nil {
    http.Error(w, "No log parsed", http.StatusBadRequest)
    return
}
```

**Nil safety:**
- Analyzers check for nil `*core.ParsedLog` at entry points and return nil/empty results
- Constructors do not validate nil inputs (callers expected to provide valid data)

## Logging

**Framework:** Standard library `log` package

**Patterns:**
- Server uses `log.Printf` and `log.Println` for operational messages
- CLI uses `fmt.Printf` for user-facing output (Chinese language)
- No structured logging framework (no zap, logrus, or slog detected)
- No log levels configured

**Server logging locations:**
- `cmd/gst-server/main.go` — startup, shutdown, browser open
- No request logging middleware detected

## Comments

**When to Comment:**
- Every exported type and function has a Chinese comment explaining its purpose
- Complex parsing logic has inline Chinese comments
- HTTP handler comments include route path and method (e.g., `// R1: ParseLog handles POST /api/log/parse`)

**JSDoc/TSDoc:**
- Not applicable (Go project)
- Go doc comments follow standard `//` format, written in Chinese

## Function Design

**Size:**
- Functions are generally short (under 50 lines)
- Handler methods in `handlers.go` are longer (~20-40 lines each) due to HTTP boilerplate
- Parser `Parse()` methods are longer (~100-200 lines) due to line-by-line processing

**Parameters:**
- Prefer value receivers for stateless structs (e.g., `TXTExporter`)
- Prefer pointer receivers for mutable structs or large structs (e.g., `*FrameAnalyzer`, `*Handler`)
- `io.Reader` / `io.Writer` used for I/O abstractions

**Return Values:**
- `(result, error)` pattern for functions that can fail
- `nil` returned for empty/not-found cases in analyzers
- HTTP handlers return void and write directly to `http.ResponseWriter`

## Module Design

**Exports:**
- Types and functions intended for cross-package use are exported (`PascalCase`)
- Package-internal helpers are unexported (`camelCase`)
- `internal/` path prefix prevents external imports of core packages

**Barrel Files:**
- No barrel/index files used
- Each package exposes its types directly from individual files
- `core` package (`internal/core/types.go`) acts as a shared type definitions module

---

*Convention analysis: 2026-05-07*
