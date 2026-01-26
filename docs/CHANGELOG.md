# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Planned for v0.3.0 - Emulator Suite Integration
- **Emulator Integration**: Secret Manager + KMS call IAM for authz
- **REST API**: HTTP/JSON gateway for REST clients
- **Hot Reload**: `--watch` flag for config file reloading
- **Additional Permissions**: Complete coverage of emulator operations

## [0.2.0] - 2026-01-26

### Added - Drop-in CI Ready

**Principal Injection:**
- Accept caller identity via `x-emulator-principal` gRPC metadata
- Support `user:email@example.com` format
- Support `serviceAccount:name@project.iam.gserviceaccount.com` format
- Principal matching against binding members
- Backward compatible (no principal = check permission only)

**Policy Inheritance:**
- Resource hierarchy resolution (walk parent resources)
- Project-level policies inherited by child resources
- Resource-level policies override parent policies
- First policy match wins
- Example: `projects/p/secrets/s` inherits from `projects/p`

**Config File:**
- YAML config file loader (`--config policy.yaml`)
- Projects → bindings → resources structure
- Load policies at startup
- Example config file: `policy.yaml.example`

**Trace Mode:**
- `--trace` flag enables authz decision logging
- Structured logging with `log/slog`
- Logs: decision, principal, resource, permission, reason
- Helps debug "why was this denied?" questions

**CLI Updates:**
- `--config <path>`: Load policy YAML
- `--trace`: Enable trace mode
- `--port <int>`: Port (unchanged from v0.1.0)

**Testing:**
- Config loader tests (100% coverage)
- Policy inheritance tests (100% coverage)
- Principal matching tests
- Backward compatibility tests

### Technical Details
- Thread-safe policy loading
- Member matching: exact, allUsers, allAuthenticatedUsers
- Policy resolution walks resource hierarchy
- slog structured logging integration

**Coverage:** Storage 59.2%, Server 71.4%, Config 100%

## [0.1.0] - 2026-01-26

### Added
- **Initial Release**: Core IAM policy operations for emulator integration
- **IAM Policy API**:
  - SetIamPolicy: Set policies on any resource
  - GetIamPolicy: Retrieve policies for resources
  - TestIamPermissions: Check permission grants
- **Permission Evaluation Engine**:
  - Role-to-permission mapping
  - Support for primitive roles (owner, editor, viewer)
  - Support for Secret Manager roles (admin, secretAccessor)
  - Support for KMS roles (admin, cryptoKeyEncrypterDecrypter)
- **In-Memory Storage**:
  - Thread-safe concurrent access
  - Resource-level policy storage
  - No persistence (ephemeral for testing)
- **Docker Support**:
  - Single-stage build
  - Alpine-based for small image size

### Technical Details
- Simple permission evaluation (no conditions or inheritance)
- Resource-agnostic (works with any GCP resource path)
- 8 permissions across Secret Manager and KMS
- 7 predefined roles (3 primitive + 4 service-specific)
- Thread-safe with sync.RWMutex

### Limitations
- No service accounts or token minting
- No custom roles
- No conditional role bindings
- No organization/folder hierarchy
- No permission inheritance
- No audit logging

**Coverage:** 3 of ~3 core IAM policy methods (100% of MVP scope)

[Unreleased]: https://github.com/blackwell-systems/gcp-iam-emulator/compare/v0.2.0...HEAD
[0.2.0]: https://github.com/blackwell-systems/gcp-iam-emulator/releases/tag/v0.2.0
[0.1.0]: https://github.com/blackwell-systems/gcp-iam-emulator/releases/tag/v0.1.0
