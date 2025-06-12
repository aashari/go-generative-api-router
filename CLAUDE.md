# Claude AI Development Guide for Generative API Router

This guide provides essential context for AI assistants working on the Generative API Router project. It consolidates information from `.cursorrules` (if they existed), documentation, and development practices.

## 🏗️ Project Overview

### Core Identity: Multi-Vendor OpenAI-Compatible Router

**CRITICAL UNDERSTANDING**: This service is NOT just an OpenAI service with some vendor support. It's a **true multi-vendor OpenAI-compatible router** designed from the ground up for multiple vendors.

- **Primary Function**: Transparent proxy providing unified OpenAI-compatible API for multiple LLM vendors
- **Current Scale**: 19 credentials (18 Gemini + 1 OpenAI) with 5 models
- **Architecture**: 95 total vendor-credential-model combinations with even distribution
- **Core Principle**: Preserves original model names in responses while intelligently routing to vendors

### Enterprise-Grade Improvements (2024)
- 🔒 **Security**: AES-GCM encryption, sensitive data masking
- 🔄 **Reliability**: Exponential backoff, circuit breaker patterns
- 📊 **Monitoring**: Health checks with vendor connectivity
- ⚡ **Performance**: Production-optimized logging
- 🧹 **Code Quality**: DRY principles, centralized utilities
- 🎯 **Advanced Features**: Audio/image/file processing, context-aware selection

## 🚨 Essential Development Rules

### Mandatory Approach
1. **Research First**: Always investigate existing code, patterns, and conventions
2. **Multi-Vendor Awareness**: Remember this handles multiple vendors, not just OpenAI
3. **Complete Logging**: Log full data without truncation or redaction (with smart masking in production)
4. **Environment Variables**: Always use proper environment variables for AWS profiles, regions
5. **Sequential Thinking**: Break down complex requests systematically
6. **Monitor Logs**: Always check `logs/server.log` during development
7. **Build Verification**: Run full CI checks locally before any commit

### Development Commands
```bash
# Standard development workflow
make build && ./build/server > logs/server.log 2>&1 & sleep 3 && curl http://localhost:8082/health | cat

# Check for issues
tail -20 logs/server.log | grep -E "(error|Error|failed|Failed)" | cat

# Full verification
make ci-check

# Multi-vendor testing
curl -X POST "http://localhost:8082/v1/chat/completions?vendor=openai" -H "Content-Type: application/json" -d '{"model":"test","messages":[{"role":"user","content":"test"}]}' | jq | cat
curl -X POST "http://localhost:8082/v1/chat/completions?vendor=gemini" -H "Content-Type: application/json" -d '{"model":"test","messages":[{"role":"user","content":"test"}]}' | jq | cat
```

## 🗂️ Project Structure

### Core Application Files
- **`cmd/server/main.go`**: Application entry point with automatic .env loading
- **`internal/app/`**: Centralized application configuration and models
- **`internal/proxy/`**: Core proxy logic, client communication, response processing, image/audio/file processing
- **`internal/selector/`**: Vendor selection strategies (even distribution and context-aware)
- **`internal/validator/`**: Request validation and model name extraction
- **`internal/config/`**: Configuration loading, secure credentials, environment management
- **`internal/handlers/`**: HTTP handlers for API endpoints
- **`internal/middleware/`**: Request correlation and middleware
- **`internal/monitoring/`**: Performance metrics and profiling
- **`internal/router/`**: Route setup and configuration
- **`internal/types/`**: Type definitions and payload structures
- **`internal/utils/`**: Centralized utilities, sanitization, and environment helpers
- **`internal/logger/`**: Comprehensive logging system
- **`internal/errors/`**: Standardized error handling
- **`internal/filter/`**: Vendor and model filtering utilities

### Configuration Files
- **`configs/credentials.json`**: API keys for vendors (19 working credentials)
- **`configs/models.json`**: Available models and vendors (5 models)
- **`.env`**: Environment variables (use `.env.example` as template)

### Key Documentation Files
- **`.cursor/rules/development_guide.mdc`**: Complete workflow and architecture
- **`.cursor/rules/running_guide.mdc`**: Setup and testing procedures (note: actual file is `running_and_testing.mdc`)
- **`docs/development-guide.md`**: Quick start and daily commands
- **`docs/contributing-guide.md`**: PR process and coding standards
- **`docs/logging-guide.md`**: Comprehensive logging documentation
- **`docs/api-reference.md`**: Complete API documentation

## 🔧 Environment Management

### Critical Environment Setup
```bash
# STEP 1: Load environment variables from .env file (MANDATORY)
export $(cat .env | grep -v '^#' | xargs) && echo "✅ Environment loaded from .env" | cat

# STEP 2: Set up AWS cluster/service names based on SERVICE_NAME
export AWS_CLUSTER_DEV=dev-$SERVICE_NAME AWS_SERVICE_DEV=dev-$SERVICE_NAME AWS_CLUSTER_PROD=prod-$SERVICE_NAME AWS_SERVICE_PROD=prod-$SERVICE_NAME && echo "✅ AWS environment configured" | cat

# STEP 3: Verify configuration
echo "Service Name: $SERVICE_NAME" && echo "AWS Region: $AWS_REGION" && echo "AWS Profile: $AWS_PROFILE" | cat
```

### Key Environment Variables
- `SERVICE_NAME`: Service name for AWS resources
- `AWS_ACCOUNT_ID`: AWS Account ID
- `AWS_PROFILE`: AWS CLI profile  
- `AWS_REGION`: AWS region (ap-southeast-3)
- `PORT`: Application port (8082)
- `LOG_LEVEL`: Logging level (info, debug)

## 📝 Comprehensive Logging Principles

### Core Logging Rules
1. **Log Complete Data**: Always log entire objects without truncation
2. **No Cherry-Picking**: Log complete data structures, not selected attributes
3. **Smart Masking**: Production logs mask sensitive data automatically
4. **No Truncation**: Never use substring(), slice(), or size limits
5. **Complete Context**: Include all relevant objects and state
6. **Environment-Aware**: Production uses conditional detail levels

### Logging Examples
```go
// ✅ Correct comprehensive logging
logger.LogRequest(ctx, "Processing request", map[string]any{
    "complete_request": request,           // Entire request object
    "headers": headers,                    // All headers including auth
    "vendor_config": vendorConfig,         // Complete vendor configuration
    "api_keys": credentials,               // Complete credentials (masked in production)
})

// ❌ Avoid partial logging
logger.Info("Processing request", map[string]any{
    "model": request.Model,              // Only one field
    "message_count": len(request.Messages), // Derived data instead of actual messages
})
```

## 🧪 Testing Requirements

### Essential Testing Commands
```bash
# Run linting
make lint

# Run tests
make test

# Run with coverage
make test-coverage

# Full CI check
make ci-check

# Security scan
make security-scan
```

### Multi-Vendor Testing
Always verify functionality with both vendors:
```bash
# Test OpenAI vendor
curl -X POST "http://localhost:8082/v1/chat/completions?vendor=openai" \
  -H "Content-Type: application/json" \
  -d '{"model":"test-model","messages":[{"role":"user","content":"Hello"}]}' | cat

# Test Gemini vendor
curl -X POST "http://localhost:8082/v1/chat/completions?vendor=gemini" \
  -H "Content-Type: application/json" \
  -d '{"model":"test-model","messages":[{"role":"user","content":"Hello"}]}' | cat

# Monitor vendor distribution
grep "Even distribution selected combination" logs/server.log | tail -5 | cat
```

## 🚀 AWS Deployment & Monitoring

### Infrastructure Overview
- **AWS Account**: `${AWS_ACCOUNT_ID}` (from .env)
- **Region**: `ap-southeast-3` (Asia Pacific - Jakarta)
- **Architecture**: CodeBuild → ECR → ECS Fargate
- **Environments**: Separate dev and prod with automated CI/CD

### Essential AWS Commands
```bash
# Service status
aws --region $AWS_REGION ecs describe-services --cluster $AWS_CLUSTER_PROD --services $AWS_SERVICE_PROD | cat

# Recent errors (last 30 minutes)
START_TS=$(( $(date -u -d '30 minutes ago' +%s) * 1000 )) && aws logs filter-log-events --profile $AWS_PROFILE --region $AWS_REGION --log-group-name "/aws/ecs/service/$AWS_SERVICE_PROD" --start-time $START_TS --filter-pattern "ERROR" | jq -r '.events[-3:][].message' | cat

# Vendor distribution analysis
START_TS=$(( $(date -u -d '1 hour ago' +%s) * 1000 )) && aws logs filter-log-events --profile $AWS_PROFILE --region $AWS_REGION --log-group-name "/aws/ecs/service/$AWS_SERVICE_PROD" --start-time $START_TS --filter-pattern "\"Proxy request initiated\"" | jq -r '.events[].message | fromjson | .attributes.selected_vendor' | sort | uniq -c | sort -nr | cat
```

## 🔒 Security Best Practices

### Sensitive Data Management
1. **Check for sensitive data before committing**:
   ```bash
   # Check for AWS account IDs
   grep -r -E '\b[0-9]{12}\b' --exclude-dir={.git,build,logs} .
   
   # Check for AWS access keys
   grep -r -E '(AKIA|ASIA|aws_access_key|aws_secret)' --exclude-dir={.git,build,logs} .
   ```

2. **Use gitignored files for sensitive data**:
   - `configs/credentials.json`
   - `scripts/deploy.sh`
   - `.env`

3. **Leverage secure credential handling**:
   - AES-GCM encryption in `internal/config/secure.go`
   - Sensitive data masking in `internal/utils/sanitization.go`

## 📋 Common Pitfalls & Solutions

### Critical Misunderstandings
- **Multi-Vendor Nature**: This is NOT an OpenAI service with vendor support - it's a true multi-vendor router
- **Configuration Priority**: Service prioritizes `configs/credentials.json` over environment variables
- **Model Name Handling**: Service preserves original model names in responses while routing to actual vendor models

### Common Issues
- **Port Conflicts**: Ensure port 8082 is free (`lsof -i :8082`)
- **Server Startup**: Always add delay after starting server (`sleep 3`)
- **Terminal Commands**: Always pipe to `| cat` and use single-line commands
- **Background Processes**: Manage server processes correctly (`pgrep -f "build/server$"`)

## 🎯 Development Workflow

### Sequential Thinking Approach
1. **Analysis Phase**: Research existing patterns, understand architecture
2. **Planning Phase**: Break down requests, identify affected components
3. **Implementation Phase**: Implement one component at a time
4. **Verification Phase**: Run comprehensive checks, verify multi-vendor functionality

### Standard Workflow
```bash
# 1. Make code changes
# 2. Run linting
make lint

# 3. Format code (required before CI)
make format

# 4. Build and run locally  
make build && ./build/server > logs/server.log 2>&1 & sleep 5

# 5. Check logs for issues
tail -20 logs/server.log | grep -E "(error|Error|failed|Failed)" | cat

# 6. Verify multi-vendor functionality
curl -X POST http://localhost:8082/v1/chat/completions -H "Content-Type: application/json" -d '{"model":"test","messages":[{"role":"user","content":"test"}]}' | jq '.model' | cat

# 7. Full verification
make ci-check
```

## 📚 Documentation Quick Reference

### When to Use Which Guide
- **Architecture/Workflow**: `.cursor/rules/development_guide.mdc`
- **Setup/Testing**: `.cursor/rules/running_and_testing.mdc`
- **Daily Development**: `docs/development-guide.md`
- **Contributing**: `docs/contributing-guide.md`
- **API Integration**: `docs/user-guide.md`
- **API Reference**: `docs/api-reference.md`
- **Debugging**: `docs/logging-guide.md`
- **AWS Deployment**: `docs/deployment-guide.md`

### Key Architecture Principle
The service acts as a **transparent proxy** with model name preservation:
1. Client sends original model name
2. Router selects actual vendor/model using even distribution
3. Router sends request to vendor with actual model name
4. Router returns response with original model name restored
5. All other request/response data passes through exactly

## 🏷️ Release Process

### Version Management
```bash
# Get current version
CURRENT_VERSION=$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")

# Review changes since last release
git log $(git describe --tags --abbrev=0 2>/dev/null || echo "HEAD~10")..HEAD --oneline | cat

# Create new version (increment based on changes)
NEW_VERSION="v2.1.0"  # patch/minor/major

# Create annotated tag with release notes
git tag -a $NEW_VERSION -m "$NEW_VERSION: Brief release title" -m "Detailed features/fixes"

# Push and create GitHub release
git push origin $NEW_VERSION
gh release create $NEW_VERSION --title "🚀 Generative API Router $NEW_VERSION" --notes-file release-notes.md
```

---

**Remember**: This is a production-ready multi-vendor service with enterprise-grade features. Always consider the multi-vendor nature, test with both vendors, monitor logs comprehensively, and follow security best practices.