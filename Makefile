.PHONY: format build run test test-coverage clean

# Format all Go files
format:
	go fmt ./...

# Build the application
build:
	go build -o ollama-proxy

# Run the application
run: format build
	./ollama-proxy

# Run tests
test:
	go test -v ./...

# Run tests with coverage
test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

# Clean build artifacts
clean:
	rm -f ollama-proxy coverage.out 