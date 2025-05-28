.PHONY: build install test integration-test clean dev-setup watch release-build release-clean abis bindings clean check-bindings lint lint-install lint-fix 

# Version information
VERSION ?= $(shell git describe --tags --always --dirty)
COMMIT ?= $(shell git rev-parse HEAD)
DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS = -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)
# ABI bindings configuration
ABI_PKG_DIR := "cli/pkg/abi"
DEPLOYMENT_ABI := "treb-sol/out/Deployment.sol/Deployment.json"
PROXY_ABI := "treb-sol/out/ProxyDeployment.sol/ProxyDeployment.json"
LIBRARY_ABI := "treb-sol/out/LibraryDeployment.sol/LibraryDeployment.json"

# Build the CLI binary
build: bindings lint
	@echo "🔨 Building treb..."
	@go build -ldflags="$(LDFLAGS)" -tags dev -o bin/treb ./cli 

bindings: forge_build
	@echo "🔨 Generating Go bindings..."
	@echo ">> Extracting ABIs..."
	@jq -r '.abi' $(DEPLOYMENT_ABI) > $(ABI_PKG_DIR)/deployment/abi.json
	@jq -r '.abi' $(PROXY_ABI) > $(ABI_PKG_DIR)/proxy/abi.json
	@jq -r '.abi' $(LIBRARY_ABI) > $(ABI_PKG_DIR)/library/abi.json
	@echo ">> Generating deployment bindings..."
	@abigen --v2 --abi $(ABI_PKG_DIR)/deployment/abi.json --pkg deployment --type Deployment --out $(ABI_PKG_DIR)/deployment/bindings.go
	@echo ">> Generating proxy bindings..."
	@abigen --v2 --abi $(ABI_PKG_DIR)/proxy/abi.json --pkg proxy --type ProxyDeployment --out $(ABI_PKG_DIR)/proxy/bindings.go
	@echo ">> Generating library bindings..."
	@abigen --v2 --abi $(ABI_PKG_DIR)/library/abi.json --pkg library --type LibraryDeployment --out $(ABI_PKG_DIR)/library/bindings.go
	@rm -f $(ABI_PKG_DIR)/*/abi.json
	@echo "✅ Bindings generated"

forge_build:
	@echo ">> forge build"
	@cd treb-sol && forge build

check-bindings: bindings
	@echo "🔍 Checking if ABI bindings are up to date..."
	@if [ -n "$$(git status --porcelain cli/pkg/abi/*/bindings.go)" ]; then \
		echo "❌ ABI bindings are not up to date. Run 'make bindings' and commit changes."; \
		git status --porcelain cli/pkg/abi/*/bindings.go; \
		exit 1; \
	else \
		echo "✅ ABI bindings are up to date"; \
	fi

# Install globally
install: build
	@echo "📦 Installing treb..."
	@cp bin/treb /usr/local/bin/treb
	@echo "✅ treb installed to /usr/local/bin/treb"

# Run tests
test:
	@echo "🧪 Running tests..."
	@go test -v ./...

# Run integration tests  
integration-test: build
	@echo "🔗 Running integration tests..."
	@cd test && go mod download && go test -v -timeout=10m

# Run integration tests with coverage
integration-test-coverage: build
	@echo "🔗 Running integration tests with coverage..."
	@cd test && go test -v -timeout=10m -coverprofile=coverage.out
	@cd test && go tool cover -html=coverage.out -o coverage.html
	@echo "✅ Coverage report generated: test/coverage.html"

# Clean build artifacts
clean:
	@echo "🧹 Cleaning..."
	@rm -rf bin/
	@rm -rf out/
	@rm -rf cache/
	@rm -f $(ABI_PKG_DIR)/deployment/bindings.go
	@rm -f $(ABI_PKG_DIR)/proxy/bindings.go
	@rm -f $(ABI_PKG_DIR)/library/bindings.go
	@rm -f combined*.json
	@echo "✅ Cleaned"

# Development setup
dev-setup: lint-install
	@echo "🛠️  Setting up development environment..."
	@mkdir -p bin
	@go mod download
	@echo "✅ Development environment ready"

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

# Install golangci-lint
lint-install:
	@command -v golangci-lint >/dev/null 2>&1 || { \
		echo "Installing golangci-lint..."; \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh | sh -s -- -b $(go env GOPATH)/bin v2.1.6 \
		echo "✅ golangci-lint installed"; \
	}

# Run linter
lint:
	@echo "🔍 Running linter..."
	@golangci-lint run || { \
		echo "❌ Linting failed. Run 'make lint-fix' to automatically fix some issues."; \
		exit 1; \
	}
	@echo "✅ Linting passed"

# Fix linting issues automatically
lint-fix:
	@echo "🔧 Fixing linting issues..."
	@golangci-lint run --fix
	@echo "✅ Linting issues fixed (where possible)"
