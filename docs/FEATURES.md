# Features

## The Core Idea

**GCP IAM Emulator is a deterministic, offline IAM policy engine for local dev + CI.**

- The built-in roles are **bootstrap only**.
- Your test environment defines the *permission universe* via **custom roles in YAML**.
- **Strict mode (default)** denies unknown roles to prevent accidentally-permissive tests.
- **Compat mode** (`--allow-unknown-roles`) exists for migration (wildcard/prefix fallback).

If you need more services, **you don't wait on this repo** - you define the roles you need and the emulator enforces them.

---

## What You Get

### Core IAM Policy API

Complete implementation of the Google Cloud IAM Policy API:

- **SetIamPolicy** - Set IAM policy on any resource path
- **GetIamPolicy** - Retrieve IAM policy for a resource
- **TestIamPermissions** - Check which permissions are granted

Works with official GCP client libraries via gRPC and REST.

### Custom Roles

Define your own role-to-permission mappings for any GCP service:

```yaml
roles:
  roles/custom.dataReader:
    permissions:
      - bigquery.datasets.get
      - bigquery.tables.list
      
  roles/custom.pubsubPublisher:
    permissions:
      - pubsub.topics.publish
```

- Override built-in roles with custom definitions
- Support for ANY GCP service (BigQuery, Pub/Sub, Storage, etc.)
- Extensible without modifying emulator code

### Strict Mode (Default)

Unknown roles are denied to catch misconfigurations:

```bash
server --config policy.yaml
```

- Forces explicit role definitions
- Prevents overly permissive tests
- Tests fail if you use undefined roles

### Compat Mode (Opt-in)

Wildcard matching for migration scenarios:

```bash
server --config policy.yaml --allow-unknown-roles
```

- Unknown roles match by service prefix
- Example: `roles/secretmanager.customRole` grants `secretmanager.*`
- Less strict, useful when migrating existing tests

**Decision order:** Custom roles → Built-in roles → Wildcard (compat only) → Deny

### Conditional Bindings

Restrict access with CEL expressions:

```yaml
bindings:
  - role: roles/secretmanager.secretAccessor
    members:
      - serviceAccount:ci@test.iam.gserviceaccount.com
    condition:
      expression: 'resource.name.startsWith("projects/test/secrets/prod-")'
      title: "Production secrets only"
```

Supported conditions:
- `resource.name.startsWith("prefix")` - Resource name prefix matching
- `resource.type == "SECRET"` - Resource type equality
- `request.time < timestamp("2026-12-31T23:59:59Z")` - Time-based access

### Groups Support

Define reusable principal collections:

```yaml
groups:
  developers:
    members:
      - user:alice@example.com
      - user:bob@example.com
      - group:oncall  # Nested groups (1 level)

projects:
  test-project:
    bindings:
      - role: roles/viewer
        members:
          - group:developers
```

### Policy Schema v3

Full support for IAM Policy v3 features:

- **etag** - Optimistic concurrency control (SHA256-based)
- **version** - Policy format version (1=basic, 3=conditions)
- **auditConfigs** - Audit logging configuration
- **bindings[].condition** - Conditional role bindings

### REST API

HTTP/JSON gateway for all IAM operations:

```bash
server --config policy.yaml --http-port 8081

curl -X POST http://localhost:8081/v1/projects/test:testIamPermissions \
  -H "X-Emulator-Principal: serviceAccount:ci@test.iam.gserviceaccount.com" \
  -d '{"permissions": ["secretmanager.versions.access"]}'
```

### Principal Authentication

Inject identity via gRPC metadata or HTTP headers:

- **gRPC:** `x-emulator-principal` metadata
- **HTTP:** `X-Emulator-Principal` header

Supported formats:
- `user:email@example.com`
- `serviceAccount:name@project.iam.gserviceaccount.com`
- `group:group-name`
- `allUsers`
- `allAuthenticatedUsers`

### Policy Inheritance

Resource hierarchy resolution:

```
Policy on: projects/test
Resource:  projects/test/secrets/db-password
Result:    Inherits project policy

Policy on: projects/test/secrets/db-password
Resource:  projects/test/secrets/db-password
Result:    Uses resource policy (overrides project)
```

### Trace Mode

Debug authorization decisions:

```bash
# Basic trace
server --config policy.yaml --trace

# Verbose trace with JSON output
server --config policy.yaml --explain --trace-output trace.json
```

Logs every permission check with:
- Decision (ALLOW/DENY)
- Principal
- Resource
- Permission
- Reason
- Duration metrics

### Configuration Management

Load policies from YAML:

```yaml
projects:
  test-project:
    bindings:
      - role: roles/owner
        members:
          - user:admin@example.com
    resources:
      secrets/db-password:
        bindings:
          - role: roles/secretmanager.secretAccessor
            members:
              - serviceAccount:app@test.iam.gserviceaccount.com
```

### Hot Reload

Watch config file for changes:

```bash
server --config policy.yaml --watch
```

Updates policies without restart.

### Built-in Roles (Bootstrap Set)

**10 built-in roles, 26 permissions** - intentionally small:

**Primitive roles:**
- `roles/owner` - Full access
- `roles/editor` - Read/write (no IAM)
- `roles/viewer` - Read-only

**Secret Manager roles:**
- `roles/secretmanager.admin`
- `roles/secretmanager.secretAccessor`
- `roles/secretmanager.secretVersionManager`

**KMS roles:**
- `roles/cloudkms.admin`
- `roles/cloudkms.cryptoKeyEncrypterDecrypter`
- `roles/cloudkms.viewer`

**Strategy:** Small built-in core for immediate use. Define custom roles for production tests.

---

## Why This Matters

**Offline, deterministic authorization testing:**
- No GCP credentials required
- No network connectivity required
- Instant feedback in CI/CD

**Catch bugs before production:**
- Missing permissions
- Wrong roles
- Misconfigured principals
- Overly permissive policies

**Composable infrastructure:**
- Foundation for GCP emulator stacks
- Works with Secret Manager emulator
- Works with KMS emulator
- Standard gRPC + REST APIs

**Sustainable and extensible:**
- Small emulator core (never bloats)
- Users define their own permission universe
- No maintenance hell from hardcoded permission databases
- Strict mode prevents configuration drift

---

## CLI Reference

```bash
server [flags]

Flags:
  --port <int>              gRPC port (default: 8080)
  --http-port <int>         HTTP REST port (0 = disabled)
  --config <path>           Policy YAML file
  --trace                   Enable trace mode (decision logging)
  --explain                 Enable verbose trace output (implies --trace)
  --trace-output <path>     Output file for JSON trace logs
  --watch                   Watch config file for changes and hot reload
  --allow-unknown-roles     Enable wildcard role matching (compat mode)
```

**Examples:**

```bash
# Basic server
server

# With policy config
server --config policy.yaml

# Strict mode (default) - deny unknown roles
server --config policy.yaml --trace

# Compat mode - allow wildcard matching
server --config policy.yaml --allow-unknown-roles

# Full CI setup
server --config ci-policies.yaml --trace --http-port 8081 --watch
```

---

## Docker Support

```bash
# Run with config
docker run -p 8080:8080 -v $(pwd)/policy.yaml:/policy.yaml \
  ghcr.io/blackwell-systems/gcp-iam-emulator:latest --config /policy.yaml

# Run with trace mode
docker run -p 8080:8080 \
  ghcr.io/blackwell-systems/gcp-iam-emulator:latest --trace
```

---

## What's Not Included

**By design:**
- No organization/folder hierarchy (project is root)
- No service account management (no CRUD, no keys)
- No token minting (pure policy evaluation only)
- No audit logging enforcement (auditConfigs accepted but not enforced)
- No large built-in permission database (define what you need via custom roles)

**For complete details, see [CHANGELOG.md](CHANGELOG.md) and [docs/ARCHITECTURE.md](ARCHITECTURE.md).**
