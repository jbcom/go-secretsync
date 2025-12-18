# SecretSync - Complete System Requirements (v1.0 → v1.2.0)

## Introduction

SecretSync is a production-ready Go application for synchronizing secrets from HashiCorp Vault to AWS Secrets Manager and other external secret stores. This document consolidates ALL requirements from v1.0, v1.1.0, and v1.2.0 into a single source of truth.

**Target Users:**
- DevOps Engineers managing multi-account AWS environments
- Platform Engineers building secret management infrastructure
- Security Teams enforcing secret rotation policies
- Organizations migrating from Vault to AWS Secrets Manager

**Version Status:**
- ✅ v1.0: COMPLETE (113+ tests passing)
- ⚠️  v1.1.0: PARTIAL (has 2 lint errors, most features done)
- ⏳ v1.2.0: PLANNED (some features complete, others pending)

## Glossary

- **SecretSync**: The system being specified - a Go application for secret synchronization
- **Vault**: HashiCorp Vault - source secret management system using KV2 engine
- **AWS Secrets Manager**: Amazon Web Services secret storage service - target for synchronization
- **Pipeline**: Two-phase process consisting of merge phase and sync phase
- **Merge Phase**: First phase where secrets from multiple Vault sources are combined
- **Sync Phase**: Second phase where merged secrets are synchronized to target stores
- **Merge Store**: S3 bucket used to store merged secret configurations for inheritance
- **Target**: External secret store destination (e.g., AWS Secrets Manager instance)
- **Source**: Vault mount path from which secrets are read
- **Inheritance**: Mechanism allowing targets to import configuration from other targets
- **Discovery**: Automatic detection of AWS accounts and resources from AWS Organizations
- **BFS**: Breadth-First Search - traversal algorithm used for recursive secret listing
- **IRSA**: IAM Roles for Service Accounts - Kubernetes authentication method for AWS
- **OIDC**: OpenID Connect - authentication protocol used by GitHub Actions
- **Circuit Breaker**: Reliability pattern that prevents cascade failures by failing fast
- **Deep Merge**: Recursive merging strategy for combining complex data structures
- **Dry-Run Mode**: Execution mode that computes changes without applying them

---

## CORE FEATURES (v1.0) - ✅ COMPLETE

### Requirement 1: Vault Authentication
**Status:** ✅ COMPLETE  
**User Story:** As a DevOps engineer, I want SecretSync to authenticate with HashiCorp Vault, so that I can securely read secrets from Vault mounts.

#### Acceptance Criteria
1. WHEN `VAULT_ROLE_ID` and `VAULT_SECRET_ID` environment variables are set, THE SecretSync SHALL authenticate successfully to Vault
2. WHEN Vault address is configured via `VAULT_ADDR` environment variable, THE SecretSync SHALL connect to that address
3. IF authentication fails, THEN THE SecretSync SHALL provide a clear error message explaining the cause
4. WHEN token expires, THE SecretSync SHALL automatically renew the token
5. IF token renewal fails, THEN THE SecretSync SHALL re-authenticate using AppRole credentials

**Implementation:** `pkg/client/vault/vault.go`

---

### Requirement 2: Vault Recursive Secret Listing
**Status:** ✅ COMPLETE (PR #29)  
**User Story:** As a DevOps engineer, I want SecretSync to recursively discover all secrets in Vault mount paths, so that I can synchronize entire secret hierarchies without manual enumeration.

#### Acceptance Criteria
1. WHEN listing a Vault path, THE SecretSync SHALL discover all nested secrets using BFS traversal
2. WHEN a directory is encountered (path ends with `/`), THE SecretSync SHALL traverse into that directory
3. WHEN a secret is found, THE SecretSync SHALL return its full path without leading slash
4. WHEN cycles are detected during traversal, THE SecretSync SHALL prevent infinite loops
5. WHEN the `max_secrets` limit is reached, THE SecretSync SHALL stop traversal
6. IF path is invalid, THEN THE SecretSync SHALL provide an error explaining the validation failure
7. IF permissions are insufficient, THEN THE SecretSync SHALL provide an error indicating the permission issue

**Implementation:** `pkg/client/vault/vault.go` lines 479-545  
**Tests:** 6 test scenarios in `vault_test.go`

---

### Requirement 3: Vault Secret Reading
**Status:** ✅ COMPLETE  
**User Story:** As a DevOps engineer, I want SecretSync to read secret values from Vault, so that I can synchronize secret data to target stores.

#### Acceptance Criteria
1. WHEN reading a secret, THE SecretSync SHALL retrieve both metadata and data
2. IF secret does not exist, THEN THE SecretSync SHALL return a clear error
3. IF secret is deleted, THEN THE SecretSync SHALL return an appropriate error
4. WHEN secret has multiple versions, THE SecretSync SHALL use the latest version
5. IF reading fails due to network error, THEN THE SecretSync SHALL attempt retry with backoff

**Implementation:** `pkg/client/vault/vault.go`

---

### Requirement 4: Path Security
**Status:** ✅ COMPLETE (Enhanced in PR #29)  
**User Story:** As a security engineer, I want SecretSync to validate all Vault paths, so that path traversal attacks are prevented.

#### Acceptance Criteria
1. IF path contains `..`, THEN THE SecretSync SHALL reject the path
2. IF path contains null bytes (`\x00`), THEN THE SecretSync SHALL reject the path
3. WHEN path contains `//`, THE SecretSync SHALL normalize it to single `/`
4. WHEN path is absolute (starts with `/`), THE SecretSync SHALL handle it correctly
5. WHEN path is relative, THE SecretSync SHALL resolve it against the mount path

**Implementation:** Path validation in `pkg/client/vault/vault.go`

---

### Requirement 5: AWS Authentication
**Status:** ✅ COMPLETE  
**User Story:** As a platform engineer, I want SecretSync to authenticate with AWS using multiple methods, so that I can deploy it in different environments.

#### Acceptance Criteria
1. WHILE running in Kubernetes, THE SecretSync SHALL use IRSA for authentication
2. WHILE running in GitHub Actions, THE SecretSync SHALL use OIDC for authentication
3. WHERE `AWS_ROLE_ARN` is configured, THE SecretSync SHALL perform role assumption
4. WHILE running locally, THE SecretSync SHALL use AWS credentials from environment
5. IF authentication fails, THEN THE SecretSync SHALL provide an error explaining which method was attempted

**Implementation:** `pkg/client/aws/aws.go`

---

### Requirement 6: AWS Secrets Manager Operations
**Status:** ✅ COMPLETE  
**User Story:** As a DevOps engineer, I want SecretSync to perform CRUD operations on AWS Secrets Manager, so that I can synchronize secrets to AWS.

#### Acceptance Criteria
1. WHEN listing secrets, THE SecretSync SHALL handle pagination for more than 100 secrets
2. WHEN creating a secret, THE SecretSync SHALL create it with appropriate metadata
3. WHEN updating a secret, THE SecretSync SHALL update only changed values
4. WHEN deleting a secret, THE SecretSync SHALL confirm deletion
5. IF secret already exists, THEN THE SecretSync SHALL perform update instead of create
6. WHERE `NoEmptySecrets` configuration is true, THE SecretSync SHALL skip empty secrets

**Implementation:** `pkg/client/aws/aws.go`

---

### Requirement 7: Cross-Account Access
**Status:** ✅ COMPLETE  
**User Story:** As a platform engineer, I want SecretSync to sync secrets to multiple AWS accounts, so that I can manage secrets across my organization.

#### Acceptance Criteria
1. WHERE `role_arn` is configured for a Target, THE SecretSync SHALL assume that role
2. IF role assumption fails, THEN THE SecretSync SHALL provide an error indicating the role ARN and reason
3. WHERE external ID is required, THE SecretSync SHALL support configurable external ID
4. WHEN role session expires, THE SecretSync SHALL create a new session automatically
5. WHEN assuming roles in multiple accounts, THE SecretSync SHALL manage sessions independently

**Implementation:** `pkg/client/aws/aws.go`

---

### Requirement 8: S3 Merge Store
**Status:** ✅ COMPLETE (PR #29)  
**User Story:** As a platform engineer, I want SecretSync to store merged configurations in S3, so that targets can inherit from each other.

#### Acceptance Criteria
1. WHEN Merge Phase completes, THE SecretSync SHALL write merged secrets to S3
2. WHEN Sync Phase starts, THE SecretSync SHALL read secrets from S3
3. WHERE S3 bucket is in different account, THE SecretSync SHALL perform role assumption
4. WHEN listing S3 objects, THE SecretSync SHALL handle pagination for more than 1000 objects
5. IF S3 object does not exist, THEN THE SecretSync SHALL return a clear error
6. IF S3 access is denied, THEN THE SecretSync SHALL provide an error including bucket and prefix

**Implementation:** `pkg/pipeline/s3_store.go` lines 107-180

---

### Requirement 9: Merge Phase
**Status:** ✅ COMPLETE (PR #29)  
**User Story:** As a platform engineer, I want SecretSync to merge secrets from multiple Vault sources, so that I can combine base configurations with environment-specific overrides.

#### Acceptance Criteria
1. WHEN multiple Sources provide the same secret path, THE SecretSync SHALL deep merge the values
2. WHEN merging lists, THE SecretSync SHALL append items (not replace)
3. WHEN merging maps, THE SecretSync SHALL recursively merge keys
4. WHEN merging scalars, THE SecretSync SHALL override with later Source value
5. IF type conflict occurs (list vs map), THEN THE SecretSync SHALL use the later Source value
6. WHEN Merge Phase completes, THE SecretSync SHALL make the result available for Sync Phase

**Implementation:** `pkg/utils/deepmerge.go` + `pkg/pipeline/merge.go`  
**Tests:** 13 test functions covering all merge strategies

---

### Requirement 10: Sync Phase
**Status:** ✅ COMPLETE  
**User Story:** As a DevOps engineer, I want SecretSync to sync merged secrets to configured targets, so that secrets are propagated to all destination stores.

#### Acceptance Criteria
1. WHERE Target has no dependencies, THE SecretSync SHALL sync it immediately
2. WHERE Target has dependencies, THE SecretSync SHALL sync dependencies first
3. IF circular dependency is detected, THEN THE SecretSync SHALL raise a clear error
4. WHERE Target imports from another Target, THE SecretSync SHALL resolve the import from S3
5. IF sync to one Target fails, THEN THE SecretSync SHALL still attempt other Targets
6. WHERE `--dry-run` flag is specified, THE SecretSync SHALL not make actual changes

**Implementation:** `pkg/pipeline/sync.go`

---

### Requirement 11: Target Inheritance
**Status:** ✅ COMPLETE (PR #29)  
**User Story:** As a platform engineer, I want targets to inherit from other targets, so that I can reuse common configurations across environments.

#### Acceptance Criteria
1. WHERE Target imports from another Target, THE SecretSync SHALL read the merged output from S3
2. WHEN resolving imports, THE SecretSync SHALL use topological sort to determine order
3. WHERE multi-level inheritance exists (A→B→C), THE SecretSync SHALL resolve all levels correctly
4. IF imported Target does not exist in S3, THEN THE SecretSync SHALL provide an error indicating the Target name
5. WHERE Target overrides imported values, THE SecretSync SHALL apply overrides with precedence

**Implementation:** `pkg/pipeline/inheritance.go` + `pkg/pipeline/resolver.go`

---

### Requirement 12: Configuration Management
**Status:** ✅ COMPLETE  
**User Story:** As a user, I want to configure SecretSync via YAML files, so that I can version control my configuration.

#### Acceptance Criteria
1. WHEN `--config` flag is provided, THE SecretSync SHALL load that file
2. IF YAML syntax is invalid, THEN THE SecretSync SHALL show a clear parse error
3. IF required fields are missing, THEN THE SecretSync SHALL fail validation with specific field names
4. WHEN unknown fields are present, THE SecretSync SHALL log a warning
5. IF file does not exist, THEN THE SecretSync SHALL indicate the path in error

**Implementation:** `pkg/pipeline/config.go`

---

### Requirement 13: Diff Computation
**Status:** ✅ COMPLETE  
**User Story:** As a user, I want to see what changes will be made before applying them, so that I can review and approve changes.

#### Acceptance Criteria
1. WHEN `--diff` flag is provided, THE SecretSync SHALL compute differences
2. WHEN secret is new, THE SecretSync SHALL mark it as "added"
3. WHEN secret value changes, THE SecretSync SHALL mark it as "modified"
4. WHEN secret is removed, THE SecretSync SHALL mark it as "deleted"
5. WHEN secret metadata changes, THE SecretSync SHALL mark it as "modified"
6. WHEN no changes exist, THE SecretSync SHALL show "no differences" message

**Implementation:** `pkg/diff/diff.go`

---

### Requirement 14: Dry-Run Mode
**Status:** ✅ COMPLETE  
**User Story:** As a user, I want to validate configuration without making changes, so that I can test safely.

#### Acceptance Criteria
1. WHERE `--dry-run` is specified, THE SecretSync SHALL not make actual changes
2. WHILE in dry-run mode, THE SecretSync SHALL still compute diff
3. WHILE in dry-run mode, THE SecretSync SHALL still perform all validation
4. WHILE in dry-run mode, THE SecretSync SHALL clearly indicate "DRY RUN" mode in output
5. IF errors occur in dry-run, THEN THE SecretSync SHALL still report them

**Implementation:** CLI flag handling in `cmd/secretsync/cmd/pipeline.go`

---

## OBSERVABILITY & RELIABILITY (v1.1.0) - ✅ COMPLETE

### Requirement 15: Prometheus Metrics
**Status:** ✅ COMPLETE (PR #69, verified)  
**User Story:** As an operator, I want to monitor SecretSync performance and health through Prometheus metrics.

#### Acceptance Criteria
1. WHERE `--metrics-port` flag is specified, THE SecretSync SHALL expose Prometheus metrics on `/metrics` endpoint
2. WHEN Vault API is called, THE SecretSync SHALL record request duration
3. WHEN AWS API is called, THE SecretSync SHALL record request duration
4. WHEN Pipeline executes, THE SecretSync SHALL record execution duration
5. WHEN errors occur, THE SecretSync SHALL increment error counters
6. WHEN metrics are scraped, THE SecretSync SHALL include standard Go runtime metrics

**Implementation:** `pkg/observability/metrics.go` + `cmd/secretsync/cmd/root.go`  
**Status:** ✅ Fully integrated with CLI flags

---

### Requirement 16: Circuit Breaker Pattern
**Status:** ✅ COMPLETE (PR #70, lint error fixed)  
**User Story:** As an operator, I want SecretSync to fail fast and recover gracefully when external services are degraded.

#### Acceptance Criteria
1. WHEN Vault API fails 5 times in 10 seconds, THE SecretSync SHALL open circuit and reject requests for 30 seconds
2. WHILE circuit is open, THE SecretSync SHALL fail requests immediately with clear error message
3. WHILE circuit is half-open, THE SecretSync SHALL allow one request through to test recovery
4. IF test request succeeds, THEN THE SecretSync SHALL close circuit and resume normal operation
5. WHEN AWS API fails 5 times in 10 seconds, THE SecretSync SHALL open circuit independently of Vault circuit
6. WHEN circuit state changes, THE SecretSync SHALL log event with timestamp and reason
7. WHERE metrics are enabled, THE SecretSync SHALL expose circuit state as metric

**Implementation:** `pkg/circuitbreaker/circuitbreaker.go`  
**Issue:** Has staticcheck QF1003 lint error on line 142  
**Action Required:** Fix switch statement pattern

---

### Requirement 17: Enhanced Error Messages
**Status:** ✅ COMPLETE (PR #71, verified)  
**User Story:** As a developer debugging issues, I want detailed error context including request IDs and timing information.

#### Acceptance Criteria
1. WHEN any API request is made, THE SecretSync SHALL generate a unique request ID
2. WHEN errors occur, THE SecretSync SHALL include request ID, operation name, resource path, duration, and retry count
3. WHEN operations start, THE SecretSync SHALL log request ID at INFO level
4. WHEN operations fail, THE SecretSync SHALL wrap error with structured context
5. WHERE structured logging is enabled, THE SecretSync SHALL include fields: `request_id`, `operation`, `path`, `duration_ms`, `retries`

**Implementation:** `pkg/context/error_context.go` + `pkg/context/request_context.go`  
**Action Required:** Verify adoption throughout codebase

---

### Requirement 18: Docker Image Version Pinning
**Status:** ✅ COMPLETE (PR #64)  
**User Story:** As a security-conscious operator, I want reproducible builds with pinned dependency versions.

#### Acceptance Criteria
1. WHEN `docker-compose.test.yml` is used, THE SecretSync SHALL use specific version tags for all images
2. WHEN `Dockerfile` builds, THE SecretSync SHALL use specific versions for base images
3. WHEN `action.yml` references Docker image, THE SecretSync SHALL use digest pinning
4. WHEN images are updated, THE SecretSync SHALL document version changes in CHANGELOG.md

**Implementation:** Verified in `docker-compose.test.yml`, `Dockerfile`, `action.yml`

---

### Requirement 19: Configurable Queue Compaction
**Status:** ✅ COMPLETE (PR #67, verified)  
**User Story:** As an operator with varying secret volumes, I want configurable queue compaction thresholds.

#### Acceptance Criteria
1. WHERE Vault client is configured, THE SecretSync SHALL support configurable queue compaction threshold
2. WHERE threshold is not set, THE SecretSync SHALL use default of `min(1000, maxSecretsPerMount/100)`
3. WHEN queue index exceeds threshold AND exceeds half queue length, THE SecretSync SHALL compact queue
4. WHEN compaction occurs, THE SecretSync SHALL log event with old/new queue sizes
5. WHERE configuration is loaded, THE SecretSync SHALL reject invalid thresholds with clear error

**Action Required:** Verify config field exists and is used

---

### Requirement 20: Race Condition Prevention
**Status:** ✅ COMPLETE (PR #68)  
**User Story:** As a developer, I want confidence that concurrent operations are thread-safe.

#### Acceptance Criteria
1. WHEN `accountSecretArns` map is accessed, THE SecretSync SHALL protect it with `arnMu sync.RWMutex`
2. WHEN tests run with `-race` flag, THE SecretSync SHALL detect no race conditions
3. WHEN concurrent reads occur, THE SecretSync SHALL use `RLock()`
4. WHEN concurrent writes occur, THE SecretSync SHALL use `Lock()`
5. WHEN tests run, THE SecretSync SHALL validate safety under high load with concurrent access test

**Implementation:** `pkg/client/aws/aws.go` + tests in `aws_test.go`  
**Verification:** ✅ Tests pass with `-race` flag

---

### Requirement 21: CI/CD Improvements
**Status:** ✅ COMPLETE  
**User Story:** As a maintainer, I want modern CI workflows that enforce quality gates.

#### Acceptance Criteria
1. WHEN CI workflow references actions, THE SecretSync SHALL use semantic versions
2. WHEN linting runs, THE SecretSync SHALL use golangci-lint v2.7.2
3. WHEN tests run, THE SecretSync SHALL include race detector
4. WHEN PRs are created, THE SecretSync SHALL run all quality checks
5. WHEN main branch is updated, THE SecretSync SHALL run full test suite

**Status:**
- ✅ Semantic versions in CI
- ✅ golangci-lint v2.7.2 configured
- ✅ All lint errors fixed
- ✅ Integration tests added to CI workflow
- ✅ CI passing on all checks

---

## ADVANCED FEATURES (v1.2.0) - ⏳ PLANNED

### Requirement 22: AWS Organizations Discovery
**Status:** ⏳ PLANNED  
**User Story:** As an enterprise user, I want automatic discovery of AWS accounts in my organization.

#### Acceptance Criteria
1. WHERE discovery is enabled, THE SecretSync SHALL find all accounts in organization
2. WHERE account tags exist, THE SecretSync SHALL use them for filtering
3. WHERE delegated administrator is configured, THE SecretSync SHALL use that role
4. WHERE organizational units are specified, THE SecretSync SHALL discover only accounts in those OUs
5. WHEN discovery completes, THE SecretSync SHALL make account IDs and names available for target generation
6. IF discovery fails, THEN THE SecretSync SHALL explain permission requirements

**Implementation:** `pkg/discovery/organizations/` (to be created)

---

### Requirement 23: AWS Identity Center Integration
**Status:** ⏳ PLANNED  
**User Story:** As an Identity Center user, I want to sync permission sets and account assignments.

#### Acceptance Criteria
1. WHERE Identity Center is configured, THE SecretSync SHALL discover permission sets
2. WHERE account assignments exist, THE SecretSync SHALL map them to permission sets
3. WHEN syncing, THE SecretSync SHALL use permission set names as secret paths
4. WHEN assignment changes, THE SecretSync SHALL reflect updates in sync
5. WHERE Identity Center instance is in different region, THE SecretSync SHALL handle cross-region calls

**Implementation:** `pkg/discovery/identitycenter/` (partially exists)

---

### Requirement 24: Secret Versioning Support
**Status:** ⏳ PLANNED  
**User Story:** As a user, I want to track secret versions and roll back if needed.

#### Acceptance Criteria
1. WHEN secrets are synced, THE SecretSync SHALL preserve version metadata
2. WHERE AWS Secrets Manager versions exist, THE SecretSync SHALL use latest version by default
3. WHERE specific version is requested, THE SecretSync SHALL sync that version
4. WHERE version history is enabled, THE SecretSync SHALL make previous versions accessible
5. WHEN displaying diffs, THE SecretSync SHALL show version numbers

**Implementation:** Enhance `pkg/diff/` and `pkg/pipeline/s3_store.go`

---

### Requirement 25: Enhanced Diff Output
**Status:** ⏳ PLANNED  
**User Story:** As a user reviewing changes, I want detailed diff output with side-by-side comparison.

#### Acceptance Criteria
1. WHERE running with `--diff`, THE SecretSync SHALL clearly highlight changes
2. WHEN secret value changes, THE SecretSync SHALL show old and new values (masked)
3. WHEN new secrets are added, THE SecretSync SHALL mark them with `+` prefix
4. WHEN secrets are deleted, THE SecretSync SHALL mark them with `-` prefix
5. WHERE output format is `github`, THE SecretSync SHALL create annotations for PR reviews
6. WHERE output format is `json`, THE SecretSync SHALL provide structured diff
7. WHEN large diffs occur, THE SecretSync SHALL show summary statistics

**Implementation:** Enhance `pkg/diff/diff.go`

---

## Non-Functional Requirements

### Performance
- Pipeline SHALL complete within 5 minutes for 1,000 secrets
- Vault listing SHALL process 100 directories/second minimum
- AWS Secrets Manager sync SHALL process 50 secrets/second minimum
- Memory usage SHALL not exceed 500MB for typical workloads
- API response time p95 SHALL be < 500ms

### Reliability
- Pipeline SHALL succeed 99.9% of the time when services are healthy
- Transient failures SHALL be retried automatically
- Circuit breaker SHALL prevent cascade failures
- State SHALL be consistent (all or nothing for targets)
- Concurrent executions SHALL not interfere with each other

### Security
- Credentials SHALL never be logged
- All external connections SHALL use TLS
- Secrets SHALL never be written to disk unencrypted
- Path traversal attacks SHALL be prevented
- Input validation SHALL prevent injection attacks
- Least privilege principle SHALL be followed for IAM policies

### Maintainability
- Code coverage SHALL be ≥ 80%
- All public APIs SHALL have documentation comments
- Complex logic SHALL have inline comments explaining why
- Git commits SHALL follow Conventional Commits format
- Breaking changes SHALL be documented in CHANGELOG.md

### Usability
- Error messages SHALL be clear and actionable
- `--help` flag SHALL provide complete usage information
- Common operations SHALL be achievable with single command
- Configuration SHALL be validated before execution
- Progress indicators SHALL show long-running operations

---

## Current Status Summary

### ✅ Complete (v1.0)
- Requirements 1-14: Core pipeline functionality
- 113+ test functions passing
- Integration test infrastructure
- Race detector clean

### ✅ Complete (v1.1.0)
- Requirement 15: ✅ Metrics (complete, verified)
- Requirement 16: ✅ Circuit breaker (complete, lint fixed)
- Requirement 17: ✅ Error context (complete, verified)
- Requirement 18: ✅ Docker pinning (complete)
- Requirement 19: ✅ Queue compaction (complete, verified)
- Requirement 20: ✅ Race prevention (complete)
- Requirement 21: ✅ CI/CD (complete, integration tests added)

### ⏳ Planned (v1.2.0)
- Requirements 22-25: Advanced features

---

## Completed Actions (v1.1.0 Release)

1. ✅ **Fixed Lint Errors**
   - Fixed copylocks in VaultClient.DeepCopy()
   - Fixed staticcheck QF1003 in CircuitBreaker.WrapError()

2. ✅ **Verified v1.1.0 Features**
   - Verified metrics endpoint works end-to-end
   - Verified circuit breaker integration
   - Verified error context adoption
   - Verified queue compaction config

3. ✅ **Added Integration Tests to CI**
   - Added docker-compose job to CI workflow
   - Made it required check
   - Updated documentation

4. ✅ **Cleaned Up Issues**
   - Closed v1.0 issues (#20-25, #4)
   - All v1.1.0 issues verified complete

---

**Document Version:** 2.0 (Consolidated)  
**Last Updated:** December 9, 2025  
**Status:** Single source of truth for all versions
