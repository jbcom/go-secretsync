# SecretSync v1.2.0 Release Notes

## 🎉 Major Release: Advanced Enterprise Features

SecretSync v1.2.0 represents a significant advancement in enterprise secret management capabilities, building on the solid foundation of v1.1.0's observability features with powerful new discovery, versioning, and user experience enhancements.

## 🚀 What's New in v1.2.0

### 🔍 Enhanced AWS Organizations Discovery

Transform how you manage secrets across large AWS Organizations:

- **Advanced Tag Filtering**: Multiple tag filters with wildcard support (`Environment=prod*`, `Team=platform-?`)
- **Flexible Logic**: Configure AND/OR combinations for complex filtering scenarios
- **OU-Based Discovery**: Target specific organizational units with nested traversal
- **Smart Filtering**: Automatically exclude suspended and closed accounts
- **Performance Caching**: In-memory cache with 1-hour TTL for faster subsequent discoveries

```yaml
discovery:
  aws_organizations:
    tag_filters:
      - key: "Environment"
        values: ["prod*", "staging"]
        operator: "contains"
    organizational_units:
      - "ou-12345678"
      - "ou-87654321"
    tag_logic: "AND"
    cache_ttl: "1h"
```

### 🔐 AWS Identity Center Integration

Seamlessly integrate with AWS Identity Center (SSO) for advanced permission management:

- **Permission Set Discovery**: Automatic mapping of permission sets to readable names
- **Account Assignments**: Track which users/groups have access to which accounts
- **Cross-Region Support**: Automatically discover Identity Center instances across regions
- **Assignment Caching**: 30-minute TTL for assignment data to optimize performance

```yaml
discovery:
  identity_center:
    instance_arn: "arn:aws:sso:::instance/ssoins-1234567890abcdef"
    identity_store_id: "d-1234567890"
    region: "us-east-1"
```

### 📚 Secret Versioning System

Enterprise-grade secret versioning with full audit trail:

- **Version Tracking**: Every secret change tracked with metadata
- **S3-Based Storage**: Reliable, scalable version history storage
- **Retention Policies**: Configurable cleanup of old versions
- **Version Rollback**: CLI support for rolling back to specific versions
- **Diff Integration**: See version transitions in diff output (v1 → v2)

```bash
# View version history
secretsync versions --secret-path "app/database/password"

# Rollback to specific version
secretsync sync --version 5 --target production

# Configure retention
secretsync config set versioning.retention_days 90
```

### 🎨 Enhanced Diff Output

Professional-grade diff visualization with security-first design:

- **Side-by-Side Comparison**: Visual comparison with aligned columns and color coding
- **Intelligent Masking**: Automatically detect and mask sensitive values (API keys, passwords, tokens)
- **Multiple Formats**: Human-readable, JSON, GitHub Actions annotations, compact
- **Rich Statistics**: Detailed counts of changes, size differences, and execution timing

```bash
# Side-by-side comparison with masking
secretsync diff --format side-by-side

# Show actual values (use with caution)
secretsync diff --format side-by-side --show-values

# JSON output for automation
secretsync diff --format json | jq '.changes[] | select(.type == "modified")'
```

## 🛡️ Security & Reliability Improvements

### From v1.1.0 (Now Stable)
- **Circuit Breakers**: Automatic failure detection and recovery for Vault and AWS
- **Prometheus Metrics**: Production-ready observability with `/metrics` endpoint
- **Enhanced Error Context**: Request ID tracking and structured error messages
- **Race Condition Prevention**: Thread-safe operations with comprehensive testing

### New in v1.2.0
- **Value Masking**: Intelligent detection of sensitive patterns in diff output
- **Audit Trail**: Complete version history for compliance requirements
- **Secure Defaults**: Sensitive values masked by default, explicit flag required to show

## 📊 Performance & Scale

- **150+ Test Functions**: Comprehensive test coverage including integration tests
- **Caching Optimizations**: Multi-level caching for AWS Organizations and Identity Center
- **Concurrent Processing**: Thread-safe operations with race condition testing
- **Memory Efficiency**: Optimized data structures for large-scale deployments

## 🔧 Configuration Examples

### Complete AWS Organizations Setup
```yaml
sources:
  - name: "vault-prod"
    type: "vault"
    config:
      address: "https://vault.company.com"
      path: "secret/data/applications"

discovery:
  aws_organizations:
    enabled: true
    tag_filters:
      - key: "Environment"
        values: ["production", "staging"]
        operator: "equals"
      - key: "Team"
        values: ["platform*"]
        operator: "contains"
    organizational_units:
      - "ou-production-12345"
    account_status_filter: ["ACTIVE"]
    tag_logic: "AND"
    cache_ttl: "1h"

  identity_center:
    enabled: true
    region: "us-east-1"
    cache_ttl: "30m"

versioning:
  enabled: true
  s3_bucket: "company-secretsync-versions"
  retention_days: 90

observability:
  metrics:
    enabled: true
    port: 9090
    address: "0.0.0.0"
```

### Enhanced Diff Configuration
```yaml
diff:
  format: "side-by-side"
  show_values: false  # Mask sensitive values by default
  include_metadata: true
  color_output: true
```

## 🚀 Migration Guide

### From v1.1.x
No breaking changes! All existing configurations continue to work. New features are opt-in.

### From v1.0.x
1. Update your configuration to use new discovery options (optional)
2. Enable versioning if desired (optional)
3. Configure enhanced diff output (optional)

## 🐛 Bug Fixes

- Fixed critical lint errors that were blocking CI/CD
- Resolved race conditions in concurrent map access
- Improved error handling in AWS Organizations discovery
- Enhanced memory management in large-scale deployments

## 📈 What's Next (v1.3.0 Preview)

- OpenTelemetry distributed tracing integration
- Additional secret store integrations (Azure Key Vault, GCP Secret Manager)
- Advanced RBAC and audit logging
- Kubernetes operator enhancements

## 🙏 Acknowledgments

This release represents months of focused development on enterprise-grade features. Special thanks to the community for feedback and testing.

## 📦 Installation

### Docker
```bash
docker pull extended-data-library/secretssync:v1.2.0
```

### Helm
```bash
helm upgrade --install secretsync oci://registry-1.docker.io/extended-data-library/secretssync --version 1.2.0
```

### GitHub Action
```yaml
- uses: extended-data-library/secretssync@v1.2.0
  with:
    config-file: .secretsync.yaml
    dry-run: false
```

### Binary Releases
Download from [GitHub Releases](https://github.com/extended-data-library/secretssync/releases/tag/v1.2.0)

---

**Full Changelog**: https://github.com/extended-data-library/secretssync/compare/v1.1.0...v1.2.0

**Documentation**: https://github.com/extended-data-library/secretssync/blob/main/README.md

**Issues**: https://github.com/extended-data-library/secretssync/issues