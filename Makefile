.PHONY: help run build proto-gen

# Application
APP_NAME=kfc-crm
BINARY_NAME=crm

help:
	@echo "Available targets:"
	@echo "  run		Run the server"
	@echo "  build		Build binary to bin/crm"
	@echo "  proto-gen	Generate protobuf files"

# Run the application
run:
	@go run cmd/crm/main.go

# Build the application
build:
	@mkdir -p bin
	@go build -o bin/$(BINARY_NAME) cmd/crm/main.go
	@echo "built bin/crm"

# Generate the protobuf files
proto-gen:
	@echo "Generating protobuf files..."
	@find proto -name '*.proto' -print0 | xargs -0 -n1 \
		protoc \
			--go_out=. \
			--go_opt=module=github.com/ramisoul84/kfc-crm \
			--go-grpc_out=. \
			--go-grpc_opt=module=github.com/ramisoul84/kfc-crm \
			--proto_path=.
	@echo "✅ Proto files generated"