# Codebase Structure

**Analysis Date:** 2026-05-07

## Directory Layout

```
/root/code/GPUSupportToolkit/GPUSupportToolkit/gst/
├── cmd/                    # Application entry points
│   ├── cli/                # CLI binary (gst-cli)
│   ├── functionaltest/     # Smoke-test binary
│   └── gst-server/         # HTTP server binary (gst-server)
│       └── internal/
│           └── handlers/   # HTTP handlers (not importable)
├── internal/               # Private application code
│   ├── core/               # Domain logic
│   │   ├── analyzer/       # Frame / function / shader / buffer analyzers
│   │   ├── exporter/       # TXT / CSV / JSON export
│   │   ├── parser/         # Log parsers (apitrace, profile, raw trace)
│   │   ├── search/         # Keyword and time-range search
│   │   └── types.go        # Shared domain types
│   └── platform/           # OS and file I/O abstractions
├── web/                    # Static frontend assets (Vue.js SPA)
│   ├── css/
│   ├── js/
│   ├── index.html
│   └── logs.html
├── skills/                 # Agent skill definitions
│   └── gpu-log-analyst/
│       ├── SKILL.md
│       ├── README.md
│       └── INDEX.md
├── assets/                 # Static project assets
├── docs/                   # Documentation
├── packaging/              # Debian / RPM packaging files
├── bin/                    # Build output (binaries, .deb)
├── Makefile                # Build, test, package automation
├── go.mod                  # Go module (module gst, Go 1.18)
├── go.sum                  # Dependency checksums
├── VERSION                 # Current version string
└── README.md
```

## Directory Purposes

**`cmd/`:**
- Purpose: Contains `main` packages for each executable
- Contains: `cli/main.go`, `gst-server/main.go`, `functionaltest/main.go`
- Key files: `cmd/gst-server/internal/handlers/handlers.go`

**`internal/core/`:**
- Purpose: All domain logic — parsing, analysis, search, export
- Contains: Sub-packages by concern; `types.go` at root for shared structs
- Key files: `internal/core/types.go`, `internal/core/parser/parser.go`

**`internal/core/analyzer/`:**
- Purpose: Specialized analyzers that operate on `*core.ParsedLog`
- Contains: One file per analysis dimension (frame, func, shader, buffer, draw call, state change, sync stall, swap, texture, FBO, redundancy, bandwidth)
- Key files: `frame_analyzer.go`, `func_analyzer.go`, `buffer_analyzer.go`, `draw_call_analyzer.go`

**`internal/core/parser/`:**
- Purpose: Log format detection and parsing
- Contains: `parser.go` (interface + factory), `api_parser.go`, `raw_trace_parser.go`, `profile_parser.go`
- Key files: `parser.go`, `api_parser.go`

**`internal/core/search/`:**
- Purpose: Keyword and time-range search implementations
- Contains: `keyword_search.go`, `time_range_search.go`, `index.go`
- Key files: `keyword_search.go`

**`internal/core/exporter/`:**
- Purpose: Export analysis results to TXT/CSV/JSON
- Contains: `exporter.go` (interface + implementations)
- Key files: `exporter.go`

**`internal/platform/`:**
- Purpose: OS abstraction and file utilities
- Contains: `file_reader.go` (streaming reader + index), `os_detector.go`
- Key files: `file_reader.go`

**`web/`:**
- Purpose: Static frontend served by `gst-server`
- Contains: `index.html`, `logs.html`, `vue.global.js`, `logs.js`, `app.css`
- Key files: `logs.html`, `logs.js`

**`skills/`:**
- Purpose: Agent skill metadata for DeerFlow / Claude Code
- Contains: `gpu-log-analyst/SKILL.md`
- Key files: `skills/gpu-log-analyst/SKILL.md`

## Key File Locations

**Entry Points:**
- `cmd/gst-server/main.go`: HTTP server entry point
- `cmd/cli/main.go`: CLI entry point
- `cmd/functionaltest/main.go`: Smoke-test entry point

**Configuration:**
- `go.mod`: Module definition (Go 1.18, no external dependencies)
- `Makefile`: Build, test, lint, packaging targets
- `VERSION`: Single-line version string read by Makefile

**Core Logic:**
- `internal/core/types.go`: Central domain model
- `internal/core/parser/parser.go`: Parser interface and auto-detection
- `internal/core/parser/api_parser.go`: Primary parser implementation
- `internal/core/analyzer/frame_analyzer.go`: Frame statistics
- `internal/core/analyzer/func_analyzer.go`: Function statistics
- `internal/core/analyzer/buffer_analyzer.go`: Buffer object tracking
- `cmd/gst-server/internal/handlers/handlers.go`: HTTP API orchestration

**Testing:**
- `internal/core/parser/parser_test.go`: Parser unit tests
- `internal/core/analyzer/analyzer_test.go`: Analyzer unit tests
- `internal/core/search/search_test.go`: Search unit tests
- `internal/core/exporter/exporter_test.go`: Exporter unit tests

## Naming Conventions

**Files:**
- Analyzer files: `{concern}_analyzer.go` (e.g., `draw_call_analyzer.go`)
- Test files: `{name}_test.go` (e.g., `parser_test.go`)
- Parser files: `{format}_parser.go` (e.g., `api_parser.go`)

**Directories:**
- Package directories match Go package names (`parser`, `analyzer`, `search`, `exporter`)
- `internal/` prefix indicates non-importable code outside module

**Types:**
- Structs: PascalCase, often suffixed with analyzer name (e.g., `DrawCallAnalyzer`)
- Constructor: `New{Type}(log *core.ParsedLog) *{Type}`
- Methods: `Analyze()` for primary computation, `GetSummary()` for aggregated stats

## Where to Add New Code

**New Analyzer:**
- Implementation: `internal/core/analyzer/{concern}_analyzer.go`
- Types: Add per-frame stats struct to `internal/core/types.go`
- HTTP endpoint: Add handler method in `cmd/gst-server/internal/handlers/handlers.go`
- Route registration: Add `mux.HandleFunc` in `cmd/gst-server/main.go`
- Comprehensive report: Include in `Handler.AnalyzeComprehensive`

**New Parser Format:**
- Implementation: `internal/core/parser/{format}_parser.go`
- Register: Update `LogKind` consts and `CreateParser` in `internal/core/parser/parser.go`
- Detection: Update `detectFromLine` and `DetectKindFromReader`

**New Export Format:**
- Implementation: Add struct implementing `Exporter` in `internal/core/exporter/exporter.go`
- Register: Update `ExportAnalysisResult` and handler export switch

**New Search Feature:**
- Implementation: `internal/core/search/{feature}.go`
- Wire into handler in `cmd/gst-server/internal/handlers/handlers.go`

**New CLI Command:**
- Flag: Add to `cmd/cli/main.go` flag block
- Logic: Add function in `cmd/cli/main.go`

## Special Directories

**`bin/`:**
- Purpose: Build artifacts and generated packages
- Generated: Yes (by Makefile)
- Committed: No (in `.gitignore`)

**`cmd/gst-server/internal/`:**
- Purpose: Handler code private to the server binary
- Generated: No
- Committed: Yes
- Note: Go enforces that `internal/` is not importable by modules outside the tree

**`packaging/`:**
- Purpose: Debian and RPM packaging metadata
- Generated: No
- Committed: Yes

---

*Structure analysis: 2026-05-07*
