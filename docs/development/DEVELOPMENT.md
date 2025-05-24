# Development Guide

## Prerequisites
- Go 1.21 or higher
- Docker and Docker Compose (optional)
- golangci-lint (for linting)

## Quick Start
1. Clone the repository
2. Run setup: `make setup`
3. Configure API keys in `configs/credentials.json`
4. Build: `make build`
5. Run: `make run`

## Development Workflow
- Use `make run-dev` for development (auto-recompiles)
- Run tests frequently: `make test`
- Format code: `make format`
- Lint before committing: `make lint`

See [CONTRIBUTING.md](./CONTRIBUTING.md) for contribution guidelines. 