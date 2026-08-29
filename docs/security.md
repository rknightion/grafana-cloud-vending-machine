---
title: Security
description: Supply-chain verification, credential handling, and the Retain-by-default lifecycle with platform-authorized deletion
---

# Security

## Supply-chain controls

The Grafana provider manifest (`platform/provider/provider-grafana.yaml`):

- Pins an immutable OCI digest, carried identically in spec.package and in the verification job's argv.
- Verifies Grafana's keyless signature against the exact publishing workflow identity, using a
  PreSync Argo CD hook Job running Cosign 3.1.2 at an immutable digest. The pinned build comes from
  main rather than a tag, so that identity ends `refs/heads/main` and is satisfied by any main build;
  the digest is what pins this artifact.
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

## External-resource lifecycle

This reference contains no one-command destructive path. The request field
`spec.lifecycle.externalResources` defaults to `Retain`: ordinary pruning removes the Kubernetes
composite and composed objects but **orphans** external resources. Stack-local Grafana content
remains retain/orphan because deleting the Stack destroys that content.

The platform-owned Composition input controls the exceptional `Delete` mode:

- `allowedUsages` must contain the immutable `spec.usage`; the reference vocabulary is
  `development` and `production`.
- `deletionAuthorizations` binds permission to an exact request namespace, name, Kubernetes UID, and immutable
  profile and is empty by default. Selecting a profile cannot authorize a consumer's request.
- An authorized `Delete` value is first an intent change. `status.deletionArmed=true` confirms the
  intent was accepted; `status.deletionReady=true` is required before Stage 2 and means observed
  provider state reports the Stack's `deleteProtection=false` while ESO reports current successful
  sync and has installed its deletion finalizer on every enabled credential document. Stage 3 follows only
  after Stage 2 access-claim finalizers have cleared.

The decommission therefore has three reviewed stages. Stage 1 changes the request to `Delete` and
waits for readiness. Stage 2 removes the dependent access claims, merges or syncs that change, and
waits until their Kubernetes objects and finalizers are gone while the Stack and request still
exist. Stage 3 removes the request from Git. Armed deletion affects only state that can outlive the Stack:
the Stack, administrator service account and token, telemetry access policy and token, and
administrator/telemetry `PushSecret` documents. It does not turn stack-local content into
independently deleted resources.

For AWS Secrets Manager, deleting a `PushSecret` target defaults to a 30-day recovery window, and
the supplied IAM policy permits `DeleteSecret` only for output documents carrying the function's
stable `grafana-cloud-vending-machine: managed` tag. A platform operator using another secret
backend must verify that its `PushSecret` implementation supports Delete before authorizing the
workflow. See the [decommission runbook](https://github.com/rknightion/grafana-cloud-vending-machine#decommission-runbook)
for the ordered procedure.

## Public-release scanning

`just check` runs `scripts/public-release-scan.sh`, which scans the working tree and
reachable Git history for source identifiers, credential prefixes, private keys, local paths,
private endpoints, account IDs, JWT-like values, Kubernetes Secret manifests, sensitive file
names, and tracked archives/key containers. Run `just public-release-scan` before making a fork public — see
[Troubleshooting](troubleshooting.md#running-the-validation-gate).

This scan covers reachable local Git objects; it cannot inspect deleted remote refs or external
artefacts no longer present in a checkout. Before making a repository public, also review its
GitHub settings, issues, workflow logs, releases, packages, and commit-author metadata by hand.

## Known limitations

- The Grafana provider is experimental and may lag the Terraform provider it is generated from.
- Provider schemas and Grafana APIs may expose fields that do not round-trip cleanly — test drift
  rather than assuming it behaves correctly.
- The reference has no one-command destructive workflow by design; the explicit, platform-authorized
  Delete lifecycle still requires three reviewed stages and a readiness wait.
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
