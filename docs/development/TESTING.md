# Testing Guide

## Running Tests
```bash
# Run all tests
make test

# Run with coverage
make test-coverage

# Run specific package tests
go test ./internal/handlers

# Run with race detection
go test -race ./...
```

## Test Structure
- Unit tests are colocated with source files (`*_test.go`)
- Test fixtures are in `testdata/fixtures/`
- Test utilities are in `testdata/analysis/`

## Writing Tests
- Use table-driven tests where appropriate
- Mock external dependencies
- Test both success and error cases
- Aim for >80% coverage 