.PHONY: build test run clean lint

# Build the application
build:
	@echo "Building asana-extractor..."
	@mkdir -p bin
	@go build -o bin/asana-extractor ./cmd/extractor

# Run tests
test:
	@echo "Running tests..."
	@go test ./... -v -race

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	@go test ./... -cover -coverprofile=coverage.out
	@go tool cover -html=coverage.out -o coverage.html
	@open coverage.html

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


######################## 
# 		Docker 	 	   #
########################

# Docker configuration
IMAGE_NAME := asana-extractor
VERSION    := latest

# Build the Docker image
docker-build:
	@echo "Building Docker image $(IMAGE_NAME):$(VERSION)..."
	@docker build -t $(IMAGE_NAME):$(VERSION) .

# Run the application in a Docker container
# Note: This uses the .env file and mounts a local output directory
docker-run:
	@echo "Running $(IMAGE_NAME) in Docker..."
	@docker run --rm -it \
		--name $(IMAGE_NAME)-instance \
		--env-file .env \
		-v $(shell pwd)/output:/root/output \
		$(IMAGE_NAME):$(VERSION)

# Stop and remove a running container
docker-stop:
	@echo "Stopping Docker container..."
	@docker stop $(IMAGE_NAME)-instance || true
	@docker rm $(IMAGE_NAME)-instance || true

# Remove Docker images
docker-clean:
	@echo "Removing Docker images..."
	@docker rmi $(IMAGE_NAME):$(VERSION) || true

# View container logs
docker-logs:
	@docker logs -f $(IMAGE_NAME)-instance