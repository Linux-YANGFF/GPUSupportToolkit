# Learnings — Task 2: 核心类型定义重构

## 发现

1. **APILogEntry vs APICall** — 代码库中不存在 `APICall` 结构体，APILogEntry 是实际的 API 调用记录类型。
   计划引用 `APICall` 但这是对等类型的误称。新字段添加到了 APILogEntry。

2. **FrameSummary/BufferSummary 不存在** — 计划引用 types.go:38-59 和 61-80 的 FrameSummary/BufferSummary，
   但这些结构体之前不存在。根据 frame_analyzer.GetFrameSummary() 和 buffer_analyzer.GetBufferSummary()
   的实际返回字段创建了它们。

3. **现有类型无 JSON 标签** — 大多数现有核心类型（APILogEntry, FrameInfo, BufferInfo, SearchResult, FuncStats,
   APISummary, ShaderCompileInfo）没有 JSON 标签。唯一的例外是 ShaderInfo.CommandLine 使用了
   PascalCase 的 json 标签。新类型统一使用 snake_case JSON 标签。

4. **分析器数量有限** — 计划引用了 sync_stall_analyzer, swap_classifier, texture_upload_analyzer,
   fbo_analyzer, draw_call_analyzer, state_change_analyzer, redundancy_analyzer, bandwidth_analyzer,
   但这些文件都不存在。目前只有 4 个分析器：frame, func, shader, buffer。

5. **零外部依赖** — go.mod 保持零 require，所有代码使用 Go 标准库。

## 约定

- 新类型使用 snake_case JSON 标签（如 `json:"total_frames"`）
- 可选字段使用 `omitempty` 保持向后兼容
- Go 字段名使用 PascalCase（Go 标准）
- 类型定义集中放在 `internal/core/types.go`

## Task 4: Security Hardening

### Pattern: Path validation with whitelist
- `validateLogPath()` uses two-layer defense: string-based `..` rejection + absolute path whitelist check
- Env var `GST_LOG_DIR` sets the allowed directory, defaults to CWD
- Uses `filepath.Abs()` for resolution before prefix check
- Returns descriptive errors for each failure mode

### Decisions
- PID file validation: Rejects system dirs (/etc/, /sys/, /proc/, /dev/) rather than full whitelist (PID file path is CLI flag, not HTTP input)
- Dead code removal: The `_ = search.KeywordSearchSimple([]string{""}, lines)` call was a no-op waste — removed

## Task 3: Error Handling Fixes

### Pattern: os.Open error handling
- Replaced all `file, _ := os.Open(filePath)` with proper error checks
- Errors written to stderr via `fmt.Fprintf(os.Stderr, ...)`
- Added `defer file.Close()` after each successful open
- Removed explicit `file.Close()` calls where defer now handles it

### Issues encountered
- `"time"` import got removed during batch edits (edit tool truncation) — re-added
- `timeit` helper function got removed during batch edits — re-added
- Go 1.18 compiler used but module requires Go 1.22 — works fine with just a note

### Decisions
- Used stderr for open errors (better CLI practice), kept stdout for parse errors (existing convention)
- Used `:=` pattern in switch cases for export error checks to avoid variable shadowing across cases

## task-14-parser-enhance: Raw Trace Parser Enhancements

### Patterns
- gc=/tid= prefix in raw trace format: `(gc=0xfffe6985a840, tid=0x797f6fc0): glXxx ...`
- Lines may have `[N]` line number prefix before the gc/tid prefix
- __glSetError format: `ERROR!!! __glSetError (gl_error=0xNNNN, errno=N, msg=(nil))`
- Segment fault markers: `段错误 (SIGSEGV) (core dumped)`
- pt=(nil) appears in function parameters for both legitimate (VBO bound) and dangerous (no VBO) cases
- glXMakeCurrent/glXCreateContextAttribsARB are parsed through colon format in parseAPICall()

### Decisions
- Used two regexes for gc/tid extraction: one with `[N]` prefix, one without
- Error/special lines (__glSetError, segfault) are inserted as APILogEntry with sentinel API names
- ensureFrame() helper reduces code duplication for frame initialization

### Gotchas
- removeRawTraceLinePrefix strips `[N]` but leaves a leading space, so gcTidRegex needs to handle it
- The DANGER comment line in test data gets skipped (expected - no gl/egl prefix)
- Existing tests all pass without modification - the enhancement is purely additive
