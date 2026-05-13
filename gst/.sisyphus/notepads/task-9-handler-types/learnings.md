# Learnings - Task 9: Replace map[string]interface{} with concrete types

## Patterns
- Go struct JSON tags control API output field names; dot-notation struct field access replaces map indexing
- When replacing map returns with struct returns, caller code using map indexing (summary["key"]) must change to struct field access (summary.FieldName)

## Conventions
- Response DTO types in handlers.go use interface{} for slices where the element type varies (e.g., Frames, Shaders fields)
- Summary types (FrameSummary, ShaderSummary, BufferSummary) live in internal/core/types.go with JSON tags

## Successful Approaches
- Used edit tool for targeted replacements in analyzer files
- Used write tool for full-file replacement when edit corrupted the function body
- Ran go clean -cache to clear stale build cache after syntax errors were fixed

## Issues
- The edit tool's oldString/newString matching produced overlapping matches, stripping function bodies and leaving orphaned duplicate code. Solution: use write tool for full-file replacement when edits are complex.
