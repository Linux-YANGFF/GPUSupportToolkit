# Learnings

## Driver Error Detector Implementation

- The `Diagnoser` interface requires `Diagnose(*core.ParsedLog) []core.Finding`
- Pattern: iterate frames, then iterate APICalls, find `__glSetError` entries with `IsError=true`
- Error code mapping: use lowercase keys in map, lowercase input for normalization
- `strings.ToUpper("0x0500")` converts to `"0X0500"` which breaks hex prefix - use `strings.ToLower` instead
- Backward trace: skip other `__glSetError` calls, match GCAddr for context isolation
- glGetError detection: pre-scan frame for any `glGetError` calls per context, flag when absent
- All findings use `SeverityCritical` and category `"driver_error"` as specified
