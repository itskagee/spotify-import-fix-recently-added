# Makefile
BINARY_NAME=fix

# Default target: Run tests and then build
all: test build

# Build the binary executable
build:
	@echo "Building"
	go build -o $(BINARY_NAME) main.go

# Run the tests
test:
	@echo "Running tests"
	go test -v ./...

# Run the script directly without building a binary first
run:
	@echo "Running script"
	go run main.go

# Clean up binaries
clean:
	@echo "Cleaning"
	go clean
	rm -f $(BINARY_NAME)

docker:
	docker compose run --build --service-ports --rm spotify-import-fix-recently-added