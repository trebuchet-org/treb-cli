.PHONY: build install test integration-test clean dev-setup watch release-build release-clean lint lint-install lint-fix setup-worktree

# Version information
VERSION ?= $(shell git describe --tags --always --dirty)
COMMIT ?= $(shell git rev-parse HEAD)
DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
TREB_SOL_COMMIT ?= $(shell cd treb-sol 2>/dev/null && git rev-parse HEAD || echo "unknown")

# abigen must match the go-ethereum version in go.mod: it generates code against
# the bind package, and a newer abigen emits symbols an older library lacks.
# Derived from go.mod so the two cannot drift apart.
GETH_VERSION ?= $(shell go list -m -f '{{.Version}}' github.com/ethereum/go-ethereum)

LDFLAGS = -X main.version=$(VERSION) \
					-X main.commit=$(COMMIT) \
					-X main.date=$(DATE) \
					-X main.trebSolCommit=$(TREB_SOL_COMMIT)

setup: 
	@echo "🔨 Getting submodules"
	@git submodule init
	@git submodule update
	@echo "🔨 Installing forge deps"
	@cd treb-sol && forge install
	@echo "🔨 Installing abigen ($(GETH_VERSION))"
	@go install github.com/ethereum/go-ethereum/cmd/abigen@$(GETH_VERSION)

# Full setup for git worktrees (one-shot: setup + integration test deps)
setup-worktree: setup setup-integration-test
	@echo "✅ Worktree fully set up"

forge_build:
	@echo ">> forge build"
	@cd treb-sol && forge build

bindings: forge_build
	@echo "🔨 Building bindings..."
	@cat treb-sol/out/ITrebEvents.sol/ITrebEvents.json | jq ".abi" | abigen --v2 --pkg bindings --type Treb --out internal/domain/bindings/treb.go --abi -
	@cat treb-sol/out/ICreateX.sol/ICreateX.json | jq ".abi" | abigen --v2 --pkg bindings --type CreateX --out internal/domain/bindings/createx.go --abi -

# Build the CLI binary
build: setup bindings
	@echo "🔨 Building treb..."
	@go build -ldflags="$(LDFLAGS)" -tags dev -o bin/treb ./cli

# Install globally
install: 
	@trebup --path $(shell pwd)

# Run tests
unit-test:
	@echo "🧪 Running tests..."
	@go test -v ./...

# Setup integration test dependencies
setup-integration-test:
	@echo "🔧 Setting up integration test dependencies..."
	@go install gotest.tools/gotestsum@latest
	@cd test/testdata/project && \
	if [ ! -d "lib/forge-std" ]; then \
		echo "Installing forge-std..."; \
		forge install foundry-rs/forge-std --no-git; \
	fi && \
	if [ ! -d "lib/openzeppelin-contracts" ]; then \
		echo "Installing openzeppelin-contracts..."; \
		forge install OpenZeppelin/openzeppelin-contracts --no-git; \
	fi && \
	if [ ! -d "lib/openzeppelin-contracts-upgradeable" ]; then \
		echo "Installing openzeppelin-contracts-upgradeable..."; \
		forge install OpenZeppelin/openzeppelin-contracts-upgradeable --no-git; \
	fi && \
	if [ ! -L "lib/treb-sol" ]; then \
		echo "Creating treb-sol symlink..."; \
		ln -sf ../../../../treb-sol lib/treb-sol; \
	fi
	@echo "✅ Integration test dependencies ready"

# Run integration tests  
integration-test: setup-integration-test
	@echo "🔗 Running integration tests..."
	@gotestsum --format=testname --no-summary=output --rerun-fails --rerun-fails-max-failures=5 --packages=./test/integration -- -v -timeout=15m

# Clean build artifacts
clean:
	@echo "🧹 Cleaning..."
	@rm -rf bin/
	@rm -rf treb-sol/out/
	@rm -rf treb-sol/cache/
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
RELEASE_LDFLAGS = -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE) -X main.trebSolCommit=$(TREB_SOL_COMMIT)

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
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh | sh -s -- -b $$(go env GOPATH)/bin v2.1.6; \
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
	@gofmt -w .
	@echo "✅ Linting issues fixed (where possible)"

fmt:
	@gofmt -s -w ./cli/
	@gofmt -s -w ./test/
