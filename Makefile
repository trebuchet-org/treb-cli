.PHONY: build install test clean dev-setup watch release-build release-clean

# Version information
VERSION ?= $(shell git describe --tags --always --dirty)
COMMIT ?= $(shell git rev-parse HEAD)
DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS = -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)

# Build the CLI binary
build:
	@echo "🔨 Building treb..."
	@go build -ldflags="$(LDFLAGS)" -o bin/treb ./cli

# Install globally
install: build
	@echo "📦 Installing treb..."
	@cp bin/treb /usr/local/bin/treb
	@echo "✅ treb installed to /usr/local/bin/treb"

# Run tests
test:
	@echo "🧪 Running tests..."
	@go test -v ./...

# Clean build artifacts
clean:
	@echo "🧹 Cleaning..."
	@rm -rf bin/
	@rm -rf out/
	@rm -rf cache/

# Development setup
dev-setup:
	@echo "🛠️  Setting up development environment..."
	@mkdir -p bin
	@go mod download
	@echo "✅ Development environment ready"

# Run the CLI locally
run: build
	@./bin/treb $(ARGS)

# Install forge if not present
install-forge:
	@command -v forge >/dev/null 2>&1 || { \
		echo "⚡ Installing Foundry..."; \
		curl -L https://foundry.paradigm.xyz | bash; \
		echo "Please run 'source ~/.bashrc' and 'foundryup' to complete installation"; \
	}

# Initialize example project
example: build
	@echo "📝 Creating example project..."
	@mkdir -p example
	@cd example && ../bin/treb init example-protocol --createx
	@echo "✅ Example project created in ./example/"

# Watch for file changes and rebuild
watch: build
	@echo "👀 Watching for changes in cli/..."
	@command -v fswatch >/dev/null 2>&1 || { \
		echo "❌ fswatch not found. Install it with:"; \
		echo "   macOS: brew install fswatch"; \
		echo "   Linux: apt-get install fswatch"; \
		exit 1; \
	}
	@fswatch -o cli/ | while read f; do \
		echo ""; \
		echo "🔄 Changes detected, rebuilding..."; \
		make build; \
		echo "✅ Build complete"; \
		echo ""; \
	done

# Release build targets
RELEASE_LDFLAGS = -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)

# Build all platform binaries for release
release-build: release-clean
	@echo "🚀 Building release binaries..."
	@mkdir -p release
	
	@echo "📦 Building Linux amd64..."
	@GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="$(RELEASE_LDFLAGS)" -o release/treb_linux_amd64 ./cli
	@cd release && tar czf treb_$(VERSION)_linux_amd64.tar.gz treb_linux_amd64 && rm treb_linux_amd64
	
	@echo "📦 Building Linux arm64..."
	@GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="$(RELEASE_LDFLAGS)" -o release/treb_linux_arm64 ./cli
	@cd release && tar czf treb_$(VERSION)_linux_arm64.tar.gz treb_linux_arm64 && rm treb_linux_arm64
	
	@echo "📦 Building macOS amd64..."
	@GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="$(RELEASE_LDFLAGS)" -o release/treb_darwin_amd64 ./cli
	@cd release && tar czf treb_$(VERSION)_darwin_amd64.tar.gz treb_darwin_amd64 && rm treb_darwin_amd64
	
	@echo "📦 Building macOS arm64..."
	@GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="$(RELEASE_LDFLAGS)" -o release/treb_darwin_arm64 ./cli
	@cd release && tar czf treb_$(VERSION)_darwin_arm64.tar.gz treb_darwin_arm64 && rm treb_darwin_arm64
	
	@echo "📦 Building Windows amd64..."
	@GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="$(RELEASE_LDFLAGS)" -o release/treb_windows_amd64.exe ./cli
	@cd release && zip treb_$(VERSION)_windows_amd64.zip treb_windows_amd64.exe && rm treb_windows_amd64.exe
	
	@echo "📦 Building Windows arm64..."
	@GOOS=windows GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="$(RELEASE_LDFLAGS)" -o release/treb_windows_arm64.exe ./cli
	@cd release && zip treb_$(VERSION)_windows_arm64.zip treb_windows_arm64.exe && rm treb_windows_arm64.exe
	
	@echo "🔐 Generating checksums..."
	@cd release && if command -v sha256sum >/dev/null 2>&1; then \
		sha256sum treb_$(VERSION)_*.{tar.gz,zip} > checksums.txt; \
	else \
		shasum -a 256 treb_$(VERSION)_*.{tar.gz,zip} > checksums.txt; \
	fi
	
	@echo "✅ Release binaries built in ./release/"
	@echo "📊 Release contents:"
	@ls -la release/

# Build specific platform binary
release-platform:
	@if [ -z "$(GOOS)" ] || [ -z "$(GOARCH)" ]; then \
		echo "❌ Please specify GOOS and GOARCH. Example:"; \
		echo "   make release-platform GOOS=linux GOARCH=amd64"; \
		exit 1; \
	fi
	@echo "🔨 Building $(GOOS)/$(GOARCH)..."
	@mkdir -p release
	@GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=0 go build -ldflags="$(RELEASE_LDFLAGS)" -o release/treb_$(GOOS)_$(GOARCH)$(if $(filter windows,$(GOOS)),.exe) ./cli
	@echo "✅ Built release/treb_$(GOOS)_$(GOARCH)$(if $(filter windows,$(GOOS)),.exe)"

# Clean release artifacts
release-clean:
	@echo "🧹 Cleaning release artifacts..."
	@rm -rf release/