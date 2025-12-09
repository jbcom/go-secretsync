# vault-secret-sync Makefile

.PHONY: all build test test-unit test-integration lint clean help

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOMOD=$(GOCMD) mod
GOLINT=golangci-lint

# Build info
BINARY_NAME=vault-secret-sync
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

all: lint test build

## Build targets
build:
	$(GOBUILD) -ldflags "-X main.Version=$(VERSION)" -o $(BINARY_NAME) ./...

## Test targets
test: test-unit

test-unit:
	$(GOTEST) -v -race ./...

# Integration tests require LocalStack + Vault
# Either run manually with docker-compose or let CI handle it
test-integration:
	@echo "Running integration tests..."
	@if [ -z "$$VAULT_ADDR" ] || [ -z "$$AWS_ENDPOINT_URL" ]; then \
		echo "Starting test environment with docker-compose..."; \
		docker-compose -f docker-compose.test.yml up --abort-on-container-exit --exit-code-from test-runner; \
	else \
		echo "Using existing environment (VAULT_ADDR=$$VAULT_ADDR, AWS_ENDPOINT_URL=$$AWS_ENDPOINT_URL)"; \
		$(GOTEST) -v -tags=integration ./tests/integration/...; \
	fi

# Run integration tests with docker-compose (always starts fresh)
test-integration-docker:
	docker-compose -f docker-compose.test.yml down -v 2>/dev/null || true
	docker-compose -f docker-compose.test.yml up --build --abort-on-container-exit --exit-code-from test-runner
	docker-compose -f docker-compose.test.yml down -v

# Start the test environment (for manual testing)
test-env-up:
	docker-compose -f docker-compose.test.yml up -d localstack vault
	@echo "Waiting for services to be healthy..."
	@sleep 5
	@echo ""
	@echo "Test environment ready. Export these variables:"
	@echo "  export VAULT_ADDR=http://localhost:8200"
	@echo "  export VAULT_TOKEN=test-root-token"
	@echo "  export AWS_ENDPOINT_URL=http://localhost:4566"
	@echo "  export AWS_ACCESS_KEY_ID=test"
	@echo "  export AWS_SECRET_ACCESS_KEY=test"
	@echo "  export AWS_REGION=us-east-1"
	@echo ""
	@echo "Then run: make test-integration"

test-env-down:
	docker-compose -f docker-compose.test.yml down -v

## Lint targets
lint:
	$(GOLINT) run

lint-fix:
	$(GOLINT) run --fix

## Dependency management
deps:
	$(GOMOD) download
	$(GOMOD) tidy

## Clean targets
clean:
	rm -f $(BINARY_NAME)
	docker-compose -f docker-compose.test.yml down -v 2>/dev/null || true

## Help
help:
	@echo "Available targets:"
	@echo "  build                 - Build the binary"
	@echo "  test                  - Run unit tests"
	@echo "  test-unit             - Run unit tests"
	@echo "  test-integration      - Run integration tests (auto-detects environment)"
	@echo "  test-integration-docker - Run integration tests via docker-compose"
	@echo "  test-env-up           - Start LocalStack + Vault for local testing"
	@echo "  test-env-down         - Stop test environment"
	@echo "  lint                  - Run linters"
	@echo "  lint-fix              - Run linters with auto-fix"
	@echo "  deps                  - Download and tidy dependencies"
	@echo "  clean                 - Clean build artifacts and test containers"
