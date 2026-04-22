.PHONY: dev build build-cli build-all test lint clean help deb rpm package

# Go parameters
GOCMD = go
GOBUILD = $(GOCMD) build
GOTEST = $(GOCMD) test
GOMOD = $(GOCMD) mod
GOVET = $(GOCMD) vet
GOFMT = $(GOCMD) fmt

# Binary names
GUI_BINARY = gst
CLI_BINARY = gst-cli
ALL_BINARIES = $(GUI_BINARY) $(CLI_BINARY)

# Build directories
BIN_DIR = bin
GUI_BINARY_PATH = $(BIN_DIR)/$(GUI_BINARY)
CLI_BINARY_PATH = $(BIN_DIR)/$(CLI_BINARY)

# Source directories
GUI_CMD = ./cmd/gst
CLI_CMD = ./cmd/cli

# Architecture support
ARCH ?= $(shell uname -m)
ifeq ($(ARCH),x86_64)
    TARGET_ARCH = amd64
else ifeq ($(ARCH),aarch64)
    TARGET_ARCH = arm64
else
    TARGET_ARCH = $(ARCH)
endif

# Cross-compilation targets
TARGETS = linux-amd64 linux-arm64

# Help text
help:
	@echo "GST (GPU Support Toolkit) Build System"
	@echo ""
	@echo "Usage:"
	@echo "  make dev          Run GUI in development mode"
	@echo "  make build        Build GUI binary"
	@echo "  make build-cli    Build CLI binary"
	@echo "  make build-all    Build both GUI and CLI binaries"
	@echo "  make test         Run all tests"
	@echo "  make test-core    Run core module tests only"
	@echo "  make lint         Run code linting"
	@echo "  make fmt          Format code"
	@echo "  make vet          Run go vet"
	@echo "  make clean        Clean build artifacts"
	@echo "  make deps         Download and tidy dependencies"
	@echo "  make cross        Cross-compile for multiple architectures"
	@echo "  make deb          Build deb package"
	@echo "  make rpm          Build rpm package"
	@echo "  make package      Build both deb and rpm packages"
	@echo ""
	@echo "Environment variables:"
	@echo "  CGO_ENABLED       Set to 1 for GUI builds (default: 1)"
	@echo "  ARCH              Target architecture (auto-detected)"
	@echo ""

# Development mode - runs GUI directly
dev:
	@echo "Starting GST GUI in development mode..."
	@echo "Note: Requires X11/Wayland display and CGO dependencies"
	CGO_ENABLED=1 $(GOCMD) run $(GUI_CMD)

# Build GUI binary
build: $(GUI_BINARY_PATH)

$(GUI_BINARY_PATH): $(GUI_CMD)/*.go
	@echo "Building GUI binary..."
	CGO_ENABLED=1 $(GOBUILD) -o $(GUI_BINARY_PATH) $(GUI_CMD)
	@echo "GUI binary built: $(GUI_BINARY_PATH)"

# Build CLI binary
build-cli: $(CLI_BINARY_PATH)

$(CLI_BINARY_PATH): $(CLI_CMD)/*.go
	@echo "Building CLI binary..."
	CGO_ENABLED=0 $(GOBUILD) -o $(CLI_BINARY_PATH) $(CLI_CMD)
	@echo "CLI binary built: $(CLI_BINARY_PATH)"

# Build gst-server binary
SERVER_BINARY_PATH = $(BIN_DIR)/gst-server
SERVER_CMD = ./cmd/gst-server

build-server: $(SERVER_BINARY_PATH)

$(SERVER_BINARY_PATH): $(SERVER_CMD)/*.go
	@echo "Building gst-server binary..."
	CGO_ENABLED=0 $(GOBUILD) -o $(SERVER_BINARY_PATH) $(SERVER_CMD)
	@echo "Server binary built: $(SERVER_BINARY_PATH)"

# Build both GUI and CLI
build-all: build build-cli
	@echo "All binaries built successfully"
	@ls -la $(BIN_DIR)/

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf $(BIN_DIR)
	@echo "Clean complete"

# Run all tests
test:
	@echo "Running all tests..."
	$(GOTEST) -v ./...

# Run core module tests only
test-core:
	@echo "Running core module tests..."
	$(GOTEST) -v ./internal/core/...

# Run tests with coverage
test-cover:
	@echo "Running tests with coverage..."
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

# Download and tidy dependencies
deps:
	@echo "Downloading dependencies..."
	export GOPROXY=https://goproxy.cn,direct && $(GOMOD) download
	@echo "Tidying dependencies..."
	export GOPROXY=https://goproxy.cn,direct && $(GOMOD) tidy

# Format code
fmt:
	@echo "Formatting code..."
	$(GOFMT) ./...

# Run go vet
vet:
	@echo "Running go vet..."
	$(GOVET) ./...

# Run linting
lint:
	@echo "Running linter..."
	golangci-lint run

# Cross-compile for multiple architectures
cross: cross-linux-amd64 cross-linux-arm64

cross-linux-amd64:
	@echo "Cross-compiling for Linux amd64..."
	CGO_ENABLED=1 $(GOBUILD) -o $(BIN_DIR)/$(GUI_BINARY)-linux-amd64 $(GUI_CMD)
	CGO_ENABLED=0 $(GOBUILD) -o $(BIN_DIR)/$(CLI_BINARY)-linux-amd64 $(CLI_CMD)
	@echo "Linux amd64 binaries built"

cross-linux-arm64:
	@echo "Cross-compiling for Linux arm64..."
	CGO_ENABLED=1 GOOS=linux GOARCH=arm64 $(GOBUILD) -o $(BIN_DIR)/$(GUI_BINARY)-linux-arm64 $(GUI_CMD)
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 $(GOBUILD) -o $(BIN_DIR)/$(CLI_BINARY)-linux-arm64 $(CLI_CMD)
	@echo "Linux arm64 binaries built"

# Build with specific architecture
build-arch:
	@echo "Building for $(TARGET_ARCH)..."
	CGO_ENABLED=1 $(GOBUILD) -o $(GUI_BINARY_PATH) $(GUI_CMD)

# Verify build environment
check-env:
	@echo "Checking build environment..."
	@echo "Go version: $$(go version)"
	@echo "Architecture: $(ARCH) -> Target: $(TARGET_ARCH)"
	@echo "CGO_ENABLED: $(CGO_ENABLED)"

# Build all packages (deb and rpm)
package: deb rpm
	@echo "All packages built successfully"
	@echo "Packages:"
	@ls -la bin/gst-*.deb bin/gst-*.rpm 2>/dev/null || ls -la bin/gst-*.deb 2>/dev/null

# Build deb package using fpm
package-deb: build-server
	@echo "Building deb package using fpm..."
	@mkdir -p bin
	fpm -s dir -t deb \
		-n gst \
		-v $(VERSION) \
		-a $(ARCH) \
		-p bin/gst_$(VERSION)_$(ARCH).deb \
		--prefix=/usr \
		-f bin/gst-server=/usr/bin/gst-server \
		--directories /var/lib/gst \
		--post-install packaging/deb/DEBIAN/postinst \
		--pre-remove packaging/deb/DEBIAN/prerm \
		packaging/deb/usr/share/applications/gst.desktop=/usr/share/applications/gst.desktop
	@echo "Deb package built: bin/gst_$(VERSION)_$(ARCH).deb"

# Build rpm package using fpm
package-rpm: build-server
	@echo "Building rpm package using fpm..."
	@mkdir -p bin
	fpm -s dir -t rpm \
		-n gst \
		-v $(VERSION) \
		-a $(ARCH) \
		-p bin/gst_$(VERSION)_$(ARCH).rpm \
		--prefix=/usr \
		-f bin/gst-server=/usr/bin/gst-server \
		--directories /var/lib/gst
	@echo "RPM package built: bin/gst_$(VERSION)_$(ARCH).rpm"

# Version from VERSION file or git tag
VERSION ?= $(shell cat VERSION 2>/dev/null || git describe --tags --abbrev=0 2>/dev/null || echo "1.0.0")

# Architecture mapping (uname -m -> deb/rpm arch names)
ARCH ?= $(shell uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')

# Legacy deb target (uses dpkg-deb)
deb: build-server build-cli
	@echo "Building deb package (version $(VERSION), arch $(ARCH))..."
	@mkdir -p bin/deb/usr/bin
	@mkdir -p bin/deb/DEBIAN
	@mkdir -p "bin/deb/usr/share/applications"
	@mkdir -p "bin/deb/usr/share/gst/web"
	cp bin/gst-server bin/deb/usr/bin/gst-server
	cp bin/gst-cli bin/deb/usr/bin/gst
	cp -r web/* bin/deb/usr/share/gst/web/
	cp packaging/deb/DEBIAN/* bin/deb/DEBIAN/
	cp packaging/deb/usr/share/applications/* bin/deb/usr/share/applications/
	chmod 755 bin/deb/DEBIAN/postinst bin/deb/DEBIAN/prerm
	cd bin/deb && dpkg-deb --build . ../gst-$(VERSION)-$(ARCH).deb
	@echo "Deb package built: bin/gst-$(VERSION)-$(ARCH).deb"

# Legacy rpm target (uses rpmbuild)
rpm: build-server
	@echo "Building rpm package..."
	@mkdir -p bin/rpm BUILD RPMS/x86_64
	rpmbuild --define "_topdir $(shell pwd)/bin/rpm" -bb packaging/rpm/gst.spec
	@mv bin/rpm/RPMS/x86_64/*.rpm bin/gst-1.0.0.x86_64.rpm 2>/dev/null || echo "RPM built in bin/rpm/RPMS/x86_64/"
	@echo "RPM package built"

# Full build check - verifies everything compiles
check: check-env deps vet fmt build-all
	@echo ""
	@echo "Build verification complete!"
	@echo "Binaries:"
	@ls -la $(BIN_DIR)/
