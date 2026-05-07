# Codebase Concerns

**Analysis Date:** 2026-05-07

## Tech Debt

### Error Handling: Silent Failures and Ignored Errors

**Issue:** Multiple locations ignore errors or silently swallow parse failures, leading to incorrect data without warning.
**Files:**
- `cmd/cli/main.go:125` — `file, _ = os.Open(filePath)` (repeated at lines 202, 226, 269, 296, 328)
- `cmd/cli/main.go:344-350` — ` _ = exp.Export(&buf)` for all three export formats
- `cmd/gst-server/internal/handlers/handlers.go:564-565` — `startUs, _ := strconv.ParseInt(...)` and `endUs, _ := strconv.ParseInt(...)` in `SearchTimeRange`
- `internal/core/parser/api_parser.go:163` — `frameCostMs, _ := strconv.ParseInt(matches[3], 10, 64)`
- `internal/core/parser/api_parser.go:236` — `swapTimeUs, _ := strconv.ParseInt(matches[2], 10, 64)`
- `cmd/gst-server/main.go:178` — `absWebDir, _ := filepath.Abs(*webDir)`
- `cmd/functionaltest/main.go:68,84` — `tmpFile, _ := os.CreateTemp(...)`

**Impact:** CLI crashes are masked, server returns incorrect time-range results, frame timing may be zeroed out silently, and static file serving may use wrong paths.
**Fix approach:** Propagate every error. Use `if err != nil { return err }` or log fatal in CLI entry points. In handlers, return `http.StatusBadRequest` for invalid query params.

### Duplicate Logic Across Analyzers

**Issue:** Every analyzer re-implements the same frame-iteration and nil-check patterns. Counting logic (`if call.Count <= 0 { count++ }`) is duplicated in `DrawCallAnalyzer`, `FBOAnalyzer`, `TextureUploadAnalyzer`, `SyncStallAnalyzer`, `StateChangeAnalyzer`, and `ShaderAnalyzer`.
**Files:**
- `internal/core/analyzer/draw_call_analyzer.go`
- `internal/core/analyzer/fbo_analyzer.go`
- `internal/core/analyzer/texture_upload_analyzer.go`
- `internal/core/analyzer/sync_stall_analyzer.go`
- `internal/core/analyzer/state_change_analyzer.go`
- `internal/core/analyzer/shader_analyzer.go`

**Impact:** Adding a new analyzer requires copying boilerplate. Fixing a counting bug requires touching six files.
**Fix approach:** Extract a `FrameVisitor` helper or embed a base `Analyzer` struct that handles nil checks and iteration.

### Hard-Coded Thresholds and Magic Numbers

**Issue:** Performance thresholds are scattered as literal constants with no configuration mechanism.
**Files:**
- `internal/core/analyzer/texture_upload_analyzer.go:73` — `frameUploads > 5` for thrashing
- `internal/core/analyzer/fbo_analyzer.go:77` — `frameBinds > 3` for thrashing
- `cmd/gst-server/internal/handlers/handlers.go:51` — `100 << 20` (100MB) multipart limit
- `cmd/gst-server/internal/handlers/handlers.go:322` — `10*1024*1024` scanner buffer
- `internal/core/parser/api_parser.go:49` — `10*1024*1024` scanner buffer
- `internal/platform/file_reader.go:113` — `64*1024*1024` scanner buffer

**Impact:** Thresholds cannot be tuned for different workloads without recompiling. Large log files may still hit scanner limits.
**Fix approach:** Move thresholds to a `config` package or environment variables. Document expected log sizes and test with representative files.

### CLI Re-Parses File for Every Sub-Command

**Issue:** `cmd/cli/main.go` opens and parses the log file independently for each flag (`-search`, `-time`, `-top`, `-funcs`, `-shader`, `-export`).
**Files:** `cmd/cli/main.go:112-362`

**Impact:** Running `gst-cli -parse file.txt -search x -top 10 -funcs` parses the same file three times, wasting CPU and I/O.
**Fix approach:** Parse once in `main`, store `*core.ParsedLog` in a struct, and pass it to sub-command handlers.

### Server Handler Monolith

**Issue:** `handlers.go` is 973 lines, containing all HTTP handlers, DTOs, and business logic in one file. There is no middleware layer, no request validation beyond basic checks, and no structured logging.
**Files:** `cmd/gst-server/internal/handlers/handlers.go`

**Impact:** Hard to test individual endpoints, hard to add cross-cutting concerns (auth, metrics, rate limiting), and high risk of merge conflicts.
**Fix approach:** Split into `handlers/frames.go`, `handlers/analyze.go`, `handlers/export.go`, `handlers/search.go`. Introduce a small middleware chain for logging and recovery.

## Known Bugs

### Frame Cost Parsing Silently Fails on Invalid Numbers

**Symptoms:** Frame total time becomes 0 when `frame cost` line contains unexpected formatting.
**Files:** `internal/core/parser/api_parser.go:163`
**Trigger:** A log line like `abc frame cost ms` (missing number) is matched by regex but `strconv.ParseInt` fails silently.
**Workaround:** None. The frame will have `TotalTimeUs = 0`.

### Swap Time Parsing Silently Fails

**Symptoms:** `SwapBufferTimeUs` is 0 even when `swapBuffers: X us` is present.
**Files:** `internal/core/parser/api_parser.go:236`
**Trigger:** Regex group mismatch or non-numeric value.
**Workaround:** None.

### `CreateParserAuto` Requires Seekable Reader but Does Not Document It

**Symptoms:** Passing a non-seekable reader (e.g., network stream, pipe) panics or returns an error.
**Files:** `internal/core/parser/parser.go:223-239`
**Trigger:** `reader.(io.ReadSeeker)` type assertion fails.
**Workaround:** Buffer the reader into memory first.

### `BufferAnalyzer` Updates All Buffers of Same Target on `glBufferData`

**Symptoms:** Buffer sizes and usage flags are incorrectly shared across buffers with the same target.
**Files:** `internal/core/analyzer/buffer_analyzer.go:183-193`
**Trigger:** Multiple buffers bound to `GL_ARRAY_BUFFER` in the same frame.
**Workaround:** None. The analyzer cannot distinguish which buffer received the `glBufferData` call.

## Security Considerations

### Path Traversal in `ParseLog` Handler

**Risk:** The server accepts a JSON `path` field and opens it directly with `os.Open`. While the static file server has traversal protection, the API endpoint does not validate the path.
**Files:** `cmd/gst-server/internal/handlers/handlers.go:88`
**Current mitigation:** None for API path.
**Recommendations:** Validate that `req.Path` is within an allowed directory whitelist. Reject absolute paths or paths containing `..`.

### No Input Size Limits on JSON Endpoints

**Risk:** `json.NewDecoder(r.Body).Decode` is used without `http.MaxBytesReader`, allowing large request bodies to exhaust memory.
**Files:** `cmd/gst-server/internal/handlers/handlers.go:83`, `cmd/gst-server/internal/handlers/handlers.go:771`
**Current mitigation:** Only `/api/log/parse` has a multipart limit.
**Recommendations:** Wrap `r.Body` with `http.MaxBytesReader` for all JSON endpoints.

### No Authentication or Authorization

**Risk:** Anyone with network access can upload files, trigger analysis, and shut down the server via `/api/shutdown`.
**Files:** `cmd/gst-server/main.go:136-158`
**Current mitigation:** None.
**Recommendations:** Add a simple API key check or bind to localhost only by default.

## Performance Bottlenecks

### Full-Frame Scan in `GetFrameDetail` and `GetFrameFuncs`

**Problem:** Both handlers iterate over `current.Frames` linearly to find a frame by `FrameNum`.
**Files:** `cmd/gst-server/internal/handlers/handlers.go:208-217`, `cmd/gst-server/internal/handlers/handlers.go:239-262`
**Cause:** `ParsedLog.Frames` is a slice, not a map.
**Improvement path:** Build a `map[int]*core.FrameInfo` index during parsing, or use binary search if frames are guaranteed ordered.

### Search Re-Opens Raw Log File on Every Request

**Problem:** `Handler.Search` opens `rawLogPath` from disk and scans every line on each query.
**Files:** `cmd/gst-server/internal/handlers/handlers.go:311-317`
**Cause:** Raw lines are stored in memory (`h.lines`) but search still uses the file.
**Improvement path:** Use `h.lines` for search instead of re-reading the file. The `KeywordIndex` exists but is not used for the HTTP search endpoint.

### Comprehensive Analysis Re-Runs All Analyzers on Every Request

**Problem:** `AnalyzeComprehensive` instantiates and runs every analyzer from scratch.
**Files:** `cmd/gst-server/internal/handlers/handlers.go:678-753`
**Cause:** No caching of analyzer results.
**Improvement path:** Cache analyzer results in `Handler` and invalidate on new parse. Most analyzers are pure functions of `ParsedLog`.

### Large Scanner Buffers Allocated Per-Parse

**Problem:** Each parser allocates a 10MB or 64MB byte slice for `bufio.Scanner`.
**Files:** `internal/core/parser/api_parser.go:49`, `internal/core/parser/raw_trace_parser.go:33`, `internal/platform/file_reader.go:113`
**Cause:** Fixed-size buffers to handle long lines.
**Improvement path:** Use a shared buffer pool (`sync.Pool`) or switch to a streaming line reader that does not require a single contiguous buffer.

## Fragile Areas

### Regex-Heavy Log Parsing

**Files:** `internal/core/parser/api_parser.go`, `internal/core/parser/raw_trace_parser.go`, `internal/core/parser/parser.go`
**Why fragile:** Log formats are detected and parsed with a large set of regexes. Any change in apitrace output format (new prefixes, different spacing, additional fields) will break parsing silently or produce garbage frames.
**Safe modification:** Add new test cases to `parser_test.go` before changing regexes. Keep a corpus of real log samples.
**Test coverage:** `parser_test.go` covers basic cases but does not test malformed lines, mixed formats, or very large frames.

### `normalizeHexToName` Has Incorrect Mapping for `GL_ELEMENT_ARRAY_BUFFER`

**Files:** `internal/core/analyzer/buffer_analyzer.go:339-364`
**Why fragile:** `0x8D40` is mapped to `TargetElementArrayBuffer`, but `0x8893` is the actual OpenGL constant for `GL_ELEMENT_ARRAY_BUFFER`. `0x8D40` is `GL_FRAMEBUFFER`.
**Safe modification:** Verify all hex constants against the OpenGL specification before changing.
**Test coverage:** `analyzer_test.go` only tests `0x8892` (GL_ARRAY_BUFFER).

### `targetToHex` Duplicate Mapping Maintenance

**Files:** `internal/core/analyzer/buffer_analyzer.go:366-391`
**Why fragile:** `normalizeHexToName` and `targetToHex` are inverse maps maintained manually. They can drift out of sync.
**Safe modification:** Generate one from the other, or use a single bidirectional map.

### Shader Source Truncation Mutates Shared Data

**Files:** `cmd/gst-server/internal/handlers/handlers.go:443-448`
**Why fragile:** `AnalyzeShaders` truncates `shader.Source` in-place. If the same `ParsedLog` is analyzed again or exported, the shader source is permanently shortened.
**Safe modification:** Return a copy of the shader struct with truncated source, or truncate during JSON encoding only.

## Scaling Limits

### In-Memory Log Storage

**Current capacity:** The entire log is parsed into `[]core.FrameInfo` with all `APICalls`, `Shaders`, and `BufferCreations` held in RAM.
**Limit:** Logs larger than available memory will cause OOM. No streaming or pagination at parse time.
**Scaling path:** Implement a streaming parser that yields frames one at a time, or store parsed data in a temporary on-disk format (e.g., SQLite or flat file with offsets).

### Single-Threaded Server

**Current capacity:** Go's `http.Server` uses one goroutine per request, but `Handler` uses a single `sync.RWMutex` around the entire parsed log.
**Limit:** Concurrent requests for analysis block each other on read locks. Large comprehensive reports hold the read lock for a long time, blocking new uploads.
**Scaling path:** Use copy-on-write for `ParsedLog` (immutable after parse) so reads need no lock. Or shard analyzers per session.

## Dependencies at Risk

### No Dependency Pinning Beyond `go.mod`

**Risk:** `go.mod` exists but there is no `vendor/` directory. If upstream packages are removed or compromised, builds break.
**Impact:** Build reproducibility is not guaranteed in air-gapped environments.
**Migration plan:** Add `go mod vendor` to the build pipeline, or use a private Go module proxy.

## Missing Critical Features

### No Structured Logging

**Problem:** The server uses `log.Printf` for all output. There is no request ID, no log level, and no JSON formatting.
**Blocks:** Production observability, log aggregation, and alerting.
**Files:** `cmd/gst-server/main.go`, `cmd/gst-server/internal/handlers/handlers.go`

### No Request Timeout or Rate Limiting

**Problem:** HTTP handlers have no timeout. A large log parse or comprehensive analysis can hang the connection indefinitely.
**Blocks:** Safe exposure to untrusted networks.
**Files:** `cmd/gst-server/internal/handlers/handlers.go`

### No Health Check Beyond "OK"

**Problem:** `/health` returns a static string with no checks for disk space, memory pressure, or parser readiness.
**Blocks:** Reliable deployment in container orchestrators.
**Files:** `cmd/gst-server/internal/handlers/handlers.go:970-973`

## Test Coverage Gaps

### Server Handlers Untested

**What's not tested:** All HTTP handlers in `handlers.go` (parse, frames, search, analyze, export).
**Files:** `cmd/gst-server/internal/handlers/handlers.go`
**Risk:** Handler logic changes can break API contracts without any test failure.
**Priority:** High

### Parser Edge Cases Untested

**What's not tested:** Malformed lines, mixed log formats, empty files, files with only headers, very long lines exceeding scanner buffer, and `KindUnknown` fallback behavior.
**Files:** `internal/core/parser/parser_test.go`
**Risk:** Production logs with unexpected formatting crash the parser or produce empty results.
**Priority:** High

### Platform Package Untested

**What's not tested:** `file_reader.go` stream reading, encoding detection, index read/write, and `os_detector.go` OS detection.
**Files:** `internal/platform/file_reader.go`, `internal/platform/os_detector.go`
**Risk:** File encoding mis-detection or index corruption goes unnoticed.
**Priority:** Medium

### Exporter Edge Cases Untested

**What's not tested:** Export of empty data, very large data sets, and CSV escaping of fields containing commas or newlines.
**Files:** `internal/core/exporter/exporter.go`
**Risk:** Exported CSV/JSON may be malformed.
**Priority:** Medium

---

*Concerns audit: 2026-05-07*
