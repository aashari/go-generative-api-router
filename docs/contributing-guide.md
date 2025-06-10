# Contributing Guide

Thank you for your interest in contributing! This guide will help you get started with contributing to the project.

## 🚀 Getting Started

### Prerequisites
- **Go 1.21+** installed
- **Git** configured with your GitHub account
- **Make** for build automation
- **golangci-lint** for code quality checks
- **Python 3.8+** with markitdown for file processing (automatically installed via setup)

### Initial Setup
1. **Fork the repository** on GitHub
2. **Clone your fork**:
   ```bash
   git clone https://github.com/yourusername/go-generative-api-router.git
   cd go-generative-api-router
   ```
3. **Add upstream remote**:
   ```bash
   git remote add upstream https://github.com/aashari/go-generative-api-router.git
   ```
4. **Run setup**:
   ```bash
   make setup
   ```

## 🔄 Development Workflow

### 1. Create a Feature Branch
```bash
# Update your main branch
git checkout main
git pull upstream main

# Create feature branch
git checkout -b feat/your-feature-name
```

### 2. Make Your Changes
- Follow the [coding standards](#-coding-standards) below
- Write tests for new functionality
- Update documentation as needed
- Run tests frequently: `make test`

### 3. Pre-commit Checks
Before committing, ensure all checks pass:
```bash
# Format code
make format

# Run linter
make lint

# Run all tests
make test

# Build to ensure no compilation errors
make build

# Run full CI check
make ci-check
```

### 4. Commit Your Changes
Use conventional commit messages:
```bash
# Examples of good commit messages
git commit -m "feat: add vendor filtering support"
git commit -m "fix: resolve streaming response timeout"
git commit -m "docs: update API documentation"
git commit -m "test: add unit tests for selector package"
```

**Commit Types:**
- `feat:` - New features
- `fix:` - Bug fixes
- `docs:` - Documentation changes
- `test:` - Adding or updating tests
- `refactor:` - Code refactoring
- `perf:` - Performance improvements
- `chore:` - Maintenance tasks

### 5. Push and Create Pull Request
```bash
# Push your branch
git push origin feat/your-feature-name

# Create PR using GitHub CLI (optional)
gh pr create --title "feat: your feature description" --body "Detailed description of changes"
```

## 📋 Pull Request Guidelines

### PR Requirements
- [ ] **Clear title** following conventional commit format
- [ ] **Detailed description** explaining what and why
- [ ] **Tests included** for new functionality
- [ ] **Documentation updated** if needed
- [ ] **All CI checks passing**
- [ ] **No merge conflicts** with main branch

### PR Template
When creating a PR, include:

```markdown
## Summary
Brief description of what this PR does.

## Changes
- List of specific changes made
- Any breaking changes (if applicable)

## Testing
- [ ] Unit tests added/updated
- [ ] Manual testing completed
- [ ] All existing tests pass

## Documentation
- [ ] Code comments added where needed
- [ ] Documentation updated (if applicable)
- [ ] Examples updated (if applicable)

## Checklist
- [ ] Code follows project standards
- [ ] Tests pass locally
- [ ] Linting passes
- [ ] No sensitive data in commits
```

### Review Process
1. **Automated checks** must pass (CI/CD pipeline)
2. **Code review** by maintainers
3. **Address feedback** if requested
4. **Approval and merge** by maintainers

## 🎯 Coding Standards

### Go Code Style
- **Follow Go conventions**: Use `gofmt`, `go vet`
- **Naming**: Use clear, descriptive names
- **Comments**: Document public APIs and complex logic
- **Error handling**: Use structured errors from `internal/errors`
- **Testing**: Aim for >80% test coverage

### Code Quality Checks
```bash
# Format code (required before commit)
make format

# Run linter (must pass)
make lint

# Check test coverage
make test-coverage
```

### Project-Specific Guidelines

#### 1. **Transparent Proxy Principle**
- Always preserve original model names in responses
- Maintain OpenAI API compatibility
- Log vendor selection decisions clearly

#### 2. **Error Handling**
```go
// Use structured errors
import "github.com/aashari/go-generative-api-router/internal/errors"

if err != nil {
    return errors.NewValidationError("invalid request format", err)
}
```

#### 3. **Logging**
```go
// Use structured logging with context
import "github.com/aashari/go-generative-api-router/internal/logger"

logger.InfoCtx(ctx, "Processing request", "vendor", vendor, "model", model)
```

#### 4. **Testing**
```go
// Use table-driven tests
func TestVendorSelection(t *testing.T) {
    tests := []struct {
        name     string
        input    SelectionRequest
        expected string
        wantErr  bool
    }{
        // Test cases...
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation...
        })
    }
}
```

## 🧪 Testing Guidelines

### Writing Tests
- **Unit tests**: Test individual functions/methods
- **Integration tests**: Test component interactions
- **Table-driven tests**: For multiple scenarios
- **Error cases**: Test both success and failure paths
- **Edge cases**: Test boundary conditions

### Test Structure
```go
func TestFunctionName(t *testing.T) {
    // Arrange
    input := setupTestData()
    
    // Act
    result, err := FunctionUnderTest(input)
    
    // Assert
    assert.NoError(t, err)
    assert.Equal(t, expected, result)
}
```

### Running Tests
```bash
# All tests
make test

# Specific package
go test ./internal/selector

# With coverage
make test-coverage

# With race detection
go test -race ./...
```

## 📝 Documentation Guidelines

### What to Document
- **Public APIs**: All exported functions and types
- **Complex logic**: Non-obvious implementation details
- **Configuration**: New config options or changes
- **Examples**: Usage examples for new features

### Documentation Standards
- **Clear and concise**: Easy to understand
- **Up-to-date**: Reflects current implementation
- **Examples included**: Show practical usage
- **Proper formatting**: Use markdown effectively

## 🐛 Reporting Issues

### Bug Reports
Include:
- **Clear description** of the issue
- **Steps to reproduce** the problem
- **Expected vs actual behavior**
- **Environment details** (Go version, OS, etc.)
- **Logs or error messages** if available

### Feature Requests
Include:
- **Use case description** - why is this needed?
- **Proposed solution** - how should it work?
- **Alternatives considered** - other approaches
- **Implementation ideas** - if you have any

## 🔒 Security

### Reporting Security Issues
- **Do not** open public issues for security vulnerabilities
- **Email** security issues to the maintainers
- **Include** detailed description and reproduction steps

### Security Guidelines
- **Never commit** API keys or sensitive data
- **Use environment variables** for configuration
- **Validate all inputs** thoroughly
- **Follow security best practices** for Go

## 📚 Additional Resources

- **[Development Guide](development-guide.md)** - Complete development setup
- **[Testing Guide](testing-guide.md)** - Detailed testing information
- **[Logging Guide](logging-guide.md)** - Logging system details
- **[API Reference](api-reference.md)** - Complete API documentation

## 🤝 Community

### Getting Help
- **GitHub Issues** - For bugs and feature requests
- **GitHub Discussions** - For questions and general discussion
- **Code Review** - Learn from PR feedback

### Code of Conduct
- **Be respectful** and inclusive
- **Provide constructive feedback**
- **Help others** learn and contribute
- **Follow GitHub's community guidelines**

---

**Thank you for contributing!** Your efforts help make this project better for everyone. 🎉