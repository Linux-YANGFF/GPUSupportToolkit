# Installation Guide

GST can be installed via deb or rpm packages, or built from source.

## Pre-built Packages

Download the appropriate package for your system:

### Debian/Ubuntu (.deb)

```bash
sudo dpkg -i gst_1.0.0_amd64.deb
```

### RHEL/CentOS/Fedora (.rpm)

```bash
sudo rpm -i gst-1.0.0-1.x86_64.rpm
```

## Build from Source

### Prerequisites

- Go 1.21 or later
- For gst-server web UI: no additional dependencies
- For building packages: fpm tool

### Build Binaries

```bash
# Download dependencies
go mod tidy

# Build gst-server (web UI)
go build -o bin/gst-server ./cmd/gst-server

# Build gst-cli (command line)
go build -o bin/gst-cli ./cmd/cli
```

### Build Packages

Using make:

```bash
# Build deb package
make deb

# Build rpm package
make rpm

# Build both
make package
```

Manual build with fpm:

```bash
# Debian package
fpm -s dir -t deb \
  -n gst \
  -v 1.0.0 \
  -a amd64 \
  -p gst_1.0.0_amd64.deb \
  --prefix=/usr \
  -f bin/gst-server=/usr/bin/gst-server

# RPM package
fpm -s dir -t rpm \
  -n gst \
  -v 1.0.0 \
  -a x86_64 \
  -p gst-1.0.0-1.x86_64.rpm \
  --prefix=/usr \
  -f bin/gst-server=/usr/bin/gst-server
```

## Post-Installation

### gst-server

Start the web UI server:

```bash
# Default port 8080
gst-server

# Custom port
gst-server -port 9000

# Disable auto-open browser
gst-server -browser=false
```

Access the UI at http://localhost:8080

### gst-cli

```
# Show help
gst --help

# Parse a log file
gst -parse /path/to/log.trace
```

## Directory Structure

After installation:
- Binary: `/usr/bin/gst-server` and `/usr/bin/gst`
- Web files: `/usr/share/gst/web/`
- Data directory: `/var/lib/gst/` (created automatically)