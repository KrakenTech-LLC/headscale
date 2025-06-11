# Headscale Control Plane Library Makefile

.PHONY: test build example clean lint fmt vet

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=gofmt
GOVET=$(GOCMD) vet

# Build the example
example:
	@echo "Building example..."
	$(GOBUILD) -o bin/basic_usage ./examples/basic_usage.go

# Run tests
test:
	@echo "Running tests..."
	$(GOTEST) -v ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

# Run short tests only
test-short:
	@echo "Running short tests..."
	$(GOTEST) -short -v ./...

# Clean build artifacts
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -f bin/basic_usage
	rm -f coverage.out coverage.html

# Format code
fmt:
	@echo "Formatting code..."
	$(GOFMT) -s -w .

# Vet code
vet:
	@echo "Vetting code..."
	$(GOVET) ./...

# Lint code (requires golangci-lint)
lint:
	@echo "Linting code..."
	golangci-lint run

# Tidy dependencies
tidy:
	@echo "Tidying dependencies..."
	$(GOMOD) tidy

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download

# Run all checks
check: fmt vet test-short

# Build everything
build: example

# Create bin directory
bin:
	mkdir -p bin

# Help
help:
	@echo "Available targets:"
	@echo "  example      - Build the basic usage example"
	@echo "  test         - Run all tests"
	@echo "  test-coverage- Run tests with coverage report"
	@echo "  test-short   - Run short tests only"
	@echo "  clean        - Clean build artifacts"
	@echo "  fmt          - Format code"
	@echo "  vet          - Vet code"
	@echo "  lint         - Lint code (requires golangci-lint)"
	@echo "  tidy         - Tidy dependencies"
	@echo "  deps         - Download dependencies"
	@echo "  check        - Run fmt, vet, and short tests"
	@echo "  build        - Build everything"
	@echo "  help         - Show this help"

# Default target
all: check build
