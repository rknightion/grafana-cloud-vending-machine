---
title: Configuration
description: The GrafanaCloudStackRequest and access API fields, their defaults, and what each controls
---

# Configuration

!!! warning "`platform.example.org` is a documentation placeholder"
    Every field below is shown under the API group `platform.example.org`, which ships in this
    repository's XRDs, Compositions, examples, and Argo CD health configuration. It is not a
    production API group — replace it everywhere in the XRDs (`spec.group`), the Compositions
    (`spec.compositeTypeRef.apiVersion` and the Composition input `apiVersion`), the examples,
    and your own request manifests before adopting this API in production. See
    [Public base, private environment overlay](#public-base-private-environment-overlay) below
    for how to keep a public reference and a private production fork in sync.

This is the field-by-field reference for the four custom resources this repository defines. It
is the companion to the worked examples under `examples/catalog/` — see
[Reference → Catalog](reference/catalog.md).

## `GrafanaCloudStackRequest`

The primary API, at `platform.example.org/v1beta1`. It is namespaced so teams or environments can
be separated with Kubernetes namespaces and RBAC.

| Field | Default | Description |
| --- | --- | --- |
| `spec.displayName` | required | Human-readable Grafana Cloud stack name (1-128 chars). |
| `spec.slug` | required | Immutable Grafana Cloud stack slug and external identity. Must equal `metadata.name`, and must match `^[a-z0-9]+$` (3-32 chars). Grafana Cloud stack slugs are globally unique. |
| `spec.region` | required | Grafana Cloud region slug, must match `^prod-[a-z0-9-]+-[0-9]+$` (e.g. `prod-us-central-0`). |
| `spec.usage` | required | Generic usage classification and output-secret path segment. Must match `^[a-z0-9-]+$` (1-32 chars). |
| `spec.profile` | `standard` | Platform-owned configuration profile selected by the request. |
| `spec.changeReference` | `""` | Optional generic request/change identifier published in the output document (max 128 chars). |
| `spec.configurationItemReference` | `""` | Optional generic configuration-item identifier published in the output document (max 128 chars). |
| `spec.baselineDashboards.enabled` | `true` | Creates three baseline folder/dashboard pairs (billing/usage, telemetry endpoints, stack home). |
| `spec.telemetryAccess.enabled` | `true` | Creates a stack-scoped, rotating telemetry publisher access policy and mirrors its token to a derived AWS Secrets Manager path. |
| `spec.plugins` | `[]` | List of `{slug, version}` plugin installations owned by this request. Max 30 items. `version` defaults to `latest`. |
| `spec.reconciliation.dashboards` | `createOnly` | One of `enforced`, `createOnly`. Controls whether Crossplane repairs later UI edits to baseline dashboard JSON. |
| `spec.reconciliation.homePreference` | `createOnly` | One of `enforced`, `createOnly`. Controls whether Crossplane repairs later UI changes to the organization home dashboard. |
| `spec.sso.mode` | `disabled` | One of `enforced`, `createOnly`, `observeOnly`, `disabled`. See [SSO](sso.md). |
| `spec.sso.profile` | `""` | Selects a platform-owned SSO profile and Secret reference. Required (non-empty) unless `mode: disabled`. |
| `spec.monthlyReport.enabled` | `false` | Enables a scheduled monthly usage report (requires an applicable Grafana Cloud plan and report capability). |
| `spec.monthlyReport.recipients` | `[]` | Report recipient email addresses, max 20. Required (non-empty) when `enabled: true`. |
| `spec.monthlyReport.replyTo` | `""` | Report reply-to address. Required (non-empty) when `enabled: true`. |
| `spec.incidentIntegration.enabled` | `false` | Creates four OnCall outgoing webhooks (test/production firing and resolved) and two Alerting contact points calling a platform-owned relay. |
| `spec.incidentIntegration.profile` | `""` | Selects a platform-owned relay URL and Secret reference. Required (non-empty) when `enabled: true`. |
| `spec.incidentIntegration.templateMode` | `createOnly` | One of `enforced`, `createOnly`. Controls whether Crossplane repairs or preserves later OnCall outgoing-webhook data-template edits. |

### Validation rules

The XRD enforces these with CEL validation, rejected at admission time rather than left to the
composition function:

- `metadata.name` must equal `spec.slug`.
- `spec.slug` is immutable after creation.
- An enabled `monthlyReport` requires non-empty `recipients` and `replyTo`.
- `sso.mode` other than `disabled` requires a non-empty `sso.profile`.
- An enabled `incidentIntegration` requires a non-empty `incidentIntegration.profile`.

### Status fields

| Field | Description |
| --- | --- |
| `status.outputSecretPath` | The external path the generated administrator document was published to. |
| `status.telemetrySecretPath` | The external path the generated telemetry document was published to. |
| `status.stack.id` | The Grafana Cloud stack ID once observed. |
| `status.stack.url` | The stack's Grafana URL once observed. |

## Access APIs

Three additional APIs sit beside the stack request. All three reference a stack by
`spec.stackRef.name`, which must be the `GrafanaCloudStackRequest` name (and per-stack
`ProviderConfig` name) in the same namespace.

### `GrafanaCustomRoleBinding`

One Team, one custom Role, one whole-role `RoleAssignment`. A compact pattern for a unique custom
role bound to a directory-synced team.

| Field | Default | Description |
| --- | --- | --- |
| `spec.stackRef.name` | required | Target stack. |
| `spec.team.name` | required | Team display name (1-190 chars). |
| `spec.team.groups` | required | Identity-provider groups synced to this team, 1-50 entries. |
| `spec.role.name` | required | Custom role display name. |
| `spec.role.uid` | required | Custom role UID, must match `^[a-zA-Z0-9._-]+$`. |
| `spec.role.displayName` | `""` | Optional display name shown in Grafana. |
| `spec.role.description` | `""` | Optional description (max 1024 chars). |
| `spec.role.permissions` | required | List of `{action, scope}` pairs, 1-200 entries. `scope` is optional per entry. |

### `GrafanaTeamAccess`

One Team plus zero-to-many custom and fixed-role assignments. Use this for directory sync, direct
membership, team preferences, additive custom roles, and existing fixed roles.

| Field | Default | Description |
| --- | --- | --- |
| `spec.stackRef.name` | required | Target stack. |
| `spec.team.name` | required | Team display name (1-190 chars). |
| `spec.team.email` | `""` | Optional team email (max 254 chars). |
| `spec.team.members` | `[]` | Direct members by Grafana login email; each user must already exist. Max 100. |
| `spec.team.externalGroups` | `[]` | Identity-provider groups synchronized through Grafana Team Sync. Max 50. |
| `spec.team.ignoreExternallySyncedMembers` | `true` | When true, provider membership reconciliation ignores members supplied by Team Sync while still managing the direct member set declared in Git. |
| `spec.team.preferences.homeDashboardUid` | `""` | Team home dashboard UID (max 190 chars). |
| `spec.team.preferences.theme` | `""` | One of `""`, `light`, `dark`, `system`. |
| `spec.team.preferences.timezone` | `""` | One of `""`, `utc`, `browser`. |
| `spec.team.preferences.weekStart` | `""` | One of `""`, `sunday`, `monday`, `saturday`. |
| `spec.customRoles` | `[]` | List of custom role objects (same shape as `GrafanaCustomRoleBinding.spec.role`, plus optional `group` and `hidden`). Max 20, keyed by `uid`. |
| `spec.fixedRoleUids` | `[]` | Existing Grafana fixed-role UIDs assigned item-by-item, to avoid whole-role assignment collisions. Max 50. |

`GrafanaTeamAccess` uses `RoleAssignmentItem` rather than the whole-set `RoleAssignment`
resource, so independently owned team bundles are additive — see
[Architecture → reconciliation and ownership](architecture.md).

### `GrafanaContentAccessPolicy`

The complete ACL for one Folder or Dashboard. Only one policy may own a given target.

| Field | Default | Description |
| --- | --- | --- |
| `spec.stackRef.name` | required | Target stack. |
| `spec.target.kind` | required | `Folder` or `Dashboard`. |
| `spec.target.ref.name` | one of `name`/`uid` | Kubernetes name of a provider `Folder`/`Dashboard` managed resource. |
| `spec.target.ref.uid` | one of `name`/`uid` | Existing Grafana folder/dashboard UID when no managed-resource reference is available. |
| `spec.permissions` | required | List of grants, 1-100 entries. Each entry sets exactly one of `basicRole`, `teamRef`, or `userId`, plus a `permission` of `View`, `Edit`, or `Admin`. |

`GrafanaContentAccessPolicy` manages the entire ACL for its target; omitting an entry removes it
on the next enforced reconciliation. Put every grant for a target — including basic-role grants
that should remain — into the one policy that owns it.

## Platform configuration

The Composition input (a `GrafanaVendingConfig` object embedded in
`platform/apis/v1beta1.yaml`) is the platform-owned policy boundary. It controls:

| Field | Description |
| --- | --- |
| `organizationProviderConfigName` | The namespaced `ProviderConfig` used for organization-level Cloud operations. |
| `outputSecretPrefix` | The external path prefix for generated per-stack documents. |
| `secretStoreRef` | Either a namespaced `SecretStore` or a `ClusterSecretStore`. |
| `ssoProfiles` | Approved OAuth or SAML settings and Secret references. |
| `incidentProfiles` | Approved relay URLs and authorization Secret references. |

Consumers select a profile by name in `spec.sso.profile` or `spec.incidentIntegration.profile`.
They cannot supply an arbitrary identity endpoint, client secret, incident URL, or authorization
value directly in a stack request — see [SSO](sso.md).

## Public base, private environment overlay

The reusable implementation and a live environment have different publication boundaries:

| Public reference owns | Private environment owns |
| --- | --- |
| Provider and function packages at immutable digests | Approved public-reference Git commit |
| XRD schemas and Composition behaviour | Production API group under a controlled domain |
| Non-destructive lifecycle policy | Secret-store kind/name, cloud region, and workload identity |
| Placeholder SSO and incident profile shapes | Real endpoints and `ExternalSecret` remote paths |
| Comprehensive request with reserved example identities | Globally unique stack slug, intended recipients, user IDs, groups, and verified fixed-role UIDs |

An Argo CD `Application` can source `platform/` directly from this repository at an immutable
commit. Its Kustomize patches can replace the four XRD group-qualified names, each XRD
`spec.group`, and each Composition `spec.compositeTypeRef.apiVersion`; the stack Composition
input `apiVersion` must be patched as well. A second source in the same `Application` can point
at a small private directory containing only the environment `SecretStore`, `ExternalSecrets`,
and organization `ProviderConfig`. This keeps one Argo owner while avoiding a copied platform
implementation.

Do not put a patched live request in this public repository's `enabled/` directory — that
publishes a real cloud-resource identity and couples a production deployment to mutable public
data. Keep enabled requests or overlays in a private GitOps repository, and pin public platform
and catalog sources to reviewed commits.

## Next steps

- [Reference → Catalog](reference/catalog.md) — every worked example and what it demonstrates.
- [SSO](sso.md) — the four OAuth/SAML providers and the four reconciliation modes.
- [Architecture](architecture.md) — how these fields flow through Crossplane into managed
  resources.
