## Task 24-25 Learnings

- `NewDefaultRegistry()` exists in `report.go` (from prior Task 23) - registers all 7 analyzers
- `GenerateMarkdownReport(findings, sourceFile)` generates Chinese-markdown report
- `GenerateReport(sourceFile, findings)` generates JSON-ready `DiagnosisReport`
- Handler follows existing pattern: validate path → open file → parse → run analyzers → return JSON
- CLI follows same pattern as other operations: open → parse → analyze → output to stdout
- The `-parse` flag triggers all sub-commands due to default flag values (e.g., `-top 10`); diagnose should use `-top 0` to avoid

## Task 32: 文档更新

- 系统安装了两个 Go 版本：/usr/bin/go=1.18, /usr/local/go/bin/go=1.22.10。构建/测试时需要使用 PATH 指向 1.22
- 文档更新涉及 6 个文件：VERSION, README.md, gst/README.md, gst/docs/cli.md, gst/docs/install.md, CLAUDE.md
- gst/docs/cli.md 中新增的 Bug Diagnosis 章节包含完整的 7 个分析器中文说明
- install.md 中的 fpm 打包示例版本号也需要同步更新
## DiagnosisPanel Component Integration

### Architecture Decision
- The project uses two parallel code paths: the old `main.ts` (inline templates in `logs.html`) and the new `App.vue` (Vue SFC component-based architecture).
- `main.ts` now imports `App.vue` as the root component, which uses `provide/inject` pattern via `useLogAnalysis` composable.
- Components use `<script setup lang="ts">` with Composition API.

### Key Files Modified
- `src/components/DiagnosisPanel.vue` - New SFC component for bug diagnosis
- `src/components/AnalysisTabs.vue` - Added diagnose tab button and slot
- `src/components/FrameList.vue` - Fixed pre-existing undefined `formatDuration`/`formatMs` calls
- `src/App.vue` - Imported and wired DiagnosisPanel with filePath prop
- `src/composables/useLogAnalysis.ts` - Added diagnose tab definition
- `logs.html` - Added diagnose tab section (also kept for backward compat)

### API Contract
- `POST /api/diagnose` with `{ path: string }` returns `DiagnosisReport` JSON
- Response structure: `{ source_file, generated_at, summary: { total_findings, critical_count, high_count, medium_count, low_count, info_count }, findings: [{ severity, category, description, evidence, root_cause_chain, fix_suggestion }] }`

### Build Notes
- Pre-existing components (FrameList, SearchPanel, etc.) had build errors that blocked compilation
- Fixed by inlining the duration formatting expressions from the original logs.html template
- The build produces separate CSS chunk for scoped DiagnosisPanel styles
