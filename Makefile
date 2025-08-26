# Makefile for Tubbym Backend

# Variables
BINARY_NAME=tubbym-backend
BUILD_DIR=./bin
CMD_DIR=./cmd/api
MIGRATION_DIR=./internal/db/migrations
DB_NAME=data.db
GOOSE_DRIVER=sqlite3
GOOSE_DBSTRING=./$(DB_NAME)

# Colors for output
RED=\033[0;31m
GREEN=\033[0;32m
YELLOW=\033[1;33m
BLUE=\033[0;34m
NC=\033[0m # No Color

.PHONY: help build run dev clean test deps goose-up goose-down goose-status goose-create-migration goose-reset docker-build docker-run

# Default target
all: build

## help: Show this help message
help:
	@echo "$(BLUE)Tubbym Backend Makefile$(NC)"
	@echo ""
	@echo "$(YELLOW)Available targets:$(NC)"
	@grep -E '^## [a-zA-Z_-]+:.*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = "## "}; {printf "  $(GREEN)%-20s$(NC) %s\n", $$2, $$3}' | \
		sed 's/: /\t/'

## deps: Download and tidy dependencies
deps:
	@echo "$(BLUE)Downloading dependencies...$(NC)"
	go mod download
	go mod tidy
	@echo "$(GREEN)Dependencies updated!$(NC)"

## build: Build the binary
build: deps
	@echo "$(BLUE)Building $(BINARY_NAME)...$(NC)"
	@mkdir -p $(BUILD_DIR)
	go build -ldflags="-s -w" -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR)
	@echo "$(GREEN)Binary built: $(BUILD_DIR)/$(BINARY_NAME)$(NC)"

## run: Run the built binary
run: build
	@echo "$(BLUE)Running $(BINARY_NAME)...$(NC)"
	$(BUILD_DIR)/$(BINARY_NAME)

## dev: Run the application in development mode (with live reload using air if available)
dev:
	@if command -v air > /dev/null; then \
		echo "$(BLUE)Starting development server with air...$(NC)"; \
		air; \
	else \
		echo "$(YELLOW)Air not found. Installing air...$(NC)"; \
		go install github.com/air-verse/air@latest; \
		echo "$(BLUE)Starting development server with air...$(NC)"; \
		air; \
	fi

## dev-simple: Run the application in development mode without live reload
dev-simple:
	@echo "$(BLUE)Starting development server...$(NC)"
	go run $(CMD_DIR)/main.go

## clean: Clean build artifacts
clean:
	@echo "$(BLUE)Cleaning build artifacts...$(NC)"
	rm -rf $(BUILD_DIR)
	go clean
	@echo "$(GREEN)Clean complete!$(NC)"

## test: Run tests
test:
	@echo "$(BLUE)Running tests...$(NC)"
	go test -v ./...
	@echo "$(GREEN)Tests complete!$(NC)"

## test-coverage: Run tests with coverage
test-coverage:
	@echo "$(BLUE)Running tests with coverage...$(NC)"
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)Coverage report generated: coverage.html$(NC)"

## lint: Run linter (requires golangci-lint)
lint:
	@if command -v golangci-lint > /dev/null; then \
		echo "$(BLUE)Running linter...$(NC)"; \
		golangci-lint run; \
	else \
		echo "$(YELLOW)golangci-lint not found. Please install it:$(NC)"; \
		echo "  go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

## fmt: Format Go code
fmt:
	@echo "$(BLUE)Formatting Go code...$(NC)"
	go fmt ./...
	@echo "$(GREEN)Code formatted!$(NC)"

## vet: Run go vet
vet:
	@echo "$(BLUE)Running go vet...$(NC)"
	go vet ./...
	@echo "$(GREEN)Vet complete!$(NC)"

## install-goose: Install goose migration tool
install-goose:
	@if ! command -v goose > /dev/null; then \
		echo "$(BLUE)Installing goose...$(NC)"; \
		go install github.com/pressly/goose/v3/cmd/goose@latest; \
		echo "$(GREEN)Goose installed!$(NC)"; \
	else \
		echo "$(GREEN)Goose is already installed$(NC)"; \
	fi

# Get the full path to goose binary
GOOSE_BIN := $(shell command -v goose 2>/dev/null || echo "$$(go env GOPATH)/bin/goose")

## goose-status: Check migration status
goose-status: install-goose
	@echo "$(BLUE)Checking migration status...$(NC)"
	$(GOOSE_BIN) -dir $(MIGRATION_DIR) $(GOOSE_DRIVER) $(GOOSE_DBSTRING) status

## goose-up: Run all pending migrations
goose-up: install-goose
	@echo "$(BLUE)Running migrations up...$(NC)"
	$(GOOSE_BIN) -dir $(MIGRATION_DIR) $(GOOSE_DRIVER) $(GOOSE_DBSTRING) up
	@echo "$(GREEN)Migrations complete!$(NC)"

## goose-up-one: Run one migration up
goose-up-one: install-goose
	@echo "$(BLUE)Running one migration up...$(NC)"
	$(GOOSE_BIN) -dir $(MIGRATION_DIR) $(GOOSE_DRIVER) $(GOOSE_DBSTRING) up-by-one

## goose-down: Roll back one migration
goose-down: install-goose
	@echo "$(YELLOW)Rolling back one migration...$(NC)"
	$(GOOSE_BIN) -dir $(MIGRATION_DIR) $(GOOSE_DRIVER) $(GOOSE_DBSTRING) down

## goose-down-to: Roll back to specific version (usage: make goose-down-to VERSION=20250814160324)
goose-down-to: install-goose
	@if [ -z "$(VERSION)" ]; then \
		echo "$(RED)Error: VERSION is required. Usage: make goose-down-to VERSION=20250814160324$(NC)"; \
		exit 1; \
	fi
	@echo "$(YELLOW)Rolling back to version $(VERSION)...$(NC)"
	$(GOOSE_BIN) -dir $(MIGRATION_DIR) $(GOOSE_DRIVER) $(GOOSE_DBSTRING) down-to $(VERSION)

## goose-reset: Reset all migrations (WARNING: This will drop all tables)
goose-reset: install-goose
	@echo "$(RED)WARNING: This will reset all migrations and drop all tables!$(NC)"
	@read -p "Are you sure? [y/N] " -n 1 -r; \
	echo; \
	if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
		echo "$(YELLOW)Resetting database...$(NC)"; \
		$(GOOSE_BIN) -dir $(MIGRATION_DIR) $(GOOSE_DRIVER) $(GOOSE_DBSTRING) reset; \
		echo "$(GREEN)Database reset complete!$(NC)"; \
	else \
		echo "$(BLUE)Reset cancelled$(NC)"; \
	fi

## goose-create-migration: Create a new migration (usage: make goose-create-migration NAME=add_users_table)
goose-create-migration: install-goose
	@if [ -z "$(NAME)" ]; then \
		echo "$(RED)Error: NAME is required. Usage: make goose-create-migration NAME=add_users_table$(NC)"; \
		exit 1; \
	fi
	@echo "$(BLUE)Creating migration: $(NAME)$(NC)"
	$(GOOSE_BIN) -dir $(MIGRATION_DIR) create $(NAME) sql
	@echo "$(GREEN)Migration created!$(NC)"

## db-setup: Initialize database with all migrations
db-setup: goose-up
	@echo "$(GREEN)Database setup complete!$(NC)"

## docker-build: Build Docker image
docker-build:
	@echo "$(BLUE)Building Docker image...$(NC)"
	docker build -t $(BINARY_NAME) .

## docker-run: Run Docker container
docker-run:
	@echo "$(BLUE)Running Docker container...$(NC)"
	docker run -p 8080:8080 $(BINARY_NAME)

## full-setup: Complete setup for new development environment
full-setup: deps db-setup
	@echo "$(GREEN)Full setup complete! You can now run 'make dev' to start development$(NC)"

## check: Run all checks (format, vet, lint, test)
check: fmt vet lint test
	@echo "$(GREEN)All checks passed!$(NC)"

## release-build: Build optimized binary for release
release-build: clean deps
	@echo "$(BLUE)Building release binary...$(NC)"
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w -X main.version=$$(git describe --tags --always)" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(CMD_DIR)
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w -X main.version=$$(git describe --tags --always)" -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(CMD_DIR)
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="-s -w -X main.version=$$(git describe --tags --always)" -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(CMD_DIR)
	@echo "$(GREEN)Release binaries built in $(BUILD_DIR)/$(NC)"

## install-tools: Install development tools
install-tools:
	@echo "$(BLUE)Installing development tools...$(NC)"
	go install github.com/air-verse/air@latest
	go install github.com/pressly/goose/v3/cmd/goose@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "$(GREEN)Development tools installed!$(NC)"
