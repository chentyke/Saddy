# Saddy Makefile

.PHONY: help build run test clean docker docker-run docker-stop format lint install

# Default target
help:
	@echo "Saddy - Lightweight Reverse Proxy Server"
	@echo ""
	@echo "Available targets:"
	@echo "  help        - Show this help message"
	@echo "  build       - Build the application"
	@echo "  run         - Run the application with default config"
	@echo "  test        - Run tests"
	@echo "  clean       - Clean build artifacts"
	@echo "  docker      - Build Docker image"
	@echo "  docker-run  - Run with Docker Compose"
	@echo "  docker-stop - Stop Docker Compose services"
	@echo "  format      - Format Go code"
	@echo "  lint        - Run linter"
	@echo "  install     - Install dependencies"

# Variables
APP_NAME = saddy
VERSION = $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME = $(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS = -ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)"

# Build targets
build:
	@echo "Building $(APP_NAME)..."
	@mkdir -p build
	go build $(LDFLAGS) -o build/$(APP_NAME) ./cmd/$(APP_NAME)
	@echo "Build completed: build/$(APP_NAME)"

build-all:
	@echo "Building for multiple platforms..."
	@mkdir -p build
	# Linux AMD64
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o build/$(APP_NAME)-linux-amd64 ./cmd/$(APP_NAME)
	# Linux ARM64
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o build/$(APP_NAME)-linux-arm64 ./cmd/$(APP_NAME)
	# macOS AMD64
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o build/$(APP_NAME)-darwin-amd64 ./cmd/$(APP_NAME)
	# macOS ARM64
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o build/$(APP_NAME)-darwin-arm64 ./cmd/$(APP_NAME)
	# Windows AMD64
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o build/$(APP_NAME)-windows-amd64.exe ./cmd/$(APP_NAME)
	@echo "Multi-platform build completed"

run: build
	@echo "Running $(APP_NAME)..."
	./build/$(APP_NAME)

run-dev:
	@echo "Running $(APP_NAME) in development mode..."
	go run ./cmd/$(APP_NAME) -config configs/config.yaml

test:
	@echo "Running tests..."
	go test -v ./...

test-coverage:
	@echo "Running tests with coverage..."
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

clean:
	@echo "Cleaning build artifacts..."
	rm -rf build/
	rm -f coverage.out coverage.html
	go clean -cache
	@echo "Clean completed"

# Docker targets
docker:
	@echo "Building Docker image..."
	docker build -t $(APP_NAME):$(VERSION) .
	docker tag $(APP_NAME):$(VERSION) $(APP_NAME):latest
	@echo "Docker image built: $(APP_NAME):$(VERSION)"

docker-run:
	@echo "Starting services with Docker Compose..."
	docker-compose up -d

docker-stop:
	@echo "Stopping Docker Compose services..."
	docker-compose down

docker-logs:
	docker-compose logs -f

# Development targets
format:
	@echo "Formatting Go code..."
	go fmt ./...
	@echo "Code formatted"

lint:
	@echo "Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed. Install with: curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b \$$(go env GOPATH)/bin v1.54.2"; \
	fi

install:
	@echo "Installing dependencies..."
	go mod tidy
	go mod download
	@echo "Dependencies installed"

# Development setup
setup:
	@echo "Setting up development environment..."
	go mod tidy
	@if ! command -v golangci-lint >/dev/null 2>&1; then \
		echo "Installing golangci-lint..."; \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin v1.54.2; \
	fi
	@echo "Development environment setup completed"

# Release targets
release: clean build-all
	@echo "Creating release package..."
	@mkdir -p release
	# Create tarballs
	cd build && tar -czf ../release/$(APP_NAME)-$(VERSION)-linux-amd64.tar.gz $(APP_NAME)-linux-amd64
	cd build && tar -czf ../release/$(APP_NAME)-$(VERSION)-linux-arm64.tar.gz $(APP_NAME)-linux-arm64
	cd build && tar -czf ../release/$(APP_NAME)-$(VERSION)-darwin-amd64.tar.gz $(APP_NAME)-darwin-amd64
	cd build && tar -czf ../release/$(APP_NAME)-$(VERSION)-darwin-arm64.tar.gz $(APP_NAME)-darwin-arm64
	cd build && zip ../release/$(APP_NAME)-$(VERSION)-windows-amd64.zip $(APP_NAME)-windows-amd64.exe
	@echo "Release packages created in release/ directory"

# Installation target
install-local: build
	@echo "Installing $(APP_NAME) to /usr/local/bin..."
	sudo cp build/$(APP_NAME) /usr/local/bin/
	sudo chmod +x /usr/local/bin/$(APP_NAME)
	@echo "Installation completed"

# Uninstallation target
uninstall-local:
	@echo "Uninstalling $(APP_NAME) from /usr/local/bin..."
	sudo rm -f /usr/local/bin/$(APP_NAME)
	@echo "Uninstallation completed"

# Release helper targets
tag:
	@echo "Creating a new release tag..."
	@./scripts/tag-release.sh

release-build:
	@echo "Building release packages..."
	@./scripts/release.sh

.PHONY: tag release-build