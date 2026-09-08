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

This is the field-by-field reference for the custom resources this repository defines. It
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
| `spec.usage` | required | Immutable platform-approved usage classification and output-secret path segment. The reference vocabulary is `development` and `production`; it must pass both the platform-wide and selected organization's allowed-usage lists. |
| `spec.organization` | required | Immutable organization registry key. The entry selects the organization ProviderConfig and must allow this request's region and usage. |
| `spec.products.applicationObservability` | `false` | Activation toggle for the Application Observability global singleton; it does not configure the product. |
| `spec.products.kubernetesObservability` | `false` | Activation toggle for the Kubernetes Observability global singleton; configuration remains in Helm values. |
| `spec.products.databaseObservability` | `false` | Activation toggle for the Database Observability global singleton; configuration remains in its onboarding flow. |
| `spec.profile` | `standard` | Immutable platform configuration profile selected when the request is created. |
| `spec.lifecycle.externalResources` | `Retain` | `Retain` or `Delete`. `Delete` requires an exact namespace/name/UID/profile match in the platform input `deletionAuthorizations`; the default list is empty. |
| `spec.changeReference` | `""` | Optional generic request/change identifier published in the output document (max 128 chars). |
| `spec.configurationItemReference` | `""` | Optional generic configuration-item identifier published in the output document (max 128 chars). |
| `spec.baselineDashboards.enabled` | `true` | Creates three baseline folder/dashboard pairs (billing/usage, telemetry endpoints, stack home). |
| `spec.telemetryAccess.enabled` | `true` | Creates a stack-scoped, rotating telemetry publisher access policy and mirrors its token to `{outputSecretPrefix}/{organization}/{usage}/{slug}/telemetry-publisher`. |
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

The XRD enforces the schema and immutable-field rules below with CEL validation, rejecting them at
admission time rather than leaving them to the composition function:

- `metadata.name` must equal `spec.slug`.
- `spec.slug` is immutable after creation.
- `spec.usage`, `spec.organization`, and `spec.profile` are immutable after creation.
- An enabled `monthlyReport` requires non-empty `recipients` and `replyTo`.
- `sso.mode` other than `disabled` requires a non-empty `sso.profile`.
- An enabled `incidentIntegration` requires a non-empty `incidentIntegration.profile`.

The composition function resolves the organization registry and fails closed for an unknown
organization, a region absent from that organization's `allowedRegions`, or a usage absent from
that organization's `allowedUsages`. It also rejects `Delete` unless the request's
namespace, name, Kubernetes UID, and immutable profile match one platform-owned `deletionAuthorizations` entry.

`spec.usage` is part of external credential identity. Generated documents use the exact path
`{outputSecretPrefix}/{organization}/{usage}/{slug}`. Immutability prevents a request from
publishing future documents at a new organization or usage path while leaving previous credential
documents orphaned. A stack belongs to one organization; an installation may register several.

`spec.lifecycle.externalResources` defaults to `Retain`. Selecting `Delete` is a platform-governed
decommission intent, not a consumer-only switch: the exact request namespace, name, UID, and immutable profile
must be present in `deletionAuthorizations`, whose default is empty. Arming the intent does not delete resources; wait
for `status.deletionReady=true` before Stage 2 removes access claims and waits for their finalizers.
Stage 3 removes the request only after those objects are gone.

### Status fields

| Field | Description |
| --- | --- |
| `status.outputSecretPath` | The external path for the generated administrator document: `{outputSecretPrefix}/{organization}/{usage}/{slug}`. |
| `status.telemetrySecretPath` | The external path for the generated telemetry document: `{outputSecretPrefix}/{organization}/{usage}/{slug}/telemetry-publisher`. |
| `status.deletionArmed` | `true` after an authorized `Delete` intent has been accepted; arming alone does not delete resources. |
| `status.deletionReady` | `true` only after observed provider state reports the Stack's `deleteProtection=false` and ESO has installed its deletion finalizer and successfully synced every enabled credential PushSecret at its current generation; this gates Stage 2, while Stage 3 waits for access-claim objects and finalizers to be gone. |
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
membership and administration, team preferences, additive custom roles, and existing fixed roles.

| Field | Default | Description |
| --- | --- | --- |
| `spec.stackRef.name` | required | Target stack. |
| `spec.team.name` | required | Team display name (1-190 chars). |
| `spec.team.email` | `""` | Optional team email (max 254 chars). |
| `spec.team.members` | `[]` | Ordinary members by Grafana login email; each user must already exist. Max 100. |
| `spec.team.administrators` | omitted | Team administrators by Grafana login email; each user must already exist. Max 100. Setting the field claims ownership of team administration, so an empty list demotes every administrator including ones added in the UI. Omit it to leave existing administrators alone. |
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

## Specialist modules

These APIs are opt-in modules. All require `spec.stackRef.name` to name a Ready
`GrafanaCloudStackRequest` in the same namespace and route through its selected per-stack
ProviderConfig. None is inferred from stack creation.

### `GrafanaStackInventory`

An observe-only migration/adoption inventory. `spec.declared` names an API version/kind with an
external name, object name, email, or login selector. It publishes declared, managed, and unmanaged
classifications; it never emits mutating resources. It activates FolderSet, DashboardSet, TeamSet,
UserSet, LibraryPanelSet, ProbeSet, CollectorSet, and selected `OrganizationUser` lookups.

### `GrafanaFleetPipelines`

`spec.profile` selects a platform-owned Fleet pipeline profile; a request cannot supply pipeline
contents, matchers, labels, or attribution. The baseline enforces team, cost-centre, and environment
labels. It creates Pipelines, never Collectors because they self-register, and publishes the minted
Fleet credential as `fleet_management_auth` through the external secret store. Usage groups are
UI-only and Advanced-tier, so entitlement must be verified.

### `GrafanaAlertingBundle`

`spec.provenance` is required. `enforced` retains provisioning provenance and locks UI edits;
`createOnly` seeds values with disabled provenance and preserves later UI edits. The bundle owns
RuleGroup, ContactPoint, MuteTiming, MessageTemplate, and InhibitionRuleV1Beta1 and uses per-rule
`notificationSettings`. It never renders the organization-wide NotificationPolicy or
RoutingtreeV1Beta1 singleton. Bundle-prefixed names coexist with incident-relay contact points.

### `GrafanaAgentObservability`

`spec.guards` is platform-owned and renders HookRule/RuleAction; `spec.workload` is workload-owned
and renders Collection/Evaluator/EvaluationRule. The module never infers policy from a stack request,
installs the plugin, or mints credentials. RuleAction waits for a provider-assigned Collection ID;
plugin availability and permissions are environment prerequisites.

### `GrafanaAssistantGovernance`

`spec.termsAcceptance.accepted` gates Rules and MCPServers. They are withheld until acceptance is
observed and a false value withdraws them. Rule content is platform-selected; MCP tool approval
defaults to `always_ask`; custom headers are write-only data in a referenced Secret.

### `GrafanaDatasourceAccess`

One composite owns one DataSource, an authoritative whole-set DataSourcePermission, and aggregated
DataSourceConfigLbacRules. `metadata.name` must equal the immutable datasource UID; every team needs
its Grafana UID, numeric ID, and non-empty LBAC rules. Basic auth is required. Omitted managed,
non-inherited Query grants are removed, but inherited/fixed/custom/independent grants are additive
and can bypass LBAC. LBAC requires Grafana 11.5 or later plus Cloud or Enterprise entitlement; PDC
is out of scope and namespace uniqueness is not cluster-wide.

### `GrafanaProvisioningRepository`

This opt-in preview API owns a Git-provisioned folder subtree. It references a required,
separately credential-managed `ConnectionV0Alpha1` and never embeds secure data or creates that
connection. It rejects classic dashboard ownership for the subtree. Classic Crossplane Dashboard
remains the default until the preview matures. This route is Grafana Cloud only.

## Platform configuration

The Composition input (a `GrafanaVendingConfig` object embedded in
`platform/apis/stack-v1beta1.yaml`) is the platform-owned policy boundary. It controls:

| Field | Description |
| --- | --- |
| `organizations` | Registry entries keyed by organization name. Each supplies an organization ProviderConfig name, resolved in the request namespace, plus its allowed regions and usages. There is no fallback entry. |
| `allowedUsages` | Additional platform-wide usage allow-list. A request must pass this list as well as the selected organization's list; the reference values are `development` and `production`. |
| `outputSecretPrefix` | The external path prefix for generated per-stack documents. |
| `publicDashboardProfiles` | Platform-owned profiles allowed to retain the exact custom-role action `dashboards.public:write`; every other profile has that action stripped. Built-in Viewer/Editor/Admin roles are unchanged. |
| `deletionAuthorizations` | Platform-owned list of exact request `namespace`, `name`, Kubernetes `uid`, and immutable `profile` tuples authorized for `spec.lifecycle.externalResources: Delete`. It is empty by default; UID binding prevents an authorization from applying to a later request that reuses the same name. |
| `secretStoreRef` | Either a namespaced `SecretStore` or a `ClusterSecretStore`. |
| `fleetPipelineProfiles` | Required by `GrafanaFleetPipelines`. Each selectable profile supplies at least one matcher plus platform-owned `team`, `cost-centre`, and `environment` labels; the shipped `standard` profile is defined in `platform/apis/fleet-v1beta1.yaml`. Requests select a name and cannot supply pipeline contents or attribution values. |
| `ssoProfiles` | Approved OAuth or SAML settings and Secret references. |
| `incidentProfiles` | Approved relay URLs and authorization Secret references. |

Every organization-plane child uses the selected registry ProviderConfig in the request namespace.
The v2 ProviderConfig is namespaced, so every namespace allowed to contain requests must carry a
same-named credential Secret and ProviderConfig for each registry entry. Consumers select a
profile by name in `spec.sso.profile` or `spec.incidentIntegration.profile`.
They cannot supply an arbitrary identity endpoint, client secret, incident URL, or authorization
value directly in a stack request — see [SSO](sso.md).

## Public base, private environment overlay

The reusable implementation and a live environment have different publication boundaries:

| Public reference owns | Private environment owns |
| --- | --- |
| Provider and function packages at immutable digests | Approved public-reference Git commit |
| XRD schemas and Composition behaviour | Production API group under a controlled domain |
| Retain-by-default lifecycle and platform-controlled Delete authorization | Secret-store kind/name, cloud region, and workload identity |
| Placeholder SSO and incident profile shapes | Real endpoints and `ExternalSecret` remote paths |
| Comprehensive request with reserved example identities | Globally unique stack slug, intended recipients, user IDs, groups, and verified fixed-role UIDs |

An Argo CD `Application` can source `platform/` directly from this repository at an immutable
commit. Its Kustomize patches must replace every XRD group-qualified name, every XRD `spec.group`,
every Composition `spec.compositeTypeRef.apiVersion`, and the stack Composition input `apiVersion`.
A second source in the same `Application` can hold the environment `SecretStore`, `ExternalSecrets`,
and one ProviderConfig/credential Secret per registered organization in every request namespace.
The copies keep the registry's ProviderConfig name so namespaced references resolve locally. This keeps one Argo owner
while avoiding a copied platform implementation.

Do not put a patched live request in this public repository's `enabled/` directory — that
publishes a real cloud-resource identity and couples a production deployment to mutable public
data. Keep enabled requests or overlays in a private GitOps repository, and pin public platform
and catalog sources to reviewed commits.

## Next steps

- [Reference → Catalog](reference/catalog.md) — every worked example and what it demonstrates.
- [SSO](sso.md) — the four OAuth/SAML providers and the four reconciliation modes.
- [Architecture](architecture.md) — how these fields flow through Crossplane into managed
  resources.
