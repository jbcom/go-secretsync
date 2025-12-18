# SecretSync - Complete System Design

## Executive Summary

SecretSync is a production-ready Go application that synchronizes secrets from HashiCorp Vault to AWS Secrets Manager and other external secret stores. It uses a two-phase pipeline architecture (merge + sync) with S3-based configuration inheritance, enabling enterprise-scale secret management across multi-account AWS environments.

**Current State:**
- ✅ v1.0: COMPLETE (113+ tests passing)
- ⚠️ v1.1.0: PARTIAL (2 lint errors, most features done)
- ⏳ v1.2.0: PLANNED (some features complete, others pending)

**Target Users:**
- DevOps Engineers managing multi-account AWS environments
- Platform Engineers building secret management infrastructure
- Security Teams enforcing secret rotation policies
- Organizations migrating from Vault to AWS Secrets Manager

## System Architecture

### High-Level Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                         SecretSync Pipeline                      │
├─────────────────────────────────────────────────────────────────┤
│                                                                   │
│  ┌─────────────┐      ┌─────────────┐      ┌─────────────┐     │
│  │   Sources   │─────▶│    Merge    │─────▶│  S3 Store   │     │
│  │   (Vault)   │      │    Phase    │      │  (Optional) │     │
│  └─────────────┘      └─────────────┘      └─────────────┘     │
│                              │                      │            │
│                              │                      │            │
│                              ▼                      ▼            │
│                       ┌─────────────┐      ┌─────────────┐     │
│                       │    Sync     │◀─────│ Inheritance │     │
│                       │    Phase    │      │  Resolution │     │
│                       └─────────────┘      └─────────────┘     │
│                              │                                   │
│  ┌─────────────┐            │            ┌─────────────┐       │
│  │   Targets   │◀───────────┴───────────▶│  Discovery  │       │
│  │ (AWS SM)    │                          │ (AWS Orgs)  │       │
│  └─────────────┘                          └─────────────┘       │
│                                                                   │
└─────────────────────────────────────────────────────────────────┘
```

### Component Breakdown

#### 1. Vault Client (`pkg/client/vault/`)

**Purpose:** Read secrets from HashiCorp Vault KV2 engine

**Key Features:**
- Recursive secret listing using BFS traversal
- Cycle detection to prevent infinite loops
- AppRole authentication
- Token renewal
- Path validation and security

**Implementation:**
```go
type VaultClient struct {
    client      *vault.Client
    api         LogicalClient  // Interface for dependency injection
    mountPath   string
    visited     map[string]bool // Cycle detection
}

// BFS recursive listing
func (vc *VaultClient) ListSecretsRecursive(ctx context.Context, path string) ([]string, error)
```

**Security:**
- Path traversal prevention (`..`, null bytes)
- Type-safe response parsing
- No credentials in logs

**Queue Compaction (v1.1.0 - Requirement 19):**

**Status:** ❓ Needs verification

**Purpose:** Optimize memory usage during BFS traversal of large secret hierarchies

**Configuration:**
```yaml
vault_sources:
  - mount: secret/base/
    max_secrets: 10000
    queue_compaction_threshold: 500  # Compact when queue exceeds this
```

**Behavior:**
- Default threshold: `min(1000, maxSecretsPerMount/100)`
- Compaction triggers when: `queue_index > threshold AND queue_index > len(queue)/2`
- Compaction removes processed items from queue
- Logs compaction events with old/new queue sizes

**Implementation:**
```go
type VaultClient struct {
    // ... existing fields
    queueCompactionThreshold int
}

func (vc *VaultClient) compactQueue(queue []string, index int) []string {
    if index > vc.queueCompactionThreshold && index > len(queue)/2 {
        log.Infof("compacting queue: old_size=%d new_size=%d", len(queue), len(queue)-index)
        return queue[index:]
    }
    return queue
}
```

**Action Required:** Verify config field exists and is used in BFS traversal

#### 2. AWS Client (`pkg/client/aws/`)

**Purpose:** Interact with AWS Secrets Manager and other AWS services

**Key Features:**
- Secrets Manager CRUD operations
- Pagination handling (NextToken)
- Empty secret filtering
- ARN caching with TTL
- Cross-account role assumption

**Implementation:**
```go
type AwsClient struct {
    smClient           *secretsmanager.Client
    s3Client           *s3.Client
    orgsClient         *organizations.Client
    accountSecretArns  map[string]string
    arnMu              sync.RWMutex  // Race condition protection
}
```

**Performance:**
- TTL-based caching for ListSecrets
- Connection pooling
- Parallel secret fetching (bounded concurrency)

#### 3. Pipeline (`pkg/pipeline/`)

**Purpose:** Orchestrate the two-phase sync process

**Architecture:**
```
Merge Phase:
1. Read secrets from Vault sources
2. Apply deep merge strategy
3. Write merged output to S3 merge store (optional)

Sync Phase:
1. Read merged secrets from memory or S3
2. Resolve target inheritance
3. Sync to AWS Secrets Manager targets
4. Compute diffs (optional)
```

**Key Functions:**
```go
func (p *Pipeline) Execute(ctx context.Context, config *Config) error {
    // Merge phase
    merged := p.mergeSecrets(ctx, config.VaultSources)
    p.writeMergeStore(ctx, merged)
    
    // Sync phase
    targets := p.resolveInheritance(ctx, config.Targets)
    return p.syncTargets(ctx, targets)
}
```

**Topological Sorting:**
- Determines execution order based on dependencies
- Detects circular dependencies
- Enables target inheritance

#### 4. Deep Merge (`pkg/utils/deepmerge.go`)

**Purpose:** Merge configuration from multiple sources

**Strategy:**
- Lists: Append (not replace)
- Maps: Recursive merge
- Sets: Union
- Scalars: Override
- Type conflicts: Override with new value

**Example:**
```go
base := map[string]interface{}{
    "api_keys": []interface{}{"key1", "key2"},
    "config": map[string]interface{}{
        "timeout": 30,
        "retries": 3,
    },
}

overlay := map[string]interface{}{
    "api_keys": []interface{}{"key3"},
    "config": map[string]interface{}{
        "timeout": 60,  // Override
        "debug": true,  // Add new
    },
}

result := DeepMerge(base, overlay)
// api_keys: ["key1", "key2", "key3"]
// config: {timeout: 60, retries: 3, debug: true}
```

#### 5. S3 Merge Store (`pkg/pipeline/s3_store.go`)

**Purpose:** Store merged secrets for inheritance and auditing

**Operations:**
```go
type S3MergeStore struct {
    client     *s3.Client
    bucketName string
    prefix     string
}

func (s *S3MergeStore) WriteSecret(ctx context.Context, target, path string, data map[string]interface{}) error
func (s *S3MergeStore) ReadSecret(ctx context.Context, target, path string) (map[string]interface{}, error)
func (s *S3MergeStore) ListSecrets(ctx context.Context, target string) ([]string, error)
```

**Storage Format:**
```
s3://bucket/prefix/
├── target-a/
│   ├── secret1.json
│   └── secret2.json
└── target-b/
    └── secret3.json
```

#### 6. Discovery (`pkg/discovery/`)

**Purpose:** Automatically discover AWS resources

**Current Implementation:**
- AWS Organizations account discovery (basic)
- Tag-based filtering (basic)
- Organizational Unit filtering (basic)

**Planned Enhancements (v1.2.0 - Requirement 22):**
- Multiple tag filters with wildcards
- Tag combination logic (AND/OR)
- Nested OU traversal
- Account status filtering (exclude suspended/closed)
- Discovery caching with TTL (1 hour)

**AWS Identity Center Integration (v1.2.0 - Requirement 23):**
- Permission set discovery
- Account assignment mapping
- Cross-region support
- Assignment caching (TTL: 30 min)

**Implementation:**
```go
type DiscoveryService struct {
    orgsClient    *organizations.Client
    ssoClient     *ssoadmin.Client
    cache         *DiscoveryCache
    filters       []Filter
}

func (d *DiscoveryService) DiscoverAccounts(ctx context.Context) ([]Account, error)
func (d *DiscoveryService) DiscoverPermissionSets(ctx context.Context) ([]PermissionSet, error)
```

#### 7. Secret Versioning (v1.2.0 - Requirement 24)

**Purpose:** Track secret versions and enable rollback

**Status:** ⏳ PLANNED

**Features:**
- Version tracking in diff engine
- Version metadata storage in S3
- Version rollback capability
- Version history retention

**Implementation:**
```go
type SecretVersion struct {
    Path      string
    Version   int
    Data      map[string]interface{}
    Timestamp time.Time
    Author    string
}

type VersionStore interface {
    GetVersion(ctx context.Context, path string, version int) (*SecretVersion, error)
    ListVersions(ctx context.Context, path string) ([]SecretVersion, error)
    GetLatest(ctx context.Context, path string) (*SecretVersion, error)
}
```

**S3 Storage Format:**
```
s3://bucket/prefix/
├── target-a/
│   ├── secret1/
│   │   ├── v1.json
│   │   ├── v2.json
│   │   └── latest.json (symlink)
│   └── secret2/
│       └── v1.json
```

**CLI Usage:**
```bash
# Sync specific version
secretsync pipeline --config config.yaml --version 5

# Show version history
secretsync version list --path production/api/key

# Rollback to previous version
secretsync version rollback --path production/api/key --version 3
```

**Diff Output with Versions:**
```
production/db/password:
  Old: v5 (2025-12-08 10:30:00)
  New: v6 (2025-12-09 14:15:00)
  Changed: value
```

#### 8. Diff Engine (`pkg/diff/`)

**Purpose:** Compute differences between secret states

**Current Output Formats:**
- Text (colored terminal output)
- JSON (structured data)
- GitHub (PR annotations)

**Example Output:**
```
Diff Summary:
  Added:    5 secrets
  Modified: 3 secrets
  Deleted:  1 secret

Changes:
  + production/api/new-key
  ~ production/db/password (value changed)
  - staging/old-token
```

**Planned Enhancements (v1.2.0 - Requirement 25):**

**Side-by-Side Comparison:**
```
production/db/password:
  Old: ********** (masked)
  New: ********** (masked)
  Changed: value
```

**Value Masking:**
- Mask sensitive values by default
- `--show-values` flag to reveal
- Pattern-based masking (API keys, passwords)

**GitHub Output Format:**
```json
{
  "annotations": [
    {
      "path": "production/api/new-key",
      "annotation_level": "notice",
      "message": "Secret added"
    }
  ]
}
```

**JSON Output Format:**
```json
{
  "summary": {
    "added": 5,
    "modified": 3,
    "deleted": 1
  },
  "changes": [
    {
      "path": "production/api/new-key",
      "type": "added",
      "old_value": null,
      "new_value": "***"
    }
  ]
}
```

**Implementation:**
```go
type DiffFormatter interface {
    Format(diff *Diff) (string, error)
}

type SideBySideFormatter struct { }
type GitHubFormatter struct { }
type JSONFormatter struct { }
```

### Data Flow

#### Merge Phase Flow

```
1. Load Configuration
   ↓
2. Initialize Vault Client
   ↓
3. For each VaultSource:
   ├─ List secrets recursively
   ├─ Read secret values
   └─ Add to merge map
   ↓
4. Apply Deep Merge
   ↓
5. Write to S3 Merge Store (optional)
```

#### Sync Phase Flow

```
1. Load Merged Secrets (memory or S3)
   ↓
2. Resolve Target Dependencies
   ├─ Topological sort
   ├─ Detect circular refs
   └─ Build execution order
   ↓
3. For each Target (in order):
   ├─ Resolve imports (S3)
   ├─ Apply deep merge
   ├─ Initialize AWS client
   ├─ List existing secrets
   ├─ Compute diff
   ├─ Apply changes
   └─ Record metrics
```

## Configuration Model

### Complete Configuration Example

```yaml
# Vault sources to read from
vault_sources:
  - mount: secret/base/
    max_secrets: 10000
    queue_compaction_threshold: 500
  - mount: secret/production/
    max_secrets: 5000

# S3 merge store (optional)
merge_store:
  enabled: true
  type: s3
  bucket: my-secrets-merge-store
  prefix: merged/
  region: us-east-1

# Sync targets
targets:
  - name: production-us-east-1
    type: aws_secretsmanager
    region: us-east-1
    role_arn: arn:aws:iam::123456789012:role/SecretSync
    imports:
      - base_merged  # From merge store
    overrides:
      environment: production

  - name: staging-us-west-2
    type: aws_secretsmanager
    region: us-west-2
    role_arn: arn:aws:iam::987654321098:role/SecretSync
    imports:
      - production-us-east-1  # Inherit from another target
    overrides:
      environment: staging

# Discovery (optional)
discovery:
  enabled: true
  type: aws_organizations
  filters:
    - tag: Environment
      values: [production, staging]
    - ou: ou-prod-xxxx
  role_arn: arn:aws:iam::123456789012:role/OrgDiscovery

# Observability (v1.1.0 - Requirements 15, 16)
metrics:
  enabled: true
  port: 9090
  path: /metrics

circuit_breaker:
  enabled: true
  failure_threshold: 5      # Open after 5 failures
  timeout: 30s              # Stay open for 30 seconds
  max_requests: 1           # Allow 1 request in half-open state
  window: 10s               # Count failures in 10 second window
```

## Deployment Models

### 1. CLI Usage

```bash
# Dry run to preview changes
secretsync pipeline --config config.yaml --dry-run

# Execute sync
secretsync pipeline --config config.yaml

# With diff output
secretsync pipeline --config config.yaml --diff --output github

# Merge phase only
secretsync pipeline --config config.yaml --merge-only

# Sync phase only (reads from S3)
secretsync pipeline --config config.yaml --sync-only

# With metrics endpoint (v1.1.0)
secretsync pipeline --config config.yaml --metrics-port 9090
```

### 2. GitHub Action

```yaml
- uses: jbcom/secretsync@v1
  with:
    config: .secretsync/config.yaml
    dry-run: 'false'
    diff: 'true'
    output-format: 'github'
  env:
    VAULT_ADDR: ${{ secrets.VAULT_ADDR }}
    VAULT_ROLE_ID: ${{ secrets.VAULT_ROLE_ID }}
    VAULT_SECRET_ID: ${{ secrets.VAULT_SECRET_ID }}
```

### 3. Kubernetes CronJob (Future)

```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: secretsync
spec:
  schedule: "*/15 * * * *"
  jobTemplate:
    spec:
      template:
        spec:
          serviceAccountName: secretsync
          containers:
          - name: secretsync
            image: jbcom/secretsync:v1
            args:
            - pipeline
            - --config
            - /config/config.yaml
```

## Security Model

### Authentication

**Vault:**
- AppRole (role_id + secret_id)
- Environment variables: `VAULT_ROLE_ID`, `VAULT_SECRET_ID`
- Token renewal handled automatically

**AWS:**
- IRSA (IAM Roles for Service Accounts) in Kubernetes
- OIDC for GitHub Actions
- Role assumption for cross-account access
- Environment variables: `AWS_ROLE_ARN` or config role_arn

### Authorization

**Vault Policies:**
```hcl
# Read-only access to secret mounts
path "secret/data/*" {
  capabilities = ["read", "list"]
}

path "secret/metadata/*" {
  capabilities = ["list"]
}
```

**AWS IAM Policies:**
```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "secretsmanager:ListSecrets",
        "secretsmanager:GetSecretValue",
        "secretsmanager:CreateSecret",
        "secretsmanager:UpdateSecret",
        "secretsmanager:DeleteSecret"
      ],
      "Resource": "*"
    },
    {
      "Effect": "Allow",
      "Action": [
        "s3:GetObject",
        "s3:PutObject",
        "s3:ListBucket"
      ],
      "Resource": [
        "arn:aws:s3:::merge-store-bucket",
        "arn:aws:s3:::merge-store-bucket/*"
      ]
    }
  ]
}
```

### Data Protection

- **In Transit:** TLS for all external connections
- **At Rest:** S3 server-side encryption (SSE-S3 or SSE-KMS)
- **In Memory:** Secrets cleared after use
- **Logging:** Sensitive values never logged

### Input Validation

- Path traversal prevention (`..`, null bytes, `//`)
- YAML injection prevention
- SQL injection N/A (no database)
- Command injection prevention (no shell execution)

## Performance Characteristics

### Scalability

| Metric | Target | Current |
|--------|--------|---------|
| Secrets per sync | 10,000+ | ✅ Tested |
| Vault mounts | 100+ | ✅ Supported |
| AWS accounts | 100+ | ⏳ v1.2.0 |
| Pipeline duration | < 5 min (1000 secrets) | ✅ Achieved |
| Memory usage | < 500 MB | ✅ Typical |

### Optimization Techniques

1. **Caching:**
   - TTL-based caching for AWS ListSecrets (reduces API calls by 90%)
   - Vault response caching (planned)

2. **Parallelization:**
   - Bounded concurrency for secret fetching
   - Worker pool pattern (10 workers default)

3. **Efficient Traversal:**
   - BFS instead of DFS (prevents stack overflow)
   - Cycle detection (prevents infinite loops)
   - Early termination on max_secrets

4. **Resource Management:**
   - Connection pooling for AWS SDK
   - Proper cleanup with defer
   - Context cancellation support

## Error Handling Strategy

### Error Categories

1. **Configuration Errors:**
   - Invalid YAML syntax
   - Missing required fields
   - Invalid references
   - **Action:** Fail fast with clear message

2. **Authentication Errors:**
   - Invalid Vault credentials
   - AWS permission denied
   - Expired tokens
   - **Action:** Fail with authentication instructions

3. **Transient Errors:**
   - Network timeouts
   - Rate limiting
   - Temporary service unavailability
   - **Action:** Retry with exponential backoff

4. **Data Errors:**
   - Invalid secret format
   - Merge conflicts
   - Circular dependencies
   - **Action:** Log warning, continue or fail based on severity

### Error Context (v1.1.0 - Requirement 17)

**Status:** ❓ Needs verification of adoption throughout codebase

**Purpose:** Provide detailed error context for debugging

**Context Fields:**
- Request ID (for correlation)
- Operation name
- Resource path
- Duration
- Retry count

**Example:**
```
[req=abc123] failed to list secrets at path "secret/data/app" after 1250ms (retries: 2): permission denied
```

**Implementation:**
```go
type ErrorContext struct {
    RequestID  string
    Operation  string
    Path       string
    Duration   time.Duration
    Retries    int
}

func (ec *ErrorContext) Wrap(err error) error {
    return fmt.Errorf("[req=%s] %s at path %q after %dms (retries: %d): %w",
        ec.RequestID, ec.Operation, ec.Path, ec.Duration.Milliseconds(), ec.Retries, err)
}
```

**Location:** `pkg/context/error_context.go` + `pkg/context/request_context.go`

**Action Required:** Verify adoption in Vault and AWS clients

### Circuit Breaker (v1.1.0 - Requirement 16)

**Status:** ⚠️ Implemented but has lint error (staticcheck QF1003)

**Purpose:** Prevents cascade failures when external services are degraded

**Behavior:**
- Opens after 5 failures in 10 seconds
- Fails fast when open (30 second timeout)
- Half-open state allows test request
- Independent circuits per service (Vault, AWS)

**Implementation:**
```go
type CircuitBreaker struct {
    state          State  // closed, open, half_open
    failureCount   int
    lastFailure    time.Time
    timeout        time.Duration
    threshold      int
}

func (cb *CircuitBreaker) Execute(ctx context.Context, fn func() error) error
```

**Metrics:**
- `secretsync_circuit_breaker_state{service="vault|aws",state="closed|open|half_open"}`

**Known Issue:**
- Line 142: staticcheck QF1003 - switch statement pattern needs fixing

## Testing Strategy

### Unit Tests

**Coverage Target:** 80%+

**Approach:**
- Table-driven tests
- Mock external dependencies
- Test edge cases explicitly

**Example:**
```go
func TestDeepMerge_ListAppend(t *testing.T) {
    tests := []struct {
        name string
        base map[string]interface{}
        overlay map[string]interface{}
        want map[string]interface{}
    }{
        // Test cases...
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test logic...
        })
    }
}
```

### Integration Tests

**Environment:** Docker Compose (Vault + LocalStack)

**Workflows Tested:**
- Full pipeline (Vault → S3 → AWS)
- Vault recursive listing
- Target inheritance
- Discovery integration

**Location:** `tests/integration/`

### Race Detection

All tests run with `-race` flag in CI

**Protected Resources:**
- `accountSecretArns` map (sync.RWMutex)
- Cache structures
- Shared configuration

### Security Testing

- SAST with gosec
- Dependency scanning with Dependabot
- Container scanning with Trivy
- Manual security reviews

## Observability (v1.1.0)

### Metrics

**Prometheus Metrics Exposed:**

```
# Vault metrics
secretsync_vault_request_duration_seconds{operation="list|read"}
secretsync_vault_requests_total{operation="list|read"}
secretsync_vault_errors_total{operation="list|read"}
secretsync_vault_secrets_total

# AWS metrics
secretsync_aws_request_duration_seconds{service="secretsmanager|s3",operation="list|get|put"}
secretsync_aws_requests_total{service="secretsmanager|s3",operation="list|get|put"}
secretsync_aws_errors_total{service="secretsmanager|s3"}
secretsync_aws_pagination_calls_total

# Pipeline metrics
secretsync_pipeline_duration_seconds{phase="merge|sync"}
secretsync_secrets_synced_total{target="name"}

# Circuit breaker metrics
secretsync_circuit_breaker_state{service="vault|aws",state="closed|open|half_open"}
```

### Logging

**Structured Logging with Logrus:**

```go
log.WithFields(logrus.Fields{
    "request_id": "abc123",
    "operation": "vault.list",
    "path": "secret/data/app",
    "duration_ms": 150,
}).Info("secrets listed successfully")
```

**Log Levels:**
- ERROR: Actionable errors
- WARN: Degraded state
- INFO: Normal operations
- DEBUG: Detailed troubleshooting

### Tracing (Future - v1.3.0)

OpenTelemetry integration planned for distributed tracing

## Current Status Summary

### v1.0 - ✅ COMPLETE
- All core features implemented
- 113+ test functions passing
- Integration test infrastructure
- Race detector clean
- Production-ready

### v1.1.0 - ⚠️ PARTIAL (Release Blocked)

**Completed Features:**
- ✅ Prometheus metrics endpoint (PR #69, Issue #46)
- ✅ Docker image version pinning (PR #64, Issue #40)
- ✅ Race condition prevention (PR #68, Issue #44)

**Blocked by Lint Errors:**
- 🔴 Circuit breaker staticcheck error (PR #70, Issue #47)
- 🔴 VaultClient copylocks error

**Needs Verification:**
- ❓ Error context adoption (PR #71, Issue #48)
- ❓ Queue compaction config (PR #67, Issue #43)

**Pending Work:**
- ⏳ Integration tests in CI (Issue #51, #52)
- ⏳ Documentation fixes (Issue #50)

**Estimated Time to Release:** 8-10 hours

### v1.2.0 - ⏳ PLANNED

**Infrastructure Complete:**
- ✅ Vault recursive listing
- ✅ Deep merge
- ✅ Target inheritance
- ✅ S3 merge store

**Planned Features:**
- ⏳ AWS Organizations discovery enhancements
- ⏳ AWS Identity Center integration
- ⏳ Secret versioning
- ⏳ Enhanced diff output

## Roadmap

### v1.1.0 - Observability & Reliability (Current - ⚠️ PARTIAL)

**Status:** Most features complete, 2 critical lint errors blocking release

**Completed:**
- [x] Prometheus metrics endpoint (Requirement 15)
- [x] Docker image version pinning (Requirement 18)
- [x] Race condition prevention (Requirement 20)

**In Progress:**
- [⚠️] Circuit breaker pattern (Requirement 16) - Has staticcheck QF1003 lint error
- [❓] Enhanced error messages with request IDs (Requirement 17) - Needs verification
- [❓] Configurable queue compaction (Requirement 19) - Needs verification

**Pending:**
- [ ] CI/CD modernization (Requirement 21) - Integration tests not in CI
- [ ] Fix lint errors to unblock release

### v1.2.0 - Advanced Features (⏳ PLANNED)

**Completed Infrastructure:**
- [x] Vault recursive listing (Requirement 2)
- [x] Deep merge compatibility (Requirement 9)
- [x] Target inheritance (Requirement 11)
- [x] S3 merge store (Requirement 8)

**Planned Enhancements:**
- [ ] AWS Organizations discovery enhancements (Requirement 22)
  - Comprehensive tag filtering
  - OU-based filtering
  - Account status filtering
  - Discovery caching
- [ ] AWS Identity Center integration (Requirement 23)
  - Permission set discovery
  - Account assignment mapping
- [ ] Secret versioning support (Requirement 24)
  - Version tracking in diff engine
  - Version metadata in S3
  - Version rollback capability
- [ ] Enhanced diff output (Requirement 25)
  - Side-by-side comparison
  - Value masking
  - GitHub output format
  - JSON output format
  - Summary statistics

### v1.3.0 - Enterprise Scale (Future)

- [ ] Distributed tracing with OpenTelemetry
- [ ] Secret rotation automation
- [ ] Multi-region replication
- [ ] Webhook notifications
- [ ] Policy-as-code validation
- [ ] Audit log export
- [ ] Performance optimizations for 100k+ secrets

### v2.0.0 - Multi-Cloud (Future)

- [ ] Google Cloud Secret Manager support
- [ ] Azure Key Vault support
- [ ] Generic webhook targets
- [ ] Plugin system for custom targets
- [ ] Advanced secret transformations
- [ ] Encryption key rotation

## Technical Decisions

### Why Go 1.25+?

- Latest stable release with modern features
- Excellent concurrency primitives
- Strong standard library
- Fast compilation and execution
- Great AWS SDK v2 support

### Why Two-Phase Pipeline?

**Merge Phase Benefits:**
- Configuration reuse via inheritance
- Audit trail in S3
- Decoupling from target sync

**Sync Phase Benefits:**
- Independent execution
- Easier rollback
- Incremental updates

### Why S3 for Merge Store?

- Durable, versioned storage
- Native AWS integration
- Cost-effective
- S3 Event notifications for automation

### Why Not Kubernetes Operator?

**Previous Architecture:** Kubernetes operator with CRDs

**Issues:**
- Over-engineered for use case
- Added ~13k lines of boilerplate
- Kubernetes-specific deployment
- Harder to test

**Current Architecture:** Simple CLI + GitHub Action
- Runs anywhere
- Easy to test
- Clear execution model
- Can still run in Kubernetes as CronJob

### Why BFS for Vault Traversal?

- Prevents stack overflow on deep hierarchies
- Easier to implement cycle detection
- More predictable memory usage
- Better for large secret trees

### Why Prometheus for Metrics? (v1.1.0)

**Rationale:**
- Industry standard for cloud-native monitoring
- Pull-based model (no external dependencies)
- Rich ecosystem (Grafana, AlertManager)
- Native Kubernetes integration
- Simple HTTP endpoint

**Alternatives Considered:**
- StatsD: Push-based, requires aggregation server
- CloudWatch: AWS-specific, vendor lock-in
- Custom logging: No standardization, harder to query

### Why Circuit Breaker Pattern? (v1.1.0)

**Problem:** Cascade failures when Vault or AWS are degraded

**Solution:** Fail fast and recover gracefully

**Benefits:**
- Prevents resource exhaustion
- Reduces latency during outages
- Automatic recovery testing
- Independent circuits per service

**Implementation Choice:**
- Custom implementation (not library)
- Simpler, fewer dependencies
- Tailored to SecretSync needs
- ~100 lines of code

## Immediate Actions Required (v1.1.0 Release Blockers)

### Critical Issues

**1. Fix Lint Errors (Estimated: 1-2 hours)**

**Issue 1: Circuit Breaker - staticcheck QF1003**
- **File:** `pkg/circuitbreaker/circuitbreaker.go:142`
- **Problem:** Switch statement pattern issue
- **Impact:** Blocks CI/CD pipeline
- **Priority:** P0 - CRITICAL

**Issue 2: VaultClient DeepCopy - copylocks**
- **File:** `pkg/client/vault/vault.go`
- **Problem:** Copying sync.Mutex in DeepCopy method
- **Impact:** Blocks CI/CD pipeline
- **Priority:** P0 - CRITICAL

**2. Verify v1.1.0 Feature Integration (Estimated: 2-3 hours)**

- [ ] Verify metrics endpoint works end-to-end (Requirement 15)
- [ ] Verify circuit breaker integration in clients (Requirement 16)
- [ ] Verify error context adoption in codebase (Requirement 17)
- [ ] Verify queue compaction configuration (Requirement 19)

**3. Add Integration Tests to CI (Estimated: 2-3 hours)**

- [ ] Create integration test job in CI workflow
- [ ] Add docker-compose step
- [ ] Run `tests/integration/` suite
- [ ] Make it required check
- [ ] Document integration test setup

**Total Estimated Time to Clean v1.1.0:** 8-10 hours

### Non-Functional Requirements

**Performance Targets:**
- Pipeline SHALL complete within 5 minutes for 1,000 secrets
- Vault listing SHALL process 100 directories/second minimum
- AWS Secrets Manager sync SHALL process 50 secrets/second minimum
- Memory usage SHALL not exceed 500MB for typical workloads
- API response time p95 SHALL be < 500ms

**Reliability Targets:**
- Pipeline SHALL succeed 99.9% of the time when services are healthy
- Transient failures SHALL be retried automatically
- Circuit breaker SHALL prevent cascade failures
- State SHALL be consistent (all or nothing for targets)
- Concurrent executions SHALL not interfere with each other

**Security Requirements:**
- Credentials SHALL never be logged
- All external connections SHALL use TLS
- Secrets SHALL never be written to disk unencrypted
- Path traversal attacks SHALL be prevented
- Input validation SHALL prevent injection attacks
- Least privilege principle SHALL be followed for IAM policies

**Maintainability Requirements:**
- Code coverage SHALL be ≥ 80%
- All public APIs SHALL have documentation comments
- Complex logic SHALL have inline comments explaining why
- Git commits SHALL follow Conventional Commits format
- Breaking changes SHALL be documented in CHANGELOG.md

**Usability Requirements:**
- Error messages SHALL be clear and actionable
- `--help` flag SHALL provide complete usage information
- Common operations SHALL be achievable with single command
- Configuration SHALL be validated before execution
- Progress indicators SHALL show long-running operations

## Glossary

- **Vault Source:** Configuration defining Vault mount to read from
- **Target:** External secret store to sync to (e.g., AWS Secrets Manager)
- **Merge Store:** S3 bucket storing merged secret configurations
- **Inheritance:** Target importing configuration from another target
- **Deep Merge:** Recursive merging strategy for complex data structures
- **Pipeline:** Two-phase process (merge + sync)
- **Discovery:** Automatic detection of AWS resources
- **Circuit Breaker:** Pattern to prevent cascade failures
- **BFS:** Breadth-First Search traversal algorithm
- **TTL:** Time-To-Live for cache expiration
- **IRSA:** IAM Roles for Service Accounts (Kubernetes AWS auth)
- **OIDC:** OpenID Connect (GitHub Actions AWS auth)

---

**Document Version:** 2.0  
**Last Updated:** 2025-12-09  
**Status:** Consolidated design (v1.0-v1.2.0)

