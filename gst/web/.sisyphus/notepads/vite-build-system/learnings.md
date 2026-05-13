# Vite+TypeScript build system setup

## Patterns
- Vite multi-page: Configure `build.rollupOptions.input` with named HTML entry points at project root
- Vue 3 global build → npm import: Replace `const { createApp, ref } = Vue` with `import { createApp, ref } from 'vue'`
- HTML files stay at project root for Vite (entry points), source files go into `src/`
- CSS referenced via `<link>` in HTML gets processed/versioned by Vite automatically
- `vue-tsc --noEmit` for type checking before build

## Decisions
- Used Vite 6.3.5 + Vue 3.5.13 + TypeScript 5.8
- Multi-page setup: index.html (landing) + logs.html (SPA) as separate rollup inputs
- Go server serves from `web/dist/` directory (use `-web-dir dist` flag)

## Issues
- `Object.entries()` on `unknown` type: cast with `as Record<K, V>` before destructuring
- Go server mimeTypes don't include `.mjs`/`.ts` - but Vite outputs `.js` bundles so this is fine
