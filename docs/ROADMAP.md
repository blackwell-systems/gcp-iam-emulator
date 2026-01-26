# Roadmap

This document outlines the strategic direction for the GCP IAM Emulator.

## Strategic Vision

The IAM emulator is the **keystone auth layer** for the Blackwell GCP emulator ecosystem. The goal is not to implement every IAM feature, but to provide the minimal, highest-ROI features that make local GCP testing realistic for teams.

**Philosophy:** Real teams need **policy evaluation correctness**, not full IAM identity management. We focus on the leverage points that unlock CI/CD adoption.

---

## v0.2.0 - Drop-in CI Ready (Next Release)

**Theme:** Make it useful in real CI pipelines with minimal client changes.

### 1. Principal Injection
- Accept caller identity via gRPC metadata (`x-emulator-principal`)
- Support `user:email@example.com` and `serviceAccount:name@project.iam.gserviceaccount.com`
- Document CI patterns for identity injection
- Match principals against `policy.bindings[].members`

**Impact:** TestIamPermissions now checks WHO is calling, not just WHAT permissions exist.

### 2. Policy Model Correctness
- **Resource hierarchy resolution**: walk up resource paths (secret → project)
- **Binding-based evaluation**: match principal to binding members
- **Member matching**: parse and match `user:` and `serviceAccount:` prefixes

**Impact:** Policy evaluation behaves like real GCP (inheritance, binding scopes).

### 3. Policy Configuration File
- YAML config file defining project policies and resource overrides
- `--config policy.yaml` flag to load at startup
- `--watch` flag for hot reload on file changes
- **Export command**: dump current policies to YAML for debugging

**Impact:** Teams can commit policy files to git, not just call SetIamPolicy APIs.

### 4. Trace Mode
- `--trace` flag enables decision logging
- Output format: `[ALLOW/DENY] principal=X resource=Y permission=Z reason="matched binding: role=R"`
- Log why permissions were denied (missing binding, wrong role, etc.)

**Impact:** "Why was this denied?" debugging without GCP's opaque errors.

### 5. Drop-in Launch
- Simplified CLI: `gcp-iam-emulator start --config policy.yaml --port 8080 --trace`
- Docker one-liner: `docker run -p 8080:8080 -v $(pwd)/policy.yaml:/policy.yaml gcp-iam-emulator`
- README section: "Drop-in CI Setup" with GitHub Actions + Docker Compose examples

**Impact:** Zero-friction adoption for existing teams.

---

## v0.3.0 - Emulator Suite Integration

**Theme:** Become the shared auth brain for Secret Manager + KMS emulators.

### Integration Surface
- Secret Manager + KMS emulators call IAM emulator for authz checks
- Shared principal format across all emulators
- Single policy file controls access to all resources
- Integration examples: docker-compose with full stack

**Impact:** "Blackwell GCP local stack" - coordinated emulator suite.

### Additional Permissions
- Expand permission mappings for full Secret Manager + KMS coverage
- Add resourcemanager permissions (projects.get, projects.list)
- Document permission-to-method mapping clearly

**Impact:** Complete coverage of existing emulator operations.

---

## v0.4.0 - Advanced Policy Features

**Theme:** Handle real-world policy patterns teams actually use.

### Conditional Bindings (Simplified)
- Support `condition.expression` with minimal CEL subset
- Common patterns: time-based (`request.time`), resource-based (`resource.name`)
- No full CEL interpreter - just the 20% that covers 80% of use cases

### Workload Identity Patterns
- Support `principalSet://` member format for workload identity pools
- Pattern matching for federated identities
- Document workload identity setup for Kubernetes

### Policy Export / Import
- Export current state to Terraform format
- Import from GCP (via gcloud or Terraform state)

**Impact:** Bridge between local development and production GCP policies.

---

## v1.0.0 - Production-Ready

**Theme:** Rock-solid foundation for long-term adoption.

### Stability
- Comprehensive integration test suite
- Performance benchmarks (100k+ policy checks/sec target)
- Documented compatibility guarantees
- Security audit of policy evaluation logic

### Observability
- Prometheus metrics (authz decisions, deny rates, latency)
- OpenTelemetry tracing integration
- Audit log export (JSON lines format)

### Documentation
- Architecture deep-dive
- Policy design patterns guide
- Migration guide from GCP IAM to emulator config
- Troubleshooting runbook

**Impact:** Enterprise-grade reliability and supportability.

---

## Out of Scope

These features are explicitly NOT planned:

### Service Account Management
- No service account creation/deletion
- No key generation or token minting
- **Rationale:** Teams use ADC/workload identity in CI; emulator just needs to accept the identity, not mint it.

### Organization / Folder Hierarchy
- No org-level or folder-level policies
- **Rationale:** Most CI tests operate at project scope; org policies are for production governance.

### Custom Roles
- No custom role definitions beyond config file role-to-permission mappings
- **Rationale:** Predefined roles + config overrides cover 95% of testing needs.

### IAM Admin API
- No `CreateRole`, `UpdateRole`, `CreateServiceAccount`
- **Rationale:** Policy evaluation is the core value, not IAM CRUD.

---

## Feature Requests

The roadmap prioritizes **drop-in adoption** and **correctness** over breadth. Feature requests are evaluated on:

1. **CI/CD impact**: Does this unblock real teams from using the emulator in CI?
2. **Policy correctness**: Does this make permission evaluation more accurate?
3. **Integration value**: Does this enable better coordination with other emulators?
4. **Complexity ratio**: Is the implementation complexity justified by adoption impact?

If you need a feature not on this roadmap, open a GitHub issue explaining your use case.

---

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines on proposing roadmap changes.

**Current focus:** v0.2.0 milestone (principal injection + policy inheritance + config file + trace mode).
