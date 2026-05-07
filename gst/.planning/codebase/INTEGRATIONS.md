# External Integrations

**Analysis Date:** 2026-05-07

## APIs & External Services

**GPU Log Analysis (Internal):**
- No external SaaS APIs, cloud services, or third-party SDKs
- All parsing, analysis, and export logic is self-contained in Go standard library

**System Utilities:**
- `xdg-open` - Used to auto-launch browser on server startup (`cmd/gst-server/main.go:97`)
- `xprop -root WM_NAME` - Desktop environment detection (`internal/platform/os_detector.go:66`)
- `xdpyinfo` - X11 display verification (`internal/platform/os_detector.go:72`)

## Data Storage

**Databases:**
- None - No database or persistent storage
- All log data is held in memory (`sync.RWMutex`-protected `Handler` struct in `cmd/gst-server/internal/handlers/handlers.go`)
- Parsed logs are ephemeral; re-parsed on each server restart

**File Storage:**
- Local filesystem only
- Log files read from absolute paths provided by user
- Uploaded files handled via `multipart/form-data` and parsed from memory stream
- Static web assets served from `web/` directory

**Caching:**
- None - No Redis, Memcached, or in-memory cache beyond parsed log state

## Authentication & Identity

**Auth Provider:**
- None - No authentication or authorization layer
- Server runs locally; no user sessions, tokens, or identity management

## Monitoring & Observability

**Error Tracking:**
- None - No Sentry, Rollbar, or similar service
- Errors logged to stdout/stderr via standard `log` package

**Logs:**
- Standard Go `log` package
- Server startup/shutdown events logged to console
- CLI outputs results to stdout or specified file

**Health Checks:**
- `GET /health` endpoint returns `200 OK` with body `"OK"` (`cmd/gst-server/internal/handlers/handlers.go:970`)

## CI/CD & Deployment

**Hosting:**
- Self-hosted / on-premise
- Runs as local binary or system service
- No container registry or cloud deployment configured

**CI Pipeline:**
- None detected - No GitHub Actions, GitLab CI, Jenkins, or other CI files

**Packaging:**
- `.deb` package built via `dpkg-deb` or `fpm` (`Makefile` targets `deb`, `package-deb`)
- `.rpm` package built via `rpmbuild` or `fpm` (`Makefile` targets `rpm`, `package-rpm`)
- Desktop entry: `packaging/deb/usr/share/applications/gst.desktop`
- Post-install/pre-remove scripts: `packaging/deb/DEBIAN/postinst`, `packaging/deb/DEBIAN/prerm`

## Environment Configuration

**Required env vars:**
- `DISPLAY` - Used for desktop environment detection (`internal/platform/os_detector.go`)
- `XDG_CONFIG_HOME` - Collected in env info (`internal/platform/os_detector.go`)
- `HOME` - Collected in env info (`internal/platform/os_detector.go`)
- `CGO_ENABLED` - Build-time variable (default 1, set to 0 in Makefile)
- `GOPROXY` - Used during dependency download (`https://goproxy.cn,direct`)

**Secrets location:**
- Not applicable - No secrets, API keys, or credentials used

## Webhooks & Callbacks

**Incoming:**
- None - No webhook endpoints

**Outgoing:**
- None - No callbacks to external services

## Browser / Client Integration

**Frontend:**
- Vue 3 loaded from local `web/js/vue.global.js` (no CDN)
- All API calls use relative paths to same-origin server (`/api/log/*`)
- No CORS configuration needed (same-origin)
- Fetch API used for all HTTP requests (`web/js/logs.js`)

---

*Integration audit: 2026-05-07*
