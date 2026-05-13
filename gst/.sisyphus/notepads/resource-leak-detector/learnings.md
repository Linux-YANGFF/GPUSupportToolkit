# Resource Leak Detector - Learnings

## Implementation Notes

### Gen/Create calls don't reliably contain resource IDs
- `glGenBuffers 3` → "3" is a count, not a buffer ID
- `glCreateShader 0x8b31 5` → "5" IS the actual shader ID (second param)
- Solution: split into `pluralGenCreates` (skip ID extraction) and `singularCreates` (extract IDs)

### Resource ID discovery via bind/use calls
- Buffer/texture/framebuffer/renderbuffer/VAO IDs are discovered from bind calls (glBindBuffer, etc.)
- Shader IDs discovered from glShaderSource/glCompileShader
- Program IDs discovered from glUseProgram

### Delete call formats
- Singular deletes (glDeleteShader, glDeleteProgram): extract specific ID from params
- Plural deletes (glDeleteBuffers, etc.): first param is count, can't extract specific IDs
- `parseDeleteCount()` handles both hex pointers and decimal counts

### Leak detection logic
- Known IDs - deletions_from_singular_deletes - plural_delete_count > 0 → leak

## Pre-existing issues fixed
- shader_error_detector.go: removed duplicate `diagnoseFrame` method (used bare `itoa`)
- antipattern_detector.go: replaced bare `itoa()` with `strconv.Itoa()` and added `strconv` import
