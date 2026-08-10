---
title: Security
description: Supply-chain verification, credential handling, and the non-destructive lifecycle this reference enforces
---

# Security

## Supply-chain controls

The Grafana provider manifest (`platform/provider/provider-grafana.yaml`):

- Pins the v2.13.0 OCI digest.
- Verifies Grafana's keyless signature against the exact tag workflow identity, using a
  PreSync Argo CD hook Job running Cosign 3.1.2 at an immutable digest.
- Runs the provider with `--safe-start`.
- Activates only the managed-resource kinds used by this reference, through a
  `ManagedResourceActivationPolicy`.

The repository's function workflow:

1. Tidies and checks the Go module.
2. Runs race-enabled tests and `go vet`.
3. Builds amd64 and arm64 distroless images from pinned bases.
4. Assembles a multi-platform Crossplane package.
5. Publishes an immutable commit-derived version.
6. Signs the OCI index with keyless Cosign.

`platform/function/install.yaml` pins the signed digest and verifies it against this
repository's exact `main`-branch workflow identity before Crossplane installs it — the same
PreSync-hook pattern as the provider. The verification Job name contains the digest prefix, so
changing the digest creates a new gate rather than reusing an old successful Job.

Forking this repository means changing the package repository and the expected Cosign workflow
identity in `install.yaml`. If the package is private, provide a dedicated read-only registry
credential through an external secret — do not commit a Docker config or reuse a developer
token.

## Credential handling

No Grafana credential belongs in Git, a request object, Composition input, status, or function
log. See [Secrets](secrets.md) for the full organization-credential and rotating-token model.
Key properties:

- The organization credential and every generated per-stack token are rotated automatically
  (30-day lifetime, 7-day early rotation window) rather than issued as static, indefinite-lived
  values.
- Static `StackServiceAccountToken`, `AccessPolicyToken`, and `ServiceAccountToken` resources
  remain available in the upstream provider but are deliberately not used — their rotating
  counterparts avoid a permanent credential lifecycle outside the control plane.
- OAuth client secrets and SAML key material are always `LocalSecretKeySelector`/Secret
  references, never literal fields copied into a managed resource.
- AWS access should come from workload identity (EKS Pod Identity, IRSA, or the equivalent for
  your platform), never long-lived keys in the `SecretStore`.

## Non-destructive lifecycle

This reference contains no one-command destructive path. Deletion protection and the omission of
provider `Delete` permissions are deliberate:

- The `Stack` sets `deleteProtection`.
- Every composed managed resource omits the `Delete` management policy.
- Rotating tokens set `deleteOnDestroy: false`.
- `PushSecret` uses `deletionPolicy: None`.

Pruning a request from Git deletes the Kubernetes composite and composed objects but **orphans**
the external Grafana stack and its credential documents rather than destroying them. A
destructive decommission must be a separately reviewed operation — see the decommission runbook
in the project
[README](https://github.com/rknightion/grafana-cloud-vending-machine#decommission-runbook), which
covers resolving the exact stack identity, revoking credentials, and confirming external deletion
before removing the request from Git.

## Public-release scanning

`scripts/validate.sh` runs `scripts/public-release-scan.sh`, which scans the working tree and
reachable Git history for source identifiers, credential prefixes, private keys, local paths,
private endpoints, account IDs, JWT-like values, Kubernetes Secret manifests, sensitive file
names, and tracked archives/key containers. Run it before making a fork public — see
[Troubleshooting](troubleshooting.md#running-the-validation-gate).

This scan covers reachable local Git objects; it cannot inspect deleted remote refs or external
artefacts no longer present in a checkout. Before making a repository public, also review its
GitHub settings, issues, workflow logs, releases, packages, and commit-author metadata by hand.

## Known limitations

- The Grafana provider is experimental and may lag the Terraform provider it is generated from.
- Provider schemas and Grafana APIs may expose fields that do not round-trip cleanly — test drift
  rather than assuming it behaves correctly.
- The reference has no destructive workflow by design.
- It does not provision Kubernetes, AWS infrastructure, DNS, identity providers, or incident
  relays.
- It uses generic starter dashboards rather than a full observability content library.
- Report, Enterprise, OnCall, plugin, and other resources require the relevant Grafana Cloud
  capabilities and entitlements.
- SSO profiles shipped in this repository are examples and must be replaced with reviewed
  identity settings.
- Built-in Viewer, Editor, and Admin basic-role definitions cannot be globally rewritten through
  the current provider — use SSO mapping, fixed/custom roles, and content ACLs instead.
- Fixed-role UIDs and available RBAC actions vary by Grafana version, edition, and entitlement —
  inventory and test them before assignment.
- `FolderPermission`, `DashboardPermission`, `RoleAssignment`, `NotificationPolicy`, and similar
  whole-set APIs need exactly one declarative owner per external target.
- Crossplane readiness reports only the child resources currently desired — an optional disabled
  domain is not health-checked.
- Secret-store publication is eventually consistent with the configured ESO refresh interval.
- A successful render or unit test does not prove acceptance by a specific Grafana Cloud region
  or account — use a disposable stack for live acceptance.

## Next steps

- [Secrets](secrets.md) — the organization credential and rotating-token model.
- [Architecture](architecture.md) — the ownership boundaries that keep controllers from stepping
  on each other.
