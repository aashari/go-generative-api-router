.PHONY: build test run clean docker-build docker-run lint format setup deploy test-coverage

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

# Run tests
test:
	@echo "$(GREEN)Running tests...$(NC)"
	@go test -v ./...

# Run tests with coverage
test-coverage:
	@echo "$(GREEN)Running tests with coverage...$(NC)"
	@go test -cover -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)Coverage report generated: coverage.html$(NC)"

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
	@rm -f coverage.out coverage.html
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
	@echo "$(GREEN)Running linter...$(NC)"
	@golangci-lint run || (echo "$(RED)golangci-lint not installed. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest$(NC)" && exit 1)

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

# Help
help:
	@echo "Available targets:"
	@echo "  $(GREEN)build$(NC)         - Build the application"
	@echo "  $(GREEN)test$(NC)          - Run tests"
	@echo "  $(GREEN)test-coverage$(NC) - Run tests with coverage report"
	@echo "  $(GREEN)run$(NC)           - Build and run the application"
	@echo "  $(GREEN)run-dev$(NC)       - Run without building (using go run)"
	@echo "  $(GREEN)clean$(NC)         - Clean build artifacts"
	@echo "  $(GREEN)docker-build$(NC)  - Build Docker image"
	@echo "  $(GREEN)docker-run$(NC)    - Run with Docker Compose"
	@echo "  $(GREEN)docker-stop$(NC)   - Stop Docker containers"
	@echo "  $(GREEN)lint$(NC)          - Run linter"
	@echo "  $(GREEN)format$(NC)        - Format code"
	@echo "  $(GREEN)setup$(NC)         - Setup development environment"
	@echo "  $(GREEN)deploy$(NC)        - Deploy to AWS"
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