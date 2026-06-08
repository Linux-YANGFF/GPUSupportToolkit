.PHONY: dev build build-cli build-server build-web build-all test test-core test-web check-web check-all lint clean help deb deb-amd64 deb-arm64 rpm rpm-amd64 rpm-arm64 package package-all

# Go parameters
GOCMD ?= $(or $(wildcard /usr/local/go/bin/go),go)
GOBUILD ?= $(GOCMD) build
GOTEST ?= $(GOCMD) test
GOMOD ?= $(GOCMD) mod
GOVET ?= $(GOCMD) vet
GOFMT ?= $(GOCMD) fmt
NPM ?= npm
GOCACHE ?= /tmp/gst-go-build

# Binary names
SERVER_BINARY = gst-server
CLI_BINARY = gst-cli
INSTALL_CLI = gst
ALL_BINARIES = $(SERVER_BINARY) $(CLI_BINARY)

# Build directories
BIN_DIR = bin
SERVER_BINARY_PATH = $(BIN_DIR)/$(SERVER_BINARY)
CLI_BINARY_PATH = $(BIN_DIR)/$(CLI_BINARY)
WEB_DIST = web/dist

# Source directories
SERVER_CMD = ./cmd/gst-server
CLI_CMD = ./cmd/cli
GO_SOURCES := $(shell find cmd internal -name '*.go')

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
SUPPORTED_PACKAGE_ARCHES = amd64 arm64
PACKAGE_ARCH ?= $(TARGET_ARCH)
RPM_ARCH = $(if $(filter arm64,$(PACKAGE_ARCH)),aarch64,x86_64)

# Help text
help:
	@echo "GST (GPU Support Toolkit) Build System"
	@echo ""
	@echo "Usage:"
	@echo "  make dev          Run server in development mode"
	@echo "  make build        Build gst-server binary"
	@echo "  make build-cli    Build CLI binary"
	@echo "  make build-web    Build web assets"
	@echo "  make build-all    Build web assets, gst-server, and gst-cli"
	@echo "  make test         Run all tests"
	@echo "  make test-core    Run core module tests only"
	@echo "  make test-web     Run web TypeScript checks"
	@echo "  make check-all    Run Go tests, vet, and web TypeScript checks"
	@echo "  make lint         Run code linting"
	@echo "  make fmt          Format code"
	@echo "  make vet          Run go vet"
	@echo "  make clean        Clean build artifacts"
	@echo "  make deps         Download and tidy dependencies"
	@echo "  make cross        Cross-compile for multiple architectures"
	@echo "  make deb          Build deb package for current arch"
	@echo "  make deb-amd64    Build amd64 deb package"
	@echo "  make deb-arm64    Build arm64 deb package"
	@echo "  make rpm          Build rpm package for current arch (requires rpmbuild)"
	@echo "  make rpm-amd64    Build x86_64 rpm package (requires rpmbuild)"
	@echo "  make rpm-arm64    Build aarch64 rpm package (requires rpmbuild)"
	@echo "  make package      Build deb and rpm for current arch"
	@echo "  make package-all  Build deb/rpm for amd64 and arm64"
	@echo ""
	@echo "Environment variables:"
	@echo "  GOCMD             Go command (default: /usr/local/go/bin/go when present)"
	@echo "  GOCACHE           Go build cache (default: /tmp/gst-go-build)"
	@echo "  PACKAGE_ARCH      Target package arch: amd64 or arm64"
	@echo ""

# Development mode - runs server directly
dev:
	@echo "Starting GST server in development mode..."
	GOCACHE=$(GOCACHE) CGO_ENABLED=0 $(GOCMD) run $(SERVER_CMD) -browser=false -web-dir $(WEB_DIST)

# Build server binary
build: build-server

build-server: $(SERVER_BINARY_PATH)

$(SERVER_BINARY_PATH): $(GO_SOURCES)
	@echo "Building gst-server binary..."
	GOCACHE=$(GOCACHE) CGO_ENABLED=0 $(GOBUILD) -o $(SERVER_BINARY_PATH) $(SERVER_CMD)
	@echo "Server binary built: $(SERVER_BINARY_PATH)"

# Build CLI binary
build-cli: $(CLI_BINARY_PATH)

$(CLI_BINARY_PATH): $(GO_SOURCES)
	@echo "Building CLI binary..."
	GOCACHE=$(GOCACHE) CGO_ENABLED=0 $(GOBUILD) -o $(CLI_BINARY_PATH) $(CLI_CMD)
	@echo "CLI binary built: $(CLI_BINARY_PATH)"

# Build frontend assets
build-web:
	@echo "Building web assets..."
	$(NPM) --prefix web run build

# Build web, server, and CLI
build-all: build-web build-server build-cli
	@echo "All artifacts built successfully"
	@ls -la $(BIN_DIR)/

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf $(BIN_DIR)
	@echo "Clean complete"

# Run all tests
test:
	@echo "Running all tests..."
	GOCACHE=$(GOCACHE) $(GOTEST) -v ./...

# Run core module tests only
test-core:
	@echo "Running core module tests..."
	GOCACHE=$(GOCACHE) $(GOTEST) -v ./internal/core/...

# Run web TypeScript checks without producing dist assets
test-web:
	@echo "Running web TypeScript checks..."
	$(NPM) --prefix web run typecheck

check-web: test-web

check-all: test vet test-web
	@echo "All verification checks passed"

# Run tests with coverage
test-cover:
	@echo "Running tests with coverage..."
	GOCACHE=$(GOCACHE) $(GOTEST) -v -coverprofile=coverage.out ./...
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
	GOCACHE=$(GOCACHE) $(GOVET) ./...

# Run linting
lint:
	@echo "Running linter..."
	golangci-lint run

# Cross-compile for multiple architectures
cross: cross-linux-amd64 cross-linux-arm64

cross-linux-amd64:
	@echo "Cross-compiling for Linux amd64..."
	GOCACHE=$(GOCACHE) CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BIN_DIR)/$(SERVER_BINARY)-linux-amd64 $(SERVER_CMD)
	GOCACHE=$(GOCACHE) CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BIN_DIR)/$(CLI_BINARY)-linux-amd64 $(CLI_CMD)
	@echo "Linux amd64 binaries built"

cross-linux-arm64:
	@echo "Cross-compiling for Linux arm64..."
	GOCACHE=$(GOCACHE) CGO_ENABLED=0 GOOS=linux GOARCH=arm64 $(GOBUILD) -o $(BIN_DIR)/$(SERVER_BINARY)-linux-arm64 $(SERVER_CMD)
	GOCACHE=$(GOCACHE) CGO_ENABLED=0 GOOS=linux GOARCH=arm64 $(GOBUILD) -o $(BIN_DIR)/$(CLI_BINARY)-linux-arm64 $(CLI_CMD)
	@echo "Linux arm64 binaries built"

# Build with specific architecture
build-arch:
	@echo "Building for $(TARGET_ARCH)..."
	GOCACHE=$(GOCACHE) CGO_ENABLED=0 GOOS=linux GOARCH=$(TARGET_ARCH) $(GOBUILD) -o $(SERVER_BINARY_PATH) $(SERVER_CMD)

# Verify build environment
check-env:
	@echo "Checking build environment..."
	@echo "Go version: $$(go version)"
	@echo "Architecture: $(ARCH) -> Target: $(TARGET_ARCH)"
	@echo "CGO_ENABLED: $(CGO_ENABLED)"

# Build packages for current arch
package: deb rpm
	@echo "All packages built successfully"
	@echo "Packages:"
	@ls -la bin/gst-*.deb bin/gst-*.rpm 2>/dev/null || ls -la bin/gst-*.deb 2>/dev/null

# Version from VERSION file or git tag
VERSION ?= $(shell cat VERSION 2>/dev/null || git describe --tags --abbrev=0 2>/dev/null || echo "1.0.0")

define stage_package
	rm -rf bin/pkg-$(1) && \
	mkdir -p bin/pkg-$(1)/usr/bin bin/pkg-$(1)/usr/share/applications bin/pkg-$(1)/usr/share/icons/hicolor/scalable/apps bin/pkg-$(1)/usr/share/gst/web bin/pkg-$(1)/DEBIAN bin/pkg-$(1)/var/lib/gst && \
	cp bin/$(SERVER_BINARY)-linux-$(1) bin/pkg-$(1)/usr/bin/$(SERVER_BINARY) && \
	cp bin/$(CLI_BINARY)-linux-$(1) bin/pkg-$(1)/usr/bin/$(INSTALL_CLI) && \
	ln -s $(INSTALL_CLI) bin/pkg-$(1)/usr/bin/$(CLI_BINARY) && \
	cp packaging/scripts/gst-ui bin/pkg-$(1)/usr/bin/gst-ui && \
	cp packaging/scripts/gst-create-desktop-shortcut bin/pkg-$(1)/usr/bin/gst-create-desktop-shortcut && \
	cp -r $(WEB_DIST)/* bin/pkg-$(1)/usr/share/gst/web/ && \
	cp packaging/deb/DEBIAN/postinst bin/pkg-$(1)/DEBIAN/postinst && \
	cp packaging/deb/DEBIAN/prerm bin/pkg-$(1)/DEBIAN/prerm && \
	cp packaging/deb/DEBIAN/postrm bin/pkg-$(1)/DEBIAN/postrm && \
	cp packaging/deb/usr/share/applications/* bin/pkg-$(1)/usr/share/applications/ && \
	cp packaging/deb/usr/share/icons/hicolor/scalable/apps/* bin/pkg-$(1)/usr/share/icons/hicolor/scalable/apps/ && \
	chmod 755 bin/pkg-$(1)/DEBIAN/postinst bin/pkg-$(1)/DEBIAN/prerm bin/pkg-$(1)/DEBIAN/postrm bin/pkg-$(1)/usr/bin/gst-ui bin/pkg-$(1)/usr/bin/gst-create-desktop-shortcut
endef

bin/$(SERVER_BINARY)-linux-%: $(GO_SOURCES)
	@echo "Cross-compiling gst-server for Linux $*..."
	@mkdir -p bin
	GOCACHE=$(GOCACHE) CGO_ENABLED=0 GOOS=linux GOARCH=$* $(GOBUILD) -o $@ $(SERVER_CMD)

bin/$(CLI_BINARY)-linux-%: $(GO_SOURCES)
	@echo "Cross-compiling gst-cli for Linux $*..."
	@mkdir -p bin
	GOCACHE=$(GOCACHE) CGO_ENABLED=0 GOOS=linux GOARCH=$* $(GOBUILD) -o $@ $(CLI_CMD)

deb: PACKAGE_ARCH := $(PACKAGE_ARCH)
deb: build-web bin/$(SERVER_BINARY)-linux-$(PACKAGE_ARCH) bin/$(CLI_BINARY)-linux-$(PACKAGE_ARCH)
	@echo "Building deb package (version $(VERSION), arch $(PACKAGE_ARCH))..."
	@$(call stage_package,$(PACKAGE_ARCH))
	@printf "Package: gst\nVersion: $(VERSION)\nSection: utils\nPriority: optional\nArchitecture: $(PACKAGE_ARCH)\nDepends: libc6\nRecommends: xdg-utils\nMaintainer: GST Team\nDescription: GPU Support Toolkit - GPU log analysis tool\n A tool for analyzing apitrace/profile/rawtrace logs.\n" > bin/pkg-$(PACKAGE_ARCH)/DEBIAN/control
	dpkg-deb --build bin/pkg-$(PACKAGE_ARCH) bin/gst-$(VERSION)-$(PACKAGE_ARCH).deb
	@echo "Deb package built: bin/gst-$(VERSION)-$(PACKAGE_ARCH).deb"

deb-amd64:
	$(MAKE) deb PACKAGE_ARCH=amd64

deb-arm64:
	$(MAKE) deb PACKAGE_ARCH=arm64

rpm: PACKAGE_ARCH := $(PACKAGE_ARCH)
rpm: build-web bin/$(SERVER_BINARY)-linux-$(PACKAGE_ARCH) bin/$(CLI_BINARY)-linux-$(PACKAGE_ARCH)
	@command -v rpmbuild >/dev/null 2>&1 || { echo "rpmbuild is required for rpm packaging. Install rpm-build or use a packaging container."; exit 127; }
	@echo "Building rpm package (version $(VERSION), arch $(RPM_ARCH))..."
	@$(call stage_package,$(PACKAGE_ARCH))
	@rm -rf bin/rpm-$(PACKAGE_ARCH)
	@mkdir -p bin/rpm-$(PACKAGE_ARCH)/BUILD bin/rpm-$(PACKAGE_ARCH)/BUILDROOT bin/rpm-$(PACKAGE_ARCH)/RPMS bin/rpm-$(PACKAGE_ARCH)/SOURCES bin/rpm-$(PACKAGE_ARCH)/SPECS bin/rpm-$(PACKAGE_ARCH)/SRPMS
	@tar -C bin/pkg-$(PACKAGE_ARCH) -czf bin/rpm-$(PACKAGE_ARCH)/SOURCES/gst-$(VERSION).tar.gz .
	rpmbuild --target $(RPM_ARCH) --define "_topdir $(shell pwd)/bin/rpm-$(PACKAGE_ARCH)" --define "pkg_version $(VERSION)" -bb packaging/rpm/gst.spec
	@find bin/rpm-$(PACKAGE_ARCH)/RPMS -name '*.rpm' -exec cp {} bin/ \;
	@echo "RPM package built under bin/"

rpm-amd64:
	$(MAKE) rpm PACKAGE_ARCH=amd64

rpm-arm64:
	$(MAKE) rpm PACKAGE_ARCH=arm64

package-all: deb-amd64 deb-arm64 rpm-amd64 rpm-arm64
	@echo "All amd64/arm64 deb/rpm package targets completed"

# Full build check - verifies everything compiles
check: check-env deps vet fmt build-all
	@echo ""
	@echo "Build verification complete!"
	@echo "Binaries:"
	@ls -la $(BIN_DIR)/
