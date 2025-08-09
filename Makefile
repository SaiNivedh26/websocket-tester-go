# WebSocket Load Testing Tool Makefile

# Variables
BINARY_NAME=ws-load
BUILD_DIR=build
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS=-ldflags "-X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME}"

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
BINARY_UNIX=$(BINARY_NAME)_unix

# Default target
all: clean build

# Build the application
build:
	@echo "Building $(BINARY_NAME)..."
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) .
	@echo "Build complete: $(BINARY_NAME)"

# Build for multiple platforms
build-all: clean
	@echo "Building for multiple platforms..."
	@mkdir -p $(BUILD_DIR)
	
	# Linux
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 .
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 .
	
	# macOS
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 .
	
	# Windows
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe .
	
	@echo "Multi-platform build complete in $(BUILD_DIR)/"

# Run tests
test:
	@echo "Running tests..."
	$(GOTEST) -v ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run benchmarks
bench:
	@echo "Running benchmarks..."
	$(GOTEST) -bench=. -benchmem ./...

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_UNIX)
	rm -rf $(BUILD_DIR)
	rm -f coverage.out
	rm -f coverage.html

# Install dependencies
deps:
	@echo "Installing dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

# Install the binary
install: build
	@echo "Installing $(BINARY_NAME)..."

ifeq ($(OS),Windows_NT)
	@if not exist C:\\tools mkdir C:\\tools
	copy $(BINARY_NAME).exe C:\\tools
	@powershell -Command "if (-not ($$env:PATH -like '*C:\\tools*')) { setx PATH \"$$env:PATH;C:\\tools\"; Write-Host 'Added C:\\tools to PATH'; } else { Write-Host 'C:\\tools already in PATH'; }"
	@echo "Installed to C:\\tools. Restart your terminal to use it globally."
else
	sudo cp $(BINARY_NAME) /usr/local/bin/
	@echo "Installed to /usr/local/bin."
endif


# Uninstall the binary
uninstall:
	@echo "Uninstalling $(BINARY_NAME)..."

ifeq ($(OS),Windows_NT)
	del C:\\tools\\$(BINARY_NAME).exe
else
	sudo rm -f /usr/local/bin/$(BINARY_NAME)
endif
	@echo "Uninstallation complete"


# Run the application with example parameters
run:
	@echo "Running $(BINARY_NAME) with example parameters..."
	./$(BINARY_NAME) test -u ws://echo.websocket.org -d 10s -c 5

# Run with verbose output
run-verbose:
	@echo "Running $(BINARY_NAME) with verbose output..."
	./$(BINARY_NAME) test -u ws://echo.websocket.org -d 10s -c 5 -v

# Show help
help:
	@echo "Available targets:"
	@echo "  build         - Build the application"
	@echo "  build-all     - Build for multiple platforms (Linux, macOS, Windows)"
	@echo "  test          - Run tests"
	@echo "  test-coverage - Run tests with coverage report"
	@echo "  bench         - Run benchmarks"
	@echo "  clean         - Clean build artifacts"
	@echo "  deps          - Install dependencies"
	@echo "  install       - Install the binary to /usr/local/bin"
	@echo "  uninstall     - Uninstall the binary"
	@echo "  run           - Run with example parameters"
	@echo "  run-verbose   - Run with verbose output"
	@echo "  help          - Show this help message"

# Lint the code
lint:
	@echo "Linting code..."
	golangci-lint run

# Format the code
fmt:
	@echo "Formatting code..."
	$(GOCMD) fmt ./...

# Vet the code
vet:
	@echo "Vetting code..."
	$(GOCMD) vet ./...

# Security check
security:
	@echo "Running security checks..."
	$(GOCMD) list -json -deps . | nancy sleuth

# Generate documentation
docs:
	@echo "Generating documentation..."
	godoc -http=:6060 &
	@echo "Documentation available at http://localhost:6060"

# Release preparation
release: clean build-all test-coverage
	@echo "Release preparation complete"
	@echo "Version: $(VERSION)"
	@echo "Build artifacts in: $(BUILD_DIR)/"

.PHONY: all build build-all test test-coverage bench clean deps install uninstall run run-verbose help lint fmt vet security docs release 
