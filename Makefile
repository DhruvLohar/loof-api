.PHONY: build run dev fmt lint test help create-admin

BINARY_NAME=loof-api
MAIN_PATH=./cmd/api

help:
	@echo "Available targets:"
	@echo "  make build        - Build the application"
	@echo "  make run          - Run the compiled binary"
	@echo "  make dev          - Run in development mode (with auto-reload)"
	@echo "  make fmt          - Format code"
	@echo "  make lint         - Run linter"
	@echo "  make test         - Run tests"
	@echo "  make create-admin - Create an admin user interactively"
	@echo "  make clean        - Remove compiled binary"

build:
	@echo "Building $(BINARY_NAME)..."
	go build -o bin/$(BINARY_NAME) $(MAIN_PATH)/main.go

run: build
	@echo "Running $(BINARY_NAME)..."
	./build/$(BINARY_NAME)

dev:
	@echo "Starting in development mode..."
	go run cmd/api/main.go

fmt:
	go fmt ./...

lint:
	@which golangci-lint > /dev/null || (echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	golangci-lint run ./...

test:
	go test -v ./...

create-admin:
	@go run cmd/script/create-admin-user.go

clean:
	@echo "Cleaning up..."
	rm -f bin/$(BINARY_NAME)
	go clean

.DEFAULT_GOAL := help
