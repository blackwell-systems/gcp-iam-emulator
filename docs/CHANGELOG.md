# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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

[Unreleased]: https://github.com/blackwell-systems/gcp-iam-emulator/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/blackwell-systems/gcp-iam-emulator/releases/tag/v0.1.0
