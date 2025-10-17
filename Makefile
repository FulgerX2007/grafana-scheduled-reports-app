.PHONY: help install build build-frontend build-backend clean dev test package install-plugin

# Default target
help:
	@echo "Grafana Scheduled Reports Plugin - Build Commands"
	@echo ""
	@echo "Available targets:"
	@echo "  make install         - Install all dependencies"
	@echo "  make build           - Build both frontend and backend"
	@echo "  make build-frontend  - Build frontend only"
	@echo "  make build-backend   - Build backend only"
	@echo "  make package         - Create distribution archive"
	@echo "  make dev             - Start development mode with watch"
	@echo "  make clean           - Remove build artifacts"
	@echo "  make test            - Run all tests"
	@echo "  make install-plugin  - Install plugin to Grafana (requires root)"
	@echo ""

# Install dependencies
install:
	@echo "Installing frontend dependencies..."
	@npm install
	@echo "Downloading Go dependencies..."
	@go mod download
	@echo "✓ Dependencies installed"

# Build everything
build: build-frontend build-backend
	@echo "✓ Build complete"

# Build frontend
build-frontend:
	@echo "Building frontend..."
	@npm run build
	@echo "✓ Frontend built"

# Build backend (for local development - uses platform-specific name)
build-backend:
	@echo "Building backend..."
	@GOOS=$$(uname -s | tr '[:upper:]' '[:lower:]') && \
	GOARCH=$$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/;s/armv7l/arm/') && \
	BINARY="dist/gpx_reporting_$${GOOS}_$${GOARCH}" && \
	go build -o $${BINARY} ./cmd/backend && \
	chmod +x $${BINARY} && \
	echo "✓ Backend built ($${BINARY})"

# Create distribution package
package:
	@echo "Creating distribution package..."
	@chmod +x build.sh
	@./build.sh

# Development mode with file watching
dev:
	@echo "Starting development mode..."
	@echo "Frontend will rebuild automatically on changes"
	@npm run dev

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf dist/
	@rm -f *.zip
	@echo "✓ Clean complete"

# Run tests
test:
	@echo "Running Go tests..."
	@go test ./pkg/... -v
	@echo "Running frontend tests..."
	@npm test
	@echo "✓ Tests complete"

# Install plugin to Grafana (Linux only)
install-plugin:
	@echo "Installing plugin to Grafana..."
	@if [ ! -d "dist" ]; then \
		echo "Error: Plugin not built. Run 'make build' first."; \
		exit 1; \
	fi
	@sudo mkdir -p /var/lib/grafana/plugins/scheduled-reports-app
	@sudo rm -rf /var/lib/grafana/plugins/scheduled-reports-app/*
	@sudo cp -r dist/* /var/lib/grafana/plugins/scheduled-reports-app/
	@sudo chown -R grafana:grafana /var/lib/grafana/plugins/scheduled-reports-app
	@echo "✓ Plugin installed to /var/lib/grafana/plugins/scheduled-reports-app"
	@echo ""
	@echo "Next steps:"
	@echo "  1. Add to grafana.ini: allow_loading_unsigned_plugins = scheduled-reports-app"
	@echo "  2. Restart Grafana: sudo systemctl restart grafana-server"
