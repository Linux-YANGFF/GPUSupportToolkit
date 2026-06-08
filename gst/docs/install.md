# Installation Guide

GST can be installed via deb or rpm packages, or built from source.

## Pre-built Packages

Download the appropriate package for your system:

### Debian/Ubuntu (.deb)

```bash
sudo dpkg -i gst-1.1.0-amd64.deb
# or
sudo dpkg -i gst-1.1.0-arm64.deb
```

### RHEL/CentOS/Fedora (.rpm)

```bash
sudo rpm -i gst-1.1.0-1.x86_64.rpm
# or
sudo rpm -i gst-1.1.0-1.aarch64.rpm
```

## Build from Source

### Prerequisites

- Go 1.22 or later
- For gst-server web UI build: Node.js/npm
- For deb packages: `dpkg-deb`
- For rpm packages: `rpmbuild` from `rpm-build`

This repository requires Go 1.22 or later. In the current development environment, use `/usr/local/go/bin/go` or put `/usr/local/go/bin` before the system Go in `PATH`.

### Build Binaries

```bash
# Optional when dependencies changed
go mod tidy

# Build gst-server (web UI)
go build -o bin/gst-server ./cmd/gst-server

# Build gst-cli (command line)
go build -o bin/gst-cli ./cmd/cli
```

### Build Packages

Using make:

```bash
# Build web assets and binaries
make build-all

# Build package for the current architecture
make deb
make rpm

# Build both package formats for the current architecture
make package

# Build all release packages
make deb-amd64
make deb-arm64
make rpm-amd64
make rpm-arm64
```

The deb targets cross-compile `gst-server` and `gst-cli` for Linux `amd64`/`arm64`, build `web/dist`, and stage:
- `/usr/bin/gst-server`
- `/usr/bin/gst`
- `/usr/bin/gst-cli`
- `/usr/bin/gst-ui`
- `/usr/bin/gst-create-desktop-shortcut`
- `/usr/share/gst/web/`
- `/usr/share/applications/gst.desktop`
- `/usr/share/icons/hicolor/scalable/apps/gst.svg`
- `/var/lib/gst/`

The rpm targets use the same staged payload and require `rpmbuild` on the build machine.

## Post-Installation

### gst-server

Start the web UI server:

```bash
# Default port 8080
gst-server

# Desktop/UI launcher, chooses a free port and opens the browser
gst-ui

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
gst-cli --help

# Parse a log file
gst -parse /path/to/log.trace
```

## Directory Structure

After installation:
- Binary: `/usr/bin/gst-server`, `/usr/bin/gst`, `/usr/bin/gst-cli`, and `/usr/bin/gst-ui`
- Web files: `/usr/share/gst/web/`
- Desktop entry: `/usr/share/applications/gst.desktop`
- Icon: `/usr/share/icons/hicolor/scalable/apps/gst.svg`
- Data directory: `/var/lib/gst/` (created automatically)
