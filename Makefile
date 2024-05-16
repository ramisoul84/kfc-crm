.PHONY: help run build

# Application
APP_NAME=kfc-crm
BINARY_NAME=crm

help:
	@echo "Available targets:"
	@echo "  run     Run the server"
	@echo "  build   Build binary to bin/crm"
	@echo "  test    Run tests"
	@echo "  clean   Remove build artifacts"

# Run the application
run:
	@go run cmd/crm/main.go

# Build the application
build:
	@mkdir -p bin
	@go build -o bin/$(BINARY_NAME) cmd/crm/main.go
	@echo "built bin/crm"
	