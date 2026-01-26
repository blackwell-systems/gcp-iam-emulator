# GCP IAM Emulator

[![Blackwell Systems](https://raw.githubusercontent.com/blackwell-systems/blackwell-docs-theme/main/badge-trademark.svg)](https://github.com/blackwell-systems)
[![Go Reference](https://pkg.go.dev/badge/github.com/blackwell-systems/gcp-iam-emulator.svg)](https://pkg.go.dev/github.com/blackwell-systems/gcp-iam-emulator)
[![Go Version](https://img.shields.io/badge/go-1.24+-blue.svg)](https://go.dev/)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)

> The reference local implementation of the Google Cloud IAM API for development and CI

A production-grade IAM policy engine providing complete, behaviorally-accurate permission evaluation for local development and CI/CD. The missing auth layer for GCP emulators. No GCP credentials or network connectivity required.

## Features

- **Complete IAM Policy API** - SetIamPolicy, GetIamPolicy, TestIamPermissions (gRPC + REST)
- **Real Permission Evaluation** - Accurate role-to-permission mapping
- **Conditional Bindings** - CEL expression support for resource-based access control
- **Groups Support** - Define reusable groups with nested membership (1 level)
- **Policy Schema v3** - Full support for etag, version, auditConfigs, conditions
- **Enhanced Trace Mode** - JSON output, verbose logging, duration metrics
- **Custom Roles** - Define any GCP permission in YAML (extensible, not hardcoded)
- **Small Built-in Core** - Primitive roles + Secret Manager + KMS (bootstrap only)
- **No GCP Credentials** - Works entirely offline without authentication
- **Fast & Lightweight** - In-memory storage, starts in milliseconds
- **Thread-Safe** - Concurrent access with proper synchronization
- **Integrates with Emulators** - Works with gcp-secret-manager-emulator, gcp-kms-emulator

## Supported Operations

### IAM Policy Management
- `SetIamPolicy` - Set IAM policy on any resource
- `GetIamPolicy` - Retrieve IAM policy for a resource
- `TestIamPermissions` - Check which permissions are granted

### Supported Roles

**Primitive roles:**
- `roles/owner` - Full access to all resources
- `roles/editor` - Read/write access (no IAM management, no delete)
- `roles/viewer` - Read-only access

**Secret Manager roles:**
- `roles/secretmanager.admin` - Full secret management
- `roles/secretmanager.secretAccessor` - Read secret values only
- `roles/secretmanager.secretVersionManager` - Manage versions (add, enable, disable, destroy)

**KMS roles:**
- `roles/cloudkms.admin` - Full KMS management
- `roles/cloudkms.cryptoKeyEncrypterDecrypter` - Encrypt/decrypt only
- `roles/cloudkms.viewer` - Read-only KMS access

### Supported Permissions

**Secret Manager (12 permissions):**
- `secretmanager.secrets.get` - Get secret metadata
- `secretmanager.secrets.create` - Create new secrets
- `secretmanager.secrets.update` - Update secret metadata
- `secretmanager.secrets.delete` - Delete secrets
- `secretmanager.secrets.list` - List secrets
- `secretmanager.versions.add` - Add new secret versions
- `secretmanager.versions.get` - Get version metadata
- `secretmanager.versions.access` - Read secret values
- `secretmanager.versions.list` - List versions
- `secretmanager.versions.enable` - Enable versions
- `secretmanager.versions.disable` - Disable versions
- `secretmanager.versions.destroy` - Destroy versions

**KMS (14 permissions):**
- `cloudkms.keyRings.create` - Create key rings
- `cloudkms.keyRings.get` - Get key ring metadata
- `cloudkms.keyRings.list` - List key rings
- `cloudkms.cryptoKeys.create` - Create crypto keys
- `cloudkms.cryptoKeys.get` - Get key metadata
- `cloudkms.cryptoKeys.list` - List keys
- `cloudkms.cryptoKeys.update` - Update key metadata
- `cloudkms.cryptoKeys.encrypt` - Encrypt data
- `cloudkms.cryptoKeys.decrypt` - Decrypt data
- `cloudkms.cryptoKeyVersions.create` - Create key versions
- `cloudkms.cryptoKeyVersions.get` - Get version metadata
- `cloudkms.cryptoKeyVersions.list` - List versions
- `cloudkms.cryptoKeyVersions.update` - Update version state
- `cloudkms.cryptoKeyVersions.destroy` - Destroy versions

**Total:** 10 roles, 26 permissions (complete coverage of emulator operations)

## Quick Start

### Install

```bash
go install github.com/blackwell-systems/gcp-iam-emulator/cmd/server@latest
```

### Run Server

**Basic:**
```bash
# Start on default port 8080
server

# Custom port
server --port 9090
```

**With policy config (recommended for CI):**
```bash
# Load policies from YAML
server --config policy.yaml

# Enable trace mode for debugging
server --config policy.yaml --trace

# Enable HTTP REST API
server --config policy.yaml --http-port 8081

# Enable verbose trace with JSON output
server --config policy.yaml --explain --trace-output trace.json

# Hot reload policies on file changes
server --config policy.yaml --watch
```

**Docker:**
```bash
# Run with mounted config
docker run -p 8080:8080 -v $(pwd)/policy.yaml:/policy.yaml \
  ghcr.io/blackwell-systems/gcp-iam-emulator:latest --config /policy.yaml

# Run with trace mode
docker run -p 8080:8080 \
  ghcr.io/blackwell-systems/gcp-iam-emulator:latest --trace
```

### Example Policy Config

Create `policy.yaml`:

```yaml
projects:
  test-project:
    bindings:
      - role: roles/owner
        members:
          - user:admin@example.com
      
      - role: roles/secretmanager.secretAccessor
        members:
          - serviceAccount:ci@test-project.iam.gserviceaccount.com
    
    resources:
      secrets/db-password:
        bindings:
          - role: roles/secretmanager.secretAccessor
            members:
              - serviceAccount:app@test-project.iam.gserviceaccount.com
```

### Use with GCP SDK

**Go client with principal injection:**

```go
package main

import (
    "context"

    iampb "google.golang.org/genproto/googleapis/iam/v1"
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"
    "google.golang.org/grpc/metadata"
)

func main() {
    ctx := context.Background()

    conn, _ := grpc.NewClient(
        "localhost:8080",
        grpc.WithTransportCredentials(insecure.NewCredentials()),
    )
    defer conn.Close()

    client := iampb.NewIAMPolicyClient(conn)

    // Inject principal identity via metadata
    md := metadata.Pairs("x-emulator-principal", "serviceAccount:ci@test-project.iam.gserviceaccount.com")
    ctx = metadata.NewOutgoingContext(ctx, md)

    // Test permissions (checks if this principal has access)
    resp, _ := client.TestIamPermissions(ctx, &iampb.TestIamPermissionsRequest{
        Resource: "projects/test-project/secrets/db-password",
        Permissions: []string{
            "secretmanager.versions.access",
            "secretmanager.secrets.delete",
        },
    })

    // resp.Permissions = ["secretmanager.versions.access"]
    // (delete denied - secretAccessor role doesn't include it)
}
```

**Setting policies dynamically (no config file):**

```go
// Set policy via API
policy := &iampb.Policy{
    Bindings: []*iampb.Binding{
        {
            Role: "roles/secretmanager.secretAccessor",
            Members: []string{
                "serviceAccount:ci@my-project.iam.gserviceaccount.com",
            },
        },
    },
}

client.SetIamPolicy(ctx, &iampb.SetIamPolicyRequest{
    Resource: "projects/my-project/secrets/my-secret",
    Policy:   policy,
})
```

## Use Cases

- **CI/CD Pipelines** - Drop-in IAM for hermetic testing without GCP credentials
- **Policy Development** - Iterate on IAM policies locally with instant feedback
- **Security Testing** - Validate "who can access what" before production
- **Permission Debugging** - Trace mode explains why access was granted/denied
- **Integration Testing** - Real permission evaluation with Secret Manager + KMS emulators
- **Cost Reduction** - Avoid GCP API charges during development

## CI/CD Integration

### GitHub Actions

```yaml
name: Integration Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    
    services:
      iam-emulator:
        image: ghcr.io/blackwell-systems/gcp-iam-emulator:latest
        ports:
          - 8080:8080
        options: --mount type=bind,source=${{ github.workspace }}/policy.yaml,target=/policy.yaml
        args: --config /policy.yaml --trace
    
    steps:
      - uses: actions/checkout@v4
      
      - name: Run integration tests
        env:
          IAM_EMULATOR_HOST: localhost:8080
        run: |
          go test ./...
```

### Docker Compose

**Full GCP emulator stack:**

```yaml
# docker-compose.yml
services:
  iam:
    image: ghcr.io/blackwell-systems/gcp-iam-emulator:latest
    ports:
      - "8080:8080"
    volumes:
      - ./policy.yaml:/policy.yaml
    command: --config /policy.yaml --trace
  
  secret-manager:
    image: ghcr.io/blackwell-systems/gcp-secret-manager-emulator:latest
    environment:
      IAM_EMULATOR_HOST: iam:8080
    ports:
      - "9090:9090"
    depends_on:
      - iam
  
  kms:
    image: ghcr.io/blackwell-systems/gcp-kms-emulator:latest
    environment:
      IAM_EMULATOR_HOST: iam:8080
    ports:
      - "9091:9090"
    depends_on:
      - iam
```

**Run:**
```bash
docker-compose up

# Your tests connect to localhost:8080 (IAM), localhost:9090 (Secret Manager), localhost:9091 (KMS)
```

## Trace Mode

Enable trace mode to debug authorization decisions:

```bash
server --config policy.yaml --trace
```

**Example output:**

```
2026/01/26 10:30:15 GCP IAM Emulator v0.2.0
2026/01/26 10:30:15 Loading policy config from policy.yaml
2026/01/26 10:30:15 Loaded 3 policies from config
2026/01/26 10:30:15 Trace mode: ENABLED (authz decisions will be logged)
2026/01/26 10:30:15 Server ready - listening on :8080

level=INFO msg="authz decision" decision=ALLOW principal=serviceAccount:ci@test.iam.gserviceaccount.com resource=projects/test/secrets/api-key permission=secretmanager.versions.access reason="matched binding: role=roles/secretmanager.secretAccessor member=serviceAccount:ci@test.iam.gserviceaccount.com"

level=INFO msg="authz decision" decision=DENY principal=user:dev@example.com resource=projects/test/secrets/db-password permission=secretmanager.secrets.delete reason="no matching binding found for principal"
```

**Use trace mode to:**
- Understand why access was denied
- Debug policy inheritance
- Verify principal matching
- Audit authz decisions in local testing

**Enhanced trace mode (v0.3.0+):**

```bash
# Verbose logging with --explain
server --config policy.yaml --explain

# JSON output to file
server --config policy.yaml --trace-output trace.json
```

**JSON trace format:**
```json
{
  "time":"2026-01-26T10:30:15Z",
  "level":"INFO",
  "msg":"permission_check",
  "resource":"projects/test/secrets/api-key",
  "principal":"serviceAccount:ci@test.iam.gserviceaccount.com",
  "allowed_permissions":["secretmanager.versions.access"],
  "duration_ms":2,
  "timestamp":"2026-01-26T10:30:15Z"
}
```

## v0.3.0 Features

### Conditional Bindings

Use CEL expressions to restrict access based on resource properties:

```yaml
projects:
  test-project:
    bindings:
      # CI can only access production secrets
      - role: roles/secretmanager.secretAccessor
        members:
          - serviceAccount:ci@test-project.iam.gserviceaccount.com
        condition:
          expression: 'resource.name.startsWith("projects/test-project/secrets/prod-")'
          title: "Production secrets only"
          description: "CI service account restricted to production secrets"
      
      # Time-based access
      - role: roles/cloudkms.cryptoKeyEncrypterDecrypter
        members:
          - serviceAccount:temp-access@test-project.iam.gserviceaccount.com
        condition:
          expression: 'request.time < timestamp("2026-12-31T23:59:59Z")'
          title: "Temporary access"
```

**Supported CEL expressions:**
- `resource.name.startsWith("prefix")` - Match resource name prefix
- `resource.type == "SECRET"` - Match resource type (SECRET, CRYPTO_KEY, KEY_RING)
- `request.time < timestamp("2026-12-31T00:00:00Z")` - Time-based access

### Groups Support

Define reusable groups to reduce duplication:

```yaml
groups:
  developers:
    members:
      - user:alice@example.com
      - user:bob@example.com
      - serviceAccount:dev-bot@test-project.iam.gserviceaccount.com
  
  operators:
    members:
      - user:ops@example.com
      - group:oncall  # Nested groups (1 level supported)
  
  oncall:
    members:
      - user:charlie@example.com
      - user:diana@example.com

projects:
  test-project:
    bindings:
      - role: roles/viewer
        members:
          - group:developers  # Reference group
```

### REST API

HTTP REST gateway for all IAM operations:

```bash
# Start with REST API
server --config policy.yaml --http-port 8081
```

**Example requests:**

```bash
# Set IAM policy
curl -X POST http://localhost:8081/v1/projects/test-project:setIamPolicy \
  -H "Content-Type: application/json" \
  -d '{
    "policy": {
      "bindings": [{
        "role": "roles/viewer",
        "members": ["user:dev@example.com"]
      }]
    }
  }'

# Get IAM policy
curl http://localhost:8081/v1/projects/test-project:getIamPolicy

# Test permissions
curl -X POST http://localhost:8081/v1/projects/test-project/secrets/api-key:testIamPermissions \
  -H "Content-Type: application/json" \
  -H "X-Emulator-Principal: serviceAccount:ci@test.iam.gserviceaccount.com" \
  -d '{
    "permissions": [
      "secretmanager.versions.access",
      "secretmanager.secrets.delete"
    ]
  }'
```

**Response:**
```json
{
  "permissions": ["secretmanager.versions.access"]
}
```

### Policy Schema v3

Full support for IAM Policy v3 features:

- **etag** - Optimistic concurrency control (SHA256-based)
- **version** - Policy format version (1=basic, 3=with conditions)
- **auditConfigs** - Audit logging configuration
- **bindings[].condition** - Conditional role bindings

```yaml
projects:
  test-project:
    auditConfigs:
      - service: secretmanager.googleapis.com
        auditLogConfigs:
          - logType: ADMIN_READ
          - logType: DATA_READ
          - logType: DATA_WRITE
            exemptedMembers:
              - serviceAccount:logging@test-project.iam.gserviceaccount.com
```

### Custom Roles (v0.4.0)

Define your own role-to-permission mappings for any GCP service:

```yaml
roles:
  roles/custom.dataReader:
    permissions:
      - bigquery.datasets.get
      - bigquery.tables.list
      - bigquery.tables.getData
  
  roles/custom.pubsubPublisher:
    permissions:
      - pubsub.topics.publish
      - pubsub.topics.get
  
  roles/custom.storageAdmin:
    permissions:
      - storage.buckets.create
      - storage.buckets.delete
      - storage.objects.create
      - storage.objects.delete

projects:
  test-project:
    bindings:
      - role: roles/custom.dataReader
        members:
          - user:analyst@example.com
```

**Features:**
- **Extensible** - Define permissions for any GCP service
- **Override built-in roles** - Custom roles take precedence
- **Strict mode by default** - Unknown roles denied (catches misconfigurations)
- **Compat mode available** - Wildcard fallback with `--allow-unknown-roles`
  - `roles/secretmanager.customRole` → grants `secretmanager.*`
  - `roles/cloudkms.encryptOnly` → grants `cloudkms.*`

**Why this matters:**
- GCP has thousands of permissions - hardcoding doesn't scale
- Each test environment needs different permissions
- Keeps emulator offline, deterministic, and CI-friendly

**Modes:**

**Strict mode (default, recommended):**
```bash
server --config policy.yaml
```
- Unknown roles → **DENIED**
- Custom roles → allowed
- Built-in roles → allowed
- **Catches bugs**: Tests fail if you use a role you haven't defined

**Compat mode (less strict):**
```bash
server --config policy.yaml --allow-unknown-roles
```
- Unknown roles → **wildcard match** (if service prefix matches)
- More permissive, but can hide bugs
- Use when migrating existing tests

**Decision order:**
1. Custom roles (highest priority)
2. Built-in roles (primitives + Secret Manager + KMS)
3. Wildcard match (only in compat mode)
4. Deny (strict mode default)

## Architecture

**In-memory policy storage** with thread-safe concurrent access. **Simple permission engine** mapping roles to permissions. **Resource-level policies** (no organization/folder hierarchy in MVP). **No token minting** (pure policy evaluation only).

## Design Philosophy

**This emulator is not a full IAM reimplementation.** It provides deterministic, CI-friendly policy evaluation. Custom roles let you model the subset of permissions your tests require, without hardcoding the entire GCP permission universe.

**Goal:** Catch authorization bugs locally (missing permissions, wrong roles, misconfigured principals) without requiring GCP credentials or network access.

**Non-goal:** Perfect fidelity with every GCP IAM edge case. If your test passes here but fails in real GCP, that's a signal to refine your custom role definitions.

## Roadmap

**Future Considerations:**

**Role Packs (optional convenience):**
- Optional import packs like `packs/pubsub.yaml`, `packs/bigquery.yaml`
- Users import only what they need
- Community-maintained, not hardcoded in emulator
- Example: `roles: !include packs/storage.yaml`

**NOT planned:**
- Large built-in permission database (creates maintenance hell)
- Auto-syncing with GCP IAM API (adds network dependency)
- Perfect GCP IAM fidelity (not the goal)

**Strategy:** Keep the emulator **focused and sustainable**. Users define what they need via custom roles. The built-in set stays small (primitives + Secret Manager + KMS only).

See [ROADMAP.md](docs/ROADMAP.md) for full details.

## API Parity with GCP IAM

### What's Implemented (v0.2.0)

**Methods:**
- ✅ `SetIamPolicy` - Full implementation
- ✅ `GetIamPolicy` - Full implementation  
- ✅ `TestIamPermissions` - Full implementation with principal matching

**Policy Fields:**
- ✅ `bindings[]` - Role assignments with members
  - ✅ `role` - IAM role string
  - ✅ `members[]` - Principal identifiers

**Features:**
- ✅ Principal injection via gRPC metadata
- ✅ Resource hierarchy policy inheritance
- ✅ 26 permissions across Secret Manager + KMS
- ✅ 10 roles (primitive + service-specific)
- ✅ Trace mode for debugging authz decisions

### What's Missing (Planned for v0.3.0)

**Policy Fields:**
- ❌ `version` - Policy format version (0, 1, 3)
- ❌ `etag` - Optimistic concurrency control
- ❌ `auditConfigs[]` - Audit logging configuration
- ❌ `bindings[].condition` - Conditional role bindings (CEL expressions)

**Impact on v0.2.0:**
- Clients expecting `etag` will not see it (optimistic locking not enforced)
- Policies with conditions won't work (version 3 not supported)
- Full policy roundtrip may lose `auditConfigs` data
- **Workaround:** Use emulator for testing, not for production policy management

### Additional Limitations

- No custom roles (primitive + service roles only)
- No organization/folder hierarchy (project is root)
- No service accounts or token minting
- No audit logging enforcement
- No REST API (gRPC only in v0.2.0, REST planned for v0.3.0)

**Current scope:** Core IAM policy operations for CI/CD testing with emulators

## Project Status

Extracted as the strategic "keystone" auth layer to enable complete GCP emulator ecosystem testing.

## Disclaimer

This project is not affiliated with, endorsed by, or sponsored by Google LLC or Google Cloud Platform. "Google Cloud", "IAM", and related trademarks are property of Google LLC. This is an independent open-source implementation for testing and development purposes.

## Maintained By

Maintained by **Dayna Blackwell** — founder of Blackwell Systems, building reference infrastructure for cloud-native development.

[GitHub](https://github.com/blackwell-systems) · [LinkedIn](https://linkedin.com/in/dayna-blackwell) · [Blog](https://blog.blackwell-systems.com)

## Trademarks

**Blackwell Systems™** and the **Blackwell Systems logo** are trademarks of Dayna Blackwell. You may use the name "Blackwell Systems" to refer to this project, but you may not use the name or logo in a way that suggests endorsement or official affiliation without prior written permission. See [BRAND.md](BRAND.md) for usage guidelines.

## Related Projects

- [GCP Secret Manager Emulator](https://github.com/blackwell-systems/gcp-secret-manager-emulator) - Reference implementation for Secret Manager API
- [GCP KMS Emulator](https://github.com/blackwell-systems/gcp-kms-emulator) - Reference implementation for KMS API

## License

Apache License 2.0 - See [LICENSE](LICENSE) for details.
