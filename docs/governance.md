---
title: Governance and product modules
description: Platform token policy, bounded product vending, promotion ladders, retention and identity scope
---

# Governance and product modules

Platform policy lives in Composition input. Request authors select approved names and supply their own workload definitions; they cannot supply policy ceilings or credential values. Catalog examples are inert Kustomize bases. Only `enabled/*` is watched for live requests.

## Token lifetime and where tokens may be used

`spec.maximumTokenLifetime` in the platform Composition input is mandatory. Missing, malformed, zero, negative or sub-second values fail closed. The administrator, Fleet Management and telemetry rotating tokens use the lesser of the standard 30-day request and the platform ceiling. The seven-day early rotation window is shortened when needed to remain strictly inside that lifetime. Each provider-observed expiry appears in `status.tokenExpiries.administrator`, `fleetManagement` or `telemetryPublisher`; a timestamp is absent until the provider supplies it. These are observed expiries, not predicted rotation times.

`spec.tokenUseNetworkProfiles` selects `allowedSubnets` using the stack's immutable profile. It applies to the Fleet Management and telemetry Cloud access policies. If the profile has no subnet list, no `conditions` block is emitted. An explicitly empty subnet list is rejected so a mistaken restriction cannot silently become unrestricted token use. Request authors cannot pass CIDRs through the stack API. This restricts where those tokens may be used from. It does not filter inbound traffic to a Grafana stack and does not add a subnet condition to the Grafana administrator service-account token.

## Golden SLO

A platform `goldenSLOProfiles` entry keyed by usage supplies a ratio objective, queries and destination datasource UID. The stack observes that datasource before rendering its SLO. The SLO is created in handoff mode, with initial parameters under `initProvider`, so service teams can own later changes. Grafana generates recording rules and fastburn, slowburn and budget-remaining alerts from the SLO. The function does not invent workload metrics or include the preview alert-enrichment assistant block.

The [golden SLO catalog](reference/catalog.md) supplies inert ratio metric names. Adopters must replace the entire profile with reviewed workload queries and an existing datasource. A locally rendered resource does not prove those queries return useful traffic.

## k6 projects

`GrafanaK6Project` vends a project, `ProjectLimits` and `ProjectAllowedLoadZones`. Platform `k6LimitProfiles` select monthly VUH, per-test VUs, browser VUs and maximum duration from the referenced stack's immutable usage. Requested load zones must be an explicit subset of the platform list. An empty array allows no private zones; omission is rejected. The module creates no tests, schedules or private load zones.

Installation exchanges the referenced stack's observed service-account token and a supplied Grafana user for a derived k6 credential. The function resolves the referenced stack and canonical local bootstrap Secret by identity, and passes references rather than token values. The derived credential is the downstream k6 provider credential and the only credential this module publishes to the secret store. The pinned provider's Installation fields and connection mapping are the compatibility contract; its upstream credential surface has changed rapidly.

## Synthetic Monitoring

`GrafanaSyntheticMonitoring` uses the existing local telemetry access-policy token, whose scope includes `stacks:read`, metrics, logs and traces write. It publishes only the derived Synthetic Monitoring credential. Private probe tokens belong in secret-store outputs and never in composite status.

Installation `Ready` alone is insufficient: an upstream defect can report creation complete while Synthetic Monitoring is not configured. Children require a disabled verification Check created and read through the derived-token SM provider. It targets a reserved `.invalid` address and schedules no executions. Its Ready condition plus observed tenant identity verifies a tenant-scoped API operation; public probe inventory is only an input and is not installation proof. This API creates no private probes or probe tokens.

Consuming teams author explicit check definitions. The Composition validates the whole submitted set against platform limits for count, probe locations, frequency and weighted executions, with browser executions costed separately. It never invents targets. This caps the API's managed checks, not a service-side tenant quota. Adopters must deny request authors direct writes to provider Check resources and platform policy, or those paths bypass the budget.

## Promotion ladders and tenancy

`GrafanaStackLadder` declares ordered rungs, branch-specific provisioning repositories, and an explicit `forward` or `reverse` content direction. Status records each rung's readiness, observed revision, source rung and drift. A missing revision means `Unknown`, not in sync. Promotion remains a reviewed Git operation; the renderer writes no Git content.

Organization, region and rung identities are immutable. Grafana's region slug forces replacement, so changing region requires a separately reviewed migration, never a ladder promotion. Cloud stacks are single-organization. Free plans allow one stack and self-service paid plans three; larger ladders require a negotiated cap, reflected in platform `ladderPolicy.maxStacks`. Multi-stack datasources require one region and were limited to ten stacks in preview.

Both shared-stack and stack-per-tenant topologies are supported deliberately. Grafana recommends one production stack with teams, folders, RBAC, datasource permissions and LBAC for isolation, and also supports multiple stacks for dev/staging ladders and complete departmental isolation. The stack API supports stack-per-tenant; the access APIs support slices of a shared stack. Shared stacks reduce stack count and centralize operations; separate stacks give stronger lifecycle isolation at the cost of duplicated configuration and contractual limits.

## Retention is decided at creation

`spec.retention.class` is immutable and must be selected at creation. It selects platform-owned Fleet Management collector configuration for durable fan-out from the start of the stack's life. The request cannot supply contents, destination credentials, or a retention period. A configured Pipeline proves the control-plane resource; an adopter must separately verify collector selection, receipt at the durable sink and the sink's retention policy.

Cloud Logs export syncs Loki chunks to a customer-owned bucket over a rolling window of roughly seven to thirty days. It extends retention forward and does not dump history. No equivalent bulk export was found for metrics or traces. Logs retention has a self-serve API in thirty-day multiples up to one year; values below thirty days need a support request. The logs API is not reconciled by this Composition and remains a future integration option. Metrics and traces have no self-serve retention API.

Stack deletion is permanent. Review 1 of the decommission runbook must verify the creation-time retention decision and durable fan-out evidence. Adding fan-out during decommission cannot recover telemetry that was never forwarded.

## SCIM remains outside the API

The supported identity model is Team Sync external-group mapping through `GrafanaTeamAccess.spec.team.externalGroups` and `GrafanaCustomRoleBinding.spec.team.groups`. Team Sync maps groups onto existing Teams; these APIs create and reconcile those Teams. SCIM group sync can create and delete Teams from identity-provider changes, so it is mutually exclusive with that ownership model.

SCIM requests are rejected by schema validation and defensively by the function, including attempts to mix SCIM with external-group mappings. The schema's rejected compatibility input exists only to prevent silent unknown-field pruning; it exposes no usable SCIM configuration. SAML's external-UID assertion attribute alone is not treated as SCIM activation.

Enabling SCIM later is a breaking migration for every existing tenant using these mappings. A future contract must couple SCIM, SAML SSO and an external-UID assertion attribute matching the identity provider's SCIM identifier. The identity-provider half has no declarative coverage and would require manual per-tenant setup. Any migration must inventory and resolve existing Team ownership before activation; its reversal cost grows with tenant count.

Two product questions remain explicitly unresolved: the Cloud plan floor (sources conflict between Pro-and-above and Advanced-only), and whether disabling user sync removes, disables or freezes existing users. The pinned provider schema has no entitlement or lifecycle evidence to settle them. Current authoritative product documentation or controlled tenant tests are required before either assumption enters an admission or deprovisioning workflow.

## Sandbox expiry uses the reviewed deletion path

Only platform-approved usage classes may select expiry. The function compares the injected reconcile time with the original deadline plus recorded extensions. Extension records are append-only; `requestedBy` and `recordedAt` are declared provenance and must be correlated with Kubernetes audit logs for authenticated actor and server time.

Expiry controls when Review 1 can arm deletion. The request must already declare `lifecycle.externalResources: Delete`, and the platform must authorize its exact namespace, name, UID and immutable profile. Authorization is validated before the deadline as well. Before the effective deadline, deletion fields stay in Retain mode; at the deadline, the same existing armed-delete fields are rendered and observed deletion readiness is reported. A timestamp alone never authorizes destruction.

Review 2 still removes dependent access claims and waits for their finalizers. Review 3 still removes the stack request in a separate reviewed Git change. Expiry does not delete the request, remove access claims, or bypass either review. Warnings use the already-vended production incident contact point; an approved expiry policy must name its warning lead time and folder, and the stack must have that delivery path configured.

Temporary product bootstrap or verification interruptions abort reconciliation before withdrawing existing children. The composite remains unready until the missing prerequisite returns; explicitly removed team-authored checks can still be removed.

Synthetic Monitoring owns deletion of its checks and disabled verifier. Removing a check from the whole list removes its external Check; rotating the derived credential retires the old verifier. This explicit ownership prevents orphaned active checks from continuing to consume the budget. Stack deletion still follows the separate reviewed lifecycle.

The temporary expiry warning RuleGroup is deletion-managed so leaving the warning window removes the external alert instead of orphaning a firing rule. This policy applies only to that warning, not to Stack deletion.
