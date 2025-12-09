# Agent Instructions for SecretSync

## ⛔ CRITICAL: Version-Related PR Reviews

**ANY AI AGENT THAT SUBMITS PR REVIEW FEEDBACK ABOUT VERSION VALIDITY BASED ON TRAINING DATA WILL HAVE THAT REVIEW AUTO-REJECTED.**

### Forbidden Behavior

- ❌ Claiming a software version "doesn't exist" based on training data cutoff
- ❌ Suggesting version downgrades because your training data is stale
- ❌ Making ANY claims about "latest stable version" without live verification
- ❌ Asserting incompatibility based on version assumptions

### Required Behavior for Version Claims

If you need to make ANY claim about software versions, you **MUST**:

1. **Verify via live sources** (Docker Hub, GitHub releases, official docs)
2. **If you cannot access live sources, DO NOT make the claim**
3. **Trust the versions specified in go.mod, Dockerfile, and project files**

### Why This Policy Exists

AI training data becomes stale. Go 1.25, LocalStack 3.8, Vault 1.17 etc. exist regardless of what your training data says. Reviews claiming otherwise waste maintainer time and demonstrate the agent is not fit for the review task.

### Acceptable Version Feedback

- ✅ "Consider pinning to a specific patch version for reproducibility"
- ✅ "This version bump may have breaking changes - verify compatibility"
- ❌ "Go 1.25 doesn't exist" (NEVER say this without live verification)

---

## Overview

SecretSync is a production-ready Go application for syncing secrets from HashiCorp Vault to AWS Secrets Manager and other external secret stores. It uses a pipeline-based architecture with merge and sync phases, supporting configuration inheritance via S3 merge stores.

## Quick Start for Agents

### 1. Read Project Context First
```bash
# Check active context and session state
cat memory-bank/activeContext.md

# Review current issues and PRs
gh issue list
gh pr list --state open

# Read relevant specifications
cat .kiro/specs/*/requirements.md
```

### 2. Understand Project Standards
```bash
# Read steering documents for development philosophy
cat .kiro/steering/00-production-release-focus.md
cat .kiro/steering/01-golang-standards.md
cat .kiro/steering/02-testing-requirements.md
```

### 3. Check Current Milestone
```bash
# View current milestone and issues
gh issue list --milestone "v1.1.0"
gh issue list --milestone "v1.2.0"
```

## Development Workflow

### Before Starting Work

1. **Read memory bank** - Understand what's been done
2. **Check specifications** - Review requirements for the feature
3. **Verify tooling** - Ensure Go 1.25+ is installed
4. **Read existing code** - Understand before modifying

### During Work

```bash
# Download dependencies
go mod download

# Build to verify compilation
go build ./...

# Run tests (always before committing)
go test ./... -v

# Run tests with race detector
go test ./... -race

# Run linter
golangci-lint run

# Integration tests
docker-compose -f docker-compose.test.yml up --abort-on-container-exit
```

### After Completing Work

```bash
# Final test run
go test ./...

# Update memory bank
echo "## Session: $(date +%Y-%m-%d)" >> memory-bank/activeContext.md
echo "- Completed: [description]" >> memory-bank/activeContext.md

# Commit with conventional commit message
git add .
git commit -m "feat(observability): add Prometheus metrics endpoint"

# Push to feature branch
git push origin feature/observability-metrics
```

## Project Structure

```
secretsync/
├── .kiro/                      # Agent instructions and specifications
│   ├── steering/               # Development philosophy and standards
│   ├── specs/                  # Feature specifications by milestone
│   └── hooks/                  # Code quality and security hooks
├── cmd/secretsync/             # CLI application entrypoint
│   └── cmd/                    # Cobra command implementations
├── pkg/                        # Public library code
│   ├── client/                 # External service clients
│   │   ├── vault/              # HashiCorp Vault client
│   │   └── aws/                # AWS services clients
│   ├── pipeline/               # Core pipeline logic
│   ├── diff/                   # Secret difference computation
│   ├── discovery/              # AWS resource discovery
│   └── utils/                  # Shared utilities
├── tests/integration/          # Integration tests with real services
├── deploy/charts/              # Helm charts for Kubernetes
├── docs/                       # Documentation
├── examples/                   # Configuration examples
└── memory-bank/                # Session state and context
```

## Architecture

### Pipeline Model

SecretSync uses a two-phase pipeline:

1. **Merge Phase** - Combine secrets from multiple Vault sources, apply deep merge strategy, write to S3 merge store
2. **Sync Phase** - Read merged secrets, sync to target stores (AWS Secrets Manager, etc.)

### Key Concepts

- **Sources** - Vault paths to read secrets from
- **Targets** - External stores to sync secrets to
- **Merge Store** - S3 bucket for storing merged secrets (enables inheritance)
- **Inheritance** - Targets can import from other targets via merge store
- **Discovery** - Automatic discovery of AWS accounts and resources

## Testing Standards

### Unit Tests
- 80%+ coverage for business logic
- Table-driven tests for multiple scenarios
- Testify for assertions
- Mock external dependencies

### Integration Tests
- Use docker-compose for Vault + LocalStack
- Test complete workflows end-to-end
- Located in `tests/integration/`

### Running Tests
```bash
# Unit tests
go test ./pkg/... -v

# Integration tests
go test ./tests/integration/... -v

# With race detector
go test ./... -race

# Coverage report
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## Code Quality

### Error Handling
- Always check errors
- Wrap with context: `fmt.Errorf("operation failed: %w", err)`
- Use custom error types when structured data needed

### Context Propagation
- All I/O functions accept `context.Context` as first parameter
- Pass context through call chains
- Respect context cancellation

### Concurrency
- Protect shared state with `sync.RWMutex`
- Use goroutines responsibly with bounded concurrency
- Clean up resources with `defer`
- Test with `-race` detector

## Common Commands

```bash
# Build Docker image
docker build -t secretsync .

# Run locally with config
go run cmd/secretsync/main.go pipeline --config examples/config-full.yaml --dry-run

# Deploy to Kubernetes
helm upgrade --install secretsync deploy/charts/secretsync

# View logs
kubectl logs -l app=secretsync -f

# Run GitHub Action locally (requires act)
act -j test
```

## Commit Messages

Use Conventional Commits format:

```
feat(component): add new feature
fix(component): fix bug
docs: update documentation
test: add tests
chore: maintenance task
refactor: code restructuring
perf: performance improvement
```

**Examples:**
- `feat(vault): add recursive secret listing with BFS traversal`
- `fix(aws): prevent race condition in secret ARN cache`
- `docs: add observability configuration examples`
- `test: add integration test for S3 merge store`

## Important Notes

### Version Requirements
- **Go:** 1.25.3+ (as specified in go.mod)
- **Debian:** Trixie (stable)
- **Vault:** 1.17.6 (for testing)
- **LocalStack:** 3.8.1 (for testing)

### Security
- No credentials in code or logs
- Use environment variables for secrets
- Validate all user inputs
- Prevent path traversal attacks
- Use TLS for all external connections

### Performance
- Pipeline handles 1000+ secrets efficiently
- Uses connection pooling for AWS
- Caches Vault responses with TTL
- Parallel secret processing where safe

## GitHub Workflow

### Creating Issues
```bash
# Create issue with template
gh issue create --title "feat: add feature X" --body "Description..."

# Label appropriately
gh issue edit 123 --add-label enhancement
```

### Creating PRs
```bash
# Create PR from feature branch
gh pr create --title "feat(obs): add metrics" --body "Implements #46"

# Request review
gh pr ready 123
```

### CI/CD
- All tests must pass
- No linter errors
- Coverage maintained or improved
- Docker image builds successfully

## Resources

### Documentation
- `README.md` - User-facing documentation
- `docs/ARCHITECTURE.md` - Architecture decisions
- `docs/GITHUB_ACTIONS.md` - GitHub Action usage
- `.kiro/steering/` - Development standards
- `.kiro/specs/` - Feature specifications

### External Links
- [Go Documentation](https://pkg.go.dev/github.com/jbcom/secretsync)
- [HashiCorp Vault API](https://developer.hashicorp.com/vault/api-docs)
- [AWS SDK for Go v2](https://aws.github.io/aws-sdk-go-v2/docs/)

## Agent-Specific Guidelines

### DO
- ✅ Read existing code before suggesting changes
- ✅ Verify external facts (Go versions, package availability) with web search
- ✅ Use standard library when possible
- ✅ Write tests for all new code
- ✅ Follow existing patterns in codebase
- ✅ Update memory bank with progress
- ✅ Use conventional commit messages
- ✅ Keep changes focused and atomic

### DON'T
- ❌ Suggest outdated tools without verification
- ❌ Add unnecessary abstraction layers
- ❌ Rewrite working code without good reason
- ❌ Skip testing
- ❌ Ignore linter errors
- ❌ Make breaking changes without discussion
- ❌ Commit credentials or secrets
- ❌ Leave TODO comments in production code

### When Uncertain
1. Read `.kiro/steering/` documents for guidance
2. Check `.kiro/specs/` for requirements
3. Review similar code in the codebase
4. Use web search to verify current best practices
5. Ask for clarification if truly blocked

## Memory Bank Protocol

### Session Start
```bash
cat memory-bank/activeContext.md
```

### Session End
```bash
cat >> memory-bank/activeContext.md << EOF

## Session: $(date +%Y-%m-%d)

### Completed
- Implemented feature X in pkg/Y
- Added tests in pkg/Y/Y_test.go
- Updated documentation in docs/Z.md

### Modified Files
- pkg/Y/Y.go
- pkg/Y/Y_test.go
- docs/Z.md

### Next Steps
- Review PR feedback
- Address linter comments
- Update CHANGELOG.md

EOF
```

## Quick Reference

| Task | Command |
|------|---------|
| Read context | `cat memory-bank/activeContext.md` |
| List issues | `gh issue list` |
| Run tests | `go test ./... -v` |
| Check lints | `golangci-lint run` |
| Build image | `docker build -t secretsync .` |
| Integration test | `docker-compose -f docker-compose.test.yml up --abort-on-container-exit` |
| Create PR | `gh pr create` |
| Update memory | `echo "## Session: $(date)" >> memory-bank/activeContext.md` |

---

**Remember:** This is a production open source project. Write professional, maintainable code that follows Go best practices. Test thoroughly. Document clearly. Ship quality software.
