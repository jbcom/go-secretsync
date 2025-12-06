# Go Development Guidelines

## Core Philosophy

Write clean, tested, production-ready Go code. No shortcuts, no placeholders.

## Development Flow

1. **Read the requirements** from specs or issues
2. **Write tests first** (TDD approach)
3. **Implement the feature** completely
4. **Run tests**: `go test ./...`
5. **Run linting**: `golangci-lint run`
6. **Commit** with conventional commits

## Testing Commands

```bash
# Run all tests
go test ./...

# Run with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run specific package tests
go test ./internal/sync/...

# Linting
golangci-lint run

# Build
go build ./cmd/vss
```

## Docker Commands

```bash
# Build Docker image
docker build -t vault-secret-sync .

# Run locally
docker run -v $(pwd)/config.yaml:/config.yaml vault-secret-sync
```

## Commit Messages

Use conventional commits:
- `feat(scope): description` → minor bump
- `fix(scope): description` → patch bump
- `feat!: breaking change` → major bump

## Quality Standards

- ✅ All tests passing
- ✅ No linter errors
- ✅ Proper error handling
- ✅ Clear documentation
- ❌ No TODOs or placeholders
- ❌ No shortcuts
