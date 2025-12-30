# 🚀 Announcing SecretSync v1.2.0 - Enterprise-Grade Secret Synchronization

We're excited to announce the public release of **SecretSync v1.2.0**, an enterprise-grade secret synchronization pipeline that transforms how organizations manage secrets across HashiCorp Vault and AWS Secrets Manager.

## 🎯 What is SecretSync?

SecretSync provides **fully automated, enterprise-grade secret synchronization** with a unique **two-phase pipeline architecture** (merge → sync) that supports inheritance, dynamic target discovery, and CI/CD-friendly diff reporting.

**Perfect for:** Multi-account AWS environments, Kubernetes deployments, CI/CD pipelines, and enterprise secret management at scale.

## 🤔 Why SecretSync?

| Feature | SecretSync | Alternatives |
|---------|------------|--------------|
| **Two-Phase Pipeline** | ✅ Merge → Sync with inheritance | ❌ Simple 1:1 sync only |
| **AWS Organizations** | ✅ Dynamic discovery with tag filtering | ❌ Manual account management |
| **Secret Versioning** | ✅ Complete audit trail with rollback | ❌ No version tracking |
| **Enhanced Diff** | ✅ Side-by-side with intelligent masking | ❌ Basic text diff |
| **Enterprise Scale** | ✅ 1000+ accounts, circuit breakers | ❌ Limited scalability |
| **CI/CD Integration** | ✅ GitHub Action + exit codes | ❌ Manual scripting required |

## ✨ Key Features in v1.2.0

### 🔍 Enhanced AWS Organizations Discovery
- **Advanced Tag Filtering**: Multiple tag filters with wildcard support (`Environment=prod*`, `Team=platform-?`)
- **Flexible Logic**: Configure AND/OR combinations for complex filtering scenarios
- **OU-Based Discovery**: Target specific organizational units with nested traversal
- **Smart Filtering**: Automatically exclude suspended and closed accounts
- **Performance Caching**: In-memory cache with 1-hour TTL for faster subsequent discoveries

### 🔐 AWS Identity Center Integration
- **Permission Set Discovery**: Automatic mapping of permission sets to readable names
- **Account Assignments**: Track which users/groups have access to which accounts
- **Cross-Region Support**: Automatically discover Identity Center instances across regions
- **Assignment Caching**: 30-minute TTL for assignment data to optimize performance

### 📚 Secret Versioning System
- **Version Tracking**: Every secret change tracked with metadata
- **S3-Based Storage**: Reliable, scalable version history storage
- **Retention Policies**: Configurable cleanup of old versions
- **Version Rollback**: CLI support for rolling back to specific versions
- **Diff Integration**: See version transitions in diff output (v1 → v2)

### 🎨 Enhanced Diff Output
- **Side-by-Side Comparison**: Visual comparison with aligned columns and color coding
- **Intelligent Masking**: Automatically detect and mask sensitive values (API keys, passwords, tokens)
- **Multiple Formats**: Human-readable, JSON, GitHub Actions annotations, compact
- **Rich Statistics**: Detailed counts of changes, size differences, and execution timing

### 🛡️ Enterprise Reliability (v1.1.0)
- **Circuit Breakers**: Automatic failure detection and recovery for Vault and AWS
- **Prometheus Metrics**: Production-ready observability with `/metrics` endpoint
- **Enhanced Error Context**: Request ID tracking and structured error messages
- **Race Condition Prevention**: Thread-safe operations with comprehensive testing

## 🚀 Quick Start

### Installation
```bash
# Go install
go install github.com/extended-data-library/secretssync/cmd/secretsync@latest

# Docker
docker pull extended-data-library/secretssync:latest

# GitHub Action
- uses: extended-data-library/secretssync@v1
```

### Basic Usage
```bash
# Dry run with enhanced diff output
secretsync pipeline --config config.yaml --dry-run --format side-by-side

# Full pipeline execution with metrics
secretsync pipeline --config config.yaml --metrics-port 9090

# CI/CD mode with exit codes
secretsync pipeline --config config.yaml --dry-run --exit-code
```

### Example Configuration
```yaml
vault:
  address: "https://vault.example.com"
  auth:
    approle:
      role_id: "${VAULT_ROLE_ID}"
      secret_id: "${VAULT_SECRET_ID}"

# Advanced discovery (v1.2.0)
discovery:
  aws_organizations:
    enabled: true
    tag_filters:
      - key: "Environment"
        values: ["production", "staging"]
        operator: "equals"
    cache_ttl: "1h"

# Secret versioning (v1.2.0)
versioning:
  enabled: true
  s3_bucket: "company-secretsync-versions"
  retention_days: 90

sources:
  app-secrets:
    vault:
      path: "secret/data/myapp"

targets:
  production:
    aws_secretsmanager:
      region: "us-east-1"
    imports:
      - app-secrets
```

## 📊 Production Ready

- **150+ Test Functions**: Comprehensive unit and integration test coverage
- **Zero Critical Bugs**: Battle-tested in production environments
- **Full CI/CD Pipeline**: Automated testing, building, and releasing
- **Multi-Platform Support**: Docker images for amd64/arm64
- **Professional Documentation**: Complete guides from getting started to advanced topics

## 🌟 Community & Support

### Getting Started
- **📚 [Documentation](https://github.com/extended-data-library/secretssync/tree/main/docs)**: Comprehensive guides and examples
- **🚀 [Getting Started Guide](https://github.com/extended-data-library/secretssync/blob/main/docs/GETTING_STARTED.md)**: Step-by-step tutorial
- **❓ [FAQ](https://github.com/extended-data-library/secretssync/blob/main/docs/FAQ.md)**: Common questions answered
- **📋 [Examples](https://github.com/extended-data-library/secretssync/tree/main/examples)**: Real-world configurations

### Community
- **💬 [GitHub Discussions](https://github.com/extended-data-library/secretssync/discussions)**: Community Q&A and feature discussions
- **🐛 [Issues](https://github.com/extended-data-library/secretssync/issues)**: Bug reports and feature requests
- **🤝 [Contributing](https://github.com/extended-data-library/secretssync/blob/main/CONTRIBUTING.md)**: How to contribute
- **🗺️ [Roadmap](https://github.com/extended-data-library/secretssync/blob/main/docs/ROADMAP.md)**: Future development plans

## 🙏 Attribution

SecretSync originated as a fork of [vault-secret-sync](https://github.com/robertlestak/vault-secret-sync) by **Robert Lestak**. We thank Robert for creating the original foundation.

SecretSync has evolved into an independent project with its own architecture, features, and roadmap, while maintaining the same MIT license and open-source spirit.

## 🔗 Links

- **🌟 [GitHub Repository](https://github.com/extended-data-library/secretssync)** - Star us!
- **🐳 [Docker Hub](https://hub.docker.com/r/extended-data-library/secretssync)** - Pull the latest image
- **📦 [GitHub Releases](https://github.com/extended-data-library/secretssync/releases)** - Download binaries
- **🎯 [GitHub Action](https://github.com/marketplace/actions/secretsync)** - Use in your workflows

---

**Ready to transform your secret management? Give SecretSync a try and let us know what you think!**

⭐ **[Star SecretSync on GitHub](https://github.com/extended-data-library/secretssync)** ⭐

---

## Social Media Posts

### Twitter/X
```
🚀 Excited to announce SecretSync v1.2.0! 

Enterprise-grade secret sync from HashiCorp Vault to AWS Secrets Manager with:
✅ Two-phase pipeline architecture
✅ AWS Organizations discovery
✅ Secret versioning & rollback
✅ Enhanced diff with masking
✅ Native GitHub Action

⭐ https://github.com/extended-data-library/secretssync

#DevOps #SecretManagement #Vault #AWS #OpenSource
```

### LinkedIn
```
🎉 Proud to announce the public release of SecretSync v1.2.0!

After months of development, we've created an enterprise-grade secret synchronization pipeline that transforms how organizations manage secrets across HashiCorp Vault and AWS Secrets Manager.

🔥 What makes SecretSync unique:
• Two-phase pipeline architecture with inheritance
• Dynamic AWS Organizations discovery with tag filtering
• Complete secret versioning with audit trails
• Enhanced diff output with intelligent value masking
• Production-ready with 150+ tests and comprehensive CI/CD

Perfect for DevOps teams managing multi-account AWS environments, Kubernetes deployments, and enterprise-scale secret management.

🚀 Ready to try it? Check out our comprehensive documentation and examples:
https://github.com/extended-data-library/secretssync

#DevOps #SecretManagement #HashiCorpVault #AWS #OpenSource #Enterprise #Security
```

### Reddit (r/devops)
```
Title: [Open Source] SecretSync v1.2.0 - Enterprise-grade secret sync from Vault to AWS Secrets Manager

Hey r/devops! 

I'm excited to share SecretSync v1.2.0, an open-source secret synchronization pipeline I've been working on. It's designed specifically for enterprise environments that need to sync secrets from HashiCorp Vault to AWS Secrets Manager at scale.

**What makes it different:**
- **Two-phase architecture**: Merge secrets from multiple sources, then sync to targets with inheritance
- **AWS Organizations integration**: Automatically discover accounts with tag filtering and wildcards
- **Secret versioning**: Complete audit trail with rollback capability
- **Enhanced diff**: Side-by-side comparison with intelligent masking of sensitive values
- **Production ready**: 150+ tests, circuit breakers, Prometheus metrics

**Perfect for:**
- Multi-account AWS environments (we handle 1000+ accounts)
- Kubernetes deployments with complex secret hierarchies
- CI/CD pipelines (includes native GitHub Action)
- Organizations migrating from Vault to AWS Secrets Manager

**Quick example:**
```yaml
discovery:
  aws_organizations:
    tag_filters:
      - key: "Environment"
        values: ["prod*", "staging"]
        operator: "contains"
    cache_ttl: "1h"

versioning:
  enabled: true
  s3_bucket: "my-secretsync-versions"
  retention_days: 90
```

The project originated as a fork of vault-secret-sync but has evolved into something much more comprehensive. It's MIT licensed and ready for production use.

**Links:**
- GitHub: https://github.com/extended-data-library/secretssync
- Documentation: https://github.com/extended-data-library/secretssync/tree/main/docs
- Getting Started: https://github.com/extended-data-library/secretssync/blob/main/docs/GETTING_STARTED.md

Would love to hear your thoughts and feedback! Happy to answer any questions about the architecture or use cases.
```

### Hacker News
```
Title: SecretSync v1.2.0 – Enterprise secret synchronization from Vault to AWS Secrets Manager

SecretSync is an open-source secret synchronization pipeline designed for enterprise environments. It provides automated sync from HashiCorp Vault to AWS Secrets Manager with a unique two-phase architecture that supports inheritance and complex organizational structures.

Key features:
- Two-phase pipeline (merge → sync) with configuration inheritance
- AWS Organizations discovery with tag filtering and wildcards  
- Secret versioning with complete audit trails and rollback
- Enhanced diff output with intelligent value masking
- Production-ready with circuit breakers and Prometheus metrics
- Native GitHub Action for CI/CD integration

The project handles enterprise scale (1000+ AWS accounts) and includes comprehensive testing (150+ test functions). It originated as a fork of vault-secret-sync but has evolved significantly with its own architecture and feature set.

Perfect for organizations managing secrets across multiple AWS accounts, Kubernetes environments, or those migrating from Vault to AWS Secrets Manager.

GitHub: https://github.com/extended-data-library/secretssync
Documentation: https://github.com/extended-data-library/secretssync/tree/main/docs

MIT licensed and ready for production use.
```