# Makefile
BINARY_NAME=fix
OUTPUT_DIR=bin

# Default target: Run tests and then build
all: test build

# Build the binary executable
build:
	@echo "Building for current system"
	go build -o $(OUTPUT_DIR)/$(BINARY_NAME) ./src

# --- Cross-Compilation Targets ---

build-windows:
	@echo "Building for Windows (x86 64-bit & ARM 64-bit)"
	GOOS=windows GOARCH=amd64 go build -o $(OUTPUT_DIR)/$(BINARY_NAME)-windows-64bit.exe ./src
	GOOS=windows GOARCH=arm64 go build -o $(OUTPUT_DIR)/$(BINARY_NAME)-windows-arm-64bit.exe ./src

build-mac:
	@echo "Building for macOS (Apple Silicon & Intel)"
	GOOS=darwin GOARCH=arm64 go build -o $(OUTPUT_DIR)/$(BINARY_NAME)-mac-apple-silicon ./src
	GOOS=darwin GOARCH=amd64 go build -o $(OUTPUT_DIR)/$(BINARY_NAME)-mac-intel ./src

build-linux:
	@echo "Building for Linux (x86 64-bit & ARM 64-bit)"
	GOOS=linux GOARCH=amd64 go build -o $(OUTPUT_DIR)/$(BINARY_NAME)-linux-64bit ./src
	GOOS=linux GOARCH=arm64 go build -o $(OUTPUT_DIR)/$(BINARY_NAME)-linux-arm-64bit ./src

# Build everything at once
build-all: clean build-windows build-mac build-linux
	@echo "All binaries generated in the /$(OUTPUT_DIR) folder!"

# --- Utilities ---

# Run the tests
test:
	@echo "Running tests"
	go test -v ./src/...

# Run the script directly without building a binary first
run:
	@echo "Running script"
	@mkdir -p $(OUTPUT_DIR)
	@cd $(OUTPUT_DIR) && go run ../src

# Clean up binaries
clean:
	@echo "Cleaning"
	go clean
	rm -rf $(OUTPUT_DIR)

docker:
	docker compose -f ./docker/compose.yml run --build --service-ports --rm spotify-import-fix-recently-added