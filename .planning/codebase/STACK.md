---
last_mapped_commit: 0e48e4f
analysis_date: 2026-05-09
focus: tech
---

# Technology Stack

**Analysis Date:** 2026-05-09

## Languages

**Primary:**
- Go 1.22 - 后端核心、CLI、HTTP Server（`go.mod` 指定 `go 1.22`）
- TypeScript - 前端逻辑（Vue 3 + Composition API）
- HTML/CSS - 前端模板与样式

**Secondary:**
- Shell (Makefile) - 构建与打包脚本

## Runtime

**Environment:**
- Go 1.22 或更高版本
- Node.js（仅用于前端构建，运行时不需要）

**Package Manager:**
- Go Modules（`go.mod` / `go.sum`）
- npm（前端，`web/package.json`）
- Lockfile: `go.sum` 存在但为空（零外部依赖）；`web/package-lock.json` 存在

## Frameworks

**Core Backend:**
- 标准库 `net/http` - HTTP Server（`cmd/gst-server/main.go`）
- 标准库 `log/slog` - 结构化日志（`internal/platform/logger.go`）
- 无第三方 Web 框架

**Frontend:**
- Vue 3.5.13 - 前端框架（`web/package.json`）
- Vite 6.3.5 - 构建工具（`web/vite.config.ts`）
- @vitejs/plugin-vue 5.2.4 - Vue SFC 支持

**Testing:**
- Go `testing` 标准库 - 单元测试
- 无第三方测试框架

**Build/Dev:**
- Vite - 前端开发与生产构建
- vue-tsc - Vue TypeScript 类型检查
- Make - 后端构建与打包（`Makefile`）

## Key Dependencies

**Go 后端 — 零外部依赖**

项目所有 Go 代码仅使用标准库，无 `go.mod` 中的 `require` 条目，`go.sum` 为空。

常用标准库包：
- `net/http` - HTTP Server（`cmd/gst-server/main.go`）
- `log/slog` - 结构化日志（`internal/platform/logger.go`）
- `encoding/json` - JSON 序列化/反序列化（`cmd/gst-server/internal/handlers/handlers.go`）
- `encoding/csv` - CSV 导出（`internal/core/exporter/exporter.go`）
- `regexp` - 日志解析正则匹配（`internal/core/parser/`）
- `sync` - 并发安全（`cmd/gst-server/internal/handlers/handlers.go` 中 `sync.RWMutex`）

**前端依赖（`web/package.json`）：**

生产依赖：
- `vue` ^3.5.13 - 前端框架

开发依赖：
- `@vitejs/plugin-vue` ^5.2.4 - Vite Vue 插件
- `typescript` ~5.8.3 - TypeScript 编译器
- `vite` ^6.3.5 - 构建工具
- `vue-tsc` ^2.2.10 - Vue TypeScript 类型检查

## Configuration

**环境：**
- 无 `.env` 文件，配置通过命令行参数或环境变量传递
- `GST_LOG_DIR` 环境变量控制日志文件访问路径（`cmd/gst-server/internal/handlers/handlers.go:720`）
- `GOPROXY=https://goproxy.cn,direct` 用于国内依赖下载（`Makefile`）

**构建配置：**
- `web/vite.config.ts` - Vite 配置，多入口（`index.html`, `logs.html`），输出到 `dist`
- `web/tsconfig.json` - TypeScript 配置，`baseUrl: "."`，路径别名 `@/*` -> `src/*`
- `Makefile` - Go 构建、测试、打包、交叉编译

**Go 构建标志：**
- `CGO_ENABLED=1` - GUI 构建（旧版 `cmd/gst`，当前未使用）
- `CGO_ENABLED=0` - CLI 和 Server 构建（纯 Go，无 CGO 依赖）

## Platform Requirements

**开发：**
- Go 1.22+
- Node.js + npm（用于前端构建）
- Make（可选，用于构建脚本）
- Linux 桌面环境（用于 `xdg-open` 自动打开浏览器）

**生产：**
- Linux 系统（Ubuntu、Kylin、UOS、Debian、RHEL 等）
- 无运行时外部依赖（Server/CLI 为静态二进制）
- 前端静态文件通过 `gst-server` 内置 HTTP 服务提供

**打包：**
- `dpkg-deb` 或 `fpm` - 构建 `.deb` 包
- `rpmbuild` 或 `fpm` - 构建 `.rpm` 包

---

*Stack analysis: 2026-05-09*
