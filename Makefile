.PHONY: build run clean docker-build docker-run test

# Default build directory
BUILD_DIR=./bin

# Build the application
build:
	mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/shorter ./cmd/app

# Run the application
run:
	go run ./cmd/app/main.go

# Clean build artifacts
clean:
	rm -rf $(BUILD_DIR)

# Build Docker image
docker-build:
	docker build -t shorter:latest .

# Run Docker container
docker-run:
	docker run -p 8080:8080 shorter:latest

# Run tests
test:
	go test -v ./... 