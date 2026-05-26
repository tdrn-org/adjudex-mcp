.PHONY: build test lint fmt dev web clean run

BINARY_NAME := adjudex
BUILD_DIR := build
CMD_DIR := cmd/adjudex

# Format Go source files
fmt:
	@echo "📏 Formatting Go sources..."
	@gofmt -w .
	@echo "✅ Formatting done"

# Build the binary (depends on fmt for consistent formatting)
build: fmt
	@echo "🔨 Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY_NAME) ./$(CMD_DIR)
	@echo "✅ Binary: $(BUILD_DIR)/$(BINARY_NAME)"

# Run tests (depends on fmt for consistent formatting)
test: fmt
	@echo "🧪 Running tests..."
	go test ./... -count=1 -timeout=30s
	@echo "✅ All tests passed"

# Run tests with race detector
test-race:
	@echo "🧪 Running tests with race detector..."
	go test ./... -race -count=1 -timeout=60s
	@echo "✅ All tests passed"

# Run linter
lint:
	@echo "🔍 Running golangci-lint..."
	@which golangci-lint > /dev/null || (echo "❌ golangci-lint not installed. Run: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest" && exit 1)
	golangci-lint run ./...
	@echo "✅ Lint clean"

# Development mode — build and run with default config
dev: build
	@echo "🚀 Starting $(BINARY_NAME) in dev mode..."
	$(BUILD_DIR)/$(BINARY_NAME) --config adjudex-mcp.yaml

# Build SvelteKit frontend
web:
	@echo "🎨 Building SvelteKit frontend..."
	cd web && npm ci && npm run build
	@echo "✅ Frontend built: internal/web/static/"

# Run the binary
run: build
	$(BUILD_DIR)/$(BINARY_NAME)

# Clean build artifacts
clean:
	@echo "🧹 Cleaning..."
	rm -rf $(BUILD_DIR)
	rm -rf internal/web/static/*
	@echo "✅ Clean"

# Show help
help:
	@echo "adjudex — Agent juDy eXchange"
	@echo ""
	@echo "Targets:"
	@echo "  build     Build the adjudex binary"
	@echo "  test      Run unit tests"
	@echo "  test-race Run tests with race detector"
	@echo "  lint      Run golangci-lint"
	@echo "  dev       Build and run in dev mode"
	@echo "  web       Build SvelteKit frontend"
	@echo "  run       Build and run"
	@echo "  clean     Remove build artifacts"
