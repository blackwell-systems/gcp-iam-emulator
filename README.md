# GCP IAM Emulator

[![Blackwell Systems](https://raw.githubusercontent.com/blackwell-systems/blackwell-docs-theme/main/badge-trademark.svg)](https://github.com/blackwell-systems)
[![Go Reference](https://pkg.go.dev/badge/github.com/blackwell-systems/gcp-iam-emulator.svg)](https://pkg.go.dev/github.com/blackwell-systems/gcp-iam-emulator)
[![Go Version](https://img.shields.io/badge/go-1.24+-blue.svg)](https://go.dev/)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)

> The reference local implementation of the Google Cloud IAM API for development and CI

A production-grade IAM policy engine providing complete, behaviorally-accurate permission evaluation for local development and CI/CD. The missing auth layer for GCP emulators. No GCP credentials or network connectivity required.

## Features

- **Complete IAM Policy API** - SetIamPolicy, GetIamPolicy, TestIamPermissions
- **Real Permission Evaluation** - Accurate role-to-permission mapping
- **Primitive Roles** - owner, editor, viewer support
- **Service-Specific Roles** - Secret Manager, KMS, and more
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
- `roles/editor` - Read/write access (no IAM management)
- `roles/viewer` - Read-only access

**Secret Manager roles:**
- `roles/secretmanager.admin` - Full secret management
- `roles/secretmanager.secretAccessor` - Read secret values only

**KMS roles:**
- `roles/cloudkms.admin` - Full KMS management
- `roles/cloudkms.cryptoKeyEncrypterDecrypter` - Encrypt/decrypt only

### Supported Permissions

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

## Architecture

**In-memory policy storage** with thread-safe concurrent access. **Simple permission engine** mapping roles to permissions. **Resource-level policies** (no organization/folder hierarchy in MVP). **No token minting** (pure policy evaluation only).

## Roadmap

**v0.2.0 (Next Release) - Drop-in CI Ready:**
- Principal injection via gRPC metadata (`x-emulator-principal`)
- Policy inheritance (resource hierarchy resolution)
- YAML config file with hot reload
- Trace mode (explain authz decisions)
- Drop-in launch for CI/CD

**v0.3.0 - Emulator Suite Integration:**
- Secret Manager + KMS integration (shared auth layer)
- Complete permission mappings
- Docker Compose full-stack examples

**v1.0.0 - Production-Ready:**
- Enterprise stability + observability
- Performance benchmarks (100k+ authz/sec)
- Comprehensive documentation

See [ROADMAP.md](docs/ROADMAP.md) for full details.

## Limitations (MVP)

- No principal checking yet (v0.2.0 adds this)
- No policy inheritance yet (v0.2.0 adds this)
- No custom roles (primitive + service roles only)
- No conditional role bindings
- No organization/folder hierarchy
- No service accounts or token minting
- No audit logging

**Current scope:** Core IAM policy operations for testing emulator integrations

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
