## Task 6: slog Migration Learnings

### Patterns
- slog uses key=value pairs for structured logging: `slog.Info("msg", "key", val, "key2", val2)`
- Replace `log.Fatalf` with `slog.Error` + `os.Exit(1)` — slog has no Fatal equivalent
- For CLI tools, route errors to slog.Error (stderr), keep table output on stdout
- Use `slog.SetDefault()` in InitLogger to propagate config to all packages

### Conventions
- Text handler with time+level+msg format matching the original log output pattern
- Level mapping: Chinese error messages preserved, but structured with key=value pairs
- Verbose flag in CLI maps to debug level, default maps to warn (errors only)

### Pre-existing Issues Fixed
- frame_analyzer.go: duplicate dead code after GetFrameSummary()
- shader_analyzer.go: duplicate dead code after GetShaderSummary()
- handlers.go: extra closing brace after exportAnalysisJSON()
- Exporter: missing CSVExporter[T] generic type; JSONExporter used without type parameter
