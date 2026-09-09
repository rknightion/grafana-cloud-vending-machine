---
title: Governance and product modules
description: Platform token policy, bounded product vending, promotion ladders, retention and identity scope
---

# Governance and product modules

Platform policy lives in Composition input. Request authors select approved names and supply their own workload definitions; they cannot supply policy ceilings or credential values. Catalog examples are inert Kustomize bases. Only `enabled/*` is watched for live requests. Product credentials use identity-bound bootstrap exchanges; provider readiness is not live product proof.

## Token lifetime and where tokens may be used

`spec.maximumTokenLifetime` in the platform Composition input is mandatory. Missing, malformed, zero, negative or sub-second values fail closed. The administrator, Fleet Management and telemetry rotating tokens use the lesser of the standard 30-day request and the platform ceiling. The seven-day early rotation window is shortened when needed to remain strictly inside that lifetime. Each provider-observed expiry appears in `status.tokenExpiries.administrator`, `fleetManagement` or `telemetryPublisher`; a timestamp is absent until the provider supplies it. These are observed expiries, not predicted rotation times.

`spec.tokenUseNetworkProfiles` selects `allowedSubnets` using the stack's immutable profile. It applies to the Fleet Management and telemetry Cloud access policies. If the profile has no subnet list, no `conditions` block is emitted. An explicitly empty subnet list is rejected so a mistaken restriction cannot silently become unrestricted token use. Request authors cannot pass CIDRs through the stack API. This restricts where those tokens may be used from. It does not filter inbound traffic to a Grafana stack and does not add a subnet condition to the Grafana administrator service-account token.

## Golden SLO

A platform `goldenSLOProfiles` entry keyed by usage supplies a ratio objective, queries and destination datasource UID. The stack observes that datasource before rendering its SLO. The SLO is created in handoff mode, with initial parameters under `initProvider`, so service teams can own later changes. Grafana generates recording rules and fastburn, slowburn and budget-remaining alerts from the SLO. The function does not invent workload metrics or include the preview alert-enrichment assistant block.

The [golden SLO catalog](reference/catalog.md) supplies inert ratio metric names. Adopters must replace the entire profile with reviewed workload queries and an existing datasource. A locally rendered resource does not prove those queries return useful traffic.

## k6 projects

`GrafanaK6Project` vends a project, `ProjectLimits` and `ProjectAllowedLoadZones`. Platform `k6LimitProfiles` select monthly VUH, per-test VUs, browser VUs and maximum duration from the referenced stack's immutable usage. Requested load zones must be an explicit subset of the platform list. An empty array allows no private zones; omission is rejected. The module adds explicit load tests and schedules after the project caps are observed Ready; it never creates private load zones. Admission checks the declared test VUs, browser VUs, duration and zones against the Composition profile, and reconciliation rejects a usage declaration that differs from the observed stack. Scripts pass through unchanged, so admission does not prove that executable JavaScript options match those declarations. k6 owns runtime enforcement of ProjectLimits. Schedules wait for observed LoadTest IDs and have Delete management enabled; the tests prove that declarative deletion path, not live remote deletion.

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

The schema rejects non-null SCIM input. Real API-server readback proves that explicit `spec.scim: null` is admitted and persisted with its key present; CEL `has()` does not see the null-valued field. The function sees that persisted key and refuses reconciliation with zero children. The owner accepted this [reconcile-time boundary](migration-1.0.md) on 2026-09-08, including attempts to mix SCIM with external-group mappings. No SCIM configuration is supported. SAML's external-UID assertion attribute alone is not treated as SCIM activation.

Enabling SCIM later is a breaking migration for every existing tenant using these mappings. A future contract must couple SCIM, SAML SSO and an external-UID assertion attribute matching the identity provider's SCIM identifier. The identity-provider half has no declarative coverage and would require manual per-tenant setup. Any migration must inventory and resolve existing Team ownership before activation; its reversal cost grows with tenant count.

The repository owner, acting as Grafana staff, answered both product questions on 2026-09-08: SCIM is available on all Grafana Cloud plans; disabling user sync leaves already-provisioned users unchanged and frozen. Their records remain active, authentication continues, and identity-provider updates stop. Disabling sync is therefore reversible and non-destructive. These are owner statements, not claims attributed to dated public documentation. The pinned provider includes `enterprise/ScimConfig` with `enableUserSync`, `enableGroupSync` and `rejectNonProvisionedUsers`; its schema does not establish disable semantics. Neither answer reopens the SCIM exclusion.

## Sandbox expiry uses the reviewed deletion path

Only platform-approved usage classes may select expiry. The function compares the injected reconcile time with the original deadline plus recorded extensions. Extension records are append-only; `requestedBy` and `recordedAt` are declared provenance and must be correlated with Kubernetes audit logs for authenticated actor and server time.

Expiry controls when Review 1 can arm deletion. The request must already declare `lifecycle.externalResources: Delete`, and the platform must authorize its exact namespace, name, UID and immutable profile. Authorization is validated before the deadline as well. Before the effective deadline, deletion fields stay in Retain mode; at the deadline, the same existing armed-delete fields are rendered and observed deletion readiness is reported. A timestamp alone never authorizes destruction.

Review 2 still removes dependent access claims and waits for their finalizers. Review 3 still removes the stack request in a separate reviewed Git change. Expiry does not delete the request, remove access claims, or bypass either review. Warnings use the already-vended production incident contact point; an approved expiry policy must name its warning lead time and folder, and the stack must have that delivery path configured.

Temporary product bootstrap or verification interruptions abort reconciliation before withdrawing existing children. The composite remains unready until the missing prerequisite returns; explicitly removed team-authored checks can still be removed.

Synthetic Monitoring owns deletion of its checks and disabled verifier. Removing a check from the whole list removes its external Check; rotating the derived credential retires the old verifier. This explicit ownership prevents orphaned active checks from continuing to consume the budget. Stack deletion still follows the separate reviewed lifecycle.

The temporary expiry warning RuleGroup is deletion-managed so leaving the warning window removes the external alert instead of orphaning a firing rule. This policy applies only to that warning, not to Stack deletion.
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

## Topology and retention reference

This reference covers token lifetime ceilings and token-use subnet profiles, golden SLO handoff, bounded k6 and Synthetic Monitoring vending, explicit promotion ladders, creation-time retention classes, and the enforced SCIM exclusion. Both shared-stack access slices and stack-per-tenant deployments are intentional topologies. Product credentials use identity-bound bootstrap exchanges; provider readiness is not live product proof.


Token-use subnet profiles restrict where Fleet Management and telemetry policy tokens may be used from. They are not inbound Grafana stack filtering; no inbound stack IP-filtering mechanism was identified. The administrator service-account token has no equivalent conditions field.

Shared stacks centralize operations and isolate tenants through teams, folders, RBAC, datasource permissions and LBAC. Separate stacks provide independent lifecycles and complete departmental isolation, with duplicated configuration and stack-cap costs. Both are supported deliberately. Ladders require one organization and immutable region/rung identities. Free plans allow one stack and self-service paid plans three; larger ladders need a negotiated cap. Multi-stack datasources require one region and were capped at ten stacks in preview. Promotion direction is explicit and never causes a region replacement.

Retention classes select durable collector fan-out at creation; they do not set retention periods. Logs export forwards a rolling window of roughly seven to thirty days. Logs retention can be changed through a self-serve API in thirty-day multiples up to one year, while shorter periods require a support request; this Composition does not reconcile that API. Metrics and traces have no self-serve retention API, and no equivalent bulk export was found. Stack deletion is permanent, so decommission cannot recover telemetry that was never forwarded.


## Adaptive products: deliberately out of scope

Adaptive Metrics is Grafana Cloud's largest cost lever, but this reference does not vend it.
Adaptive Logs, Adaptive Traces, and Adaptive Profiles are also out of scope. This is a deliberate
non-adoption decision, not an unsupported activation toggle hidden in the API: the stack request
does not expose any adaptive-product configuration.

Use the Grafana Cloud UI and the applicable ticket-based service route for those products. The
reversal cost is low because no request, credential, or composition here depends on this decision.
The feasible future route is a dedicated provider/module once upstream packaging is ready; any
future Adaptive Metrics design must choose either rulesets or individual rules as the sole owner
per segment, because mixing them overwrites rules. That merge question is not applicable while the
product remains out of scope.
