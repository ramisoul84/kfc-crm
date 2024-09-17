.PHONY: run build

# Application
APP_NAME=kfc-crm
BINARY_NAME=crm

# Run the application
run:
	go run cmd/crm/main.go

# Build the application
build:
	go build -o bin/$(BINARY_NAME) cmd/crm/main.go