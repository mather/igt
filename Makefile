.PHONY: build test clean install run fmt lint

# Build the binary
build:
	go build -o igt ./cmd/igt

# Run tests
test:
	go test ./internal/... -v

# Run tests with coverage
test-coverage:
	go test ./internal/... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html

# Clean build artifacts
clean:
	rm -f igt
	rm -f coverage.out coverage.html

# Install the binary
install:
	go install ./cmd/igt

# Run the application
run:
	go run ./cmd/igt

# Format code
fmt:
	go fmt ./...

# Run linter (requires golangci-lint)
lint:
	golangci-lint run

# Tidy dependencies
tidy:
	go mod tidy

# Run all checks (fmt, lint, test)
check: fmt lint test

# Build for multiple platforms
build-all:
	GOOS=linux GOARCH=amd64 go build -o dist/igt-linux-amd64 ./cmd/igt
	GOOS=linux GOARCH=arm64 go build -o dist/igt-linux-arm64 ./cmd/igt
	GOOS=darwin GOARCH=amd64 go build -o dist/igt-darwin-amd64 ./cmd/igt
	GOOS=darwin GOARCH=arm64 go build -o dist/igt-darwin-arm64 ./cmd/igt
	GOOS=windows GOARCH=amd64 go build -o dist/igt-windows-amd64.exe ./cmd/igt

help:
	@echo "Available targets:"
	@echo "  build         - Build the binary"
	@echo "  test          - Run tests"
	@echo "  test-coverage - Run tests with coverage report"
	@echo "  clean         - Clean build artifacts"
	@echo "  install       - Install the binary"
	@echo "  run           - Run the application"
	@echo "  fmt           - Format code"
	@echo "  lint          - Run linter"
	@echo "  tidy          - Tidy dependencies"
	@echo "  check         - Run fmt, lint, and test"
	@echo "  build-all     - Build for multiple platforms"
