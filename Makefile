.PHONY: all build clean test deps install

VERSION ?= dev

# Default target
all: build

# Resolve Go dependencies (run once or after adding new imports)
deps:
	@echo "Resolving Go dependencies..."
	cd cli && go mod tidy

# Build the Go CLI (pure Go — no Rust dependency)
build:
	@echo "Building NeoHtop CLI..."
	cd cli && CGO_ENABLED=1 go build -ldflags="-s -w -X main.version=$(VERSION)" -o ../neohtop-cli .
	@echo "Binary built: ./neohtop-cli"

# Run Go tests
test:
	@echo "Running tests..."
	cd cli && CGO_ENABLED=1 go test ./...

# Clean build artifacts
clean:
	cd cli && go clean
	rm -f neohtop-cli neohtop-cli.exe

# Development build (race detector enabled)
dev:
	cd cli && CGO_ENABLED=1 go build -race -o ../neohtop-cli .

# Install to system
install: build
	cp neohtop-cli /usr/local/bin/

# Cross-compilation targets
# Note: macOS builds require CGo (for libproc/mach), Linux builds are pure Go
build-linux-amd64:
	cd cli && GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o ../neohtop-cli-linux-amd64 .

build-linux-arm64:
	cd cli && GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o ../neohtop-cli-linux-arm64 .

build-macos-arm64:
	cd cli && GOOS=darwin GOARCH=arm64 CGO_ENABLED=1 go build -o ../neohtop-cli-macos-arm64 .

build-macos-amd64:
	cd cli && GOOS=darwin GOARCH=amd64 CGO_ENABLED=1 go build -o ../neohtop-cli-macos-amd64 .
