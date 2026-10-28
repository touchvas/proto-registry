# Variables
BUF_VERSION := latest
# The gateway plugins are pinned: generated *.pb.gw.go must match the grpc-gateway runtime
# version that api-gateway-service imports.
GRPC_GATEWAY_VERSION := v2.29.0
GEN_DIR := gen/go
OPENAPI_DIR := gen/openapiv2

.PHONY: all
all: generate

## install: Install buf and required go plugins
.PHONY: install
install:
	@echo "Installing buf and plugins..."
	go install github.com/bufbuild/buf/cmd/buf@$(BUF_VERSION)
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@$(GRPC_GATEWAY_VERSION)
	go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@$(GRPC_GATEWAY_VERSION)

## generate: Clean and regenerate all proto files
.PHONY: generate
generate: clean
	@echo "Generating Go code..."
	buf generate proto
	@echo "Generation complete. Files are in $(GEN_DIR)"

## lint: Run buf linting to check for formatting/best practices
.PHONY: lint
lint:
	@echo "Linting protos..."
	buf lint

## breaking: Check for breaking changes against the main branch
.PHONY: breaking
breaking:
	@echo "Checking for breaking changes..."
	buf breaking proto --against '.git#branch=master'

## clean: Remove all generated files
.PHONY: clean
clean:
	@echo "Cleaning generated files..."
	rm -rf $(GEN_DIR) $(OPENAPI_DIR)

## help: Show this help message
.PHONY: help
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'