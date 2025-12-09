# Changelog

All notable changes to SecretSync will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- **SecretSync 1.0 Release** - Complete rebranding and architecture overhaul
- Recursive Vault KV2 listing with BFS traversal
- Target inheritance with circular dependency detection
- Deepmerge support (list append, dict merge, scalar override)
- AWS Organizations dynamic account discovery
- Fuzzy name matching for AWS accounts
- S3 merge store support
- TTL-based caching for AWS ListSecrets
- Enhanced path validation (directory traversal, null byte injection protection)
- LogicalClient interface for testability
- Comprehensive test suite (113+ test functions)

### Changed
- **PROJECT RENAME**: vault-secret-sync → SecretSync
- CLI renamed from `vss` to `secretsync`
- Docker images published to `docker.io/jbcom/secretsync`
- Helm charts published to `oci://registry-1.docker.io/jbcom/secretsync`
- Simplified pipeline architecture (removed legacy operator complexity)
- Environment variable prefix changed from `VSS_` to `SECRETSYNC_`

### Removed
- Legacy Kubernetes operator architecture (~13k lines)
- Backend packages (kube, file)
- Queue packages (redis, nats, sqs, memory)
- Notification packages (webhook, slack, email)
- GCP/GitHub/Doppler/HTTP store implementations
- Event processing system

### Security
- Race condition fixes in AWS client with mutex protection
- Cache invalidation on writes to prevent stale data
- Path traversal attack prevention
- Type-safe Vault API response parsing
- Safe type assertions throughout codebase

### Fixed
- Cache invalidation after WriteSecret and DeleteSecret operations
- Structured logging consistency
- Error context in vault traversal with depth and count info

---

## Ownership & Attribution

### Current Maintainer
- **Organization**: jbcom
- **Repository**: [jbcom/secretsync](https://github.com/jbcom/secretsync)

### Original Source
- **Author**: Robert Lestak
- **Repository**: [robertlestak/vault-secret-sync](https://github.com/robertlestak/vault-secret-sync)
- **License**: MIT

### Fork Rationale

This project is a complete rebranding and reimplementation of vault-secret-sync,
focused on providing a streamlined, pipeline-based secret synchronization tool.

**Key differences from upstream:**
- Pipeline-driven architecture instead of Kubernetes operator
- Support for dynamic AWS Organizations discovery
- Enhanced merge strategies matching Python's deepmerge
- Simplified configuration and deployment
- GitHub Marketplace Action support

### License

MIT License - see [LICENSE](LICENSE) for details.

Original work Copyright (c) Robert Lestak
Modified work Copyright (c) 2025 jbcom
