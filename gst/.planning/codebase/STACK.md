# Technology Stack

**Analysis Date:** 2026-05-07

## Languages

**Primary:**
- Go 1.18 - Backend server, CLI tool, log parsers, analyzers, exporters

**Secondary:**
- HTML/CSS - Static web UI pages (`web/index.html`, `web/logs.html`, `web/css/app.css`)
- JavaScript (ES6+) - Frontend SPA logic (`web/js/logs.js`)

## Runtime

**Environment:**
- Go 1.18+ (module `gst`)
- No CGO required (`CGO_ENABLED=0` for all builds)
- Cross-compilation supported: linux/amd64, linux/arm64

**Package Manager:**
- Go modules (`go.mod`, `go.sum`)
- Lockfile: `go.sum` present
- Proxy: `https://goproxy.cn,direct` (used in Makefile)

## Frameworks

**Core:**
- Standard library `net/http` - HTTP server for `gst-server` (`cmd/gst-server/main.go`)
- Standard library `flag` - CLI argument parsing (`cmd/cli/main.go`, `cmd/gst-server/main.go`)
- Standard library `encoding/json`, `encoding/csv` - Data serialization
- Standard library `regexp` - Log line pattern matching (`internal/core/parser/`)
- Standard library `sync` (`sync.RWMutex`) - Handler concurrency (`cmd/gst-server/internal/handlers/handlers.go`)

**Frontend:**
- Vue 3 (global build, no build step) - `web/js/vue.global.js`
- Composition API (`createApp`, `ref`, `computed`, `watch`) - `web/js/logs.js`

**Testing:**
- Go standard library `testing` - Unit tests for analyzers, parsers, exporters, search

**Build/Dev:**
- GNU Make - Build orchestration (`Makefile`)
- `go vet` / `go fmt` - Linting and formatting
- `golangci-lint` - Optional linting (target in Makefile)
- `fpm` / `dpkg-deb` / `rpmbuild` - Package building (deb/rpm)

## Key Dependencies

**Critical:**
- No external Go dependencies - the project uses only the Go standard library
- `go.mod` contains no `require` blocks
- `go.sum` is empty

**Infrastructure:**
- `xdg-open` (system binary) - Browser auto-launch in dev mode (`cmd/gst-server/main.go:97`)
- `xprop`, `xdpyinfo` (system binaries) - Desktop environment detection (`internal/platform/os_detector.go`)

## Configuration

**Environment:**
- No `.env` file or external config file detected
- Configuration via command-line flags only

**CLI Flags (`gst-cli`):**
- `-parse <file>` - Parse log file
- `-search <keyword>` - Keyword search
- `-time <start,end>` - Time range search (microseconds)
- `-top <N>` - Top N slowest frames (default 10)
- `-funcs` - Function statistics
- `-shader` - Shader statistics
- `-export <txt|csv|json>` - Export format
- `-output <file>` - Output file

**Server Flags (`gst-server`):**
- `-port` - Server port (default "8080")
- `-browser` - Auto-open browser (default true)
- `-web-dir` - Static files directory (default "web")
- `-pidfile` - PID file path

**Build:**
- `Makefile` - Primary build configuration
- `VERSION` file - Semantic version (current: `1.0.0`)
- `go.mod` - Module definition

## Platform Requirements

**Development:**
- Go 1.18 or later
- Linux environment (Ubuntu, Kylin, UOS, Debian, RHEL supported)
- `make` for build automation
- Optional: `golangci-lint`, `fpm`, `dpkg-deb`, `rpmbuild` for packaging

**Production:**
- Linux x86_64 (amd64) or ARM64
- No runtime dependencies beyond the compiled binary
- Static binary (CGO disabled)
- Desktop environment optional (for browser auto-open)

---

*Stack analysis: 2026-05-07*
