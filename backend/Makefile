.PHONY: help build run test clean docker-up docker-down migrate-up migrate-down

# Variables
APP_NAME=aegis-v14-api
GO=go
GOFLAGS=-v
BUILD_DIR=./bin
MAIN_PATH=./cmd/api
DOCKER_COMPOSE=docker-compose

# Help
help: ## Show this help message
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Build
build: ## Build the application
	@echo "Building $(APP_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(APP_NAME) $(MAIN_PATH)

# Run
run: ## Run the application
	@echo "Running $(APP_NAME)..."
	$(GO) run $(MAIN_PATH)

# Run with air (hot reload)
dev: ## Run with hot reload (requires air)
	@which air > /dev/null || (echo "air not found. Install: go install github.com/cosmtrek/air@latest" && exit 1)
	air

# Test
test: ## Run tests
	$(GO) test -v -race -coverprofile=coverage.out ./...

test-coverage: test ## Run tests with coverage report
	$(GO) tool cover -html=coverage.out

# Dependencies
deps: ## Download dependencies
	$(GO) mod download
	$(GO) mod tidy

# Clean
clean: ## Clean build artifacts
	@echo "Cleaning..."
	rm -rf $(BUILD_DIR)
	rm -f coverage.out

# Docker
docker-up: ## Start Docker containers
	$(DOCKER_COMPOSE) up -d

docker-down: ## Stop Docker containers
	$(DOCKER_COMPOSE) down

docker-logs: ## Show Docker logs
	$(DOCKER_COMPOSE) logs -f

docker-ps: ## Show running containers
	$(DOCKER_COMPOSE) ps

# Database
db-init: ## Initialize database (run scripts/db/01 and 02)
	@echo "Initializing database..."
	psql -U postgres -f ../scripts/db/01_create_database.sql
	psql -U aegis_v14 -d aegis_v14 -f ../scripts/db/02_create_schemas.sql
	@echo "Database initialized!"

db-check: ## Check database permissions
	psql -U aegis_v14 -d aegis_v14 -f ../scripts/db/03_check_permissions.sql

db-fix: ## Fix database permissions
	psql -U aegis_v14 -d aegis_v14 -f ../scripts/db/04_fix_permissions.sql

migrate-up: ## Run database migrations
	@which migrate > /dev/null || (echo "golang-migrate not found. Install: brew install golang-migrate" && exit 1)
	migrate -path migrations -database "postgresql://aegis_v14:aegis_v14_won@localhost:5432/aegis_v14?sslmode=disable" up

migrate-down: ## Rollback database migrations
	migrate -path migrations -database "postgresql://aegis_v14:aegis_v14_won@localhost:5432/aegis_v14?sslmode=disable" down 1

migrate-create: ## Create a new migration (usage: make migrate-create NAME=create_users_table)
	@test -n "$(NAME)" || (echo "NAME is required. Usage: make migrate-create NAME=create_users_table" && exit 1)
	migrate create -ext sql -dir migrations -seq $(NAME)

# Linting
lint: ## Run linter
	@which golangci-lint > /dev/null || (echo "golangci-lint not found. Install: brew install golangci-lint" && exit 1)
	golangci-lint run

# Format
fmt: ## Format code
	$(GO) fmt ./...

# Generate
generate: ## Run go generate
	$(GO) generate ./...

# Wire (dependency injection)
wire: ## Generate wire DI code
	@which wire > /dev/null || (echo "wire not found. Install: go install github.com/google/wire/cmd/wire@latest" && exit 1)
	cd cmd/api && wire

# All
all: clean deps build test ## Clean, download deps, build, and test
