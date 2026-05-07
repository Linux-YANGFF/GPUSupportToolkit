<!-- refreshed: 2026-05-07 -->
# Architecture

**Analysis Date:** 2026-05-07

## System Overview

```text
┌─────────────────────────────────────────────────────────────┐
│                      Presentation Layer                      │
│  ┌──────────────┐  ┌─────────────────────────────────────┐  │
│  │   CLI Tool   │  │         Web UI (Vue.js SPA)         │  │
│  │`cmd/cli/`    │  │`web/`                               │  │
│  └──────┬───────┘  └──────────────────┬──────────────────┘  │
└─────────┼─────────────────────────────┼─────────────────────┘
          │                             │ HTTP
          ▼                             ▼
┌─────────────────────────────────────────────────────────────┐
│                    HTTP API / Transport Layer                │
│              `cmd/gst-server/internal/handlers/`             │
│                                                              │
│  Routes: /api/log/parse | /api/log/frames | /api/log/search │
│          /api/log/analyze/* | /api/log/export | /health      │
└─────────────────────────────────────────────────────────────┘
          │
          ▼
┌─────────────────────────────────────────────────────────────┐
│                      Core Domain Layer                       │
│                    `internal/core/`                          │
│  ┌──────────────┬──────────────┬──────────────┬───────────┐ │
│  │   Parser     │   Analyzer   │    Search    │ Exporter  │ │
│  │`parser/`     │`analyzer/`   │`search/`     │`exporter/`│ │
│  └──────┬───────┘──────┬───────┘──────┬───────┘─────┬─────┘ │
└─────────┼──────────────┼──────────────┼─────────────┼───────┘
          │              │              │             │
          ▼              ▼              ▼             ▼
┌─────────────────────────────────────────────────────────────┐
│                    Shared Model (`internal/core/types.go`)   │
│  ParsedLog → []FrameInfo → []APILogEntry / ShaderInfo / ... │
└─────────────────────────────────────────────────────────────┘
          │
          ▼
┌─────────────────────────────────────────────────────────────┐
│              Platform Abstraction (`internal/platform/`)     │
│         StreamReader · OS Detection · File Indexing          │
└─────────────────────────────────────────────────────────────┘
```

## Component Responsibilities

| Component | Responsibility | File |
|-----------|----------------|------|
| `APIParser` | Parse aggregated apitrace format (`count=`, `time=`) | `internal/core/parser/api_parser.go` |
| `RawTraceParser` | Parse raw apiTrace format (one API call per line) | `internal/core/parser/raw_trace_parser.go` |
| `ProfileParser` | Thin wrapper over `APIParser` for profile logs | `internal/core/parser/profile_parser.go` |
| `FrameAnalyzer` | Find top-N slow frames and compute frame summaries | `internal/core/analyzer/frame_analyzer.go` |
| `FuncAnalyzer` | Aggregate function call counts and timing across all frames | `internal/core/analyzer/func_analyzer.go` |
| `ShaderAnalyzer` | Compile/source/create statistics per frame and globally | `internal/core/analyzer/shader_analyzer.go` |
| `BufferAnalyzer` | Track buffer lifetimes, targets, sizes, usage patterns | `internal/core/analyzer/buffer_analyzer.go` |
| `DrawCallAnalyzer` | Count draw calls per frame and primitive-type histograms | `internal/core/analyzer/draw_call_analyzer.go` |
| `StateChangeAnalyzer` | State-change-to-draw ratio and redundant bind detection | `internal/core/analyzer/state_change_analyzer.go` |
| `SyncStallAnalyzer` | Explicit (`glFinish`) and inferred sync stall detection | `internal/core/analyzer/sync_stall_analyzer.go` |
| `SwapClassifier` | Classify frames as VSyncBound / GPULimited / CPULimited | `internal/core/analyzer/swap_classifier.go` |
| `TextureUploadAnalyzer` | Per-frame texture upload counts and format breakdown | `internal/core/analyzer/texture_upload_analyzer.go` |
| `FBOAnalyzer` | FBO bind counts and attachment-change tracking | `internal/core/analyzer/fbo_analyzer.go` |
| `RedundancyAnalyzer` | Detect redundant state calls within a single frame | `internal/core/analyzer/redundancy_analyzer.go` |
| `BandwidthAnalyzer` | Estimate upload bandwidth from `glBufferData` / `glTexImage2D` | `internal/core/analyzer/bandwidth_analyzer.go` |
| `KeywordSearch` | Regex and simple substring search over raw lines | `internal/core/search/keyword_search.go` |
| `TimeRangeSearch` | Filter frames by time range or frame number range | `internal/core/search/time_range_search.go` |
| `KeywordIndex` | In-memory inverted index of API names to line numbers | `internal/core/search/index.go` |
| `Exporter` | TXT / CSV / JSON export with type-switch formatting | `internal/core/exporter/exporter.go` |
| `Handler` | HTTP request routing, request parsing, response encoding | `cmd/gst-server/internal/handlers/handlers.go` |
| `StreamReader` | Large-file streaming with BOM detection and progress hooks | `internal/platform/file_reader.go` |

## Pattern Overview

**Overall:** Parser-Analyzer-Exporter Pipeline with Strategy-pattern analyzers

**Key Characteristics:**
- Immutable input: `ParsedLog` is built once and shared read-only across all analyzers
- Each analyzer is a standalone struct accepting `*core.ParsedLog` and producing domain-specific stats
- HTTP handlers orchestrate analyzers and format responses; they do not contain business logic
- Parser auto-detection selects the correct strategy (`KindAPITrace`, `KindProfile`, `KindRawTrace`)

## Layers

**Presentation Layer:**
- Purpose: User-facing interfaces (CLI flags, web SPA)
- Location: `cmd/cli/`, `web/`
- Contains: Flag parsing, HTTP client calls, static HTML/JS/CSS
- Depends on: Nothing (CLI is a standalone binary; web UI talks to server via HTTP)
- Used by: End users

**Transport / API Layer:**
- Purpose: HTTP routing, request validation, JSON serialization, file upload handling
- Location: `cmd/gst-server/internal/handlers/`
- Contains: `Handler` struct with `sync.RWMutex`, route methods, response DTOs
- Depends on: `internal/core/*` packages
- Used by: Web UI, external HTTP clients, skill agents

**Core Domain Layer:**
- Purpose: Log parsing, analysis algorithms, search, export formatting
- Location: `internal/core/parser/`, `internal/core/analyzer/`, `internal/core/search/`, `internal/core/exporter/`
- Contains: Parsers, analyzers, search implementations, exporter implementations
- Depends on: `internal/core/types.go` (shared model)
- Used by: Transport layer, CLI, functional tests

**Shared Model:**
- Purpose: Central domain types consumed by all core packages
- Location: `internal/core/types.go`
- Contains: `ParsedLog`, `FrameInfo`, `APILogEntry`, `ShaderInfo`, `FuncStats`, and per-analysis stat structs
- Depends on: Nothing
- Used by: All `internal/core/*` packages, `cmd/cli`, `cmd/gst-server`

**Platform Layer:**
- Purpose: OS abstraction, file I/O utilities, indexing
- Location: `internal/platform/`
- Contains: `StreamReader`, `LogIndex`, OS detection helpers
- Depends on: Standard library only
- Used by: Currently imported but not heavily utilized by core parsers (parsers accept `io.Reader`)

## Data Flow

### Primary Request Path (Web Server)

1. **Upload / path request** arrives at `Handler.ParseLog` (`cmd/gst-server/internal/handlers/handlers.go:40`)
2. **Auto-detect** log kind via `parser.CreateParserAuto(file)` (`handlers.go:68`)
3. **Parse** into `*core.ParsedLog` via `p.Parse(file)` (`handlers.go:73`)
4. **Build search index** and store under `sync.RWMutex` (`handlers.go:111-117`)
5. **Analysis request** (e.g., `/api/log/analyze/top`) reads locked `ParsedLog`, instantiates analyzer, returns JSON (`handlers.go:379-424`)
6. **Export request** selects formatter by type-switch and writes to `http.ResponseWriter` (`handlers.go:756-835`)

### CLI Path

1. `cmd/cli/main.go` parses flags (`-parse`, `-search`, `-top`, etc.)
2. Opens file, detects kind, creates parser, parses log
3. Instantiates analyzers directly and prints to stdout
4. Optionally exports via `exporter.*Exporter` to file or stdout

### Raw Log Search Path

1. `Handler.Search` opens the original raw log file path (`handlers.go:312`)
2. Streams lines with `bufio.Scanner` (10MB buffer)
3. Applies keyword AND-match (case-insensitive substring)
4. Paginates in-memory and returns JSON

## Key Abstractions

**Parser Interface:**
- Purpose: Decouple log-format detection from parsing logic
- Examples: `internal/core/parser/parser.go:22-26`
- Pattern: Strategy pattern with factory `CreateParser` / `CreateParserAuto`

**Analyzer Structs:**
- Purpose: Encapsulate a single analysis dimension
- Examples: `internal/core/analyzer/frame_analyzer.go`, `internal/core/analyzer/draw_call_analyzer.go`
- Pattern: Each analyzer holds `*core.ParsedLog` and exposes `Analyze()` plus optional `GetSummary()`

**Exporter Interface:**
- Purpose: Uniform export to TXT/CSV/JSON
- Examples: `internal/core/exporter/exporter.go:13-16`
- Pattern: Interface with struct implementations; CSV uses large type-switch for formatting

**KeywordIndex:**
- Purpose: Fast API-name lookup by inverted index
- Examples: `internal/core/search/index.go`
- Pattern: In-memory map guarded by `sync.RWMutex`

## Entry Points

**gst-server:**
- Location: `cmd/gst-server/main.go`
- Triggers: `go run ./cmd/gst-server` or `./bin/gst-server`
- Responsibilities: Flag parsing, HTTP mux setup, static file serving, graceful shutdown

**gst-cli:**
- Location: `cmd/cli/main.go`
- Triggers: `./bin/gst-cli -parse <file>`
- Responsibilities: Flag parsing, file open/parse, analyzer invocation, stdout export

**functionaltest:**
- Location: `cmd/functionaltest/main.go`
- Triggers: `go run ./cmd/functionaltest`
- Responsibilities: End-to-end smoke test of parser, analyzer, search, exporter using embedded test data

## Architectural Constraints

- **Threading:** Single-process Go server; `sync.RWMutex` protects shared `ParsedLog` in `Handler`. No goroutine pools or worker queues.
- **Global state:** `var srv *http.Server` in `cmd/gst-server/main.go` (module-level singleton used by shutdown handler)
- **Circular imports:** None detected; dependency graph is strictly top-down (cmd → internal/core/* → internal/core)
- **Memory:** Entire `ParsedLog` loaded into memory; no streaming analysis. Large logs will consume RAM proportional to frame and API-call count.
- **Parser buffer:** 10MB scanner buffer (`api_parser.go:50`, `raw_trace_parser.go:33`) to handle long lines; may still OOM on pathological input.

## Anti-Patterns

### Analyzer Re-parses Raw Params with Regex

**What happens:** `BufferAnalyzer`, `BandwidthAnalyzer`, `DrawCallAnalyzer`, and others re-parse `RawParams` strings at analysis time using `strings.Fields`, `strconv`, and regex.
**Why it's wrong:** The parser already has the line; deferring param parsing to analysis time duplicates parsing logic and is fragile against format variations.
**Do this instead:** Enrich `APILogEntry` during parsing with typed fields (e.g., `Params []string` or a small struct), or introduce a `ParamParser` utility shared between parser and analyzer.

### Handler DTOs Mixed with Business Logic

**What happens:** `cmd/gst-server/internal/handlers/handlers.go` contains both HTTP routing and large response DTOs (`ComprehensiveReport`, `FrameSummary`, `ParseResult`).
**Why it's wrong:** Changes to API response shape require editing the same file that holds route security and orchestration logic.
**Do this instead:** Move DTOs to a `dto` or `api` sub-package (e.g., `cmd/gst-server/internal/api/`) and keep handlers thin.

### Raw Log Search Re-opens File on Every Request

**What happens:** `Handler.Search` opens the raw log file path, scans all lines, and filters in memory for every search request.
**Why it's wrong:** Repeated large-file scans are slow and redundant; the `KeywordIndex` exists but is not used for search.
**Do this instead:** Use the built `KeywordIndex` for API-name searches, or cache raw lines in memory if file size permits.

## Error Handling

**Strategy:** Return errors up the call stack; HTTP handlers translate to `http.Error` with 400/500 status codes.

**Patterns:**
- Parsers return `(*ParsedLog, error)`; scanner errors propagated at end of parse
- Analyzers return `nil` on nil input (defensive)
- Exporters return `fmt.Errorf("unsupported data type for CSV export")` for unknown types

## Cross-Cutting Concerns

**Logging:** Standard library `log` package only; no structured logging framework
**Validation:** Minimal; HTTP handlers check method and content-type manually
**Authentication:** None; server is intended for local development use

---

*Architecture analysis: 2026-05-07*
