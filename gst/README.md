# GST - GPU Support Toolkit

GST is a GPU log analysis tool for parsing and analyzing apitrace/profile logs.

## Two Main Tools

- **gst-server**: Web-based UI server for visual log analysis
- **gst-cli**: Command-line tool for parsing and analyzing logs

## Quick Start

### Build

```bash
cd gst

# Download dependencies
go mod tidy

# Build gst-server
go build -o bin/gst-server ./cmd/gst-server

# Build gst-cli
go build -o bin/gst-cli ./cmd/cli
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
```

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
│   └── core/
│       ├── parser/    # Log parsers
│       ├── analyzer/  # Frame, function, shader analysis
│       ├── search/    # Keyword and time range search
│       └── exporter/  # JSON/CSV/TXT export
├── web/               # Web UI files
└── packaging/         # Package configurations
```

## Documentation

- [CLI Reference](docs/cli.md) - Complete gst-cli usage guide
- [Installation Guide](docs/install.md) - Install via deb/rpm packages