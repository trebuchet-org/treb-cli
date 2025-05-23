.PHONY: build install test clean dev-setup

# Build the CLI binary
build:
	@echo "🔨 Building fdeploy..."
	@go build -o bin/fdeploy ./cli

# Install globally
install: build
	@echo "📦 Installing fdeploy..."
	@cp bin/fdeploy /usr/local/bin/fdeploy
	@echo "✅ fdeploy installed to /usr/local/bin/fdeploy"

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
	@./bin/fdeploy $(ARGS)

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
	@cd example && ../bin/fdeploy init example-protocol --createx
	@echo "✅ Example project created in ./example/"