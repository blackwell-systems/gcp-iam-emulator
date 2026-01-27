# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.5.0] - 2026-01-27

### Added
- **Integration Contract Specification**: Complete contract for emulator mesh integration
  - Canonical resource naming patterns
  - Operation-to-permission mapping requirements
  - Principal injection and propagation standards
  - IAM mode configuration (off/permissive/strict)
  - Error response formats
  - Testing requirements
  - Documentation requirements
  - Backward compatibility guarantees
- **Implementation Checklist**: Detailed checklist for new emulators joining the mesh
  - Resource naming validation
  - Permission mapping documentation
  - Principal extraction patterns
  - IAM client integration
  - Test coverage requirements

### Changed
- Clarified non-breaking, opt-in IAM integration strategy
- Refined integration contract with implementation details
- Updated documentation to emphasize reference implementation role

### Technical Details
- Integration contract defines stable API for data plane emulators
- All emulators implement same contract for consistent behavior
- Documentation improved for emulator authors

## [0.4.0] - 2026-01-26

### Added
- Group membership resolution
- CEL condition evaluation with `resource.name` context
- Trace logging mode with `--trace` flag
- Health check endpoint at `/health`
- Policy inheritance for project hierarchies

### Changed
- Improved permission check performance
- Enhanced error messages with detailed denial reasons
- Optimized role expansion algorithm

### Fixed
- Condition evaluation edge cases
- Group expansion recursion issues

## [0.3.0] - 2026-01-25

### Added
- Common Expression Language (CEL) support for conditions
- Conditional bindings with CEL expressions
- Support for `resource.name.startsWith()` and other CEL functions
- Advanced condition evaluation with logical operators

### Changed
- Policy format supports `condition` field in bindings
- Permission checks now evaluate conditions

## [0.2.0] - 2026-01-24

### Added
- `TestIamPermissions` gRPC API implementation
- Group support in policy.yaml
- Custom role definitions
- Project-level IAM bindings
- Permission checking against policy

### Changed
- Policy format expanded to include groups and projects
- IAM evaluation logic matches GCP behavior

## [0.1.0] - 2026-01-21

### Added
- Initial release
- Basic policy loading from YAML file
- Role-based access control (RBAC)
- Simple permission evaluation
- gRPC server implementation
- Docker container support
- Basic documentation

### Features
- Load policy from `policy.yaml`
- Define custom roles with permissions
- Assign roles to principals
- Evaluate permission checks
- Support for `user:`, `serviceAccount:`, `group:` principals

### Security
- Runs as non-root user in Docker
- No authentication (emulator design)
- File-based policy (no persistence)

[Unreleased]: https://github.com/blackwell-systems/gcp-iam-emulator/compare/v0.5.0...HEAD
[0.5.0]: https://github.com/blackwell-systems/gcp-iam-emulator/compare/v0.4.0...v0.5.0
[0.4.0]: https://github.com/blackwell-systems/gcp-iam-emulator/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/blackwell-systems/gcp-iam-emulator/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/blackwell-systems/gcp-iam-emulator/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/blackwell-systems/gcp-iam-emulator/releases/tag/v0.1.0
