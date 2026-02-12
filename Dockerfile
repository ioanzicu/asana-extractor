# Stage 1: Build the binary
FROM golang:1.25-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

WORKDIR /app

# Copy dependency files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the application
# CGO_ENABLED=0 ensures a static binary for the lean alpine image
RUN CGO_ENABLED=0 GOOS=linux go build -o asana-extractor ./cmd/extractor/main.go

# Stage 2: Final lightweight image
FROM alpine:latest

# Install CA certificates (required for HTTPS requests to Asana API)
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

# Copy the binary from the builder stage
COPY --from=builder /app/asana-extractor .

# Create the output directory to ensure correct permissions
RUN mkdir -p /root/output

# Set environment variables defaults
ENV OUTPUT_DIR=/root/output

# Run the application
CMD ["./asana-extractor"]