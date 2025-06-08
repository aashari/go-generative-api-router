.PHONY: build run clean docker-build docker-run lint format setup deploy

# Variables
BINARY_NAME=server
BUILD_DIR=build
CONFIG_DIR=configs
DOCKER_COMPOSE=deployments/docker/docker-compose.yml

# Colors for output
GREEN=\033[0;32m
RED=\033[0;31m
NC=\033[0m # No Color

# Build the application
build:
	@echo "$(GREEN)Building...$(NC)"
	@mkdir -p ${BUILD_DIR}
	@go build -o ${BUILD_DIR}/${BINARY_NAME} cmd/server/main.go
	@echo "$(GREEN)Build complete: ${BUILD_DIR}/${BINARY_NAME}$(NC)"

# Run the application
run: build
	@echo "$(GREEN)Running application...$(NC)"
	@./${BUILD_DIR}/${BINARY_NAME}

# Run without building
run-dev:
	@echo "$(GREEN)Running application in development mode...$(NC)"
	@go run cmd/server/main.go

# Clean build artifacts
clean:
	@echo "$(GREEN)Cleaning...$(NC)"
	@rm -rf ${BUILD_DIR}
	@echo "$(GREEN)Clean complete$(NC)"

# Docker operations
docker-build:
	@echo "$(GREEN)Building Docker image...$(NC)"
	@docker-compose -f ${DOCKER_COMPOSE} build

docker-run:
	@echo "$(GREEN)Running with Docker...$(NC)"
	@docker-compose -f ${DOCKER_COMPOSE} up

docker-stop:
	@echo "$(GREEN)Stopping Docker containers...$(NC)"
	@docker-compose -f ${DOCKER_COMPOSE} down

# Lint the code
lint:
	@echo "Running linter..."
	@go vet ./...
	@echo "✅ Linting passed"

# Check linting (CI mode - fails on issues)
lint-check:
	@echo "Running linter check..."
	@go vet ./...
	@echo "✅ Linting passed"

# Format the code
format:
	@echo "$(GREEN)Formatting code...$(NC)"
	@go fmt ./...
	@echo "$(GREEN)Code formatted$(NC)"

# Setup development environment
setup:
	@echo "$(GREEN)Setting up development environment...$(NC)"
	@go mod download
	@go mod tidy
	@cp ${CONFIG_DIR}/credentials.json.example ${CONFIG_DIR}/credentials.json 2>/dev/null || echo "$(RED)Warning: credentials.json already exists, skipping copy$(NC)"
	@echo "$(GREEN)Setup complete. Please edit ${CONFIG_DIR}/credentials.json with your API keys$(NC)"

# Deploy to AWS
deploy:
	@echo "$(GREEN)Deploying to AWS...$(NC)"
	@./scripts/deploy.sh

format-check:
	@echo "$(GREEN)Checking code formatting...$(NC)"
	@unformatted=$$(gofmt -l .); \
	if [ -n "$$unformatted" ]; then \
		echo "$(RED)The following files are not properly formatted:$(NC)"; \
		echo "$$unformatted"; \
		echo "$(RED)Please run 'make format' to fix formatting issues.$(NC)"; \
		exit 1; \
	fi
	@echo "$(GREEN)✅ All files are properly formatted$(NC)"

security-scan:
	@echo "$(GREEN)Running security scan...$(NC)"
	@if command -v gosec >/dev/null 2>&1; then \
		gosec ./... || echo "$(RED)Security issues found. Please review and fix.$(NC)"; \
	else \
		echo "$(RED)gosec not installed. Installing...$(NC)"; \
		go install github.com/securego/gosec/v2/cmd/gosec@latest; \
		export PATH=$$PATH:$$(go env GOPATH)/bin && gosec ./... || echo "$(RED)Security issues found. Please review and fix.$(NC)"; \
	fi
	@echo "$(GREEN)✅ Security scan completed$(NC)"

# Help
help:
	@echo "Available targets:"
	@echo "  $(GREEN)build$(NC)         - Build the application"
	@echo "  $(GREEN)run$(NC)           - Build and run the application"
	@echo "  $(GREEN)run-dev$(NC)       - Run without building (using go run)"
	@echo "  $(GREEN)clean$(NC)         - Clean build artifacts"
	@echo "  $(GREEN)docker-build$(NC)  - Build Docker image"
	@echo "  $(GREEN)docker-run$(NC)    - Run with Docker Compose"
	@echo "  $(GREEN)docker-stop$(NC)   - Stop Docker containers"
	@echo "  $(GREEN)lint$(NC)          - Run linter"
	@echo "  $(GREEN)format$(NC)        - Format code"
	@echo "  $(GREEN)format-check$(NC)  - Check code formatting without fixing"
	@echo "  $(GREEN)security-scan$(NC) - Run security scanner"
	@echo "  $(GREEN)setup$(NC)         - Setup development environment"
	@echo "  $(GREEN)deploy$(NC)        - Deploy to AWS"
	@echo "  $(GREEN)swagger-generate$(NC) - Generate Swagger documentation"
	@echo "  $(GREEN)help$(NC)          - Show this help message"

# Log management
logs-dir:
	@mkdir -p logs

run-with-logs: build logs-dir
	@echo "$(GREEN)Running application with logging...$(NC)"
	@./${BUILD_DIR}/${BINARY_NAME} 2>&1 | tee logs/server-$(shell date +%Y%m%d-%H%M%S).log

clean-logs:
	@echo "$(GREEN)Cleaning log files...$(NC)"
	@rm -f logs/*.log
	@rm -f *.log
	@echo "$(GREEN)Logs cleaned$(NC)"

# Generate Swagger documentation
swagger-generate:
	@echo "$(GREEN)Generating Swagger documentation...$(NC)"
	@$(shell go env GOPATH)/bin/swag init -g cmd/server/main.go --output docs/api/ --parseDependency --parseInternal
	@cp docs/api/swagger.json docs/
	@cp docs/api/swagger.yaml docs/
	@echo "$(GREEN)Swagger documentation generated and copied to docs/$(NC)" 