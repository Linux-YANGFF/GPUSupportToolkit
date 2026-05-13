## Decisions
- Used robust trimmed statistics (two-pass: compute mean/stddev, exclude outliers, recompute) to prevent single outliers from inflating the detection threshold
- When trimmed set has zero stddev (all equal), use 10% of trimmed mean as proxy stddev with 100us minimum
- For draw call spikes: median-based threshold (5x median delta) instead of mean/stddev since deltas are non-normal distributed
- Draw call detection uses a whitelist of GL draw APIs, with fallback to substring match on "draw" for unrecognized APIs

## Patterns
- Diagnoser interface: `Diagnose(log *core.ParsedLog) []core.Finding`
- Finding structure: Severity, Category, Description, Evidence, RootCauseChain, FixSuggestion
- All findings use Severity=Medium, category="perf_anomaly"
- Test pattern: inline test data with FrameInfo slices, no external test files needed
