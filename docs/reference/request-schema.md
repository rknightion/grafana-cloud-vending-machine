---
title: Request Schema Reference
description: The CompositeResourceDefinitions this repository ships, their kinds, short names, and CRD identity
---

# Request Schema Reference

This is the CRD identity and field-level request reference. Platform-owned policy, profiles,
and the organization registry are in [Configuration](../configuration.md); worked manifests are in the
[catalog](catalog.md). All types are namespaced, use
`platform.example.org/v1beta1` as a placeholder group, set
`defaultCompositionUpdatePolicy: Automatic`, and enforce one Pipeline Composition.

## CompositeResourceDefinitions

| Kind | Plural | Short name | Enforced Composition | Required `spec` fields |
| --- | --- | --- | --- | --- |
| `GrafanaCloudStackRequest` | `grafanacloudstackrequests` | `gcstackrequest` | `grafana-cloud-stack-request-v1beta1` | `displayName`, `slug`, `region`, `usage`, `organization` |
| `GrafanaCustomRoleBinding` | `grafanacustomrolebindings` | `gcrole` | `grafana-custom-role-binding-v1beta1` | `stackRef`, `team`, `role` |
| `GrafanaTeamAccess` | `grafanateamaccesses` | `gcteamaccess` | `grafana-team-access-v1beta1` | `stackRef`, `team` |
| `GrafanaContentAccessPolicy` | `grafanacontentaccesspolicies` | `gccontentaccess` | `grafana-content-access-policy-v1beta1` | `stackRef`, `target`, `permissions` |
| `GrafanaStackInventory` | `grafanastackinventories` | `gcinventory` | `grafana-stack-inventory-v1beta1` | `stackRef` |
| `GrafanaFleetPipelines` | `grafanafleetpipelines` | `gcfleet` | `grafana-fleet-pipelines-v1beta1` | `stackRef`, `profile` |
| `GrafanaAlertingBundle` | `grafanaalertingbundles` | `gcalerts` | `grafana-alerting-bundle-v1beta1` | `stackRef`, `provenance` |
| `GrafanaAgentObservability` | `grafanaagentobservabilities` | `gcagento11y` | `grafana-agent-observability-v1beta1` | `stackRef` |
| `GrafanaAssistantGovernance` | `grafanaassistantgovernances` | `gcassistant` | `grafana-assistant-governance-v1beta1` | `stackRef`, `termsAcceptance` |
| `GrafanaDatasourceAccess` | `grafanadatasourceaccesses` | `gcdatasourceaccess` | `grafana-datasource-access-v1beta1` | `stackRef`, `datasource`, `teams` |
| `GrafanaProvisioningRepository` | `grafanaprovisioningrepositories` | `gcprovisioningrepo` | `grafana-provisioning-repository-v1beta1` | `stackRef`, `repository` |

| `GrafanaK6Project` | `grafanak6projects` | `gck6` | `grafana-k6-project-v1beta1` | `stackRef`, `grafanaUser`, `allowedLoadZones` |
| `GrafanaSyntheticMonitoring` | `grafanasyntheticmonitorings` | `gcsm` | `grafana-synthetic-monitoring-v1beta1` | `stackRef` |
| `GrafanaStackLadder` | `grafanastackladders` | None | `grafana-stack-ladder-v1beta1` | `organization`, `region`, `promotionDirection`, `rungs`, `repository` |
| `GrafanaAlertingRouting` | `grafanaalertingroutings` | `gcarouting` | `grafana-alerting-routing-v1beta1` | `stackRef`, `contactPoints`, `defaultContactPoint` |
| `GrafanaOnCall` | `grafanaoncalls` | `gconcall` | `grafana-oncall-v1beta1` | `stackRef`, `responders`, `shiftStart`, `escalation`, `route` |
| `GrafanaCloudIntegrations` | `grafanacloudintegrations` | `gcci` | `grafana-cloud-integrations-v1beta1` | `stackRef`, `profile`, `scrapeJobs` |
| `GrafanaPDC` | `grafanapdcs` | `gcpdc` | `grafana-pdc-v1beta1` | `stackRef`, `profile`, `networks`, `token` |
| `GrafanaServiceAccounts` | `grafanaserviceaccounts` | `gcsa` | `grafana-service-accounts-v1beta1` | `stackRef`, `profile`, `accounts` |
| `GrafanaFrontendObservability` | `grafanafrontendobservabilities` | `gcfaro` | `grafana-frontend-observability-v1beta1` | `stackRef`, `profile` |
| `GrafanaML` | `grafanamls` | `gcml` | `grafana-ml-v1beta1` | `stackRef`, `profile` |

The XRDs and Compositions are split by API under `platform/apis/`. Every Composition has one
Pipeline step that calls `function-grafana-vending`; the function, rather than a separate
templating language, renders the managed resources.

## API overview

The primary API is GrafanaCloudStackRequest at platform.example.org/v1beta1. It is namespaced so teams or environments can be separated with Kubernetes namespaces and RBAC.

| Field | Purpose |
| --- | --- |
| spec.displayName | Human-readable stack name |
| spec.slug | Immutable Grafana Cloud slug; must equal metadata.name |
| spec.region | Grafana Cloud region slug |
| spec.usage | Immutable platform-approved classification and output-secret path segment |
| spec.organization | Required immutable platform registry key selecting the organization ProviderConfig and its allowed regions/usages |
| spec.profile | Platform-defined policy/profile label |
| spec.products | Activation toggles only for Application, Kubernetes, and Database Observability singletons; configuration stays in product-specific Helm/onboarding surfaces |
| spec.lifecycle.externalResources | `Retain` (default) or reviewed `Delete` lifecycle for external resources that can outlive the stack |
| spec.changeReference | Optional request or change identifier published in output metadata |
| spec.configurationItemReference | Optional inventory identifier published in output metadata |
| spec.baselineDashboards.enabled | Enables three starter folders and dashboards |
| spec.telemetryAccess.enabled | Enables a least-privilege rotating publisher credential |
| spec.plugins | Optional plugin slug/version list |
| spec.reconciliation | Selects content ownership behavior |
| spec.sso | Selects SSO profile and reconciliation mode |
| spec.monthlyReport | Optional scheduled usage report |
| spec.incidentIntegration | Optional relay-backed OnCall and Alerting endpoints plus OnCall template ownership mode |

Three access APIs sit beside the stack request:

| API | Ownership unit | Use it for |
| --- | --- | --- |
| GrafanaCustomRoleBinding | One Team, one custom Role, one whole-role RoleAssignment | Compact compatibility pattern for a unique custom role |
| GrafanaTeamAccess | One Team plus zero-to-many custom and fixed-role assignments | Directory sync, direct membership and administration, team preferences, additive custom roles, and existing fixed roles |
| GrafanaContentAccessPolicy | The complete ACL for one Folder or Dashboard | Basic-role, team, or user grants with one unambiguous owner per target |

GrafanaTeamAccess uses RoleAssignmentItem rather than the whole-set RoleAssignment resource. That makes independently owned team bundles additive. GrafanaContentAccessPolicy deliberately uses the whole-set FolderPermission or DashboardPermission resource; only one policy may own a given target.

The specialist APIs are opt-in modules rather than stack-request fields. Each names a stack in the
same namespace and uses its selected per-stack ProviderConfig:

| API | Activation or configuration boundary | Important limitation |
| --- | --- | --- |
| GrafanaStackInventory | Explicit observe-only inventory request | Reports declared, managed, and unmanaged objects; never emits a mutating child |
| GrafanaFleetPipelines | Selects a platform-owned pipeline profile | Collectors self-register; usage groups are UI-only and Advanced-tier |
| GrafanaAlertingBundle | Explicit alert resources plus required `enforced`/`createOnly` provenance | Never owns the organization-wide NotificationPolicy/Routingtree singleton |
| GrafanaAgentObservability | Explicit `guards` and `workload` sections | Does not install a plugin, mint credentials, or infer workload policy |
| GrafanaAssistantGovernance | Terms-acceptance-gated rule profile and MCP allow-list | Withholds rules/MCP servers until acceptance is observed; headers are write-only Secret data |
| GrafanaDatasourceAccess | One datasource's connection and authoritative team/LBAC set | Requires basic auth and entitlement; inherited or independent grants can bypass LBAC |
| GrafanaProvisioningRepository | Preview Git-provisioned folder subtree | References an existing credential-managed Grafana Connection; classic Dashboards remain the default |

The XRD uses `defaultCompositionUpdatePolicy: Automatic` and an enforced Composition reference. Existing requests therefore move to the latest Composition revision automatically after a platform update. Treat an XRD or function change like a production API release: render it, inspect the desired-resource diff, and roll it through a non-production request first.

## Additional specialist APIs

These APIs are separate opt-in requests. Admission validates local references and
selected Composition policy; reconciliation waits for identity-checked stack
context and provider-assigned IDs. Provider readback tests prove the rendered
configuration, not a live cloud transaction or notification delivery.

| API | Request fields | Policy and ownership |
| --- | --- | --- |
| `GrafanaAlertingRouting` | `stackRef`, `contactPoints`, `defaultContactPoint`, optional `routes` and `ruleGroups` | One request named after its stack owns the complete notification-policy tree. Every receiver is reachable. Each contact point selects literal `email` or a same-stack `onCallRef`. Ordinary rules omit direct notification settings and use this tree. |
| `GrafanaOnCall` | `stackRef`, UTC `shiftStart`, `responders`, `escalation`, `route` | One request per stack; individual responders rotate in weekly groups from the explicit anchor. Users and dependent identities are observed. Only the vended schedule and catch-all route are accepted. |
| `GrafanaCloudIntegrations` | `stackRef`, `profile`, `scrapeJobs` | One request per stack. Platform profiles own accounts, credential references, scrape count and interval budgets. A profile usage mismatch with the observed stack fails reconciliation. |
| `GrafanaPDC` | `stackRef`, `profile`, `networks`, `token.expiresAfter`, optional `datasources` | Datasources may reference only this request's networks. Tokens wait for observed network IDs and use the existing Composition lifetime ceiling. |
| `GrafanaServiceAccounts` | `stackRef`, `profile`, `accounts` | One request per stack; platform profiles own roles, rotating token lifetime and whole-set permissions. Static tokens and permission-item writers are excluded. |
| `GrafanaFrontendObservability` | `stackRef`, `profile` | One request per stack; platform profiles own Faro apps and origins. The observed collector endpoint contains a browser-visible app key; request-supplied keys are refused. |
| `GrafanaML` | `stackRef`, `profile` | One request per stack; the profile's jobs and outlier detectors must fit `maxRunningResources`. Jobs wait for Holiday IDs. A change that withdraws an observed child is refused until explicit decommission. |

Cloud-integration, PDC, service-account and ML admission policies use the actual
Composition as their parameter resource. Missing parameters deny admission.
Keep Composition and admission-policy writes platform-only. A budget bounds the
vended request set; it is not an inventory of unmanaged resources or proof of a
live account's total spend.

`GrafanaK6Project` additionally accepts `loadTests` and `schedules`. A request
with tests declares `usage`; reconciliation must bind it to the observed stack
usage. Admission checks structured HTTP workload limits against the declared
Composition profile; the function generates the executable script and execution
options. Arbitrary JavaScript and browser workloads are excluded from this new
subset. A false usage declaration can still pass admission and is refused at
reconciliation; cross-object budget admission remains unfinished. The existing
project limits remain the runtime enforcement boundary.
Schedules reference only vended tests and are Delete-managed.

`GrafanaSyntheticMonitoring.spec.checks[].alerts` optionally adds provider-native
CheckAlerts after the Check ID is observed. Existing checks may omit it. Private
probes remain refused because the pinned provider has no token-expiry control.
The existing usage-specific weighted check budget is enforced at reconciliation;
there is no cross-resource admission claim for that budget. CheckAlerts does not
select a receiver, and delivery through a vended notification policy remains
unproven for Synthetic Monitoring.

The [catalog](catalog.md) links each complete field example. All schemas and
Compositions live together under `platform/apis/`.

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
| `spec.sso.mode` | `disabled` | One of `enforced`, `createOnly`, `observeOnly`, `disabled`. See [SSO](../sso.md). |
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
[Architecture → reconciliation and out-of-band changes](../architecture.md#reconciliation-and-out-of-band-changes).

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

## Teams, roles, and content ACLs

Grafana authorization has several layers that should not be collapsed into one giant role
document:

1. SSO maps a user to the organization basic role Viewer, Editor, Admin, or None.
2. Teams group users through direct membership and/or identity-provider Team Sync.
3. Grafana-managed fixed roles provide reusable capability bundles and can be assigned to a Team by UID.
4. Custom roles contain explicit action/scope pairs and are assigned to Teams.
5. Folder and dashboard ACLs grant View, Edit, or Admin to a basic role, Team, or individual actor for one content target.

The current provider does not offer a resource that rewrites the global definitions of the
built-in Viewer, Editor, and Admin basic roles. The supported customization points are SSO
basic-role mapping, fixed/custom role assignments, and per-folder or per-dashboard permissions.
Fixed roles are Grafana-managed definitions. Assignment uses the role UUID, not its display name;
inventory the target stack before using any UUID because availability varies by Grafana version,
edition, entitlement, and stack creation date.

`GrafanaTeamAccess` supports both `members` and `externalGroups`. Direct members must already
exist in Grafana. External groups require the applicable Team Sync capability. With
`ignoreExternallySyncedMembers: true`, provider membership reconciliation ignores members supplied
by Team Sync while still managing the direct member set declared in Git. Give each Team one
Kubernetes owner.

`members` carries ordinary membership only; team administrators are a separate set. Declaring
administrators claims ownership of team administration, and an empty list demotes every
administrator, including ones added in the Grafana UI. Omitting the field leaves administration
alone, which is the safer choice for a team adopted from an existing stack.

Custom role permissions are an allow list; there is no deny rule. Prefer the narrowest action and
scope, keep `global: false` for stack-local roles, and validate actions against the Grafana version
deployed to the target stack. Narrow `folders:*` and `datasources:*` to named UIDs in a real
catalog rather than granting organization Admin.

The pinned provider has a Role initializer defect: although `autoIncrementVersion` is optional in
its CRD, the initializer rejects a Role that omits it. The function emits
`autoIncrementVersion: false` explicitly and never owns the deprecated, server-managed version
field. Recheck this workaround when upgrading the provider.

`RoleAssignment` manages the entire set of actors for a role and conflicts with
`RoleAssignmentItem`. `GrafanaCustomRoleBinding` is safe only because it creates a unique role and
owns that role's entire assignment set. `GrafanaTeamAccess` uses `RoleAssignmentItem` so multiple
team bundles can add assignments independently. Never manage the same role/actor pair through both
APIs.

`FolderPermission` and `DashboardPermission` also manage the entire ACL. Omitting an entry removes
it on the next enforced reconciliation. Put all grants for a target into one
`GrafanaContentAccessPolicy`, including basic-role grants that should remain. These stack-local ACL
resources retain or orphan their external state when their Kubernetes policy is removed; armed
Delete is reserved for state that can outlive the Stack. Edits while the policy exists are still
repaired.
### Role and ACL ownership

Grafana authorization has several layers that should not be collapsed into one giant role document:

1. SSO maps a user to the organization basic role Viewer, Editor, Admin, or None.
2. Teams group users through direct membership and/or identity-provider Team Sync.
3. Grafana-managed fixed roles provide reusable capability bundles and can be assigned to a Team by UID.
4. Custom roles contain explicit action/scope pairs and are assigned to Teams.
5. Folder and dashboard ACLs grant View, Edit, or Admin to a basic role, Team, or individual actor for one content target.

The current provider does not offer a resource that rewrites the global definitions of the built-in Viewer, Editor, and Admin basic roles. Do not model that as if it did. The supported customization points are SSO basic-role mapping, fixed/custom role assignments, and per-folder or per-dashboard permissions. Fixed roles are Grafana-managed definitions. Assignment uses the role UUID, not its display name: for example, the reviewed catalog maps fixed:datasources:reader to fixed_C2x8IxkiBc1KZVjyYH775T9jNMQ. Inventory the target stack before using any UUID because availability varies by version, edition, entitlement, and stack creation date.

GrafanaTeamAccess supports both members and externalGroups. Direct members must already exist in Grafana. External groups require the applicable Team Sync capability. With ignoreExternallySyncedMembers=true, provider membership reconciliation ignores members supplied by Team Sync while still managing the direct member set declared in Git. Give each Team one Kubernetes owner.

members carries ordinary membership only; team administrators are a separate set. Declaring administrators claims ownership of team administration, and because the provider treats an empty set as authoritative, an empty list demotes every administrator including ones added in the Grafana UI. Omitting the field leaves them alone, which is what a team adopted from an existing stack normally wants.

Custom role permissions are an allow list; there is no deny rule. Prefer the narrowest action and scope, keep global=false for stack-local roles, and validate actions against the Grafana version deployed to the target stack. The access-and-rbac example demonstrates alert-rule read/create/write plus the supporting folder-read and data-source-query permissions rather than granting organization Admin. Narrow folders:* and datasources:* to named UIDs in a real catalog.

The pinned provider has a Role initializer defect: although autoIncrementVersion is optional in its CRD, the initializer rejects a Role that omits it. The workaround still applies at v2.14.0. The function therefore emits autoIncrementVersion=false explicitly and never owns the deprecated, server-managed version field. Recheck this workaround when upgrading the provider.

RoleAssignment manages the entire set of actors for a role and conflicts with RoleAssignmentItem. GrafanaCustomRoleBinding is safe only because it creates a unique role and owns that role's entire assignment set. GrafanaTeamAccess uses RoleAssignmentItem so multiple team bundles can add assignments independently. Never manage the same role/actor pair through both APIs.

FolderPermission and DashboardPermission also manage the entire ACL. Omitting an entry removes it on the next enforced reconciliation. Put all grants for a target into one GrafanaContentAccessPolicy, including basic-role grants that should remain. These stack-local ACL resources retain/orphan their external state when their Kubernetes policy is removed; armed Delete is reserved for state that can outlive the Stack. Edits while the policy exists are still repaired.


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
### Observability products

`spec.products` activates Application, Kubernetes, and Database Observability global singletons.
It carries no configuration fields: Kubernetes configuration belongs in Helm values, while
Application and Database configuration belongs in their onboarding flows. Disabling a toggle
removes the composed child but the standard policy does not request external Delete.

### Role safety

`GrafanaCustomRoleBinding` and `GrafanaTeamAccess` filter the exact
`dashboards.public:write` action from custom roles unless the referenced stack's platform-owned
profile is in `publicDashboardProfiles`; request authors cannot grant that exception. Built-in
Viewer, Editor, and Admin roles are unchanged, and custom roles explicitly keep
`autoIncrementVersion: false`.


## Key admission and ownership guards

- A stack request requires immutable `spec.organization`, `spec.slug`, `spec.usage`, and
  `spec.profile`. Its organization key must resolve in the platform-owned registry; the selected
  organization must allow the requested region and usage.
- `GrafanaDatasourceAccess.metadata.name` must equal `spec.datasource.uid`; its datasource UID and
  stack reference are immutable. This gives one namespace-local owner for the datasource access
  set.
- `GrafanaProvisioningRepository` rejects a `dashboard` field. A Git-provisioned subtree and a
  classic Crossplane Dashboard must never claim the same content route.
- `GrafanaAlertingBundle.provenance` is required: `enforced` retains provisioning provenance and
  locks UI changes, while `createOnly` seeds values and preserves later UI edits.
- `GrafanaAssistantGovernance.termsAcceptance.accepted` is the safety gate. Rules and MCP servers
  are withheld until acceptance is observed, and a false value withdraws them.

## Quick lookups by short name

```bash
kubectl get gcstackrequest,gcrole,gcteamaccess,gccontentaccess -A
kubectl get gcinventory,gcfleet,gcalerts,gcagento11y,gcassistant -A
kubectl get gcdatasourceaccess,gcprovisioningrepo -A
```

## Status conditions

Every kind reports standard Crossplane `Synced` and `Ready` conditions. The stack request also
publishes `status.outputSecretPath`, `status.telemetrySecretPath` when enabled,
`status.deletionArmed`, `status.deletionReady`, `status.stack.id`, and `status.stack.url`.
`GrafanaStackInventory` additionally publishes its declared, managed, and unmanaged observation
classification. These status values are observations, not credentials or a substitute for an
inventory/adoption review.

## Next steps

- [Configuration](../configuration.md) - platform policy, profiles, and the organization registry.
- [Catalog Reference](catalog.md) - inert examples for every public API.

## Governance additions

`spec.retention.class` selects an immutable creation-time durable fan-out profile, not a retention period. `spec.expiry` declares an initial RFC3339 timestamp and append-only extension records (`extendedTo`, `reason`, `requestedBy`, `recordedAt`); declared requester/time fields require Kubernetes audit-log correlation for authenticated provenance. The function rejects SCIM input. The schema rejects non-null input. Explicit null is admitted and persisted, then refused by the renderer because the key remains present; see the [accepted reconcile-time boundary](../migration-1.0.md).

`GrafanaK6Project`, `GrafanaSyntheticMonitoring` and `GrafanaStackLadder` have separate XRDs. Their platform policy and evidence boundaries are in [Governance](../governance.md).
