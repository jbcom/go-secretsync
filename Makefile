# SecretSync Makefile
# See 'make help' for available targets

.PHONY: all build test test-unit test-integration test-coverage lint lint-fix clean help
.PHONY: deps deps-update fmt vet pre-commit pre-commit-install pre-commit-update
.PHONY: test-env-up test-env-down test-integration-docker install tools

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOMOD=$(GOCMD) mod
GOFMT=$(GOCMD) fmt
GOVET=$(GOCMD) vet
GOLINT=golangci-lint

# Build info
BINARY_NAME=secretsync
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT?=$(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE?=$(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
LDFLAGS=-s -w \
	-X github.com/jbcom/secretsync/cmd/secretsync/cmd.Version=$(VERSION) \
	-X github.com/jbcom/secretsync/cmd/secretsync/cmd.Commit=$(COMMIT) \
	-X github.com/jbcom/secretsync/cmd/secretsync/cmd.Date=$(DATE)

# Coverage
COVERAGE_DIR=coverage
COVERAGE_FILE=$(COVERAGE_DIR)/coverage.out
COVERAGE_HTML=$(COVERAGE_DIR)/coverage.html

# Default target
all: lint test build

##@ Development

.PHONY: fmt
fmt: ## Format Go source code
	$(GOFMT) ./...

.PHONY: vet
vet: ## Run go vet
	$(GOVET) ./...

.PHONY: lint
lint: ## Run golangci-lint
	$(GOLINT) run

.PHONY: lint-fix
lint-fix: ## Run golangci-lint with auto-fix
	$(GOLINT) run --fix

##@ Build

.PHONY: build
build: ## Build the binary
	$(GOBUILD) -ldflags "$(LDFLAGS)" -o $(BINARY_NAME) ./cmd/secretsync

.PHONY: install
install: ## Install the binary to GOPATH/bin
	$(GOCMD) install -ldflags "$(LDFLAGS)" ./cmd/secretsync

##@ Testing

.PHONY: test
test: test-unit ## Run all tests (alias for test-unit)

.PHONY: test-unit
test-unit: ## Run unit tests with race detector
	$(GOTEST) -v -race ./...

.PHONY: test-coverage
test-coverage: ## Run tests with coverage report
	@mkdir -p $(COVERAGE_DIR)
	$(GOTEST) -v -race -coverprofile=$(COVERAGE_FILE) -covermode=atomic ./...
	$(GOCMD) tool cover -html=$(COVERAGE_FILE) -o $(COVERAGE_HTML)
	@echo "Coverage report: $(COVERAGE_HTML)"

.PHONY: test-integration
test-integration: ## Run integration tests (auto-detects environment)
	@echo "Running integration tests..."
	@if [ -z "$$VAULT_ADDR" ] || [ -z "$$AWS_ENDPOINT_URL" ]; then \
		echo "Starting test environment with docker-compose..."; \
		docker-compose -f docker-compose.test.yml up --abort-on-container-exit --exit-code-from test-runner; \
	else \
		echo "Using existing environment (VAULT_ADDR=$$VAULT_ADDR, AWS_ENDPOINT_URL=$$AWS_ENDPOINT_URL)"; \
		$(GOTEST) -v -tags=integration ./tests/integration/...; \
	fi

.PHONY: test-integration-docker
test-integration-docker: ## Run integration tests via docker-compose (starts fresh)
	docker-compose -f docker-compose.test.yml down -v 2>/dev/null || true
	docker-compose -f docker-compose.test.yml up --build --abort-on-container-exit --exit-code-from test-runner
	docker-compose -f docker-compose.test.yml down -v

.PHONY: test-env-up
test-env-up: ## Start LocalStack + Vault for local testing
	docker-compose -f docker-compose.test.yml up -d localstack vault
	@echo "Waiting for services to be healthy..."
	@for i in 1 2 3 4 5 6 7 8 9 10 11 12; do \
		if docker-compose -f docker-compose.test.yml ps | grep -q "(healthy)" 2>/dev/null; then \
			echo "Services are healthy!"; \
			break; \
		fi; \
		if [ $$i -eq 12 ]; then \
			echo "Warning: Services may not be fully healthy, proceeding anyway"; \
		else \
			echo "Waiting... ($$i/12)"; \
			sleep 5; \
		fi; \
	done
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

.PHONY: test-env-down
test-env-down: ## Stop test environment
	docker-compose -f docker-compose.test.yml down -v

##@ Dependencies

.PHONY: deps
deps: ## Download and tidy dependencies
	$(GOMOD) download
	$(GOMOD) tidy

.PHONY: deps-update
deps-update: ## Update all dependencies to latest versions
	$(GOCMD) get -u ./...
	$(GOMOD) tidy

##@ Pre-commit

.PHONY: pre-commit
pre-commit: ## Run pre-commit hooks on all files
	pre-commit run --all-files

.PHONY: pre-commit-install
pre-commit-install: ## Install pre-commit hooks
	@if command -v pre-commit >/dev/null 2>&1; then \
		pre-commit install; \
		echo "Pre-commit hooks installed successfully"; \
	else \
		echo "pre-commit not found. Install with: pip install pre-commit"; \
		exit 1; \
	fi

.PHONY: pre-commit-update
pre-commit-update: ## Update pre-commit hooks to latest versions
	pre-commit autoupdate

##@ Tools

.PHONY: tools
tools: ## Install development tools
	@echo "Installing development tools..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo ""
	@echo "Optional: Install pre-commit with: pip install pre-commit"
	@echo "Then run: make pre-commit-install"

##@ Clean

.PHONY: clean
clean: ## Clean build artifacts and test containers
	rm -f $(BINARY_NAME)
	rm -rf $(COVERAGE_DIR)
	docker-compose -f docker-compose.test.yml down -v 2>/dev/null || true

##@ Help

.PHONY: help
help: ## Display this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-25s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)
