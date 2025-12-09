# SecretSync

> **Universal Secrets Synchronization Pipeline**

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![GitHub release](https://img.shields.io/github/release/jbcom/secretsync.svg)](https://github.com/jbcom/secretsync/releases)
[![Docker Image](https://img.shields.io/badge/docker-jbcom%2Fsecretsync-blue)](https://hub.docker.com/r/jbcom/secretsync)
[![Go Report Card](https://goreportcard.com/badge/github.com/jbcom/secretsync)](https://goreportcard.com/report/github.com/jbcom/secretsync)

SecretSync provides fully automated, real-time secret synchronization across multiple cloud providers and secret stores. It supports a two-phase pipeline architecture (merge → sync) with inheritance, dynamic target discovery, and CI/CD-friendly diff reporting.

## Attribution

SecretSync originated as a fork of [robertlestak/vault-secret-sync](https://github.com/robertlestak/vault-secret-sync) (MIT License). We thank **Robert Lestak** for creating the original codebase.

**SecretSync is an independent product** with its own roadmap and development direction. It has been substantially rewritten with:
- Two-phase pipeline architecture (merge → sync)
- S3 merge store support  
- Dynamic target discovery (AWS Organizations, Identity Center)
- Comprehensive diff/dry-run system with CI/CD integration
- DeepMerge semantics for secret aggregation
- Kubernetes operator with CRD support

## Supported Secret Stores

| Store | Source | Target | Merge Store |
|-------|--------|--------|-------------|
| HashiCorp Vault (KV2) | ✅ | ✅ | ✅ |
| AWS Secrets Manager | ✅ | ✅ | ❌ |
| AWS S3 | ❌ | ❌ | ✅ |
| AWS Identity Center | Discovery | ❌ | ❌ |

## Two-Phase Pipeline Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                    MERGE PHASE (Optional)                        │
│  Source1 ──┐                                                     │
│  Source2 ──┼──▶ Merge Store (Vault/S3) ──▶ Aggregated Secrets   │
│  Source3 ──┘    (deepmerge, inheritance)                         │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                        SYNC PHASE                                │
│  Merge Store ──┬──▶ AWS Account 1 (via STS AssumeRole)          │
│  (or Source)   ├──▶ AWS Account 2                                │
│                ├──▶ Vault Cluster                                │
│                └──▶ GCP Project                                  │
└─────────────────────────────────────────────────────────────────┘
```

See [Two-Phase Architecture](./docs/TWO_PHASE_ARCHITECTURE.md) for detailed documentation.

## Quick Start

### Installation

```bash
# Go install
go install github.com/jbcom/secretsync/cmd/secretsync@latest

# Or download binary from releases
curl -LO https://github.com/jbcom/secretsync/releases/latest/download/secretsync-linux-amd64
chmod +x secretsync-linux-amd64
sudo mv secretsync-linux-amd64 /usr/local/bin/secretsync
```

### Basic Usage

```bash
# Validate configuration
secretsync validate --config pipeline.yaml

# Dry run with diff output
secretsync pipeline --config pipeline.yaml --dry-run --output json

# Full pipeline execution
secretsync pipeline --config pipeline.yaml

# CI/CD mode (exit codes: 0=no changes, 1=changes, 2=errors)
secretsync pipeline --config pipeline.yaml --dry-run --exit-code
```

### Example Configuration

```yaml
# pipeline.yaml
vault:
  address: "https://vault.example.com"
  namespace: "admin"

aws:
  region: "us-east-1"
  execution_role_pattern: "arn:aws:iam::{account_id}:role/SecretsSync"

merge_store:
  vault:
    mount: "secret/merged"

sources:
  api-keys:
    vault:
      path: "secret/api-keys"
  database:
    vault:
      path: "secret/database"

targets:
  Staging:
    imports: [api-keys, database]
    account_id: "111111111111"
  
  Production:
    inherits: Staging
    imports: [production-overrides]
    account_id: "222222222222"
```

## GitHub Actions

SecretSync is available as a GitHub Action for seamless CI/CD integration:

```yaml
- name: Sync Secrets
  uses: jbcom/secretsync@v1
  with:
    config: config.yaml
    dry-run: 'false'
    output-format: 'github'
  env:
    VAULT_ROLE_ID: ${{ secrets.VAULT_ROLE_ID }}
    VAULT_SECRET_ID: ${{ secrets.VAULT_SECRET_ID }}
```

**Key Features:**
- 🔒 Native OIDC support for AWS authentication
- 📊 GitHub-native diff annotations in PRs
- 🎯 Exit codes for CI/CD control flow
- 🔄 Automatic Docker multi-arch builds
- ⚡ Zero configuration needed beyond config file

**Quick Start:**
1. Add `config.yaml` to your repository
2. Configure AWS OIDC and Vault secrets
3. Use the action in your workflow

See [GitHub Actions documentation](./docs/GITHUB_ACTIONS.md) for complete usage guide and examples.

## CI/CD Integration (CLI)

### GitHub Actions (CLI)

```yaml
- name: Validate secrets pipeline
  run: |
    secretsync pipeline --config pipeline.yaml --dry-run --output github --exit-code
  
- name: Apply secrets (on merge to main)
  if: github.ref == 'refs/heads/main'
  run: |
    secretsync pipeline --config pipeline.yaml
```

### Output Formats

| Format | Use Case |
|--------|----------|
| `human` | Interactive terminal output |
| `json` | Machine parsing, logging |
| `github` | GitHub Actions annotations |
| `compact` | One-line CI status |

## Documentation

- [Architecture Overview](./docs/ARCHITECTURE.md)
- [Two-Phase Pipeline](./docs/TWO_PHASE_ARCHITECTURE.md)
- [Pipeline Configuration](./docs/PIPELINE.md)
- [Deployment Guide](./docs/DEPLOYMENT.md)
- [Security Configuration](./docs/SECURITY.md)
- [Usage Reference](./docs/USAGE.md)

## Helm Deployment

```bash
# Add Helm repo
helm repo add secretsync https://jbcom.github.io/secretsync

# Install
helm install secretsync secretsync/secretsync \
  --set vault.address=https://vault.example.com
```

## Docker

```bash
# Run with config file
docker run -v $(pwd)/config.yaml:/config.yaml \
  jbcom/secretsync pipeline --config /config.yaml

# Multi-arch images available: linux/amd64, linux/arm64
```

## Observability

SecretSync exposes Prometheus metrics for production monitoring and debugging.

### Enabling Metrics

```bash
# Enable metrics server on port 9090
secretsync pipeline --config config.yaml --metrics-port 9090

# Custom address and port
secretsync pipeline --config config.yaml --metrics-addr 0.0.0.0 --metrics-port 9090
```

### Available Metrics

**Vault Metrics:**
- `secretsync_vault_api_call_duration_seconds` - Vault API call latency
- `secretsync_vault_secrets_listed_total` - Total secrets listed from Vault
- `secretsync_vault_traversal_depth` - BFS traversal depth reached
- `secretsync_vault_queue_size` - Current traversal queue size
- `secretsync_vault_errors_total` - Vault error count by operation/type

**AWS Metrics:**
- `secretsync_aws_api_call_duration_seconds` - AWS API call latency
- `secretsync_aws_pagination_pages` - Number of pagination pages processed
- `secretsync_aws_cache_hits_total` - Cache hit count
- `secretsync_aws_cache_misses_total` - Cache miss count
- `secretsync_aws_secrets_operations_total` - Secret operations (create/update/delete)

**Pipeline Metrics:**
- `secretsync_pipeline_execution_duration_seconds` - Pipeline phase duration
- `secretsync_pipeline_targets_processed_total` - Targets processed by phase
- `secretsync_pipeline_parallel_workers` - Active parallel workers
- `secretsync_pipeline_errors_total` - Pipeline error count

**S3 Metrics:**
- `secretsync_s3_operation_duration_seconds` - S3 operation latency
- `secretsync_s3_object_size_bytes` - S3 object sizes

### Prometheus Configuration

```yaml
scrape_configs:
  - job_name: 'secretsync'
    static_configs:
      - targets: ['localhost:9090']
    metrics_path: '/metrics'
```

### Health Check

The metrics server also exposes a `/health` endpoint:

```bash
curl http://localhost:9090/health
# Returns: OK
```

## Development

```bash
# Clone
git clone https://github.com/jbcom/secretsync.git
cd secretsync

# Build
go build ./...

# Test
go test ./...

# Lint
golangci-lint run
```

## License

[MIT License](./LICENSE)

## Original Author

**Robert Lestak** - [github.com/robertlestak](https://github.com/robertlestak)

Original project: [vault-secret-sync](https://github.com/robertlestak/vault-secret-sync)

## Current Maintainer

**jbcom** - [github.com/jbcom](https://github.com/jbcom)
