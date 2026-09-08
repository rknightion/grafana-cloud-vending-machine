---
title: Security
description: Supply-chain verification, credential handling, and the Retain-by-default lifecycle with platform-authorized deletion
---

# Security

## Supply-chain controls

The Grafana provider manifest (`platform/provider/provider-grafana.yaml`):

- Pins an immutable OCI digest, carried identically in spec.package and in the verification job's argv.
- Verifies Grafana's keyless signature against the exact publishing workflow identity, using a
  PreSync Argo CD hook Job running Cosign 3.1.2 at an immutable digest. The identity is scoped to
  `refs/tags/v2.14.0`, and the package reference also pins the verified artifact digest.
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
log. See [Secrets](secrets.md) for the per-organization credential and rotating-token model.
Key properties:

- Organization credentials are supplied and rotated by the environment's credential owner; this
  repository does not set their lifetime or automate their rotation. Generated per-stack
  administrator, telemetry, and Fleet Management tokens use a platform-capped lifetime (30-day standard) and a bounded early
  rotation window.
- Static `StackServiceAccountToken`, `AccessPolicyToken`, and `ServiceAccountToken` resources
  remain available in the upstream provider but are deliberately not used — their rotating
  counterparts avoid a permanent credential lifecycle outside the control plane.
- OAuth client secrets and SAML key material are always `LocalSecretKeySelector`/Secret
  references, never literal fields copied into a managed resource.
- AWS access should come from workload identity (EKS Pod Identity, IRSA, or the equivalent for
  your platform), never long-lived keys in the `SecretStore`.
## Secret and token design

No Grafana credential belongs in Git, a request object, Composition input, status, or function log.

### Organization credential

Create a Grafana Cloud access policy token with only the organization-level capabilities needed to manage stacks and their Cloud resources. Store it in the external secret manager as JSON:

~~~json
{
  "cloud_access_policy_token": "REPLACE_SECURELY"
}
~~~

The example expects one document per registry entry at `/platform/grafana-cloud/organizations/<organization>/credentials`. In every namespace that accepts requests, the environment overlay maps each path to that organization's same-named Secret and ProviderConfig. Do not put the real token directly in a shell command, terminal history, CI variable dump, or Kubernetes manifest. Use your secret-management workflow or the AWS CLI file input mechanism from a permission-restricted temporary file.

The environment overlay materializes each organization credential into a separate Kubernetes Secret
formatted for the provider. Each namespaced organization ProviderConfig references only its own
Secret. Repeat those resources for every request namespace, keeping the registry name identical;
namespace RBAC should prevent request authors from reading any of them.

### Rotating administrator token

For every stack, the Composition creates:

1. a StackServiceAccount with Admin role;
2. a StackServiceAccountRotatingToken after Grafana reports the service-account ID;
3. a Kubernetes connection Secret written by the rotating-token resource;
4. a PushSecret that exports a structured administrator document;
5. an ExternalSecret that reads the exported token and URL back into the stack namespace;
6. a stack-local ProviderConfig used for Grafana resources inside that stack.

The platform Composition supplies a mandatory maximum token lifetime. The standard 30-day lifetime is capped at that ceiling, with a seven-day early rotation window shortened as needed. Missing or invalid policy fails closed. Provider-observed expiries are published in `status.tokenExpiries`. The PushSecret refresh interval is one hour, so a newly rotated token is copied to the external store well inside the overlap window.

The exported document has this shape:

~~~json
{
  "stack_name": "Example stack",
  "stack_slug": "example",
  "stack_url": "https://example.grafana.net",
  "stack_region": "prod-us-central-0",
  "usage": "development",
  "change_reference": "CHANGE-EXAMPLE",
  "configuration_item_reference": "CONFIG-EXAMPLE",
  "stack_service_account_token": "GENERATED",
  "telemetry_access_policy_secret_path": "{outputSecretPrefix}/{organization}/{usage}/{slug}/telemetry-publisher"
}
~~~

The generated token comes from the connection Secret at reconciliation time; it is not embedded in rendered YAML.

### Rotating telemetry token

When telemetryAccess.enabled is true, the Composition creates a stack-realm AccessPolicy with only:

- stacks:read
- metrics:write
- logs:write
- traces:write

An AccessPolicyRotatingToken uses the same 30-day lifetime and seven-day early rotation window. A
separate PushSecret publishes the token and policy metadata under
`{outputSecretPrefix}/{organization}/{usage}/{slug}/telemetry-publisher`. The immutable,
platform-approved organization and usage segments keep this identity stable. Workloads should use
this token for telemetry and never receive the administrator token.

Static StackServiceAccountToken, AccessPolicyToken, and ServiceAccountToken resources remain available in the upstream provider but are deliberately not used. Their rotating counterparts avoid creating a permanent credential lifecycle outside the control plane.

### AWS permissions

deploy/aws/iam-policy.json is the minimum policy shape for the example paths. Attach it to the external-secrets ServiceAccount through the workload-identity mechanism for your cluster:

- EKS Pod Identity: create a Pod Identity association for namespace external-secrets and ServiceAccount external-secrets.
- IRSA: annotate that ServiceAccount with its role and use the standard EKS OIDC trust relationship.
- Other Kubernetes platforms: use the cloud identity integration recommended for that platform.

Do not put long-lived AWS keys in SecretStore. The controller should obtain short-lived credentials from workload identity. Restrict CreateSecret, PutSecretValue, and TagResource to the output prefix; allow DeleteSecret only for output documents carrying the function's stable `grafana-cloud-vending-machine: managed` tag; restrict read access to the input and output paths actually required.

The example uses AWS Secrets Manager, not Systems Manager Parameter Store. The function itself only emits SecretStore references, so another ESO provider can be substituted if it supports ExternalSecret and PushSecret with the required structured-value behavior.

`PushSecret` uses retain behaviour by default, so removing an unarmed request does not delete its
external credential documents. When an authorized request is armed with
`spec.lifecycle.externalResources: Delete`, the administrator and telemetry `PushSecret` documents
are deleted during Stage 3 of the separately reviewed request-removal sequence, after Stage 2 has
cleared the access claims. With AWS Secrets Manager, ESO's
deletion defaults to a 30-day recovery window; the supplied IAM policy includes tag-conditioned
`DeleteSecret` on the output prefix. Operators using another backend must verify that its
`PushSecret` implementation supports Delete before authorizing this lifecycle.


## External-resource lifecycle

This reference contains no one-command destructive path. The request field
`spec.lifecycle.externalResources` defaults to `Retain`: ordinary pruning removes the Kubernetes
composite and composed objects but **orphans** external resources. Stack-local Grafana content
remains retain/orphan because deleting the Stack destroys that content.

The platform-owned Composition input controls the exceptional `Delete` mode:

- `allowedUsages` must contain the immutable `spec.usage`; the reference vocabulary is
  `development` and `production`.
- `spec.organization` is immutable and must resolve to a registry entry whose ProviderConfig,
  allowed regions, and allowed usages match the request. There is no organization fallback.
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
workflow. See the [decommission runbook](governance.md#decommission-runbook)
for the ordered procedure.

## Decommission runbook

The default `spec.lifecycle.externalResources: Retain` is safe for ordinary pruning: removing an
unarmed request from Git or pruning it from Argo orphans external resources. Stack-local Grafana
content is safe to orphan because deleting the Stack destroys it; credential-bearing state that can
outlive the Stack is the state covered by the optional Delete path.

An actual deletion has three reviewed Git stages. It is not a one-command path. Approved sandbox expiry can delay Stage 1 arming until the effective deadline; it never executes Stages 2 or 3. See [Governance](governance.md#sandbox-expiry-uses-the-reviewed-deletion-path).

### Review 1: arm deletion

1. Inventory and record the exact stack identity (`status.stack.id`, slug, and URL), dependants,
   access claims, credential consumers, and data-retention requirements. Verify the creation-time
   `spec.retention.class` decision and actual receipt at its durable fan-out sink; decommission cannot
   recover telemetry that was never forwarded.
2. Have the platform owner add this request's exact namespace, name, Kubernetes UID, and immutable profile to the
   platform-owned `deletionAuthorizations` list. If there is no exact match, stop; the request must
   remain `Retain`.
3. Change only the request lifecycle intent to
   `spec.lifecycle.externalResources: Delete` and submit that change for review. Arming is intent
   only; it does not delete anything.
4. After reconciliation, confirm `status.deletionArmed=true`. Wait for
   `status.deletionReady=true`; this means observed provider state reports the Stack's
   `deleteProtection=false` and ESO has installed its deletion finalizer and successfully synced
   each enabled external credential document at the current generation. Do not proceed on desired
   specs or a stale Ready condition alone. New access claims fail closed as soon as deletion is
   armed; already-observed access children remain managed until Stage 2 removes their claims.

### Review 2: remove access claims

1. In Stage 2, remove all dependent `GrafanaCustomRoleBinding`,
   `GrafanaTeamAccess`, and `GrafanaContentAccessPolicy` objects. Merge or sync this change and
   wait until each access-claim Kubernetes object and its finalizer are gone while the Stack and its
   request still exist. Do not remove the request until this check passes.

### Review 3: remove the stack request

1. In a third reviewed change, remove the `GrafanaCloudStackRequest` from Git only after the
   dependent access claims and their finalizers have cleared.
2. Let Crossplane and ESO reconcile the armed deletion. The Delete mode covers only the Stack, its
   administrator service account and token, the telemetry access policy and token, and the
   administrator and telemetry `PushSecret` documents. Stack-local Grafana content remains
   retain/orphan because Stack deletion destroys it.
3. Verify the external deletion result and the credential-document outcome. AWS Secrets Manager
   deletion defaults to a 30-day recovery window, and the supplied IAM policy includes
   tag-conditioned `DeleteSecret` on the output prefix. A platform operator using another secret backend must
   verify its `PushSecret` Delete support before enabling this workflow.

This repository intentionally contains no one-command destructive path. If the request is pruned
without first being armed and ready, the default Retain behaviour applies and external resources
are orphaned.

## Public-release scanning

`just check` runs `scripts/public-release-scan.sh`, which scans the working tree and
reachable Git history for source identifiers, credential prefixes, private keys, local paths,
private endpoints, account IDs, JWT-like values, Kubernetes Secret manifests, sensitive file
names, and tracked archives/key containers. Run `just public-release-scan` before making a fork public — see
[Troubleshooting](troubleshooting.md#running-the-validation-gate).

This scan covers reachable local Git objects; it cannot inspect deleted remote refs or external
artefacts no longer present in a checkout. Before making a repository public, also review its
GitHub settings, issues, workflow logs, releases, packages, and commit-author metadata by hand.

## Token model and network allow-list

The administrator token and the telemetry token have separate audiences and permissions. The
administrator service-account token is used by the stack-local provider; workloads receive only the
least-privilege telemetry token with `stacks:read`, `metrics:write`, `logs:write`, and `traces:write`.
Fleet Management has its own derived credential. Static token resources remain unused so that
rotation stays inside the control-plane lifecycle. See [Secrets](secrets.md) for the complete
External Secrets flow.

Platform policy may attach `allowedSubnets` conditions to Fleet Management and telemetry access
policies through an immutable profile. A missing profile subnet list emits no conditions, while an
explicitly empty list is rejected so a mistaken restriction cannot silently become unrestricted.
Request authors cannot pass CIDRs through the stack API. These conditions restrict where those
tokens may be used from; they do not filter inbound traffic to a Grafana stack, and the
administrator service-account token has no equivalent condition field.

The Crossplane chart already grants its controller cluster-wide Secret access through its aggregate
role. The product credential handoffs reuse that capability; this reference adds no duplicate
core-Secret grant. Namespaced request and Secret identity checks constrain this function's
selection, but do not provide hard namespace RBAC isolation for the Crossplane controller. A
deployment requiring that boundary needs a separately designed control-plane isolation model.

## Known limitations

- The Grafana provider is experimental and may lag the Terraform provider it is generated from.
- The upstream Synthetic Monitoring Installation resource can report Ready/Synced without
  configuring the product. The module therefore requires an independently observed disabled
  Check through the derived credential; full bootstrap and zero-execution behaviour still require
  deployed validation.
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
- This reference supports Grafana Cloud only, not self-managed Grafana.
- Adaptive Metrics, Logs, Traces, and Profiles are deliberately out of scope; the API exposes no
  adaptive-product configuration.
- The pinned provider requires the optional Role `autoIncrementVersion` field because of an
  initializer defect; this reference pins it to `false` and omits the server-managed version.
- AccessPolicy `realm` is a Block List at the pinned build where an earlier provider build emitted a
  Block Set. This reference emits exactly one realm entry; a multi-realm policy must treat order as
  significant.
- Stack status allow-list URL fields are endpoint references for retrieving source IP addresses to
  allow, not a means of restricting inbound access to a stack.
- Datasource LBAC requires Grafana 11.5 or later, a Cloud or Enterprise entitlement, basic auth,
  and governance of inherited, fixed, or independently managed grants that can bypass its rules.
- Fleet usage groups are UI-only and Advanced-tier. Agent Observability plugin availability and
  permissions, and Assistant terms, remain environment prerequisites.

## Next steps

- [Secrets](secrets.md) — the per-organization credential and rotating-token model.
- [Architecture](architecture.md) — the ownership boundaries that keep controllers from stepping
  on each other.

See [Governance](governance.md) for token-use subnet restrictions, bounded product APIs and enforced SCIM exclusion. Direct provider writes and Composition policy updates must remain platform-only.

The pinned Crossplane chart already grants its controller cluster-wide Secret access through `crossplane:system:aggregate-to-crossplane`. The product credential handoffs reuse that existing controller capability; this reference adds no duplicate core-Secret grant. Namespaced request and Secret identity checks constrain this function's selection, but do not provide hard namespace RBAC isolation for the Crossplane controller. A deployment requiring that boundary needs a separately designed control-plane isolation model.
