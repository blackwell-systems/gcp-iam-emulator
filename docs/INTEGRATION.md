# Emulator Integration Guide

This document defines the integration contract for GCP emulators to use IAM Emulator as a shared authorization layer.

## Vision

**One policy file, one principal injection method, consistent resource naming, and every emulator enforces auth the same way.**

The IAM Emulator becomes the keystone of a coherent local GCP emulator mesh where:
- Authorization is centralized and deterministic
- Policy is defined once and enforced everywhere
- Conditional access works across all services
- Integration tests mirror production behavior

---

## Integration Contract

### 1. Resource Naming Standard

All emulators must use consistent resource naming that matches GCP conventions:

#### Secret Manager
```
projects/{project}/secrets/{secret}
projects/{project}/secrets/{secret}/versions/{version}
```

#### KMS
```
projects/{project}/locations/{location}/keyRings/{keyring}
projects/{project}/locations/{location}/keyRings/{keyring}/cryptoKeys/{key}
projects/{project}/locations/{location}/keyRings/{keyring}/cryptoKeys/{key}/cryptoKeyVersions/{version}
```

#### Future Services
Follow GCP's canonical resource naming:
```
projects/{project}/[locations/{location}/].../{resource-type}/{resource-id}
```

**Why this matters:**
- Enables cross-service conditional policies
- `resource.name.startsWith("projects/prod/secrets/")` works consistently
- Resource hierarchy inheritance works correctly
- Matches production GCP behavior

---

### 2. Permission Mapping

Each emulator must map operations to standard GCP permissions:

#### Secret Manager Operations
| Operation | Permission |
|-----------|-----------|
| CreateSecret | `secretmanager.secrets.create` |
| GetSecret | `secretmanager.secrets.get` |
| UpdateSecret | `secretmanager.secrets.update` |
| DeleteSecret | `secretmanager.secrets.delete` |
| ListSecrets | `secretmanager.secrets.list` |
| AddSecretVersion | `secretmanager.versions.add` |
| AccessSecretVersion | `secretmanager.versions.access` |
| GetSecretVersion | `secretmanager.versions.get` |
| ListSecretVersions | `secretmanager.versions.list` |
| EnableSecretVersion | `secretmanager.versions.enable` |
| DisableSecretVersion | `secretmanager.versions.disable` |
| DestroySecretVersion | `secretmanager.versions.destroy` |

#### KMS Operations
| Operation | Permission |
|-----------|-----------|
| CreateKeyRing | `cloudkms.keyRings.create` |
| GetKeyRing | `cloudkms.keyRings.get` |
| ListKeyRings | `cloudkms.keyRings.list` |
| CreateCryptoKey | `cloudkms.cryptoKeys.create` |
| GetCryptoKey | `cloudkms.cryptoKeys.get` |
| ListCryptoKeys | `cloudkms.cryptoKeys.list` |
| UpdateCryptoKey | `cloudkms.cryptoKeys.update` |
| Encrypt | `cloudkms.cryptoKeys.encrypt` |
| Decrypt | `cloudkms.cryptoKeys.decrypt` |
| CreateCryptoKeyVersion | `cloudkms.cryptoKeyVersions.create` |
| GetCryptoKeyVersion | `cloudkms.cryptoKeyVersions.get` |
| ListCryptoKeyVersions | `cloudkms.cryptoKeyVersions.list` |
| UpdateCryptoKeyVersion | `cloudkms.cryptoKeyVersions.update` |
| DestroyCryptoKeyVersion | `cloudkms.cryptoKeyVersions.destroy` |

**Consistency rule:** Use GCP's exact permission names. If implementing a new service, check GCP's official permission list first.

---

### 3. Principal Injection

All emulators must support the same principal injection mechanism:

#### gRPC
Extract principal from metadata key: `x-emulator-principal`

```go
md, ok := metadata.FromIncomingContext(ctx)
if ok {
    if principals := md.Get("x-emulator-principal"); len(principals) > 0 {
        principal = principals[0]
    }
}
```

#### HTTP/REST
Extract principal from header: `X-Emulator-Principal`

```go
principal := r.Header.Get("X-Emulator-Principal")
```

#### Supported Principal Formats
- `user:email@example.com`
- `serviceAccount:name@project.iam.gserviceaccount.com`
- `group:groupname` (if groups configured in IAM)
- `allUsers`
- `allAuthenticatedUsers`

**Default behavior:** If no principal provided, deny by default (strict mode).

---

### 4. IAM Client Implementation

Each emulator needs an IAM client to check permissions:

#### Configuration
```go
// Environment variable
iamHost := os.Getenv("IAM_EMULATOR_HOST")
if iamHost == "" {
    iamHost = "localhost:8080" // default
}

// Connect to IAM emulator
conn, err := grpc.NewClient(
    iamHost,
    grpc.WithTransportCredentials(insecure.NewCredentials()),
)
```

#### Permission Check
```go
type IAMClient struct {
    client iampb.IAMPolicyClient
}

func (c *IAMClient) CheckPermission(
    ctx context.Context,
    principal string,
    resource string,
    permission string,
) (bool, error) {
    // Forward principal via metadata
    md := metadata.Pairs("x-emulator-principal", principal)
    ctx = metadata.NewOutgoingContext(ctx, md)
    
    resp, err := c.client.TestIamPermissions(ctx, &iampb.TestIamPermissionsRequest{
        Resource: resource,
        Permissions: []string{permission},
    })
    if err != nil {
        return false, err
    }
    
    return len(resp.Permissions) > 0, nil
}
```

#### Integration Pattern
```go
// In each RPC handler:
func (s *Server) GetSecret(ctx context.Context, req *pb.GetSecretRequest) (*pb.Secret, error) {
    // 1. Extract principal from incoming request
    principal := extractPrincipal(ctx)
    
    // 2. Build resource name
    resource := req.Name // e.g., "projects/test/secrets/db-password"
    
    // 3. Check permission
    allowed, err := s.iamClient.CheckPermission(ctx, principal, resource, "secretmanager.secrets.get")
    if err != nil {
        return nil, status.Error(codes.Internal, "IAM check failed")
    }
    if !allowed {
        return nil, status.Error(codes.PermissionDenied, "Permission denied")
    }
    
    // 4. Proceed with operation
    return s.storage.GetSecret(req.Name)
}
```

---

### 5. Environment Variables

Standardized environment variables across all emulators:

| Variable | Purpose | Default |
|----------|---------|---------|
| `IAM_EMULATOR_HOST` | IAM emulator gRPC endpoint | `localhost:8080` |
| `IAM_ENABLED` | Enable/disable IAM checks | `true` |
| `IAM_TRACE` | Enable IAM decision logging | `false` |

**Disable for testing:**
```bash
IAM_ENABLED=false server
```

This allows emulators to run standalone without IAM for simple testing.

---

### 6. Error Handling

#### Permission Denied
When IAM check fails, return standard gRPC error:
```go
status.Error(codes.PermissionDenied, "Permission denied")
```

HTTP equivalent: `403 Forbidden`

#### IAM Unavailable
When IAM emulator is unreachable:
- **Strict mode (default):** Fail closed (deny all requests)
- **Permissive mode (opt-in):** Fail open (allow all requests)

```go
if err != nil {
    if s.iamPermissive {
        // Fail open (allow)
        return true, nil
    }
    // Fail closed (deny)
    return false, err
}
```

---

## Docker Compose Stack

Complete working example:

```yaml
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
      IAM_ENABLED: "true"
    ports:
      - "9090:9090"
    depends_on:
      - iam
  
  kms:
    image: ghcr.io/blackwell-systems/gcp-kms-emulator:latest
    environment:
      IAM_EMULATOR_HOST: iam:8080
      IAM_ENABLED: "true"
    ports:
      - "9091:9090"
    depends_on:
      - iam
```

### Example Policy
```yaml
# policy.yaml
roles:
  roles/custom.ciRunner:
    permissions:
      - secretmanager.secrets.get
      - secretmanager.versions.access
      - cloudkms.cryptoKeys.encrypt
      - cloudkms.cryptoKeys.decrypt

groups:
  developers:
    members:
      - user:alice@example.com
      - user:bob@example.com

projects:
  test-project:
    bindings:
      # Developers get full access
      - role: roles/owner
        members:
          - group:developers
      
      # CI only gets specific permissions
      - role: roles/custom.ciRunner
        members:
          - serviceAccount:ci@test-project.iam.gserviceaccount.com
        condition:
          expression: 'resource.name.startsWith("projects/test-project/secrets/prod-")'
          title: "CI limited to production secrets"
      
      # KMS encrypt-only for backup service
      - role: roles/cloudkms.cryptoKeyEncrypterDecrypter
        members:
          - serviceAccount:backup@test-project.iam.gserviceaccount.com
```

### Usage
```bash
# Start stack
docker-compose up

# Test with principal injection
curl -X POST http://localhost:9090/v1/projects/test-project/secrets \
  -H "X-Emulator-Principal: user:alice@example.com" \
  -d '{"secretId": "my-secret", "payload": {"data": "c2VjcmV0"}}'

# Verify IAM traces
docker-compose logs iam
```

---

## Policy Packs (Optional)

Reusable policy definitions for common scenarios:

### packs/secretmanager.yaml
```yaml
roles:
  roles/secretmanager.admin:
    permissions:
      - secretmanager.secrets.create
      - secretmanager.secrets.get
      - secretmanager.secrets.update
      - secretmanager.secrets.delete
      - secretmanager.secrets.list
      - secretmanager.versions.add
      - secretmanager.versions.access
      - secretmanager.versions.get
      - secretmanager.versions.list
      - secretmanager.versions.enable
      - secretmanager.versions.disable
      - secretmanager.versions.destroy
  
  roles/secretmanager.secretAccessor:
    permissions:
      - secretmanager.versions.access
  
  roles/secretmanager.secretVersionManager:
    permissions:
      - secretmanager.versions.add
      - secretmanager.versions.get
      - secretmanager.versions.list
      - secretmanager.versions.enable
      - secretmanager.versions.disable
      - secretmanager.versions.destroy
```

### packs/kms.yaml
```yaml
roles:
  roles/cloudkms.admin:
    permissions:
      - cloudkms.keyRings.create
      - cloudkms.keyRings.get
      - cloudkms.keyRings.list
      - cloudkms.cryptoKeys.create
      - cloudkms.cryptoKeys.get
      - cloudkms.cryptoKeys.list
      - cloudkms.cryptoKeys.update
      - cloudkms.cryptoKeys.encrypt
      - cloudkms.cryptoKeys.decrypt
      - cloudkms.cryptoKeyVersions.create
      - cloudkms.cryptoKeyVersions.get
      - cloudkms.cryptoKeyVersions.list
      - cloudkms.cryptoKeyVersions.update
      - cloudkms.cryptoKeyVersions.destroy
  
  roles/cloudkms.cryptoKeyEncrypterDecrypter:
    permissions:
      - cloudkms.cryptoKeys.encrypt
      - cloudkms.cryptoKeys.decrypt
  
  roles/cloudkms.cryptoKeyEncrypter:
    permissions:
      - cloudkms.cryptoKeys.encrypt
  
  roles/cloudkms.cryptoKeyDecrypter:
    permissions:
      - cloudkms.cryptoKeys.decrypt
  
  roles/cloudkms.viewer:
    permissions:
      - cloudkms.keyRings.get
      - cloudkms.keyRings.list
      - cloudkms.cryptoKeys.get
      - cloudkms.cryptoKeys.list
      - cloudkms.cryptoKeyVersions.get
      - cloudkms.cryptoKeyVersions.list
```

### packs/ci.yaml
```yaml
groups:
  ci:
    members:
      - serviceAccount:ci@test-project.iam.gserviceaccount.com
      - serviceAccount:github-actions@test-project.iam.gserviceaccount.com
  
  developers:
    members:
      - user:alice@example.com
      - user:bob@example.com

roles:
  roles/custom.ciRunner:
    permissions:
      - secretmanager.secrets.get
      - secretmanager.versions.access
      - cloudkms.cryptoKeys.encrypt
      - cloudkms.cryptoKeys.decrypt
```

**Usage:**
```yaml
# Import packs (future feature)
imports:
  - packs/secretmanager.yaml
  - packs/kms.yaml
  - packs/ci.yaml

projects:
  test-project:
    bindings:
      - role: roles/secretmanager.admin
        members:
          - group:developers
```

---

## Cross-Emulator Integration Tests

Example integration test demonstrating the mesh:

```go
func TestCrossServiceAuthorization(t *testing.T) {
    // Start IAM emulator with policy
    iam := startIAM(t, "testdata/policy.yaml")
    defer iam.Stop()
    
    // Start Secret Manager with IAM integration
    sm := startSecretManager(t, iam.Endpoint())
    defer sm.Stop()
    
    // Start KMS with IAM integration
    kms := startKMS(t, iam.Endpoint())
    defer kms.Stop()
    
    // Test 1: Authorized principal can create secret
    ctx := withPrincipal(context.Background(), "user:alice@example.com")
    _, err := sm.CreateSecret(ctx, &pb.CreateSecretRequest{
        Parent:   "projects/test-project",
        SecretId: "db-password",
    })
    assert.NoError(t, err)
    
    // Test 2: Unauthorized principal cannot access secret
    ctx = withPrincipal(context.Background(), "user:bob@example.com")
    _, err = sm.AccessSecretVersion(ctx, &pb.AccessSecretVersionRequest{
        Name: "projects/test-project/secrets/db-password/versions/latest",
    })
    assert.Error(t, err)
    assert.Equal(t, codes.PermissionDenied, status.Code(err))
    
    // Test 3: Conditional access works
    ctx = withPrincipal(context.Background(), "serviceAccount:ci@test-project.iam.gserviceaccount.com")
    
    // Can access prod- secrets
    _, err = sm.AccessSecretVersion(ctx, &pb.AccessSecretVersionRequest{
        Name: "projects/test-project/secrets/prod-api-key/versions/latest",
    })
    assert.NoError(t, err)
    
    // Cannot access dev- secrets
    _, err = sm.AccessSecretVersion(ctx, &pb.AccessSecretVersionRequest{
        Name: "projects/test-project/secrets/dev-api-key/versions/latest",
    })
    assert.Error(t, err)
    
    // Test 4: KMS encrypt-only role
    ctx = withPrincipal(context.Background(), "serviceAccount:backup@test-project.iam.gserviceaccount.com")
    
    // Can encrypt
    _, err = kms.Encrypt(ctx, &pb.EncryptRequest{
        Name:      "projects/test-project/locations/global/keyRings/main/cryptoKeys/backup-key",
        Plaintext: []byte("data"),
    })
    assert.NoError(t, err)
    
    // Cannot decrypt
    _, err = kms.Decrypt(ctx, &pb.DecryptRequest{
        Name:       "projects/test-project/locations/global/keyRings/main/cryptoKeys/backup-key",
        Ciphertext: ciphertext,
    })
    assert.Error(t, err)
    assert.Equal(t, codes.PermissionDenied, status.Code(err))
}
```

---

## Implementation Checklist

### IAM Emulator (this repo)
- [x] Core IAM Policy API
- [x] Principal injection (gRPC + REST)
- [x] Custom roles system
- [x] Conditional bindings
- [x] Groups support
- [x] Strict mode
- [x] Trace mode
- [ ] Policy packs (optional imports)
- [ ] Integration examples repository

### Secret Manager Emulator
- [ ] IAM client implementation
- [ ] Principal extraction from metadata/headers
- [ ] Permission checks before operations
- [ ] `IAM_EMULATOR_HOST` environment variable
- [ ] Fail-closed error handling
- [ ] Integration tests with IAM emulator
- [ ] Docker Compose example

### KMS Emulator
- [ ] IAM client implementation
- [ ] Principal extraction from metadata/headers
- [ ] Permission checks before operations
- [ ] `IAM_EMULATOR_HOST` environment variable
- [ ] Fail-closed error handling
- [ ] Integration tests with IAM emulator
- [ ] Docker Compose example

### Documentation
- [x] Integration contract specification
- [ ] End-to-end tutorial
- [ ] Migration guide for existing deployments
- [ ] Policy examples repository
- [ ] Blog post: "Building a coherent GCP emulator mesh"

---

## Benefits

**For users:**
- Define authorization once, enforce everywhere
- Test complex multi-service scenarios locally
- Catch permission bugs before production
- Mirror production IAM behavior in CI

**For the ecosystem:**
- Unique differentiator (no other emulator suite does this)
- Composable infrastructure (mix and match services)
- Sustainable strategy (shared auth layer, not duplicated logic)
- Community can add new emulators following same contract

**Strategic positioning:**
- Not "three emulators", but "one coherent local GCP platform"
- IAM Emulator becomes foundational infrastructure
- Natural entry point for users ("I need IAM, oh it integrates with Secret Manager too")

---

## Future Services

When adding new emulators to the mesh:

1. **Define resource naming** (follow GCP conventions)
2. **Map operations to permissions** (use GCP's exact permission names)
3. **Add IAM client** (use standard implementation pattern)
4. **Support principal injection** (`x-emulator-principal` / `X-Emulator-Principal`)
5. **Write integration tests** (verify authz enforcement)
6. **Create policy pack** (optional, for convenience)

The contract is stable - new services just follow it.

---

## Questions?

Open an issue in any of the emulator repositories or start a discussion in:
- [gcp-iam-emulator](https://github.com/blackwell-systems/gcp-iam-emulator/discussions)
- [gcp-secret-manager-emulator](https://github.com/blackwell-systems/gcp-secret-manager-emulator/discussions)
- [gcp-kms-emulator](https://github.com/blackwell-systems/gcp-kms-emulator/discussions)
