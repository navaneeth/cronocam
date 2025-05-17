# Binary name
BINARY_NAME=cronocam

# Build directory
BUILD_DIR=build

# Version (can be overridden)
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# Default target
.PHONY: all
all: build

# Help target
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build       - Build for current platform"
	@echo "  build-all   - Build for all platforms"
	@echo "  build-linux - Build for Linux (amd64, arm64, armv7)"
	@echo "  build-darwin- Build for macOS (amd64, arm64)"
	@echo "  build-windows- Build for Windows (amd64)"
	@echo "  install     - Install binary to GOBIN"
	@echo "  release     - Create release tarballs for all platforms"
	@echo "  clean       - Remove build artifacts"
	@echo "  test        - Run tests"
	@echo "  fmt         - Run go fmt"
	@echo "  lint        - Run go vet"
	@echo ""
	@echo "Build info:"
	@echo "  Platforms:      $(PLATFORMS)"
	@echo "  Architectures:  $(ARCHS)"
	@echo "  Version:        $(VERSION)"
	@echo ""
	@echo "Variables:"
	@echo "  VERSION     - Set version for build (default: git describe or 'dev')"

# Build for the local architecture
.PHONY: build
build:
	go build -o $(BINARY_NAME) -ldflags "-X github.com/navaneethkn/cronocam/internal/cmd.Version=$(VERSION)"

# Install the binary
.PHONY: install
install: build
	install -d $(GOBIN)
	install $(BINARY_NAME) $(GOBIN)/$(BINARY_NAME)

# Build directory for releases
RELEASE_DIR=dist

# Platforms to build for
PLATFORMS=linux darwin windows
ARCHS=amd64 arm64

# Release tarball target
.PHONY: release
release: clean
	mkdir -p $(RELEASE_DIR)
	$(foreach platform,$(PLATFORMS),\
		$(foreach arch,$(ARCHS),\
			GOOS=$(platform) GOARCH=$(arch) go build -o $(RELEASE_DIR)/$(BINARY_NAME)$(if $(findstring windows,$(platform)),.exe,) -ldflags "-X github.com/navaneethkn/cronocam/internal/cmd.Version=$(VERSION)" && \
			tar -czf $(RELEASE_DIR)/$(BINARY_NAME)-$(platform)-$(arch)-$(VERSION).tar.gz -C $(RELEASE_DIR) $(BINARY_NAME)$(if $(findstring windows,$(platform)),.exe,) && \
			rm $(RELEASE_DIR)/$(BINARY_NAME)$(if $(findstring windows,$(platform)),.exe,) \
		;)\
	)

# Clean build artifacts
.PHONY: clean
clean:
	rm -rf $(BUILD_DIR) $(RELEASE_DIR)
	rm -f $(BINARY_NAME)

# Cross compilation targets
.PHONY: build-all
build-all: build-linux build-darwin build-windows

.PHONY: build-linux
build-linux:
	mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 -ldflags "-X github.com/navaneethkn/cronocam/internal/cmd.Version=$(VERSION)"
	GOOS=linux GOARCH=arm64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 -ldflags "-X github.com/navaneethkn/cronocam/internal/cmd.Version=$(VERSION)"
	GOOS=linux GOARCH=arm GOARM=7 go build -o $(BUILD_DIR)/$(BINARY_NAME)-linux-armv7 -ldflags "-X github.com/navaneethkn/cronocam/internal/cmd.Version=$(VERSION)"

.PHONY: build-darwin
build-darwin:
	mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 -ldflags "-X github.com/navaneethkn/cronocam/internal/cmd.Version=$(VERSION)"
	GOOS=darwin GOARCH=arm64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 -ldflags "-X github.com/navaneethkn/cronocam/internal/cmd.Version=$(VERSION)"

.PHONY: build-windows
build-windows:
	mkdir -p $(BUILD_DIR)
	GOOS=windows GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe -ldflags "-X github.com/navaneethkn/cronocam/internal/cmd.Version=$(VERSION)"

# Test
.PHONY: test
test:
	go test -v ./...

# Run format and lint
.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: lint
lint:
	go vet ./...

