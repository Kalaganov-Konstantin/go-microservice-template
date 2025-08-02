# 🚀 Microservice Makefile

.PHONY: help build test lint docker-up docker-down migrate-up

# Colors for output
GREEN  := \033[0;32m
YELLOW := \033[0;33m
BLUE   := \033[0;34m
RED    := \033[0;31m
CYAN   := \033[0;36m
BOLD   := \033[1m
NC     := \033[0m # No Color

# Default target
help: ## 📖 Show this help message
	@printf '$(CYAN)$(BOLD)🚀 Microservice Development Tools$(NC)\n'
	@printf '\n'
	@printf '$(BOLD)Usage:$(NC) make [target]\n'
	@printf '\n'
	@printf '$(BOLD)Available targets:$(NC)\n'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[0;36m%-15s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# 🔨 Build
build: ## 🔨 Build the application
	@printf "$(YELLOW)🔨 Building application...$(NC)\n"
	go build -o bin/main cmd/http-server/main.go
	@printf "$(GREEN)✅ Build completed: bin/main$(NC)\n"

# 🧪 Test
test: ## 🧪 Run tests
	@printf "$(CYAN)🧪 Running tests...$(NC)\n"
	go test -cover ./...

# 🔍 Code quality
lint: ## 🔍 Run linter
	@printf "$(BLUE)🔍 Running linter...$(NC)\n"
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run; \
	else \
		printf "$(RED)❌ golangci-lint not installed. Run: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest$(NC)\n"; \
	fi

# 🐳 Docker
docker-up: ## 🚀 Start services
	@printf "$(GREEN)🚀 Starting services...$(NC)\n"
	./scripts/generate-configs.sh .env
	docker-compose up -d
	@printf "$(GREEN)✅ Services started successfully$(NC)\n"

docker-down: ## 🛑 Stop services
	@printf "$(YELLOW)🛑 Stopping services...$(NC)\n"
	docker-compose down
	@printf "$(GREEN)✅ Services stopped$(NC)\n"

# 🗄️ Database
migrate-up: ## ⬆️ Run database migrations
	@printf "$(GREEN)⬆️ Running migrations...$(NC)\n"
	@migrate -path migrations -database "$(DATABASE_URL)" up
	@printf "$(GREEN)✅ Migrations completed$(NC)\n"
