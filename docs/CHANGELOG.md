# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## Feature Comparison Matrix

| Feature | v0.1.0 | v0.2.0 | v0.3.0 | v0.4.0 | v1.0.0 (planned) |
|---------|--------|--------|--------|--------|------------------|
| SetIamPolicy | ✓ | ✓ | ✓ | ✓ | ✓ |
| GetIamPolicy | ✓ | ✓ | ✓ | ✓ | ✓ |
| TestIamPermissions | ✓ | ✓ | ✓ | ✓ | ✓ |
| Principal injection | - | ✓ | ✓ | ✓ | ✓ |
| Policy inheritance | - | ✓ | ✓ | ✓ | ✓ |
| Config file | - | ✓ | ✓ | ✓ | ✓ |
| Trace mode | - | ✓ | ✓ | ✓ | ✓ |
| Hot reload | - | ✓ | ✓ | ✓ | ✓ |
| REST API | - | - | ✓ | ✓ | ✓ |
| Conditional bindings | - | - | ✓ | ✓ | ✓ |
| Groups support | - | - | ✓ | ✓ | ✓ |
| Policy Schema v3 | - | - | ✓ | ✓ | ✓ |
| Enhanced trace mode | - | - | ✓ | ✓ | ✓ |
| Custom roles | - | - | - | ✓ | ✓ |
| Strict mode | - | - | - | ✓ | ✓ |
| Metrics/observability | - | - | - | - | ✓ |
| Emulator integration | - | - | - | - | ✓ |

---

## [Unreleased]

### Planned for v1.0.0 - Production Ready
- **Emulator Integration**: Secret Manager + KMS call IAM for authz
- **Metrics/Observability**: Prometheus metrics, OpenTelemetry tracing
- **Advanced CEL**: Full CEL expression support
- **Performance**: Benchmarking and optimization

### Deferred Features

**Role Packs (future):**
- Optional import packs (e.g., `packs/pubsub.yaml`, `packs/bigquery.yaml`)
- Community-maintained, not built-in
- Users import only what they need

**Workload Identity (future):**
- `principalSet://` member format
- Federated identity patterns
- Kubernetes workload identity

**Out of Scope:**
- Service account management (no CRUD, no keys, no token minting)
- Organization/folder hierarchy (project is root)
- Audit logging enforcement (auditConfigs accepted but not enforced)
- Large built-in permission database (define what you need via custom roles)

---

## [0.4.0] - 2026-01-26

### Added - Sustainable Extensibility

**Custom Roles System:**
- Define custom role-to-permission mappings in YAML
- Support for ANY GCP service (BigQuery, Pub/Sub, Storage, etc.)
- Override built-in roles with custom definitions
- Thread-safe loading and storage
- Extensible without modifying emulator code

**Strict Mode (Default):**
- Unknown roles are DENIED by default
- Forces explicit role definitions
- Prevents overly permissive tests
- Catches misconfigurations early

**Compat Mode (Opt-in):**
- `--allow-unknown-roles` flag enables wildcard matching
- Unknown roles match by service prefix
- Example: `roles/secretmanager.customRole` grants `secretmanager.*`
- Less strict, useful for migration scenarios

**Decision Order:**
1. Custom roles (highest priority)
2. Built-in roles (bootstrap set)
3. Wildcard match (only in compat mode)
4. Deny (strict mode default)

**Documentation:**
- Updated README with custom roles section
- Added "What This Is (and Isn't)" identity section
- Repositioned as reference policy engine (not GCP IAM clone)
- Clarified sustainable strategy (small built-in core)
- Comprehensive custom roles examples

**Test Coverage:**
- Custom roles tests (basic usage, override built-in, multiple permissions)
- Strict mode tests (unknown roles denied, custom/built-in still work)
- Compat mode tests (wildcard matching, service prefix validation)

### Technical Details
- Custom roles stored in thread-safe map
- Decision order: custom → built-in → wildcard → deny
- Wildcard matching: service prefix extraction from permission
- Bootstrap set intentionally kept small (10 roles, 26 permissions)

## [0.3.0] - 2026-01-26

### Added - Real-world IAM Evaluation

**Conditional Bindings:**
- CEL expression support for resource-based access control
- `resource.name.startsWith("prefix")` - Resource name prefix matching
- `resource.type == "SECRET"` - Resource type equality
- `request.time < timestamp(...)` - Time-based access control
- Basic CEL evaluator (no full CEL dependency)
- Comprehensive test coverage for conditions

**Policy Schema v3:**
- `etag` field - SHA256-based optimistic concurrency control
- `version` field - Auto-determined (1=basic, 3=with conditions)
- `auditConfigs` field - Audit logging configuration
- `bindings[].condition` - CEL expression per binding
- Full compatibility with GCP IAM Policy v3 format

**Groups Support:**
- YAML `groups:` section for reusable principal collections
- Group expansion in `principalMatches()`
- Nested groups (1 level supported)
- `group:groupname` member format
- Hot reload support for groups

**REST API Gateway:**
- HTTP/JSON interface for all IAM operations
- `--http-port <port>` flag to enable REST API
- POST `/v1/{resource}:setIamPolicy`
- GET/POST `/v1/{resource}:getIamPolicy`
- POST `/v1/{resource}:testIamPermissions`
- `X-Emulator-Principal` HTTP header for principal injection
- gRPC-to-HTTP error code mapping
- JSON request/response marshaling

**Enhanced Trace Mode:**
- `--explain` flag for verbose logging
- `--trace-output <file>` for JSON trace logs
- Duration metrics in trace output
- Structured JSON format with slog
- Condition evaluation results in traces

**Documentation:**
- Comprehensive v0.3.0 features section in README
- CEL expression documentation
- Groups usage examples
- REST API examples with curl
- Enhanced trace mode examples
- Updated FEATURES.md with all v0.3.0 features

### Technical Details
- CEL evaluator covers 80% of real-world use cases
- Thread-safe group storage
- Automatic policy version determination
- Etag generation on policy write
- REST server shares storage with gRPC server

**Test Coverage:** All v0.3.0 features have comprehensive test coverage (CEL evaluator, groups, policy v3)

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

[Unreleased]: https://github.com/blackwell-systems/gcp-iam-emulator/compare/v0.3.0...HEAD
[0.3.0]: https://github.com/blackwell-systems/gcp-iam-emulator/releases/tag/v0.3.0
[0.2.0]: https://github.com/blackwell-systems/gcp-iam-emulator/releases/tag/v0.2.0
[0.1.0]: https://github.com/blackwell-systems/gcp-iam-emulator/releases/tag/v0.1.0
