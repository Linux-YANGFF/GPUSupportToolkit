
## Task 10 Learnings

- `convertLine` was already removed in Task 5; only dead encoding detection code remained (BOM vars, DetectEncoding, detectEncoding)
- `DetectEncoding` had zero external callers — safe to remove entirely
- `detectEncoding()` was called but result discarded via `_ = sr.detectEncoding()` — true dead code
- `NewStreamReader` had BOM preamble read that was never used after convertLine removal
- Platform package has no panics, so no panic-to-error conversions needed
- `CheckDesktopEnvironment` in os_detector.go was silently swallowing exec errors — now wrapped with fmt.Errorf
