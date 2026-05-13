## Task 15 Learnings

### Module path
- Module is `gst` (mod.go L1), imports use `gst/internal/core`
- No external dependencies needed for core types

### Package naming
- Subpackage under `internal/core/` is `bug` (package name = `bug`)
- Imports core: `import "gst/internal/core"`

### Key types (confirmed in types.go)
- `core.ParsedLog` — Frames, TotalTimeUs, FPS
- `core.Finding` — Severity, Category, Description, Evidence, RootCauseChain, FixSuggestion
- `core.DiagnosisReport` — SourceFile, GeneratedAt, Summary, Findings
- `core.DiagnosisSummary` — TotalFindings, CriticalCount, HighCount, MediumCount, LowCount, InfoCount
- `core.Severity` constants: SeverityCritical/High/Medium/Low/Info

### Registry design
- Registration returns no error (fire-and-forget)
- RegisterNamed stores by name for GetDiagnoser lookup
- RunAll collects findings from ALL registered diagnosers (not filtered by severity)
