# Active Context

## Project Overview

SecretSync is a production-ready Go application for syncing secrets from HashiCorp Vault to AWS Secrets Manager and other external secret stores. It uses a pipeline-based architecture with configuration inheritance via S3 merge stores.

### Key Features
- Vault KV2 recursive secret listing
- Deep merge strategy for configuration
- Target inheritance model via S3
- AWS Organizations discovery
- Pipeline-based sync (merge + sync phases)
- GitHub Action and CLI

### Current Status
- **Version:** Working toward v1.1.0
- **Go Version:** 1.25.3+
- **Test Coverage:** 113+ test functions
- **Architecture:** Simplified from Kubernetes operator to focused CLI tool

## Active Development

### v1.1.0 Milestone - Observability & Reliability

**In Progress (PRs #64-71):**
- #69, #46: Prometheus metrics endpoint
- #70, #47: Circuit breaker pattern for API resilience
- #71, #48: Enhanced error messages with request IDs
- #64, #40: Docker image version pinning
- #67, #43: Configurable queue compaction
- #68, #44: Race condition prevention tests

**Pending:**
- #50: Fix documentation workflow
- #51: Modernize CI workflows
- #52: Consolidate documentation CI
- #41: Command injection prevention

### v1.2.0 Milestone - Advanced Features

**Completed Infrastructure:**
- ✅ Vault recursive listing (PR #29, Issue #23)
- ✅ Deep merge compatibility (PR #29, Issue #21)
- ✅ Target inheritance (PR #29, Issue #22)
- ✅ S3 merge store (PR #29, Issue #4)
- ✅ AWS pagination (#24)
- ✅ Path security (#25)

**Planned Features:**
- AWS Organizations discovery enhancements
- AWS Identity Center integration
- Secret versioning support
- Enhanced diff output

## Architecture

### Package Structure
```
pkg/
├── client/          # External service clients
│   ├── vault/       # Vault KV2 client with recursive listing
│   └── aws/         # AWS services (Secrets Manager, S3, Organizations)
├── pipeline/        # Core pipeline logic (merge + sync)
├── diff/            # Secret difference computation
├── discovery/       # AWS resource discovery
└── utils/           # Shared utilities (deepmerge, etc.)
```

### Key Patterns
- **Dependency Injection:** Interfaces for testability
- **Context Propagation:** All I/O operations accept `context.Context`
- **Error Wrapping:** Structured errors with `fmt.Errorf(...: %w, err)`
- **Table-Driven Tests:** Comprehensive test coverage

## Development Workflow

### Session Protocol

**Start of Session:**
1. Read `memory-bank/activeContext.md` (this file)
2. Check `.kiro/steering/` for development standards
3. Review `.kiro/specs/` for feature requirements
4. Check current issues: `gh issue list`
5. Check open PRs: `gh pr list --state open`

**During Session:**
- Work on one issue/feature at a time
- Read code before modifying
- Write tests for all changes
- Run `go test ./...` before committing
- Update this file with progress

**End of Session:**
- Document completed work below
- Note any blockers or next steps
- Update issue/PR status
- Commit with conventional commit message

## Session History

### Session: 2024-12-09 - .kiro Structure Setup

**Completed:**
- Created `.kiro/` directory structure for agent instructions
- Added steering documents:
  - `00-production-release-focus.md` - Project philosophy and release focus
  - `01-golang-standards.md` - Go coding standards and patterns
  - `02-testing-requirements.md` - Testing philosophy and requirements
- Added specifications:
  - `v1.1.0-observability/requirements.md` - v1.1.0 detailed requirements
  - `v1.2.0-advanced-features/requirements.md` - v1.2.0 detailed requirements
- Added hooks for code quality:
  - `go-security-scanner.kiro.hook` - Security vulnerability detection
  - `go-code-quality.kiro.hook` - Code quality and Go idioms
  - `docs-consistency.kiro.hook` - Documentation consistency checking
- Updated `AGENTS.md` with comprehensive agent workflow instructions
- Updated `memory-bank/activeContext.md` with handoff protocol

**Purpose:**
- Establish clear standards for AI-assisted development
- Prevent common mistakes (version confusion, unnecessary rewrites)
- Document requirements for both active milestones
- Enable consistent, high-quality contributions from agents

**Files Modified:**
- `.kiro/steering/00-production-release-focus.md` (new)
- `.kiro/steering/01-golang-standards.md` (new)
- `.kiro/steering/02-testing-requirements.md` (new)
- `.kiro/specs/v1.1.0-observability/requirements.md` (new)
- `.kiro/specs/v1.2.0-advanced-features/requirements.md` (new)
- `.kiro/hooks/go-security-scanner.kiro.hook` (new)
- `.kiro/hooks/go-code-quality.kiro.hook` (new)
- `.kiro/hooks/docs-consistency.kiro.hook` (new)
- `AGENTS.md` (updated)
- `memory-bank/activeContext.md` (this file, updated)

**Next Steps:**
- Continue work on open PRs (#64-71)
- Address remaining v1.1.0 issues (#50, #51, #52, #41)
- Prepare for v1.1.0 release

---

### Session: 2024-12-09 - Dependency Management and Issue Triage

**Completed:**
- Merged 7 Dependabot PRs (#31-35, #37, #39)
- Updated Dependabot configuration to group updates (#49)
- Filed new issues for CI problems (#50, #51, #52)
- Created milestones and triaged all open issues
- Updated GitHub project board with all issues

**Files Modified:**
- `.github/dependabot.yml`
- `go.mod`, `go.sum` (via merged PRs)
- `memory-bank/activeContext.md`

**Milestones Created:**
- v1.1.0 - CI/Security/Observability (10 issues)
  - #40, #41, #43, #44, #46, #47, #48, #50, #51, #52
- v1.2.0 - Advanced Features (7 issues)
  - #4, #20, #21, #22, #23, #24, #25

**GitHub Project Updated:**
- Added all 17 secretsync issues to "jbcom Ecosystem Integration" project

### Release Branch Created
- `release/v1.1.0` created from main
- All v1.1.0 issues linked to this branch
- Feature work should branch from `release/v1.1.0`
- PRs should target `release/v1.1.0`

---

### Session: 2024-12-08 - Epic #26 Assessment

**Completed:**
- Comprehensive assessment of EPIC #26 implementation
- Documented all P0, P1, P2 requirements status
- Created `EPIC_26_ASSESSMENT.md` with detailed analysis
- Verified 95% completion of epic requirements

**Key Findings:**
- All P0 and P1 technical work complete
- 113+ test functions with comprehensive coverage
- ~8,500 lines of code removed (simplified architecture)
- Security enhancements: type safety, path validation
- Performance improvements: caching, efficient traversal

---

### Session: 2024-12-08 - SecretSync 1.0 Implementation (PR #29)

**Completed:**
- Implemented Vault recursive listing with BFS traversal (#23)
- Added deepmerge compatibility tests (#21)
- Implemented target inheritance model (#22)
- Enhanced S3 merge store operations (#4)
- Security hardening: path validation, type-safe parsing
- Performance: TTL caching for AWS ListSecrets
- Removed Kubernetes operator complexity (~13k lines)

**Files Modified:**
- `pkg/client/vault/vault.go` - BFS recursive listing
- `pkg/utils/deepmerge.go`, `pkg/utils/deepmerge_test.go` - Merge strategy tests
- `pkg/pipeline/config.go` - Target inheritance detection
- `pkg/pipeline/s3_store.go` - S3 merge store operations
- `pkg/client/aws/aws.go` - Caching and security
- Removed: `api/`, `internal/controller/` (operator code)

**Tests Added:**
- 6 scenarios for Vault recursive listing
- 13 functions for deepmerge compatibility
- 6 scenarios for target inheritance
- Integration tests with docker-compose

---

## Current Priorities

1. **Complete v1.1.0 PRs** - Finish in-progress work (#64-71)
2. **Fix CI/CD workflows** - Address #50, #51, #52
3. **Security hardening** - Address #41 (command injection)
4. **Release v1.1.0** - Tag, publish, announce
5. **Plan v1.2.0** - Advanced features and enterprise use cases

## Known Issues

- Documentation workflow needs Go-friendly configuration (#50)
- CI workflows use SHA pins instead of semantic versions (#51)
- Multiple documentation workflows should be consolidated (#52)
- Command injection risk in external script execution (#41)

## Resources

### Documentation
- `README.md` - User guide
- `AGENTS.md` - Agent instructions (comprehensive)
- `docs/ARCHITECTURE.md` - Architecture decisions
- `docs/GITHUB_ACTIONS.md` - GitHub Action usage
- `.kiro/steering/` - Development standards
- `.kiro/specs/` - Feature specifications

### Key Files
- `go.mod` - Go 1.25.3, production dependencies
- `docker-compose.test.yml` - Integration test environment
- `Dockerfile` - Multi-stage build (Go 1.25-trixie → Debian trixie-slim)
- `action.yml` - GitHub Action definition

### External Services
- **Docker Hub:** `extended-data-library/secretssync` (image registry)
- **GitHub:** Issues, PRs, Actions, Packages
- **Vault:** HashiCorp Vault 1.17.6 (testing)
- **LocalStack:** AWS service mocks for testing

---

*Last updated: 2025-12-09*

## Progress Update: 2025-12-09
- PR #68: Enhanced test robustness merged
- PR #64: Docker image pinning merged
- PR #71: Error formatting and type assertion safety merged
- PR #67: Queue compaction threshold merged
- Circuit breaker work integrated (PR #70 closed after cherry-pick)

## Session Progress: 2025-12-09

### Completed PRs
- **PR #68**: Enhanced test robustness (DeepCopyConcurrentSafety)
- **PR #64**: Fixed critical action.yml placeholder digest issue
- **PR #71**: Fixed error message formatting and type assertion safety
- **PR #67**: Adaptive queue compaction threshold
- **PR #70**: Circuit breaker across Vault/AWS/S3 with nil-breaker fixes

### Key Fixes
- Organizations paginators wrapped with circuit breakers
- VaultClient/AwsClient: sync.Once ensureBreaker to avoid races
- Error context: leading space bug fixed, strings.Join formatting
- action.yml: stable tag reference; digest automation deferred

### Next Steps
- PR #69: Observability metrics/docs/tests to finish and merge
- Release PR #61: finalize after #69, then tag v1.1.0
