.PHONY: build run dev fmt lint test help create-admin \
	migrate-up migrate-down migrate-status migrate-create

BINARY_NAME=loof-api
MAIN_PATH=./cmd/api
MIGRATIONS_DIR=./migrations

# Which env file the migrate targets read credentials from. Override for
# production: make migrate-up ENV_FILE=.env.prod
ENV_FILE ?= .env

# Sources ENV_FILE and builds the DSN from the same vars the app uses.
GOOSE = set -a && . ./$(ENV_FILE) && set +a && \
	goose -dir $(MIGRATIONS_DIR) postgres \
	"postgresql://$$DB_USER:$$DB_PASS@$$DB_HOST:$$DB_PORT/$$DB_NAME?sslmode=$${DB_SSLMODE:-require}"

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
	@echo ""
	@echo "  make migrate-up      - Apply pending migrations"
	@echo "  make migrate-down    - Roll back the last migration"
	@echo "  make migrate-status  - Show applied/pending migrations"
	@echo "  make migrate-create name=add_foo - Scaffold a new migration"
	@echo "  (append ENV_FILE=.env.prod to target production)"

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

migrate-up:
	@test -f ./$(ENV_FILE) || (echo "$(ENV_FILE) not found" && exit 1)
	@$(GOOSE) up

migrate-down:
	@test -f ./$(ENV_FILE) || (echo "$(ENV_FILE) not found" && exit 1)
	@$(GOOSE) down

migrate-status:
	@test -f ./$(ENV_FILE) || (echo "$(ENV_FILE) not found" && exit 1)
	@$(GOOSE) status

migrate-create:
	@test -n "$(name)" || (echo "usage: make migrate-create name=add_foo" && exit 1)
	goose -dir $(MIGRATIONS_DIR) create $(name) sql

clean:
	@echo "Cleaning up..."
	rm -f bin/$(BINARY_NAME)
	go clean

.DEFAULT_GOAL := help
