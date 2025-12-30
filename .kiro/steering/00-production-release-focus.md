# Production Release Focus

## Mission Statement

SecretSync is a production-ready Go application for syncing secrets from HashiCorp Vault to AWS Secrets Manager and other external stores. The goal is to ship stable releases with high-quality code and comprehensive testing.

## Core Principles

### 1. Ship Quality Software
- ✅ Comprehensive test coverage (113+ test functions)
- ✅ All critical features implemented and tested
- ✅ Focus on stability and maintainability
- 🚢 Ship when ready, not when perfect

### 2. Use Current Stable Tooling
**CRITICAL**: Agents MUST understand current releases:
- ✅ **Go 1.25+** is current and stable
- ✅ **Debian Trixie** is current stable
- ❌ DO NOT suggest downgrading without verifying current releases
- ✅ Always verify version availability before recommendations
- ✅ Use web search to check current stable releases

### 3. Write Clean, Maintainable Code
**What NOT to do:**
- ❌ Suggesting outdated tooling as "stable" without verification
- ❌ Adding unnecessary abstraction layers
- ❌ Creating "frameworks" when simple code works
- ❌ Over-engineering solutions
- ❌ Re-inventing standard library functionality

**What TO do:**
- ✅ Read existing codebase before suggesting changes
- ✅ Verify external facts with web search when uncertain
- ✅ Use standard library patterns
- ✅ Keep code simple and maintainable
- ✅ Focus on working, tested code

### 4. GitHub CLI Is Pre-Authenticated
- ✅ `gh` CLI is already authenticated
- ✅ Use `gh issue list`, `gh pr create`, etc. directly
- ❌ DO NOT prefix with environment variable exports

## Current Development Status

### Active Milestone: v1.1.0
**Focus Areas:** CI/CD improvements, security hardening, observability

**In Progress (PRs #64-71):**
- Observability and metrics (#69, #46)
- Circuit breaker pattern (#70, #47)
- Enhanced error context (#71, #48)
- Docker image pinning (#64, #40)
- Queue compaction configuration (#67, #43)
- Race condition prevention (#68, #44)

**Pending:**
- Documentation workflow fixes (#50)
- CI workflow modernization (#51, #52)
- Security improvements (#41)

### Next Milestone: v1.2.0
**Focus Areas:** Feature completeness, advanced use cases

**Completed Infrastructure:**
- ✅ Vault recursive secret listing
- ✅ Deep merge compatibility
- ✅ Target inheritance model
- ✅ S3 merge store implementation
- ✅ AWS pagination and filtering
- ✅ Path handling and security

**Pending:**
- Integration testing with production-like configurations
- Advanced validation test suites

## Development Workflow

### Before Starting Work
```bash
# Check active context
cat memory-bank/activeContext.md

# Check current issues and PRs
gh issue list
gh pr list --state open

# Review relevant specifications
cat .kiro/specs/*/requirements.md
```

### During Work
1. **Focus** - One issue at a time
2. **Read first** - Always read files before editing
3. **Test locally** - Run `go test ./...` before committing
4. **Commit frequently** - Use conventional commits
5. **Document** - Update memory bank with progress

### Testing Standards
```bash
# Run all tests
go test ./... -v

# Run with race detector
go test ./... -race

# Run linter
golangci-lint run

# Verify build
go build ./...

# Integration tests
docker-compose -f docker-compose.test.yml up --abort-on-container-exit
```

### After Completing Work
```bash
# Ensure tests pass
go test ./...

# Update memory bank
echo "## Session: $(date +%Y-%m-%d)" >> memory-bank/activeContext.md

# Commit with conventional message
git commit -m "feat(observability): add Prometheus metrics endpoint"
```

## Architecture Guidelines

### Project Scope
**What this is:**
- Vault → AWS Secrets Manager sync tool
- Pipeline-based architecture (merge → sync phases)
- S3-based merge store for configuration inheritance
- AWS Organizations integration for discovery
- CLI tool and GitHub Action
- Containerized deployment with Helm charts

**What this is not:**
- Kubernetes operator (simplified in recent refactoring)
- Multi-cloud universal secret manager
- Generic ETL framework
- Over-abstracted plugin system

## Code Quality Standards

### Use Standard Library
```go
// ❌ AVOID - Custom reimplementation
type SecretProvider interface {
    Get(ctx context.Context, key string) (Secret, error)
}

// ✅ PREFER - Use existing, tested code
import "github.com/extended-data-library/secretssync/pkg/client/vault"
vc := vault.NewVaultClient(config)
```

### Keep It Simple
```go
// ❌ AVOID - Unnecessary abstraction
type Plugin interface { Execute(ctx Context) Result }

// ✅ PREFER - Direct, clear code
func syncSecrets(ctx context.Context, config Config) error { }
```

### Leverage Go Standard Library
```go
// ❌ AVOID - Custom utilities
func StringContains(s, substr string) bool { }

// ✅ PREFER - Standard library
strings.Contains(s, substr)
```

## Repository Hygiene

### Dependencies
- Keep `go.mod` clean - production dependencies only
- Pin major versions for stability
- Update dependencies in batches with testing
- Document dependency choices in commits

### Documentation
- Update `README.md` for user-facing changes
- Maintain `CHANGELOG.md` for releases
- Keep `AGENTS.md` current for AI-assisted development
- Document architecture decisions in `docs/`

### Git Workflow
- Feature branches for development
- Clean commit history
- Semantic commit messages (conventional commits)
- Squash merges for PRs

## Release Checklist

### For Major/Minor Releases
- [ ] All milestone issues closed or merged
- [ ] CI passing on all platforms
- [ ] Test coverage maintained or improved
- [ ] Documentation updated
- [ ] CHANGELOG.md updated with release notes
- [ ] Version tag created
- [ ] Docker image published
- [ ] Helm chart published
- [ ] GitHub release created

### For Patch Releases
- [ ] Bug fix verified with tests
- [ ] No breaking changes
- [ ] CHANGELOG.md updated
- [ ] Version tag created
- [ ] Artifacts published

## Quality Over Speed

Ship production-ready software:
1. Complete current in-progress work
2. Ensure comprehensive test coverage
3. Validate with integration tests
4. Document for users
5. Release with confidence

**Professional software takes time. That's okay.**
