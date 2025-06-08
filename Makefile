.PHONY: build run clean deps build-all help

# Build the application
build:
	go build -o bin/nitpick ./cmd/nitpick

# Run the application directly
run:
	go run ./cmd/nitpick

# Clean build artifacts
clean:
	rm -rf bin/

# Install dependencies
deps:
	go mod tidy
	go mod download

# Build for multiple platforms
build-all:
	GOOS=linux GOARCH=amd64 go build -o bin/nitpick-linux-amd64 ./cmd/nitpick
	GOOS=linux GOARCH=arm64 go build -o bin/nitpick-linux-arm64 ./cmd/nitpick
	GOOS=linux GOARCH=arm go build -o bin/nitpick-linux-arm ./cmd/nitpick
	GOOS=darwin GOARCH=amd64 go build -o bin/nitpick-darwin-amd64 ./cmd/nitpick
	GOOS=darwin GOARCH=arm64 go build -o bin/nitpick-darwin-arm64 ./cmd/nitpick
	GOOS=windows GOARCH=amd64 go build -o bin/nitpick-windows-amd64.exe ./cmd/nitpick
	GOOS=windows GOARCH=arm64 go build -o bin/nitpick-windows-arm64.exe ./cmd/nitpick

# Help message
help:
	@echo "Available commands:"
	@echo "  build      Build the application"
	@echo "  run        Run the application directly"
	@echo "  clean      Clean build artifacts"
	@echo "  deps       Install and tidy dependencies"
	@echo "  build-all  Build for multiple platforms"
	@echo "  help       Show this help message" 
