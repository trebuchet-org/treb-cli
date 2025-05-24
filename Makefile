.PHONY: build install test clean dev-setup

# Build the CLI binary
build:
	@echo "🔨 Building treb..."
	@go build -o bin/treb ./cli

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