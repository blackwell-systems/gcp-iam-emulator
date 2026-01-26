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

```bash
# Start on default port 8080
server

# Custom port
server --port 9090
```

### Use with GCP SDK

```go
package main

import (
    "context"

    iampb "google.golang.org/genproto/googleapis/iam/v1"
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"
)

func main() {
    ctx := context.Background()

    conn, _ := grpc.NewClient(
        "localhost:8080",
        grpc.WithTransportCredentials(insecure.NewCredentials()),
    )
    defer conn.Close()

    client := iampb.NewIAMPolicyClient(conn)

    // Set policy
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

    _, _ = client.SetIamPolicy(ctx, &iampb.SetIamPolicyRequest{
        Resource: "projects/my-project/secrets/my-secret",
        Policy:   policy,
    })

    // Test permissions
    resp, _ := client.TestIamPermissions(ctx, &iampb.TestIamPermissionsRequest{
        Resource: "projects/my-project/secrets/my-secret",
        Permissions: []string{
            "secretmanager.versions.access",
            "secretmanager.secrets.delete",
        },
    })

    // resp.Permissions = ["secretmanager.versions.access"]
    // (delete denied - secretAccessor role doesn't include it)
}
```

## Use Cases

- **Security Testing** - Validate IAM policies in CI without GCP
- **Permission Debugging** - Test "who can access what" locally
- **Integration Testing** - Real permission evaluation with emulators
- **Policy Development** - Iterate on IAM policies without cloud deployment
- **Cost Reduction** - Avoid GCP API charges during development

## Integration with Other Emulators

The IAM emulator provides the auth layer for other GCP emulators:

```yaml
# docker-compose.yml
services:
  iam:
    image: gcp-iam-emulator:latest
    ports:
      - "8080:8080"
  
  secret-manager:
    image: gcp-secret-manager-emulator:latest
    environment:
      IAM_EMULATOR_HOST: iam:8080
    ports:
      - "9090:9090"
  
  kms:
    image: gcp-kms-emulator:latest
    environment:
      IAM_EMULATOR_HOST: iam:8080
    ports:
      - "9091:9090"
```

## Architecture

**In-memory policy storage** with thread-safe concurrent access. **Simple permission engine** mapping roles to permissions. **Resource-level policies** (no organization/folder hierarchy in MVP). **No token minting** (pure policy evaluation only).

## Limitations (MVP)

- No custom roles (primitive + service roles only)
- No conditional role bindings
- No organization/folder hierarchy
- No service accounts or token minting (coming soon)
- No audit logging
- No permission inheritance

**Current scope:** Core IAM policy operations for testing emulator integrations

## Maintained By

Maintained by **Dayna Blackwell** — founder of Blackwell Systems, building reference infrastructure for cloud-native development.

[GitHub](https://github.com/blackwell-systems) · [LinkedIn](https://linkedin.com/in/daynablackwell) · [Blog](https://blog.blackwell-systems.com)

## Related Projects

- [GCP Secret Manager Emulator](https://github.com/blackwell-systems/gcp-secret-manager-emulator) - Reference implementation for Secret Manager API
- [GCP KMS Emulator](https://github.com/blackwell-systems/gcp-kms-emulator) - Reference implementation for KMS API

## License

Apache 2.0 - See [LICENSE](LICENSE) for details.
