## Thread Safety Detector Implementation Notes

### Design Decisions
- Detector scans all calls across all frames linearly, maintaining gcLastTID map
- glXMakeCurrent/wglMakeCurrent/eglMakeCurrent with null params (drawable=0, drawable=None, gc=nil, ctx=nil, draw=0) detected as unbind
- Duplicate T1->T2 switches suppressed via reported map keyed on "gcAddr:prevTID:newTID"
- Three detection levels:
  1. Per-incident thread switch without unbind (High)
  2. Rapid alternation >= 3 switches (High)
  3. Multi-thread access with proper unbind (Medium, summary only)

### Patterns Followed
- Diagnoser interface: `Diagnose(log *core.ParsedLog) []core.Finding`
- Finding structure: Severity, Category, Description, Evidence, RootCauseChain, FixSuggestion
- Tests follow existing patterns: construct ParsedLog with Frames/APICalls, call Diagnose, assert findings
- uses stringsContain helper matching existing test helpers style
