## Component Split Learnings

### Architecture
- Used provide/inject pattern via `useLogAnalysis()` composable for shared state between components
- `App.vue` is the orchestrator that provides context and composes all sub-components
- Components use `<script setup>` Composition API with `defineProps`/`defineEmits` for explicit interfaces
- FileUpload uses props + emits for parent-controlled v-model pattern
- AnalysisTabs uses named slots for tab content panels (frames, search, analyze, export)
- FrameList, FunctionStats, SearchPanel use inject to access shared state

### Files Created
- `src/types.ts` - Shared TypeScript interfaces
- `src/composables/useLogAnalysis.ts` - All business logic extracted from monolithic main.ts
- `src/components/FileUpload.vue` - File input with path/browse/parse/reset/stop
- `src/components/AnalysisTabs.vue` - Tab navigation with named slots for content
- `src/components/FrameList.vue` - Frame table with pagination and detail modal
- `src/components/FunctionStats.vue` - Top N frames + shader stats (analyze tab)
- `src/components/SearchPanel.vue` - Search bar + results with pagination
- `src/App.vue` - Root component, wires everything together
- `src/main.ts` - Minimal entry point, just creates and mounts App

### Key Decisions
- CSS variables theme system preserved in app.css (no changes needed)
- `logs.html` kept as-is (only imports main.ts + app.css, mount point unchanged)
- Export tab UI kept inline in App.vue since it only uses simple radio buttons / one button
- The `formatDuration` and `formatMs` helpers are local to FrameList.vue
