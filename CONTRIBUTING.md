# Contributing to Generative API Router

Thank you for your interest in contributing to the Generative API Router! This document provides guidelines and information for contributors.

## Table of Contents

- [Development Setup](#development-setup)
- [Development Workflow](#development-workflow)
- [Code Quality Standards](#code-quality-standards)
- [Testing Guidelines](#testing-guidelines)
- [Pull Request Process](#pull-request-process)
- [CI/CD Pipeline](#cicd-pipeline)
- [Security Guidelines](#security-guidelines)

## Development Setup

### Prerequisites

- **Go**: Version 1.24.3 or later
- **Docker**: For containerized testing
- **Make**: For build automation
- **Git**: For version control

### Initial Setup

1. **Clone the repository**:
   ```bash
   git clone https://github.com/aashari/go-generative-api-router.git
   cd go-generative-api-router
   ```

2. **Setup development environment**:
   ```bash
   make setup
   ```

3. **Configure credentials** (for testing):
   ```bash
   # Edit configs/credentials.json with your API keys
   cp configs/credentials.json.example configs/credentials.json
   ```

4. **Verify setup**:
   ```bash
   make ci-check
   ```

## Development Workflow

### Branch Strategy

- **`main`**: Production-ready code
- **Feature branches**: `feat/feature-name`
- **Bug fixes**: `fix/bug-description`
- **Documentation**: `docs/update-description`

### Making Changes

1. **Create a feature branch**:
   ```bash
   git checkout -b feat/your-feature-name
   ```

2. **Make your changes** following our [code quality standards](#code-quality-standards)

3. **Run pre-commit checks**:
   ```bash
   make pre-commit
   ```

4. **Commit your changes**:
   ```bash
   git commit -m "feat: add new feature" \
     -m "- First bullet point" \
     -m "- Second bullet point" \
     -m "" \
     -m "Additional details"
   ```

5. **Push and create PR**:
   ```bash
   git push -u origin feat/your-feature-name
   gh pr create --title "feat: your feature" --body-file pr-body.md
   ```

## Code Quality Standards

### Code Style

- **Formatting**: Use `gofmt` (run `make format`)
- **Linting**: Pass `golangci-lint` checks (run `make lint`)
- **Naming**: Follow Go naming conventions
- **Comments**: Document exported functions and complex logic

### Architecture Principles

- **Separation of Concerns**: Keep handlers, business logic, and data access separate
- **Interface-Based Design**: Use interfaces for testability
- **Error Handling**: Use structured error types from `internal/errors`
- **Configuration**: Use JSON configuration files in `configs/`

### Key Components

- **Handlers** (`internal/handlers`): HTTP request handling
- **Proxy** (`internal/proxy`): Core proxy logic and vendor communication
- **Selector** (`internal/selector`): Vendor and model selection strategies
- **Validator** (`internal/validator`): Request validation
- **Config** (`internal/config`): Configuration management

## Testing Guidelines

### Test Coverage

- **Minimum Coverage**: 70% (enforced by CI)
- **Unit Tests**: Test individual functions and methods
- **Integration Tests**: Test component interactions
- **End-to-End Tests**: Test complete request flows

### Writing Tests

1. **Test Structure**:
   ```go
   func TestFunctionName(t *testing.T) {
       tests := []struct {
           name    string
           input   InputType
           want    ExpectedType
           wantErr bool
       }{
           // Test cases
       }
       
       for _, tt := range tests {
           t.Run(tt.name, func(t *testing.T) {
               // Test implementation
           })
       }
   }
   ```

2. **Use testify for assertions**:
   ```go
   require.NoError(t, err)
   assert.Equal(t, expected, actual)
   ```

3. **Mock external dependencies**:
   ```go
   // Use interfaces and dependency injection for testability
   ```

### Running Tests

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Run specific package tests
go test ./internal/selector -v
```

## Pull Request Process

### Before Creating a PR

1. **Run all checks locally**:
   ```bash
   make ci-check
   ```

2. **Ensure tests pass**:
   ```bash
   make test
   ```

3. **Update documentation** if needed

4. **Add/update tests** for new functionality

### PR Requirements

- [ ] **Clear description** using the PR template
- [ ] **All CI checks pass** (formatting, linting, tests, build)
- [ ] **Test coverage maintained** (≥70%)
- [ ] **Documentation updated** (if applicable)
- [ ] **Security considerations** addressed
- [ ] **Breaking changes documented** (if applicable)

### Review Process

1. **Automated checks** must pass
2. **Code review** by maintainers
3. **Manual testing** (if required)
4. **Approval** from at least one maintainer
5. **Merge** to main branch

## CI/CD Pipeline

### Automated Checks

Our CI pipeline runs on every PR and includes:

1. **Code Quality**:
   - Go module verification
   - Code formatting check (`gofmt`)
   - Linting (`golangci-lint`)

2. **Testing**:
   - Unit tests with race detection
   - Coverage reporting (minimum 70%)
   - Integration tests

3. **Security**:
   - Security scanning (`gosec`)
   - Dependency vulnerability checks

4. **Build & Deploy**:
   - Application build verification
   - Docker image build test
   - Deployment readiness check

### Local CI Simulation

Run the same checks locally:

```bash
# Run all CI checks
make ci-check

# Individual checks
make format-check
make lint
make test-coverage
make build
make security-scan
```

### Pipeline Configuration

- **Workflow**: `.github/workflows/ci.yml`
- **Go Version**: 1.24.3
- **Test Timeout**: 5 minutes
- **Coverage Threshold**: 70%

## Security Guidelines

### Sensitive Data

- **Never commit** API keys, credentials, or secrets
- **Use environment variables** or secure secret management
- **Scan for sensitive data** before committing:
  ```bash
  # Check for AWS account IDs
  grep -r -E '\b[0-9]{12}\b' --exclude-dir={.git,vendor} .
  
  # Check for API keys
  grep -r -E '(AKIA|ASIA|sk-|api[_-]?key)' --exclude-dir={.git,vendor} .
  ```

### Security Best Practices

- **Input validation** for all user inputs
- **Error handling** without exposing sensitive information
- **Dependency management** with regular updates
- **Security scanning** with `gosec`

### Reporting Security Issues

Please report security vulnerabilities privately to the maintainers rather than creating public issues.

## Getting Help

### Resources

- **Documentation**: Check the `docs/` directory
- **Examples**: See `examples/` for usage examples
- **Issues**: Search existing issues before creating new ones
- **Discussions**: Use GitHub Discussions for questions

### Contact

- **GitHub Issues**: For bugs and feature requests
- **GitHub Discussions**: For questions and general discussion
- **Pull Requests**: For code contributions

## Code of Conduct

Please be respectful and professional in all interactions. We're committed to providing a welcoming and inclusive environment for all contributors.

---

Thank you for contributing to the Generative API Router! 🚀 