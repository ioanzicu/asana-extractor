.PHONY: build test run clean lint

# Build the application
build:
	@echo "Building asana-extractor..."
	@mkdir -p bin
	@go build -o bin/asana-extractor ./cmd/extractor

# Run tests
test:
	@echo "Running tests..."
	@go test ./... -v

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	@go test ./... -cover -coverprofile=coverage.out
	@go tool cover -html=coverage.out -o coverage.html

# Run tests with race detection
test-race:
	@echo "Running tests with race detection..."
	@go test -race ./...

# Run the application
run:
	@echo "Running asana-extractor..."
	@go run ./cmd/extractor/main.go

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf bin/
	@rm -rf ./output/
	@rm -f coverage.out coverage.html

# Lint the code
lint:
	@echo "Running linters..."
	@go fmt ./...
	@go vet ./...

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	@go mod download
	@go mod tidy

# Install the application
install: build
	@echo "Installing asana-extractor..."
	@cp bin/asana-extractor $(GOPATH)/bin/
