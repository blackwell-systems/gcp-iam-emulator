# Emulator Integration Guide

This document defines the integration contract for GCP emulators to use IAM Emulator as a shared authorization layer.

## TL;DR

```
IAM_MODE=off (default)      → Legacy behavior, all requests allowed
IAM_MODE=permissive         → Enforce IAM when available, fail-open on connectivity issues
IAM_MODE=strict             → Enforce IAM always, fail-closed (recommended for CI)
```

**The built-in roles stay small by design. The permission universe is defined by your policy file via custom roles.**

---

## Vision

**One policy file, one principal injection method, consistent resource naming, and every emulator enforces auth the same way.**

The IAM Emulator becomes the keystone of a coherent local GCP emulator mesh where:
- Authorization is centralized and deterministic
- Policy is defined once and enforced everywhere
- Conditional access works across all services
- Integration tests mirror production behavior

## Non-Breaking Integration

**IAM integration is completely opt-in and backwards compatible.**

Existing emulators currently have no authentication/authorization - all requests succeed. This behavior is preserved by default when IAM integration is added.

**Key principles:**
- `IAM_MODE=off` by default (current behavior maintained)
- Existing deployments continue working without changes
- New users opt-in by setting `IAM_MODE=permissive` or `IAM_MODE=strict`
- No code changes required for existing users
- Clear migration path for gradual adoption

**Versioning:**
- Secret Manager v1.x: No IAM integration
- Secret Manager v2.0+: IAM integration available (opt-in via IAM_MODE)
- KMS v1.x: No IAM integration  
- KMS v2.0+: IAM integration available (opt-in via IAM_MODE)

Major version bump signals new optional feature, but existing behavior is default.

---

## Integration Contract

### 1. Resource Naming Standard

All emulators must use consistent resource naming that matches GCP conventions.

#### Canonical Forms

**Secret Manager:**
```
projects/{project}/secrets/{secret}
projects/{project}/secrets/{secret}/versions/{version}
```

**KMS:**
```
projects/{project}/locations/{location}/keyRings/{keyring}
projects/{project}/locations/{location}/keyRings/{keyring}/cryptoKeys/{key}
projects/{project}/locations/{location}/keyRings/{keyring}/cryptoKeys/{key}/cryptoKeyVersions/{version}
```

**Future Services:**
```
projects/{project}/[locations/{location}/].../{resource-type}/{resource-id}
```

#### Resource Normalization

Emulators must normalize incoming requests to canonical resource strings before IAM checks:

```go
// Secret Manager normalization
func NormalizeSecretResource(name string) string {
    // Input: "projects/test/secrets/db-password/versions/1"
    // Output: "projects/test/secrets/db-password"
    parts := strings.Split(name, "/")
    if len(parts) >= 4 {
        return strings.Join(parts[:4], "/") // projects/{p}/secrets/{s}
    }
    return name
}

func NormalizeSecretVersionResource(name string) string {
    // Input: "projects/test/secrets/db-password/versions/latest"
    // Output: "projects/test/secrets/db-password/versions/latest" (preserve alias)
    // Keep full path for version-specific permissions
    // Note: Alias resolution (latest → numeric) happens in emulator logic, not normalization
    return name
}
```

**Why normalization matters:**
- Requests come in different shapes (aliases, partial paths)
- IAM policies reference canonical forms
- Normalization happens once before IAM check
- Enables nonbreaking evolution of request formats

**Why this matters:**
- Enables cross-service conditional policies
- `resource.name.startsWith("projects/prod/secrets/")` works consistently
- Resource hierarchy inheritance works correctly
- Matches production GCP behavior

---

### 2. Permission Mapping

Each emulator must map operations to standard GCP permissions.

#### Permission Mapping Table

**Centralize the mapping** in one place per emulator (e.g., `internal/authz/permissions.go`):

```go
// Permission mapping for Secret Manager operations
var operationPermissions = map[string]PermissionCheck{
    "CreateSecret": {
        Permission: "secretmanager.secrets.create",
        Resource:   ResourceParent, // Check against parent project
    },
    "GetSecret": {
        Permission: "secretmanager.secrets.get",
        Resource:   ResourceTarget, // Check against secret itself
    },
    "AccessSecretVersion": {
        Permission: "secretmanager.versions.access",
        Resource:   ResourceTarget, // Check against version path
    },
    // ...
}
```

#### Secret Manager Operations
| Operation | Permission | Check Against |
|-----------|-----------|---------------|
| CreateSecret | `secretmanager.secrets.create` | Parent (`projects/{project}`) |
| GetSecret | `secretmanager.secrets.get` | Secret (`projects/{p}/secrets/{s}`) |
| UpdateSecret | `secretmanager.secrets.update` | Secret |
| DeleteSecret | `secretmanager.secrets.delete` | Secret |
| ListSecrets | `secretmanager.secrets.list` | Parent |
| AddSecretVersion | `secretmanager.versions.add` | Secret |
| AccessSecretVersion | `secretmanager.versions.access` | Version (`projects/{p}/secrets/{s}/versions/{v}`) |
| GetSecretVersion | `secretmanager.versions.get` | Version |
| ListSecretVersions | `secretmanager.versions.list` | Secret |
| EnableSecretVersion | `secretmanager.versions.enable` | Version |
| DisableSecretVersion | `secretmanager.versions.disable` | Version |
| DestroySecretVersion | `secretmanager.versions.destroy` | Version |

#### KMS Operations
| Operation | Permission | Check Against |
|-----------|-----------|---------------|
| CreateKeyRing | `cloudkms.keyRings.create` | Parent (`projects/{p}/locations/{l}`) |
| GetKeyRing | `cloudkms.keyRings.get` | KeyRing |
| ListKeyRings | `cloudkms.keyRings.list` | Parent |
| CreateCryptoKey | `cloudkms.cryptoKeys.create` | KeyRing (parent) |
| GetCryptoKey | `cloudkms.cryptoKeys.get` | CryptoKey |
| ListCryptoKeys | `cloudkms.cryptoKeys.list` | KeyRing |
| UpdateCryptoKey | `cloudkms.cryptoKeys.update` | CryptoKey |
| Encrypt | `cloudkms.cryptoKeys.encrypt` | CryptoKey |
| Decrypt | `cloudkms.cryptoKeys.decrypt` | CryptoKey |
| CreateCryptoKeyVersion | `cloudkms.cryptoKeyVersions.create` | CryptoKey (parent) |
| GetCryptoKeyVersion | `cloudkms.cryptoKeyVersions.get` | CryptoKeyVersion |
| ListCryptoKeyVersions | `cloudkms.cryptoKeyVersions.list` | CryptoKey |
| UpdateCryptoKeyVersion | `cloudkms.cryptoKeyVersions.update` | CryptoKeyVersion |
| DestroyCryptoKeyVersion | `cloudkms.cryptoKeyVersions.destroy` | CryptoKeyVersion |

#### Create Operation Semantics

**Important:** Create operations check permissions against the **parent resource**, not the resource being created (which doesn't exist yet).

```go
// CreateSecret checks against parent
resource := req.Parent // "projects/test-project"
permission := "secretmanager.secrets.create"

// GetSecret checks against the secret itself
resource := req.Name // "projects/test-project/secrets/db-password"
permission := "secretmanager.secrets.get"
```

**Consistency rule:** Use GCP's exact permission names. If implementing a new service, check GCP's official permission list first.

---

### 3. Principal Injection

All emulators must support the same principal injection mechanism. **Principal flows end-to-end through a single metadata channel.**

#### Inbound: Extract Principal from Request

**gRPC:**
```go
func extractPrincipal(ctx context.Context) string {
    md, ok := metadata.FromIncomingContext(ctx)
    if ok {
        if principals := md.Get("x-emulator-principal"); len(principals) > 0 {
            return principals[0]
        }
    }
    return ""
}
```

**HTTP/REST:**
```go
func extractPrincipal(r *http.Request) string {
    return r.Header.Get("X-Emulator-Principal")
}
```

#### Outbound: Propagate Principal to IAM Emulator

**Do not pass principal as a function parameter.** Instead, inject it into the outbound context:

```go
func (c *IAMClient) CheckPermission(
    ctx context.Context,
    principal string,
    resource string,
    permission string,
) (bool, error) {
    // Inject principal into outbound context
    if principal != "" {
        ctx = metadata.AppendToOutgoingContext(ctx, "x-emulator-principal", principal)
    }
    
    // IAM emulator extracts principal from metadata (not from request fields)
    resp, err := c.client.TestIamPermissions(ctx, &iampb.TestIamPermissionsRequest{
        Resource:    resource,
        Permissions: []string{permission},
    })
    if err != nil {
        return false, err
    }
    
    return len(resp.Permissions) == 1, nil
}
```

**Why propagate via metadata:**
- Single identity channel end-to-end
- No principal parameter in IAM API (matches GCP behavior)
- Consistent with how emulator receives principal
- Enables transparent pass-through in proxies/gateways

#### Supported Principal Formats
- `user:email@example.com`
- `serviceAccount:name@project.iam.gserviceaccount.com`
- `group:groupname` (if groups configured in IAM)
- `allUsers`
- `allAuthenticatedUsers`

#### Default Behavior
- **IAM disabled:** No principal check (all requests succeed)
- **IAM enabled, no principal:** Behavior depends on mode (see error handling)
- **IAM enabled, principal provided:** Full authorization check

---

### 4. IAM Client Implementation

Each emulator needs an IAM client with proper timeout, caching, and failure mode handling.

#### Authorization Modes

```go
type AuthMode string

const (
    // AuthModeOff: No IAM checks (legacy behavior, default)
    AuthModeOff AuthMode = "off"
    
    // AuthModePermissive: Call IAM when available; fail-open on connectivity issues
    // Use for development where IAM might not be running yet
    AuthModePermissive AuthMode = "permissive"
    
    // AuthModeStrict: Require IAM checks; fail-closed on any error
    // Recommended for CI/CD to catch permission issues
    AuthModeStrict AuthMode = "strict"
)

// Parse from environment variable
func ParseAuthMode(env string) AuthMode {
    switch strings.ToLower(env) {
    case "permissive":
        return AuthModePermissive
    case "strict":
        return AuthModeStrict
    default:
        return AuthModeOff
    }
}
```

#### Client Implementation

```go
type IAMClient struct {
    client  iampb.IAMPolicyClient
    timeout time.Duration
    mode    AuthMode
}

func NewIAMClient(host string, mode AuthMode) (*IAMClient, error) {
    conn, err := grpc.NewClient(
        host,
        grpc.WithTransportCredentials(insecure.NewCredentials()),
    )
    if err != nil {
        return nil, err
    }
    
    return &IAMClient{
        client:  iampb.NewIAMPolicyClient(conn),
        timeout: 2 * time.Second, // Short timeout
        mode:    mode,
    }, nil
}

func (c *IAMClient) CheckPermission(
    ctx context.Context,
    principal string,
    resource string,
    permission string,
) (bool, error) {
    // Inject principal into outbound metadata
    if principal != "" {
        ctx = metadata.AppendToOutgoingContext(ctx, "x-emulator-principal", principal)
    }
    
    // Apply timeout
    ctx, cancel := context.WithTimeout(ctx, c.timeout)
    defer cancel()
    
    resp, err := c.client.TestIamPermissions(ctx, &iampb.TestIamPermissionsRequest{
        Resource:    resource,
        Permissions: []string{permission},
    })
    
    if err != nil {
        // Classify error type
        if isConnectivityError(err) {
            // IAM emulator unreachable/timeout
            if c.mode == AuthModePermissive {
                // Fail-open: allow on connectivity issues
                return true, nil
            }
            // Strict mode: fail-closed on connectivity issues
            return false, err
        }
        
        // Bad request/config error: always deny (both modes)
        // This indicates emulator misconfiguration that should be fixed
        return false, err
    }
    
    return len(resp.Permissions) == 1, nil
}

func isConnectivityError(err error) bool {
    // Classify as connectivity if:
    // - Unavailable (connection refused)
    // - DeadlineExceeded (timeout)
    // - Canceled (context cancelled)
    code := status.Code(err)
    return code == codes.Unavailable || 
           code == codes.DeadlineExceeded || 
           code == codes.Canceled
}
```

#### Optional: Decision Caching

For high-throughput scenarios, add short-lived caching:

```go
type cacheKey struct {
    principal  string
    resource   string
    permission string
}

type IAMClient struct {
    client  iampb.IAMPolicyClient
    timeout time.Duration
    mode    AuthMode
    cache   *sync.Map // map[cacheKey]cacheEntry
}

type cacheEntry struct {
    allowed bool
    expires time.Time
}

func (c *IAMClient) CheckPermission(
    ctx context.Context,
    principal string,
    resource string,
    permission string,
) (bool, error) {
    // Check cache first (5-second TTL)
    key := cacheKey{principal, resource, permission}
    if entry, ok := c.cache.Load(key); ok {
        if e := entry.(cacheEntry); time.Now().Before(e.expires) {
            return e.allowed, nil
        }
    }
    
    // ... (rest of check logic)
    
    // Cache result
    c.cache.Store(key, cacheEntry{
        allowed: allowed,
        expires: time.Now().Add(5 * time.Second),
    })
    
    return allowed, nil
}
```

**Caching considerations:**
- Tiny TTL (5 seconds max)
- Optional optimization (not required for MVP)
- Bounded cache size (LRU eviction)
- Clear on policy updates (if watch is implemented)

#### Integration Pattern (Opt-in, Non-Breaking)
```go
// In each RPC handler:
func (s *Server) GetSecret(ctx context.Context, req *pb.GetSecretRequest) (*pb.Secret, error) {
    // 1. Check if IAM is enabled (default: false)
    if s.iamEnabled {
        // 2. Extract principal from incoming request
        principal := extractPrincipal(ctx)
        
        // 3. Build resource name
        resource := req.Name // e.g., "projects/test/secrets/db-password"
        
        // 4. Check permission
        allowed, err := s.iamClient.CheckPermission(ctx, principal, resource, "secretmanager.secrets.get")
        if err != nil {
            return nil, status.Error(codes.Internal, "IAM check failed")
        }
        if !allowed {
            return nil, status.Error(codes.PermissionDenied, "Permission denied")
        }
    }
    
    // 5. Proceed with operation (works with or without IAM)
    return s.storage.GetSecret(req.Name)
}
```

**Server initialization:**
```go
type Server struct {
    storage    *storage.Storage
    iamEnabled bool
    iamClient  *IAMClient
}

func NewServer() (*Server, error) {
    s := &Server{
        storage: storage.NewStorage(),
    }
    
    // Parse IAM mode from environment
    iamMode := ParseAuthMode(os.Getenv("IAM_MODE"))
    
    // Only connect to IAM if not in "off" mode
    if iamMode != AuthModeOff {
        iamHost := os.Getenv("IAM_EMULATOR_HOST")
        if iamHost == "" {
            iamHost = "localhost:8080"
        }
        
        client, err := NewIAMClient(iamHost, iamMode)
        if err != nil {
            return nil, fmt.Errorf("failed to connect to IAM emulator: %w", err)
        }
        s.iamClient = client
        s.iamMode = iamMode
    }
    
    return s, nil
}
```

---

### 5. Environment Variables

Standardized environment variables across all emulators:

| Variable | Purpose | Default | Values |
|----------|---------|---------|--------|
| `IAM_MODE` | Authorization mode | `off` | `off`, `permissive`, `strict` |
| `IAM_EMULATOR_HOST` | IAM emulator gRPC endpoint | `localhost:8080` | `host:port` |
| `IAM_TRACE` | Enable IAM decision logging | `false` | `true`, `false` |

**Important: IAM integration is opt-in and non-breaking.**

#### Mode Behaviors

**Off (default - legacy behavior):**
```bash
# No IAM checks - all requests succeed
server
# or explicitly:
IAM_MODE=off server
```

**Permissive (development mode):**
```bash
# Check IAM when available, allow on connectivity issues
IAM_MODE=permissive IAM_EMULATOR_HOST=iam:8080 server
```
- IAM reachable → enforce permissions
- IAM unreachable → allow (fail-open)
- Bad config → deny (prevents masking bugs)

**Strict (CI/production mode):**
```bash
# Require IAM checks, deny on any error
IAM_MODE=strict IAM_EMULATOR_HOST=iam:8080 server
```
- IAM reachable → enforce permissions
- IAM unreachable → deny (fail-closed)
- Bad config → deny

**Migration path:**
1. Existing users: no changes needed (`IAM_MODE=off` by default)
2. Development: opt-in with `IAM_MODE=permissive`
3. CI/CD: use `IAM_MODE=strict` to catch permission issues

---

### 6. Error Handling

Errors from IAM checks fall into three categories with different behaviors:

#### 1. Permission Denied (Normal Authorization Denial)

When IAM successfully evaluates and denies permission:

**gRPC:**
```go
status.Error(codes.PermissionDenied, "Permission denied")
```

**HTTP:**
```
403 Forbidden
```

**Behavior:** Always deny in all modes (this is the normal authorization flow).

#### 2. Connectivity Errors (IAM Unreachable)

When IAM emulator is unreachable, timeout, or connection refused:

**Error codes:**
- `codes.Unavailable` (connection refused)
- `codes.DeadlineExceeded` (timeout)
- `codes.Canceled` (context cancelled)

**Behavior:**
- **Permissive mode:** Allow (fail-open, minimizes disruption during development)
- **Strict mode:** Deny (fail-closed, security-first for CI)

#### 3. Bad Request / Configuration Errors

When IAM returns error due to misconfiguration:
- Invalid resource format
- Malformed permission string
- Internal IAM error
- Bad policy syntax

**Behavior:** Always deny in both modes (indicates a bug/config issue that must be fixed).

**Implementation pattern:**
```go
if err != nil {
    if isConnectivityError(err) {
        // Handle based on mode (fail-open vs fail-closed)
        if mode == AuthModePermissive {
            return true, nil
        }
        return false, err
    }
    // All other errors: deny (config/bug)
    return false, err
}
```

---

### 7. Integration Contract Checklist

**Every integrated emulator MUST:**

- [ ] Canonical resource naming (projects/{project}/...)
- [ ] Operation → permission mapping documented in README
- [ ] Principal extraction (gRPC `x-emulator-principal` + HTTP `X-Emulator-Principal`)
- [ ] Principal propagation to IAM emulator via metadata
- [ ] Three IAM modes supported: off/permissive/strict
- [ ] Environment variables: `IAM_MODE` and `IAM_HOST`
- [ ] Uses `gcp-emulator-auth` shared library
- [ ] Resource normalization functions (parent vs self)
- [ ] Permission check before each operation
- [ ] Error responses (403 PermissionDenied, 500 Internal)
- [ ] Unit tests with IAM_MODE=off (default behavior)
- [ ] Integration tests with IAM emulator running
- [ ] Integration tests for permissive vs strict modes
- [ ] README with IAM integration section
- [ ] Permission reference table (operation → permission → resource)
- [ ] Docker Compose integration example
- [ ] Backward compatible (IAM_MODE=off as default)
- [ ] Non-breaking for existing users

**Recommended (but not required):**

- [ ] Short timeout for IAM calls (2 seconds)
- [ ] Cache IAM decisions with tiny TTL (5 seconds)
- [ ] Log IAM decisions when `IAM_TRACE=true`
- [ ] Provide health check endpoint
- [ ] Document resource naming conventions

---

## Mesh Semantics

When IAM integration is enabled, understand how authorization decisions flow:

### Operation-Level Behavior

**When IAM is enabled (`IAM_MODE=permissive` or `IAM_MODE=strict`):**

1. **Request arrives** at emulator (Secret Manager, KMS, etc.)
2. **Principal extracted** from `x-emulator-principal` metadata/header
3. **Resource normalized** to canonical form
4. **Permission mapped** from operation (e.g., GetSecret → `secretmanager.secrets.get`)
5. **IAM check** via TestIamPermissions (with principal in outbound metadata)
6. **Authorization decision:**
   - Allowed → proceed with operation
   - Denied → return `PermissionDenied` error
   - IAM error → behavior depends on mode (fail-open vs fail-closed)

### Principal Behavior

| Scenario | Auth Mode Off | Auth Mode Permissive | Auth Mode Strict |
|----------|---------------|---------------------|------------------|
| No principal provided | Allow (legacy) | Allow | Deny |
| Principal provided | Allow (legacy) | Check IAM, allow on error | Check IAM, deny on error |
| IAM unavailable | Allow (legacy) | Allow (fail-open) | Deny (fail-closed) |

### Error Scenarios

**Missing principal + IAM enabled:**
- **Permissive:** Allow (legacy compatible, useful for gradual migration)
- **Strict:** Deny with `PermissionDenied`

**IAM emulator unreachable (connectivity error):**
- **Permissive:** Allow (fail-open, minimizes disruption during development)
- **Strict:** Deny (fail-closed, security-first for CI)

**IAM returns configuration error (invalid resource, bad policy syntax):**
- **Both modes:** Deny (indicates bug/misconfiguration that must be fixed)

### Why This Matters

**Prevents surprises:**
- "Why did my CI suddenly start denying requests?" → Check `IAM_MODE` and principal injection
- "Why are all requests allowed despite policy?" → Check IAM mode and connectivity
- "IAM is down but tests still pass?" → You're in permissive mode (expected)
- "IAM config error but tests pass?" → Both modes deny config errors (check logs)

**Clear contract:**
- `off` = legacy (no checks, default)
- `permissive` = best-effort (allow on connectivity issues, deny on config errors)
- `strict` = security-first (deny on any error, recommended for CI)

---

## Docker Compose Stack

Complete working example:

```yaml
services:
  # IAM Emulator - authorization engine
  iam:
    image: ghcr.io/blackwell-systems/gcp-iam-emulator:latest
    ports:
      - "8080:8080"
    volumes:
      - ./policy.yaml:/policy.yaml
    command: --config /policy.yaml --trace
  
  # Secret Manager with IAM (strict mode for CI)
  secret-manager:
    image: ghcr.io/blackwell-systems/gcp-secret-manager-emulator:latest
    environment:
      IAM_MODE: strict              # Fail-closed (recommended for CI)
      IAM_EMULATOR_HOST: iam:8080
    ports:
      - "9090:9090"
    depends_on:
      - iam
  
  # KMS with IAM (permissive mode for development)
  kms:
    image: ghcr.io/blackwell-systems/gcp-kms-emulator:latest
    environment:
      IAM_MODE: permissive         # Fail-open (useful during development)
      IAM_EMULATOR_HOST: iam:8080
    ports:
      - "9091:9090"
    depends_on:
      - iam
  
  # Example: Secret Manager without IAM (legacy behavior)
  secret-manager-legacy:
    image: ghcr.io/blackwell-systems/gcp-secret-manager-emulator:latest
    # No IAM_MODE set - defaults to "off"
    # All requests succeed (backwards compatible)
    ports:
      - "9092:9090"
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

### Usage Patterns

**Option 1: Copy/Paste (immediate, no implementation needed)**

Users copy pack contents into their policy.yaml:

```yaml
# policy.yaml
# Paste from packs/secretmanager.yaml
roles:
  roles/secretmanager.admin:
    permissions:
      - secretmanager.secrets.create
      - secretmanager.secrets.get
      # ...

projects:
  test-project:
    bindings:
      - role: roles/secretmanager.admin
        members:
          - group:developers
```

**Option 2: Directory Loading (future feature)**

Load multiple YAML files and merge:

```bash
# Load all files in packs/ plus main policy
server --config-dir ./packs --config policy.yaml
```

**Option 3: YAML Anchors (native YAML feature)**

```yaml
# packs/secretmanager.yaml
secretmanager_roles: &secretmanager
  roles/secretmanager.admin:
    permissions: [...]

# policy.yaml
roles:
  <<: *secretmanager
  roles/custom.myRole:
    permissions: [...]
```

**Recommendation:** Start with copy/paste (works immediately). Implement directory loading later if demand exists.

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

### Shared Auth Package (Prevents Drift)
- [ ] Create `github.com/blackwell-systems/gcp-emulator-auth` module
- [ ] Principal extractors (gRPC + HTTP)
- [ ] Environment parsing (`IAM_MODE`, `IAM_EMULATOR_HOST`)
- [ ] Minimal IAM client with timeout and modes
- [ ] Error classification (connectivity vs config)
- [ ] Common test helpers

**Why separate package:**
- Prevents copy/paste drift across emulators
- No monorepo coupling (standard Go module)
- Easy to version independently
- Shared by all emulators (Secret Manager, KMS, future services)

### Secret Manager Emulator
- [ ] Import `gcp-emulator-auth` package
- [ ] Resource normalization functions
- [ ] Permission mapping table
- [ ] `IAM_MODE` support (`off`, `permissive`, `strict`)
- [ ] Integration tests with IAM emulator
- [ ] Docker Compose example

### KMS Emulator
- [ ] Import `gcp-emulator-auth` package
- [ ] Resource normalization functions
- [ ] Permission mapping table
- [ ] `IAM_MODE` support (`off`, `permissive`, `strict`)
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
