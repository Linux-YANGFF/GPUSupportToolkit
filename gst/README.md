# GST - GPU Support Toolkit

GST is a GPU log analysis tool for parsing and analyzing apitrace/profile logs.

## Two Main Tools

- **gst-server**: Web-based UI server for visual log analysis and bug diagnosis
- **gst-cli**: Command-line tool for parsing, analyzing, and diagnosing GPU logs

## Quick Start

### Build

```bash
cd gst

# Use Go 1.22+
export PATH=/usr/local/go/bin:$PATH

# Build web assets, gst-server, and gst-cli
make build-all
```

### Run gst-server

```bash
# Start server (default port 8080)
./bin/gst-server

# Specify port
./bin/gst-server -port 8080

# Disable auto-open browser
./bin/gst-server -browser=false
```

Then open http://localhost:8080 in your browser.

### Use gst-cli

```bash
# Parse a log file
./bin/gst-cli -parse /path/to/log.trace

# Show top 20 slowest frames
./bin/gst-cli -top 20 -parse /path/to/log.trace

# Search for keywords
./bin/gst-cli -search glDrawElements -parse /path/to/log.trace

# Run bug diagnosis (7 analyzers)
./bin/gst-cli -diagnose -parse /path/to/log.trace
```

## Features

| Feature | Description |
|:---|:---|
| Log Parsing | Streaming parse of apitrace/profile logs (>1GB) |
| Keyword Search | Multi-keyword AND matching, case-insensitive |
| Time Range Search | Search API calls by time range |
| Frame Analysis | Find top N slowest frames |
| Function Stats | Call count and total time per function |
| Shader Stats | Shader compilation statistics |
| Multi-format Export | TXT/CSV/JSON export |
| **Bug Diagnosis** | 7 analyzers: null pointer, resource leak, shader error, API anti-pattern, perf anomaly, thread safety, driver error |
| Structured Logging | Go 1.22+ `log/slog` with configurable levels |
| AI JSON API | v2 HTTP endpoints and CLI JSON summaries for downstream AI tools |
| Release Packages | Cross-architecture deb/rpm targets for Linux amd64 and arm64 |

## Log Formats Supported

GST supports two apitrace log formats:

**Aggregated format:**
```
[ 31085] swapBuffers: 64205 us
[ 31086] <<gc = 0xffff60638d80>>
[ 35645] 2 frame cost 8061ms
```

**Raw format:**
```
[146982] (gc=0xfffe6985a840, tid=0x797f6fc0): glBindBuffer 0x8893 199
[146981] 38 frame cost 110ms
```

## Project Structure

```
gst/
├── cmd/
│   ├── gst-server/    # Web server
│   └── cli/           # CLI tool
├── internal/
│   ├── core/
│   │   ├── parser/    # Log parsers
│   │   ├── analyzer/  # Frame, function, shader analysis
│   │   ├── search/    # Keyword and time range search
│   │   ├── exporter/  # JSON/CSV/TXT export
│   │   └── bug/       # Bug diagnosis engine (7 analyzers + state machine)
│   └── platform/      # File I/O, OS detection, slog logger
├── web/               # Web UI files
└── packaging/         # Package configurations
```

## Documentation

- [CLI Reference](docs/cli.md) - Complete gst-cli usage guide
- [HTTP API Reference](docs/http-api.md) - gst-server REST endpoints
- [Installation Guide](docs/install.md) - Install via deb/rpm packages
