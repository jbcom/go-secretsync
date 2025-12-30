# SecretSync v1.1.0 Requirements

## Overview

Version 1.1.0 focuses on operational excellence: observability, reliability, and security hardening. This release makes SecretSync production-ready for critical infrastructure use.

## Epic #60: v1.1.0 Release

**Goal:** Ship production-ready improvements for CI/CD, security, and observability

**Target Date:** Q1 2025

## Requirements

### Requirement 1: Prometheus Metrics Endpoint (#46, #69)

**User Story:** As an operator, I want to monitor SecretSync performance and health through Prometheus metrics.

**Acceptance Criteria:**

1. WHEN SecretSync starts with `--metrics-port` flag THEN it SHALL expose Prometheus metrics on `/metrics` endpoint
2. WHEN Vault API calls are made THEN metrics SHALL track:
   - Request duration histogram (`secretsync_vault_request_duration_seconds`)
   - Request count by operation (`secretsync_vault_requests_total`)
   - Error count by type (`secretsync_vault_errors_total`)
   - Secrets processed counter (`secretsync_vault_secrets_total`)
3. WHEN AWS API calls are made THEN metrics SHALL track:
   - Request duration histogram (`secretsync_aws_request_duration_seconds`)
   - Request count by service and operation (`secretsync_aws_requests_total`)
   - Error count by service (`secretsync_aws_errors_total`)
   - Pagination count (`secretsync_aws_pagination_calls_total`)
4. WHEN pipeline executes THEN metrics SHALL track:
   - Pipeline execution duration (`secretsync_pipeline_duration_seconds`)
   - Merge phase duration (`secretsync_merge_duration_seconds`)
   - Sync phase duration (`secretsync_sync_duration_seconds`)
   - Secrets synced count (`secretsync_secrets_synced_total`)
5. WHEN metrics are exposed THEN they SHALL include standard Go runtime metrics

**Implementation Notes:**
- Use `github.com/prometheus/client_golang/prometheus`
- Package: `pkg/observability/metrics`
- CLI flag: `--metrics-port` (default: disabled)
- Environment variable: `METRICS_PORT`

---

### Requirement 2: Circuit Breaker Pattern (#47, #70)

**User Story:** As an operator, I want SecretSync to fail fast and recover gracefully when external services are degraded.

**Acceptance Criteria:**

1. WHEN Vault API fails 5 times in 10 seconds THEN circuit SHALL open and reject requests for 30 seconds
2. WHEN circuit is open THEN requests SHALL fail immediately with clear error message
3. WHEN circuit is half-open THEN one request SHALL be allowed through to test recovery
4. WHEN test request succeeds THEN circuit SHALL close and resume normal operation
5. WHEN AWS API fails 5 times in 10 seconds THEN circuit SHALL open independently of Vault circuit
6. WHEN circuit state changes THEN event SHALL be logged with timestamp and reason
7. WHEN metrics are enabled THEN circuit state SHALL be exposed as metric

**Configuration:**
```yaml
circuit_breaker:
  failure_threshold: 5
  timeout: 30s
  max_requests: 1  # half-open state
```

**Implementation Notes:**
- Use `github.com/sony/gobreaker` library
- Package: `pkg/resilience/breaker`
- Wrap all external API clients
- Default: enabled with conservative settings

---

### Requirement 3: Enhanced Error Messages (#48, #71)

**User Story:** As a developer debugging issues, I want detailed error context including request IDs and timing information.

**Acceptance Criteria:**

1. WHEN any API request is made THEN a unique request ID SHALL be generated
2. WHEN errors occur THEN error messages SHALL include:
   - Request ID
   - Operation name (e.g., "vault.list", "aws.create_secret")
   - Resource path or secret name
   - Operation duration in milliseconds
   - Retry count (if retried)
3. WHEN operations start THEN request ID SHALL be logged at INFO level
4. WHEN operations fail THEN error SHALL wrap with structured context
5. WHEN structured logging is enabled THEN errors SHALL include fields: `request_id`, `operation`, `path`, `duration_ms`, `retries`

**Example Error:**
```
[req=abc123] failed to list secrets at path "secret/data/app" after 1250ms (retries: 2): permission denied
```

**Implementation Notes:**
- Package: `pkg/context`
- Use context.Context to propagate request IDs
- Use `github.com/google/uuid` for request ID generation
- Error wrapping with `fmt.Errorf(..., %w, err)`

---

### Requirement 4: Docker Image Version Pinning (#40, #64)

**User Story:** As a security-conscious operator, I want reproducible builds with pinned dependency versions.

**Acceptance Criteria:**

1. WHEN `docker-compose.test.yml` is used THEN all images SHALL use specific version tags:
   - `localstack/localstack:3.8.1`
   - `hashicorp/vault:1.17.6`
   - `amazon/aws-cli:2.22.17`
   - `golang:1.25-trixie`
2. WHEN `Dockerfile` builds THEN base images SHALL use specific versions:
   - `golang:1.25-trixie` (builder)
   - `debian:trixie-slim` (runtime)
3. WHEN `action.yml` references Docker image THEN it SHALL use digest pinning:
   - `extended-data-library/secretssync:v1@sha256:<digest>`
4. WHEN images are updated THEN `CHANGELOG.md` SHALL document version changes

**Files to Update:**
- `docker-compose.test.yml`
- `Dockerfile`
- `action.yml`

---

### Requirement 5: Configurable Queue Compaction (#43, #67)

**User Story:** As an operator with varying secret volumes, I want configurable queue compaction thresholds.

**Acceptance Criteria:**

1. WHEN Vault client is configured THEN queue compaction threshold SHALL be configurable
2. WHEN threshold is not set THEN default SHALL be `min(1000, maxSecretsPerMount/100)`
3. WHEN queue index exceeds threshold AND exceeds half queue length THEN queue SHALL compact
4. WHEN compaction occurs THEN event SHALL be logged with old/new queue sizes
5. WHEN configuration is loaded THEN invalid thresholds SHALL be rejected with clear error

**Configuration:**
```yaml
vault_sources:
  - mount: secret/
    queue_compaction_threshold: 500
```

**Implementation Notes:**
- Add field to `VaultSource` config struct
- Update `pkg/client/vault/vault.go`
- Validate: threshold > 0

---

### Requirement 6: Race Condition Prevention (#44, #68)

**User Story:** As a developer, I want confidence that concurrent operations are thread-safe.

**Acceptance Criteria:**

1. WHEN `accountSecretArns` map is accessed THEN it SHALL be protected by `arnMu sync.RWMutex`
2. WHEN tests run with `-race` flag THEN no race conditions SHALL be detected
3. WHEN concurrent reads occur THEN `RLock()` SHALL be used
4. WHEN concurrent writes occur THEN `Lock()` SHALL be used
5. WHEN tests run THEN concurrent access test SHALL validate safety under high load

**Test Coverage:**
- Concurrent map reads (20 goroutines, 100 iterations each)
- Concurrent map writes (10 goroutines, 100 iterations each)
- Mixed read/write operations
- DeepCopy during concurrent modifications

**Implementation Notes:**
- File: `pkg/client/aws/aws.go`
- Tests: `pkg/client/aws/aws_test.go`
- Already implemented - tests verify

---

### Requirement 7: Documentation Workflow Fix (#50)

**User Story:** As a contributor, I want documentation builds to work correctly.

**Acceptance Criteria:**

1. WHEN docs workflow runs THEN it SHALL NOT attempt to install Python dependencies for a Go project
2. WHEN documentation is built THEN Sphinx SHALL use only RST files in `docs/`
3. WHEN PRs are created THEN documentation SHALL build successfully
4. WHEN documentation changes THEN preview link SHALL be available

**Implementation:**
- Remove `pyproject.toml` reference (this is a Go project)
- Update `.github/workflows/docs.yml`
- Use `sphinx-build` directly on `docs/` directory
- Generate API docs from Go code if needed

---

### Requirement 8: CI Workflow Modernization (#51)

**User Story:** As a maintainer, I want modern CI workflows with semantic versioning.

**Acceptance Criteria:**

1. WHEN CI workflow references actions THEN it SHALL use semantic versions (e.g., `v4`)
2. WHEN CI workflow uses SHA pins THEN it SHALL replace with semantic versions
3. WHEN dependencies update THEN Dependabot SHALL group minor/patch updates
4. WHEN actions are updated THEN `CHANGELOG.md` SHALL document changes
5. WHEN workflow runs THEN it SHALL complete in < 10 minutes

**Files to Update:**
- `.github/workflows/*.yml`
- Replace SHA1 hashes with `@v4` style tags
- Ensure actions are from verified publishers

---

### Requirement 9: Consolidated Documentation CI (#52)

**User Story:** As a maintainer, I want a single CI workflow for all documentation.

**Acceptance Criteria:**

1. WHEN documentation changes THEN a single workflow SHALL build and validate
2. WHEN PRs are opened THEN documentation SHALL be built and checked
3. WHEN main branch is updated THEN documentation SHALL be published
4. WHEN documentation build fails THEN PR checks SHALL fail with clear error

**Implementation:**
- Merge `docs.yml` and other doc workflows into single workflow
- Add documentation build step to main CI workflow
- Cache Sphinx dependencies
- Deploy to GitHub Pages on main branch

---

### Requirement 10: Command Injection Prevention (#41)

**User Story:** As a security engineer, I want protection against command injection in external script execution.

**Acceptance Criteria:**

1. WHEN external commands are executed THEN they SHALL use `exec.CommandContext` with explicit arguments
2. WHEN user input is used in commands THEN it SHALL be validated against allowlist
3. WHEN shell execution is required THEN it SHALL be avoided or heavily restricted
4. WHEN paths are used THEN they SHALL be validated and sanitized
5. WHEN security scan runs THEN no command injection vulnerabilities SHALL be found

**Implementation:**
- Audit all uses of `os/exec`
- Replace shell commands with direct binary execution
- Validate inputs against strict patterns
- Document in security policy

---

## Non-Functional Requirements

### Performance
- Pipeline execution: < 5 minutes for 1000 secrets
- API response time: p95 < 500ms
- Memory usage: < 500MB for typical workloads

### Security
- No credentials in logs
- All external connections use TLS
- Secrets never stored on disk unencrypted
- Follow OWASP secure coding practices

### Reliability
- Graceful handling of API rate limits
- Automatic retry with exponential backoff
- Clear error messages for operational issues
- Circuit breakers prevent cascade failures

### Maintainability
- 80%+ test coverage
- All public APIs documented
- Architecture decision records for major changes
- Clean Git history with conventional commits

## Release Checklist

- [ ] All 10 requirements implemented
- [ ] All PRs (#64, #67-71) merged
- [ ] Test coverage ≥ 80%
- [ ] Integration tests passing
- [ ] Documentation updated
- [ ] CHANGELOG.md updated
- [ ] Security scan passed
- [ ] Docker image built and tested
- [ ] Helm chart tested
- [ ] Git tag `v1.1.0` created
- [ ] GitHub release published
- [ ] Announcement drafted

## Success Metrics

- Zero production incidents related to observability gaps
- Circuit breakers prevent >= 90% of cascade failures
- Mean time to debug reduced by 50% (via enhanced errors)
- All security scans pass with zero high/critical findings
- Docker builds are reproducible across environments

