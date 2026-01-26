# Features

This document tracks all implemented features in the GCP IAM Emulator.

## Core IAM Policy API (v0.1.0)

**Status:** ✓ Complete

### SetIamPolicy
- Set IAM policy on any resource path
- Accepts standard `google.iam.v1.Policy` protobuf
- Thread-safe with concurrent access
- In-memory storage (ephemeral)

### GetIamPolicy
- Retrieve IAM policy for a resource
- Returns empty policy if not found (no errors)
- Thread-safe read access

### TestIamPermissions
- Check which permissions are granted
- Accepts list of permissions to test
- Returns subset of allowed permissions
- Foundation for all authorization decisions

## Principal Authentication (v0.2.0)

**Status:** ✓ Complete

### gRPC Metadata Injection
- Extracts principal from `x-emulator-principal` metadata header
- Supports `user:email@example.com` format
- Supports `serviceAccount:name@project.iam.gserviceaccount.com` format
- No principal provided = backward compatible (permission check only)

### Principal Matching
- Exact string match: `user:alice@example.com` matches binding member `user:alice@example.com`
- Wildcard support: `allUsers` matches any principal
- Wildcard support: `allAuthenticatedUsers` matches any principal
- Case-sensitive matching

### Backward Compatibility
- No principal provided: checks if ANY binding grants permission (v0.1.0 behavior)
- Principal provided: checks if principal is in binding members (v0.2.0 behavior)

## Policy Inheritance (v0.2.0)

**Status:** ✓ Complete

### Resource Hierarchy Resolution
- Walks resource path hierarchy: `projects/p/locations/l/keyRings/k/cryptoKeys/ck` → `projects/p/locations/l/keyRings/k` → `projects/p/locations/l` → `projects/p`
- First policy match wins (resource-level overrides parent)
- Project-level policies inherited by all child resources
- No organization or folder hierarchy (project is root)

### Examples
```
Policy on: projects/test
Resource:   projects/test/secrets/db-password
Result:     Inherits project policy

Policy on: projects/test/secrets/db-password
Resource:   projects/test/secrets/db-password
Result:     Uses resource policy (overrides project)
```

## Configuration Management (v0.2.0)

**Status:** ✓ Complete

### YAML Config File
- Load policies from YAML at startup
- `--config policy.yaml` flag
- Projects → bindings structure
- Optional resource-level overrides

### Config Format
```yaml
projects:
  project-id:
    bindings:
      - role: roles/owner
        members:
          - user:admin@example.com
    resources:
      secrets/db-password:
        bindings:
          - role: roles/secretmanager.secretAccessor
            members:
              - serviceAccount:app@project.iam.gserviceaccount.com
```

### Config Validation
- YAML parsing errors reported at startup
- Invalid structure fails fast
- Example file: `policy.yaml.example`

## Trace Mode (v0.2.0)

**Status:** ✓ Complete

### Decision Logging
- `--trace` flag enables authz decision logging
- Uses Go `log/slog` structured logging
- Logs every permission check (ALLOW or DENY)

### Log Fields
- `decision`: "ALLOW" or "DENY"
- `principal`: who made the request
- `resource`: what resource was accessed
- `permission`: what permission was checked
- `reason`: why decision was made

### Example Output
```
level=INFO msg="authz decision" decision=ALLOW principal=user:alice@example.com resource=projects/test/secrets/db-password permission=secretmanager.versions.access reason="matched binding: role=roles/secretmanager.secretAccessor member=user:alice@example.com"

level=INFO msg="authz decision" decision=DENY principal=user:bob@example.com resource=projects/test/secrets/db-password permission=secretmanager.secrets.delete reason="no matching binding found for principal"
```

### Use Cases
- Debug "why was this denied?" questions
- Understand policy evaluation flow
- Audit authz decisions locally

## Roles & Permissions (v0.1.0)

**Status:** ✓ Complete (MVP set)

### Primitive Roles
- `roles/owner`: Full access to all resources
- `roles/editor`: Read/write access (no IAM management)
- `roles/viewer`: Read-only access

### Secret Manager Roles
- `roles/secretmanager.admin`: Full secret management
- `roles/secretmanager.secretAccessor`: Read secret values only

### KMS Roles
- `roles/cloudkms.admin`: Full KMS management
- `roles/cloudkms.cryptoKeyEncrypterDecrypter`: Encrypt/decrypt only

### Permission Mappings

**Secret Manager:**
- `secretmanager.secrets.get`
- `secretmanager.secrets.create`
- `secretmanager.secrets.delete`
- `secretmanager.versions.access`

**KMS:**
- `cloudkms.cryptoKeys.get`
- `cloudkms.cryptoKeys.encrypt`
- `cloudkms.cryptoKeys.decrypt`
- `cloudkms.cryptoKeyVersions.create`

**Total:** 7 roles, 8 permissions

## CLI & Launch (v0.2.0)

**Status:** ✓ Complete

### Command Line Flags
- `--port <int>`: Port to listen on (default: 8080)
- `--config <path>`: Path to policy YAML file
- `--trace`: Enable trace mode (decision logging)

### Launch Examples
```bash
# Basic server
server --port 8080

# With policy config
server --config policy.yaml

# With trace mode
server --config policy.yaml --trace

# Full CI setup
server --config ci-policies.yaml --trace --port 9090
```

### Docker Support
```bash
# Build
docker build -t gcp-iam-emulator:latest .

# Run with config
docker run -p 8080:8080 -v $(pwd)/policy.yaml:/policy.yaml gcp-iam-emulator --config /policy.yaml

# Run with trace
docker run -p 8080:8080 gcp-iam-emulator --trace
```

## gRPC Server (v0.1.0)

**Status:** ✓ Complete

### Protocol Support
- gRPC only (no REST yet - planned for v0.3.0)
- Server reflection enabled
- Standard Google IAM protobuf definitions
- Port 8080 default (configurable)

### Client Compatibility
- Works with official `cloud.google.com/go/iam/apiv1` clients
- Compatible with `google.golang.org/genproto/googleapis/iam/v1`
- Standard gRPC status codes

## Storage & Performance (v0.1.0)

**Status:** ✓ Complete

### In-Memory Storage
- Thread-safe with `sync.RWMutex`
- No persistence (ephemeral, resets on restart)
- Fast policy lookups
- Concurrent read/write support

### Performance Characteristics
- Sub-millisecond policy evaluation
- Scales to thousands of policies
- No external dependencies
- Suitable for CI/CD workloads

## Testing & Quality (v0.2.0)

**Status:** ✓ Complete

### Test Coverage
- Storage: 59.2% coverage
- Server: 71.4% coverage
- Config: 100% coverage (2/2 tests)
- Inheritance: 100% coverage (4/4 tests)

### Test Categories
- Unit tests: storage, server, config
- Integration tests: inheritance, principal matching
- Backward compatibility tests

### CI/CD
- GitHub Actions workflows
- Test + lint + build on PR/push
- Multi-platform Docker builds
- Automated releases

---

## Not Yet Implemented

### Deferred to Future Releases

**Hot Reload (v0.2.0 scope, deferred):**
- `--watch` flag for config file reloading
- SIGHUP signal handling
- Dynamic policy updates without restart

**REST API (v0.3.0):**
- HTTP/JSON gateway
- REST-native clients
- Dual protocol support (gRPC + REST)

**Custom Roles (future):**
- User-defined role-to-permission mappings
- Config file role definitions

**Conditional Bindings (v0.4.0):**
- CEL expression evaluation
- Time-based conditions
- Resource-based conditions

**Workload Identity (v0.4.0):**
- `principalSet://` member format
- Federated identity patterns
- Kubernetes workload identity

**Service Account Management (out of scope):**
- No service account CRUD
- No key generation
- No token minting

**Organization/Folder Hierarchy (out of scope):**
- No org-level policies
- No folder-level policies
- Project is root of hierarchy

---

## Feature Comparison Matrix

| Feature | v0.1.0 | v0.2.0 | v0.3.0 (planned) | v1.0.0 (planned) |
|---------|--------|--------|------------------|------------------|
| SetIamPolicy | ✓ | ✓ | ✓ | ✓ |
| GetIamPolicy | ✓ | ✓ | ✓ | ✓ |
| TestIamPermissions | ✓ | ✓ | ✓ | ✓ |
| Principal injection | - | ✓ | ✓ | ✓ |
| Policy inheritance | - | ✓ | ✓ | ✓ |
| Config file | - | ✓ | ✓ | ✓ |
| Trace mode | - | ✓ | ✓ | ✓ |
| Hot reload | - | - | ✓ | ✓ |
| REST API | - | - | ✓ | ✓ |
| Conditional bindings | - | - | - | ✓ |
| Metrics/observability | - | - | - | ✓ |

---

## Version History

### v0.2.0 (2026-01-26)
- Principal injection via gRPC metadata
- Policy inheritance (resource hierarchy)
- YAML config file loader
- Trace mode for authz debugging
- Enhanced CLI flags
- Comprehensive test suite

### v0.1.0 (2026-01-26)
- Core IAM Policy API (Set/Get/TestIamPermissions)
- 7 predefined roles (primitive + Secret Manager + KMS)
- 8 permission mappings
- Thread-safe in-memory storage
- gRPC server with reflection
- Docker support
