# Mock BI-FAST Service Makefile

.PHONY: help build run test clean docker-build docker-up docker-down migrate lint format

# Variables
APP_NAME=mock-bifast-service
VERSION?=1.0.0
DOCKER_IMAGE=rezkyauliapratama/$(APP_NAME)
GO=go
GOFLAGS=-v

# Colors for terminal output
RED=\033[0;31m
GREEN=\033[0;32m
YELLOW=\033[1;33m
NC=\033[0m # No Color

help: ## Show this help message
	@echo "$(GREEN)Mock BI-FAST Service - Makefile Commands$(NC)"
	@echo ""
	@awk 'BEGIN {FS = ":.*##"; printf "Usage:\n  make $(YELLOW)<target>$(NC)\n\nTargets:\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  $(YELLOW)%-15s$(NC) %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

deps: ## Download Go dependencies
	@echo "$(GREEN)Downloading dependencies...$(NC)"
	$(GO) mod download
	$(GO) mod tidy

build: deps ## Build the application
	@echo "$(GREEN)Building application...$(NC)"
	$(GO) build $(GOFLAGS) -o bin/server ./cmd/server
	@echo "$(GREEN)Build completed: bin/server$(NC)"

run: ## Run the application locally
	@echo "$(GREEN)Starting application...$(NC)"
	$(GO) run ./cmd/server/main.go

dev: ## Run with hot reload (requires air)
	@echo "$(GREEN)Starting development server with hot reload...$(NC)"
	@command -v air >/dev/null 2>&1 || { echo "$(RED)air is not installed. Run: go install github.com/cosmtrek/air@latest$(NC)"; exit 1; }
	air

test: ## Run tests
	@echo "$(GREEN)Running tests...$(NC)"
	$(GO) test -v -race -cover ./...

test-coverage: ## Run tests with coverage report
	@echo "$(GREEN)Running tests with coverage...$(NC)"
	$(GO) test -v -race -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)Coverage report generated: coverage.html$(NC)"

lint: ## Run linter
	@echo "$(GREEN)Running linter...$(NC)"
	@command -v golangci-lint >/dev/null 2>&1 || { echo "$(RED)golangci-lint is not installed. See: https://golangci-lint.run/usage/install/$(NC)"; exit 1; }
	golangci-lint run ./...

format: ## Format Go code
	@echo "$(GREEN)Formatting code...$(NC)"
	$(GO) fmt ./...
	goimports -w .

clean: ## Clean build artifacts
	@echo "$(YELLOW)Cleaning build artifacts...$(NC)"
	rm -rf bin/
	rm -f coverage.out coverage.html
	$(GO) clean
	@echo "$(GREEN)Clean completed$(NC)"

docker-build: ## Build Docker image
	@echo "$(GREEN)Building Docker image...$(NC)"
	docker build -t $(DOCKER_IMAGE):$(VERSION) -t $(DOCKER_IMAGE):latest .
	@echo "$(GREEN)Docker image built: $(DOCKER_IMAGE):$(VERSION)$(NC)"

docker-up: ## Start Docker Compose services
	@echo "$(GREEN)Starting Docker Compose services...$(NC)"
	docker-compose up -d
	@echo "$(GREEN)Services started$(NC)"
	@echo "$(YELLOW)Service URL: http://localhost:8080$(NC)"
	@echo "$(YELLOW)PgAdmin URL: http://localhost:5050 (run 'make docker-up-tools' first)$(NC)"

docker-up-tools: ## Start Docker Compose with tools (PgAdmin)
	@echo "$(GREEN)Starting Docker Compose services with tools...$(NC)"
	docker-compose --profile tools up -d
	@echo "$(GREEN)Services started with tools$(NC)"
	@echo "$(YELLOW)Service URL: http://localhost:8080$(NC)"
	@echo "$(YELLOW)PgAdmin URL: http://localhost:5050$(NC)"

docker-down: ## Stop Docker Compose services
	@echo "$(YELLOW)Stopping Docker Compose services...$(NC)"
	docker-compose down
	@echo "$(GREEN)Services stopped$(NC)"

docker-logs: ## View Docker Compose logs
	docker-compose logs -f

docker-restart: docker-down docker-up ## Restart Docker Compose services

migrate-up: ## Run database migrations
	@echo "$(GREEN)Running database migrations...$(NC)"
	@command -v migrate >/dev/null 2>&1 || { echo "$(RED)golang-migrate is not installed. See: https://github.com/golang-migrate/migrate$(NC)"; exit 1; }
	migrate -path migrations -database "postgres://root:password@localhost:5432/bifast?sslmode=disable" up
	@echo "$(GREEN)Migrations completed$(NC)"

migrate-down: ## Rollback database migrations
	@echo "$(YELLOW)Rolling back database migrations...$(NC)"
	migrate -path migrations -database "postgres://root:password@localhost:5432/bifast?sslmode=disable" down
	@echo "$(GREEN)Rollback completed$(NC)"

db-reset: ## Reset database (drop and recreate)
	@echo "$(RED)Resetting database...$(NC)"
	docker-compose exec postgres psql -U postgres -c "DROP DATABASE IF EXISTS mockbifast;"
	docker-compose exec postgres psql -U postgres -c "CREATE DATABASE mockbifast;"
	docker-compose exec postgres psql -U postgres -d mockbifast -f /docker-entrypoint-initdb.d/001_create_transactions.sql
	@echo "$(GREEN)Database reset completed$(NC)"

install-tools: ## Install development tools
	@echo "$(GREEN)Installing development tools...$(NC)"
	go install github.com/cosmtrek/air@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install golang.org/x/tools/cmd/goimports@latest
	@echo "$(GREEN)Tools installed$(NC)"

api-test: ## Test API endpoints using curl
	@echo "$(GREEN)Testing API endpoints...$(NC)"
	@echo "\n$(YELLOW)1. Health Check:$(NC)"
	@curl -s http://localhost:8080/health | jq .
	@echo "\n$(YELLOW)2. Account Inquiry:$(NC)"
	@curl -s -X POST http://localhost:8080/api/v1/account-inquiry \
		-H "Content-Type: application/json" \
		-d '{"bankCode":"CENAIDJA","accountNumber":"1234567890","referenceId":"REF-TEST-001"}' | jq .
	@echo "\n$(YELLOW)3. Statistics (Admin):$(NC)"
	@curl -s http://localhost:8080/api/v1/admin/statistics \
		-H "Authorization: Bearer your-secure-admin-token-change-in-production" | jq .

.DEFAULT_GOAL := help
