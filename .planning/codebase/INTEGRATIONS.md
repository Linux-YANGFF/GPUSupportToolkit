---
last_mapped_commit: 0e48e4f
analysis_date: 2026-05-09
focus: tech
---

# External Integrations

**Analysis Date:** 2026-05-09

## APIs & External Services

**无外部 API 调用**

项目为纯离线工具，不调用任何第三方 SaaS、云 API 或外部网络服务。所有功能在本地完成。

## Data Storage

**Databases:**
- 无数据库。数据全部保存在内存中，以 Go 结构体形式组织。
- 解析后的日志存储在 `core.ParsedLog` 中（`internal/core/types.go`）
- Server 模式下通过 `Handler` 结构体持有当前解析状态（`cmd/gst-server/internal/handlers/handlers.go:33-40`）

**File Storage:**
- 本地文件系统读写
- 支持用户上传日志文件（multipart/form-data）或指定本地路径
- 导出格式：TXT、CSV、JSON（`internal/core/exporter/exporter.go`）

**Caching:**
- 无外部缓存服务
- Server 内存中缓存当前解析结果（`Handler.current *core.ParsedLog`）
- 关键字搜索索引 `search.KeywordIndex` 在内存中构建（`internal/core/search/index.go`）

## Authentication & Identity

**Auth Provider:**
- 无身份验证
- Server 为本地单用户服务，无登录/鉴权机制
- 文件路径通过 `validateLogPath` 做目录遍历防护（`cmd/gst-server/internal/handlers/handlers.go:716-736`）

## Monitoring & Observability

**Error Tracking:**
- 无外部错误追踪服务
- 使用标准库 `log/slog` 输出日志到 `os.Stderr`

**Logs:**
- `log/slog` 结构化日志（`internal/platform/logger.go`）
- 日志级别：debug、info、warn、error（通过 `-log-level` 参数或 `InitLogger` 控制）

## CI/CD & Deployment

**Hosting:**
- 本地部署，非云服务
- `gst-server` 作为本地 HTTP 服务运行（默认端口 8080）
- 通过 systemd 或手动启动

**CI Pipeline:**
- 无 CI 配置文件（未检测到 `.github/workflows/`、`.gitlab-ci.yml` 等）
- 构建通过本地 `make` 完成

**打包与分发：**
- `.deb` 包：通过 `dpkg-deb` 或 `fpm` 构建（`Makefile` 中 `deb` / `package-deb` 目标）
- `.rpm` 包：通过 `rpmbuild` 或 `fpm` 构建（`Makefile` 中 `rpm` 目标）
- 安装后文件位置：
  - 二进制：`/usr/bin/gst-server`、`/usr/bin/gst`
  - Web 静态文件：`/usr/share/gst/web/`
  - 数据目录：`/var/lib/gst/`

## Environment Configuration

**Required env vars:**
- `GST_LOG_DIR` - 限制 Server 可访问的日志文件目录（默认 `".."`，即上级目录）
- `DISPLAY` - 桌面环境检测（`internal/platform/os_detector.go`）
- `GOPROXY` - Go 依赖下载代理（开发时）

**Secrets location:**
- 无密钥/凭证管理需求
- 无 `.env`、密钥文件或凭证配置

## Webhooks & Callbacks

**Incoming:**
- 无 Webhook

**Outgoing:**
- 无外部回调
- Server 启动时可选调用 `xdg-open` 打开本地浏览器（`cmd/gst-server/main.go:88`）

## 浏览器集成

**自动打开浏览器：**
- `gst-server` 启动时通过 `os/exec` 调用 `xdg-open http://localhost:8080`
- 可通过 `-browser=false` 禁用

---

*Integration audit: 2026-05-09*
